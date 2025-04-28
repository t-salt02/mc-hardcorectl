package main

import (
	"context"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	ns      = "minecraft-hard"    // PVC がある Namespace
	stsName = "hard"              // StatefulSet 名
)

func main() {
	dg, err := discordgo.New("Bot " + must("BOT_TOKEN"))
	if err != nil { log.Fatal(err) }

	// ─── Interaction ハンドラ ───────────────────────────
	dg.AddHandler(onSlashCommand)
	dg.AddHandler(onButton)

	if err := dg.Open(); err != nil { log.Fatal(err) }
	log.Println("🤖 bot is running")
	select {}
}

// ─── Slash Command (`/hardcorectl destroy`) 受信 ──────────
func onSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand { return }
	d := i.ApplicationCommandData()
	if d.Name != "hardcorectl" || d.Options[0].Name != "destroy" { return }

	// DangerButton で確認
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🚨 **2度と復元できませんがworld dataを削除しても良いですか？**",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Yes, destroy",
						Style:    discordgo.DangerButton, // 赤色ボタン :contentReference[oaicite:0]{index=0}
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

// ─── ボタン押下ハンドラ ─────────────────────────────
func onButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent { return }

	switch i.MessageComponentData().CustomID {
	case "destroy_no":
		edit(i, "⏹ キャンセルしました")
	case "destroy_yes":
		pvc := stsName + "-" + stsName + "-0"
		err := deletePVC(clientSet, pvc)
		if err != nil {
			edit(i, "❌ 失敗: "+err.Error())
		} else {
			edit(i, "✅ PVC "+pvc+" を削除しました")
		}
	
}

// ─── PVC 削除（client-go）─────────────────────────────
func deletePVC(cs kubernetes.Interface, pvcName string) error {
    // ❶ Delete リクエスト
    if err := cs.CoreV1().
        PersistentVolumeClaims(ns).
        Delete(context.TODO(), pvcName, metav1.DeleteOptions{}); err != nil {

        // すでに無い場合はユーザーに分かるように
        if apierrors.IsNotFound(err) {
            return fmt.Errorf("PVC %s はすでに存在しません", pvcName)
        }
        return err // それ以外は失敗
    }

    // ❷ 実際に消えたかポーリング確認
    return wait.PollImmediate(1*time.Second, 15*time.Second, func() (bool, error) {
        _, err := cs.CoreV1().
            PersistentVolumeClaims(ns).
            Get(context.TODO(), pvcName, metav1.GetOptions{})

        switch {
        case apierrors.IsNotFound(err):
            // Get して NotFound ＝ 完全に削除完了
            return true, nil
        case err != nil:
            // API エラー発生 → リトライ打ち切り
            return false, err
        default:
            // まだ残っている → 継続ポーリング
            return false, nil
        }
    })
}
