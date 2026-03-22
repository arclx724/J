// RoboKaty — utils/helpers.go
// All utility helpers: admin checks, user extraction, keyboard builder, command parser

package utils

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"

	"github.com/robokatybot/robokaty/config"
)

// ─── Admin Cache ──────────────────────────────────────────────────────────────

type adminCacheEntry struct {
	admins      []int64
	lastUpdated time.Time
}

var (
	adminCache   = make(map[int64]*adminCacheEntry)
	adminCacheMu sync.RWMutex
	cacheTTL     = time.Hour
)

func GetAdminList(b *gotgbot.Bot, chatID int64) ([]int64, error) {
	adminCacheMu.RLock()
	entry, ok := adminCache[chatID]
	adminCacheMu.RUnlock()

	if ok && time.Since(entry.lastUpdated) < cacheTTL {
		return entry.admins, nil
	}

	members, err := b.GetChatAdministrators(chatID, nil)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.GetUser().Id)
	}

	adminCacheMu.Lock()
	adminCache[chatID] = &adminCacheEntry{admins: ids, lastUpdated: time.Now()}
	adminCacheMu.Unlock()

	return ids, nil
}

func InvalidateAdminCache(chatID int64) {
	adminCacheMu.Lock()
	delete(adminCache, chatID)
	adminCacheMu.Unlock()
}

func IsAdmin(b *gotgbot.Bot, chatID, userID int64) bool {
	if config.IsSudo(userID) {
		return true
	}
	member, err := b.GetChatMember(chatID, userID, nil)
	if err != nil {
		return false
	}
	s := member.GetStatus()
	return s == "creator" || s == "administrator"
}

func IsOwner(b *gotgbot.Bot, chatID, userID int64) bool {
	member, err := b.GetChatMember(chatID, userID, nil)
	if err != nil {
		return false
	}
	return member.GetStatus() == "creator"
}

func IsBotAdmin(b *gotgbot.Bot, chatID int64) bool {
	return IsAdmin(b, chatID, b.Id)
}

func IsInAdminList(b *gotgbot.Bot, chatID, userID int64) bool {
	admins, err := GetAdminList(b, chatID)
	if err != nil {
		return false
	}
	for _, id := range admins {
		if id == userID {
			return true
		}
	}
	return false
}

func HasPermission(b *gotgbot.Bot, chatID, userID int64, permission string) bool {
	if config.IsSudo(userID) {
		return true
	}
	member, err := b.GetChatMember(chatID, userID, nil)
	if err != nil {
		return false
	}
	if member.GetStatus() == "creator" {
		return true
	}
	adm, ok := member.(gotgbot.ChatMemberAdministrator)
	if !ok {
		return false
	}
	switch permission {
	case "can_delete_messages":
		return adm.CanDeleteMessages
	case "can_restrict_members":
		return adm.CanRestrictMembers
	case "can_promote_members":
		return adm.CanPromoteMembers
	case "can_change_info":
		return adm.CanChangeInfo
	case "can_invite_users":
		return adm.CanInviteUsers
	case "can_pin_messages":
		return adm.CanPinMessages
	case "can_manage_video_chats":
		return adm.CanManageVideoChats
	case "can_manage_chat":
		return adm.CanManageChat
	case "can_post_messages":
		return adm.CanPostMessages
	case "can_edit_messages":
		return adm.CanEditMessages
	}
	return false
}

// ─── User Extraction ──────────────────────────────────────────────────────────

func ExtractUser(b *gotgbot.Bot, ctx *ext.Context) (*gotgbot.User, error) {
	msg := ctx.EffectiveMessage

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		return msg.ReplyToMessage.From, nil
	}

	args := strings.Fields(msg.Text)
	if len(args) < 2 {
		return nil, fmt.Errorf("no user specified")
	}

	// text_mention entity
	for _, e := range msg.Entities {
		if e.Type == "text_mention" && e.User != nil {
			return e.User, nil
		}
	}

	target := args[1]
	if strings.HasPrefix(target, "@") {
		chat, err := b.GetChat(target, nil)
		if err != nil {
			return nil, fmt.Errorf("user %q not found", target)
		}
		return &gotgbot.User{Id: chat.Id, FirstName: chat.FirstName, Username: chat.Username}, nil
	}

	uid, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user identifier %q", target)
	}
	chat, err := b.GetChat(uid, nil)
	if err != nil {
		return nil, fmt.Errorf("user id %d not found", uid)
	}
	return &gotgbot.User{Id: chat.Id, FirstName: chat.FirstName, Username: chat.Username}, nil
}

func ExtractUserAndReason(b *gotgbot.Bot, ctx *ext.Context) (*gotgbot.User, string, error) {
	msg := ctx.EffectiveMessage

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		parts := strings.SplitN(msg.Text, " ", 2)
		reason := ""
		if len(parts) == 2 {
			reason = strings.TrimSpace(parts[1])
		}
		return msg.ReplyToMessage.From, reason, nil
	}

	args := strings.Fields(msg.Text)
	if len(args) < 2 {
		return nil, "", fmt.Errorf("no user specified")
	}

	// text_mention entity
	for _, e := range msg.Entities {
		if e.Type == "text_mention" && e.User != nil {
			reason := ""
			if len(args) > 2 {
				reason = strings.Join(args[2:], " ")
			}
			return e.User, reason, nil
		}
	}

	target := args[1]
	reason := ""
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}

	if strings.HasPrefix(target, "@") {
		chat, err := b.GetChat(target, nil)
		if err != nil {
			return nil, "", fmt.Errorf("user %q not found", target)
		}
		return &gotgbot.User{Id: chat.Id, FirstName: chat.FirstName, Username: chat.Username}, reason, nil
	}

	uid, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("invalid identifier %q", target)
	}
	chat, err := b.GetChat(uid, nil)
	if err != nil {
		return nil, "", fmt.Errorf("user id %d not found", uid)
	}
	return &gotgbot.User{Id: chat.Id, FirstName: chat.FirstName, Username: chat.Username}, reason, nil
}

