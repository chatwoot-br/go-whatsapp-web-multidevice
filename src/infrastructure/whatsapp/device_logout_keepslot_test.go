package whatsapp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
)

// keepSlotStubStorage is a minimal IChatStorageRepository that records the device
// registry calls exercised by the logout/purge paths and can be told to fail, so the
// tests can assert error propagation. All other interface methods are nil (embedded)
// and must not be called by the code under test.
type keepSlotStubStorage struct {
	domainChatStorage.IChatStorageRepository
	saveErr        error
	deleteDataErr  error
	savedRecords   []*domainChatStorage.DeviceRecord
	deletedData    []string
	deletedRecords []string
	lastJIDs       map[string]string
	// recordJIDs is the persisted `jid` column, distinct from lastJIDs (`last_jid`).
	// A logout that failed before persisting the reset leaves the two out of step:
	// jid still set, last_jid empty.
	recordJIDs map[string]string
}

func (s *keepSlotStubStorage) SaveDeviceRecord(rec *domainChatStorage.DeviceRecord) error {
	cloned := *rec
	s.savedRecords = append(s.savedRecords, &cloned)
	if s.saveErr != nil {
		// A failed write must not move the persisted state — that is exactly the
		// half-applied logout the retry has to recover from.
		return s.saveErr
	}
	if s.recordJIDs == nil {
		s.recordJIDs = map[string]string{}
	}
	s.recordJIDs[rec.DeviceID] = rec.JID
	return nil
}

func (s *keepSlotStubStorage) DeleteDeviceData(deviceID string) error {
	s.deletedData = append(s.deletedData, deviceID)
	return s.deleteDataErr
}

func (s *keepSlotStubStorage) DeleteDeviceRecord(deviceID string) error {
	s.deletedRecords = append(s.deletedRecords, deviceID)
	return nil
}

func (s *keepSlotStubStorage) SetDeviceLastJID(deviceID, lastJID string) error {
	if s.lastJIDs == nil {
		s.lastJIDs = map[string]string{}
	}
	s.lastJIDs[deviceID] = lastJID
	return nil
}

func (s *keepSlotStubStorage) GetDeviceRecord(deviceID string) (*domainChatStorage.DeviceRecord, error) {
	return &domainChatStorage.DeviceRecord{
		DeviceID: deviceID,
		JID:      s.recordJIDs[deviceID],
		LastJID:  s.lastJIDs[deviceID],
	}, nil
}

func (s *keepSlotStubStorage) ListDeviceRecords() ([]*domainChatStorage.DeviceRecord, error) {
	seen := map[string]bool{}
	var out []*domainChatStorage.DeviceRecord
	for id := range s.recordJIDs {
		seen[id] = true
	}
	for id := range s.lastJIDs {
		seen[id] = true
	}
	for id := range seen {
		out = append(out, &domainChatStorage.DeviceRecord{
			DeviceID: id,
			JID:      s.recordJIDs[id],
			LastJID:  s.lastJIDs[id],
		})
	}
	return out, nil
}

// assertStoreLacksJID fails if any device row in the container still matches the given
// NonAD JID. Matching mirrors deleteStoreRowsForJID / LoadExistingDevices.
func assertStoreLacksJID(t *testing.T, ctx context.Context, c *sqlstore.Container, nonADJID string) {
	t.Helper()
	devices, err := c.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("get all devices: %v", err)
	}
	for _, d := range devices {
		if d != nil && d.ID != nil && d.ID.ToNonAD().String() == nonADJID {
			t.Fatalf("expected store to no longer contain jid %s", nonADJID)
		}
	}
}

