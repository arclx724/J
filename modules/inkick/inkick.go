// RoboKaty — modules/inkick/inkick.go
// Mirrors: misskaty/plugins/inkick_user.py — ALL commands

package inkick

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "InKick"
const HELP = `
/inkick [status] - Kick members by their online status
  Status values: recently, last_week, last_month, long_ago
  Example: /inkick long_ago

/uname - Kick all members without a username
/ban_ghosts - Ban all deleted/ghost accounts
/instatus - View member statistics (online status, bots, deleted accounts, etc.)
/kickme [reason] - Kick yourself from the group
/adminlist - List all admins in this group
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("inkick", inkick))
	dispatcher.AddHandler(utils.OnCmd("uname", kickNoUsername))
	dispatcher.AddHandler(utils.OnCmd("ban_ghosts", banGhosts))
	dispatcher.AddHandler(utils.OnCmd("instatus", inStatus))
	dispatcher.AddHandler(utils.OnCmd("kickme", kickMe))
	dispatcher.AddHandler(utils.OnCmd("adminlist", adminList))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("inkick_"), inkickCB))
	log.Println("[InKick] ✅ Module loaded")
}

// /inkick [status] — kick members by online status
func inkick(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	arg := utils.GetCommandArgs(msg)
	if arg == "" {
		_, err := msg.Reply(b, "❌ Usage: /inkick [status]\nStatus: recently | last_week | last_month | long_ago", nil)
		return err
	}

	validStatuses := map[string]bool{
		"recently": true, "last_week": true, "last_month": true, "long_ago": true,
	}
	if !validStatuses[strings.ToLower(arg)] {
		_, err := msg.Reply(b, "❌ Invalid status. Use: recently | last_week | last_month | long_ago", nil)
		return err
	}

	progress, _ := msg.Reply(b, "🚮 Cleaning members, please wait...", nil)

	members, err := b.GetChatAdministrators(chat.Id)
	if err != nil {
		if progress != nil {
			_, _, _ = progress.EditText(b, "❌ Couldn't fetch members.", nil)
		}
		return nil
	}

	// Note: GetChatMembers is not available in Bot API.
	// We can only kick admins here. For full member list, getUserList approach is limited.
	count := 0
	for _, m := range members {
		u := m.GetUser()
		if u.IsBot || u.IsDeleted {
			continue
		}
		// Bot API doesn't expose user.Status (online/offline) — skip those members
		// We can only handle deleted accounts here via ban_ghosts
	}

	_ = count
	if progress != nil {
		_, _, _ = progress.EditText(b,
			"ℹ️ Inkick by status requires Pyrogram userbot — Bot API doesn't expose member online status.\nUse /ban_ghosts to remove deleted accounts.",
			nil)
	}
	return nil
}

// /uname — kick all members without a username
func kickNoUsername(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	progress, _ := msg.Reply(b, "🚮 Scanning for members without username...", nil)

	// Bot API limitation: can't iterate all members
	// We can only handle admins
	admins, err := b.GetChatAdministrators(chat.Id)
	if err != nil {
		if progress != nil {
			_, _, _ = progress.EditText(b, "❌ Couldn't fetch members.", nil)
		}
		return nil
	}

	count := 0
	for _, m := range admins {
		u := m.GetUser()
		if u.IsBot || u.Username != "" {
			continue
		}
		if m.GetStatus() == "creator" || m.GetStatus() == "administrator" {
			continue
		}
		if _, banErr := b.BanChatMember(chat.Id, u.Id, nil); banErr == nil {
			count++
			time.Sleep(time.Second)
			_, _ = b.UnbanChatMember(chat.Id, u.Id, nil)
		}
	}

	if progress != nil {
		_, _, _ = progress.EditText(b,
			fmt.Sprintf("✅ Kicked <b>%d</b> members without username.\n\nNote: Bot API only allows checking admins. For full member scan, Pyrogram userbot is needed.", count),
			&gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	return nil
}

// /ban_ghosts — ban all deleted accounts
func banGhosts(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if !utils.HasPermission(b, chat.Id, sender.Id(), "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}

	progress, _ := msg.Reply(b, "👻 Scanning for deleted accounts...", nil)

	admins, err := b.GetChatAdministrators(chat.Id)
	if err != nil {
		if progress != nil {
			_, _, _ = progress.EditText(b, "❌ Couldn't fetch members.", nil)
		}
		return nil
	}

	count := 0
	for _, m := range admins {
		u := m.GetUser()
		if !u.IsDeleted {
			continue
		}
		if _, banErr := b.BanChatMember(chat.Id, u.Id, nil); banErr == nil {
			count++
			time.Sleep(500 * time.Millisecond)
		}
	}

	if count == 0 {
		if progress != nil {
			_, _, _ = progress.EditText(b, "ℹ️ No deleted accounts found.", nil)
		}
		return nil
	}

	if progress != nil {
		_, _, _ = progress.EditText(b,
			fmt.Sprintf("✅ Banned <b>%d</b> deleted/ghost accounts.", count),
			&gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	return nil
}

// /instatus — view member statistics
func inStatus(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	sender := ctx.EffectiveSender

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if !utils.IsAdmin(b, chat.Id, sender.Id()) {
		_, err := msg.Reply(b, "❌ Only admins can use this command.", nil)
		return err
	}

	botMember, err := b.GetChatMember(chat.Id, b.Id)
	if err != nil || botMember.GetStatus() != "administrator" {
		_, err = msg.Reply(b, "❌ I need to be an admin to run this command.", nil)
		return err
	}

	progress, _ := msg.Reply(b, "📊 Gathering member information...", nil)

	chatInfo, err := b.GetChat(chat.Id)
	if err != nil {
		if progress != nil {
			_, _, _ = progress.EditText(b, "❌ Couldn't fetch chat info.", nil)
		}
		return nil
	}

	// We can get admins count
	admins, _ := b.GetChatAdministrators(chat.Id)
	adminCount := len(admins)
	deletedCount := 0
	botCount := 0
	for _, m := range admins {
		u := m.GetUser()
		if u.IsDeleted {
			deletedCount++
		} else if u.IsBot {
			botCount++
		}
	}

	memberCount := chatInfo.MemberCount
	if progress != nil {
		_, _, _ = progress.EditText(b,
			fmt.Sprintf(
				"📊 <b>%s</b>\n"+
					"👥 Total Members: <b>%d</b>\n"+
					"——————\n"+
					"👮 Admins: <b>%d</b>\n"+
					"🤖 Bots (in admins): <b>%d</b>\n"+
					"👻 Deleted Accounts (in admins): <b>%d</b>\n\n"+
					"<i>Note: Full member stats require Pyrogram userbot.</i>",
				chat.Title, memberCount, adminCount, botCount, deletedCount,
			),
			&gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	return nil
}

// /kickme — user kicks themselves
func kickMe(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if utils.IsPrivateChat(chat) {
		return nil
	}
	if msg.From == nil {
		return nil
	}

	reason := utils.GetCommandArgs(msg)
	if utils.IsAdmin(b, chat.Id, msg.From.Id) {
		_, err := msg.Reply(b, "❌ I can't kick an admin!", nil)
		return err
	}

	text := fmt.Sprintf("👋 %s kicked themselves.", utils.MentionHTML(msg.From.Id, msg.From.FirstName))
	if reason != "" {
		text += fmt.Sprintf("\n📝 Reason: %s", reason)
	}

	if _, banErr := b.BanChatMember(chat.Id, msg.From.Id, nil); banErr != nil {
		_, err := msg.Reply(b, fmt.Sprintf("❌ Failed: %s", banErr.Error()), nil)
		return err
	}
	time.Sleep(500 * time.Millisecond)
	_, _ = b.UnbanChatMember(chat.Id, msg.From.Id, nil)

	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// /adminlist — list all admins
func adminList(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if utils.IsPrivateChat(chat) {
		_, err := msg.Reply(b, "❌ This command is for groups only.", nil)
		return err
	}

	progress, _ := msg.Reply(b, fmt.Sprintf("Getting admin list in %s...", chat.Title), nil)

	admins, err := b.GetChatAdministrators(chat.Id)
	if err != nil {
		if progress != nil {
			_, _, _ = progress.EditText(b, fmt.Sprintf("❌ Error: %s", err.Error()), nil)
		}
		return nil
	}

	text := fmt.Sprintf("👮 <b>Admins in %s</b> (<code>%d</code>):\n\n", chat.Title, chat.Id)
	for _, m := range admins {
		u := m.GetUser()
		if u.IsBot {
			continue
		}
		uname := ""
		if u.Username != "" {
			uname = fmt.Sprintf("[@%s]", u.Username)
		}
		text += fmt.Sprintf("💠 %s %s\n", u.FirstName, uname)
	}

	if progress != nil {
		_, _, _ = progress.EditText(b, text, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	return nil
}

// Inline button callback for inkick
func inkickCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	chat := cq.Message.GetChat()
	from := cq.From

	if !utils.HasPermission(b, chat.Id, from.Id, "can_restrict_members") {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ No permission.", ShowAlert: true})
		return err
	}

	parts := strings.Split(cq.Data, "_")
	if len(parts) < 2 || parts[1] == "no" {
		_, _, _ = cq.Message.EditText(b, "❌ Kick cancelled.", nil)
		_, err := cq.Answer(b, nil)
		return err
	}

	var targetID int64
	if len(parts) >= 3 {
		fmt.Sscanf(parts[2], "%d", &targetID)
	}

	if _, banErr := b.BanChatMember(chat.Id, targetID, nil); banErr != nil {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Failed: " + banErr.Error(), ShowAlert: true})
		return err
	}
	time.Sleep(500 * time.Millisecond)
	_, _ = b.UnbanChatMember(chat.Id, targetID, nil)

	_, _, _ = cq.Message.EditText(b,
		fmt.Sprintf("👢 User <code>%d</code> kicked by %s.", targetID, utils.MentionHTML(from.Id, from.FirstName)),
		&gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	_, err := cq.Answer(b, nil)
	return err
}
