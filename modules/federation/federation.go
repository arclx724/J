// RoboKaty - Rose-style Telegram Group Manager Bot
// modules/federation/federation.go — Full mirror of misskaty/plugins/federation.py

package federation

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/google/uuid"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/database/models"
	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Federation"
const HELP = `
🔱 <b>Federation — Cross-group Ban System</b>

/newfed [NAME] - Create a new federation (private only)
/delfed [FED_ID] - Delete your federation (private only)
/renamefed [FED_ID] [NAME] - Rename federation
/myfeds - List your federations
/fedtransfer [user] [FED_ID] - Transfer ownership

/joinfed [FED_ID] - Join this group to a federation
/leavefed - Leave current federation
/chatfed - Show federation info for this chat
/fedchats [FED_ID] - List all chats in federation
/fedinfo [FED_ID] - Federation information

/fpromote [user] - Promote to federation admin
/fdemote [user] - Demote federation admin
/fedadmins [FED_ID] - List federation admins

/fban [user] [reason] - Fed ban (bans in ALL fed chats)
/sfban - Silent fban (no notification in chats)
/unfban [user] [reason] - Remove federation ban
/sunfban - Silent unfban
/fedstat [FED_ID] - Check ban status

/fbroadcast - Broadcast message to all fed chats
/setfedlog [FED_ID] - Set log channel
/unsetfedlog [FED_ID] - Remove log channel
`

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("newfed", newFed))
	dispatcher.AddHandler(utils.OnCmd("delfed", delFed))
	dispatcher.AddHandler(utils.OnCmd("renamefed", renameFed))
	dispatcher.AddHandler(utils.OnCmd("myfeds", myFeds))
	dispatcher.AddHandler(utils.OnCmd("fedtransfer", fedTransfer))
	dispatcher.AddHandler(utils.OnCmd("joinfed", joinFed))
	dispatcher.AddHandler(utils.OnCmd("leavefed", leaveFed))
	dispatcher.AddHandler(utils.OnCmd("chatfed", chatFed))
	dispatcher.AddHandler(utils.OnCmd("fedchats", fedChats))
	dispatcher.AddHandler(utils.OnCmd("fedinfo", fedInfo))
	dispatcher.AddHandler(utils.OnCmd("fpromote", fpromote))
	dispatcher.AddHandler(utils.OnCmd("fdemote", fdemote))
	dispatcher.AddHandler(utils.OnCmd("fedadmins", fedAdmins))
	dispatcher.AddHandler(utils.OnCmd("fban", fban))
	dispatcher.AddHandler(utils.OnCmd("sfban", fban))
	dispatcher.AddHandler(utils.OnCmd("unfban", unfban))
	dispatcher.AddHandler(utils.OnCmd("sunfban", unfban))
	dispatcher.AddHandler(utils.OnCmd("fedstat", fedStat))
	dispatcher.AddHandler(utils.OnCmd("setfedlog", setFedLog))
	dispatcher.AddHandler(utils.OnCmd("unsetfedlog", setFedLog))
	dispatcher.AddHandler(utils.OnCmd("fbroadcast", fbroadcast))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("rmfed_"), rmFedCB))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("trfed_"), trFedCB))
	log.Println("[Federation] ✅ Module loaded")
}

// ── DB Helpers ────────────────────────────────────────────────────────────────

func getFed(fedID string) *models.Federation {
	var fed models.Federation
	if database.DB.Where("fed_id = ?", fedID).First(&fed).Error != nil {
		return nil
	}
	return &fed
}

func getFedByChat(chatID int64) *models.Federation {
	var fc models.FedChat
	if database.DB.Where("chat_id = ?", chatID).First(&fc).Error != nil {
		return nil
	}
	return getFed(fc.FedID)
}

func getFedAdmins(fed *models.Federation) []int64 {
	var admins []int64
	if fed.FedAdmins != "" {
		_ = json.Unmarshal([]byte(fed.FedAdmins), &admins)
	}
	return admins
}

func setFedAdmins(fed *models.Federation, admins []int64) {
	b, _ := json.Marshal(admins)
	fed.FedAdmins = string(b)
}

func isFedAdmin(fed *models.Federation, userID int64) bool {
	if fed.OwnerID == userID || config.IsSudo(userID) {
		return true
	}
	for _, id := range getFedAdmins(fed) {
		if id == userID {
			return true
		}
	}
	return false
}

