package utils

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// sendLog はログチャンネルにメッセージを送信する
func SendLog(s *discordgo.Session, channelID, logMessage string) error {
	_, err := s.ChannelMessageSend(channelID, logMessage)
	if err != nil {
		return fmt.Errorf("ログチャンネルへのメッセージ送信に失敗しました: %w", err)
	}
	return nil
}
