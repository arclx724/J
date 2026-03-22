// RoboKaty - Rose-style Telegram Group Manager Bot
// modules/nightmode/nightmode.go — Mirrors misskaty/plugins/nightmodev2.py

package nightmode

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/go-co-op/gocron"

	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Nightmode"
const HELP = `
/nightmode -s=HH:MM -e=Xh - Enable night mode
  -s : Start time (24h format, e.g. 22:00)
  -e : Duration  (e.g. 6h or 120m)
  -d : Disable night mode

Examples:
  /nightmode -s=23:00 -e=6h
  /nightmode -s=22:30 -e=90m
  /nightmode -d

During night mode, the chat is locked (no messages).
It unlocks automatically after the duration.
`

var scheduler *gocron.Scheduler

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("nightmode", nightmodeCmd))

	// Start the scheduler
	scheduler = gocron.NewScheduler(time.UTC)
	scheduler.StartAsync()

	// Re-schedule all active nightmodes on startup
	go restoreNightmodes()

	log.Println("[Nightmode] ✅ Module loaded")
}

func nightmodeCmd(b *gotgbot.Bot, ctx *ext.Context) error {
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

	args := utils.GetCommandArgs(msg)

	// Disable flag
	if strings.Contains(args, "-d") {
		database.DB.Where("chat_id = ?", chat.Id).Delete(&models.NightMode{})
		scheduler.RemoveByTag(fmt.Sprintf("night_%d", chat.Id))
		_, err := msg.Reply(b, "✅ Night mode <b>disabled</b>.", &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	// Parse -s and -e flags
	startTime := extractFlag(args, "s")
	endFlag := extractFlag(args, "e")

	if startTime == "" || endFlag == "" {
		_, err := msg.Reply(b, "❌ Usage:\n/nightmode -s=22:00 -e=6h\n\nOr /nightmode -d to disable.", nil)
		return err
	}

	// Validate time format HH:MM
	timeRe := regexp.MustCompile(`^\d{2}:\d{2}$`)
	if !timeRe.MatchString(startTime) {
		_, err := msg.Reply(b, "❌ Invalid time format. Use HH:MM (e.g. 22:00)", nil)
		return err
	}

	// Parse duration
	durationSeconds, err := parseDuration(endFlag)
	if err != nil {
		_, err = msg.Reply(b, "❌ Invalid duration. Use Xh or Xm (e.g. 6h or 90m)", nil)
		return err
	}

	endTime := calculateEndTime(startTime, durationSeconds)

	// Save to DB
	var nm models.NightMode
	database.DB.Where(models.NightMode{ChatID: chat.Id}).FirstOrCreate(&nm)
	nm.Enabled = true
	nm.StartTime = startTime
	nm.EndTime = endTime
	database.DB.Save(&nm)

	// Schedule
	scheduleNightmode(b, chat.Id, startTime, durationSeconds)

	_, err = msg.Reply(b,
		fmt.Sprintf("🌙 <b>Night mode enabled!</b>\n⏰ Locks at: <b>%s</b>\n⌛ Duration: <b>%s</b>",
			startTime, endFlag),
		&gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func scheduleNightmode(b *gotgbot.Bot, chatID int64, startTime string, durationSeconds int) {
	tag := fmt.Sprintf("night_%d", chatID)
	scheduler.RemoveByTag(tag)

	// Parse HH:MM
	parts := strings.Split(startTime, ":")
	if len(parts) != 2 {
		return
	}
	hour := parts[0]
	min := parts[1]

	// Lock job
	scheduler.Every(1).Day().At(fmt.Sprintf("%s:%s", hour, min)).Tag(tag).Do(func() {
		lockChat(b, chatID)
		// Schedule unlock
		time.AfterFunc(time.Duration(durationSeconds)*time.Second, func() {
			unlockChat(b, chatID)
		})
	})
}

func lockChat(b *gotgbot.Bot, chatID int64) {
	_, err := b.SetChatPermissions(chatID, gotgbot.ChatPermissions{}, nil)
	if err != nil {
		log.Printf("[Nightmode] Failed to lock %d: %v", chatID, err)
		return
	}
	_, _ = b.SendMessage(chatID, "🌙 <b>Night mode activated!</b> Chat is now locked. Sweet dreams! 😴",
		&gotgbot.SendMessageOpts{ParseMode: "HTML"})
}

func unlockChat(b *gotgbot.Bot, chatID int64) {
	_, err := b.SetChatPermissions(chatID, gotgbot.ChatPermissions{
		CanSendMessages:       true,
		CanSendAudios:         true,
		CanSendDocuments:      true,
		CanSendPhotos:         true,
		CanSendVideos:         true,
		CanSendVideoNotes:     true,
		CanSendVoiceNotes:     true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
		CanInviteUsers:        true,
	}, nil)
	if err != nil {
		log.Printf("[Nightmode] Failed to unlock %d: %v", chatID, err)
		return
	}
	_, _ = b.SendMessage(chatID, "☀️ <b>Good morning!</b> Night mode has ended. Chat is unlocked!",
		&gotgbot.SendMessageOpts{ParseMode: "HTML"})
}

func restoreNightmodes() {
	// Called on startup — but needs bot reference
	// Will be wired up in main.go via SetBot()
	log.Println("[Nightmode] Restore called — wire up in main.go")
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func extractFlag(args, flag string) string {
	re := regexp.MustCompile(`-` + flag + `=(\S+)`)
	matches := re.FindStringSubmatch(args)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func parseDuration(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	re := regexp.MustCompile(`^(\d+)([hm])$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 3 {
		return 0, fmt.Errorf("invalid duration")
	}
	var val int
	fmt.Sscanf(matches[1], "%d", &val)
	switch matches[2] {
	case "h":
		return val * 3600, nil
	case "m":
		return val * 60, nil
	}
	return 0, fmt.Errorf("unknown unit")
}

func calculateEndTime(startTime string, durationSeconds int) string {
	parts := strings.Split(startTime, ":")
	if len(parts) != 2 {
		return "06:00"
	}
	var h, m int
	fmt.Sscanf(parts[0], "%d", &h)
	fmt.Sscanf(parts[1], "%d", &m)

	total := h*3600 + m*60 + durationSeconds
	endH := (total / 3600) % 24
	endM := (total % 3600) / 60
	return fmt.Sprintf("%02d:%02d", endH, endM)
}
