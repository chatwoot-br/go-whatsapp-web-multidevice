package config

import (
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
)

var (
	AppVersion             = "v7.11.0+2"
	AppPort                = "3000"
	AppDebug               = false
	AppOs                  = "AldinoKemal"
	AppPlatform            = waCompanionReg.DeviceProps_PlatformType(1)
	AppBasicAuthCredential []string
	AppBasePath            = ""
	AppTrustedProxies      []string // Trusted proxy IP ranges (e.g., "0.0.0.0/0" for all, or specific CIDRs)

	McpPort = "8080"
	McpHost = "localhost"

	// Admin API Configuration
	AdminPort             = "8088"
	AdminToken            = ""
	SupervisorURL         = "http://127.0.0.1:9001/RPC2"
	SupervisorUser        = ""
	SupervisorPass        = ""
	SupervisorConfDir     = "/etc/supervisor/conf.d"
	InstancesDir          = "/app/instances"
	SupervisorLogDir      = "/var/log/supervisor"
	GowaBin               = "/usr/local/bin/whatsapp"
	GowaBasicAuth         = "admin:admin"
	GowaDebug             = false
	GowaOS                = "Chrome"
	GowaAccountValidation = false
	GowaBasePath          = ""
	GowaAutoReply         = ""
	GowaAutoMarkRead      = false
	GowaWebhook           = ""
	GowaWebhookSecret     = "secret"
	GowaChatStorage       = true
	LockDir               = "/tmp"

	PathQrCode    = "statics/qrcode"
	PathSendItems = "statics/senditems"
	PathMedia     = "statics/media"
	PathStorages  = "storages"

	DBURI     = "file:storages/whatsapp.db?_foreign_keys=on"
	DBKeysURI = ""

	WhatsappAutoReplyMessage           string
	WhatsappAutoMarkRead               = false // Auto-mark incoming messages as read
	WhatsappAutoDownloadMedia         = true  // Auto-download media from incoming messages
	WhatsappWebhook                    []string
	WhatsappWebhookSecret                    = "secret"
	WhatsappWebhookInsecureSkipVerify       = false // Skip TLS certificate verification for webhooks (insecure)
	WhatsappLogLevel                         = "ERROR"
	WhatsappSettingMaxImageSize        int64 = 20000000  // 20MB
	WhatsappSettingMaxFileSize         int64 = 50000000  // 50MB
	WhatsappSettingMaxVideoSize        int64 = 100000000 // 100MB
	WhatsappSettingMaxAudioSize     int64 = 16000000  // 16MB (WhatsApp limit)
	WhatsappSettingMaxDownloadSize     int64 = 500000000 // 500MB
	WhatsappSettingAutoConvertAudio       = true      // Auto-convert audio to optimal format
	WhatsappTypeUser                         = "@s.whatsapp.net"
	WhatsappTypeGroup                        = "@g.us"
	WhatsappAccountValidation                = true

	ChatStorageURI               = "file:storages/chatstorage.db"
	ChatStorageEnableForeignKeys = true
	ChatStorageEnableWAL         = true
)
