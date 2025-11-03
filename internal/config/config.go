package config

import "github.com/cloudflare/cloudflare-go"

var (
	ApiKey         string
	AccountID      string
	Debug          bool
	EnableDevices  bool
	EnableUsers    bool
	EnableTunnels  bool
	EnableDex      bool
	EnableMagicWAN bool
	Client         *cloudflare.API
)

func InitConfig(apiKey, accountID string, debug, enableDevices, enableUsers, enableTunnels, enableDex, enableMagicWAN bool, client *cloudflare.API) {
	ApiKey = apiKey
	AccountID = accountID
	Debug = debug
	EnableDevices = enableDevices
	EnableUsers = enableUsers
	EnableTunnels = enableTunnels
	EnableDex = enableDex
	EnableMagicWAN = enableMagicWAN
	Client = client
}
