package whatsapp

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types/events"
)

// forwardMessageToWebhook is a helper function to forward message event to webhook url
func forwardMessageToWebhook(ctx context.Context, evt *events.Message, downloadedMedia *DownloadedMedia) error {
	payload, err := createMessagePayload(ctx, evt, downloadedMedia)
	if err != nil {
		return err
	}

	return forwardPayloadToConfiguredWebhooks(ctx, payload, "message event")
}

func createMessagePayload(ctx context.Context, evt *events.Message, downloadedMedia *DownloadedMedia) (map[string]any, error) {
	message := utils.BuildEventMessage(evt)
	waReaction := utils.BuildEventReaction(evt)
	forwarded := utils.BuildForwarded(evt)

	body := make(map[string]any)

	// Always include raw IDs for reference
	body["sender_id"] = evt.Info.Sender.User
	body["chat_id"] = evt.Info.Chat.User

	// Get resolver for LID <-> PN resolution
	resolver := GetLIDResolver()

	// Variables to hold resolved JIDs
	var senderPN, senderLID, chatPN, chatLID types.JID

	// Resolve JIDs if resolver is available
	if resolver != nil {
		// Resolve sender for external systems - they need phone numbers for compatibility
		senderPN, senderLID = resolver.ResolveToPNForWebhook(ctx, evt.Info.Sender)

		// Resolve chat JID similarly
		chatPN, chatLID = resolver.ResolveToPNForWebhook(ctx, evt.Info.Chat)
	} else {
		// Fallback: use original JIDs when resolver is not available
		senderPN = evt.Info.Sender
		chatPN = evt.Info.Chat
	}

	// Build the "from" field with resolved phone number for backward compatibility
	if from := evt.Info.SourceString(); from != "" {
		fromUser, fromGroup := from, ""
		if strings.Contains(from, " in ") {
			parts := strings.Split(from, " in ")
			fromUser = parts[0]
			if len(parts) > 1 {
				fromGroup = parts[1]
			}
		}

		// Build resolved "from" with phone number if available
		if !senderPN.IsEmpty() && senderPN.Server == types.DefaultUserServer {
			if fromGroup != "" {
				body["from"] = fmt.Sprintf("%s in %s", senderPN.String(), fromGroup)
			} else {
				body["from"] = senderPN.String()
			}
		} else {
			// Fallback to original source string
			body["from"] = from
		}

		// Always include from_lid if LID is available
		if !senderLID.IsEmpty() {
			body["from_lid"] = senderLID.String()
		} else if strings.HasSuffix(fromUser, "@lid") {
			// Fallback: extract LID from source string if resolver didn't return one
			body["from_lid"] = fromUser
		}
	}

	// Add resolved chat JID fields for external systems
	if !chatPN.IsEmpty() {
		body["chat_jid"] = chatPN.String()
	}
	if !chatLID.IsEmpty() {
		body["chat_lid"] = chatLID.String()
	}
	if message.ID != "" {
		tags := regexp.MustCompile(`\B@\w+`).FindAllString(message.Text, -1)
		tagsMap := make(map[string]bool)
		for _, tag := range tags {
			tagsMap[tag] = true
		}
		for tag := range tagsMap {
			lid, err := types.ParseJID(tag[1:] + "@lid")
			if err != nil {
				logrus.Errorf("Error when parse jid: %v", err)
			} else {
				pn, err := cli.Store.LIDs.GetPNForLID(ctx, lid)
				if err != nil {
					logrus.Errorf("Error when get pn for lid %s: %v", lid.String(), err)
				}
				if !pn.IsEmpty() {
					message.Text = strings.Replace(message.Text, tag, fmt.Sprintf("@%s", pn.User), -1)
				}
			}
		}
		body["message"] = message
	}
	if pushname := evt.Info.PushName; pushname != "" {
		body["pushname"] = pushname
	}
	if waReaction.Message != "" {
		body["reaction"] = waReaction
	}
	if evt.IsViewOnce {
		body["view_once"] = evt.IsViewOnce
	}
	if forwarded {
		body["forwarded"] = forwarded
	}
	if timestamp := evt.Info.Timestamp.Format(time.RFC3339); timestamp != "" {
		body["timestamp"] = timestamp
	}

	// Handle protocol messages (revoke, etc.)
	if protocolMessage := evt.Message.GetProtocolMessage(); protocolMessage != nil {
		protocolType := protocolMessage.GetType().String()

		switch protocolType {
		case "REVOKE":
			body["action"] = "message_revoked"
			if key := protocolMessage.GetKey(); key != nil {
				body["revoked_message_id"] = key.GetID()
				body["revoked_from_me"] = key.GetFromMe()
				if key.GetRemoteJID() != "" {
					body["revoked_chat"] = key.GetRemoteJID()
				}
			}
		case "MESSAGE_EDIT":
			body["action"] = "message_edited"
			// Extract the original message ID from the protocol message key
			if key := protocolMessage.GetKey(); key != nil {
				body["original_message_id"] = key.GetID()
			}
			if editedMessage := protocolMessage.GetEditedMessage(); editedMessage != nil {
				if editedText := editedMessage.GetExtendedTextMessage(); editedText != nil {
					body["edited_text"] = editedText.GetText()
				} else if editedConv := editedMessage.GetConversation(); editedConv != "" {
					body["edited_text"] = editedConv
				}
			}
		}
	}

	if audioMedia := evt.Message.GetAudioMessage(); audioMedia != nil {
		if downloadedMedia != nil && downloadedMedia.Audio != nil {
			// Use pre-downloaded media
			body["audio"] = *downloadedMedia.Audio
		} else if !config.WhatsappAutoDownloadMedia {
			// Auto-download disabled, return URL only
			body["audio"] = map[string]any{
				"url": audioMedia.GetURL(),
			}
		}
		// If auto-download enabled but no pre-downloaded media, skip (shouldn't happen)
	}

	if contactMessage := evt.Message.GetContactMessage(); contactMessage != nil {
		body["contact"] = contactMessage
	}

	if documentMedia := evt.Message.GetDocumentMessage(); documentMedia != nil {
		if downloadedMedia != nil && downloadedMedia.Document != nil {
			// Use pre-downloaded media
			body["document"] = *downloadedMedia.Document
		} else if !config.WhatsappAutoDownloadMedia {
			// Auto-download disabled, return URL only
			body["document"] = map[string]any{
				"url":      documentMedia.GetURL(),
				"filename": documentMedia.GetFileName(),
			}
		}
	}

	if imageMedia := evt.Message.GetImageMessage(); imageMedia != nil {
		if downloadedMedia != nil && downloadedMedia.Image != nil {
			// Use pre-downloaded media
			body["image"] = *downloadedMedia.Image
		} else if !config.WhatsappAutoDownloadMedia {
			// Auto-download disabled, return URL only
			body["image"] = map[string]any{
				"url":     imageMedia.GetURL(),
				"caption": imageMedia.GetCaption(),
			}
		}
	}

	if listMessage := evt.Message.GetListMessage(); listMessage != nil {
		body["list"] = listMessage
	}

	if liveLocationMessage := evt.Message.GetLiveLocationMessage(); liveLocationMessage != nil {
		body["live_location"] = liveLocationMessage
	}

	if locationMessage := evt.Message.GetLocationMessage(); locationMessage != nil {
		body["location"] = locationMessage
	}

	if orderMessage := evt.Message.GetOrderMessage(); orderMessage != nil {
		body["order"] = orderMessage
	}

	if stickerMedia := evt.Message.GetStickerMessage(); stickerMedia != nil {
		if downloadedMedia != nil && downloadedMedia.Sticker != nil {
			// Use pre-downloaded media
			body["sticker"] = *downloadedMedia.Sticker
		} else if !config.WhatsappAutoDownloadMedia {
			// Auto-download disabled, return URL only
			body["sticker"] = map[string]any{
				"url": stickerMedia.GetURL(),
			}
		}
	}

	if videoMedia := evt.Message.GetVideoMessage(); videoMedia != nil {
		if downloadedMedia != nil && downloadedMedia.Video != nil {
			// Use pre-downloaded media
			body["video"] = *downloadedMedia.Video
		} else if !config.WhatsappAutoDownloadMedia {
			// Auto-download disabled, return URL only
			body["video"] = map[string]any{
				"url":     videoMedia.GetURL(),
				"caption": videoMedia.GetCaption(),
			}
		}
	}

	// Handle PTV (Push-To-Video) messages - also known as "video notes" (circular video messages)
	if ptvMedia := evt.Message.GetPtvMessage(); ptvMedia != nil {
		if downloadedMedia != nil && downloadedMedia.VideoNote != nil {
			// Use pre-downloaded media
			body["video_note"] = *downloadedMedia.VideoNote
		} else if !config.WhatsappAutoDownloadMedia {
			// Auto-download disabled, return URL only
			body["video_note"] = map[string]any{
				"url":     ptvMedia.GetURL(),
				"caption": ptvMedia.GetCaption(),
			}
		}
	}

	return body, nil
}
