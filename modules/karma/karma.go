// RoboKaty - Rose-style Telegram Group Manager Bot
// modules/karma/karma.go — FIXED (no broken goroutines)

package karma

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Karma"
const HELP = `
/karma - View karma leaderboard for this chat
/karma_toggle [enable/disable] - Enable or disable karma system

Upvote: Reply with +, ++, +1, thanks, ty, 👍
Downvote: Reply with -, --, -1, 👎
`

var (
	upvoteRe   = regexp.MustCompile(`(?i)^(\++|\+1|thx|tnx|ty|tq|thank you|thanx|thanks|pro|cool|good|agree|makasih|👍|\+\+ .+)$`)
	downvoteRe = regexp.MustCompile(`(?i)^(-+|-1|not cool|disagree|worst|bad|👎|-- .+)$`)
)

var karmaEnabled = map[int64]bool{}

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("karma", karmaBoard))
	dispatcher.AddHandler(utils.OnCmd("karma_toggle", karmaToggle))
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Text, handleUpvote), 3)
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Text, handleDownvote), 4)
	log.Println("[Karma] ✅ Module loaded")
}

func karmaBoard(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	var karmaList []models.Karma
	database.DB.Where("chat_id = ?", chat.Id).Order("karma desc").Limit(10).Find(&karmaList)
	if len(karmaList) == 0 {
		_, err := msg.Reply(b, "📊 No karma data yet in this chat.", nil)
		return err
	}
	sort.Slice(karmaList, func(i, j int) bool { return karmaList[i].Karma > karmaList[j].Karma })
	text := fmt.Sprintf("🏆 <b>Karma Leaderboard — %s</b>\n\n", chat.Title)
	medals := []string{"🥇", "🥈", "🥉"}
	for i, k := range karmaList {
		medal := "•"
		if i < len(medals) {
			medal = medals[i]
		}
		name := fmt.Sprintf("User %d", k.UserID)
		if member, err := b.GetChatMember(chat.Id, k.UserID, nil); err == nil {
			u := member.GetUser()
			name = u.FirstName
			if u.LastName != "" {
				name += " " + u.LastName
			}
		}
		sign := "+"
		if k.Karma < 0 {
			sign = ""
		}
		text += fmt.Sprintf("%s %s — <b>%s%d</b>\n", medal, utils.MentionHTML(k.UserID, name), sign, k.Karma)
	}
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func karmaToggle(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_restrict_members") {
		_, err := msg.Reply(b, "❌ You need can_restrict_members permission.", nil)
		return err
	}
	switch strings.ToLower(utils.GetCommandArgs(msg)) {
	case "enable", "on", "yes":
		karmaEnabled[chat.Id] = true
	case "disable", "off", "no":
		karmaEnabled[chat.Id] = false
	default:
		_, err := msg.Reply(b, "❌ Usage: /karma_toggle [enable/disable]", nil)
		return err
	}
	status := "disabled ❌"
	if karmaEnabled[chat.Id] {
		status = "enabled ✅"
	}
	_, err := msg.Reply(b, fmt.Sprintf("Karma system is now <b>%s</b>.", status), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func handleUpvote(b *gotgbot.Bot, ctx *ext.Context) error  { return handleVote(b, ctx, true) }
func handleDownvote(b *gotgbot.Bot, ctx *ext.Context) error { return handleVote(b, ctx, false) }

func handleVote(b *gotgbot.Bot, ctx *ext.Context, upvote bool) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	if utils.IsPrivateChat(chat) || msg.ReplyToMessage == nil || msg.ReplyToMessage.From == nil || msg.From == nil {
		return ext.ContinueGroups
	}
	if enabled, ok := karmaEnabled[chat.Id]; ok && !enabled {
		return ext.ContinueGroups
	}
	text := strings.TrimSpace(msg.Text)
	if upvote && !upvoteRe.MatchString(text) {
		return ext.ContinueGroups
	}
	if !upvote && !downvoteRe.MatchString(text) {
		return ext.ContinueGroups
	}
	targetID := msg.ReplyToMessage.From.Id
	if targetID == msg.From.Id || msg.ReplyToMessage.From.IsBot {
		return ext.ContinueGroups
	}
	var karma models.Karma
	database.DB.Where(models.Karma{ChatID: chat.Id, UserID: targetID}).FirstOrCreate(&karma)
	if upvote {
		karma.Karma++
	} else {
		karma.Karma--
	}
	database.DB.Save(&karma)
	sign := "+"
	if karma.Karma < 0 {
		sign = ""
	}
	emoji := "⬆️"
	if !upvote {
		emoji = "⬇️"
	}
	notice, _ := msg.Reply(b,
		fmt.Sprintf("%s %s now has <b>%s%d</b> karma!", emoji, utils.MentionHTML(targetID, msg.ReplyToMessage.From.FirstName), sign, karma.Karma),
		&gotgbot.SendMessageOpts{ParseMode: "HTML"})
	if notice != nil {
		cID, mID := chat.Id, notice.MessageId
		time.AfterFunc(5*time.Second, func() { _, _ = b.DeleteMessage(cID, mID, nil) })
	}
	return ext.ContinueGroups
}