// Scenario: keep-slot logout for a slot that was loaded from storage with NO live
// client (the case aldinokemal flagged at device_manager.go:240). The orphan whatsmeow
// rows must be deleted by JID from BOTH the primary and the separate keys container,
// while the slot itself (id + display name) is preserved with an empty persisted JID.
// The slot id is deliberately different from the JID to prove matching is by JID, not
// by the slot id (the latent AD-JID-vs-slot-id mismatch in the old code).
func TestLogoutDeviceKeepSlot_NoClientDeletesStoreRowsByJIDKeepsSlot(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)
	keysStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999991", types.WhatsAppDomain, 10)
	nonAD := adJID.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}
	if err := newTestStoreDevice(keysStore, adJID, "keys").Save(ctx); err != nil {
		t.Fatalf("save keys device: %v", err)
	}

	storage := &keepSlotStubStorage{}
	manager := NewDeviceManager(primaryStore, keysStore, storage)

	const slotID = "slot-uuid-1" // intentionally != JID
	inst := &DeviceInstance{id: slotID, jid: nonAD, displayName: "tIAtendo", createdAt: time.Now()}
	manager.devices[slotID] = inst

	if err := manager.LogoutDeviceKeepSlot(ctx, slotID); err != nil {
		t.Fatalf("LogoutDeviceKeepSlot returned error: %v", err)
	}

	assertStoreLacksJID(t, ctx, primaryStore, nonAD)
	assertStoreLacksJID(t, ctx, keysStore, nonAD)

	if _, ok := manager.GetDevice(slotID); !ok {
		t.Fatal("expected slot to be kept after logout")
	}
	if got := inst.JID(); got != "" {
		t.Fatalf("expected instance JID cleared after logout, got %q", got)
	}
	if len(storage.savedRecords) == 0 {
		t.Fatal("expected SaveDeviceRecord to persist the kept slot")
	}
	last := storage.savedRecords[len(storage.savedRecords)-1]
	if last.DeviceID != slotID || last.JID != "" {
		t.Fatalf("expected persisted slot %s with empty JID, got %s / %q", slotID, last.DeviceID, last.JID)
	}
}

// Scenario: DELETE purge removes the whatsmeow rows by JID from both containers even
// when the slot id differs from the JID (the old code compared the AD JID string to the
// slot id and never matched, so the rows were never deleted), and removes the slot.
func TestPurgeDevice_DeletesStoreRowsByJIDAcrossPrimaryAndKeys(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)
	keysStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999992", types.WhatsAppDomain, 11)
	nonAD := adJID.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}
	if err := newTestStoreDevice(keysStore, adJID, "keys").Save(ctx); err != nil {
		t.Fatalf("save keys device: %v", err)
	}

	storage := &keepSlotStubStorage{}
	manager := NewDeviceManager(primaryStore, keysStore, storage)

	const slotID = "slot-uuid-2"
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: nonAD, displayName: "tIAtendo", createdAt: time.Now()}

	if err := manager.PurgeDevice(ctx, slotID); err != nil {
		t.Fatalf("PurgeDevice returned error: %v", err)
	}

	assertStoreLacksJID(t, ctx, primaryStore, nonAD)
	assertStoreLacksJID(t, ctx, keysStore, nonAD)

	if _, ok := manager.GetDevice(slotID); ok {
		t.Fatal("expected slot to be removed after purge")
	}
	if len(storage.deletedData) == 0 || storage.deletedData[0] != slotID {
		t.Fatalf("expected chatstorage data deletion for %s, got %v", slotID, storage.deletedData)
	}
	if len(storage.deletedRecords) == 0 || storage.deletedRecords[0] != slotID {
		t.Fatalf("expected device record deletion for %s, got %v", slotID, storage.deletedRecords)
	}
}

// resetDeviceKeepSlot must surface a persistence failure instead of silently
// succeeding (coderabbitai device_manager.go:252-276), and LogoutDeviceKeepSlot must
// combine that error into its return value.
func TestResetDeviceKeepSlot_PropagatesSaveError(t *testing.T) {
	ctx := context.Background()
	storage := &keepSlotStubStorage{saveErr: errors.New("disk full")}
	manager := NewDeviceManager(nil, nil, storage)

	const slotID = "slot-persist-err"
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: "6281999999993@s.whatsapp.net", createdAt: time.Now()}

	if err := manager.resetDeviceKeepSlot(slotID, "6281999999993@s.whatsapp.net"); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected resetDeviceKeepSlot to propagate save error, got %v", err)
	}

	if err := manager.LogoutDeviceKeepSlot(ctx, slotID); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected LogoutDeviceKeepSlot to surface the persistence error, got %v", err)
	}
}

// DELETE promises a real purge: a LOCAL cleanup failure (chatstorage/store/keys) must
// be surfaced, not masked as success (aldinokemal/coderabbitai usecase device.go:68).
func TestPurgeDevice_SurfacesLocalCleanupFailure(t *testing.T) {
	ctx := context.Background()
	storage := &keepSlotStubStorage{deleteDataErr: errors.New("chatstorage down")}
	manager := NewDeviceManager(nil, nil, storage)

	const slotID = "slot-cleanup-err"
	manager.devices[slotID] = &DeviceInstance{id: slotID, createdAt: time.Now()}

	if err := manager.PurgeDevice(ctx, slotID); err == nil || !strings.Contains(err.Error(), "chatstorage down") {
		t.Fatalf("expected PurgeDevice to surface the local cleanup failure, got %v", err)
	}

	// On partial failure the slot must survive: it carries the id↔jid mapping a
	// retry needs to find the surviving rows. Removing it would make the returned
	// error unactionable (the caller could never purge the leftovers again).
	if _, ok := manager.GetDevice(slotID); !ok {
		t.Fatal("expected slot to be kept after a failed purge so the delete can be retried")
	}
	if len(storage.deletedRecords) != 0 {
		t.Fatalf("expected device record to be kept after a failed purge, got deletions: %v", storage.deletedRecords)
	}
}

