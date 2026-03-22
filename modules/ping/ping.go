// RoboKaty — modules/ping/ping.go
// Mirrors: misskaty/plugins/ping.py — ALL commands including ping_dc

package ping

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/robokatybot/robokaty/utils"
)

const MODULE = "Ping"
const HELP = `
/ping - Check bot response time and uptime
/ping_dc - Ping all Telegram datacenters
`

const BotVersion = "v1.0.0"
var BotStartTime = time.Now()

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(utils.OnCmd("ping", ping))
	dispatcher.AddHandler(utils.OnCmd("ping_dc", pingDC))
	log.Println("[Ping] ✅ Module loaded")
}

func ping(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage

	start := time.Now()
	sent, err := msg.Reply(b, "🐱 Pong!...", nil)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)
	uptime := utils.FormatDuration(time.Since(BotStartTime))

	text := fmt.Sprintf(
		"<b>🐈 RoboKaty %s</b>\n\n"+
			"⚡ <b>Ping:</b> <code>%dms</code>\n"+
			"⏱️ <b>Uptime:</b> <code>%s</code>\n"+
			"🔧 <b>Go Version:</b> <code>%s</code>",
		BotVersion,
		elapsed.Milliseconds(),
		uptime,
		runtime.Version(),
	)
	_, _, err = sent.EditText(b, text, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	return err
}

func pingDC(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage

	sent, err := msg.Reply(b, "📡 Pinging Telegram datacenters...", nil)
	if err != nil {
		return err
	}

	dcs := map[string]string{
		"DC1": "149.154.175.53",
		"DC2": "149.154.167.51",
		"DC3": "149.154.175.100",
		"DC4": "149.154.167.91",
		"DC5": "91.108.56.130",
	}

	timeRe := regexp.MustCompile(`time=(.+?ms?)`)
	text := "🌐 <b>Telegram Datacenter Pings:</b>\n\n"

	for dc, ip := range dcs {
		out, pingErr := exec.Command("ping", "-c", "1", "-W", "2", ip).CombinedOutput()
		if pingErr != nil {
			text += fmt.Sprintf("  <b>%s:</b> ❌ Timeout\n", dc)
			continue
		}
		matches := timeRe.FindSubmatch(out)
		if len(matches) >= 2 {
			text += fmt.Sprintf("  <b>%s:</b> <code>%s</code> ✅\n", dc, string(matches[1]))
		} else {
			text += fmt.Sprintf("  <b>%s:</b> ✅\n", dc)
		}
	}

	if sent != nil {
		_, _, err = sent.EditText(b, text, &gotgbot.EditMessageTextOpts{ParseMode: "HTML"})
	}
	return err
}
