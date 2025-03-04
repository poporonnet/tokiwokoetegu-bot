package main

import (
	"fmt"
	"log"
	"os"
	"tokiwokoetegu-bot/cloudflare"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

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
	discord, err := initDiscord(config)
	if err != nil {
		log.Fatalf("Discordセッションの初期化に失敗しました: %v", err)
	}
	defer discord.Close()

	fmt.Println("ボットが起動しました。終了するにはCTRL+Cを押してください。")
	select {} // 無限に待機
}

// loadConfig は環境変数から設定を読み込む
func loadConfig() (*Config, error) {
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

	return &Config{
		DiscordToken:   token,
		DiscordGuildID: guildID,
		LogChannelID:   logChannelID,
		CloudflareConfig: CloudflareConfig{
			AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
			Email:     os.Getenv("CLOUDFLARE_ACCOUNT_EMAIL"),
			APIKey:    os.Getenv("CLOUDFLARE_API_KEY"),
			DBName:    os.Getenv("CLOUDFLARE_D1_DATABASE_NAME"),
		},
	}, nil
}

// initDiscord はDiscordセッションを初期化する
func initDiscord(config *Config) (*discordgo.Session, error) {
	discord, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("Discordセッションの作成に失敗しました: %w", err)
	}

	// イベントハンドラを登録
	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		handleInteraction(s, i, config)
	})

	err = discord.Open()
	if err != nil {
		return nil, fmt.Errorf("Discordへの接続に失敗しました: %w", err)
	}

	// コンテキストメニューの登録
	err = registerContextMenu(discord, config.DiscordGuildID)
	if err != nil {
		return nil, fmt.Errorf("コンテキストメニューの登録に失敗しました: %w", err)
	}

	return discord, nil
}

// sendLog はログチャンネルにメッセージを送信する
func sendLog(s *discordgo.Session, channelID string, content string, author string, attachments []*discordgo.MessageAttachment) error {
	attachmentUrls := ""
	for _, v := range attachments {
		attachmentUrls = fmt.Sprintf("%s%s\n", attachmentUrls, v.URL)
	}

	logMessage := fmt.Sprintf("from:%s\n%s\n%s", author, content, attachmentUrls)
	_, err := s.ChannelMessageSend(channelID, logMessage)
	if err != nil {
		return fmt.Errorf("ログチャンネルへのメッセージ送信に失敗しました: %w", err)
	}
	return nil
}

// handleInteraction はDiscordのインタラクションを処理する
func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, config *Config) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "pin":
		err := handlePin(s, i, config)
		if err != nil {
			log.Printf("ピン留め処理に失敗しました: %v", err)
			respondWithError(s, i, "ピン留め処理に失敗しました")
		}
	}
}

// handlePin はピン留めコマンドを処理する
func handlePin(s *discordgo.Session, i *discordgo.InteractionCreate, config *Config) error {
	msgData := i.ApplicationCommandData().Resolved.Messages[i.ApplicationCommandData().TargetID]
	msgContent := msgData.Content
	msgAuthor := msgData.Author.Username
	msgAttachments := msgData.Attachments
	msgID := msgData.ID

	msgCreatedAt, err := discordgo.SnowflakeTimestamp(msgID)
	if err != nil {
		return fmt.Errorf("メッセージ作成時間の解析に失敗しました: %w", err)
	}

	// D1データベースに記録
	cfConfig := &config.CloudflareConfig
	err = cloudflare.RecordMessage(cfConfig.DBID, cfConfig.APIKey, cfConfig.Email, cfConfig.AccountID, cfConfig.DBName, msgID, msgAuthor, msgCreatedAt)
	if err != nil {
		return fmt.Errorf("メッセージのデータベース記録に失敗しました: %w", err)
	}

	// ログチャンネルに送信
	err = sendLog(s, config.LogChannelID, msgContent, msgAuthor, msgAttachments)
	if err != nil {
		return fmt.Errorf("ログの送信に失敗しました: %w", err)
	}

	// 応答を返す
	responseContent := fmt.Sprintf("ピン留めしました: %s", msgContent)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseContent,
		},
	})
	if err != nil {
		return fmt.Errorf("インタラクションへの応答に失敗しました: %w", err)
	}

	return nil
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
	if err != nil {
		log.Printf("エラー応答の送信に失敗しました: %v", err)
	}
}

func registerContextMenu(s *discordgo.Session, guildID string) error {
	cmd := &discordgo.ApplicationCommand{
		Name: "pin",
		Type: discordgo.MessageApplicationCommand,
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		return fmt.Errorf("コンテキストメニューの作成に失敗しました: %w", err)
	}
	return nil
}
