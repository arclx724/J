// RoboKaty — modules/admin/admin.go
// Mirrors: admin.py + ban_user_or_chat.py — gotgbot v2 compatible

package admin

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Admin"
const HELP = `
/ban - Ban a user
/dban - Delete replied message and ban sender
/tban [user] [time] [reason] - Temporary ban (e.g. /tban @user 1d reason)
/unban - Unban a user
/listban - Ban user from chats listed in a message link (sudo only)
/listunban - Unban user from chats listed in a message link (sudo only)

/warn - Warn a user (3 warns = ban)
/dwarn - Delete replied and warn sender
/rmwarn - Remove all warnings
/warns - Check user warnings

/kick - Kick a user
/dkick - Delete replied and kick sender
/softban - Ban then unban (removes messages, user can rejoin)
/ban_ghosts - Ban all deleted accounts

/mute - Mute a user
/tmute [user] [time] - Temporary mute
/unmute - Unmute a user

/del - Delete replied message
/purge - Purge from replied message to current
/purge [n] - Purge n messages from replied message

/promote - Promote a member
/fullpromote - Promote with all available rights
/demote - Demote a member
/pin - Pin a message
/unpin - Unpin a message

/set_chat_title - Change group title
/set_chat_photo - Change group photo (reply to image)
/set_user_title - Change admin custom title

/mentionall - Mention all members
/report | @admin | @admins - Report to admins
`

const maxWarns = 3

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmds([]string{"ban", "dban", "tban"}, banUser))
	dispatcher.AddHandler(utils.OnCmd("unban", unbanUser))
	dispatcher.AddHandler(utils.OnCmd("listban", listBan))
	dispatcher.AddHandler(utils.OnCmd("listunban", listUnban))
	dispatcher.AddHandler(utils.OnCmds([]string{"warn", "dwarn"}, warnUser))
	dispatcher.AddHandler(utils.OnCmd("rmwarn", removeWarnings))
	dispatcher.AddHandler(utils.OnCmd("warns", checkWarns))
	dispatcher.AddHandler(utils.OnCmds([]string{"kick", "dkick"}, kickUser))
	dispatcher.AddHandler(utils.OnCmd("softban", softban))
	dispatcher.AddHandler(utils.OnCmd("ban_ghosts", banGhosts))
	dispatcher.AddHandler(utils.OnCmds([]string{"mute", "tmute"}, muteUser))
	dispatcher.AddHandler(utils.OnCmd("unmute", unmuteUser))
	dispatcher.AddHandler(utils.OnCmd("del", deleteMsg))
	dispatcher.AddHandler(utils.OnCmd("purge", purgeMessages))
	dispatcher.AddHandler(utils.OnCmds([]string{"promote", "fullpromote"}, promoteUser))
	dispatcher.AddHandler(utils.OnCmd("demote", demoteUser))
	dispatcher.AddHandler(utils.OnCmds([]string{"pin", "unpin"}, pinMessage))
	dispatcher.AddHandler(utils.OnCmd("set_chat_title", setChatTitle))
	dispatcher.AddHandler(utils.OnCmd("set_chat_photo", setChatPhoto))
	dispatcher.AddHandler(utils.OnCmd("set_user_title", setUserTitle))
	dispatcher.AddHandler(utils.OnCmd("mentionall", mentionAll))
	dispatcher.AddHandler(utils.OnCmd("report", reportUser))
	dispatcher.AddHandler(handlers.NewMessage(message.Text, handleAtAdmin))
	dispatcher.AddHandler(handlers.NewChatMemberUpdated(nil, refreshAdminCache))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("unban_"), unbanCB))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("unmute_"), unmuteCB))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("unwarn_"), unwarnCB))
	log.Println("[Admin] ✅ Module loaded")
}

// ─── BAN ─────────────────────────────────────────────────────────────────────

func banUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.IsGroupChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}
	if !utils.IsBotAdmin(b, chat.Id) {
		_, err := msg.Reply(b, "❌ I need to be an admin with ban rights.", nil)
		return err
	}

	target, reason, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Reply to a message or provide @username/user_id.", nil)
		return err
	}
	if target.Id == b.Id {
		_, err = msg.Reply(b, "😄 Won't ban myself!", nil)
		return err
	}
	if config.IsSudo(target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't ban a sudo user.", nil)
		return err
	}
	if utils.IsInAdminList(b, chat.Id, target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't ban an admin.", nil)
		return err
	}

	cmd := utils.GetCommand(msg)

	// /dban — delete replied message
	if strings.HasPrefix(cmd, "d") && msg.ReplyToMessage != nil {
		_, _ = b.DeleteMessage(chat.Id, msg.ReplyToMessage.MessageId, nil)
	}

	// /tban — temporary ban
	if cmd == "tban" {
		return handleTBan(b, ctx, chat, msg, target, reason)
	}

	text := fmt.Sprintf(
		"🔨 <b>Banned!</b>\n👤 User: %s\n🆔 ID: <code>%d</code>\n👮 By: %s",
		utils.MentionHTML(target.Id, target.FirstName), target.Id, senderMention(ctx),
	)
	if reason != "" {
		text += fmt.Sprintf("\n📝 Reason: %s", reason)
	}

	if _, banErr := b.BanChatMember(chat.Id, target.Id, nil); banErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", banErr.Error()), nil)
		return err
	}
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: utils.SingleBtn("🚨 Unban", fmt.Sprintf("unban_%d", target.Id)),
	})
	return err
}

