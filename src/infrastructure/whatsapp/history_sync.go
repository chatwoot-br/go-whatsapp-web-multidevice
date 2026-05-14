package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var historySyncID int32

// Debounce timer for history sync webhook.
// WhatsApp sends multiple sync types (RECENT, PUSH_NAME, etc.) in sequence; we only
// dispatch the fork-specific history_sync_complete event after a quiet period to
// avoid duplicate webhook deliveries.
var (
	historySyncDebounceTimer *time.Timer
	historySyncDebounceMu    sync.Mutex
	historySyncDebounceDelay = 5 * time.Second
)

// Push name cache — push names keyed by phone number (user part of JID).
// Allows applying names even if PUSH_NAME arrives before/concurrently with RECENT.
var (
	pushNameCache   = make(map[string]string)
	pushNameCacheMu sync.RWMutex
)

func handleHistorySync(ctx context.Context, evt *events.HistorySync, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) {
	if client == nil || client.Store == nil || client.Store.ID == nil {
		log.Warnf("Skipping history sync handling: WhatsApp client not initialized")
		return
	}
	id := atomic.AddInt32(&historySyncID, 1)
	fileName := fmt.Sprintf("%s/history-%d-%s-%d-%s.json",
		config.PathStorages,
		startupTime,
		client.Store.ID.String(),
		id,
		evt.Data.SyncType.String(),
	)

	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Errorf("Failed to open file to write history sync: %v", err)
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err = enc.Encode(evt.Data); err != nil {
		log.Errorf("Failed to write history sync: %v", err)
		return
	}

	log.Infof("Wrote history sync to %s", fileName)

	// Process history sync data to database
	if chatStorageRepo != nil {
		if err := processHistorySync(ctx, evt.Data, chatStorageRepo, client); err != nil {
			log.Errorf("Failed to process history sync to database: %v", err)
		}
	}

	// Debounce webhook notification — wait for all sync events to complete.
	// Only schedule when webhooks are configured to avoid wasted timers.
	if len(config.WhatsappWebhook) > 0 {
		scheduleHistorySyncWebhook(chatStorageRepo, client, evt.Data.GetSyncType().String())
	}
}

// scheduleHistorySyncWebhook debounces webhook notifications.
// Resets timer on each sync event; only fires after a quiet period (historySyncDebounceDelay).
// At fire time it runs:
//  1. applyCachedPushNamesToChats — sync push names into chat rows
//  2. deduplicateLIDChats         — collapse @lid chats into their phone counterparts
//  3. forwardHistorySyncCompleteToWebhook — dispatch the fork-specific event
//  4. clearPushNameCache          — bound memory across sync cycles
func scheduleHistorySyncWebhook(chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client, syncType string) {
	historySyncDebounceMu.Lock()
	defer historySyncDebounceMu.Unlock()

	if historySyncDebounceTimer != nil {
		historySyncDebounceTimer.Stop()
	}

	log.Infof("History sync event (%s), waiting %v for more events before webhook", syncType, historySyncDebounceDelay)

	historySyncDebounceTimer = time.AfterFunc(historySyncDebounceDelay, func() {
		log.Infof("History sync debounce complete, sending webhook")

		if chatStorageRepo != nil && client != nil && client.Store != nil && client.Store.ID != nil {
			deviceJID := client.Store.ID.ToNonAD().String()
			applyCachedPushNamesToChats(context.Background(), chatStorageRepo, deviceJID)
			deduplicateLIDChats(context.Background(), chatStorageRepo, client, deviceJID)
		}

		forwardHistorySyncCompleteToWebhook(context.Background(), client, syncType)

		clearPushNameCache()
	})
}

// processHistorySync processes history sync data and stores messages in the database
func processHistorySync(ctx context.Context, data *waHistorySync.HistorySync, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) error {
	if data == nil {
		return nil
	}

	syncType := data.GetSyncType()
	log.Infof("Processing history sync type: %s", syncType.String())

	switch syncType {
	case waHistorySync.HistorySync_INITIAL_BOOTSTRAP, waHistorySync.HistorySync_RECENT:
		return processConversationMessages(ctx, data, chatStorageRepo, client)
	case waHistorySync.HistorySync_ON_DEMAND:
		// ON_DEMAND history sync (response to BuildHistorySyncRequest) — also forwards
		// individual messages to webhooks since these represent previously-unavailable messages.
		return processOnDemandHistorySync(ctx, data, chatStorageRepo, client)
	case waHistorySync.HistorySync_PUSH_NAME:
		return processPushNames(ctx, data, chatStorageRepo, client)
	default:
		log.Debugf("Skipping history sync type: %s", syncType.String())
		return nil
	}
}