// Scenario: keep-slot logout retains the JID-scoped chat history by design, then a
// later DELETE must still purge it. The logout clears the live jid but records it as
// last_jid; the purge must read it back and delete the chat data under BOTH the slot
// id and that retained JID (plus any whatsmeow rows left for it).
func TestPurgeDevice_AfterKeepSlotLogout_DeletesRetainedJIDData(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999995", types.WhatsAppDomain, 13)
	nonAD := adJID.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}

	storage := &keepSlotStubStorage{}
	manager := NewDeviceManager(primaryStore, nil, storage)

	const slotID = "slot-uuid-3"
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: nonAD, displayName: "tIAtendo", createdAt: time.Now()}

	if err := manager.LogoutDeviceKeepSlot(ctx, slotID); err != nil {
		t.Fatalf("LogoutDeviceKeepSlot returned error: %v", err)
	}
	if got := storage.lastJIDs[slotID]; got != nonAD {
		t.Fatalf("expected logout to persist last_jid %s, got %q", nonAD, got)
	}

	if err := manager.PurgeDevice(ctx, slotID); err != nil {
		t.Fatalf("PurgeDevice returned error: %v", err)
	}

	deleted := map[string]bool{}
	for _, key := range storage.deletedData {
		deleted[key] = true
	}
	if !deleted[slotID] || !deleted[nonAD] {
		t.Fatalf("expected chat data deleted under both slot id %s and retained jid %s, got %v", slotID, nonAD, storage.deletedData)
	}
	if _, ok := manager.GetDevice(slotID); ok {
		t.Fatal("expected slot removed after purge")
	}
}

// Scenario: a first logout whose store cleanup failed already cleared the in-memory JID
// but recorded it as last_jid. The RETRY arrives with an empty inst.JID(), so without a
// fallback it would delete nothing, report success, and leave the orphan whatsmeow row
// that LoadExistingDevices resurrects on restart. The retry must recover the identity
// from last_jid and actually delete the rows.
func TestKeepSlotLogout_RetryRecoversJIDFromLastJID(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)
	keysStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999998", types.WhatsAppDomain, 16)
	nonAD := adJID.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}
	if err := newTestStoreDevice(keysStore, adJID, "keys").Save(ctx); err != nil {
		t.Fatalf("save keys device: %v", err)
	}

	// State left behind by the failed first attempt: slot kept, live JID cleared,
	// identity retained only in last_jid, store rows still present.
	storage := &keepSlotStubStorage{lastJIDs: map[string]string{"slot-retry": nonAD}}
	manager := NewDeviceManager(primaryStore, keysStore, storage)

	const slotID = "slot-retry"
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: "", displayName: "tIAtendo", createdAt: time.Now()}

	if err := manager.LogoutDeviceKeepSlot(ctx, slotID); err != nil {
		t.Fatalf("logout retry returned error: %v", err)
	}

	assertStoreLacksJID(t, ctx, primaryStore, nonAD)
	assertStoreLacksJID(t, ctx, keysStore, nonAD)

	if _, ok := manager.GetDevice(slotID); !ok {
		t.Fatal("expected slot to survive the logout retry")
	}
	if got := storage.lastJIDs[slotID]; got != nonAD {
		t.Fatalf("expected last_jid %s to be retained across the retry, got %q", nonAD, got)
	}
}

