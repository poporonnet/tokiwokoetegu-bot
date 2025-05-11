package handlers

import (
	"fmt"
	"strings"
	"time"
	"tokiwokoetegu-bot/cloudflare"
	"tokiwokoetegu-bot/discord/utils"
	"tokiwokoetegu-bot/types"

	"github.com/bwmarrin/discordgo"
)

// handlePin はピン留めコマンドを処理する
func HandlePin(s *discordgo.Session, i *discordgo.InteractionCreate, config *types.Config) error {
	msgData := i.ApplicationCommandData().Resolved.Messages[i.ApplicationCommandData().TargetID]
	msgContent := msgData.Content
	msgAuthorName := msgData.Author.Username
	msgAuthorID := msgData.Author.ID
	msgAttachments := msgData.Attachments
	msgID := msgData.ID
	msgAttachmentURLs := make([]string, 0, len(msgAttachments))
	for _, v := range msgAttachments {
		msgAttachmentURLs = append(msgAttachmentURLs, v.URL)
	}
	msgAttachmentURLsStr := strings.Join(msgAttachmentURLs, ",")
	msgCreatedAt, err := discordgo.SnowflakeTimestamp(msgID)
	if err != nil {
		return fmt.Errorf("メッセージ作成時間の解析に失敗しました: %w", err)
	}

	// D1データベースに記録
	cfConfig := &config.CloudflareConfig
	err = recordMessage(cfConfig.DBID, cfConfig.APIKey, cfConfig.Email, cfConfig.AccountID, cfConfig.DBName, msgID, msgAuthorID, msgContent, msgAttachmentURLsStr, msgCreatedAt)
	if err != nil {
		return fmt.Errorf("メッセージのデータベース記録に失敗しました: %w", err)
	}

	logMessage := fmt.Sprintf("from:%s\n%s\n%s", msgAuthorName, msgContent, msgAttachmentURLsStr)
	err = utils.SendLog(s, config.LogChannelID, logMessage)
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

func recordMessage(dbID, apiKey, email, accountID, dbName, messageID, authorID, messageContent, attachmentUrls string, messageCreatedAT time.Time) error {
	api, err := cloudflare.NewCloudflareAPI(accountID, email, apiKey, dbName)
	if err != nil {
		return fmt.Errorf("cloudflare API クライアントの初期化に失敗しました: %w", err)
	}

	api.DBID = dbID

	layout := "2006-01-02 15:04:05"
	currentTime := time.Now().Format(layout)
	msgCreatedAtStr := messageCreatedAT.Format(layout)

	query := fmt.Sprintf(
		"INSERT INTO MESSAGE (MessageID, AuthorID, MessageContent, Attachments, MessageCreatedAT, CreatedAT, UpdatedAT) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s')",
		messageID, messageContent, attachmentUrls, authorID, msgCreatedAtStr, currentTime, currentTime,
	)

	err = api.PostQuery(query)
	if err != nil {
		return fmt.Errorf("メッセージの記録に失敗しました: %w", err)
	}

	return nil
}
