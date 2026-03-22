// RoboKaty - Rose-style Telegram Group Manager Bot
// modules/urban_dict/urban_dict.go — Mirrors misskaty/plugins/urban_dict.py

package urban_dict

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Urban"
const HELP = `
/ud [WORD] or /urban [WORD] - Look up a word on Urban Dictionary
`

type udResponse struct {
	List []struct {
		Word       string `json:"word"`
		Definition string `json:"definition"`
		Example    string `json:"example"`
		ThumbsUp   int    `json:"thumbs_up"`
		ThumbsDown int    `json:"thumbs_down"`
		Permalink  string `json:"permalink"`
	} `json:"list"`
}

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("ud", lookup))
	dispatcher.AddHandler(utils.OnCmd("urban", lookup))
	log.Println("[Urban] ✅ Module loaded")
}

func lookup(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	word := utils.GetCommandArgs(msg)
	if word == "" {
		_, err := msg.Reply(b, "❌ Usage: /ud [WORD]", nil)
		return err
	}

	apiURL := fmt.Sprintf("https://api.urbandictionary.com/v0/define?term=%s", url.QueryEscape(word))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't reach Urban Dictionary.", nil)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result udResponse
	if err := json.Unmarshal(body, &result); err != nil || len(result.List) == 0 {
		_, err = msg.Reply(b, fmt.Sprintf("❌ No definition found for: <code>%s</code>", word), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	def := result.List[0]
	definition := def.Definition
	if len(definition) > 400 {
		definition = definition[:400] + "..."
	}
	example := def.Example
	if len(example) > 200 {
		example = example[:200] + "..."
	}

	text := fmt.Sprintf(
		"📖 <b>%s</b>\n\n"+
			"📝 <b>Definition:</b>\n%s\n\n"+
			"💡 <b>Example:</b>\n<i>%s</i>\n\n"+
			"👍 %d  👎 %d",
		def.Word,
		definition,
		example,
		def.ThumbsUp,
		def.ThumbsDown,
	)

	keyboard := utils.IKB([][]gotgbot.InlineKeyboardButton{
		{utils.BtnURL("🔗 View on Urban", def.Permalink)},
	})

	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: keyboard,
	})
	return err
}