// Scenario: the first logout deletes the store rows and then FAILS while persisting the
// reset. ResetClient has already cleared the in-memory JID, but the devices row still has
// `jid` set and `last_jid` empty — SetDeviceLastJID is never reached. A retry that looked
// only at last_jid would recover nothing, then clear the row's jid on its way out,
// stranding that account's chat data with no pointer left to name it. The retry must fall
// back to the record's live jid and record it as last_jid.
//
// Driven through a real failed logout rather than a hand-built state, so the two steps
// have to actually agree about what a half-applied logout leaves behind.
func TestKeepSlotLogout_RetryRecoversJIDFromRecordWhenResetNeverPersisted(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999999", types.WhatsAppDomain, 17)
	nonAD := adJID.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}

	const slotID = "slot-reset-failed"
	storage := &keepSlotStubStorage{
		saveErr:    errors.New("disk full"),
		recordJIDs: map[string]string{slotID: nonAD}, // the paired row as persisted
	}
	manager := NewDeviceManager(primaryStore, nil, storage)
	inst := &DeviceInstance{id: slotID, jid: nonAD, displayName: "tIAtendo", createdAt: time.Now()}
	manager.devices[slotID] = inst

	// First attempt: dies persisting the reset, after ResetClient cleared the JID.
	if err := manager.LogoutDeviceKeepSlot(ctx, slotID); err == nil {
		t.Fatal("expected the first logout to fail on the persistence error")
	}
	if got := inst.JID(); got != "" {
		t.Fatalf("expected the in-memory JID to be cleared by the failed attempt, got %q", got)
	}
	if storage.lastJIDs[slotID] != "" {
		t.Fatalf("precondition: last_jid must NOT have been recorded by the failed attempt, got %q", storage.lastJIDs[slotID])
	}

	// Retry, now with storage healthy. The only surviving pointer to the account is the
	// record's `jid` column.
	storage.saveErr = nil
	if err := manager.LogoutDeviceKeepSlot(ctx, slotID); err != nil {
		t.Fatalf("logout retry returned error: %v", err)
	}

	assertStoreLacksJID(t, ctx, primaryStore, nonAD)
	if got := storage.lastJIDs[slotID]; got != nonAD {
		t.Fatalf("expected the retry to retain %s as last_jid so a later purge can find its chat data, got %q", nonAD, got)
	}
	if _, ok := manager.GetDevice(slotID); !ok {
		t.Fatal("expected the slot to survive the logout retry")
	}
}

// Scenario: the slot is logged out of account A (last_jid=A), re-paired to account B,
// then logged out again. last_jid holds ONE identity, so recording B would overwrite A
// and strand A's retained chat data — no later purge could still name it. A's data must
// be dropped as the pointer to it is replaced.
func TestResetDeviceKeepSlot_PurgesSupersededRetainedJID(t *testing.T) {
	const slotID = "slot-repaired"
	const jidA = "6281999999801@s.whatsapp.net"
	const jidB = "6281999999802@s.whatsapp.net"

	storage := &keepSlotStubStorage{lastJIDs: map[string]string{slotID: jidA}}
	manager := NewDeviceManager(nil, nil, storage)
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: jidB, displayName: "tIAtendo", createdAt: time.Now()}

	if err := manager.resetDeviceKeepSlot(slotID, jidB); err != nil {
		t.Fatalf("resetDeviceKeepSlot returned error: %v", err)
	}

	deleted := map[string]bool{}
	for _, key := range storage.deletedData {
		deleted[key] = true
	}
	if !deleted[jidA] {
		t.Fatalf("expected chat data of the superseded account %s to be purged, got %v", jidA, storage.deletedData)
	}
	if deleted[jidB] {
		t.Fatalf("the newly retained account %s must keep its data (logout retains history), got %v", jidB, storage.deletedData)
	}
	if got := storage.lastJIDs[slotID]; got != jidB {
		t.Fatalf("expected last_jid to advance to %s, got %q", jidB, got)
	}
}

// The superseded-JID purge must not fire on the ordinary path: re-recording the SAME
// retained JID (a logout retry) keeps that account's history, which keep-slot logout
// exists to preserve.
func TestResetDeviceKeepSlot_SameRetainedJIDKeepsData(t *testing.T) {
	const slotID = "slot-same-jid"
	const jid = "6281999999803@s.whatsapp.net"

	storage := &keepSlotStubStorage{lastJIDs: map[string]string{slotID: jid}}
	manager := NewDeviceManager(nil, nil, storage)
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: jid, createdAt: time.Now()}

	if err := manager.resetDeviceKeepSlot(slotID, jid); err != nil {
		t.Fatalf("resetDeviceKeepSlot returned error: %v", err)
	}

	if len(storage.deletedData) != 0 {
		t.Fatalf("expected no chat data deletion when the retained JID is unchanged, got %v", storage.deletedData)
	}
}

