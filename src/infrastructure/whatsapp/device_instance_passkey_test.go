package whatsapp

import "testing"

// PairPasskeyConfirmation can land while SendPasskeyResponse is still in flight. Clearing
// the whole passkey state after submitting the assertion would then discard the
// confirmation code that has ALREADY arrived, leaving the user with no pending status and
// no code for /app/passkey/confirm. Retiring the challenge must not touch the code.
func TestClearPasskeyChallenge_PreservesAnAlreadyArrivedConfirmation(t *testing.T) {
	inst := &DeviceInstance{id: "dev-passkey"}

	// The confirmation raced ahead of the submit call returning.
	inst.SetPasskeyConfirmation("123-456", false)

	inst.ClearPasskeyChallenge()

	challenge, code, skip := inst.PasskeyState()
	if challenge != nil {
		t.Fatal("expected the consumed challenge to be cleared")
	}
	if code != "123-456" {
		t.Fatalf("expected the already-arrived confirmation code to survive, got %q", code)
	}
	if skip {
		t.Fatal("expected the skip-handoff flag to be preserved as stored")
	}

	// ClearPasskeyState still wipes everything, for the flows that want that.
	inst.ClearPasskeyState()
	if _, code, _ := inst.PasskeyState(); code != "" {
		t.Fatalf("expected ClearPasskeyState to reset the code, got %q", code)
	}
}
