// RoboKaty - Rose-style Telegram Group Manager Bot
// modules/quotly/quotly.go — Mirrors misskaty/plugins/quotly.py
// Generates quote stickers from messages

package quotly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Quotly"
const HELP = `
/q - Generate a quote sticker from replied message
/q [color] - Custom background color (e.g. /q #1a1a2e)
`

const quotlyAPI = "https://bot.lyo.su/quote/generate"

type quotlyRequest struct {
	Type     string         `json:"type"`
	Format   string         `json:"format"`
	BackgroundColor string  `json:"backgroundColor"`
	Width    int            `json:"width"`
	Scale    int            `json:"scale"`
	Messages []quotlyMsg    `json:"messages"`
}

type quotlyMsg struct {
	EntityType  string        `json:"entityType"`
	Avatar      bool          `json:"avatar"`
	From        quotlyFrom    `json:"from"`
	Text        string        `json:"text"`
	ReplyMessage *quotlyReply `json:"replyMessage,omitempty"`
}

type quotlyFrom struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Photo     string `json:"photo"`
	Username  string `json:"username"`
}

type quotlyReply struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

type quotlyResponse struct {
	Ok     bool   `json:"ok"`
	Image  string `json:"image"`
}

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("q", generateQuote))
	log.Println("[Quotly] ✅ Module loaded")
}

func generateQuote(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage

	if msg.ReplyToMessage == nil {
		_, err := msg.Reply(b, "❌ Reply to a message to quote it.", nil)
		return err
	}

	reply := msg.ReplyToMessage
	if reply.Text == "" && reply.Caption == "" {
		_, err := msg.Reply(b, "❌ Can only quote text messages.", nil)
		return err
	}

	bgColor := utils.GetCommandArgs(msg)
	if bgColor == "" {
		bgColor = "#1b1c1d"
	}

	from := reply.From
	if from == nil {
		_, err := msg.Reply(b, "❌ Can't quote this message.", nil)
		return err
	}

	text := reply.Text
	if text == "" {
		text = reply.Caption
	}

	name := from.FirstName
	if from.LastName != "" {
		name += " " + from.LastName
	}

	reqBody := quotlyRequest{
		Type:            "quote",
		Format:          "webp",
		BackgroundColor: bgColor,
		Width:           512,
		Scale:           2,
		Messages: []quotlyMsg{
			{
				EntityType: "user",
				Avatar:     true,
				From: quotlyFrom{
					ID:       from.Id,
					Name:     name,
					Username: from.Username,
				},
				Text: text,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		_, err = msg.Reply(b, "❌ Failed to prepare request.", nil)
		return err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(quotlyAPI, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		_, err = msg.Reply(b, "❌ Quotly API unavailable.", nil)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result quotlyResponse
	if err := json.Unmarshal(respBody, &result); err != nil || !result.Ok {
		_, err = msg.Reply(b, "❌ Failed to generate quote.", nil)
		return err
	}

	// Send as sticker (base64 image)
	imgData := result.Image
	if len(imgData) > 10 {
		// Strip data URL prefix if present
		if idx := bytes.Index([]byte(imgData), []byte(",")); idx != -1 {
			imgData = imgData[idx+1:]
		}
	}

	// Send as document (webp sticker)
	_, err = b.SendSticker(msg.Chat.Id, fmt.Sprintf("data:image/webp;base64,%s", imgData), &gotgbot.SendStickerOpts{
		ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
	})
	if err != nil {
		// Fallback: send as photo
		_, err = msg.Reply(b, fmt.Sprintf("<code>Quote generated!</code>\n\n(Sticker send failed: %s)", err.Error()), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}
	return err
}
