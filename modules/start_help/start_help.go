// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Mirrors: misskaty/plugins/start_help.py

package start_help

import (
	"fmt"
	"log"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Start"

// ModuleHelp stores help text per module name
var ModuleHelp = map[string]string{}

func RegisterHelp(module, help string) {
	ModuleHelp[module] = help
}

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("start", startCmd))
	dispatcher.AddHandler(utils.OnCmd("help", helpCmd))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("help_"), helpCB))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("close"), closeCB))
	log.Println("[Start/Help] ✅ Module loaded")
}

func startCmd(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if !utils.IsPrivateChat(chat) {
		keyboard := utils.IKB([][]gotgbot.InlineKeyboardButton{
			{utils.BtnURL("📖 Help in Private", fmt.Sprintf("https://t.me/%s?start=help", b.Username))},
		})
		_, err := msg.Reply(b,
			fmt.Sprintf("👋 Hi! I'm <b>RoboKaty</b>!\n📢 Support: @%s", config.SupportChat),
			&gotgbot.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard},
		)
		return err
	}

	keyboard := utils.IKB([][]gotgbot.InlineKeyboardButton{
		{
			utils.BtnURL("➕ Add to Group", fmt.Sprintf("https://t.me/%s?startgroup=true", b.Username)),
			utils.BtnURL("📢 Support", fmt.Sprintf("https://t.me/%s", config.SupportChat)),
		},
		{utils.Btn("📖 Help Menu", "help_menu")},
	})

	text := fmt.Sprintf(
		"👋 <b>Hello!</b> I'm <b>RoboKaty</b> — your cute & powerful group management bot!\n\n"+
			"🛡️ <b>Features:</b>\n"+
			"• Bans, Mutes, Warns\n"+
			"• Notes\n"+
			"• Welcome messages\n"+
			"• Locks & Blacklists\n"+
			"• Federations (cross-group bans)\n"+
			"• AFK, Karma, Night Mode\n"+
			"• And much more!\n\n"+
			"📢 Support: @%s",
		config.SupportChat,
	)
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func helpCmd(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if !utils.IsPrivateChat(chat) {
		keyboard := utils.IKB([][]gotgbot.InlineKeyboardButton{
			{utils.BtnURL("📖 Help in Private", fmt.Sprintf("https://t.me/%s?start=help", b.Username))},
		})
		_, err := msg.Reply(b, "Click below for help!", &gotgbot.SendMessageOpts{ReplyMarkup: keyboard})
		return err
	}
	return sendHelpMenu(b, msg)
}

func sendHelpMenu(b *gotgbot.Bot, msg *gotgbot.Message) error {
	keyboard := buildHelpKeyboard()
	_, err := msg.Reply(b,
		"📖 <b>RoboKaty Help Menu</b>\n\nSelect a module:",
		&gotgbot.SendMessageOpts{
			ParseMode:   "HTML",
			ReplyMarkup: &gotgbot.InlineKeyboardMarkup{InlineKeyboard: keyboard},
		},
	)
	return err
}

func buildHelpKeyboard() [][]gotgbot.InlineKeyboardButton {
	modules := []string{
		"Admin", "Notes", "Welcome",
		"Locks", "Rules", "Blacklist", "AFK",
		"Karma", "Nightmode", "Ping", "Sed",
		"Broadcast", "Stickers", "Anime", "Urban",
		"Quotly", "Sangmata", "Federation", "Dev",
		"LangSetting", "JSON", "AutoApprove", "InKick",
	}

	var rows [][]gotgbot.InlineKeyboardButton
	var row []gotgbot.InlineKeyboardButton
	for i, mod := range modules {
		row = append(row, utils.Btn(mod, "help_"+strings.ToLower(mod)))
		if (i+1)%3 == 0 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{utils.Btn("❌ Close", "close")})
	return rows
}

func helpCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	module := strings.TrimPrefix(cq.Data, "help_")

	if module == "menu" || module == "back" {
		keyboard := buildHelpKeyboard()
		_, _, _ = cq.Message.EditText(b,
			"📖 <b>RoboKaty Help Menu</b>\n\nSelect a module:",
			&gotgbot.EditMessageTextOpts{
				ParseMode:   "HTML",
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{InlineKeyboard: keyboard},
			},
		)
		_, err := cq.Answer(b, nil)
		return err
	}

	// Find help text (case-insensitive)
	var helpText string
	for mod, help := range ModuleHelp {
		if strings.EqualFold(mod, module) {
			helpText = strings.TrimSpace(help)
			break
		}
	}
	if helpText == "" {
		helpText = "No help available for this module yet."
	}

	backKeyboard := utils.IKB([][]gotgbot.InlineKeyboardButton{
		{utils.Btn("◀️ Back", "help_back"), utils.Btn("❌ Close", "close")},
	})

	// Use Capitalize instead of deprecated strings.Title
	displayName := utils.Capitalize(module)

	_, _, _ = cq.Message.EditText(b,
		fmt.Sprintf("📖 <b>%s</b>\n\n%s", displayName, helpText),
		&gotgbot.EditMessageTextOpts{
			ParseMode:   "HTML",
			ReplyMarkup: backKeyboard,
		},
	)
	_, err := cq.Answer(b, nil)
	return err
}

func closeCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	_, _ = cq.Message.Delete(b, nil)
	_, err := cq.Answer(b, nil)
	return err
}
