// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Commands: /save, /addnote, /notes, /delnote, /clear, /deleteall
// Trigger:  #notename

package notes

import (
	"fmt"
	"log"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Notes"
const HELP = `
/notes - List all notes in this chat
/save [NAME] or /addnote [NAME] - Save a note (reply to a message)
#notename - Get a note
/delnote [NAME] or /clear [NAME] - Delete a note
/deleteall - Delete ALL notes in this chat

Supported types: Text, Photo, Video, Document, Audio, Voice, Sticker, Animation, Video Note

Supported fillings in note text:
{first} {last} {fullname} {username} {mention} {id} {chatname}
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(handlers.NewCommand("save", saveNote))
	dispatcher.AddHandler(handlers.NewCommand("addnote", saveNote))
	dispatcher.AddHandler(handlers.NewCommand("notes", listNotes))
	dispatcher.AddHandler(handlers.NewCommand("delnote", deleteNote))
	dispatcher.AddHandler(handlers.NewCommand("clear", deleteNote))
	dispatcher.AddHandler(handlers.NewCommand("deleteall", deleteAllNotes))

	// #notename trigger — group chats only
	dispatcher.AddHandler(handlers.NewMessage(
		message.Text,
		getNote,
	))

	// Confirm deleteall callback
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("notedel_"), deleteAllCB))

	log.Println("[Notes] ✅ Module loaded")
}

// ─────────────────────────────────────────────────────────────────────────────
// SAVE NOTE
// ─────────────────────────────────────────────────────────────────────────────

func saveNote(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	// Permission: can_change_info or private chat
	if !utils.IsPrivateChat(chat) {
		if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
			_, err := msg.Reply(b, "❌ You need can_change_info permission to save notes.", nil)
			return err
		}
	}

	if msg.ReplyToMessage == nil {
		_, err := msg.Reply(b, "❌ Reply to a message with /save [NOTE_NAME] to save it as a note.", nil)
		return err
	}

	args := utils.GetCommandArgs(msg)
	if args == "" {
		_, err := msg.Reply(b, "❌ Usage: /save [NOTE_NAME]", nil)
		return err
	}

	// Parse name and optional extra caption
	parts := strings.SplitN(args, " ", 2)
	name := strings.ToLower(strings.TrimSpace(parts[0]))
	var extraCaption string
	if len(parts) > 1 {
		extraCaption = strings.TrimSpace(parts[1])
	}

	replied := msg.ReplyToMessage
	note := &models.Note{
		ChatID: chat.Id,
		Name:   name,
	}

	// Determine media type and content
	if replied.Text != "" {
		note.MediaType = "text"
		if extraCaption != "" {
			note.Content = extraCaption
		} else {
			note.Content = replied.Text
		}
	} else if replied.Sticker != nil {
		note.MediaType = "sticker"
		note.FileID = replied.Sticker.FileId
	} else if replied.Animation != nil {
		note.MediaType = "animation"
		note.FileID = replied.Animation.FileId
		note.Content = getCaption(replied, extraCaption)
	} else if replied.Photo != nil && len(replied.Photo) > 0 {
		note.MediaType = "photo"
		note.FileID = replied.Photo[len(replied.Photo)-1].FileId
		note.Content = getCaption(replied, extraCaption)
	} else if replied.Document != nil {
		note.MediaType = "document"
		note.FileID = replied.Document.FileId
		note.Content = getCaption(replied, extraCaption)
	} else if replied.Video != nil {
		note.MediaType = "video"
		note.FileID = replied.Video.FileId
		note.Content = getCaption(replied, extraCaption)
	} else if replied.VideoNote != nil {
		note.MediaType = "video_note"
		note.FileID = replied.VideoNote.FileId
	} else if replied.Audio != nil {
		note.MediaType = "audio"
		note.FileID = replied.Audio.FileId
		note.Content = getCaption(replied, extraCaption)
	} else if replied.Voice != nil {
		note.MediaType = "voice"
		note.FileID = replied.Voice.FileId
		note.Content = getCaption(replied, extraCaption)
	} else {
		note.MediaType = "text"
		note.Content = getCaption(replied, extraCaption)
	}

	// Upsert (insert or update)
	database.DB.Where(models.Note{ChatID: chat.Id, Name: name}).Assign(models.Note{
		Content:   note.Content,
		FileID:    note.FileID,
		MediaType: note.MediaType,
	}).FirstOrCreate(note)
	database.DB.Save(note)

	_, err := msg.Reply(b, fmt.Sprintf("✅ Saved note <code>%s</code>!", name), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// LIST NOTES
// ─────────────────────────────────────────────────────────────────────────────

func listNotes(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	var noteList []models.Note
	database.DB.Where("chat_id = ?", chat.Id).Select("name").Find(&noteList)

	if len(noteList) == 0 {
		_, err := msg.Reply(b, "📝 No notes saved in this chat.", nil)
		return err
	}

	text := fmt.Sprintf("📝 <b>Notes in %s</b>\n\n", chat.Title)
	for _, n := range noteList {
		text += fmt.Sprintf("• <code>#%s</code>\n", n.Name)
	}
	text += "\n<i>Use #notename to get a note.</i>"

	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// GET NOTE  (#notename trigger)
// ─────────────────────────────────────────────────────────────────────────────

func getNote(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.Text == "" || !strings.HasPrefix(msg.Text, "#") {
		return ext.ContinueGroups
	}

	chat := ctx.EffectiveChat
	if utils.IsPrivateChat(chat) {
		return ext.ContinueGroups
	}

	// Extract note name after #
	name := strings.ToLower(strings.TrimPrefix(strings.Fields(msg.Text)[0], "#"))
	if name == "" {
		return ext.ContinueGroups
	}

	var note models.Note
	result := database.DB.Where(models.Note{ChatID: chat.Id, Name: name}).First(&note)
	if result.Error != nil {
		return ext.ContinueGroups // note not found — silently ignore
	}

	from := msg.From
	return sendNote(b, msg, &note, from)
}

func sendNote(b *gotgbot.Bot, msg *gotgbot.Message, note *models.Note, from *gotgbot.User) error {
	content := applyFillings(note.Content, msg, from)
	opts := &gotgbot.SendMessageOpts{ParseMode: "HTML"}

	switch note.MediaType {
	case "text":
		_, err := msg.Reply(b, content, opts)
		return err

	case "sticker":
		_, err := b.SendSticker(msg.Chat.Id, note.FileID, &gotgbot.SendStickerOpts{
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err

	case "animation":
		_, err := b.SendAnimation(msg.Chat.Id, note.FileID, &gotgbot.SendAnimationOpts{
			Caption:         content,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err

	case "photo":
		_, err := b.SendPhoto(msg.Chat.Id, note.FileID, &gotgbot.SendPhotoOpts{
			Caption:         content,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err

	case "document":
		_, err := b.SendDocument(msg.Chat.Id, note.FileID, &gotgbot.SendDocumentOpts{
			Caption:         content,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err

	case "video":
		_, err := b.SendVideo(msg.Chat.Id, note.FileID, &gotgbot.SendVideoOpts{
			Caption:         content,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err

	case "video_note":
		_, err := b.SendVideoNote(msg.Chat.Id, note.FileID, &gotgbot.SendVideoNoteOpts{
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err

	case "audio":
		_, err := b.SendAudio(msg.Chat.Id, note.FileID, &gotgbot.SendAudioOpts{
			Caption:         content,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err

	case "voice":
		_, err := b.SendVoice(msg.Chat.Id, note.FileID, &gotgbot.SendVoiceOpts{
			Caption:         content,
			ParseMode:       "HTML",
			ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
		})
		return err
	}

	_, err := msg.Reply(b, content, opts)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// DELETE NOTE
// ─────────────────────────────────────────────────────────────────────────────

func deleteNote(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if !utils.IsPrivateChat(chat) {
		if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
			_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
			return err
		}
	}

	name := strings.ToLower(utils.GetCommandArgs(msg))
	if name == "" {
		_, err := msg.Reply(b, "❌ Usage: /delnote [NOTE_NAME]", nil)
		return err
	}

	result := database.DB.Where(models.Note{ChatID: chat.Id, Name: name}).Delete(&models.Note{})
	if result.RowsAffected == 0 {
		_, err := msg.Reply(b, fmt.Sprintf("❌ No note named <code>%s</code>.", name), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}

	_, err := msg.Reply(b, fmt.Sprintf("🗑️ Deleted note <code>%s</code>.", name), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// DELETE ALL NOTES
// ─────────────────────────────────────────────────────────────────────────────

func deleteAllNotes(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if msg.From == nil || !utils.HasPermission(b, chat.Id, msg.From.Id, "can_change_info") {
		_, err := msg.Reply(b, "❌ You need can_change_info permission.", nil)
		return err
	}

	var count int64
	database.DB.Model(&models.Note{}).Where("chat_id = ?", chat.Id).Count(&count)
	if count == 0 {
		_, err := msg.Reply(b, "📝 No notes in this chat.", nil)
		return err
	}

	keyboard := utils.TwoBtn("✅ Yes, delete all", fmt.Sprintf("notedel_yes_%d", chat.Id), "❌ Cancel", "notedel_no")
	_, err := msg.Reply(b, fmt.Sprintf("⚠️ Are you sure you want to delete ALL <b>%d</b> notes in this chat? This cannot be undone!", count),
		&gotgbot.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func deleteAllCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	chat := cq.Message.GetChat()
	from := cq.From

	if !utils.HasPermission(b, chat.Id, from.Id, "can_change_info") {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "❌ You need can_change_info permission.",
			ShowAlert: true,
		})
		return err
	}

	parts := strings.Split(cq.Data, "_")
	if parts[1] == "yes" {
		database.DB.Where("chat_id = ?", chat.Id).Delete(&models.Note{})
		_, _, _ = cq.Message.EditText(b, "✅ All notes deleted.", nil)
	} else {
		_, _, _ = cq.Message.EditText(b, "❌ Cancelled.", nil)
	}
	_, err := cq.Answer(b, nil)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

// applyFillings replaces {first}, {last}, {fullname}, {username}, {mention}, {id}, {chatname}
// Mirrors apply_fillings() in functions.py
func applyFillings(text string, msg *gotgbot.Message, from *gotgbot.User) string {
	if from == nil {
		return text
	}

	first := from.FirstName
	last := from.LastName
	fullname := first
	if last != "" {
		fullname = first + " " + last
	}
	username := from.Username
	mention := utils.MentionHTML(from.Id, first)
	if username == "" {
		username = mention
	} else {
		username = "@" + username
	}
	id := fmt.Sprintf("%d", from.Id)
	chatname := ""
	if msg.Chat.Title != "" {
		chatname = msg.Chat.Title
	}

	r := strings.NewReplacer(
		"{first}", first,
		"{last}", last,
		"{fullname}", fullname,
		"{username}", username,
		"{mention}", mention,
		"{id}", id,
		"{chatname}", chatname,
	)
	return r.Replace(text)
}

func getCaption(msg *gotgbot.Message, override string) string {
	if override != "" {
		return override
	}
	if msg.Caption != "" {
		return msg.Caption
	}
	return ""
}

// getPrivateNote handles /start btnnotesm_<chatID>_<noteName> deep links
// Mirrors: @app.on_message(filters.private & filters.command("start") & filters.regex(r"^/start btnnotesm_"))
