// RoboKaty - Rose-style Telegram Group Manager Bot

package autoapprove

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

const MODULE = "AutoApprove"
const HELP = `
/approve [user] - Approve a user (bypass join request approval)
/disapprove [user] - Remove approval from a user
/approved - List all approved users

Approved users can bypass join request and are never restricted.
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("approve", approve))
	dispatcher.AddHandler(utils.OnCmd("disapprove", disapprove))
	dispatcher.AddHandler(utils.OnCmd("approved", listApproved))

	// Auto-approve join requests for approved users
	dispatcher.AddHandler(handlers.NewChatJoinRequest(nil, handleJoinRequest))

	log.Println("[AutoApprove] ✅ Module loaded")
}

func approve(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_invite_users") {
		_, err := msg.Reply(b, "❌ You need can_invite_users permission.", nil)
		return err
	}

	target, err := utils.ExtractUser(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}

	var approval models.Approval
	database.DB.Where(models.Approval{ChatID: chat.Id, UserID: target.Id}).FirstOrCreate(&approval)

	text := fmt.Sprintf("✅ %s is now <b>approved</b> in this chat!", utils.MentionHTML(target.Id, target.FirstName))
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func disapprove(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_invite_users") {
		_, err := msg.Reply(b, "❌ You need can_invite_users permission.", nil)
		return err
	}

	target, err := utils.ExtractUser(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}

	database.DB.Where(models.Approval{ChatID: chat.Id, UserID: target.Id}).Delete(&models.Approval{})

	text := fmt.Sprintf("❌ %s approval <b>removed</b>.", utils.MentionHTML(target.Id, target.FirstName))
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func listApproved(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	var approvals []models.Approval
	database.DB.Where("chat_id = ?", chat.Id).Find(&approvals)

	if len(approvals) == 0 {
		_, err := msg.Reply(b, "📋 No approved users in this chat.", nil)
		return err
	}

	text := fmt.Sprintf("✅ <b>Approved users in %s:</b>\n\n", chat.Title)
	for _, a := range approvals {
		member, err := b.GetChatMember(chat.Id, a.UserID, nil)
		name := fmt.Sprintf("User %d", a.UserID)
		if err == nil {
			u := member.GetUser()
			name = u.FirstName
		}
		text += fmt.Sprintf("• %s\n", utils.MentionHTML(a.UserID, name))
	}

	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func handleJoinRequest(b *gotgbot.Bot, ctx *ext.Context) error {
	req := ctx.ChatJoinRequest
	if req == nil {
		return nil
	}

	var approval models.Approval
	result := database.DB.Where(models.Approval{ChatID: req.Chat.Id, UserID: req.From.Id}).First(&approval)
	if result.Error == nil {
		// Auto-approve
		_, _ = b.ApproveChatJoinRequest(req.Chat.Id, req.From.Id, nil)
	}
	return nil
}
