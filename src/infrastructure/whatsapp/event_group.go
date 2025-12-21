package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// createGroupInfoPayload creates a webhook payload for group information events
func createGroupInfoPayload(evt *events.GroupInfo, actionType string, jids []types.JID) map[string]any {
	body := make(map[string]any)

	// Create payload structure matching the expected format
	payload := make(map[string]any)

	// Add group chat ID
	payload["chat_id"] = evt.JID.String()

	// Add action type and affected users with LID resolution
	payload["type"] = actionType
	jidStrings, lidStrings := jidsWithLIDs(context.Background(), jids)
	payload["jids"] = jidStrings
	if len(lidStrings) > 0 {
		payload["lids"] = lidStrings
	}

	// Wrap in payload structure
	body["payload"] = payload

	// Add metadata for webhook processing
	body["event"] = "group.participants"
	body["timestamp"] = evt.Timestamp.Format(time.RFC3339)

	return body
}

// jidsToStrings converts a slice of JIDs to a slice of strings
func jidsToStrings(jids []types.JID) []string {
	if len(jids) == 0 {
		return []string{} // Return empty array instead of nil for consistent JSON
	}

	result := make([]string, len(jids))
	for i, jid := range jids {
		result[i] = jid.String()
	}
	return result
}

// jidsWithLIDs converts JIDs to strings and also resolves their LIDs
func jidsWithLIDs(ctx context.Context, jids []types.JID) ([]string, []string) {
	if len(jids) == 0 {
		return []string{}, []string{}
	}

	jidStrings := make([]string, len(jids))
	lidStrings := make([]string, 0, len(jids))

	resolver := GetLIDResolver()
	for i, jid := range jids {
		jidStrings[i] = jid.String()
		if resolver != nil {
			lidJID := resolver.ResolveToLID(ctx, jid)
			if lidJID.Server == "lid" {
				lidStrings = append(lidStrings, lidJID.String())
			}
		}
	}
	return jidStrings, lidStrings
}

// forwardGroupInfoToWebhook forwards group information events to the configured webhook URLs
func forwardGroupInfoToWebhook(ctx context.Context, evt *events.GroupInfo) error {
	logrus.Infof("Forwarding group info event to %d configured webhook(s)", len(config.WhatsappWebhook))

	// Send separate webhook events for each action type
	actions := []struct {
		actionType string
		jids       []types.JID
	}{
		{"join", evt.Join},
		{"leave", evt.Leave},
		{"promote", evt.Promote},
		{"demote", evt.Demote},
	}

	for _, action := range actions {
		if len(action.jids) > 0 {
			payload := createGroupInfoPayload(evt, action.actionType, action.jids)

			// Collect errors from all webhook URLs instead of failing fast
			var errors []error
			for _, url := range config.WhatsappWebhook {
				if err := submitWebhook(ctx, payload, url); err != nil {
					errors = append(errors, fmt.Errorf("webhook %s failed: %w", url, err))
				}
			}

			// If all webhooks failed, return combined error
			if len(errors) == len(config.WhatsappWebhook) && len(errors) > 0 {
				var errMessages []string
				for _, err := range errors {
					errMessages = append(errMessages, err.Error())
				}
				return fmt.Errorf("all webhook URLs failed: %s", strings.Join(errMessages, "; "))
			}

			// Log partial failures
			if len(errors) > 0 {
				logrus.Warnf("Some webhook URLs failed for group %s event: %v", action.actionType, errors)
			}

			logrus.Infof("Group %s event forwarded to webhook: %d users %s", action.actionType, len(action.jids), action.actionType)
		}
	}

	return nil
}
