// RoboKaty — modules/locks/locks.go
// Mirrors: misskaty/plugins/locks.py — including URL auto-detection (group=69)

package locks

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Locks"
const HELP = `
/lock [TYPE] - Lock a content type in this chat
/unlock [TYPE] - Unlock a content type
/locks - Show current lock status

Lock types:
  messages | sticker | gif | media | games | inline
  photo | video | docs | voice | audio | plain
  url | polls | group_info | useradd | pin | all

Example: /lock sticker | /lock all | /unlock url

Note: URL lock automatically deletes messages containing URLs.
`

// urlRegex detects URLs in messages
var urlRegex = regexp.MustCompile(`(?i)https?://\S+|www\.\S+|\b\w+\.\w{2,}\b`)

// lockTypeToPermission maps lock keywords to ChatPermissions fields
var lockTypeToPermission = map[string]string{
	"messages":   "can_send_messages",
	"sticker":    "can_send_stickers",
	"gif":        "can_send_other_messages",
	"media":      "can_send_documents",
	"games":      "can_send_other_messages",
	"inline":     "can_send_other_messages",
	"photo":      "can_send_photos",
	"video":      "can_send_videos",
	"docs":       "can_send_documents",
	"voice":      "can_send_voice_notes",
	"audio":      "can_send_audios",
	"plain":      "can_send_messages",
	"url":        "can_add_web_page_previews",
	"polls":      "can_send_polls",
	"group_info": "can_change_info",
	"useradd":    "can_invite_users",
	"pin":        "can_pin_messages",
}

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmds([]string{"lock", "unlock"}, lockCmd))
	dispatcher.AddHandler(utils.OnCmd("locks", showLocks))
	// URL auto-detector (group=69 same as original)
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Text, urlDetector), 69)
	log.Println("[Locks] ✅ Module loaded")
}

func lockCmd(b *gotgbot.Bot, ctx *ext.Context) error {
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

	param := strings.ToLower(strings.TrimSpace(utils.GetCommandArgs(msg)))
	if param == "" {
		_, err := msg.Reply(b, "❌ Usage: /lock [TYPE] or /unlock [TYPE]\n\nSee /locks for current status.", nil)
		return err
	}

	isLock := utils.GetCommand(msg) == "lock"

	if param == "all" {
		if isLock {
			if _, err := b.SetChatPermissions(chat.Id, gotgbot.ChatPermissions{}, nil); err != nil {
				_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
				return err
			}
			_, err := msg.Reply(b, fmt.Sprintf("🔒 Locked <b>everything</b> in %s.", chat.Title), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
			return err
		} else {
			if _, err := b.SetChatPermissions(chat.Id, gotgbot.ChatPermissions{
				CanSendMessages: true, CanSendAudios: true, CanSendDocuments: true,
				CanSendPhotos: true, CanSendVideos: true, CanSendVideoNotes: true,
				CanSendVoiceNotes: true, CanSendPolls: true, CanSendOtherMessages: true,
				CanAddWebPagePreviews: true, CanChangeInfo: true,
				CanInviteUsers: true, CanPinMessages: true,
			}, nil); err != nil {
				_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
				return err
			}
			_, err := msg.Reply(b, fmt.Sprintf("🔓 Unlocked <b>everything</b> in %s.", chat.Title), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
			return err
		}
	}

	if _, ok := lockTypeToPermission[param]; !ok {
		_, err := msg.Reply(b, "❌ Unknown lock type. See /locks for available types.", nil)
		return err
	}

	chatInfo, err := b.GetChat(chat.Id, nil)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't fetch chat info.", nil)
		return err
	}

	perms := chatInfo.Permissions
	if perms == nil {
		perms = &gotgbot.ChatPermissions{}
	}

	applyLock(perms, param, isLock)

	if _, err = b.SetChatPermissions(chat.Id, *perms, nil); err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
		return err
	}

	action := "🔒 Locked"
	if !isLock {
		action = "🔓 Unlocked"
	}
	_, err = msg.Reply(b, fmt.Sprintf("%s: <b>%s</b>", action, param), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func showLocks(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	chatInfo, err := b.GetChat(chat.Id, nil)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't fetch chat info.", nil)
		return err
	}
	perms := chatInfo.Permissions
	if perms == nil {
		perms = &gotgbot.ChatPermissions{}
	}

	icon := func(enabled bool) string {
		if enabled {
			return "🔓"
		}
		return "🔒"
	}

	text := fmt.Sprintf(
		"🔐 <b>Lock Status for %s</b>\n\n"+
			"%s Messages\n%s Sticker/GIF/Games/Inline\n%s Media/Docs\n"+
			"%s Photos\n%s Videos\n%s Audio\n%s Voice\n"+
			"%s Polls\n%s URL Previews\n%s Group Info\n"+
			"%s Add Users\n%s Pin Messages",
		chat.Title,
		icon(perms.CanSendMessages),
		icon(perms.CanSendOtherMessages),
		icon(perms.CanSendDocuments),
		icon(perms.CanSendPhotos),
		icon(perms.CanSendVideos),
		icon(perms.CanSendAudios),
		icon(perms.CanSendVoiceNotes),
		icon(perms.CanSendPolls),
		icon(perms.CanAddWebPagePreviews),
		icon(perms.CanChangeInfo),
		icon(perms.CanInviteUsers),
		icon(perms.CanPinMessages),
	)
	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// urlDetector — auto-deletes messages with URLs when URL lock is enabled
// Mirrors the group=69 handler in locks.py
func urlDetector(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if utils.IsPrivateChat(chat) || msg.Text == "" || msg.From == nil {
		return ext.ContinueGroups
	}

	// Exempt admins and sudo users
	if config.IsSudo(msg.From.Id) || utils.IsInAdminList(b, chat.Id, msg.From.Id) {
		return ext.ContinueGroups
	}

	// Check if URL lock is enabled (can_add_web_page_previews disabled)
	chatInfo, err := b.GetChat(chat.Id, nil)
	if err != nil {
		return ext.ContinueGroups
	}
	if chatInfo.Permissions == nil || chatInfo.Permissions.CanAddWebPagePreviews {
		return ext.ContinueGroups // URL lock not enabled
	}

	// Check if message contains a URL
	if !urlRegex.MatchString(msg.Text) {
		return ext.ContinueGroups
	}

	// Delete the message
	if _, delErr := b.DeleteMessage(chat.Id, msg.MessageId, nil); delErr != nil {
		_, _ = msg.Reply(b, "⚠️ This message contains a URL, but I don't have permission to delete it.", nil)
	}

	return ext.ContinueGroups
}

func applyLock(perms *gotgbot.ChatPermissions, lockType string, lock bool) {
	val := !lock
	switch lockType {
	case "messages", "plain":
		perms.CanSendMessages = val
	case "sticker", "gif", "games", "inline":
		perms.CanSendOtherMessages = val
	case "media", "docs":
		perms.CanSendDocuments = val
	case "photo":
		perms.CanSendPhotos = val
	case "video":
		perms.CanSendVideos = val
	case "voice":
		perms.CanSendVoiceNotes = val
	case "audio":
		perms.CanSendAudios = val
	case "url":
		perms.CanAddWebPagePreviews = val
	case "polls":
		perms.CanSendPolls = val
	case "group_info":
		perms.CanChangeInfo = val
	case "useradd":
		perms.CanInviteUsers = val
	case "pin":
		perms.CanPinMessages = val
	}
}