func TimeConverter(input string) (time.Time, error) {
	if len(input) < 2 {
		return time.Time{}, fmt.Errorf("invalid time format %q — use e.g. 1h 30m 2d 1w", input)
	}
	unit := input[len(input)-1]
	val, err := strconv.Atoi(input[:len(input)-1])
	if err != nil || val <= 0 {
		return time.Time{}, fmt.Errorf("invalid time value %q", input)
	}
	if val > 99 {
		return time.Time{}, fmt.Errorf("time value cannot exceed 99")
	}
	now := time.Now()
	switch unit {
	case 's':
		return now.Add(time.Duration(val) * time.Second), nil
	case 'm':
		return now.Add(time.Duration(val) * time.Minute), nil
	case 'h':
		return now.Add(time.Duration(val) * time.Hour), nil
	case 'd':
		return now.AddDate(0, 0, val), nil
	case 'w':
		return now.AddDate(0, 0, val*7), nil
	}
	return time.Time{}, fmt.Errorf("unknown time unit %q — use s/m/h/d/w", string(unit))
}

// ─── Keyboard Builder ─────────────────────────────────────────────────────────

func IKB(rows [][]gotgbot.InlineKeyboardButton) *gotgbot.InlineKeyboardMarkup {
	return &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func Btn(text, data string) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: text, CallbackData: data}
}

func BtnURL(text, url string) gotgbot.InlineKeyboardButton {
	return gotgbot.InlineKeyboardButton{Text: text, Url: url}
}

func SingleBtn(text, data string) *gotgbot.InlineKeyboardMarkup {
	return IKB([][]gotgbot.InlineKeyboardButton{{Btn(text, data)}})
}

func TwoBtn(t1, d1, t2, d2 string) *gotgbot.InlineKeyboardMarkup {
	return IKB([][]gotgbot.InlineKeyboardButton{{Btn(t1, d1), Btn(t2, d2)}})
}

// ─── Command Parser (Multi-prefix: / . !) ─────────────────────────────────────

func CommandFilter(command string) func(*gotgbot.Message) bool {
	return func(msg *gotgbot.Message) bool {
		if msg == nil {
			return false
		}
		text := msg.Text
		if text == "" {
			text = msg.Caption
		}
		if text == "" {
			return false
		}
		prefixes := config.CommandHandler
		if len(prefixes) == 0 {
			prefixes = []string{"/", ".", "!"}
		}
		for _, prefix := range prefixes {
			candidate := strings.ToLower(prefix + command)
			lower := strings.ToLower(text)
			if strings.HasPrefix(lower, candidate) {
				rest := text[len(candidate):]
				if rest == "" || rest[0] == ' ' || rest[0] == '@' || rest[0] == '\n' {
					return true
				}
			}
		}
		return false
	}
}

func GetCommand(msg *gotgbot.Message) string {
	if msg == nil {
		return ""
	}
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	word := fields[0]
	prefixes := config.CommandHandler
	if len(prefixes) == 0 {
		prefixes = []string{"/", ".", "!"}
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(word, prefix) {
			cmd := word[len(prefix):]
			if idx := strings.Index(cmd, "@"); idx != -1 {
				cmd = cmd[:idx]
			}
			return strings.ToLower(cmd)
		}
	}
	return ""
}

func GetCommandArgs(msg *gotgbot.Message) string {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func OnCmd(command string, fn func(*gotgbot.Bot, *ext.Context) error) *handlers.Message {
	return handlers.NewMessage(CommandFilter(command), fn)
}

func OnCmds(commands []string, fn func(*gotgbot.Bot, *ext.Context) error) *handlers.Message {
	return handlers.NewMessage(func(msg *gotgbot.Message) bool {
		for _, cmd := range commands {
			if CommandFilter(cmd)(msg) {
				return true
			}
		}
		return false
	}, fn)
}

// ─── Chat Type Helpers ────────────────────────────────────────────────────────

func IsPrivateChat(chat *gotgbot.Chat) bool {
	return chat.Type == "private"
}

func IsGroupChat(chat *gotgbot.Chat) bool {
	return chat.Type == "group" || chat.Type == "supergroup"
}

// ─── Text Helpers ─────────────────────────────────────────────────────────────

func MentionHTML(userID int64, name string) string {
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, userID, name)
}

func FullName(user *gotgbot.User) string {
	if user.LastName != "" {
		return user.FirstName + " " + user.LastName
	}
	return user.FirstName
}

func Capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ─── HTTP Helper ──────────────────────────────────────────────────────────────

func FetchJSON(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ─── File Helpers ─────────────────────────────────────────────────────────────

func IsBlockedExtension(fileName string, blockedExts []string, blockNoExt bool) bool {
	if fileName == "" {
		return blockNoExt
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	if ext == "" {
		return blockNoExt
	}
	for _, blocked := range blockedExts {
		if strings.EqualFold(blocked, ext) {
			return true
		}
	}
	return false
}
