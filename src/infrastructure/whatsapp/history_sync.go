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
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var historySyncID int32

// Debounce timer for history sync webhook
// Only sends webhook after no sync events for debounceDelay
var (
	historySyncDebounceTimer *time.Timer
	historySyncDebounceMu    sync.Mutex
	historySyncDebounceDelay = 5 * time.Second
)

// Push name cache - stores push names keyed by phone number (user part of JID)
// This allows applying names even if PUSH_NAME arrives before RECENT sync
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

	// Debounce webhook notification - wait for all sync events to complete
	// WhatsApp sends multiple sync types (RECENT, PUSH_NAME, etc.) in sequence
	// Only send webhook after no sync events for debounceDelay
	if len(config.WhatsappWebhook) > 0 {
		scheduleHistorySyncWebhook(chatStorageRepo, client, evt.Data.GetSyncType().String())
	}
}

// scheduleHistorySyncWebhook debounces webhook notifications
// Resets timer on each sync event, only fires after quiet period
func scheduleHistorySyncWebhook(chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client, syncType string) {
	historySyncDebounceMu.Lock()
	defer historySyncDebounceMu.Unlock()

	// Cancel existing timer if any
	if historySyncDebounceTimer != nil {
		historySyncDebounceTimer.Stop()
	}

	log.Infof("History sync event (%s), waiting %v for more events before webhook", syncType, historySyncDebounceDelay)

	// Schedule new webhook after delay
	historySyncDebounceTimer = time.AfterFunc(historySyncDebounceDelay, func() {
		log.Infof("History sync debounce complete, sending webhook")

		// Apply cached push names to any chats that still have phone numbers as names
		if chatStorageRepo != nil && client != nil && client.Store != nil && client.Store.ID != nil {
			deviceJID := client.Store.ID.ToNonAD().String()
			applyCachedPushNamesToChats(context.Background(), chatStorageRepo, deviceJID)
		}

		forwardHistorySyncCompleteToWebhook(context.Background(), client, syncType)

		// Clear the push name cache after webhook is sent
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
		// Process conversation messages
		return processConversationMessages(ctx, data, chatStorageRepo, client)
	case waHistorySync.HistorySync_PUSH_NAME:
		// Process push names to update chat names
		return processPushNames(ctx, data, chatStorageRepo, client)
	default:
		// Other sync types are not needed for message storage
		log.Debugf("Skipping history sync type: %s", syncType.String())
		return nil
	}
}

// processConversationMessages processes and stores conversation messages from history sync
func processConversationMessages(ctx context.Context, data *waHistorySync.HistorySync, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) error {
	conversations := data.GetConversations()
	log.Infof("Processing %d conversations from history sync", len(conversations))

	// Prioritize device JID from context (set by event handler with correct device instance)
	// over client.Store.ID which may point to a different device in multi-device scenarios
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

		// Parse JID to get proper format
		jid, err := types.ParseJID(rawChatJID)
		if err != nil {
			log.Warnf("Failed to parse JID %s: %v", rawChatJID, err)
			continue
		}

		// Normalize JID (convert @lid to @s.whatsapp.net if possible)
		jid = NormalizeJIDFromLID(ctx, jid, client)
		chatJID := jid.String()

		displayName := conv.GetDisplayName()

		// Try to get push name from cache (populated by PUSH_NAME sync)
		// This handles the case where PUSH_NAME arrives before or concurrently with RECENT
		cachedPushName := GetPushNameFromCache(jid.User)
		if cachedPushName != "" && displayName == "" {
			displayName = cachedPushName
		}

		// Get or create chat
		chatName := chatStorageRepo.GetChatNameWithPushName(jid, chatJID, "", displayName)

		// Extract ephemeral expiration from conversation
		ephemeralExpiration := conv.GetEphemeralExpiration()

		// Process messages in the conversation
		messages := conv.GetMessages()
		log.Debugf("Processing %d messages for chat %s", len(messages), chatJID)

		// Collect messages for batch processing
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

			// Skip messages without ID
			messageID := msgKey.GetID()
			if messageID == "" {
				continue
			}

			// Extract message content and media info
			content := utils.ExtractMessageTextFromProto(msg.GetMessage())
			mediaType, filename, url, mediaKey, fileSHA256, fileEncSHA256, fileLength := utils.ExtractMediaInfo(msg.GetMessage())

			// Skip if there's no content and no media
			if content == "" && mediaType == "" {
				continue
			}

			// Determine sender
			sender := ""
			isFromMe := msgKey.GetFromMe()
			if isFromMe {
				// For self-messages, use the full JID format to match regular message processing
				if client != nil && client.Store.ID != nil {
					sender = client.Store.ID.ToNonAD().String() // Use full JID instead of just User part
				} else {
					// Skip messages where we can't determine the sender to avoid NOT NULL violations
					log.Warnf("Skipping self-message %s: client ID unavailable", messageID)
					continue
				}
			} else {
				participant := msgKey.GetParticipant()
				if participant != "" {
					// For group messages, participant contains the actual sender
					if senderJID, err := types.ParseJID(participant); err == nil {
						// Normalize sender JID (convert @lid to @s.whatsapp.net if possible)
						senderJID = NormalizeJIDFromLID(ctx, senderJID, client)
						sender = senderJID.ToNonAD().String() // Use full JID format for consistency
					} else {
						// Fallback to participant string, but ensure it's not empty
						if participant != "" {
							sender = participant
						} else {
							log.Warnf("Skipping message %s: empty participant", messageID)
							continue
						}
					}
				} else {
					// For individual chats, use the chat JID as sender with full format
					sender = jid.String() // Use full JID format for consistency
				}
			}

			// Convert timestamp from Unix seconds to time.Time
			// WhatsApp history sync timestamps are in seconds, not milliseconds
			timestamp := time.Unix(int64(msg.GetMessageTimestamp()), 0)

			// Track latest timestamp
			if timestamp.After(latestTimestamp) {
				latestTimestamp = timestamp
			}

			// Create message object and add to batch
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

		// Store or update the chat with latest message time
		if len(messageBatch) > 0 {
			chat := &domainChatStorage.Chat{
				DeviceID:            deviceID,
				JID:                 chatJID,
				Name:                chatName,
				LastMessageTime:     latestTimestamp,
				EphemeralExpiration: ephemeralExpiration,
			}

			// Store or update the chat
			if err := chatStorageRepo.StoreChat(chat); err != nil {
				log.Warnf("Failed to store chat %s: %v", chatJID, err)
				continue
			}

			// Store messages in batch
			if err := chatStorageRepo.StoreMessagesBatch(messageBatch); err != nil {
				log.Warnf("Failed to store messages batch for chat %s: %v", chatJID, err)
			} else {
				log.Debugf("Stored %d messages for chat %s", len(messageBatch), chatJID)
			}
		}
	}

	return nil
}

