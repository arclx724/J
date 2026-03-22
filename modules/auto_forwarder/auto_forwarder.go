// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Mirrors: misskaty/plugins/auto_forwarder.py

package auto_forwarder

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"

	"github.com/robokatybot/robokaty/config"
)

const MODULE = "AutoForwarder"
const HELP = `
Auto Forwarder is configured via config.env:

FORWARD_FROM_CHAT_ID - Chat IDs to forward FROM (space-separated)
FORWARD_TO_CHAT_ID   - Chat IDs to forward TO (space-separated)
FORWARD_FILTERS      - Types: video document photo audio gif sticker text
BLOCKED_EXTENSIONS   - Extensions to skip (e.g. html htm txt)
MINIMUM_FILE_SIZE    - Minimum file size in bytes (optional)
`

func Load(dispatcher *ext.Dispatcher) {
	if len(config.ForwardFromChatId) == 0 || len(config.ForwardToChatId) == 0 {
		log.Println("[AutoForwarder] ⚠️ Skipping — FORWARD_FROM/TO not configured")
		return
	}
	dispatcher.AddHandlerToGroup(handlers.NewMessage(nil, handleForward), 50)
	log.Printf("[AutoForwarder] ✅ Forwarding from %v → %v", config.ForwardFromChatId, config.ForwardToChatId)
}

func handleForward(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg == nil {
		return ext.ContinueGroups
	}
	if !isSourceChat(msg.Chat.Id) {
		return ext.ContinueGroups
	}
	if (msg.ForwardFromChat != nil || msg.ForwardFrom != nil) && !hasFilter("forwarded") {
		return ext.ContinueGroups
	}
	msgType, fileName := getMsgType(msg)
	if msgType == "" || !hasFilter(msgType) {
		return ext.ContinueGroups
	}
	if fileName != "" && isBlockedExt(fileName) {
		return ext.ContinueGroups
	}
	if !meetsMinSize(msg) {
		return ext.ContinueGroups
	}
	for _, targetID := range config.ForwardToChatId {
		if _, err := b.ForwardMessage(targetID, msg.Chat.Id, msg.MessageId, nil); err != nil {
			log.Printf("[AutoForwarder] Failed to forward to %d: %v", targetID, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return ext.ContinueGroups
}

func isSourceChat(chatID int64) bool {
	for _, id := range config.ForwardFromChatId {
		if id == chatID {
			return true
		}
	}
	return false
}

func hasFilter(filterType string) bool {
	if len(config.ForwardFilters) >= 9 {
		return true
	}
	for _, f := range config.ForwardFilters {
		if strings.EqualFold(f, filterType) {
			return true
		}
	}
	return false
}

func getMsgType(msg *gotgbot.Message) (string, string) {
	switch {
	case msg.Video != nil:
		return "video", msg.Video.FileName
	case msg.Document != nil:
		return "document", msg.Document.FileName
	case msg.Photo != nil && len(msg.Photo) > 0:
		return "photo", ""
	case msg.Audio != nil:
		return "audio", msg.Audio.FileName
	case msg.Animation != nil:
		return "gif", msg.Animation.FileName
	case msg.Sticker != nil:
		return "sticker", ""
	case msg.Text != "":
		return "text", ""
	case msg.Poll != nil:
		return "poll", ""
	}
	return "", ""
}

func isBlockedExt(fileName string) bool {
	if fileName == "" {
		return config.BlockFilesWithoutExtension
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	if ext == "" {
		return config.BlockFilesWithoutExtension
	}
	for _, blocked := range config.BlockedExtensions {
		if strings.EqualFold(blocked, ext) {
			return true
		}
	}
	return false
}

func meetsMinSize(msg *gotgbot.Message) bool {
	if config.MinimumFileSize == "" {
		return true
	}
	var minSize int64
	fmt.Sscanf(config.MinimumFileSize, "%d", &minSize)
	if minSize <= 0 {
		return true
	}
	var fileSize int64
	switch {
	case msg.Video != nil:
		fileSize = msg.Video.FileSize
	case msg.Document != nil:
		fileSize = msg.Document.FileSize
	case msg.Audio != nil:
		fileSize = msg.Audio.FileSize
	}
	return fileSize >= minSize
}
