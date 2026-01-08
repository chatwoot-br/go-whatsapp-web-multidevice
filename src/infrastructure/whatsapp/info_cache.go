package whatsapp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/cache"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
)

// Cache TTL constants
const (
	UserInfoTTL        = 30 * time.Second
	UserAvatarTTL      = 60 * time.Second
	BusinessProfileTTL = 60 * time.Second
	GroupInfoTTL       = 30 * time.Second
	GroupLinkInfoTTL   = 60 * time.Second
)

// InfoCache provides device-scoped caching for WhatsApp info requests
type InfoCache struct {
	userInfo        *cache.Cache
	userAvatar      *cache.Cache
	businessProfile *cache.Cache
	groupInfo       *cache.Cache
	groupLinkInfo   *cache.Cache
}

// deviceCaches holds per-device cache instances
var (
	deviceCaches   = make(map[string]*InfoCache)
	deviceCachesMu sync.RWMutex
)

// GetInfoCache returns the InfoCache for a specific device
func GetInfoCache(deviceID string) *InfoCache {
	deviceCachesMu.RLock()
	ic, ok := deviceCaches[deviceID]
	deviceCachesMu.RUnlock()

	if ok {
		return ic
	}

	// Create new cache for this device
	deviceCachesMu.Lock()
	defer deviceCachesMu.Unlock()

	// Double-check after acquiring write lock
	if ic, ok := deviceCaches[deviceID]; ok {
		return ic
	}

	ic = &InfoCache{
		userInfo:        cache.New(UserInfoTTL),
		userAvatar:      cache.New(UserAvatarTTL),
		businessProfile: cache.New(BusinessProfileTTL),
		groupInfo:       cache.New(GroupInfoTTL),
		groupLinkInfo:   cache.New(GroupLinkInfoTTL),
	}
	deviceCaches[deviceID] = ic

	logrus.Debugf("Created info cache for device %s", deviceID)
	return ic
}

// ClearDeviceCache removes all cached data for a specific device
func ClearDeviceCache(deviceID string) {
	deviceCachesMu.Lock()
	defer deviceCachesMu.Unlock()

	delete(deviceCaches, deviceID)
	logrus.Debugf("Cleared info cache for device %s", deviceID)
}

// UserInfo cache methods

// UserInfoResult holds cached user info data
type UserInfoResult struct {
	Data map[types.JID]types.UserInfo
}

// GetUserInfo retrieves cached user info
func (ic *InfoCache) GetUserInfo(phone string) (*UserInfoResult, bool) {
	key := fmt.Sprintf("userinfo:%s", phone)
	if val, ok := ic.userInfo.Get(key); ok {
		if result, ok := val.(*UserInfoResult); ok {
			logrus.Debugf("Cache HIT for user info: %s", phone)
			return result, true
		}
	}
	logrus.Debugf("Cache MISS for user info: %s", phone)
	return nil, false
}

// SetUserInfo stores user info in cache
func (ic *InfoCache) SetUserInfo(phone string, data map[types.JID]types.UserInfo) {
	key := fmt.Sprintf("userinfo:%s", phone)
	ic.userInfo.Set(key, &UserInfoResult{Data: data})
	logrus.Debugf("Cache SET for user info: %s", phone)
}

// UserAvatar cache methods

// UserAvatarResult holds cached avatar data
type UserAvatarResult struct {
	URL    string
	ID     string
	Type   string
	HasURL bool
}

// GetUserAvatar retrieves cached user avatar
func (ic *InfoCache) GetUserAvatar(phone string, isPreview bool, isCommunity bool) (*UserAvatarResult, bool) {
	key := fmt.Sprintf("avatar:%s:%v:%v", phone, isPreview, isCommunity)
	if val, ok := ic.userAvatar.Get(key); ok {
		if result, ok := val.(*UserAvatarResult); ok {
			logrus.Debugf("Cache HIT for avatar: %s", phone)
			return result, true
		}
	}
	logrus.Debugf("Cache MISS for avatar: %s", phone)
	return nil, false
}