func handleTBan(b *gotgbot.Bot, ctx *ext.Context, chat *gotgbot.Chat, msg *gotgbot.Message, target *gotgbot.User, reason string) error {
	parts := strings.SplitN(strings.TrimSpace(reason), " ", 2)
	timeVal := parts[0]
	tempReason := ""
	if len(parts) > 1 {
		tempReason = parts[1]
	}

	until, err := utils.TimeConverter(timeVal)
	if err != nil {
		_, err = msg.Reply(b, "❌ Invalid time. Use: 1h 30m 2d 1w", nil)
		return err
	}

	text := fmt.Sprintf(
		"⏰ <b>Temporarily Banned!</b>\n👤 %s\n🆔 <code>%d</code>\n⏳ Duration: <b>%s</b>\n👮 By: %s",
		utils.MentionHTML(target.Id, target.FirstName), target.Id, timeVal, senderMention(ctx),
	)
	if tempReason != "" {
		text += fmt.Sprintf("\n📝 Reason: %s", tempReason)
	}

	if _, banErr := b.BanChatMember(chat.Id, target.Id, &gotgbot.BanChatMemberOpts{UntilDate: until.Unix()}); banErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", banErr.Error()), nil)
		return err
	}
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func unbanUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.IsGroupChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	target, _, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Provide @username or user_id.", nil)
		return err
	}

	if _, unbanErr := b.UnbanChatMember(chat.Id, target.Id, nil); unbanErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", unbanErr.Error()), nil)
		return err
	}
	_, err = msg.Reply(b, fmt.Sprintf("✅ Unbanned %s!", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

var tgLinkRe = regexp.MustCompile(`(?:https?://)?t(?:elegram)?\.me/(\w+)/(\d+)`)

func listBan(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if !config.IsSudo(ctx.EffectiveSender.Id()) {
		return nil
	}
	target, linkReason, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil || linkReason == "" {
		_, err = msg.Reply(b, "❌ Usage: /listban @user MSG_LINK reason", nil)
		return err
	}
	parts := strings.SplitN(linkReason, " ", 2)
	if len(parts) < 2 {
		_, err = msg.Reply(b, "❌ Provide a reason after the message link.", nil)
		return err
	}
	matches := tgLinkRe.FindStringSubmatch(parts[0])
	if matches == nil {
		_, err = msg.Reply(b, "❌ Invalid Telegram message link.", nil)
		return err
	}

	usernameRe := regexp.MustCompile(`@\w+`)
	// We can't get message text via Bot API without forwarding — inform user
	_, err = msg.Reply(b, "ℹ️ listban requires Pyrogram userbot to fetch message text. Use /fban via federation instead.", nil)
	return err
}

func listUnban(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if !config.IsSudo(ctx.EffectiveSender.Id()) {
		return nil
	}
	_, err := msg.Reply(b, "ℹ️ listunban requires Pyrogram userbot to fetch message text. Use /unfban via federation instead.", nil)
	return err
}

// ─── KICK ─────────────────────────────────────────────────────────────────────

func kickUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.IsGroupChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	target, reason, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if target.Id == b.Id {
		_, err = msg.Reply(b, "😄 Won't kick myself!", nil)
		return err
	}
	if config.IsSudo(target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't kick a sudo user.", nil)
		return err
	}
	if utils.IsInAdminList(b, chat.Id, target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't kick an admin.", nil)
		return err
	}

	cmd := utils.GetCommand(msg)
	if strings.HasPrefix(cmd, "d") && msg.ReplyToMessage != nil {
		_, _ = b.DeleteMessage(chat.Id, msg.ReplyToMessage.MessageId, nil)
	}

	text := fmt.Sprintf(
		"👢 <b>Kicked!</b>\n👤 %s\n🆔 <code>%d</code>\n👮 By: %s",
		utils.MentionHTML(target.Id, target.FirstName), target.Id, senderMention(ctx),
	)
	if reason != "" {
		text += fmt.Sprintf("\n📝 Reason: %s", reason)
	}

	// Kick = ban + unban
	if _, banErr := b.BanChatMember(chat.Id, target.Id, nil); banErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", banErr.Error()), nil)
		return err
	}
	time.Sleep(500 * time.Millisecond)
	_, _ = b.UnbanChatMember(chat.Id, target.Id, nil)

	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func softban(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	target, reason, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if config.IsSudo(target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't softban a sudo user.", nil)
		return err
	}

	if _, banErr := b.BanChatMember(chat.Id, target.Id, &gotgbot.BanChatMemberOpts{RevokeMessages: true}); banErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", banErr.Error()), nil)
		return err
	}
	time.Sleep(500 * time.Millisecond)
	_, _ = b.UnbanChatMember(chat.Id, target.Id, nil)

	text := fmt.Sprintf("🧹 <b>Soft Banned</b> %s — messages purged, user can rejoin.", utils.MentionHTML(target.Id, target.FirstName))
	if reason != "" {
		text += fmt.Sprintf("\n📝 Reason: %s", reason)
	}
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func banGhosts(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	progress, _ := msg.Reply(b, "👻 Scanning for deleted accounts...", nil)
	members, err := b.GetChatAdministrators(chat.Id, nil)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't fetch members.", nil)
		return err
	}

	count := 0
	for _, m := range members {
		u := m.GetUser()
		if u.IsDeleted {
			if _, banErr := b.BanChatMember(chat.Id, u.Id, nil); banErr == nil {
				count++
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	text := fmt.Sprintf("✅ Banned <b>%d</b> ghost accounts.", count)
	if progress != nil {
		_, _, _ = progress.EditText(b, text, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	return nil
}

// ─── WARN ─────────────────────────────────────────────────────────────────────

func warnUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.IsGroupChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	target, reason, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if target.Id == b.Id {
		_, err = msg.Reply(b, "😄 Can't warn me!", nil)
		return err
	}
	if config.IsSudo(target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't warn a sudo user.", nil)
		return err
	}
	if utils.IsInAdminList(b, chat.Id, target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't warn an admin.", nil)
		return err
	}

	cmd := utils.GetCommand(msg)
	if strings.HasPrefix(cmd, "d") && msg.ReplyToMessage != nil {
		_, _ = b.DeleteMessage(chat.Id, msg.ReplyToMessage.MessageId, nil)
	}

	var warn models.Warn
	database.DB.Where(models.Warn{ChatID: chat.Id, UserID: target.Id}).FirstOrCreate(&warn)
	warn.Count++
	database.DB.Save(&warn)

	if warn.Count >= maxWarns {
		_, _ = b.BanChatMember(chat.Id, target.Id, nil)
		database.DB.Delete(&warn)
		text := fmt.Sprintf("🚫 <b>%s banned!</b>\nReason: Reached max warnings (%d/%d).", utils.MentionHTML(target.Id, target.FirstName), maxWarns, maxWarns)
		_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	reasonText := reason
	if reasonText == "" {
		reasonText = "No reason provided."
	}
	text := fmt.Sprintf(
		"⚠️ <b>Warned!</b>\n👤 %s\n🆔 <code>%d</code>\n📊 Warns: <b>%d/%d</b>\n👮 By: %s\n📝 Reason: %s",
		utils.MentionHTML(target.Id, target.FirstName), target.Id, warn.Count, maxWarns, senderMention(ctx), reasonText,
	)
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: utils.SingleBtn("❌ Remove Warn", fmt.Sprintf("unwarn_%d", target.Id)),
	})
	return err
}

func removeWarnings(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}
	if msg.ReplyToMessage == nil || msg.ReplyToMessage.From == nil {
		_, err := msg.Reply(b, "❌ Reply to the user's message.", nil)
		return err
	}

	target := msg.ReplyToMessage.From
	result := database.DB.Where(models.Warn{ChatID: chat.Id, UserID: target.Id}).Delete(&models.Warn{})
	if result.RowsAffected == 0 {
		_, err := msg.Reply(b, fmt.Sprintf("ℹ️ %s has no warnings.", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}
	_, err := msg.Reply(b, fmt.Sprintf("✅ Cleared all warnings for %s.", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func checkWarns(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	target, err := utils.ExtractUser(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}

	var warn models.Warn
	database.DB.Where(models.Warn{ChatID: chat.Id, UserID: target.Id}).First(&warn)
	text := fmt.Sprintf("📊 <b>Warnings for</b> %s\nCount: <b>%d/%d</b>", utils.MentionHTML(target.Id, target.FirstName), warn.Count, maxWarns)
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ─── MUTE ─────────────────────────────────────────────────────────────────────

func muteUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.IsGroupChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	target, reason, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if target.Id == b.Id {
		_, err = msg.Reply(b, "😄 Can't mute myself!", nil)
		return err
	}
	if config.IsSudo(target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't mute a sudo user.", nil)
		return err
	}
	if utils.IsInAdminList(b, chat.Id, target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't mute an admin.", nil)
		return err
	}

	keyboard := utils.SingleBtn("🔊 Unmute", fmt.Sprintf("unmute_%d", target.Id))
	cmd := utils.GetCommand(msg)

	if cmd == "tmute" {
		parts := strings.SplitN(strings.TrimSpace(reason), " ", 2)
		timeVal := parts[0]
		tempReason := ""
		if len(parts) > 1 {
			tempReason = parts[1]
		}
		until, err := utils.TimeConverter(timeVal)
		if err != nil {
			_, err = msg.Reply(b, "❌ Invalid time. Use: 1h 30m 2d 1w", nil)
			return err
		}
		if _, muteErr := b.RestrictChatMember(chat.Id, target.Id, gotgbot.ChatPermissions{}, &gotgbot.RestrictChatMemberOpts{UntilDate: until.Unix()}); muteErr != nil {
			_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", muteErr.Error()), nil)
			return err
		}
		text := fmt.Sprintf("🔇 <b>Temporarily Muted!</b>\n👤 %s\n⏳ Duration: <b>%s</b>\n👮 By: %s", utils.MentionHTML(target.Id, target.FirstName), timeVal, senderMention(ctx))
		if tempReason != "" {
			text += fmt.Sprintf("\n📝 Reason: %s", tempReason)
		}
		_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
		return err
	}

	if _, muteErr := b.RestrictChatMember(chat.Id, target.Id, gotgbot.ChatPermissions{}, nil); muteErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", muteErr.Error()), nil)
		return err
	}
	text := fmt.Sprintf("🔇 <b>Muted!</b>\n👤 %s\n🆔 <code>%d</code>\n👮 By: %s", utils.MentionHTML(target.Id, target.FirstName), target.Id, senderMention(ctx))
	if reason != "" {
		text += fmt.Sprintf("\n📝 Reason: %s", reason)
	}
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func unmuteUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}
	target, err := utils.ExtractUser(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if _, unmuteErr := b.UnbanChatMember(chat.Id, target.Id, nil); unmuteErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", unmuteErr.Error()), nil)
		return err
	}
	_, err = msg.Reply(b, fmt.Sprintf("🔊 <b>Unmuted</b> %s!", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ─── DELETE / PURGE ───────────────────────────────────────────────────────────

func deleteMsg(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_delete_messages") {
		_, err := msg.Reply(b, "❌ You need can_delete_messages permission.", nil)
		return err
	}
	if msg.ReplyToMessage == nil {
		_, err := msg.Reply(b, "❌ Reply to a message to delete it.", nil)
		return err
	}
	_, _ = b.DeleteMessage(chat.Id, msg.ReplyToMessage.MessageId, nil)
	_, _ = b.DeleteMessage(chat.Id, msg.MessageId, nil)
	return nil
}

func purgeMessages(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_delete_messages") {
		_, err := msg.Reply(b, "❌ You need can_delete_messages permission.", nil)
		return err
	}
	if msg.ReplyToMessage == nil {
		_, err := msg.Reply(b, "❌ Reply to the message to start purging from.", nil)
		return err
	}
	_, _ = b.DeleteMessage(chat.Id, msg.MessageId, nil)

	startID := msg.ReplyToMessage.MessageId
	endID := msg.MessageId

	args := utils.GetCommandArgs(msg)
	if args != "" {
		var n int64
		fmt.Sscanf(args, "%d", &n)
		if n > 0 && startID+n < endID {
			endID = startID + n
		}
	}

	delTotal := 0
	for id := startID; id <= endID; id++ {
		if _, err := b.DeleteMessage(chat.Id, id, nil); err == nil {
			delTotal++
		}
	}

	notice, _ := b.SendMessage(chat.Id, fmt.Sprintf("🗑️ Purged <b>%d</b> messages.", delTotal), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	if notice != nil {
		cID, mID := chat.Id, notice.MessageId
		time.AfterFunc(5*time.Second, func() { _, _ = b.DeleteMessage(cID, mID, nil) })
	}
	return nil
}

// ─── PROMOTE / DEMOTE ─────────────────────────────────────────────────────────

func promoteUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.IsGroupChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_promote_members") {
		_, err := msg.Reply(b, "❌ You need can_promote_members permission.", nil)
		return err
	}

	target, err := utils.ExtractUser(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if target.Id == b.Id {
		_, err = msg.Reply(b, "❌ Can't promote myself!", nil)
		return err
	}

	botMember, err := b.GetChatMember(chat.Id, b.Id, nil)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't get my own permissions.", nil)
		return err
	}
	botAdmin, ok := botMember.(gotgbot.ChatMemberAdministrator)
	if !ok {
		_, err = msg.Reply(b, "❌ I'm not an admin.", nil)
		return err
	}

	cmd := utils.GetCommand(msg)
	opts := &gotgbot.PromoteChatMemberOpts{
		CanInviteUsers:      botAdmin.CanInviteUsers,
		CanDeleteMessages:   botAdmin.CanDeleteMessages,
		CanRestrictMembers:  botAdmin.CanRestrictMembers,
		CanPinMessages:      botAdmin.CanPinMessages,
		CanManageChat:       botAdmin.CanManageChat,
		CanManageVideoChats: botAdmin.CanManageVideoChats,
	}
	if cmd == "fullpromote" {
		opts.CanChangeInfo = botAdmin.CanChangeInfo
		opts.CanPromoteMembers = botAdmin.CanPromoteMembers
	}

	if _, promoteErr := b.PromoteChatMember(chat.Id, target.Id, opts); promoteErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", promoteErr.Error()), nil)
		return err
	}

	label := "⬆️ Promoted"
	if cmd == "fullpromote" {
		label = "⬆️ Full Promoted"
	}
	_, err = msg.Reply(b, fmt.Sprintf("%s %s!", label, utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func demoteUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_promote_members") {
		_, err := msg.Reply(b, "❌ You need can_promote_members permission.", nil)
		return err
	}
	target, err := utils.ExtractUser(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if target.Id == b.Id {
		_, err = msg.Reply(b, "❌ Can't demote myself!", nil)
		return err
	}
	if config.IsSudo(target.Id) {
		_, err = msg.Reply(b, "⚠️ Can't demote a sudo user.", nil)
		return err
	}

	if _, demoteErr := b.PromoteChatMember(chat.Id, target.Id, &gotgbot.PromoteChatMemberOpts{}); demoteErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", demoteErr.Error()), nil)
		return err
	}
	_, err = msg.Reply(b, fmt.Sprintf("⬇️ <b>Demoted</b> %s!", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ─── PIN ──────────────────────────────────────────────────────────────────────

func pinMessage(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_pin_messages") {
		_, err := msg.Reply(b, "❌ You need can_pin_messages permission.", nil)
		return err
	}
	if msg.ReplyToMessage == nil {
		_, err := msg.Reply(b, "❌ Reply to a message to pin/unpin.", nil)
		return err
	}

	cmd := utils.GetCommand(msg)
	r := msg.ReplyToMessage

	if cmd == "unpin" {
		if _, err := b.UnpinChatMessage(chat.Id, &gotgbot.UnpinChatMessageOpts{MessageId: r.MessageId}); err != nil {
			_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
			return err
		}
		_, err := msg.Reply(b, "📌 Unpinned that message.", nil)
		return err
	}

	if _, err := b.PinChatMessage(chat.Id, r.MessageId, &gotgbot.PinChatMessageOpts{DisableNotification: true}); err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
		return err
	}
	_, err := msg.Reply(b, "📌 <b>Pinned!</b>", &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ─── CHAT SETTINGS ────────────────────────────────────────────────────────────

func setChatTitle(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}
	newTitle := utils.GetCommandArgs(msg)
	if newTitle == "" {
		_, err := msg.Reply(b, "❌ Usage: /set_chat_title NEW TITLE", nil)
		return err
	}
	oldTitle := chat.Title
	if _, err := b.SetChatTitle(chat.Id, newTitle, nil); err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
		return err
	}
	_, err := msg.Reply(b, fmt.Sprintf("✅ Title changed from <b>%s</b> to <b>%s</b>", oldTitle, newTitle), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func setUserTitle(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}
	if msg.ReplyToMessage == nil || msg.ReplyToMessage.From == nil {
		_, err := msg.Reply(b, "❌ Reply to an admin's message.", nil)
		return err
	}
	title := utils.GetCommandArgs(msg)
	if title == "" {
		_, err := msg.Reply(b, "❌ Usage: /set_user_title ADMIN TITLE", nil)
		return err
	}
	target := msg.ReplyToMessage.From
	if _, err := b.SetChatAdministratorCustomTitle(chat.Id, target.Id, title, nil); err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
		return err
	}
	_, err := msg.Reply(b, fmt.Sprintf("✅ Changed %s's title to <b>%s</b>", utils.MentionHTML(target.Id, target.FirstName), title), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func setChatPhoto(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}
	if msg.ReplyToMessage == nil {
		_, err := msg.Reply(b, "❌ Reply to a photo or image document.", nil)
		return err
	}

	reply := msg.ReplyToMessage
	var fileID string
	if reply.Photo != nil && len(reply.Photo) > 0 {
		fileID = reply.Photo[len(reply.Photo)-1].FileId
	} else if reply.Document != nil {
		fileID = reply.Document.FileId
	} else {
		_, err := msg.Reply(b, "❌ Reply to a photo or image document.", nil)
		return err
	}

	// Download the file
	file, err := b.GetFile(fileID, nil)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't get file info.", nil)
		return err
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", config.BotToken, file.FilePath)
	resp, err := utils.FetchJSON(fileURL)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't download photo.", nil)
		return err
	}

	// Save temporarily
	tmpPath := fmt.Sprintf("/tmp/chatphoto_%d.jpg", chat.Id)
	if writeErr := os.WriteFile(tmpPath, resp, 0644); writeErr != nil {
		_, err = msg.Reply(b, "❌ Failed to save photo.", nil)
		return err
	}
	defer os.Remove(tmpPath)

	f, err := os.Open(tmpPath)
	if err != nil {
		_, err = msg.Reply(b, "❌ Failed to open photo.", nil)
		return err
	}
	defer f.Close()

	if _, setErr := b.SetChatPhoto(chat.Id, gotgbot.InputFileByReader("photo.jpg", f), nil); setErr != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", setErr.Error()), nil)
		return err
	}
	_, err = msg.Reply(b, "✅ Group photo updated!", nil)
	return err
}

// ─── MENTION ALL ──────────────────────────────────────────────────────────────

func mentionAll(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if !utils.IsAdmin(b, chat.Id, sender.Id()) {
		_, err := msg.Reply(b, "❌ Only admins can use this.", nil)
		return err
	}

	members, err := b.GetChatAdministrators(chat.Id, nil)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't fetch members.", nil)
		return err
	}

	var mentions []string
	for _, m := range members {
		u := m.GetUser()
		if u.Username != "" {
			mentions = append(mentions, "@"+u.Username)
		} else {
			mentions = append(mentions, utils.MentionHTML(u.Id, u.FirstName))
		}
	}

	for i := 0; i < len(mentions); i += 4 {
		end := i + 4
		if end > len(mentions) {
			end = len(mentions)
		}
		_, _ = b.SendMessage(chat.Id, strings.Join(mentions[i:end], " "), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}
	return nil
}

// ─── REPORT ───────────────────────────────────────────────────────────────────

func reportUser(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if !utils.IsGroupChat(chat) {
		return nil
	}

	var targetMsg *gotgbot.Message
	if msg.ReplyToMessage != nil {
		targetMsg = msg.ReplyToMessage
	} else {
		targetMsg = msg
	}

	reporter := ctx.EffectiveSender
	var targetID int64
	if targetMsg.From != nil {
		targetID = targetMsg.From.Id
	}

	if targetID == reporter.Id() {
		_, err := msg.Reply(b, "❌ Can't report yourself.", nil)
		return err
	}
	if utils.IsInAdminList(b, chat.Id, targetID) {
		_, err := msg.Reply(b, "ℹ️ That user is already an admin.", nil)
		return err
	}

	targetName := "Anonymous"
	if targetMsg.From != nil {
		targetName = utils.MentionHTML(targetMsg.From.Id, targetMsg.From.FirstName)
	}

	text := fmt.Sprintf("🚨 <b>Report!</b>\n👤 Reported: %s\n📝 By: %s",
		targetName, utils.MentionHTML(reporter.Id(), reporter.Name()))

	// Ping all admins invisibly
	admins, _ := b.GetChatAdministrators(chat.Id, nil)
	for _, admin := range admins {
		u := admin.GetUser()
		if u.IsBot || u.IsDeleted {
			continue
		}
		text += fmt.Sprintf(`<a href="tg://user?id=%d">‌</a>`, u.Id)
	}

	_, err := targetMsg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func handleAtAdmin(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.Text == "" {
		return ext.ContinueGroups
	}
	lower := strings.ToLower(msg.Text)
	if !strings.Contains(lower, "@admin") && !strings.Contains(lower, "@admins") {
		return ext.ContinueGroups
	}
	return reportUser(b, ctx)
}

// ─── ADMIN CACHE REFRESH ──────────────────────────────────────────────────────

func refreshAdminCache(_ *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.ChatMember != nil {
		utils.InvalidateAdminCache(ctx.ChatMember.Chat.Id)
	}
	return nil
}

// ─── CALLBACKS ────────────────────────────────────────────────────────────────

func unbanCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	chat := cq.Message.GetChat()
	from := cq.From

	if !utils.HasPermission(b, chat.Id, from.Id, "can_restrict_members") {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ No permission.", ShowAlert: true})
		return err
	}
	var targetID int64
	fmt.Sscanf(strings.TrimPrefix(cq.Data, "unban_"), "%d", &targetID)
	_, _ = b.UnbanChatMember(chat.Id, targetID, nil)
	newText := fmt.Sprintf("~~%s~~\n\n✅ Unbanned by %s", cq.Message.Text, utils.MentionHTML(from.Id, from.FirstName))
	_, _, _ = cq.Message.EditText(b, newText, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	_, err := cq.Answer(b, nil)
	return err
}

func unmuteCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	chat := cq.Message.GetChat()
	from := cq.From

	if !utils.HasPermission(b, chat.Id, from.Id, "can_restrict_members") {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ No permission.", ShowAlert: true})
		return err
	}
	var targetID int64
	fmt.Sscanf(strings.TrimPrefix(cq.Data, "unmute_"), "%d", &targetID)
	_, _ = b.UnbanChatMember(chat.Id, targetID, nil)
	newText := fmt.Sprintf("~~%s~~\n\n🔊 Unmuted by %s", cq.Message.Text, utils.MentionHTML(from.Id, from.FirstName))
	_, _, _ = cq.Message.EditText(b, newText, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	_, err := cq.Answer(b, nil)
	return err
}

func unwarnCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	chat := cq.Message.GetChat()
	from := cq.From

	if !utils.HasPermission(b, chat.Id, from.Id, "can_restrict_members") {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ No permission.", ShowAlert: true})
		return err
	}
	var targetID int64
	fmt.Sscanf(strings.TrimPrefix(cq.Data, "unwarn_"), "%d", &targetID)

	var warn models.Warn
	if database.DB.Where(models.Warn{ChatID: chat.Id, UserID: targetID}).First(&warn).Error == nil {
		if warn.Count > 0 {
			warn.Count--
			if warn.Count == 0 {
				database.DB.Delete(&warn)
			} else {
				database.DB.Save(&warn)
			}
		}
	}
	newText := fmt.Sprintf("~~%s~~\n\n✅ Warning removed by %s", cq.Message.Text, utils.MentionHTML(from.Id, from.FirstName))
	_, _, _ = cq.Message.EditText(b, newText, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	_, err := cq.Answer(b, nil)
	return err
}

// ─── HELPER ───────────────────────────────────────────────────────────────────

func senderMention(ctx *ext.Context) string {
	s := ctx.EffectiveSender
	if s == nil {
		return "Anon Admin"
	}
	return utils.MentionHTML(s.Id(), s.Name())
}
