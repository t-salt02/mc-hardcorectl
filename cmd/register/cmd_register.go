package main

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// ── 環境変数 ───────────────────────
	token := os.Getenv("BOT_TOKEN")  // Bot Token
	appID := os.Getenv("APP_ID")     // Application ID
	guildID := os.Getenv("GUILD_ID") // 開発用サーバーID（Globalなら ""）

	// ── Discord セッション ─────────────
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	// ── Slash Command 定義 ─────────────
	cmd := &discordgo.ApplicationCommand{
		Name:        "hardcorectl",
		Description: "Hard mode utils",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "destroy",
			Description: "Destroy PVC of statefulset 'hard'",
		}},
	}

	// ── 登録 ───────────────────────────
	if _, err := dg.ApplicationCommandCreate(appID, guildID, cmd); err != nil {
		log.Fatalf("command create: %v", err)
	}
	log.Println("✅ slash command registered!")
}
