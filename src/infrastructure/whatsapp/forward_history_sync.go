package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
)

// forwardHistorySyncCompleteToWebhook dispatches the fork-specific history_sync_complete
// event through upstream's webhook forwarder. Split into its own file per OQ3 decision
// in .workstreams/2026-05-14-upstream-v8.5-sync/04-plan.md § Slice 3 — keeping the fork's
// event-dispatch hook in a dedicated file makes the delta easier to audit during future
// upstream syncs and easier to retire if the event is ever upstreamed.
//
// Payload shape is locked by 03-structure.md § Webhook contract surface:
//
//	{
//	  "event":     "history_sync_complete",
//	  "device_id": "<non-AD JID>",
//	  "payload":   { "sync_type": "...", "timestamp": "<RFC3339>" }
//	}
func forwardHistorySyncCompleteToWebhook(ctx context.Context, client *whatsmeow.Client, syncType string) {
	deviceID := ""
	if client != nil && client.Store != nil && client.Store.ID != nil {
		deviceJID := NormalizeJIDFromLIDWithContext(client.Store.ID.ToNonAD(), client)
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
