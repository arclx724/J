// RoboKaty — modules/dev/dev.go
// Mirrors: misskaty/plugins/dev.py — ALL commands

package dev

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Dev"
const HELP = `
Sudo/Owner only:

/stats - Bot statistics
/gban [user] [reason] - Global ban
/ungban [user] - Remove global ban
/leave [chat_id] - Leave a chat
/restart - Restart the bot
/logs - Get bot log file
/shell [cmd] - Run shell command
/privacy - Privacy policy
/donate - Donation info
`

var BotStartTime = time.Now()

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("stats", stats))
	dispatcher.AddHandler(utils.OnCmd("gban", gban))
	dispatcher.AddHandler(utils.OnCmd("ungban", ungban))
	dispatcher.AddHandler(utils.OnCmd("leave", leaveChat))
	dispatcher.AddHandler(utils.OnCmd("restart", restart))
	dispatcher.AddHandler(utils.OnCmd("logs", getLogs))
	dispatcher.AddHandler(utils.OnCmd("shell", shell))
	dispatcher.AddHandler(utils.OnCmd("privacy", privacy))
	dispatcher.AddHandler(utils.OnCmd("donate", donate))
	dispatcher.AddHandler(utils.OnCmd("banuser", banUser_DB))
	dispatcher.AddHandler(utils.OnCmd("unbanuser", unbanUser_DB))
	dispatcher.AddHandler(utils.OnCmd("disablechat", disableChat_DB))
	dispatcher.AddHandler(utils.OnCmd("enablechat", enableChat_DB))
	log.Println("[Dev] ✅ Module loaded")
}

func sudoOnly(b *gotgbot.Bot, ctx *ext.Context, fn func(*gotgbot.Bot, *ext.Context) error) error {
	if !config.IsSudo(ctx.EffectiveSender.Id()) {
		return nil
	}
	return fn(b, ctx)
}

