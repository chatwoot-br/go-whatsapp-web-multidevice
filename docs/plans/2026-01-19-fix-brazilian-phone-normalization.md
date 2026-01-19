# Fix Brazilian Phone Normalization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix duplicate contacts caused by Brazilian phone number format differences (8-digit vs 9-digit) by using WhatsApp's normalized JID response instead of the original input.

**Architecture:** Modify `ValidateJidWithLogin` to return WhatsApp's canonical JID from the `IsOnWhatsAppResponse.JID` field. This ensures all downstream operations (LID lookup, message sending, storage) use the same normalized phone number that WhatsApp uses internally.

**Tech Stack:** Go 1.21+, whatsmeow library, testify for assertions

**Related Issue:** [ISSUE-005](../issues/ISSUE-005-brazilian-phone-normalization-duplicate-contacts.md)

---

## Task 1: Add ValidateAndNormalizeJID Function with Tests

**Files:**
- Modify: `src/pkg/utils/whatsapp.go` (after line 663)
- Modify: `src/pkg/utils/whatsapp_test.go`

**Step 1: Write the failing test**

Add to `src/pkg/utils/whatsapp_test.go`:

```go
import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestValidateAndNormalizeJID_GroupJIDPassthrough(t *testing.T) {
	// Group JIDs should pass through without modification
	jid := "120363123456789012@g.us"
	result, err := ValidateAndNormalizeJID(nil, jid)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := types.JID{User: "120363123456789012", Server: "g.us"}
	if result.User != expected.User || result.Server != expected.Server {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestValidateAndNormalizeJID_NonUserJIDPassthrough(t *testing.T) {
	// Newsletter and other non-user JIDs should pass through
	jid := "120363123456789012@newsletter"
	result, err := ValidateAndNormalizeJID(nil, jid)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Server != "newsletter" {
		t.Errorf("got server %s, want newsletter", result.Server)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd src && go test ./pkg/utils -run TestValidateAndNormalizeJID -v`
Expected: FAIL with "undefined: ValidateAndNormalizeJID"

**Step 3: Write minimal implementation**

Add to `src/pkg/utils/whatsapp.go` after `ValidateJidWithLogin` (line 663):

```go
// ValidateAndNormalizeJID validates JID and returns WhatsApp's normalized JID.
// For user JIDs (@s.whatsapp.net), it queries WhatsApp to get the canonical phone number.
// This handles cases like Brazilian numbers where 5566996679626 normalizes to 556696679626.
// For non-user JIDs (groups, newsletters), it returns the parsed JID unchanged.
func ValidateAndNormalizeJID(client *whatsmeow.Client, jid string) (types.JID, error) {
	// For non-user JIDs (groups, newsletters), skip normalization
	if !strings.Contains(jid, "@s.whatsapp.net") {
		return ParseJID(jid)
	}

	// If no client provided, fall back to simple parsing
	if client == nil {
		return ParseJID(jid)
	}

	MustLogin(client)

	// Extract phone number from JID
	phone := strings.TrimSuffix(jid, "@s.whatsapp.net")
	if phone == "" {
		return types.JID{}, pkgError.InvalidJID("Empty phone number")
	}

	// whatsmeow expects international format with + prefix
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}

	// Query WhatsApp for the canonical JID
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := client.IsOnWhatsApp(ctx, []string{phone})
	if err != nil {
		logrus.Warnf("Failed to query WhatsApp for %s: %v", jid, err)
		// Fall back to original JID if query fails
		if config.WhatsappAccountValidation {
			return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Failed to validate phone %s: %v", jid, err))
		}
		return ParseJID(jid)
	}

	// Empty response means number not found
	if len(data) == 0 {
		if config.WhatsappAccountValidation {
			return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Phone %s is not on WhatsApp", jid))
		}
		return ParseJID(jid)
	}

	// Check results and return normalized JID
	for _, v := range data {
		if !v.IsIn {
			if config.WhatsappAccountValidation {
				return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Phone %s is not on WhatsApp", jid))
			}
			return ParseJID(jid)
		}

		// Return WhatsApp's canonical JID (normalized phone number)
		if !v.JID.IsEmpty() {
			logrus.Debugf("Normalized JID %s to %s", jid, v.JID.String())
			return v.JID, nil
		}
	}

	// Fallback to original
	return ParseJID(jid)
}
```

**Step 4: Run test to verify it passes**

Run: `cd src && go test ./pkg/utils -run TestValidateAndNormalizeJID -v`
Expected: PASS

**Step 5: Commit**

```bash
git add src/pkg/utils/whatsapp.go src/pkg/utils/whatsapp_test.go
git commit -m "feat(utils): add ValidateAndNormalizeJID for phone normalization

Adds a new function that queries WhatsApp to get the canonical phone
number. This handles Brazilian numbers where 5566996679626 normalizes
to 556696679626, preventing duplicate contacts.

Related: ISSUE-005"
```

---

## Task 2: Update SendText to Use Normalized JID

**Files:**
- Modify: `src/usecase/send.go` (line 100)

**Step 1: Read current implementation**

Current code at line 100:
```go
dataWaRecipient, err := utils.ValidateJidWithLogin(client, request.BaseRequest.Phone)
```

