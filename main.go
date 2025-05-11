package main

import (
	"fmt"
	"log"
	"os"
	"tokiwokoetegu-bot/cloudflare"
	"tokiwokoetegu-bot/discord"
	"tokiwokoetegu-bot/types"

	"github.com/joho/godotenv"
)

func main() {
	// 設定の読み込み
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// Cloudflare D1データベースの初期化
	cfConfig := &config.CloudflareConfig
	cfConfig.DBID, err = cloudflare.D1Init(cfConfig.APIKey, cfConfig.Email, cfConfig.AccountID, cfConfig.DBName)
	if err != nil {
		log.Fatalf("D1データベースの初期化に失敗しました: %v", err)
	}

	// Discordセッションの初期化と起動
	discord, err := discord.InitDiscord(config)
	if err != nil {
		log.Fatalf("Discordセッションの初期化に失敗しました: %v", err)
	}
	defer discord.Close()

	fmt.Println("ボットが起動しました。終了するにはCTRL+Cを押してください。")
	select {} // 無限に待機
}

// loadConfig は環境変数から設定を読み込む
func loadConfig() (*types.Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("環境変数ファイルの読み込みに失敗しました: %w", err)
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKENが設定されていません")
	}

	guildID := os.Getenv("DISCORD_GUILD_ID")
	logChannelID := os.Getenv("DISCORD_LOG_CHANNEL_ID")
	if logChannelID == "" {
		return nil, fmt.Errorf("DISCORD_LOG_CHANNEL_IDが設定されていません")
	}

	return &types.Config{
		DiscordToken:   token,
		DiscordGuildID: guildID,
		LogChannelID:   logChannelID,
		CloudflareConfig: types.CloudflareConfig{
			AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
			Email:     os.Getenv("CLOUDFLARE_ACCOUNT_EMAIL"),
			APIKey:    os.Getenv("CLOUDFLARE_API_KEY"),
			DBName:    os.Getenv("CLOUDFLARE_D1_DATABASE_NAME"),
		},
	}, nil
}
