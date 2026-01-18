package whatsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func init() {
	// Initialize the package-level log for tests if not already set
	if log == nil {
		log = waLog.Stdout("Test", "WARN", true)
	}
}

func TestNormalizeJIDFromLIDWithContext_NonLIDPassthrough(t *testing.T) {
	// Non-LID JIDs should pass through unchanged
	jid := types.JID{User: "556796707788", Server: "s.whatsapp.net"}

	result := NormalizeJIDFromLIDWithContext(jid, nil)

	assert.Equal(t, jid, result)
}

func TestNormalizeJIDFromLIDWithContext_NilClientPassthrough(t *testing.T) {
	// LID JIDs with nil client should return original
	jid := types.JID{User: "215946727821336", Server: "lid"}

	result := NormalizeJIDFromLIDWithContext(jid, nil)

	assert.Equal(t, jid, result)
}