// Scenario: the re-pair (A -> B) logout fails midway — purging A's superseded data errors
// out. The row must never be left naming NO account: if `jid` were cleared before `last_jid`
// advanced to B, the row would hold jid=” + last_jid=A, and the retry (jid first, then
// last_jid) would recover A — permanently losing the pointer to B's retained chat data.
// Clearing the live jid LAST means every intermediate failure leaves B recoverable.
func TestResetDeviceKeepSlot_FailedSupersededPurgeKeepsCurrentJIDRecoverable(t *testing.T) {
	const slotID = "slot-purge-fails"
	const jidA = "6281999999811@s.whatsapp.net"
	const jidB = "6281999999812@s.whatsapp.net"

	storage := &keepSlotStubStorage{
		lastJIDs:      map[string]string{slotID: jidA},
		recordJIDs:    map[string]string{slotID: jidB},
		deleteDataErr: errors.New("chatstorage down"), // purging A's data fails
	}
	manager := NewDeviceManager(nil, nil, storage)
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: jidB, createdAt: time.Now()}

	if err := manager.resetDeviceKeepSlot(slotID, jidB); err == nil {
		t.Fatal("expected the superseded-purge failure to be surfaced")
	}

	// The row must still name B (the live account), so the retry can recover it.
	record, err := storage.GetDeviceRecord(slotID)
	if err != nil {
		t.Fatalf("GetDeviceRecord: %v", err)
	}
	if record.JID != jidB {
		t.Fatalf("expected the live jid %s to survive the failed logout (retry recovers from it), got %q", jidB, record.JID)
	}
	if record.LastJID == jidB {
		t.Fatal("last_jid must not advance to B while A's data purge is still failing")
	}
}

// Scenario: DELETE addressed by WhatsApp JID, for a slot that was already logged out. The
// slot's live JID is empty, so it can only be found via last_jid. Without that lookup the
// purge deletes chat data but removes devices[<jid>] / the <jid> record — neither of which
// exists — and reports success while the real uuid slot stays listed and persisted.
func TestPurgeDevice_ByJID_ResolvesLoggedOutSlotViaLastJID(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999820", types.WhatsAppDomain, 18)
	nonAD := adJID.ToNonAD().String()

	const slotID = "slot-logged-out-uuid"
	// Post-logout state: slot kept under its uuid, live jid cleared, account in last_jid.
	storage := &keepSlotStubStorage{
		lastJIDs:   map[string]string{slotID: nonAD},
		recordJIDs: map[string]string{slotID: ""},
	}
	manager := NewDeviceManager(primaryStore, nil, storage)
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: "", displayName: "tIAtendo", createdAt: time.Now()}

	// The caller deletes by the JID the slot used to hold.
	if err := manager.PurgeDevice(ctx, nonAD); err != nil {
		t.Fatalf("PurgeDevice by jid returned error: %v", err)
	}

	if _, ok := manager.GetDevice(slotID); ok {
		t.Fatal("expected the uuid slot to be removed — DELETE by JID must not report success while the slot survives")
	}
	deletedRecords := map[string]bool{}
	for _, id := range storage.deletedRecords {
		deletedRecords[id] = true
	}
	if !deletedRecords[slotID] {
		t.Fatalf("expected the device record for slot %s to be deleted, got %v", slotID, storage.deletedRecords)
	}
	deletedData := map[string]bool{}
	for _, key := range storage.deletedData {
		deletedData[key] = true
	}
	if !deletedData[nonAD] {
		t.Fatalf("expected chat data under the retained jid %s to be purged, got %v", nonAD, storage.deletedData)
	}
}

// deleteStoreRowsForJID is a no-op for an empty JID: a slot that was never paired has no
// store rows, and an empty JID must never scan/delete anything.
func TestDeleteStoreRowsForJID_EmptyJIDIsNoOp(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)
	adJID := types.NewADJID("6281999999994", types.WhatsAppDomain, 12)
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}

	manager := NewDeviceManager(primaryStore, nil, nil)
	if err := manager.deleteStoreRowsForJID(ctx, ""); err != nil {
		t.Fatalf("expected empty-jid no-op, got error: %v", err)
	}

	devices, err := primaryStore.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("get all devices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected the unrelated device to be retained, got %d devices", len(devices))
	}
}