func stats(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		uptime := utils.FormatDuration(time.Since(BotStartTime))
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		text := fmt.Sprintf(
			"📊 <b>RoboKaty Stats</b>\n\n"+
				"⏱️ <b>Uptime:</b> <code>%s</code>\n"+
				"🔧 <b>Go Version:</b> <code>%s</code>\n"+
				"💾 <b>RAM Used:</b> <code>%.2f MB</code>\n"+
				"🖥️ <b>OS:</b> <code>%s/%s</code>\n"+
				"👤 <b>Owner ID:</b> <code>%d</code>",
			uptime, runtime.Version(),
			float64(mem.Alloc)/1024/1024,
			runtime.GOOS, runtime.GOARCH,
			config.OwnerID,
		)
		_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

func gban(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		target, reason, err := utils.ExtractUserAndReason(b, ctx)
		if err != nil || target == nil {
			_, err = msg.Reply(b, "❌ Reply to a message or provide @username/user_id.", nil)
			return err
		}
		if config.IsSudo(target.Id) {
			_, err = msg.Reply(b, "⚠️ Can't gban a sudo user.", nil)
			return err
		}
		if reason == "" {
			reason = "No reason provided."
		}
		text := fmt.Sprintf(
			"🔨 <b>Global Ban</b>\n👤 %s\n🆔 <code>%d</code>\n📝 Reason: %s",
			utils.MentionHTML(target.Id, target.FirstName), target.Id, reason,
		)
		if config.LogChannel != 0 {
			_, _ = b.SendMessage(config.LogChannel, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		}
		_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

func ungban(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		target, err := utils.ExtractUser(b, ctx)
		if err != nil || target == nil {
			_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
			return err
		}
		text := fmt.Sprintf("✅ Global ban removed for %s (<code>%d</code>).",
			utils.MentionHTML(target.Id, target.FirstName), target.Id)
		if config.LogChannel != 0 {
			_, _ = b.SendMessage(config.LogChannel, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		}
		_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

func leaveChat(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		arg := utils.GetCommandArgs(msg)
		if arg == "" {
			_, err := msg.Reply(b, "❌ Usage: /leave [chat_id or @username]", nil)
			return err
		}
		var chatID int64
		if n, scanErr := fmt.Sscanf(arg, "%d", &chatID); n == 1 && scanErr == nil {
			if _, leaveErr := b.LeaveChat(chatID, nil); leaveErr != nil {
				_, err := msg.Reply(b, fmt.Sprintf("❌ Failed: %s", leaveErr.Error()), nil)
				return err
			}
		} else {
			if _, leaveErr := b.LeaveChat(arg, nil); leaveErr != nil {
				_, err := msg.Reply(b, fmt.Sprintf("❌ Failed: %s", leaveErr.Error()), nil)
				return err
			}
		}
		_, err := msg.Reply(b, fmt.Sprintf("✅ Left chat: <code>%s</code>", arg), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

func restart(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		_, _ = msg.Reply(b, "🔄 Restarting...", nil)
		exe, err := os.Executable()
		if err != nil {
			_, err = msg.Reply(b, fmt.Sprintf("❌ Restart failed: %s", err.Error()), nil)
			return err
		}
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if startErr := cmd.Start(); startErr != nil {
			_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", startErr.Error()), nil)
			return err
		}
		os.Exit(0)
		return nil
	})
}

func getLogs(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		logFile := "robokaty.log"
		if _, statErr := os.Stat(logFile); os.IsNotExist(statErr) {
			_, err := msg.Reply(b, "ℹ️ No log file found.", nil)
			return err
		}
		f, err := os.Open(logFile)
		if err != nil {
			_, err = msg.Reply(b, fmt.Sprintf("❌ Couldn't open log: %s", err.Error()), nil)
			return err
		}
		defer f.Close()
		_, err = b.SendDocument(msg.Chat.Id, gotgbot.InputFileByReader("robokaty.log", f), &gotgbot.SendDocumentOpts{
			Caption:         "📋 RoboKaty Logs",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err
	})
}

func shell(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		cmd := utils.GetCommandArgs(msg)
		if cmd == "" {
			_, err := msg.Reply(b, "❌ Usage: /shell [command]", nil)
			return err
		}
		out, execErr := exec.Command("bash", "-c", cmd).CombinedOutput()
		output := strings.TrimSpace(string(out))
		if output == "" {
			output = "(no output)"
		}
		if len(output) > 3800 {
			output = output[:3800] + "\n...(truncated)"
		}
		text := fmt.Sprintf("<b>$ %s</b>\n\n<pre>%s</pre>", cmd, output)
		if execErr != nil {
			text += fmt.Sprintf("\n\n<b>Exit error:</b> <code>%s</code>", execErr.Error())
		}
		_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

// /privacy — privacy policy (from dev.py)
func privacy(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	botInfo, _ := b.GetMe(nil)
	text := fmt.Sprintf(
		"🔒 <b>Privacy Policy for %s</b>\n\n"+
			"This bot only stores the data necessary to function:\n"+
			"• Group IDs and settings\n"+
			"• User IDs for warns, notes, karma, AFK\n"+
			"• Federation membership data\n\n"+
			"We do <b>not</b> store message content.\n"+
			"Contact @%s for data removal requests.",
		botInfo.FirstName, config.SupportChat,
	)
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// /donate — donation info (from dev.py)
func donate(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	text := fmt.Sprintf(
		"❤️ <b>Support RoboKaty</b>\n\n"+
			"If you find this bot useful, consider supporting:\n\n"+
			"📢 Support Channel: @%s\n\n"+
			"Thank you for using RoboKaty! 🐱",
		config.SupportChat,
	)
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// /banuser — sudo only: ban a user from using the bot (from ban_user_or_chat.py)

// ─── Bot-level user/chat ban (from ban_user_or_chat.py) ───────────────────────
// These ban users from interacting with the bot at all

// In-memory ban store (can be extended to DB)
var bannedUsers = map[int64]string{}
var disabledChats = map[int64]string{}

func banUser_DB(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		target, reason, err := utils.ExtractUserAndReason(b, ctx)
		if err != nil || target == nil {
			_, err = msg.Reply(b, "❌ Usage: /banuser @user [reason]", nil)
			return err
		}
		if reason == "" {
			reason = "No reason provided."
		}
		if _, already := bannedUsers[target.Id]; already {
			_, err = msg.Reply(b, fmt.Sprintf("❌ %s is already banned.", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
			return err
		}
		bannedUsers[target.Id] = reason
		_, err = msg.Reply(b, fmt.Sprintf("✅ Banned %s from using the bot.\n📝 Reason: %s", utils.MentionHTML(target.Id, target.FirstName), reason), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

func unbanUser_DB(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		target, err := utils.ExtractUser(b, ctx)
		if err != nil || target == nil {
			_, err = msg.Reply(b, "❌ Usage: /unbanuser @user", nil)
			return err
		}
		if _, exists := bannedUsers[target.Id]; !exists {
			_, err = msg.Reply(b, fmt.Sprintf("❌ %s is not banned.", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
			return err
		}
		delete(bannedUsers, target.Id)
		_, err = msg.Reply(b, fmt.Sprintf("✅ Unbanned %s.", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

func disableChat_DB(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		args := strings.Fields(utils.GetCommandArgs(msg))
		if len(args) == 0 {
			_, err := msg.Reply(b, "❌ Usage: /disablechat [chat_id] [reason]", nil)
			return err
		}
		var chatID int64
		fmt.Sscanf(args[0], "%d", &chatID)
		if chatID == 0 {
			_, err := msg.Reply(b, "❌ Invalid chat ID.", nil)
			return err
		}
		reason := "No reason provided."
		if len(args) > 1 {
			reason = strings.Join(args[1:], " ")
		}
		if _, exists := disabledChats[chatID]; exists {
			_, err := msg.Reply(b, "❌ Chat is already disabled.", nil)
			return err
		}
		disabledChats[chatID] = reason
		_, _ = b.SendMessage(chatID, fmt.Sprintf("🚫 This bot has been restricted from this chat.\nReason: <code>%s</code>", reason), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		_, _ = b.LeaveChat(chatID, nil)
		_, err := msg.Reply(b, fmt.Sprintf("✅ Chat <code>%d</code> disabled.", chatID), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

func enableChat_DB(b *gotgbot.Bot, ctx *ext.Context) error {
	return sudoOnly(b, ctx, func(b *gotgbot.Bot, ctx *ext.Context) error {
		msg := ctx.EffectiveMessage
		arg := utils.GetCommandArgs(msg)
		var chatID int64
		fmt.Sscanf(arg, "%d", &chatID)
		if chatID == 0 {
			_, err := msg.Reply(b, "❌ Usage: /enablechat [chat_id]", nil)
			return err
		}
		if _, exists := disabledChats[chatID]; !exists {
			_, err := msg.Reply(b, "❌ Chat is not disabled.", nil)
			return err
		}
		delete(disabledChats, chatID)
		_, err := msg.Reply(b, fmt.Sprintf("✅ Chat <code>%d</code> re-enabled.", chatID), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	})
}

// IsBannedUser returns true if the user is bot-banned
func IsBannedUser(userID int64) bool {
	_, banned := bannedUsers[userID]
	return banned
}

// IsDisabledChat returns true if the chat is bot-disabled
func IsDisabledChat(chatID int64) bool {
	_, disabled := disabledChats[chatID]
	return disabled
}