**Step 2: Update to use normalized JID**

Change line 100 in `src/usecase/send.go`:

```go
dataWaRecipient, err := utils.ValidateAndNormalizeJID(client, request.BaseRequest.Phone)
```

**Step 3: Run tests to verify no regression**

Run: `cd src && go test ./usecase -v`
Expected: PASS (or skip if no usecase tests exist)

**Step 4: Run build to verify compilation**

Run: `cd src && go build -o /dev/null .`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add src/usecase/send.go
git commit -m "fix(send): use normalized JID for SendText

Uses ValidateAndNormalizeJID instead of ValidateJidWithLogin to ensure
Brazilian phone numbers are normalized before LID lookup.

Related: ISSUE-005"
```

---

## Task 3: Update SendImage to Use Normalized JID

**Files:**
- Modify: `src/usecase/message.go` (around line 96)

**Step 1: Identify the line to change**

Find: `utils.ValidateJidWithLogin(client, request.BaseRequest.Phone)`

**Step 2: Update to use normalized JID**

Change to:
```go
dataWaRecipient, err := utils.ValidateAndNormalizeJID(client, request.BaseRequest.Phone)
```

**Step 3: Run build to verify compilation**

Run: `cd src && go build -o /dev/null .`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add src/usecase/message.go
git commit -m "fix(message): use normalized JID for SendImage

Related: ISSUE-005"
```

---

## Task 4: Update SendFile to Use Normalized JID

**Files:**
- Modify: `src/usecase/message.go` (around line 123)

**Step 1: Identify the line to change**

Find the second occurrence of: `utils.ValidateJidWithLogin(client, request.BaseRequest.Phone)`

**Step 2: Update to use normalized JID**

Change to:
```go
dataWaRecipient, err := utils.ValidateAndNormalizeJID(client, request.BaseRequest.Phone)
```

**Step 3: Run build to verify compilation**

Run: `cd src && go build -o /dev/null .`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add src/usecase/message.go
git commit -m "fix(message): use normalized JID for SendFile

Related: ISSUE-005"
```

---

## Task 5: Update SendVideo to Use Normalized JID

**Files:**
- Modify: `src/usecase/message.go` (around line 191)

**Step 1: Identify the line to change**

Find the third occurrence of: `utils.ValidateJidWithLogin(client, request.BaseRequest.Phone)`

**Step 2: Update to use normalized JID**

Change to:
```go
dataWaRecipient, err := utils.ValidateAndNormalizeJID(client, request.BaseRequest.Phone)
```

**Step 3: Run build to verify compilation**

Run: `cd src && go build -o /dev/null .`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add src/usecase/message.go
git commit -m "fix(message): use normalized JID for SendVideo

Related: ISSUE-005"
```

---

## Task 6: Update Remaining Callers of ValidateJidWithLogin

**Files:**
- Search and update all remaining callers

**Step 1: Find all callers**

Run: `cd src && grep -rn "ValidateJidWithLogin" --include="*.go"`

**Step 2: Update each caller**

For each remaining occurrence, change:
```go
utils.ValidateJidWithLogin(client, phone)
```
to:
```go
utils.ValidateAndNormalizeJID(client, phone)
```

**Step 3: Run full test suite**

Run: `cd src && go test ./... -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add -A
git commit -m "fix: update all callers to use ValidateAndNormalizeJID

Ensures consistent phone normalization across all message sending
operations.

Related: ISSUE-005"
```

---

## Task 7: Update Issue Documentation

**Files:**
- Modify: `docs/issues/ISSUE-005-brazilian-phone-normalization-duplicate-contacts.md`

**Step 1: Update status and add fix details**

Update the issue document:
- Change status from "Open" to "Fixed"
- Add "Fix Implementation" section with commit SHAs
- Add verification steps

**Step 2: Commit**

```bash
git add docs/issues/ISSUE-005-brazilian-phone-normalization-duplicate-contacts.md
git commit -m "docs: update ISSUE-005 status to Fixed"
```

---

## Task 8: Final Integration Test

**Step 1: Build the application**

Run: `cd src && go build -o whatsapp .`
Expected: Build succeeds

**Step 2: Run all tests**

Run: `cd src && go test ./... -v`
Expected: All tests pass

**Step 3: Manual verification steps**

1. Start the application: `cd src && ./whatsapp rest`
2. Connect a WhatsApp account
3. Send a message to a Brazilian number with 9-digit format (e.g., `5566996679626`)
4. Check logs for: `Normalized JID 5566996679626@s.whatsapp.net to 556696679626@s.whatsapp.net`
5. Verify the message is sent to the correct LID

---

## Files Changed Summary

| File | Change |
|------|--------|
| `src/pkg/utils/whatsapp.go` | Add `ValidateAndNormalizeJID` function |
| `src/pkg/utils/whatsapp_test.go` | Add tests for new function |
| `src/usecase/send.go` | Use `ValidateAndNormalizeJID` in `SendText` |
| `src/usecase/message.go` | Use `ValidateAndNormalizeJID` in `SendImage`, `SendFile`, `SendVideo` |
| `docs/issues/ISSUE-005-*.md` | Update status to Fixed |