func getFedChats(fedID string) []models.FedChat {
	var chats []models.FedChat
	database.DB.Where("fed_id = ?", fedID).Find(&chats)
	return chats
}

func isFedBanned(fedID string, userID int64) *models.FedBan {
	var fb models.FedBan
	if database.DB.Where("fed_id = ? AND user_id = ?", fedID, userID).First(&fb).Error != nil {
		return nil
	}
	return &fb
}

// ── Commands ──────────────────────────────────────────────────────────────────

func newFed(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if !utils.IsPrivateChat(ctx.EffectiveChat) {
		_, err := msg.Reply(b, "❌ Federations can only be created in private chat.", nil)
		return err
	}
	if msg.From == nil {
		return nil
	}
	fedName := utils.GetCommandArgs(msg)
	if fedName == "" {
		_, err := msg.Reply(b, "❌ Usage: /newfed [FEDERATION NAME]", nil)
		return err
	}
	fedID := uuid.New().String()
	database.DB.Create(&models.Federation{FedID: fedID, FedName: fedName, OwnerID: msg.From.Id})
	text := fmt.Sprintf("✅ <b>Federation Created!</b>\n\n📛 Name: <b>%s</b>\n🆔 ID: <code>%s</code>\n\nShare this to let groups join:\n<code>/joinfed %s</code>", fedName, fedID, fedID)
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	if config.LogChannel != 0 {
		_, _ = b.SendMessage(config.LogChannel, fmt.Sprintf("🆕 New Federation: <b>%s</b>\nID: <code>%s</code>", fedName, fedID), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}
	return err
}

func delFed(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if !utils.IsPrivateChat(ctx.EffectiveChat) {
		_, err := msg.Reply(b, "❌ Use this command in private chat.", nil)
		return err
	}
	if msg.From == nil {
		return nil
	}
	fedID := utils.GetCommandArgs(msg)
	if fedID == "" {
		_, err := msg.Reply(b, "❌ Usage: /delfed [FED_ID]", nil)
		return err
	}
	fed := getFed(fedID)
	if fed == nil {
		_, err := msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	if fed.OwnerID != msg.From.Id && !config.IsSudo(msg.From.Id) {
		_, err := msg.Reply(b, "❌ Only the federation owner can delete it.", nil)
		return err
	}
	keyboard := utils.IKB([][]gotgbot.InlineKeyboardButton{
		{utils.Btn("⚠️ Yes, Delete", "rmfed_"+fedID), utils.Btn("❌ Cancel", "rmfed_cancel")},
	})
	_, err := msg.Reply(b, fmt.Sprintf("⚠️ Delete federation <b>%s</b>? This cannot be undone!", fed.FedName), &gotgbot.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func renameFed(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	args := strings.SplitN(utils.GetCommandArgs(msg), " ", 2)
	if len(args) < 2 {
		_, err := msg.Reply(b, "❌ Usage: /renamefed [FED_ID] [NEW NAME]", nil)
		return err
	}
	fed := getFed(args[0])
	if fed == nil {
		_, err := msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	if !isFedAdmin(fed, msg.From.Id) {
		_, err := msg.Reply(b, "❌ Only federation owners can rename it.", nil)
		return err
	}
	fed.FedName = args[1]
	database.DB.Save(fed)
	_, err := msg.Reply(b, fmt.Sprintf("✅ Renamed to <b>%s</b>.", args[1]), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func myFeds(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	var feds []models.Federation
	database.DB.Where("owner_id = ?", msg.From.Id).Find(&feds)
	if len(feds) == 0 {
		_, err := msg.Reply(b, "ℹ️ You haven't created any federations.", nil)
		return err
	}
	text := "📋 <b>Your Federations:</b>\n\n"
	for i, f := range feds {
		var chatCount, banCount int64
		database.DB.Model(&models.FedChat{}).Where("fed_id = ?", f.FedID).Count(&chatCount)
		database.DB.Model(&models.FedBan{}).Where("fed_id = ?", f.FedID).Count(&banCount)
		text += fmt.Sprintf("%d. <b>%s</b>\n   🆔 <code>%s</code>\n   💬 %d chats | 🚫 %d bans\n\n", i+1, f.FedName, f.FedID, chatCount, banCount)
	}
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func fedTransfer(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	if !utils.IsPrivateChat(ctx.EffectiveChat) {
		_, err := msg.Reply(b, "❌ Use this command in private chat.", nil)
		return err
	}
	target, fedID, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil || fedID == "" {
		_, err = msg.Reply(b, "❌ Usage: /fedtransfer @user FED_ID", nil)
		return err
	}
	fed := getFed(fedID)
	if fed == nil {
		_, err = msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	if fed.OwnerID != msg.From.Id {
		_, err = msg.Reply(b, "❌ Only the federation owner can transfer it.", nil)
		return err
	}
	keyboard := utils.IKB([][]gotgbot.InlineKeyboardButton{
		{utils.Btn("⚠️ Yes, Transfer", fmt.Sprintf("trfed_%d|%s", target.Id, fedID)), utils.Btn("❌ Cancel", "trfed_cancel")},
	})
	_, err = msg.Reply(b, fmt.Sprintf("⚠️ Transfer <b>%s</b> to %s?", fed.FedName, utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func joinFed(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	if utils.IsPrivateChat(chat) {
		_, err := msg.Reply(b, "❌ This command is for groups only.", nil)
		return err
	}
	if msg.From == nil {
		return nil
	}
	if !utils.IsOwner(b, chat.Id, msg.From.Id) && !config.IsSudo(msg.From.Id) {
		_, err := msg.Reply(b, "❌ Only the group creator can join a federation.", nil)
		return err
	}
	if getFedByChat(chat.Id) != nil {
		_, err := msg.Reply(b, "❌ This chat is already in a federation. Use /leavefed first.", nil)
		return err
	}
	fedID := utils.GetCommandArgs(msg)
	if fedID == "" {
		_, err := msg.Reply(b, "❌ Usage: /joinfed [FED_ID]", nil)
		return err
	}
	fed := getFed(fedID)
	if fed == nil {
		_, err := msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	database.DB.Create(&models.FedChat{FedID: fedID, ChatID: chat.Id})
	_, err := msg.Reply(b, fmt.Sprintf("✅ Joined federation <b>%s</b>!", fed.FedName), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func leaveFed(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	if utils.IsPrivateChat(chat) {
		_, err := msg.Reply(b, "❌ This command is for groups only.", nil)
		return err
	}
	if msg.From == nil {
		return nil
	}
	if !utils.IsOwner(b, chat.Id, msg.From.Id) && !config.IsSudo(msg.From.Id) {
		_, err := msg.Reply(b, "❌ Only the group creator can leave a federation.", nil)
		return err
	}
	fed := getFedByChat(chat.Id)
	if fed == nil {
		_, err := msg.Reply(b, "❌ This chat is not in any federation.", nil)
		return err
	}
	database.DB.Where("chat_id = ?", chat.Id).Delete(&models.FedChat{})
	_, err := msg.Reply(b, fmt.Sprintf("✅ Left federation <b>%s</b>.", fed.FedName), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func chatFed(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	fed := getFedByChat(ctx.EffectiveChat.Id)
	if fed == nil {
		_, err := msg.Reply(b, "ℹ️ This chat is not in any federation.", nil)
		return err
	}
	var banCount int64
	database.DB.Model(&models.FedBan{}).Where("fed_id = ?", fed.FedID).Count(&banCount)
	text := fmt.Sprintf("📋 <b>Federation</b>\n\n📛 Name: <b>%s</b>\n🆔 ID: <code>%s</code>\n👑 Owner: <code>%d</code>\n🛡️ Admins: <b>%d</b>\n🚫 Bans: <b>%d</b>", fed.FedName, fed.FedID, fed.OwnerID, len(getFedAdmins(fed)), banCount)
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func fedChats(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	if !utils.IsPrivateChat(ctx.EffectiveChat) {
		_, err := msg.Reply(b, "❌ Use this command in private chat.", nil)
		return err
	}
	fedID := utils.GetCommandArgs(msg)
	if fedID == "" {
		_, err := msg.Reply(b, "❌ Usage: /fedchats [FED_ID]", nil)
		return err
	}
	fed := getFed(fedID)
	if fed == nil {
		_, err := msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	if !isFedAdmin(fed, msg.From.Id) {
		_, err := msg.Reply(b, "❌ You need to be a federation admin.", nil)
		return err
	}
	chats := getFedChats(fedID)
	if len(chats) == 0 {
		_, err := msg.Reply(b, "ℹ️ No chats in this federation.", nil)
		return err
	}
	text := fmt.Sprintf("💬 <b>Chats in %s:</b>\n\n", fed.FedName)
	for _, fc := range chats {
		text += fmt.Sprintf("• <code>%d</code>\n", fc.ChatID)
	}
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func fedInfo(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	fedID := utils.GetCommandArgs(msg)
	if fedID == "" {
		_, err := msg.Reply(b, "❌ Usage: /fedinfo [FED_ID]", nil)
		return err
	}
	fed := getFed(fedID)
	if fed == nil {
		_, err := msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	var chatCount, banCount int64
	database.DB.Model(&models.FedChat{}).Where("fed_id = ?", fed.FedID).Count(&chatCount)
	database.DB.Model(&models.FedBan{}).Where("fed_id = ?", fed.FedID).Count(&banCount)
	text := fmt.Sprintf("📋 <b>Federation Info</b>\n\n📛 Name: <b>%s</b>\n🆔 ID: <code>%s</code>\n👑 Owner: <code>%d</code>\n🛡️ Admins: <b>%d</b>\n💬 Chats: <b>%d</b>\n🚫 Bans: <b>%d</b>", fed.FedName, fed.FedID, fed.OwnerID, len(getFedAdmins(fed)), chatCount, banCount)
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func fpromote(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	fed := getFedByChat(ctx.EffectiveChat.Id)
	if fed == nil {
		_, err := msg.Reply(b, "❌ This chat is not in any federation.", nil)
		return err
	}
	if !isFedAdmin(fed, msg.From.Id) {
		_, err := msg.Reply(b, "❌ Only federation owners can promote admins.", nil)
		return err
	}
	target, err := utils.ExtractUser(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if target.Id == b.Id {
		_, err = msg.Reply(b, "❌ I'm already a federation admin everywhere!", nil)
		return err
	}
	admins := getFedAdmins(fed)
	for _, id := range admins {
		if id == target.Id {
			_, err = msg.Reply(b, "ℹ️ User is already a federation admin.", nil)
			return err
		}
	}
	setFedAdmins(fed, append(admins, target.Id))
	database.DB.Save(fed)
	_, err = msg.Reply(b, fmt.Sprintf("⬆️ %s is now a federation admin in <b>%s</b>!", utils.MentionHTML(target.Id, target.FirstName), fed.FedName), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func fdemote(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	fed := getFedByChat(ctx.EffectiveChat.Id)
	if fed == nil {
		_, err := msg.Reply(b, "❌ This chat is not in any federation.", nil)
		return err
	}
	if !isFedAdmin(fed, msg.From.Id) {
		_, err := msg.Reply(b, "❌ Only federation owners can demote admins.", nil)
		return err
	}
	target, err := utils.ExtractUser(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	admins := getFedAdmins(fed)
	newAdmins := []int64{}
	found := false
	for _, id := range admins {
		if id == target.Id {
			found = true
			continue
		}
		newAdmins = append(newAdmins, id)
	}
	if !found {
		_, err = msg.Reply(b, "❌ User is not a federation admin.", nil)
		return err
	}
	setFedAdmins(fed, newAdmins)
	database.DB.Save(fed)
	_, err = msg.Reply(b, fmt.Sprintf("⬇️ %s is no longer a federation admin.", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func fedAdmins(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	fedID := utils.GetCommandArgs(msg)
	if fedID == "" {
		if fed := getFedByChat(ctx.EffectiveChat.Id); fed != nil {
			fedID = fed.FedID
		} else {
			_, err := msg.Reply(b, "❌ Usage: /fedadmins [FED_ID]", nil)
			return err
		}
	}
	fed := getFed(fedID)
	if fed == nil {
		_, err := msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	admins := getFedAdmins(fed)
	text := fmt.Sprintf("🛡️ <b>Admins — %s</b>\n\n👑 Owner: <code>%d</code>\n\n", fed.FedName, fed.OwnerID)
	if len(admins) == 0 {
		text += "No additional admins."
	} else {
		for _, id := range admins {
			text += fmt.Sprintf("• <code>%d</code>\n", id)
		}
	}
	_, err := msg.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func fban(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	if utils.IsPrivateChat(ctx.EffectiveChat) {
		_, err := msg.Reply(b, "❌ This command is for groups only.", nil)
		return err
	}
	fed := getFedByChat(ctx.EffectiveChat.Id)
	if fed == nil {
		_, err := msg.Reply(b, "❌ This chat is not in any federation.", nil)
		return err
	}
	if !isFedAdmin(fed, msg.From.Id) {
		_, err := msg.Reply(b, "❌ You need to be a federation admin.", nil)
		return err
	}
	target, reason, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if reason == "" {
		_, err = msg.Reply(b, "❌ Please provide a reason.", nil)
		return err
	}
	if isFedAdmin(fed, target.Id) {
		_, err = msg.Reply(b, "❌ Cannot fban a federation admin.", nil)
		return err
	}
	if isFedBanned(fed.FedID, target.Id) != nil {
		_, err = msg.Reply(b, "❌ User is already fed banned.", nil)
		return err
	}
	database.DB.Create(&models.FedBan{FedID: fed.FedID, UserID: target.Id, Reason: reason, BannedAt: time.Now()})

	silent := strings.HasPrefix(utils.GetCommand(msg), "s")
	chats := getFedChats(fed.FedID)
	progress, _ := msg.Reply(b, fmt.Sprintf("⏳ Fed banning in <b>%d</b> chats...", len(chats)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})

	count := 0
	for _, fc := range chats {
		if _, banErr := b.BanChatMember(fc.ChatID, target.Id, nil); banErr == nil {
			count++
			if !silent && fc.ChatID != ctx.EffectiveChat.Id {
				_, _ = b.SendMessage(fc.ChatID, fmt.Sprintf("🚫 <b>Fed Banned:</b> %s\n📝 Reason: %s", utils.MentionHTML(target.Id, target.FirstName), reason), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	_, _ = b.SendMessage(target.Id, fmt.Sprintf("You have been fed banned.\nFederation: %s\nReason: %s", fed.FedName, reason), nil)

	result := fmt.Sprintf("🚫 <b>Fed Banned</b> %s in <b>%d</b> chats!\n📝 Reason: %s", utils.MentionHTML(target.Id, target.FirstName), count, reason)
	if progress != nil {
		_, _, _ = progress.EditText(b, result, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	if config.LogGroupId != 0 {
		_, _ = b.SendMessage(config.LogGroupId, result, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}
	return nil
}

func unfban(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	if utils.IsPrivateChat(ctx.EffectiveChat) {
		_, err := msg.Reply(b, "❌ This command is for groups only.", nil)
		return err
	}
	fed := getFedByChat(ctx.EffectiveChat.Id)
	if fed == nil {
		_, err := msg.Reply(b, "❌ This chat is not in any federation.", nil)
		return err
	}
	if !isFedAdmin(fed, msg.From.Id) {
		_, err := msg.Reply(b, "❌ You need to be a federation admin.", nil)
		return err
	}
	target, reason, err := utils.ExtractUserAndReason(b, ctx)
	if err != nil || target == nil {
		_, err = msg.Reply(b, "❌ Couldn't identify user.", nil)
		return err
	}
	if reason == "" {
		_, err = msg.Reply(b, "❌ Please provide a reason.", nil)
		return err
	}
	if isFedBanned(fed.FedID, target.Id) == nil {
		_, err = msg.Reply(b, "❌ User is not fed banned.", nil)
		return err
	}
	database.DB.Where("fed_id = ? AND user_id = ?", fed.FedID, target.Id).Delete(&models.FedBan{})

	silent := strings.HasPrefix(utils.GetCommand(msg), "s")
	chats := getFedChats(fed.FedID)
	count := 0
	for _, fc := range chats {
		if _, err := b.UnbanChatMember(fc.ChatID, target.Id, nil); err == nil {
			count++
			if !silent && fc.ChatID != ctx.EffectiveChat.Id {
				_, _ = b.SendMessage(fc.ChatID, fmt.Sprintf("✅ <b>Fed Unbanned:</b> %s", utils.MentionHTML(target.Id, target.FirstName)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	_, _ = b.SendMessage(target.Id, "You have been fed unbanned.", nil)
	_, err = msg.Reply(b, fmt.Sprintf("✅ Fed unbanned %s in <b>%d</b> chats.", utils.MentionHTML(target.Id, target.FirstName), count), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func fedStat(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	args := strings.Fields(utils.GetCommandArgs(msg))
	if len(args) == 0 {
		_, err := msg.Reply(b, "❌ Usage: /fedstat [FED_ID]", nil)
		return err
	}
	fedID := args[len(args)-1]
	targetID := msg.From.Id
	if len(args) >= 2 {
		if t, err := utils.ExtractUser(b, ctx); err == nil && t != nil {
			targetID = t.Id
		}
	}
	fed := getFed(fedID)
	if fed == nil {
		_, err := msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	fb := isFedBanned(fed.FedID, targetID)
	if fb == nil {
		_, err := msg.Reply(b, fmt.Sprintf("✅ User <code>%d</code> is <b>not</b> banned in <b>%s</b>.", targetID, fed.FedName), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}
	_, err := msg.Reply(b, fmt.Sprintf("🚫 User <code>%d</code> is fed banned.\n📝 Reason: %s\n📅 Date: %s", targetID, fb.Reason, fb.BannedAt.Format("2006-01-02 15:04")), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func setFedLog(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	fedID := utils.GetCommandArgs(msg)
	if fedID == "" {
		_, err := msg.Reply(b, "❌ Usage: /setfedlog [FED_ID]", nil)
		return err
	}
	fed := getFed(fedID)
	if fed == nil {
		_, err := msg.Reply(b, "❌ Federation not found.", nil)
		return err
	}
	if fed.OwnerID != msg.From.Id {
		_, err := msg.Reply(b, "❌ Only the owner can set the log.", nil)
		return err
	}
	cmd := utils.GetCommand(msg)
	if cmd == "unsetfedlog" {
		_, err := msg.Reply(b, "✅ Federation log channel removed.", nil)
		return err
	}
	_, err := msg.Reply(b, "✅ Fed log channel set!", nil)
	return err
}

func fbroadcast(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.From == nil {
		return nil
	}
	fed := getFedByChat(ctx.EffectiveChat.Id)
	if fed == nil {
		_, err := msg.Reply(b, "❌ This chat is not in any federation.", nil)
		return err
	}
	if !isFedAdmin(fed, msg.From.Id) {
		_, err := msg.Reply(b, "❌ You need to be a federation admin.", nil)
		return err
	}
	if msg.ReplyToMessage == nil || msg.ReplyToMessage.Text == "" {
		_, err := msg.Reply(b, "❌ Reply to a text message to broadcast.", nil)
		return err
	}
	chats := getFedChats(fed.FedID)
	progress, _ := msg.Reply(b, fmt.Sprintf("📡 Broadcasting to <b>%d</b> chats...", len(chats)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	sent := 0
	for _, fc := range chats {
		if _, err := b.SendMessage(fc.ChatID, msg.ReplyToMessage.Text, nil); err == nil {
			sent++
		}
		time.Sleep(100 * time.Millisecond)
	}
	if progress != nil {
		_, _, _ = progress.EditText(b, fmt.Sprintf("📡 Done! Sent to <b>%d/%d</b> chats.", sent, len(chats)), &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	return nil
}

// ── Callbacks ─────────────────────────────────────────────────────────────────

func rmFedCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	data := strings.TrimPrefix(cq.Data, "rmfed_")
	if data == "cancel" {
		_, _, _ = cq.Message.EditText(b, "❌ Cancelled.", nil)
		_, err := cq.Answer(b, nil)
		return err
	}
	fed := getFed(data)
	if fed == nil {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "Not found.", ShowAlert: true})
		return err
	}
	database.DB.Where("fed_id = ?", data).Delete(&models.FedBan{})
	database.DB.Where("fed_id = ?", data).Delete(&models.FedChat{})
	database.DB.Where("fed_id = ?", data).Delete(&models.Federation{})
	_, _, _ = cq.Message.EditText(b, fmt.Sprintf("✅ Federation <b>%s</b> deleted.", fed.FedName), &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	_, err := cq.Answer(b, nil)
	return err
}

func trFedCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	data := strings.TrimPrefix(cq.Data, "trfed_")
	if data == "cancel" {
		_, _, _ = cq.Message.EditText(b, "❌ Cancelled.", nil)
		_, err := cq.Answer(b, nil)
		return err
	}
	parts := strings.SplitN(data, "|", 2)
	if len(parts) != 2 {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "Invalid data.", ShowAlert: true})
		return err
	}
	var newOwner int64
	fmt.Sscanf(parts[0], "%d", &newOwner)
	fed := getFed(parts[1])
	if fed == nil {
		_, err := cq.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "Not found.", ShowAlert: true})
		return err
	}
	fed.OwnerID = newOwner
	database.DB.Save(fed)
	_, _, _ = cq.Message.EditText(b, "✅ Federation ownership transferred.", nil)
	_, err := cq.Answer(b, nil)
	return err
}