// Scenario: the remote-logout callback holds a stale device id. InitWaCLI registers the
// callback with the instance's AD JID string (device.ID.String()), but loadFromRegistry
// can replace that instance with a named registry slot keyed by uuid (same account,
// NonAD JID). keepSlotLogout must then fall back to resolving the slot by JID, so the
// cleanup still deletes the store rows and clears the slot instead of no-opping with
// "device not found" (leaving a stale JID + orphan keys row).
func TestKeepSlotLogout_StaleADJIDFallsBackToJIDResolution(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)
	keysStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999996", types.WhatsAppDomain, 14)
	nonAD := adJID.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}
	if err := newTestStoreDevice(keysStore, adJID, "keys").Save(ctx); err != nil {
		t.Fatalf("save keys device: %v", err)
	}

	storage := &keepSlotStubStorage{}
	manager := NewDeviceManager(primaryStore, keysStore, storage)

	// The registry slot that replaced the startup instance: keyed by uuid, NonAD JID.
	const slotID = "slot-uuid-3"
	inst := &DeviceInstance{id: slotID, jid: nonAD, displayName: "tIAtendo", createdAt: time.Now()}
	manager.devices[slotID] = inst

	// The stale id the startup path registered the callback with: the AD JID string.
	if err := manager.keepSlotLogout(ctx, adJID.String()); err != nil {
		t.Fatalf("keepSlotLogout with stale AD JID id returned error: %v", err)
	}

	assertStoreLacksJID(t, ctx, primaryStore, nonAD)
	assertStoreLacksJID(t, ctx, keysStore, nonAD)

	if _, ok := manager.GetDevice(slotID); !ok {
		t.Fatal("expected slot to be kept after stale-id logout")
	}
	if got := inst.JID(); got != "" {
		t.Fatalf("expected instance JID cleared after stale-id logout, got %q", got)
	}
	if len(storage.savedRecords) == 0 {
		t.Fatal("expected SaveDeviceRecord to persist the kept slot")
	}
	last := storage.savedRecords[len(storage.savedRecords)-1]
	if last.DeviceID != slotID || last.JID != "" {
		t.Fatalf("expected persisted slot %s with empty JID, got %s / %q", slotID, last.DeviceID, last.JID)
	}
}

// Scenario: explicit logout for a paired client that is currently DISCONNECTED.
// IsLoggedIn() is only true while connected, so gating the unlink on it silently skips
// the remove-companion-device IQ and the phone keeps showing the linked device forever
// (the local session is deleted, so it can never reconnect to unlink later). The unlink
// must be ATTEMPTED whenever the client is paired (Store.ID set); failure stays
// best-effort and must not fail the local keep-slot cleanup.
func TestLogoutDeviceKeepSlot_AttemptsUnlinkWhenPairedButDisconnected(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999997", types.WhatsAppDomain, 15)
	nonAD := adJID.ToNonAD().String()
	dev := newTestStoreDevice(primaryStore, adJID, "primary")
	if err := dev.Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}

	// Real paired client (Store.ID set) that is not connected.
	cli := whatsmeow.NewClient(dev, nil)

	storage := &keepSlotStubStorage{}
	manager := NewDeviceManager(primaryStore, nil, storage)

	const slotID = "slot-uuid-4"
	inst := &DeviceInstance{id: slotID, jid: nonAD, displayName: "tIAtendo", client: cli, createdAt: time.Now()}
	manager.devices[slotID] = inst

	hook := logrustest.NewLocal(logrus.StandardLogger())
	defer hook.Reset()

	if err := manager.LogoutDeviceKeepSlot(ctx, slotID); err != nil {
		t.Fatalf("LogoutDeviceKeepSlot returned error: %v", err)
	}

	// The unlink attempt fails offline (best-effort), which is observable as the
	// remote-unlink warning. No warning means the attempt was skipped entirely.
	attempted := false
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "remote unlink failed") {
			attempted = true
		}
	}
	if !attempted {
		t.Fatal("expected cli.Logout to be attempted for a paired-but-disconnected client")
	}

	// Local keep-slot cleanup must still have run.
	assertStoreLacksJID(t, ctx, primaryStore, nonAD)
	if _, ok := manager.GetDevice(slotID); !ok {
		t.Fatal("expected slot to be kept after logout")
	}
	if got := inst.JID(); got != "" {
		t.Fatalf("expected instance JID cleared after logout, got %q", got)
	}
}

// Remote logout (events.LoggedOut) keeps the slot: routing the SetOnLoggedOut callback
// through keepSlotLogout (the same behavior both the lazy EnsureClient path and the
// startup InitWaCLI path now use) deletes the whatsmeow rows by JID but preserves the
// slot, instead of deleting it via RemoveDevice (aldinokemal device_manager.go:581).
func TestRemoteLogoutCallback_KeepsSlot(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)
	keysStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999995", types.WhatsAppDomain, 13)
	nonAD := adJID.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save primary device: %v", err)
	}
	if err := newTestStoreDevice(keysStore, adJID, "keys").Save(ctx); err != nil {
		t.Fatalf("save keys device: %v", err)
	}

	storage := &keepSlotStubStorage{}
	manager := NewDeviceManager(primaryStore, keysStore, storage)

	inst := &DeviceInstance{id: nonAD, jid: nonAD, displayName: "tIAtendo", createdAt: time.Now()}
	manager.devices[nonAD] = inst

	// Wire the callback exactly as the manager does for remote logout.
	inst.SetOnLoggedOut(func(deviceID string) {
		if err := manager.keepSlotLogout(context.Background(), deviceID); err != nil {
			t.Errorf("keepSlotLogout in remote-logout callback: %v", err)
		}
	})

	inst.TriggerLoggedOut()

	if _, ok := manager.GetDevice(nonAD); !ok {
		t.Fatal("expected slot to be kept on remote logout, but it was removed")
	}
	assertStoreLacksJID(t, ctx, primaryStore, nonAD)
	assertStoreLacksJID(t, ctx, keysStore, nonAD)
}

