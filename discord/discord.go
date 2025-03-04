package discord

import (
	"fmt"
	"log"
	"tokiwokoetegu-bot/cloudflare"
	"tokiwokoetegu-bot/types"

	"github.com/bwmarrin/discordgo"
)

// initDiscord はDiscordセッションを初期化する
func InitDiscord(config *types.Config) (*discordgo.Session, error) {
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
func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, config *types.Config) {
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
func handlePin(s *discordgo.Session, i *discordgo.InteractionCreate, config *types.Config) error {
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
