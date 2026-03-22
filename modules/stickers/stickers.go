// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Mirrors: misskaty/plugins/stickers.py — ALL commands

package stickers

import (
	"fmt"
	"log"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Stickers"
const HELP = `
/getsticker or /toimage - Get file_id and info of a sticker (reply to sticker)
/stickerid - Get the file_id of a sticker (reply to sticker)
/kang or /curi [emoji] - Steal a sticker into your personal pack (reply to sticker/image)
/unkang - Remove a sticker from your pack (reply to your sticker)
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmds([]string{"getsticker", "toimage"}, getSticker))
	dispatcher.AddHandler(utils.OnCmd("stickerid", getSticker))
	dispatcher.AddHandler(utils.OnCmds([]string{"kang", "curi"}, kangSticker))
	dispatcher.AddHandler(utils.OnCmd("unkang", unkangSticker))
	log.Println("[Stickers] ✅ Module loaded")
}

func getSticker(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.ReplyToMessage == nil || msg.ReplyToMessage.Sticker == nil {
		_, err := msg.Reply(b, "❌ Reply to a sticker.", nil)
		return err
	}
	s := msg.ReplyToMessage.Sticker
	stickerType := "Static"
	if s.IsAnimated {
		stickerType = "Animated"
	} else if s.IsVideo {
		stickerType = "Video"
	}
	text := fmt.Sprintf(
		"🎯 <b>Sticker Info</b>\n\n"+
			"📦 <b>Pack:</b> <code>%s</code>\n"+
			"🆔 <b>File ID:</b>\n<code>%s</code>\n"+
			"😊 <b>Emoji:</b> %s\n"+
			"📐 <b>Size:</b> %dx%d\n"+
			"🎭 <b>Type:</b> %s",
		s.SetName, s.FileId, s.Emoji, s.Width, s.Height, stickerType,
	)
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func kangSticker(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		_, err := msg.Reply(b, "❌ Can't kang as anonymous.", nil)
		return err
	}
	if msg.ReplyToMessage == nil {
		_, err := msg.Reply(b, "❌ Reply to a sticker or image to kang it.", nil)
		return err
	}

	emoji := utils.GetCommandArgs(msg)
	if emoji == "" {
		emoji = "🤔"
	}
	// Take only first emoji
	runes := []rune(emoji)
	if len(runes) > 0 {
		emoji = string(runes[0])
	}

	userID := msg.From.Id
	packName := fmt.Sprintf("u%d_by_%s", userID, strings.ReplaceAll(b.Username, "_", ""))
	packTitle := fmt.Sprintf("%s's Pack", msg.From.FirstName)

	reply := msg.ReplyToMessage
	var stickerFile gotgbot.InputFile
	format := "static"

	if reply.Sticker != nil {
		stickerFile = gotgbot.InputFileByID(reply.Sticker.FileId)
		if reply.Sticker.Emoji != "" {
			emoji = reply.Sticker.Emoji
		}
		if reply.Sticker.IsAnimated {
			format = "animated"
		} else if reply.Sticker.IsVideo {
			format = "video"
		}
	} else if reply.Photo != nil && len(reply.Photo) > 0 {
		stickerFile = gotgbot.InputFileByID(reply.Photo[len(reply.Photo)-1].FileId)
		format = "static"
	} else if reply.Document != nil {
		stickerFile = gotgbot.InputFileByID(reply.Document.FileId)
		format = "static"
	} else {
		_, err := msg.Reply(b, "❌ Reply to a sticker or image.", nil)
		return err
	}

	inputSticker := gotgbot.InputSticker{
		Sticker:   stickerFile,
		Format:    format,
		EmojiList: []string{emoji},
	}

	// Try adding to existing pack first
	_, addErr := b.AddStickerToSet(userID, packName, inputSticker, nil)
	if addErr != nil {
		// Pack doesn't exist — create it
		_, createErr := b.CreateNewStickerSet(userID, packName, packTitle, []gotgbot.InputSticker{inputSticker}, nil)
		if createErr != nil {
			_, err := msg.Reply(b, fmt.Sprintf("❌ Failed to kang: %s", createErr.Error()), nil)
			return err
		}
	}

	packLink := fmt.Sprintf("https://t.me/addstickers/%s", packName)
	_, err := msg.Reply(b,
		fmt.Sprintf("✅ <b>Kanged!</b> %s\n\n📦 <a href='%s'>View Pack</a>", emoji, packLink),
		&gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func unkangSticker(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		_, err := msg.Reply(b, "❌ Can't unkang as anonymous.", nil)
		return err
	}
	if msg.ReplyToMessage == nil || msg.ReplyToMessage.Sticker == nil {
		_, err := msg.Reply(b, "❌ Reply to a sticker to unkang it.", nil)
		return err
	}

	sticker := msg.ReplyToMessage.Sticker
	// Check if it belongs to user's pack
	expectedPack := fmt.Sprintf("u%d_by_%s", msg.From.Id, strings.ReplaceAll(b.Username, "_", ""))
	if !strings.Contains(sticker.SetName, fmt.Sprintf("u%d_by_", msg.From.Id)) {
		_, err := msg.Reply(b, "❌ This sticker is not from your pack.", nil)
		return err
	}

	_ = expectedPack

	// DeleteStickerFromSet
	_, err := b.DeleteStickerFromSet(sticker.FileId, nil)
	if err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Failed to unkang: %s", err.Error()), nil)
		return err
	}

	_, err = msg.Reply(b, "✅ Sticker removed from your pack!", nil)
	return err
}
