// RoboKaty — modules/loader.go

package modules

import (
	"log"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/modules/admin"
	"github.com/robokatybot/robokaty/modules/afk"
	"github.com/robokatybot/robokaty/modules/anime"
	"github.com/robokatybot/robokaty/modules/auto_forwarder"
	"github.com/robokatybot/robokaty/modules/autoapprove"
	"github.com/robokatybot/robokaty/modules/blacklist"
	"github.com/robokatybot/robokaty/modules/broadcast"
	"github.com/robokatybot/robokaty/modules/copy_forward"
	"github.com/robokatybot/robokaty/modules/dev"
	"github.com/robokatybot/robokaty/modules/federation"
	"github.com/robokatybot/robokaty/modules/inkick"
	"github.com/robokatybot/robokaty/modules/json_plugin"
	"github.com/robokatybot/robokaty/modules/karma"
	"github.com/robokatybot/robokaty/modules/lang_setting"
	"github.com/robokatybot/robokaty/modules/locks"
	"github.com/robokatybot/robokaty/modules/nightmode"
	"github.com/robokatybot/robokaty/modules/notes"
	"github.com/robokatybot/robokaty/modules/ping"
	"github.com/robokatybot/robokaty/modules/quotly"
	"github.com/robokatybot/robokaty/modules/rules"
	"github.com/robokatybot/robokaty/modules/sangmata"
	"github.com/robokatybot/robokaty/modules/sed"
	"github.com/robokatybot/robokaty/modules/start_help"
	"github.com/robokatybot/robokaty/modules/stickers"
	"github.com/robokatybot/robokaty/modules/urban_dict"
	"github.com/robokatybot/robokaty/modules/welcome"
)

// LoadAll registers all modules. b is passed for future use (e.g. webhook setup).
func LoadAll(_ *gotgbot.Bot, dispatcher *ext.Dispatcher) {
	start_help.Load(dispatcher)

	// Group management
	admin.Load(dispatcher)
	locks.Load(dispatcher)
	blacklist.Load(dispatcher)
	inkick.Load(dispatcher)
	autoapprove.Load(dispatcher)
	federation.Load(dispatcher)

	// Content management
	notes.Load(dispatcher)
	rules.Load(dispatcher)
	welcome.Load(dispatcher)
	nightmode.Load(dispatcher)

	// User tools
	afk.Load(dispatcher)
	karma.Load(dispatcher)
	sed.Load(dispatcher)
	lang_setting.Load(dispatcher)

	// Info / search
	ping.Load(dispatcher)
	sangmata.Load(dispatcher)
	stickers.Load(dispatcher)
	anime.Load(dispatcher)
	urban_dict.Load(dispatcher)
	quotly.Load(dispatcher)
	json_plugin.Load(dispatcher)

	// Utility
	copy_forward.Load(dispatcher)
	broadcast.Load(dispatcher)
	auto_forwarder.Load(dispatcher)

	// Developer
	dev.Load(dispatcher)

	registerHelp()
	log.Println("[Loader] ✅ All modules loaded!")
}

func registerHelp() {
	start_help.RegisterHelp("Admin", admin.HELP)
	start_help.RegisterHelp("Notes", notes.HELP)
	start_help.RegisterHelp("Welcome", welcome.HELP)
	start_help.RegisterHelp("Locks", locks.HELP)
	start_help.RegisterHelp("Rules", rules.HELP)
	start_help.RegisterHelp("Blacklist", blacklist.HELP)
	start_help.RegisterHelp("AFK", afk.HELP)
	start_help.RegisterHelp("Karma", karma.HELP)
	start_help.RegisterHelp("Nightmode", nightmode.HELP)
	start_help.RegisterHelp("Ping", ping.HELP)
	start_help.RegisterHelp("Sed", sed.HELP)
	start_help.RegisterHelp("Broadcast", broadcast.HELP)
	start_help.RegisterHelp("Stickers", stickers.HELP)
	start_help.RegisterHelp("Anime", anime.HELP)
	start_help.RegisterHelp("Urban", urban_dict.HELP)
	start_help.RegisterHelp("Quotly", quotly.HELP)
	start_help.RegisterHelp("Sangmata", sangmata.HELP)
	start_help.RegisterHelp("CopyForward", copy_forward.HELP)
	start_help.RegisterHelp("InKick", inkick.HELP)
	start_help.RegisterHelp("AutoApprove", autoapprove.HELP)
	start_help.RegisterHelp("Federation", federation.HELP)
	start_help.RegisterHelp("JSON", json_plugin.HELP)
	start_help.RegisterHelp("LangSetting", lang_setting.HELP)
	start_help.RegisterHelp("AutoForwarder", auto_forwarder.HELP)
	start_help.RegisterHelp("Dev", dev.HELP)
}
