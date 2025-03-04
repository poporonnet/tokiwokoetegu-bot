package types

// Config は設定情報を保持する構造体
type Config struct {
	DiscordToken     string
	DiscordGuildID   string
	LogChannelID     string
	CloudflareConfig CloudflareConfig
}

// CloudflareConfig はCloudflare関連の設定を保持する構造体
type CloudflareConfig struct {
	AccountID string
	Email     string
	APIKey    string
	DBName    string
	DBID      string
}