// Scenario: a slot logged out of account A is kept with an empty live JID. On the next
// boot the store still holds a row for an UNRELATED account B (a legacy/store-only
// session). The orphan-adoption path treats any empty-JID slot as a placeholder, so it
// would bind A's logged-out slot to B — after which every reconnect/logout/delete on that
// slot acts on the wrong WhatsApp account. A slot with a retained last_jid is NOT a
// placeholder and must never adopt another account's row.
func TestLoadExistingDevices_LoggedOutSlotIsNotAdoptedByAnUnrelatedAccount(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)

	// Account B: a store row with no slot of its own.
	adB := types.NewADJID("6281999999840", types.WhatsAppDomain, 20)
	nonADB := adB.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adB, "primary").Save(ctx); err != nil {
		t.Fatalf("save store device for B: %v", err)
	}

	const slotID = "slot-logged-out-of-A"
	const jidA = "6281999999830@s.whatsapp.net"

	// Account A's slot, already logged out: live jid cleared, identity kept in last_jid.
	storage := &keepSlotStubStorage{
		lastJIDs:   map[string]string{slotID: jidA},
		recordJIDs: map[string]string{slotID: ""},
	}
	manager := NewDeviceManager(primaryStore, nil, storage)

	if err := manager.LoadExistingDevices(ctx); err != nil {
		t.Fatalf("LoadExistingDevices: %v", err)
	}

	inst, ok := manager.GetDevice(slotID)
	if !ok || inst == nil {
		t.Fatal("expected the logged-out slot to survive the load")
	}
	if got := inst.JID(); got != "" {
		t.Fatalf("logged-out slot of account A was rebound to %q — it must not adopt an unrelated account's store row", got)
	}
	if got := inst.LastJID(); got != jidA {
		t.Fatalf("expected the slot to still retain last_jid %s, got %q", jidA, got)
	}

	// B's row must instead surface as its own instance, so it stays visible/manageable.
	if _, ok := manager.GetDevice(nonADB); !ok {
		t.Fatalf("expected the unmatched store row for %s to become its own slot", nonADB)
	}
}

// A never-paired placeholder (no live jid, no last_jid) must still adopt an unmatched
// store row — that is the behavior the last_jid guard has to preserve.
func TestLoadExistingDevices_NeverPairedPlaceholderStillAdoptsStoreRow(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)

	adJID := types.NewADJID("6281999999850", types.WhatsAppDomain, 21)
	nonAD := adJID.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adJID, "primary").Save(ctx); err != nil {
		t.Fatalf("save store device: %v", err)
	}

	const slotID = "slot-never-paired"
	storage := &keepSlotStubStorage{recordJIDs: map[string]string{slotID: ""}} // no last_jid
	manager := NewDeviceManager(primaryStore, nil, storage)

	if err := manager.LoadExistingDevices(ctx); err != nil {
		t.Fatalf("LoadExistingDevices: %v", err)
	}

	inst, ok := manager.GetDevice(slotID)
	if !ok || inst == nil {
		t.Fatal("expected the placeholder slot to survive the load")
	}
	if got := inst.JID(); got != nonAD {
		t.Fatalf("expected the never-paired placeholder to adopt the store row %s, got %q", nonAD, got)
	}
}

