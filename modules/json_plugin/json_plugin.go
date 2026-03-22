// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// View Telegram message structure as JSON

package json_plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"

	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "JSON"
const HELP = `
/json - View the structure of a Telegram message as JSON (reply to a message)

Useful for developers to inspect message objects.
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("json", jsonify))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("closejson_"), closeCB))
	log.Println("[JSON] ✅ Module loaded")
}

func jsonify(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage

	// Use replied message if available, otherwise the command message itself
	target := msg
	if msg.ReplyToMessage != nil {
		target = msg.ReplyToMessage
	}

	// Marshal to pretty JSON
	data, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed to marshal: %s", err.Error()), nil)
		return err
	}

	jsonStr := string(data)
	senderID := int64(0)
	if msg.From != nil {
		senderID = msg.From.Id
	}

	keyboard := utils.SingleBtn("❌ Close", fmt.Sprintf("closejson_%d", senderID))

	// Telegram message limit is 4096 chars
	if len(jsonStr) <= 4000 {
		_, err = msg.Reply(b,
			fmt.Sprintf("<code>%s</code>", jsonStr),
			&gotgbot.SendMessageOpts{
				ParseMode:   "HTML",
				ReplyMarkup: keyboard,
			})
		return err
	}

	// Too long — send as file
	tmpFile := fmt.Sprintf("/tmp/msg_%d.json", target.MessageId)
	if writeErr := os.WriteFile(tmpFile, data, 0644); writeErr != nil {
		_, err = msg.Reply(b, "❌ Failed to write JSON file.", nil)
		return err
	}
	defer os.Remove(tmpFile)

	f, openErr := os.Open(tmpFile)
	if openErr != nil {
		_, err = msg.Reply(b, "❌ Failed to open JSON file.", nil)
		return err
	}
	defer f.Close()

	_, err = b.SendDocument(msg.Chat.Id, f, &gotgbot.SendDocumentOpts{
		Caption:         "📄 Message JSON (too long for inline display)",
		ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		ReplyMarkup:     keyboard,
	})
	return err
}

func closeCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	var ownerID int64
	fmt.Sscanf(cq.Data[len("closejson_"):], "%d", &ownerID)

	// Only the original requester can close it
	if ownerID != 0 && cq.From.Id != ownerID && !utils.IsAdmin(b, cq.Message.GetChat().Id, cq.From.Id) {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "❌ Only the requester can close this.",
			ShowAlert: true,
		})
		return err
	}

	_, _ = cq.Message.Delete(b, nil)
	_, err := cq.Answer(b, nil)
	return err
}