// SetUserAvatar stores avatar in cache
func (ic *InfoCache) SetUserAvatar(phone string, isPreview bool, isCommunity bool, url, id, avatarType string, hasURL bool) {
	key := fmt.Sprintf("avatar:%s:%v:%v", phone, isPreview, isCommunity)
	ic.userAvatar.Set(key, &UserAvatarResult{
		URL:    url,
		ID:     id,
		Type:   avatarType,
		HasURL: hasURL,
	})
	logrus.Debugf("Cache SET for avatar: %s", phone)
}

// BusinessProfile cache methods

// BusinessProfileResult holds cached business profile data
type BusinessProfileResult struct {
	Data any // types.BusinessProfile
}

// GetBusinessProfile retrieves cached business profile
func (ic *InfoCache) GetBusinessProfile(phone string) (*BusinessProfileResult, bool) {
	key := fmt.Sprintf("bizprofile:%s", phone)
	if val, ok := ic.businessProfile.Get(key); ok {
		if result, ok := val.(*BusinessProfileResult); ok {
			logrus.Debugf("Cache HIT for business profile: %s", phone)
			return result, true
		}
	}
	logrus.Debugf("Cache MISS for business profile: %s", phone)
	return nil, false
}

// SetBusinessProfile stores business profile in cache
func (ic *InfoCache) SetBusinessProfile(phone string, data any) {
	key := fmt.Sprintf("bizprofile:%s", phone)
	ic.businessProfile.Set(key, &BusinessProfileResult{Data: data})
	logrus.Debugf("Cache SET for business profile: %s", phone)
}

// GroupInfo cache methods

// GroupInfoResult holds cached group info data
type GroupInfoResult struct {
	Data *types.GroupInfo
}

// GetGroupInfo retrieves cached group info
func (ic *InfoCache) GetGroupInfo(groupJID string) (*GroupInfoResult, bool) {
	key := fmt.Sprintf("groupinfo:%s", groupJID)
	if val, ok := ic.groupInfo.Get(key); ok {
		if result, ok := val.(*GroupInfoResult); ok {
			logrus.Debugf("Cache HIT for group info: %s", groupJID)
			return result, true
		}
	}
	logrus.Debugf("Cache MISS for group info: %s", groupJID)
	return nil, false
}

// SetGroupInfo stores group info in cache
func (ic *InfoCache) SetGroupInfo(groupJID string, data *types.GroupInfo) {
	key := fmt.Sprintf("groupinfo:%s", groupJID)
	ic.groupInfo.Set(key, &GroupInfoResult{Data: data})
	logrus.Debugf("Cache SET for group info: %s", groupJID)
}

// GroupLinkInfo cache methods

// GroupLinkInfoResult holds cached group link info data
type GroupLinkInfoResult struct {
	GroupID          string
	Name             string
	Topic            string
	CreatedAt        time.Time
	ParticipantCount int
	IsLocked         bool
	IsAnnounce       bool
	IsEphemeral      bool
	Description      string
}

// GetGroupLinkInfo retrieves cached group link info
func (ic *InfoCache) GetGroupLinkInfo(link string) (*GroupLinkInfoResult, bool) {
	// Hash the link to create a shorter key
	hash := md5.Sum([]byte(link))
	key := fmt.Sprintf("grouplink:%s", hex.EncodeToString(hash[:]))

	if val, ok := ic.groupLinkInfo.Get(key); ok {
		if result, ok := val.(*GroupLinkInfoResult); ok {
			logrus.Debugf("Cache HIT for group link info")
			return result, true
		}
	}
	logrus.Debugf("Cache MISS for group link info")
	return nil, false
}

// SetGroupLinkInfo stores group link info in cache
func (ic *InfoCache) SetGroupLinkInfo(link string, result *GroupLinkInfoResult) {
	hash := md5.Sum([]byte(link))
	key := fmt.Sprintf("grouplink:%s", hex.EncodeToString(hash[:]))
	ic.groupLinkInfo.Set(key, result)
	logrus.Debugf("Cache SET for group link info")
}
