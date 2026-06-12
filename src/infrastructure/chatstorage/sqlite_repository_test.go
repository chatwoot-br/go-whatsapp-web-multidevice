package chatstorage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

// newTestRepo opens a fresh file-backed SQLite DB in a per-test temp directory,
// applies the schema, and returns the concrete *SQLiteRepository plus the
// underlying *sql.DB so tests can poke at rows directly. File-backed (not
// `:memory:`) so the same database is visible across the pool's connections —
// MergeLIDChat opens a tx alongside helper Queries on the parent *sql.DB.
func newTestRepo(t *testing.T) (*SQLiteRepository, *sql.DB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := &SQLiteRepository{db: db}
	if err := repo.InitializeSchema(); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return repo, db
}

func insertChat(t *testing.T, db *sql.DB, deviceID, jid, name string, last time.Time) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO chats (jid, device_id, name, last_message_time, ephemeral_expiration, archived) VALUES (?, ?, ?, ?, 0, 0)`,
		jid, deviceID, name, last)
	if err != nil {
		t.Fatalf("insert chat %s: %v", jid, err)
	}
}

func insertMessage(t *testing.T, db *sql.DB, id, chatJID, deviceID, sender, content string, ts time.Time) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO messages (id, chat_jid, device_id, sender, content, timestamp, is_from_me) VALUES (?, ?, ?, ?, ?, ?, 0)`,
		id, chatJID, deviceID, sender, content, ts)
	if err != nil {
		t.Fatalf("insert message %s: %v", id, err)
	}
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// TestMergeLIDChat_HappyPath asserts that messages migrate to the phone JID,
// the LID chat row is deleted, and the phone chat's metadata is preserved.
func TestMergeLIDChat_HappyPath(t *testing.T) {
	repo, db := newTestRepo(t)

	device := "dev1"
	lidJID := "215946727821336@lid"
	phoneJID := "5511999999999@s.whatsapp.net"
	now := time.Now().UTC()

	insertChat(t, db, device, lidJID, "Alice LID", now.Add(-time.Hour))
	insertChat(t, db, device, phoneJID, "5511999999999", now.Add(-2*time.Hour))
	insertMessage(t, db, "msg-1", lidJID, device, lidJID, "hello from lid", now.Add(-30*time.Minute))
	insertMessage(t, db, "msg-2", lidJID, device, lidJID, "second", now.Add(-20*time.Minute))

	if err := repo.MergeLIDChat(device, lidJID, phoneJID); err != nil {
		t.Fatalf("MergeLIDChat: %v", err)
	}

	if got := countRows(t, db, `SELECT COUNT(*) FROM chats WHERE jid=? AND device_id=?`, lidJID, device); got != 0 {
		t.Errorf("LID chat row should be deleted, got %d", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chats WHERE jid=? AND device_id=?`, phoneJID, device); got != 1 {
		t.Errorf("phone chat row should exist, got %d", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM messages WHERE chat_jid=? AND device_id=?`, phoneJID, device); got != 2 {
		t.Errorf("expected 2 messages on phone chat, got %d", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM messages WHERE chat_jid=? AND device_id=?`, lidJID, device); got != 0 {
		t.Errorf("expected 0 messages on lid chat, got %d", got)
	}

	// Phone-chat name should be overwritten because old name was a bare phone number.
	var newName string
	if err := db.QueryRow(`SELECT name FROM chats WHERE jid=?`, phoneJID).Scan(&newName); err != nil {
		t.Fatalf("scan name: %v", err)
	}
	if newName != "Alice LID" {
		t.Errorf("phone-chat name should adopt LID name (was phone-string), got %q", newName)
	}
}

// TestMergeLIDChat_RenameOnly asserts that when no phone-chat exists, the LID
// row is renamed to the phone JID in place.
func TestMergeLIDChat_RenameOnly(t *testing.T) {
	repo, db := newTestRepo(t)

	device := "dev1"
	lidJID := "999@lid"
	phoneJID := "5511888888888@s.whatsapp.net"
	now := time.Now().UTC()

	insertChat(t, db, device, lidJID, "Carol", now)
	insertMessage(t, db, "m1", lidJID, device, lidJID, "x", now)

	if err := repo.MergeLIDChat(device, lidJID, phoneJID); err != nil {
		t.Fatalf("MergeLIDChat: %v", err)
	}

	if got := countRows(t, db, `SELECT COUNT(*) FROM chats WHERE jid=?`, lidJID); got != 0 {
		t.Errorf("LID row should be renamed, found %d", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM chats WHERE jid=?`, phoneJID); got != 1 {
		t.Errorf("phone row should exist after rename, got %d", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM messages WHERE chat_jid=?`, phoneJID); got != 1 {
		t.Errorf("message should have followed rename, got %d", got)
	}
}

// TestMergeLIDChat_LIDMissing returns nil quietly when the LID chat doesn't exist.
func TestMergeLIDChat_LIDMissing(t *testing.T) {
	repo, _ := newTestRepo(t)
	if err := repo.MergeLIDChat("dev1", "missing@lid", "x@s.whatsapp.net"); err != nil {
		t.Fatalf("expected nil for missing LID chat, got %v", err)
	}
}

// TestMergeLIDChat_PreservesNonPhoneName asserts that an existing phone-chat name
// that is NOT a phone-number string (e.g. saved contact) is NOT overwritten by the
// LID chat's name.
func TestMergeLIDChat_PreservesNonPhoneName(t *testing.T) {
	repo, db := newTestRepo(t)

	device := "dev1"
	lidJID := "1@lid"
	phoneJID := "5511777777777@s.whatsapp.net"
	now := time.Now().UTC()

	insertChat(t, db, device, lidJID, "LID display", now.Add(-time.Hour))
	insertChat(t, db, device, phoneJID, "Mom", now)

	if err := repo.MergeLIDChat(device, lidJID, phoneJID); err != nil {
		t.Fatalf("MergeLIDChat: %v", err)
	}

	var name string
	if err := db.QueryRow(`SELECT name FROM chats WHERE jid=?`, phoneJID).Scan(&name); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if name != "Mom" {
		t.Errorf("expected name to remain 'Mom', got %q", name)
	}
}

// TestGetLIDChats returns only @lid rows for the given device.
func TestGetLIDChats(t *testing.T) {
	repo, db := newTestRepo(t)

	device := "dev1"
	now := time.Now().UTC()

	insertChat(t, db, device, "1@lid", "lid one", now)
	insertChat(t, db, device, "2@lid", "lid two", now.Add(-time.Minute))
	insertChat(t, db, device, "5511111111111@s.whatsapp.net", "phone", now)
	insertChat(t, db, device, "1234567890@g.us", "group", now)
	// Different device — should NOT appear.
	insertChat(t, db, "dev2", "3@lid", "lid other dev", now)

	chats, err := repo.GetLIDChats(device)
	if err != nil {
		t.Fatalf("GetLIDChats: %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("expected 2 @lid rows for device, got %d", len(chats))
	}
	for _, c := range chats {
		if c.DeviceID != device {
			t.Errorf("wrong device: %s", c.DeviceID)
		}
	}
}

// TestGetLIDChats_Empty returns nil/empty when no rows.
func TestGetLIDChats_Empty(t *testing.T) {
	repo, _ := newTestRepo(t)

	chats, err := repo.GetLIDChats("dev-empty")
	if err != nil {
		t.Fatalf("GetLIDChats: %v", err)
	}
	if len(chats) != 0 {
		t.Errorf("expected 0 chats, got %d", len(chats))
	}
}

// TestIsPhoneNumberString covers the unexported helper used by MergeLIDChat to
// decide name overwriting.
func TestIsPhoneNumberString(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"5511999999999", true},
		{"+5511999999999", true},
		{"Mom", false},
		{"5511 999", false}, // space breaks digits-only
		{"12345", true},     // 5 chars = minimum length
		{"1234", false},     // below minimum
		{"+1234", true},     // 5 chars incl. '+' meets the >=5 length check
		{"+123", false},     // 4 chars total
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := isPhoneNumberString(tc.in); got != tc.want {
				t.Errorf("isPhoneNumberString(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestMergeLIDChat_NoCrossDeviceLeakage asserts that a merge on dev1 does NOT
// touch chats or messages belonging to dev2.
func TestMergeLIDChat_NoCrossDeviceLeakage(t *testing.T) {
	repo, db := newTestRepo(t)

	now := time.Now().UTC()

	// dev1 — target of the merge.
	insertChat(t, db, "dev1", "X@lid", "lid d1", now)
	insertChat(t, db, "dev1", "phoneA@s.whatsapp.net", "phone d1", now)
	insertMessage(t, db, "m-dev1", "X@lid", "dev1", "X@lid", "hi dev1", now)

	// dev2 — must survive untouched.
	insertChat(t, db, "dev2", "X@lid", "lid d2", now)
	insertMessage(t, db, "m-dev2", "X@lid", "dev2", "X@lid", "hi dev2", now)

	if err := repo.MergeLIDChat("dev1", "X@lid", "phoneA@s.whatsapp.net"); err != nil {
		t.Fatalf("merge: %v", err)
	}

	// dev2 LID chat should still exist with its original name.
	var name string
	if err := db.QueryRow(`SELECT name FROM chats WHERE jid=? AND device_id=?`, "X@lid", "dev2").Scan(&name); err != nil {
		t.Fatalf("dev2 LID chat disappeared: %v", err)
	}
	if name != "lid d2" {
		t.Errorf("dev2 LID chat name changed: %q", name)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM messages WHERE chat_jid=? AND device_id=?`, "X@lid", "dev2"); got != 1 {
		t.Errorf("dev2 messages should remain on LID chat, got %d", got)
	}
}

// TestMergeLIDChat_NoDeadlockWithSingleConnPool reproduces the production pool
// constraint set in cmd/root.go:initChatStorage — db.SetMaxOpenConns(1). Before
// the tx-scoped read fix, MergeLIDChat would Begin() (grabbing the only conn),
// then call r.GetChatByDevice (which uses r.db.QueryRow → needs a 2nd conn from
// the same pool) → permanent deadlock that froze every subsequent CreateMessage
// from incoming WhatsApp events.
//
// The test runs MergeLIDChat under a 5 s deadline; pre-fix it hangs and the
// timeout watchdog fires t.Fatal; post-fix it returns in <50 ms.
func TestMergeLIDChat_NoDeadlockWithSingleConnPool(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Reproduce production constraint: any nested r.db.* call inside a tx will
	// deadlock against the parent transaction holding the single connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	repo := &SQLiteRepository{db: db}
	if err := repo.InitializeSchema(); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	device := "dev1"
	lidJID := "215946727821336@lid"
	phoneJID := "5511999999999@s.whatsapp.net"
	now := time.Now().UTC()

	insertChat(t, db, device, lidJID, "Alice LID", now.Add(-time.Hour))
	insertChat(t, db, device, phoneJID, "5511999999999", now.Add(-2*time.Hour))
	insertMessage(t, db, "msg-1", lidJID, device, lidJID, "hello", now.Add(-30*time.Minute))

	// Watchdog: kill the test if MergeLIDChat hangs for more than 5 s.
	done := make(chan error, 1)
	go func() { done <- repo.MergeLIDChat(device, lidJID, phoneJID) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("MergeLIDChat returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("MergeLIDChat deadlocked: did not return within 5 s under MaxOpenConns(1) — the tx must use tx.QueryRow for reads, not r.db.QueryRow")
	}

	// Sanity: the merge actually happened.
	if got := countRows(t, db, `SELECT COUNT(*) FROM chats WHERE jid=? AND device_id=?`, lidJID, device); got != 0 {
		t.Errorf("LID chat row should be deleted post-merge, got %d", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM messages WHERE chat_jid=? AND device_id=?`, phoneJID, device); got != 1 {
		t.Errorf("message should be migrated to phone chat, got %d", got)
	}
}

// Static assertion that the repo satisfies the domain interface (compile-time guard).
var _ domainChatStorage.IChatStorageRepository = (*SQLiteRepository)(nil)

func newTestSQLiteRepository(t *testing.T) *SQLiteRepository {
	t.Helper()

	db, err := sql.Open(sqlite.DriverName, filepath.Join(t.TempDir(), "chatstorage.db"))
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := &SQLiteRepository{db: db}
	if err := repo.InitializeSchema(); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}
	return repo
}

func TestSQLiteRepositoryInitializesMessageReactionsSchema(t *testing.T) {
	repo := newTestSQLiteRepository(t)

	var tableName string
	err := repo.db.QueryRow(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name = 'message_reactions'
	`).Scan(&tableName)
	if err != nil {
		t.Fatalf("expected message_reactions table to exist: %v", err)
	}
	if tableName != "message_reactions" {
		t.Fatalf("expected message_reactions table, got %q", tableName)
	}
}

func TestSQLiteRepositoryStoresUpdatesRemovesAndHydratesReactions(t *testing.T) {
	repo := newTestSQLiteRepository(t)
	deviceID := "device-a@s.whatsapp.net"
	otherDeviceID := "device-b@s.whatsapp.net"
	chatJID := "628123456789@s.whatsapp.net"
	now := time.Date(2026, time.May, 16, 8, 0, 0, 0, time.UTC)

	seedChatMessage(t, repo, deviceID, chatJID, "msg-1", "hello reaction", now)
	seedChatMessage(t, repo, otherDeviceID, chatJID, "msg-1", "hello other device", now)

	if err := repo.StoreReaction(&domainChatStorage.Reaction{
		MessageID:  "msg-1",
		ChatJID:    chatJID,
		DeviceID:   deviceID,
		ReactorJID: "628111111111@s.whatsapp.net",
		Emoji:      "\U0001f44d",
		IsFromMe:   false,
		Timestamp:  now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("store reaction: %v", err)
	}
	if err := repo.StoreReaction(&domainChatStorage.Reaction{
		MessageID:  "msg-1",
		ChatJID:    chatJID,
		DeviceID:   otherDeviceID,
		ReactorJID: "628222222222@s.whatsapp.net",
		Emoji:      "\U0001f525",
		IsFromMe:   false,
		Timestamp:  now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("store other device reaction: %v", err)
	}

	messages := getMessagesForTest(t, repo, deviceID, chatJID)
	if got := len(messages[0].Reactions); got != 1 {
		t.Fatalf("expected one device-scoped reaction, got %d", got)
	}
	if got := messages[0].Reactions[0].Emoji; got != "\U0001f44d" {
		t.Fatalf("expected hydrated thumbs-up reaction, got %q", got)
	}

	if err := repo.StoreReaction(&domainChatStorage.Reaction{
		MessageID:  "msg-1",
		ChatJID:    chatJID,
		DeviceID:   deviceID,
		ReactorJID: "628111111111@s.whatsapp.net",
		Emoji:      "\U0001f525",
		IsFromMe:   false,
		Timestamp:  now.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("update reaction: %v", err)
	}

	messages = getMessagesForTest(t, repo, deviceID, chatJID)
	if got := len(messages[0].Reactions); got != 1 {
		t.Fatalf("expected one updated reaction, got %d", got)
	}
	if got := messages[0].Reactions[0].Emoji; got != "\U0001f525" {
		t.Fatalf("expected updated fire reaction, got %q", got)
	}

	searchResults, err := repo.SearchMessages(deviceID, chatJID, "reaction", 10)
	if err != nil {
		t.Fatalf("search messages: %v", err)
	}
	if got := len(searchResults); got != 1 {
		t.Fatalf("expected one search result, got %d", got)
	}
	if got := len(searchResults[0].Reactions); got != 1 {
		t.Fatalf("expected search result to hydrate reactions, got %d", got)
	}

	if err := repo.StoreReaction(&domainChatStorage.Reaction{
		MessageID:  "msg-1",
		ChatJID:    chatJID,
		DeviceID:   deviceID,
		ReactorJID: "628111111111@s.whatsapp.net",
		Emoji:      "",
		Timestamp:  now.Add(4 * time.Minute),
	}); err != nil {
		t.Fatalf("remove reaction: %v", err)
	}

	messages = getMessagesForTest(t, repo, deviceID, chatJID)
	if got := len(messages[0].Reactions); got != 0 {
		t.Fatalf("expected reaction removal to clear reactions, got %d", got)
	}
}

func TestSQLiteRepositoryDeletesReactionsWithMessagesAndDevices(t *testing.T) {
	repo := newTestSQLiteRepository(t)
	deviceID := "device-a@s.whatsapp.net"
	otherDeviceID := "device-b@s.whatsapp.net"
	chatJID := "628123456789@s.whatsapp.net"
	now := time.Date(2026, time.May, 16, 8, 0, 0, 0, time.UTC)

	seedChatMessage(t, repo, deviceID, chatJID, "msg-1", "hello", now)
	seedReaction(t, repo, deviceID, chatJID, "msg-1", "628111111111@s.whatsapp.net")
	seedChatMessage(t, repo, otherDeviceID, chatJID, "msg-1", "hello", now)
	seedReaction(t, repo, otherDeviceID, chatJID, "msg-1", "628222222222@s.whatsapp.net")

	if err := repo.DeleteMessageByDevice(deviceID, "msg-1", chatJID); err != nil {
		t.Fatalf("delete message by device: %v", err)
	}
	if got := countMessageReactions(t, repo); got != 1 {
		t.Fatalf("expected only other device reaction to remain, got %d", got)
	}

	if err := repo.DeleteDeviceData(otherDeviceID); err != nil {
		t.Fatalf("delete device data: %v", err)
	}
	if got := countMessageReactions(t, repo); got != 0 {
		t.Fatalf("expected device cleanup to delete reactions, got %d", got)
	}
}

func TestStoreSentMessageWithContextRequiresDeviceInContext(t *testing.T) {
	repo := newTestSQLiteRepository(t)
	deviceID := "6289605618749@s.whatsapp.net"
	chatJID := "628123456789@s.whatsapp.net"
	now := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)

	err := repo.StoreSentMessageWithContext(
		context.Background(),
		"msg-sent-1",
		deviceID,
		chatJID,
		"hello from api",
		now,
		nil,
	)
	if err == nil {
		t.Fatal("expected error when storing sent message without device context")
	}
	if !errors.Is(err, domainChatStorage.ErrMissingDeviceContext) {
		t.Fatalf("expected missing device context error, got %v", err)
	}
}

func seedChatMessage(t *testing.T, repo *SQLiteRepository, deviceID, chatJID, messageID, content string, timestamp time.Time) {
	t.Helper()
	if err := repo.StoreChat(&domainChatStorage.Chat{
		DeviceID:        deviceID,
		JID:             chatJID,
		Name:            chatJID,
		LastMessageTime: timestamp,
	}); err != nil {
		t.Fatalf("store chat: %v", err)
	}
	if err := repo.StoreMessage(&domainChatStorage.Message{
		ID:        messageID,
		ChatJID:   chatJID,
		DeviceID:  deviceID,
		Sender:    "628999999999@s.whatsapp.net",
		Content:   content,
		Timestamp: timestamp,
	}); err != nil {
		t.Fatalf("store message: %v", err)
	}
}

func seedReaction(t *testing.T, repo *SQLiteRepository, deviceID, chatJID, messageID, reactorJID string) {
	t.Helper()
	if err := repo.StoreReaction(&domainChatStorage.Reaction{
		MessageID:  messageID,
		ChatJID:    chatJID,
		DeviceID:   deviceID,
		ReactorJID: reactorJID,
		Emoji:      "\U0001f44d",
		Timestamp:  time.Date(2026, time.May, 16, 8, 1, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("store reaction: %v", err)
	}
}

func getMessagesForTest(t *testing.T, repo *SQLiteRepository, deviceID, chatJID string) []*domainChatStorage.Message {
	t.Helper()
	messages, err := repo.GetMessages(&domainChatStorage.MessageFilter{
		DeviceID: deviceID,
		ChatJID:  chatJID,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected one message, got %d", len(messages))
	}
	return messages
}

func countMessageReactions(t *testing.T, repo *SQLiteRepository) int {
	t.Helper()
	var count int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM message_reactions`).Scan(&count); err != nil {
		t.Fatalf("count message reactions: %v", err)
	}
	return count
}
