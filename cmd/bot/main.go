package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	ns      = "minecraft-hard" // PVC がある Namespace
	stsName = "hard"           // StatefulSet 名
)

var clientSet kubernetes.Interface

func main() {
	// ── 環境変数 → Discord セッション ──────────────
	dg, err := discordgo.New("Bot " + must("BOT_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	// ── k8s client 初期化 (in-cluster) ─────────────
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("in-cluster config: %v", err)
	}
	clientSet, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("k8s client: %v", err)
	}

	// ── Interaction ハンドラ登録 ───────────────────
	dg.AddHandler(onSlashCommand)
	dg.AddHandler(onButton)

	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}
	log.Println("🤖 bot is running")
	select {} // block forever
}

// ========== Interaction Handlers ====================================

// `/hardcorectl destroy` を受信
func onSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	d := i.ApplicationCommandData()
	if d.Name != "hardcorectl" || d.Options[0].Name != "destroy" {
		return
	}

	// DangerButton で確認
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🚨 **2度と復元できませんが world data を削除しても良いですか？**",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Yes, destroy",
						Style:    discordgo.DangerButton,
						CustomID: "destroy_yes",
					},
					discordgo.Button{
						Label:    "Cancel",
						Style:    discordgo.SecondaryButton,
						CustomID: "destroy_no",
					},
				}},
			},
		},
	})
}

// ボタン押下
func onButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	switch i.MessageComponentData().CustomID {
	case "destroy_no":
		edit(s, i, "⏹ キャンセルしました")

	case "destroy_yes":
		pvc := fmt.Sprintf("%s-%s-0", stsName, stsName)

		// ❶ ユーザへ即時メッセージ更新（3 秒以内）
		edit(s, i, "🚀 削除リクエストを受け付けました。5分経ってもサーバに入れない場合は管理者に連絡してください。")

		// ❷ 削除は裏で実行。ユーザにはもう通知しない
		go func() {
			if err := deletePVC(clientSet, pvc); err != nil {
				log.Printf("PVC delete failed: %v", err) // ログにだけ残す
			}
		}()
	}
}

// ========== Core Logic ==============================================

// PVC 削除 ＋ 完全消滅確認
func deletePVC(cs kubernetes.Interface, pvcName string) error {
	return cs.CoreV1().
		PersistentVolumeClaims(ns).
		Delete(context.TODO(), pvcName, metav1.DeleteOptions{})
}

// ========== Helpers ==================================================

func edit(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    msg,
			Components: []discordgo.MessageComponent{},
		},
	})
}

func must(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s not set", key)
	}
	return v
}
