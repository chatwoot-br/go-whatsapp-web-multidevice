package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// ensureLogger initializes the package-level log var if it has not been set by
// InitWaDB. Required for any test that exercises a code path which logs via
// `log.Warnf` / `log.Errorf` — the bare var declaration leaves it nil, which
// panics on first method call.
func ensureLogger() {
	if log == nil {
		log = waLog.Stdout("test", "ERROR", false)
	}
}

// TestNormalizeJIDFromLIDWithContext_PassthroughNonLID asserts that any JID
// whose server is NOT "lid" is returned verbatim — the function only does work
// for LID JIDs. Exercised against a nil client to also lock the "no resolution
// attempted on phone JIDs even with nil client" invariant.
func TestNormalizeJIDFromLIDWithContext_PassthroughNonLID(t *testing.T) {
	cases := []types.JID{
		types.NewJID("5511999999999", types.DefaultUserServer),
		types.NewJID("120363111111111", types.GroupServer),
		types.NewJID("120363222222222", types.NewsletterServer),
	}
	for _, in := range cases {
		t.Run(in.Server, func(t *testing.T) {
			got := NormalizeJIDFromLIDWithContext(in, nil)
			if got.User != in.User || got.Server != in.Server {
				t.Errorf("got %s, want %s (function must passthrough non-LID JIDs)", got.String(), in.String())
			}
		})
	}
}

// TestNormalizeJIDFromLIDWithContext_NilClient asserts the safety-check branch:
// LID JID + nil client must NOT panic and must return the original JID
// unchanged. (Production calls this from deduplicateLIDChats which is
// short-circuited when client is nil — but the inner function is defensive.)
func TestNormalizeJIDFromLIDWithContext_NilClient(t *testing.T) {
	ensureLogger()
	in := types.JID{User: "215946727821336", Server: "lid"}
	got := NormalizeJIDFromLIDWithContext(in, nil)
	if got.User != in.User || got.Server != in.Server {
		t.Errorf("got %s, want %s (nil client must return input unchanged)", got.String(), in.String())
	}
}