// processConversationMessages processes and stores conversation messages from history sync.
// Uses NormalizeJIDFromLIDWithContext so LID resolution survives a cancelled event context.
func processConversationMessages(ctx context.Context, data *waHistorySync.HistorySync, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) error {
	conversations := data.GetConversations()
	log.Infof("Processing %d conversations from history sync", len(conversations))

	// Prioritize device JID from context (set by event handler with correct device instance)
	// over client.Store.ID which may point to a different device in multi-device scenarios.
	deviceID := ""
	if inst, ok := DeviceFromContext(ctx); ok && inst != nil {
		deviceID = inst.JID()
		if deviceID == "" {
			deviceID = inst.ID()
		}
	}
	if deviceID == "" && client != nil && client.Store != nil && client.Store.ID != nil {
		deviceID = client.Store.ID.ToNonAD().String()
	}

	for _, conv := range conversations {
		rawChatJID := conv.GetID()
		if rawChatJID == "" {
			continue
		}

		jid, err := types.ParseJID(rawChatJID)
		if err != nil {
			log.Warnf("Failed to parse JID %s: %v", rawChatJID, err)
			continue
		}

		jid = NormalizeJIDFromLIDWithContext(jid, client)
		chatJID := jid.String()

		displayName := conv.GetDisplayName()

		// Fall back to cached push name (populated by PUSH_NAME sync) when display name is empty.
		cachedPushName := GetPushNameFromCache(jid.User)
		if cachedPushName != "" && displayName == "" {
			displayName = cachedPushName
		}

		chatName := chatStorageRepo.GetChatNameWithPushName(jid, chatJID, "", displayName)
		ephemeralExpiration := conv.GetEphemeralExpiration()

		messages := conv.GetMessages()
		log.Debugf("Processing %d messages for chat %s", len(messages), chatJID)

		var messageBatch []*domainChatStorage.Message
		var latestTimestamp time.Time

		for _, histMsg := range messages {
			if histMsg == nil || histMsg.Message == nil {
				continue
			}

			msg := histMsg.Message
			msgKey := msg.GetKey()
			if msgKey == nil {
				continue
			}

			messageID := msgKey.GetID()
			if messageID == "" {
				continue
			}

			content := utils.ExtractMessageTextFromProto(msg.GetMessage())
			mediaType, filename, url, mediaKey, fileSHA256, fileEncSHA256, fileLength := utils.ExtractMediaInfo(msg.GetMessage())

			if content == "" && mediaType == "" {
				continue
			}

			sender := ""
			isFromMe := msgKey.GetFromMe()
			if isFromMe {
				if client != nil && client.Store.ID != nil {
					sender = client.Store.ID.ToNonAD().String()
				} else {
					log.Warnf("Skipping self-message %s: client ID unavailable", messageID)
					continue
				}
			} else {
				participant := msgKey.GetParticipant()
				if participant != "" {
					if senderJID, err := types.ParseJID(participant); err == nil {
						senderJID = NormalizeJIDFromLIDWithContext(senderJID, client)
						sender = senderJID.ToNonAD().String()
					} else {
						if participant != "" {
							sender = participant
						} else {
							log.Warnf("Skipping message %s: empty participant", messageID)
							continue
						}
					}
				} else {
					// Group messages must have a participant to identify the actual sender
					// (upstream-added safety check — see GitHub issue #609). Without it we'd
					// incorrectly store the group JID as the sender.
					if jid.Server == "g.us" {
						log.Warnf("Skipping group message %s in chat %s: no participant info available", messageID, chatJID)
						continue
					}
					sender = jid.String()
				}
			}

			timestamp := time.Unix(int64(msg.GetMessageTimestamp()), 0)

			if timestamp.After(latestTimestamp) {
				latestTimestamp = timestamp
			}

			message := &domainChatStorage.Message{
				ID:            messageID,
				ChatJID:       chatJID,
				DeviceID:      deviceID,
				Sender:        sender,
				Content:       content,
				Timestamp:     timestamp,
				IsFromMe:      isFromMe,
				MediaType:     mediaType,
				Filename:      filename,
				URL:           url,
				MediaKey:      mediaKey,
				FileSHA256:    fileSHA256,
				FileEncSHA256: fileEncSHA256,
				FileLength:    fileLength,
			}

			messageBatch = append(messageBatch, message)
		}

		if len(messageBatch) > 0 {
			chat := &domainChatStorage.Chat{
				DeviceID:            deviceID,
				JID:                 chatJID,
				Name:                chatName,
				LastMessageTime:     latestTimestamp,
				EphemeralExpiration: ephemeralExpiration,
			}

			if err := chatStorageRepo.StoreChat(chat); err != nil {
				log.Warnf("Failed to store chat %s: %v", chatJID, err)
				continue
			}

			if err := chatStorageRepo.StoreMessagesBatch(messageBatch); err != nil {
				log.Warnf("Failed to store messages batch for chat %s: %v", chatJID, err)
			} else {
				log.Debugf("Stored %d messages for chat %s", len(messageBatch), chatJID)
			}
		}
	}

	return nil
}

