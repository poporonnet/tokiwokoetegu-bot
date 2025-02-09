package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	// TOKEN, err := os.LookupEnv("DISCORD_TOKEN")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}
	TOKEN := os.Getenv("DISCORD_TOKEN")
	if TOKEN == "" {
		log.Fatal("Token is Empty")
	}
	GuildID := os.Getenv("DISCORD_GUILD_ID")

	discord, err := discordgo.New("Bot " + TOKEN)

	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	discord.AddHandler(handleInteraction)
	err = discord.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}

	registerContextMenu(discord, GuildID)

	fmt.Println("Bot is now running. Press CTRL+C to exit.")

	select {} // Wait indefinitely.
}

func sendLog(s *discordgo.Session, channelID string, content string, author string, attachments []*discordgo.MessageAttachment) {
	attachmentUrls := ""
	for _, v := range attachments {
		attachmentUrls = fmt.Sprintf("%s%s\n", attachmentUrls, v.URL)
	}

	logMessage := fmt.Sprintf("from:%s\n%s\n%s", author, content, attachmentUrls)
	_, err := s.ChannelMessageSend(channelID, logMessage)
	if err != nil {
		log.Fatalln("Failed to send message to log channel")
	}
}

func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logChannelID := os.Getenv("DISCORD_LOG_CHANNEL_ID")
	if logChannelID == "" {
		log.Fatalln("logChannelID is Empty")
		return
	}
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "pin":
			pin(s, i, logChannelID)
		}
	}
}

func pin(s *discordgo.Session, i *discordgo.InteractionCreate, logChannelID string) {
	msgContent := i.ApplicationCommandData().Resolved.Messages[i.ApplicationCommandData().TargetID].Content
	msgAuthor := i.ApplicationCommandData().Resolved.Messages[i.ApplicationCommandData().TargetID].Author.Username
	msgAttachments := i.ApplicationCommandData().Resolved.Messages[i.ApplicationCommandData().TargetID].Attachments
	responseContent := fmt.Sprintf("ピン留めしました: %s", msgContent)
	sendLog(s, logChannelID, msgContent, msgAuthor, msgAttachments)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseContent,
		},
	})
	if err != nil {
		log.Printf("Failed to respond to interaction: %v", err)
	}
}

func registerContextMenu(s *discordgo.Session, guildID string) {
	cmd := &discordgo.ApplicationCommand{
		Name: "pin",
		Type: discordgo.MessageApplicationCommand,
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Println("Failed to create context menu:", err)
	}
}
