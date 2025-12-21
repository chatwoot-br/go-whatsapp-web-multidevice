package whatsapp

import (
	"context"
	"time"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// forwardDeleteToWebhook sends a delete event to webhook
func forwardDeleteToWebhook(ctx context.Context, evt *events.DeleteForMe, message *domainChatStorage.Message) error {
	payload, err := createDeletePayload(ctx, evt, message)
	if err != nil {
		return err
	}

	return forwardPayloadToConfiguredWebhooks(ctx, payload, "delete event")
}

// createDeletePayload creates a webhook payload for delete events
func createDeletePayload(ctx context.Context, evt *events.DeleteForMe, message *domainChatStorage.Message) (map[string]any, error) {
	body := make(map[string]any)

	// Basic delete event information
	body["action"] = "event.delete_for_me"
	body["deleted_message_id"] = evt.MessageID
	body["sender_id"] = evt.SenderJID.User
	body["timestamp"] = time.Now().Format(time.RFC3339)

	// Resolve LID/JID for sender
	resolver := GetLIDResolver()
	if resolver != nil {
		senderPN, senderLID := resolver.ResolveToPNForWebhook(ctx, evt.SenderJID)
		body["sender_jid"] = senderPN.String()
		if !senderLID.IsEmpty() {
			body["sender_lid"] = senderLID.String()
		}
	}

	// Include original message information if available
	if message != nil {
		body["chat_id"] = message.ChatJID
		body["original_content"] = message.Content
		body["original_sender"] = message.Sender
		body["original_timestamp"] = message.Timestamp.Format(time.RFC3339)
		body["was_from_me"] = message.IsFromMe

		// Resolve LID/JID for chat
		if resolver != nil {
			if chatJID, err := types.ParseJID(message.ChatJID); err == nil {
				chatPN, chatLID := resolver.ResolveToPNForWebhook(ctx, chatJID)
				body["chat_jid"] = chatPN.String()
				if !chatLID.IsEmpty() {
					body["chat_lid"] = chatLID.String()
				}
			}
		}

		if message.MediaType != "" {
			body["original_media_type"] = message.MediaType
			body["original_filename"] = message.Filename
		}
	}

	// Parse sender JID for proper formatting
	if evt.SenderJID.Server != "" {
		body["from"] = evt.SenderJID.String()
	}

	return body, nil
}
