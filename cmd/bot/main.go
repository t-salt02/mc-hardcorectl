package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
		err := deletePVC(clientSet, pvc)
		if err != nil {
			edit(s, i, "❌ 失敗: "+err.Error())
		} else {
			edit(s, i, "✅ PVC "+pvc+" を削除しました")
		}
	}
}

// ========== Core Logic ==============================================

// PVC 削除 ＋ 完全消滅確認
func deletePVC(cs kubernetes.Interface, pvcName string) error {
	// ❶ Delete
	if err := cs.CoreV1().
		PersistentVolumeClaims(ns).
		Delete(context.TODO(), pvcName, metav1.DeleteOptions{}); err != nil {

		if apierrors.IsNotFound(err) {
			return fmt.Errorf("PVC %s はすでに存在しません", pvcName)
		}
		return err
	}

	// ❷ NotFound になるまでポーリング
	return wait.PollImmediate(1*time.Second, 15*time.Second, func() (bool, error) {
		_, err := cs.CoreV1().
			PersistentVolumeClaims(ns).
			Get(context.TODO(), pvcName, metav1.GetOptions{})

		switch {
		case apierrors.IsNotFound(err):
			return true, nil
		case err != nil:
			return false, err
		default:
			return false, nil
		}
	})
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
