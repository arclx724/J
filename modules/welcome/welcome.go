// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Mirrors: misskaty/plugins/grup_tools.py (welcome section)

package welcome

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Welcome"
const HELP = `
/toggle_welcome - Enable or disable welcome messages
/setwelcome [TEXT] - Set custom welcome message
/resetwelcome - Reset welcome to default
/setwelcome - with replied media to set a welcome with media

Supported fillings in welcome text:
{mention} {first} {last} {fullname} {username} {id} {chatname}
`

// Last welcome message per chat — for clean welcome (delete old on new join)
var (
	lastWelcomeMsgs   = make(map[int64]int64)
	lastWelcomeMsgsMu sync.Mutex
)

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("toggle_welcome", toggleWelcome))
	dispatcher.AddHandler(utils.OnCmd("setwelcome", setWelcome))
	dispatcher.AddHandler(utils.OnCmd("resetwelcome", resetWelcome))
	dispatcher.AddHandler(handlers.NewChatMemberUpdated(nil, handleMemberJoin))
	log.Println("[Welcome] ✅ Module loaded")
}

func toggleWelcome(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if !utils.IsGroupChat(chat) {
		return nil
	}
	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}

	var w models.Welcome
	database.DB.Where(models.Welcome{ChatID: chat.Id}).FirstOrCreate(&w)
	w.WelcomeEnabled = !w.WelcomeEnabled
	database.DB.Save(&w)

	status := "disabled ❌"
	if w.WelcomeEnabled {
		status = "enabled ✅"
	}
	_, err := msg.Reply(b, fmt.Sprintf("Welcome messages are now <b>%s</b>.", status), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func setWelcome(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if !utils.IsGroupChat(chat) {
		return nil
	}
	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}

	var w models.Welcome
	database.DB.Where(models.Welcome{ChatID: chat.Id}).FirstOrCreate(&w)

	if msg.ReplyToMessage != nil {
		// Set welcome with media
		reply := msg.ReplyToMessage
		if reply.Photo != nil && len(reply.Photo) > 0 {
			w.WelcomeFileID = reply.Photo[len(reply.Photo)-1].FileId
		} else if reply.Video != nil {
			w.WelcomeFileID = reply.Video.FileId
		} else if reply.Animation != nil {
			w.WelcomeFileID = reply.Animation.FileId
		} else if reply.Document != nil {
			w.WelcomeFileID = reply.Document.FileId
		}
		if reply.Caption != "" {
			w.WelcomeText = reply.Caption
		} else if reply.Text != "" {
			w.WelcomeText = reply.Text
		}
	} else {
		text := utils.GetCommandArgs(msg)
		if text == "" {
			_, err := msg.Reply(b, "❌ Usage: /setwelcome [TEXT]\n\nFillings: {mention} {first} {id} {chatname}", nil)
			return err
		}
		w.WelcomeText = text
	}

	w.WelcomeEnabled = true
	database.DB.Save(&w)
	_, err := msg.Reply(b, "✅ Welcome message saved!", nil)
	return err
}

func resetWelcome(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}
	database.DB.Model(&models.Welcome{}).Where("chat_id = ?", chat.Id).Updates(map[string]interface{}{
		"welcome_text":   "",
		"welcome_file_id": "",
	})
	_, err := msg.Reply(b, "✅ Welcome reset to default.", nil)
	return err
}