// processOnDemandHistorySync processes ON_DEMAND history sync responses (triggered when we
// request history for a specific chat, e.g. after unavailable messages). ON_DEMAND messages
// are forwarded individually to webhooks since they represent "new" messages not received in
// real-time. WhatsApp protocol limitations may make these rare in practice (issue #654).
func processOnDemandHistorySync(ctx context.Context, data *waHistorySync.HistorySync, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) error {
	conversations := data.GetConversations()
	log.Infof("[ON_DEMAND] Processing ON_DEMAND history sync with %d conversations", len(conversations))

	// Store messages first using existing logic; continue to forward even on storage failure.
	if err := processConversationMessages(ctx, data, chatStorageRepo, client); err != nil {
		log.Errorf("[ON_DEMAND] Failed to store messages: %v", err)
	}

	if len(config.WhatsappWebhook) > 0 {
		deviceID := ""
		if client != nil && client.Store != nil && client.Store.ID != nil {
			deviceJID := NormalizeJIDFromLIDWithContext(client.Store.ID.ToNonAD(), client)
			deviceID = deviceJID.ToNonAD().String()
		}

		messageCount := 0
		for _, conv := range conversations {
			for _, histMsg := range conv.GetMessages() {
				if histMsg == nil || histMsg.Message == nil {
					continue
				}
				forwardOnDemandMessageToWebhook(ctx, histMsg.Message, deviceID, client)
				messageCount++
			}
		}
		log.Infof("[ON_DEMAND] Forwarded %d messages to webhook", messageCount)
	}

	return nil
}

// forwardOnDemandMessageToWebhook forwards an ON_DEMAND history sync message to configured webhooks.
// These are messages that were previously unavailable (from other linked devices).
func forwardOnDemandMessageToWebhook(ctx context.Context, msg *waWeb.WebMessageInfo, deviceID string, client *whatsmeow.Client) {
	msgKey := msg.GetKey()
	if msgKey == nil {
		return
	}

	messageID := msgKey.GetID()
	if messageID == "" {
		return
	}

	content := utils.ExtractMessageTextFromProto(msg.GetMessage())
	if content == "" {
		return
	}

	chatJID := msgKey.GetRemoteJID()
	if jid, err := types.ParseJID(chatJID); err == nil {
		normalizedJID := NormalizeJIDFromLIDWithContext(jid, client)
		chatJID = normalizedJID.String()
	}

	sender := chatJID
	if msgKey.GetFromMe() && client != nil && client.Store != nil && client.Store.ID != nil {
		sender = client.Store.ID.ToNonAD().String()
	} else if participant := msgKey.GetParticipant(); participant != "" {
		if jid, err := types.ParseJID(participant); err == nil {
			normalizedJID := NormalizeJIDFromLIDWithContext(jid, client)
			sender = normalizedJID.String()
		}
	}

	payload := map[string]any{
		"event":     "message",
		"device_id": deviceID,
		"payload": map[string]any{
			"id":                messageID,
			"from":              chatJID,
			"sender":            sender,
			"body":              content,
			"timestamp":         time.Unix(int64(msg.GetMessageTimestamp()), 0).Format(time.RFC3339),
			"is_from_me":        msgKey.GetFromMe(),
			"from_history_sync": true,
			"sync_type":         "ON_DEMAND",
		},
	}

	if err := forwardPayloadToConfiguredWebhooks(ctx, payload, "on_demand_message"); err != nil {
		log.Errorf("[ON_DEMAND] Failed to forward message %s to webhook: %v", messageID, err)
	} else {
		log.Debugf("[ON_DEMAND] Forwarded message %s to webhook", messageID)
	}
}