// Purging a device must drop its per-device webhook config from the caches: they are
// keyed by JID and cache nil results too, so the deleted device's webhook URL would
// otherwise keep receiving events for the whole TTL after a re-pair of the same account.
func TestRemoveDevice_InvalidatesWebhookConfigCache(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	const jid = "6281999999860@s.whatsapp.net"
	url := "https://deleted-device.example/hook"
	deviceWebhookConfigCache.Store(jid, deviceWebhookConfigCacheEntry{
		config:    &domainChatStorage.DeviceWebhookConfig{WebhookURL: &url},
		expiresAt: time.Now().Add(time.Hour),
	})
	anyDeviceWebhookCache.Store(&anyDeviceWebhookEntry{exists: true, expiresAt: time.Now().Add(time.Hour)})

	manager := NewDeviceManager(nil, nil, &keepSlotStubStorage{})
	const slotID = "slot-to-delete"
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: jid, createdAt: time.Now()}

	manager.RemoveDevice(slotID)

	if _, ok := deviceWebhookConfigCache.Load(jid); ok {
		t.Fatal("expected the deleted device's cached webhook config to be invalidated")
	}
	if entry, _ := anyDeviceWebhookCache.Load().(*anyDeviceWebhookEntry); entry != nil && time.Now().Before(entry.expiresAt) {
		t.Fatal("expected the fleet-wide webhook-exists cache to be invalidated too")
	}
}

// P1 scenario: a slot logged out of account A (last_jid=A) is RE-PAIRED to account B, so
// it now holds jid=B with last_jid=A still recorded (deliberately — it is the only pointer
// to A's retained chat data). An operator who then calls DELETE /devices/A@s.whatsapp.net
// must NOT be handed the live B slot: purging it would log out and destroy account B while
// the caller only ever named A. A last_jid match counts only while the slot is genuinely
// logged out.
func TestPurgeDevice_ByStaleLastJID_DoesNotDestroyTheRepairedLiveAccount(t *testing.T) {
	ctx := context.Background()
	primaryStore := newTestSQLStore(t)

	adB := types.NewADJID("6281999999871", types.WhatsAppDomain, 22)
	nonADB := adB.ToNonAD().String()
	if err := newTestStoreDevice(primaryStore, adB, "primary").Save(ctx); err != nil {
		t.Fatalf("save store device for B: %v", err)
	}

	const slotID = "slot-repaired-live"
	const jidA = "6281999999870@s.whatsapp.net"

	// Logged out of A, since re-paired to B: live jid = B, last_jid still A.
	storage := &keepSlotStubStorage{
		lastJIDs:   map[string]string{slotID: jidA},
		recordJIDs: map[string]string{slotID: nonADB},
	}
	manager := NewDeviceManager(primaryStore, nil, storage)
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: nonADB, displayName: "tIAtendo", createdAt: time.Now()}

	// DELETE addressed by A — the identity the slot no longer holds.
	err := manager.PurgeDevice(ctx, jidA)
	if err == nil {
		// Not finding anything is the acceptable outcome; destroying B is not.
		t.Log("purge by stale last_jid was a no-op")
	}

	if _, ok := manager.GetDevice(slotID); !ok {
		t.Fatal("the live slot paired to account B was destroyed by a DELETE naming account A")
	}
	if got := manager.devices[slotID].JID(); got != nonADB {
		t.Fatalf("expected the live slot to remain paired to B (%s), got %q", nonADB, got)
	}
	assertStoreHasJID(t, ctx, primaryStore, nonADB)
	for _, id := range storage.deletedRecords {
		if id == slotID {
			t.Fatal("the live B slot's device record was deleted by a DELETE naming account A")
		}
	}
}

// assertStoreHasJID fails if no device row in the container matches the given NonAD JID.
func assertStoreHasJID(t *testing.T, ctx context.Context, c *sqlstore.Container, nonADJID string) {
	t.Helper()
	devices, err := c.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("get all devices: %v", err)
	}
	for _, d := range devices {
		if d != nil && d.ID != nil && d.ID.ToNonAD().String() == nonADJID {
			return
		}
	}
	t.Fatalf("expected the store to still contain jid %s", nonADJID)
}

// The logged-out case must keep working: with the slot actually logged out (empty live
// jid), DELETE by its old JID still resolves and purges it.
func TestPurgeDevice_ByLastJID_StillResolvesAGenuinelyLoggedOutSlot(t *testing.T) {
	ctx := context.Background()
	const slotID = "slot-still-logged-out"
	const jidA = "6281999999872@s.whatsapp.net"

	storage := &keepSlotStubStorage{
		lastJIDs:   map[string]string{slotID: jidA},
		recordJIDs: map[string]string{slotID: ""},
	}
	manager := NewDeviceManager(nil, nil, storage)
	manager.devices[slotID] = &DeviceInstance{id: slotID, jid: "", createdAt: time.Now()}

	if err := manager.PurgeDevice(ctx, jidA); err != nil {
		t.Fatalf("PurgeDevice by last_jid returned error: %v", err)
	}
	if _, ok := manager.GetDevice(slotID); ok {
		t.Fatal("expected the logged-out slot to be purged when addressed by its retained JID")
	}
}
