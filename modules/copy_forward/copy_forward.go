// RoboKaty - Rose-style Telegram Group Manager Bot
// modules/copy_forward/copy_forward.go — Mirrors misskaty/plugins/copy_forward.py

package copy_forward

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "CopyForward"
const HELP = `
/copy [chat_id] [msg_id] - Copy a message from another chat
/forward [chat_id] [msg_id] - Forward a message from another chat

Sudo only commands.
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("copy", copyMsg))
	dispatcher.AddHandler(utils.OnCmd("forward", forwardMsg))
	log.Println("[CopyForward] ✅ Module loaded")
}

func copyMsg(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	sender := ctx.EffectiveSender

	if !config.IsSudo(sender.Id()) {
		return nil
	}

	args := strings.Fields(utils.GetCommandArgs(msg))
	if len(args) < 2 {
		_, err := msg.Reply(b, "❌ Usage: /copy [chat_id] [msg_id]", nil)
		return err
	}

	fromChatID := args[0]
	msgID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		_, err = msg.Reply(b, "❌ Invalid message ID.", nil)
		return err
	}

	_, err = b.CopyMessage(msg.Chat.Id, fromChatID, msgID, nil)
	if err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
		return err
	}
	return nil
}

func forwardMsg(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	sender := ctx.EffectiveSender

	if !config.IsSudo(sender.Id()) {
		return nil
	}

	args := strings.Fields(utils.GetCommandArgs(msg))
	if len(args) < 2 {
		_, err := msg.Reply(b, "❌ Usage: /forward [chat_id] [msg_id]", nil)
		return err
	}

	fromChatID := args[0]
	msgID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		_, err = msg.Reply(b, "❌ Invalid message ID.", nil)
		return err
	}

	_, err = b.ForwardMessage(msg.Chat.Id, fromChatID, msgID, nil)
	if err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed: %s", err.Error()), nil)
		return err
	}
	return nil
}
