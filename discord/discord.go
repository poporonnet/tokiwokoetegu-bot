package discord

import (
	"fmt"
	"log"
	"tokiwokoetegu-bot/types"

	"github.com/bwmarrin/discordgo"
	"tokiwokoetegu-bot/discord/handlers"
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

func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, config *types.Config) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "pin":
		err := handlers.HandlePin(s, i, config)
		if err != nil {
			log.Printf("ピン留め処理に失敗しました: %v", err)
			respondWithError(s, i, "ピン留め処理に失敗しました")
		}
	}
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