// processPushNames processes push names from history sync to update chat names.
// First pass caches all push names by phone number; second pass updates existing chats.
func processPushNames(ctx context.Context, data *waHistorySync.HistorySync, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) error {
	pushnames := data.GetPushnames()
	log.Infof("Processing %d push names from history sync", len(pushnames))

	deviceID := ""
	if inst, ok := DeviceFromContext(ctx); ok && inst != nil {
		deviceID = inst.JID()
		if deviceID == "" {
			deviceID = inst.ID()
		}
	}
	if deviceID == "" && client != nil && client.Store != nil && client.Store.ID != nil {
		deviceID = client.Store.ID.ToNonAD().String()
	}

	// First pass: cache all push names by phone number for later use.
	pushNameCacheMu.Lock()
	for _, pushname := range pushnames {
		rawJIDStr := pushname.GetID()
		name := pushname.GetPushname()

		if rawJIDStr == "" || name == "" {
			continue
		}

		jid, err := types.ParseJID(rawJIDStr)
		if err != nil {
			continue
		}

		if jid.User != "" {
			pushNameCache[jid.User] = name
			log.Debugf("Cached push name for %s: %s", jid.User, name)
		}
	}
	pushNameCacheMu.Unlock()

	// Second pass: update existing chats.
	for _, pushname := range pushnames {
		rawJIDStr := pushname.GetID()
		name := pushname.GetPushname()

		if rawJIDStr == "" || name == "" {
			continue
		}

		jid, err := types.ParseJID(rawJIDStr)
		if err != nil {
			log.Warnf("Failed to parse JID %s in push names: %v", rawJIDStr, err)
			continue
		}

		var existingChat *domainChatStorage.Chat

		// Try 1: Normalized JID
		normalizedJID := NormalizeJIDFromLIDWithContext(jid, client)
		existingChat, _ = chatStorageRepo.GetChatByDevice(deviceID, normalizedJID.String())

		// Try 2: Standard s.whatsapp.net format
		if existingChat == nil && jid.User != "" {
			standardJID := jid.User + "@s.whatsapp.net"
			existingChat, _ = chatStorageRepo.GetChatByDevice(deviceID, standardJID)
		}

		// Try 3: Original JID format
		if existingChat == nil {
			existingChat, _ = chatStorageRepo.GetChatByDevice(deviceID, jid.String())
		}

		if existingChat == nil {
			// Chat doesn't exist yet — name is cached for when chat is created.
			continue
		}

		if existingChat.Name != name {
			// Only overwrite if current name is empty, a phone number, or the JID user part.
			if existingChat.Name == "" || existingChat.Name == jid.User || isPhoneNumber(existingChat.Name) {
				existingChat.Name = name
				if err := chatStorageRepo.StoreChat(existingChat); err != nil {
					log.Warnf("Failed to update chat name for %s: %v", existingChat.JID, err)
				} else {
					log.Debugf("Updated chat name for %s to %s", existingChat.JID, name)
				}
			}
		}
	}

	return nil
}

