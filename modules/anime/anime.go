// RoboKaty - Rose-style Telegram Group Manager Bot
// modules/anime/anime.go — Mirrors misskaty/plugins/anime.py

package anime

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Anime"
const HELP = `
/anime [NAME] - Search for an anime
/manga [NAME] - Search for a manga
/character [NAME] - Search for an anime character
`

const jikanBase = "https://api.jikan.moe/v4"

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("anime", searchAnime))
	dispatcher.AddHandler(utils.OnCmd("manga", searchManga))
	dispatcher.AddHandler(utils.OnCmd("character", searchCharacter))
	log.Println("[Anime] ✅ Module loaded")
}

// ── Anime ─────────────────────────────────────────────────────────────────────

type jikanAnimeResult struct {
	Data []struct {
		MalID    int    `json:"mal_id"`
		Title    string `json:"title"`
		Score    float64 `json:"score"`
		Episodes int    `json:"episodes"`
		Status   string `json:"status"`
		Synopsis string `json:"synopsis"`
		Images   struct {
			JPG struct {
				ImageURL string `json:"image_url"`
			} `json:"jpg"`
		} `json:"images"`
		URL string `json:"url"`
	} `json:"data"`
}

func searchAnime(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	query := utils.GetCommandArgs(msg)
	if query == "" {
		_, err := msg.Reply(b, "❌ Usage: /anime [NAME]", nil)
		return err
	}

	apiURL := fmt.Sprintf("%s/anime?q=%s&limit=1", jikanBase, url.QueryEscape(query))
	data, err := fetchJSON(apiURL)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't reach Jikan API.", nil)
		return err
	}

	var result jikanAnimeResult
	if err := json.Unmarshal(data, &result); err != nil || len(result.Data) == 0 {
		_, err = msg.Reply(b, fmt.Sprintf("❌ No results for: <code>%s</code>", query), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	a := result.Data[0]
	synopsis := a.Synopsis
	if len(synopsis) > 300 {
		synopsis = synopsis[:300] + "..."
	}

	text := fmt.Sprintf(
		"🎌 <b>%s</b>\n\n"+
			"⭐ Score: <b>%.1f</b>\n"+
			"📺 Episodes: <b>%d</b>\n"+
			"📊 Status: <b>%s</b>\n\n"+
			"📝 %s\n\n"+
			"🔗 <a href='%s'>More Info</a>",
		a.Title,
		a.Score,
		a.Episodes,
		a.Status,
		synopsis,
		a.URL,
	)

	if a.Images.JPG.ImageURL != "" {
		_, err = b.SendPhoto(msg.Chat.Id, a.Images.JPG.ImageURL, &gotgbot.SendPhotoOpts{
			Caption:         text,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err
	}

	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ── Manga ─────────────────────────────────────────────────────────────────────

func searchManga(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	query := utils.GetCommandArgs(msg)
	if query == "" {
		_, err := msg.Reply(b, "❌ Usage: /manga [NAME]", nil)
		return err
	}

	apiURL := fmt.Sprintf("%s/manga?q=%s&limit=1", jikanBase, url.QueryEscape(query))
	data, err := fetchJSON(apiURL)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't reach Jikan API.", nil)
		return err
	}

	var result jikanAnimeResult
	if err := json.Unmarshal(data, &result); err != nil || len(result.Data) == 0 {
		_, err = msg.Reply(b, fmt.Sprintf("❌ No results for: <code>%s</code>", query), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	a := result.Data[0]
	synopsis := a.Synopsis
	if len(synopsis) > 300 {
		synopsis = synopsis[:300] + "..."
	}

	text := fmt.Sprintf(
		"📚 <b>%s</b>\n\n"+
			"⭐ Score: <b>%.1f</b>\n"+
			"📊 Status: <b>%s</b>\n\n"+
			"📝 %s\n\n"+
			"🔗 <a href='%s'>More Info</a>",
		a.Title, a.Score, a.Status, synopsis, a.URL,
	)

	if a.Images.JPG.ImageURL != "" {
		_, err = b.SendPhoto(msg.Chat.Id, a.Images.JPG.ImageURL, &gotgbot.SendPhotoOpts{
			Caption:         text,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err
	}

	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ── Character ─────────────────────────────────────────────────────────────────

type jikanCharResult struct {
	Data []struct {
		Character struct {
			MalID  int    `json:"mal_id"`
			Name   string `json:"name"`
			Images struct {
				JPG struct {
					ImageURL string `json:"image_url"`
				} `json:"jpg"`
			} `json:"images"`
			URL string `json:"url"`
		} `json:"character"`
		Favorites int    `json:"favorites"`
		About     string `json:"about"`
	} `json:"data"`
}

func searchCharacter(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	query := utils.GetCommandArgs(msg)
	if query == "" {
		_, err := msg.Reply(b, "❌ Usage: /character [NAME]", nil)
		return err
	}

	apiURL := fmt.Sprintf("%s/characters?q=%s&limit=1", jikanBase, url.QueryEscape(query))
	data, err := fetchJSON(apiURL)
	if err != nil {
		_, err = msg.Reply(b, "❌ Couldn't reach Jikan API.", nil)
		return err
	}

	var result jikanCharResult
	if err := json.Unmarshal(data, &result); err != nil || len(result.Data) == 0 {
		_, err = msg.Reply(b, fmt.Sprintf("❌ No results for: <code>%s</code>", query), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	c := result.Data[0]
	about := c.About
	if len(about) > 300 {
		about = about[:300] + "..."
	}
	about = strings.ReplaceAll(about, "\n\n", "\n")

	text := fmt.Sprintf(
		"👤 <b>%s</b>\n\n"+
			"❤️ Favorites: <b>%d</b>\n\n"+
			"📝 %s\n\n"+
			"🔗 <a href='%s'>More Info</a>",
		c.Character.Name,
		c.Favorites,
		about,
		c.Character.URL,
	)

	if c.Character.Images.JPG.ImageURL != "" {
		_, err = b.SendPhoto(msg.Chat.Id, c.Character.Images.JPG.ImageURL, &gotgbot.SendPhotoOpts{
			Caption:         text,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err
	}

	_, err = msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func fetchJSON(apiURL string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
