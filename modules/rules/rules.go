// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Mirrors: misskaty/plugins/rules.py — ALL commands

package rules

import (
	"fmt"
	"log"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Rules"
const HELP = `
/rules - Get the chat rules (sends in private if private mode enabled)
/setrules [TEXT] - Set the rules for this chat
/resetrules - Reset chat rules
/privaterules [yes/no] - Toggle sending rules in private chat
/setrulesbutton [TEXT] - Set custom rules button name
/resetrulesbutton - Reset rules button to default
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("rules", rulesCmd))
	dispatcher.AddHandler(utils.OnCmd("setrules", setRulesCmd))
	dispatcher.AddHandler(utils.OnCmd("resetrules", resetRulesCmd))
	dispatcher.AddHandler(utils.OnCmd("privaterules", privateRulesCmd))
	dispatcher.AddHandler(utils.OnCmd("setrulesbutton", setRulesBtnCmd))
	dispatcher.AddHandler(utils.OnCmd("resetrulesbutton", resetRulesBtnCmd))
	log.Println("[Rules] ✅ Module loaded")
}

func rulesCmd(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	// Handle deep-link: /start btnrules_<chatID>
	if utils.IsPrivateChat(chat) {
		args := utils.GetCommandArgs(msg)
		if strings.HasPrefix(args, "btnrules_") {
			var srcChatID int64
			fmt.Sscanf(strings.TrimPrefix(args, "btnrules_"), "%d", &srcChatID)
			if srcChatID != 0 {
				var r models.Rules
				if database.DB.Where("chat_id = ?", srcChatID).First(&r).Error == nil && r.Rules != "" {
					_, err := msg.Reply(b, r.Rules, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
					return err
				}
				_, err := msg.Reply(b, "📋 No rules set for that chat.", nil)
				return err
			}
		}
	}

	var r models.Rules
	database.DB.Where("chat_id = ?", chat.Id).First(&r)

	if r.Rules == "" {
		_, err := msg.Reply(b, "📋 No rules set for this chat.", nil)
		return err
	}

	// Private rules mode — send button to DM
	if !utils.IsPrivateChat(chat) && r.PrivateMode {
		btnName := r.BtnName
		if btnName == "" {
			btnName = "📋 Rules"
		}
		keyboard := utils.IKB([][]gotgbot.InlineKeyboardButton{
			{utils.BtnURL(btnName, fmt.Sprintf("https://t.me/%s?start=btnrules_%d", b.Username, chat.Id))},
		})
		_, err := msg.Reply(b, "Tap the button below to read the rules in private.", &gotgbot.SendMessageOpts{ReplyMarkup: keyboard})
		return err
	}

	_, err := msg.Reply(b, fmt.Sprintf("📋 <b>Rules for %s:</b>\n\n%s", chat.Title, r.Rules), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func setRulesCmd(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}
	text := utils.GetCommandArgs(msg)
	if text == "" {
		_, err := msg.Reply(b, "❌ Usage: /setrules [TEXT]", nil)
		return err
	}
	var r models.Rules
	database.DB.Where(models.Rules{ChatID: chat.Id}).FirstOrCreate(&r)
	r.Rules = text
	database.DB.Save(&r)
	_, err := msg.Reply(b, "✅ Rules updated!", nil)
	return err
}

func resetRulesCmd(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}
	database.DB.Where("chat_id = ?", chat.Id).Delete(&models.Rules{})
	_, err := msg.Reply(b, "✅ Rules reset.", nil)
	return err
}

func privateRulesCmd(b *gotgbot.Bot, ctx *ext.Context) error {
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
		var r models.Rules
		database.DB.Where("chat_id = ?", chat.Id).First(&r)
		status := "disabled"
		if r.PrivateMode {
			status = "enabled"
		}
		_, err := msg.Reply(b, fmt.Sprintf("ℹ️ Private rules is currently <b>%s</b>.", status), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	var enable bool
	switch arg {
	case "yes", "on", "true", "enable", "enabled":
		enable = true
	case "no", "off", "false", "disable", "disabled":
		enable = false
	default:
		_, err := msg.Reply(b, "❌ Usage: /privaterules [yes/no/on/off]", nil)
		return err
	}

	var r models.Rules
	database.DB.Where(models.Rules{ChatID: chat.Id}).FirstOrCreate(&r)
	r.PrivateMode = enable
	database.DB.Save(&r)

	status := "disabled ❌"
	if enable {
		status = "enabled ✅"
	}
	_, err := msg.Reply(b, fmt.Sprintf("Private rules is now <b>%s</b>.", status), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func setRulesBtnCmd(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}
	text := utils.GetCommandArgs(msg)
	if text == "" {
		_, err := msg.Reply(b, "❌ Usage: /setrulesbutton [BUTTON TEXT]", nil)
		return err
	}
	var r models.Rules
	database.DB.Where(models.Rules{ChatID: chat.Id}).FirstOrCreate(&r)
	r.BtnName = text
	database.DB.Save(&r)
	_, err := msg.Reply(b, fmt.Sprintf("✅ Rules button set to: <b>%s</b>", text), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func resetRulesBtnCmd(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}
	database.DB.Model(&models.Rules{}).Where("chat_id = ?", chat.Id).Update("btn_name", "")
	_, err := msg.Reply(b, "✅ Rules button reset to default.", nil)
	return err
}