// isPhoneNumber checks if a string looks like a phone number (digits only, optionally with +).
func isPhoneNumber(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if c == '+' && i == 0 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// GetPushNameFromCache retrieves a cached push name by phone number.
func GetPushNameFromCache(phoneNumber string) string {
	pushNameCacheMu.RLock()
	defer pushNameCacheMu.RUnlock()
	return pushNameCache[phoneNumber]
}

// clearPushNameCache clears the push name cache after sync is complete.
func clearPushNameCache() {
	pushNameCacheMu.Lock()
	defer pushNameCacheMu.Unlock()
	pushNameCache = make(map[string]string)
	log.Debugf("Cleared push name cache")
}

// applyCachedPushNamesToChats applies any cached push names to chats that still have
// phone-number names. Called at debounce-fire time, before deduplicateLIDChats.
func applyCachedPushNamesToChats(ctx context.Context, chatStorageRepo domainChatStorage.IChatStorageRepository, deviceID string) {
	pushNameCacheMu.RLock()
	cacheSize := len(pushNameCache)
	pushNameCacheMu.RUnlock()

	if cacheSize == 0 {
		return
	}

	log.Infof("Applying %d cached push names to chats", cacheSize)

	filter := &domainChatStorage.ChatFilter{
		DeviceID: deviceID,
		Limit:    1000,
	}

	chats, err := chatStorageRepo.GetChats(filter)
	if err != nil {
		log.Warnf("Failed to get chats for push name application: %v", err)
		return
	}

	updated := 0
	for _, chat := range chats {
		phoneNumber := extractPhoneFromJID(chat.JID)
		if phoneNumber == "" {
			continue
		}

		pushName := GetPushNameFromCache(phoneNumber)
		if pushName == "" {
			continue
		}

		if chat.Name != pushName && isPhoneNumber(chat.Name) {
			chat.Name = pushName
			if err := chatStorageRepo.StoreChat(chat); err != nil {
				log.Warnf("Failed to apply push name to chat %s: %v", chat.JID, err)
			} else {
				updated++
				log.Debugf("Applied push name to chat %s: %s", chat.JID, pushName)
			}
		}
	}

	if updated > 0 {
		log.Infof("Updated %d chat names from push name cache", updated)
	}
}

// deduplicateLIDChats finds and merges LID-based chats that have phone-number mappings.
// Called at debounce-fire time, after applyCachedPushNamesToChats and before
// forwardHistorySyncCompleteToWebhook. Fork-only — confirmed not subsumed by upstream's
// LID commits (40b0875, d718ef8, 17ff32f) per investigation findings in 02-design.md.
func deduplicateLIDChats(ctx context.Context, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client, deviceID string) {
	if chatStorageRepo == nil || client == nil {
		return
	}

	lidChats, err := chatStorageRepo.GetLIDChats(deviceID)
	if err != nil {
		log.Warnf("Failed to get LID chats for deduplication: %v", err)
		return
	}

	if len(lidChats) == 0 {
		return
	}

	log.Infof("Found %d LID-based chats to check for deduplication", len(lidChats))

	merged := 0
	for _, chat := range lidChats {
		lidJID, err := types.ParseJID(chat.JID)
		if err != nil {
			log.Warnf("Failed to parse LID JID %s: %v", chat.JID, err)
			continue
		}

		phoneJID := NormalizeJIDFromLIDWithContext(lidJID, client)

		// If resolution succeeded (different JID returned), attempt to merge.
		if phoneJID.Server != "lid" {
			phoneJIDStr := phoneJID.String()

			if err := chatStorageRepo.MergeLIDChat(deviceID, chat.JID, phoneJIDStr); err != nil {
				log.Warnf("Failed to merge LID chat %s into %s: %v", chat.JID, phoneJIDStr, err)
			} else {
				merged++
				log.Debugf("Merged LID chat %s into %s", chat.JID, phoneJIDStr)
			}
		}
	}

	if merged > 0 {
		log.Infof("Deduplicated %d LID-based chats", merged)
	}
}

// extractPhoneFromJID extracts the phone number (user part) from a JID string.
// Local helper: avoids the round-trip through types.ParseJID for an O(n) string scan.
func extractPhoneFromJID(jid string) string {
	// JID format: phone@server or phone:device@server
	atIdx := -1
	colonIdx := -1
	for i, c := range jid {
		if c == '@' {
			atIdx = i
			break
		}
		if c == ':' {
			colonIdx = i
		}
	}

	if atIdx == -1 {
		return jid
	}

	if colonIdx != -1 && colonIdx < atIdx {
		return jid[:colonIdx]
	}

	return jid[:atIdx]
}
