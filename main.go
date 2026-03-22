// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved

package main

import (
	"log"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/database"
	"github.com/robokatybot/robokaty/modules"
	"github.com/robokatybot/robokaty/modules/dev"
	"github.com/robokatybot/robokaty/modules/ping"
)

func main() {
	config.Load()
	database.Init()

	b, err := gotgbot.NewBot(config.BotToken, &gotgbot.BotOpts{
		RequestOpts: &gotgbot.RequestOpts{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to create bot: %v", err)
	}
	log.Printf("[INFO] ✅ Bot: @%s (ID: %d)", b.Username, b.Id)

	startTime := time.Now()
	ping.BotStartTime = startTime
	dev.BotStartTime = startTime

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Printf("[ERROR] %v", err)
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	modules.LoadAll(b, dispatcher)

	updater := ext.NewUpdater(dispatcher, nil)
	err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			AllowedUpdates: []string{
				"message", "edited_message", "callback_query",
				"chat_member", "my_chat_member", "chat_join_request",
			},
			RequestOpts: &gotgbot.RequestOpts{Timeout: 10 * time.Second},
		},
	})
	if err != nil {
		log.Fatalf("[FATAL] Polling failed: %v", err)
	}

	log.Printf("[INFO] 🚀 RoboKaty running! Prefixes: %v | Support: @%s", config.CommandHandler, config.SupportChat)

	go func() {
		time.Sleep(2 * time.Second)
		_, _ = b.SendMessage(config.LogChannel, "🟢 <b>RoboKaty started!</b>\n🤖 @"+b.Username, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	}()

	updater.Idle()
}
