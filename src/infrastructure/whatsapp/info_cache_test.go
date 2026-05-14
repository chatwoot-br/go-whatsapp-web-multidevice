package whatsapp

import (
	"testing"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/cache"
	"go.mau.fi/whatsmeow/types"
)

// TestGetInfoCache_PerDeviceIsolation asserts that two different deviceIDs get
// distinct InfoCache instances and writes to one do not leak to the other.
func TestGetInfoCache_PerDeviceIsolation(t *testing.T) {
	t.Cleanup(func() {
		ClearDeviceCache("test-dev-A")
		ClearDeviceCache("test-dev-B")
	})

	a := GetInfoCache("test-dev-A")
	b := GetInfoCache("test-dev-B")
	if a == b {
		t.Fatal("different devices must yield different caches")
	}

	a.SetUserInfo("phoneA", map[types.JID]types.UserInfo{
		types.NewJID("123", types.DefaultUserServer): {},
	})

	if _, ok := b.GetUserInfo("phoneA"); ok {
		t.Fatal("cache HIT crossed device boundary")
	}
}

// TestGetInfoCache_SameDeviceSameInstance asserts that repeated calls for the
// same deviceID return the same cache pointer (the lazy-init double-check).
func TestGetInfoCache_SameDeviceSameInstance(t *testing.T) {
	t.Cleanup(func() { ClearDeviceCache("test-dev-same") })

	a := GetInfoCache("test-dev-same")
	b := GetInfoCache("test-dev-same")
	if a != b {
		t.Fatal("repeated GetInfoCache for same device must return the same instance")
	}
}

// TestInfoCache_UserInfo_HitThenMissAfterTTL exercises the TTL behavior end to
// end: SET then GET within TTL = hit; manual expire = miss. Uses a fresh cache
// with very short TTL instead of the deviceCaches pool to keep the test fast
// and isolated.
func TestInfoCache_UserInfo_TTLExpiry(t *testing.T) {
	short := 50 * time.Millisecond
	ic := &InfoCache{
		userInfo:        cache.New(short),
		userAvatar:      cache.New(short),
		businessProfile: cache.New(short),
		groupInfo:       cache.New(short),
		groupLinkInfo:   cache.New(short),
	}

	ic.SetUserInfo("ph", map[types.JID]types.UserInfo{
		types.NewJID("1", types.DefaultUserServer): {},
	})
	if _, ok := ic.GetUserInfo("ph"); !ok {
		t.Fatal("expected hit immediately after set")
	}

	time.Sleep(short + 30*time.Millisecond)
	if _, ok := ic.GetUserInfo("ph"); ok {
		t.Fatal("expected miss after TTL expiry")
	}
}

// TestInfoCache_GroupInfoErrorCaching asserts that a cached error result is
// retained (prevents thundering-herd on repeated failed lookups) within TTL.
func TestInfoCache_GroupInfoErrorCaching(t *testing.T) {
	t.Cleanup(func() { ClearDeviceCache("test-dev-err") })

	ic := GetInfoCache("test-dev-err")
	jid := "120363111111111@g.us"
	ic.SetGroupInfoError(jid, "not participating in group")

	got, ok := ic.GetGroupInfo(jid)
	if !ok {
		t.Fatal("expected cache hit for error entry")
	}
	if got.ErrorMsg != "not participating in group" {
		t.Errorf("error msg = %q", got.ErrorMsg)
	}
	if got.Data != nil {
		t.Error("error entry must not carry GroupInfo data")
	}
}

// TestInfoCache_UserAvatarErrorPath same pattern: error result is cached and
// retrievable, with a sentinel ErrorMsg.
func TestInfoCache_UserAvatarErrorPath(t *testing.T) {
	t.Cleanup(func() { ClearDeviceCache("test-dev-avatar-err") })

	ic := GetInfoCache("test-dev-avatar-err")
	ic.SetUserAvatarError("ph", false, false, "hidden profile picture")

	got, ok := ic.GetUserAvatar("ph", false, false)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.ErrorMsg != "hidden profile picture" {
		t.Errorf("got %q", got.ErrorMsg)
	}
	if got.HasURL {
		t.Error("HasURL must be false for cached error")
	}
}

// TestClearDeviceCache removes the entry and the next GetInfoCache returns a
// fresh empty cache (no carryover).
func TestClearDeviceCache(t *testing.T) {
	id := "test-dev-clear"
	ic := GetInfoCache(id)
	ic.SetBusinessProfile("p", "data")
	if _, ok := ic.GetBusinessProfile("p"); !ok {
		t.Fatal("setup: expected hit")
	}

	ClearDeviceCache(id)

	ic2 := GetInfoCache(id)
	if _, ok := ic2.GetBusinessProfile("p"); ok {
		t.Fatal("cache survived ClearDeviceCache")
	}
	t.Cleanup(func() { ClearDeviceCache(id) })
}
