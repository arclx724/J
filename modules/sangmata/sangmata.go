// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Tracks username/name history of users

package sangmata

import (
	"fmt"
	"log"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"

	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Sangmata"
const HELP = `
/sangmata - Get username/name history of a user (reply to their message)

Note: History is only available for users who have sent messages
while RoboKaty was active in the chat.
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("sangmata", sangmata))
	dispatcher.AddHandler(utils.OnCmd("history", sangmata))

	// Track all messages to build history
	dispatcher.AddHandlerToGroup(handlers.NewChatMemberUpdated(nil, trackUser), 99)

	log.Println("[Sangmata] ✅ Module loaded")
}

func sangmata(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage

	var targetID int64
	var targetName string

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		targetID = msg.ReplyToMessage.From.Id
		targetName = msg.ReplyToMessage.From.FirstName
	} else {
		target, err := utils.ExtractUser(b, ctx)
		if err != nil || target == nil {
			_, err = msg.Reply(b, "❌ Reply to a user's message or provide @username.", nil)
			return err
		}
		targetID = target.Id
		targetName = target.FirstName
	}

	var userChats []models.UserChat
	database.DB.Where("user_id = ? AND username != ''", targetID).Find(&userChats)

	if len(userChats) == 0 {
		_, err := msg.Reply(b,
			fmt.Sprintf("ℹ️ No username history for %s.", utils.MentionHTML(targetID, targetName)),
			&gotgbot.SendMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	seen := map[string]bool{}
	text := fmt.Sprintf("📋 <b>Name/Username history for</b> %s\n\n", utils.MentionHTML(targetID, targetName))
	for _, uc := range userChats {
		if uc.Username != "" && !seen[uc.Username] {
			text += fmt.Sprintf("• @%s\n", uc.Username)
			seen[uc.Username] = true
		}
	}

	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func trackUser(_ *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.ChatMember == nil {
		return nil
	}
	user := ctx.ChatMember.From
	if user.Id == 0 {
		return nil
	}

	var uc models.UserChat
	database.DB.Where(models.UserChat{UserID: user.Id, ChatID: ctx.ChatMember.Chat.Id}).FirstOrCreate(&uc)
	if user.Username != "" {
		uc.Username = user.Username
		database.DB.Save(&uc)
	}
	return nil
}
