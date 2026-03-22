// RoboKaty - Rose-style Telegram Group Manager Bot
// Language selection for the bot (currently supports English only — extensible)

package lang_setting

import (
	"fmt"
	"log"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"

	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "LangSetting"
const HELP = `
/setlang - Change bot language for this chat

Currently supported languages:
🇬🇧 English (en-US)
`

// ChatLang stores the selected language per chat (in-memory, backed by DB future)
var ChatLang = map[int64]string{}

// Available languages
var availableLangs = map[string]string{
	"en-US": "🇬🇧 English",
	"id-ID": "🇮🇩 Indonesian",
}

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("setlang", setLang))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("setlangsel_"), langSelectCB))
	log.Println("[LangSetting] ✅ Module loaded")
}

func GetChatLang(chatID int64) string {
	if lang, ok := ChatLang[chatID]; ok {
		return lang
	}
	return "en-US"
}

func setLang(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if !utils.IsPrivateChat(chat) {
		if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
			_, err := msg.Reply(b, "❌ You need can_change_info permission to change the language.", nil)
			return err
		}
	}

	// Build language selection keyboard
	var rows [][]gotgbot.InlineKeyboardButton
	for code, name := range availableLangs {
		current := GetChatLang(chat.Id)
		label := name
		if code == current {
			label += " ✅"
		}
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			utils.Btn(label, fmt.Sprintf("setlangsel_%d_%s", chat.Id, code)),
		})
	}

	keyboard := utils.IKB(rows)
	currentLang := availableLangs[GetChatLang(chat.Id)]
	_, err := msg.Reply(b,
		fmt.Sprintf("🌐 <b>Language Settings</b>\n\nCurrent language: <b>%s</b>\n\nSelect a language:", currentLang),
		&gotgbot.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard},
	)
	return err
}

func langSelectCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	from := cq.From

	// Parse: "setlangsel_<chatID>_<langCode>"
	data := strings.TrimPrefix(cq.Data, "setlangsel_")
	parts := strings.SplitN(data, "_", 2)
	if len(parts) != 2 {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "Invalid data.", ShowAlert: true})
		return err
	}

	var chatID int64
	fmt.Sscanf(parts[0], "%d", &chatID)
	langCode := parts[1]

	chat := cq.Message.GetChat()

	// Permission check
	if chat.Type != "private" {
		if !utils.HasPermission(b, chat.Id, from.Id, "can_change_info") {
			_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
				Text:      "❌ You need can_change_info permission.",
				ShowAlert: true,
			})
			return err
		}
	}

	if _, ok := availableLangs[langCode]; !ok {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Unknown language.", ShowAlert: true})
		return err
	}

	// Save language
	ChatLang[chatID] = langCode
	_ = database.DB // future: persist to DB

	langName := availableLangs[langCode]
	_, _, _ = cq.Message.EditText(b,
		fmt.Sprintf("✅ Language set to <b>%s</b>!", langName),
		&gotgbot.EditMessageTextOpts{ParseMode: "HTML"},
	)
	_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: fmt.Sprintf("Language changed to %s!", langName),
	})
	return err
}