func handleMemberJoin(b *gotgbot.Bot, ctx *ext.Context) error {
	update := ctx.ChatMember
	if update == nil {
		return nil
	}

	// Only trigger on new joins
	if update.OldChatMember != nil {
		switch update.OldChatMember.(type) {
		case gotgbot.ChatMemberLeft, gotgbot.ChatMemberBanned:
			// These are valid "joining" states
		default:
			return nil
		}
	}

	newMember, ok := update.NewChatMember.(gotgbot.ChatMemberMember)
	if !ok {
		// Also handle admins being added
		_, isAdmin := update.NewChatMember.(gotgbot.ChatMemberAdministrator)
		if !isAdmin {
			return nil
		}
		_ = isAdmin
	}
	_ = newMember

	user := update.NewChatMember.GetUser()
	chat := update.Chat

	if user.IsBot {
		return nil
	}

	// Check welcome settings
	var w models.Welcome
	if database.DB.Where(models.Welcome{ChatID: chat.Id}).First(&w).Error != nil {
		return nil // welcome not configured
	}
	if !w.WelcomeEnabled {
		return nil
	}

	// Special message for owner
	if user.Id == config.OwnerID {
		_, err := b.SendMessage(chat.Id, "👑 Welcome home, boss!", nil)
		return err
	}

	// Delete previous welcome message (clean welcome)
	lastWelcomeMsgsMu.Lock()
	if prevID, ok := lastWelcomeMsgs[chat.Id]; ok {
		_, _ = b.DeleteMessage(chat.Id, prevID, nil)
	}
	lastWelcomeMsgsMu.Unlock()

	// Build welcome text
	mention := utils.MentionHTML(user.Id, user.FirstName)
	text := w.WelcomeText
	if text == "" {
		text = fmt.Sprintf(
			"👋 Welcome to <b>%s</b>, %s!\n🆔 ID: <code>%d</code>",
			chat.Title, mention, user.Id,
		)
	} else {
		text = applyFillings(text, user, chat)
	}

	// Send welcome (with media if set)
	var sent *gotgbot.Message
	var sendErr error

	if w.WelcomeFileID != "" {
		sent, sendErr = b.SendPhoto(chat.Id, gotgbot.InputFileByID(w.WelcomeFileID), &gotgbot.SendPhotoOpts{
			Caption:   text,
			ParseMode: "HTML",
		})
	} else {
		sent, sendErr = b.SendMessage(chat.Id, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}

	if sendErr == nil && sent != nil {
		lastWelcomeMsgsMu.Lock()
		lastWelcomeMsgs[chat.Id] = sent.MessageId
		lastWelcomeMsgsMu.Unlock()
	}

	// CAS (Combot Anti-Spam) check in background
	go checkCAS(b, chat.Id, user.Id, mention)

	return nil
}

// checkCAS checks https://api.cas.chat and bans if flagged
// Mirrors the Combot API check in grup_tools.py
type casResponse struct {
	Ok     bool   `json:"ok"`
	Result *struct {
		Offenses int    `json:"offenses"`
		Messages []string `json:"messages"`
	} `json:"result"`
}

func checkCAS(b *gotgbot.Bot, chatID, userID int64, mention string) {
	url := fmt.Sprintf("https://api.cas.chat/check?user_id=%d", userID)
	body, err := utils.FetchJSON(url)
	if err != nil {
		return
	}

	var result casResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return
	}

	if result.Ok {
		// User is CAS banned — ban them
		_, _ = b.BanChatMember(chatID, userID, &gotgbot.BanChatMemberOpts{
			UntilDate: time.Now().Add(30 * time.Second).Unix(),
		})
		_, _ = b.SendMessage(chatID,
			fmt.Sprintf("🚨 <b>CAS Banned!</b>\n%s was automatically banned — flagged by Combot Anti-Spam.", mention),
			&gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}
}

func applyFillings(text string, user gotgbot.User, chat gotgbot.Chat) string {
	first := user.FirstName
	last := user.LastName
	fullname := first
	if last != "" {
		fullname = first + " " + last
	}
	mention := utils.MentionHTML(user.Id, first)
	username := user.Username
	if username == "" {
		username = mention
	} else {
		username = "@" + username
	}

	r := strings.NewReplacer(
		"{first}", first,
		"{last}", last,
		"{fullname}", fullname,
		"{username}", username,
		"{mention}", mention,
		"{umention}", mention,
		"{id}", fmt.Sprintf("%d", user.Id),
		"{uid}", fmt.Sprintf("%d", user.Id),
		"{chatname}", chat.Title,
		"{ttl}", chat.Title,
	)
	return r.Replace(text)
}