// processPushNames processes push names from history sync to update chat names
func processPushNames(ctx context.Context, data *waHistorySync.HistorySync, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) error {
	pushnames := data.GetPushnames()
	log.Infof("Processing %d push names from history sync", len(pushnames))

	// Extract device ID from context (same pattern as processConversationMessages)
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

	// First pass: cache all push names by phone number for later use
	pushNameCacheMu.Lock()
	for _, pushname := range pushnames {
		rawJIDStr := pushname.GetID()
		name := pushname.GetPushname()

		if rawJIDStr == "" || name == "" {
			continue
		}

		// Parse JID to extract user (phone number)
		jid, err := types.ParseJID(rawJIDStr)
		if err != nil {
			continue
		}

		// Cache by phone number (user part) for flexible lookup
		if jid.User != "" {
			pushNameCache[jid.User] = name
			log.Debugf("Cached push name for %s: %s", jid.User, name)
		}
	}
	pushNameCacheMu.Unlock()

	// Second pass: update existing chats
	for _, pushname := range pushnames {
		rawJIDStr := pushname.GetID()
		name := pushname.GetPushname()

		if rawJIDStr == "" || name == "" {
			continue
		}

		// Parse and normalize JID (convert @lid to @s.whatsapp.net if possible)
		jid, err := types.ParseJID(rawJIDStr)
		if err != nil {
			log.Warnf("Failed to parse JID %s in push names: %v", rawJIDStr, err)
			continue
		}

		// Try to find chat with multiple JID formats
		var existingChat *domainChatStorage.Chat

		// Try 1: Normalized JID
		normalizedJID := NormalizeJIDFromLID(ctx, jid, client)
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
			// Chat doesn't exist yet - name is cached for when chat is created
			continue
		}

		// Update chat name if it's different and current name looks like a phone number
		if existingChat.Name != name {
			// Only update if current name is empty, a phone number, or the JID user part
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

// isPhoneNumber checks if a string looks like a phone number (digits only, optionally with +)
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

// GetPushNameFromCache retrieves a cached push name by phone number
func GetPushNameFromCache(phoneNumber string) string {
	pushNameCacheMu.RLock()
	defer pushNameCacheMu.RUnlock()
	return pushNameCache[phoneNumber]
}

// clearPushNameCache clears the push name cache after sync is complete
func clearPushNameCache() {
	pushNameCacheMu.Lock()
	defer pushNameCacheMu.Unlock()
	pushNameCache = make(map[string]string)
	log.Debugf("Cleared push name cache")
}

// applyCachedPushNamesToChats applies any cached push names to chats that still have phone number names
func applyCachedPushNamesToChats(ctx context.Context, chatStorageRepo domainChatStorage.IChatStorageRepository, deviceID string) {
	pushNameCacheMu.RLock()
	cacheSize := len(pushNameCache)
	pushNameCacheMu.RUnlock()

	if cacheSize == 0 {
		return
	}

	log.Infof("Applying %d cached push names to chats", cacheSize)

	// Get all chats for this device
	filter := &domainChatStorage.ChatFilter{
		DeviceID: deviceID,
		Limit:    1000, // Process in batches if needed
	}

	chats, err := chatStorageRepo.GetChats(filter)
	if err != nil {
		log.Warnf("Failed to get chats for push name application: %v", err)
		return
	}

	updated := 0
	for _, chat := range chats {
		// Extract phone number from JID
		phoneNumber := extractPhoneFromJID(chat.JID)
		if phoneNumber == "" {
			continue
		}

		// Check if we have a cached push name for this number
		pushName := GetPushNameFromCache(phoneNumber)
		if pushName == "" {
			continue
		}

		// Only update if current name looks like a phone number
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

// extractPhoneFromJID extracts the phone number (user part) from a JID string
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
		return jid // No @ found, return as is
	}

	if colonIdx != -1 && colonIdx < atIdx {
		return jid[:colonIdx] // Return part before :
	}

	return jid[:atIdx] // Return part before @
}

// forwardHistorySyncCompleteToWebhook sends a webhook notification when history sync completes
func forwardHistorySyncCompleteToWebhook(ctx context.Context, client *whatsmeow.Client, syncType string) {
	deviceID := ""
	if client != nil && client.Store != nil && client.Store.ID != nil {
		deviceJID := NormalizeJIDFromLID(ctx, client.Store.ID.ToNonAD(), client)
		deviceID = deviceJID.ToNonAD().String()
	}

	payload := map[string]any{
		"event":     "history_sync_complete",
		"device_id": deviceID,
		"payload": map[string]any{
			"sync_type": syncType,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	if err := forwardPayloadToConfiguredWebhooks(ctx, payload, "history_sync_complete"); err != nil {
		log.Errorf("Failed to forward history_sync_complete webhook: %v", err)
	}
}
