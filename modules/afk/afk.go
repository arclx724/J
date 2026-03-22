// RoboKaty - Rose-style Telegram Group Manager Bot

package afk

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "AFK"
const HELP = `
/afk [REASON] - Mark yourself as AFK (Away From Keyboard)
/afk - Remove your AFK status (or just send any message)
/afkdel [enable/disable] - Toggle auto-delete of AFK notification messages (admin only)

When someone mentions you while AFK, they'll be notified.
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("afk", setAfk))
	dispatcher.AddHandler(utils.OnCmd("afkdel", afkDel))

	// Monitor all group messages to:
	// 1. Remove AFK when user sends a message
	// 2. Notify when AFK user is mentioned/replied to
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Text, monitorAfk), 10)

	log.Println("[AFK] ✅ Module loaded")
}

func setAfk(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil || msg.SenderChat != nil {
		_, err := msg.Reply(b, "❌ Channels can't go AFK.", nil)
		return err
	}

	if msg.From == nil { return nil }
	userID := msg.From.Id
	reason := utils.GetCommandArgs(msg)

	var afk models.Afk
	database.DB.Where(models.Afk{UserID: userID}).FirstOrCreate(&afk)

	if afk.IsAfk {
		// Toggle OFF
		afk.IsAfk = false
		afk.Reason = ""
		database.DB.Save(&afk)

		elapsed := formatDuration(time.Since(afk.AfkTime))
		text := fmt.Sprintf("👋 %s is no longer AFK!\n⏱️ Was AFK for: <b>%s</b>",
			utils.MentionHTML(userID, msg.From.FirstName), elapsed)
		_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	// Toggle ON
	afk.IsAfk = true
	afk.Reason = reason
	afk.AfkTime = time.Now()
	database.DB.Save(&afk)

	text := fmt.Sprintf("😴 %s is now AFK!", utils.MentionHTML(userID, msg.From.FirstName))
	if reason != "" {
		text += fmt.Sprintf("\n📝 Reason: %s", reason)
	}
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func monitorAfk(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return ext.ContinueGroups
	}

	if msg.From == nil { return nil }
	userID := msg.From.Id

	// 1. If THIS user is AFK — remove their AFK status
	var myAfk models.Afk
	result := database.DB.Where(models.Afk{UserID: userID, IsAfk: true}).First(&myAfk)
	if result.Error == nil {
		// Don't remove if the message IS the /afk command
		if !utils.CommandFilter("afk")(msg) {
			elapsed := formatDuration(time.Since(myAfk.AfkTime))
			myAfk.IsAfk = false
			myAfk.Reason = ""
			database.DB.Save(&myAfk)

			notice, _ := msg.Reply(b,
				fmt.Sprintf("👋 %s is no longer AFK! Was AFK for <b>%s</b>.",
					utils.MentionHTML(userID, msg.From.FirstName), elapsed),
				&gotgbot.SendMessageOpts{ParseMode: "HTML"})

			// Auto-delete notice after 5s
			if notice != nil {
				time.AfterFunc(5*time.Second, func() {
					_, _ = b.DeleteMessage(msg.Chat.Id, notice.MessageId, nil)
				})
			}
		}
	}

	// 2. If replying to an AFK user — notify
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		repliedID := msg.ReplyToMessage.From.Id
		if repliedID == userID {
			return ext.ContinueGroups // replying to yourself
		}

		var repliedAfk models.Afk
		result := database.DB.Where(models.Afk{UserID: repliedID, IsAfk: true}).First(&repliedAfk)
		if result.Error == nil {
			elapsed := formatDuration(time.Since(repliedAfk.AfkTime))
			text := fmt.Sprintf("😴 %s is AFK right now!\n⏱️ Since: <b>%s</b>",
				utils.MentionHTML(repliedID, msg.ReplyToMessage.From.FirstName), elapsed)
			if repliedAfk.Reason != "" {
				text += fmt.Sprintf("\n📝 Reason: %s", repliedAfk.Reason)
			}
			notice, _ := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
			if notice != nil {
				time.AfterFunc(5*time.Second, func() {
					_, _ = b.DeleteMessage(msg.Chat.Id, notice.MessageId, nil)
				})
			}
		}
	}

	// 3. Check if any @mentioned user is AFK
	if msg.Entities != nil {
		for _, entity := range msg.Entities {
			if entity.Type == "mention" {
				username := msg.Text[entity.Offset+1 : entity.Offset+entity.Length]
				checkMentionAfk(b, msg, username)
			}
		}
	}

	return ext.ContinueGroups
}

func checkMentionAfk(b *gotgbot.Bot, msg *gotgbot.Message, username string) {
	chat, err := b.GetChat("@"+username, nil)
	if err != nil {
		return
	}

	var afk models.Afk
	result := database.DB.Where(models.Afk{UserID: chat.Id, IsAfk: true}).First(&afk)
	if result.Error != nil {
		return
	}

	elapsed := formatDuration(time.Since(afk.AfkTime))
	text := fmt.Sprintf("😴 @%s is AFK!\n⏱️ Since: <b>%s</b>", username, elapsed)
	if afk.Reason != "" {
		text += fmt.Sprintf("\n📝 Reason: %s", afk.Reason)
	}
	notice, _ := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	if notice != nil {
		time.AfterFunc(5*time.Second, func() {
			_, _ = b.DeleteMessage(msg.Chat.Id, notice.MessageId, nil)
		})
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	parts := []string{}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", s))
	}
	return strings.Join(parts, " ")
}

// afkDelEnabled tracks per-chat auto-delete setting for AFK messages
var afkDelEnabled = map[int64]bool{}

// afkDel — /afkdel [enable/disable] — toggle auto-delete of AFK notification messages
// Mirrors: @app.on_cmd("afkdel", group_only=True) in afk.py
func afkDel(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}

	arg := strings.ToLower(utils.GetCommandArgs(msg))
	if arg == "" {
		current := afkDelEnabled[chat.Id]
		status := "enabled ✅"
		if !current {
			status = "disabled ❌"
		}
		_, err := msg.Reply(b, fmt.Sprintf("ℹ️ AFK auto-delete is currently <b>%s</b>.", status), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	var enable bool
	switch arg {
	case "enable", "on", "yes":
		enable = true
	case "disable", "off", "no":
		enable = false
	default:
		_, err := msg.Reply(b, "❌ Usage: /afkdel [enable/disable]", nil)
		return err
	}

	afkDelEnabled[chat.Id] = enable
	status := "disabled ❌"
	if enable {
		status = "enabled ✅"
	}
	_, err := msg.Reply(b, fmt.Sprintf("AFK auto-delete is now <b>%s</b>.", status), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}
