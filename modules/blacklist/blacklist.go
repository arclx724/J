// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved

package blacklist

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Blacklist"
const HELP = `
/blacklist [WORD] - Blacklist a word (auto-delete + warn on match)
/blacklisted - Show all blacklisted words
/whitelist [WORD] - Remove a word from blacklist

Users who send a blacklisted word get warned (3 warns = ban).
Admins and sudo users are exempt.
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("blacklist", addBlacklist))
	dispatcher.AddHandler(utils.OnCmd("blacklisted", listBlacklisted))
	dispatcher.AddHandler(utils.OnCmd("whitelist", removeBlacklist))
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Text, checkBlacklist), 8)
	log.Println("[Blacklist] ✅ Module loaded")
}

func addBlacklist(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}
	word := strings.ToLower(strings.TrimSpace(utils.GetCommandArgs(msg)))
	if word == "" {
		_, err := msg.Reply(b, "❌ Usage: /blacklist [WORD]", nil)
		return err
	}
	database.DB.Where(models.Blacklist{ChatID: chat.Id, Trigger: word}).FirstOrCreate(&models.Blacklist{ChatID: chat.Id, Trigger: word, Action: "warn"})
	_, err := msg.Reply(b, fmt.Sprintf("🚫 Blacklisted: <code>%s</code>", word), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func listBlacklisted(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	var list []models.Blacklist
	database.DB.Where("chat_id = ?", chat.Id).Find(&list)
	if len(list) == 0 {
		_, err := msg.Reply(b, "📋 No blacklisted words.", nil)
		return err
	}
	text := fmt.Sprintf("🚫 <b>Blacklisted in %s:</b>\n\n", chat.Title)
	for _, bl := range list {
		text += fmt.Sprintf("• <code>%s</code>\n", bl.Trigger)
	}
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func removeBlacklist(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}
	word := strings.ToLower(strings.TrimSpace(utils.GetCommandArgs(msg)))
	if word == "" {
		_, err := msg.Reply(b, "❌ Usage: /whitelist [WORD]", nil)
		return err
	}
	result := database.DB.Where(models.Blacklist{ChatID: chat.Id, Trigger: word}).Delete(&models.Blacklist{})
	if result.RowsAffected == 0 {
		_, err := msg.Reply(b, fmt.Sprintf("❌ <code>%s</code> is not blacklisted.", word), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}
	_, err := msg.Reply(b, fmt.Sprintf("✅ Whitelisted: <code>%s</code>", word), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func checkBlacklist(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if utils.IsPrivateChat(chat) || msg.From == nil {
		return ext.ContinueGroups
	}
	if config.IsSudo(msg.From.Id) || utils.IsInAdminList(b, chat.Id, msg.From.Id) {
		return ext.ContinueGroups
	}

	text := strings.ToLower(msg.Text)
	if text == "" {
		return ext.ContinueGroups
	}

	var list []models.Blacklist
	database.DB.Where("chat_id = ?", chat.Id).Find(&list)

	for _, bl := range list {
		pattern := `(?i)(^|[^\w])` + regexp.QuoteMeta(bl.Trigger) + `($|[^\w])`
		if matched, _ := regexp.MatchString(pattern, text); !matched {
			continue
		}

		_, _ = b.DeleteMessage(chat.Id, msg.MessageId, nil)

		var warn models.Warn
		database.DB.Where(models.Warn{ChatID: chat.Id, UserID: msg.From.Id}).FirstOrCreate(&warn)
		warn.Count++
		database.DB.Save(&warn)

		if warn.Count >= 3 {
			_, _ = b.BanChatMember(chat.Id, msg.From.Id, nil)
			database.DB.Delete(&warn)
			_, _ = b.SendMessage(chat.Id,
				fmt.Sprintf("🚫 %s <b>banned</b> — triggered blacklisted word: <code>%s</code>",
					utils.MentionHTML(msg.From.Id, msg.From.FirstName), bl.Trigger),
				&gotgbot.SendMessageOpts{ParseMode: "HTML"})
		} else {
			notice, _ := b.SendMessage(chat.Id,
				fmt.Sprintf("⚠️ %s triggered: <code>%s</code>\nWarns: <b>%d/3</b>",
					utils.MentionHTML(msg.From.Id, msg.From.FirstName), bl.Trigger, warn.Count),
				&gotgbot.SendMessageOpts{ParseMode: "HTML"})
			if notice != nil {
				cID, mID := chat.Id, notice.MessageId
				time.AfterFunc(5*time.Second, func() { _, _ = b.DeleteMessage(cID, mID, nil) })
			}
		}
		return nil
	}
	return ext.ContinueGroups
}
