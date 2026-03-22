// RoboKaty - Rose-style Telegram Group Manager Bot
// modules/broadcast/broadcast.go — Mirrors misskaty/plugins/broadcast.py

package broadcast

import (
	"fmt"
	"log"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Broadcast"
const HELP = `
/broadcast [TEXT] - Broadcast a message to all chats (sudo only)
Reply to a message with /broadcast to forward it.
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("broadcast", broadcast))
	log.Println("[Broadcast] ✅ Module loaded")
}

func broadcast(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	sender := ctx.EffectiveSender

	// Sudo only
	if !config.IsSudo(sender.Id()) {
		return nil
	}

	var text string
	var forwardMsgID int64
	var forwardChatID int64

	if msg.ReplyToMessage != nil {
		forwardMsgID = msg.ReplyToMessage.MessageId
		forwardChatID = msg.Chat.Id
	} else {
		text = utils.GetCommandArgs(msg)
		if text == "" {
			_, err := msg.Reply(b, "❌ Usage: /broadcast [TEXT] or reply to a message.", nil)
			return err
		}
	}

	// Get all unique chat IDs from user_chat table
	var userChats []models.UserChat
	database.DB.Select("DISTINCT chat_id").Where("chat_id < 0").Find(&userChats)

	if len(userChats) == 0 {
		_, err := msg.Reply(b, "ℹ️ No chats registered yet.", nil)
		return err
	}

	progress, _ := msg.Reply(b, fmt.Sprintf("📡 Broadcasting to <b>%d</b> chats...", len(userChats)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})

	sent, failed := 0, 0
	for _, uc := range userChats {
		var err error
		if forwardMsgID != 0 {
			_, err = b.ForwardMessage(uc.ChatID, forwardChatID, forwardMsgID, nil)
		} else {
			_, err = b.SendMessage(uc.ChatID, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		}
		if err != nil {
			failed++
		} else {
			sent++
		}
	}

	result := fmt.Sprintf(
		"📡 <b>Broadcast Complete</b>\n"+
			"✅ Sent: <b>%d</b>\n"+
			"❌ Failed: <b>%d</b>",
		sent, failed,
	)

	if progress != nil {
		_, _, _ = progress.EditText(b, result, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	return nil
}
