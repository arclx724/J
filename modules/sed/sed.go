// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Mirrors: misskaty/plugins/sed.py

package sed

import (
	"fmt"
	"html"
	"log"
	"regexp"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

const MODULE = "Sed"
const HELP = `
sed replacement — Reply to a message with:
  s/old/new/      Replace first occurrence
  s/old/new/g     Replace all occurrences
  s/old/new/i     Case-insensitive
  s/old/new/gi    All + case-insensitive

Example: Reply with "s/hello/world/" to fix someone's typo.
`

var sedPattern = regexp.MustCompile(`^s/`)

func Load(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(handlers.NewMessage(
		func(msg *gotgbot.Message) bool {
			return msg.ReplyToMessage != nil && msg.Text != "" && sedPattern.MatchString(msg.Text)
		},
		sedReplace,
	))
	log.Println("[Sed] ✅ Module loaded")
}

func sedReplace(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg.ReplyToMessage == nil {
		return nil
	}

	parts := splitSed(msg.Text)
	if len(parts) < 3 {
		return nil
	}

	pattern := parts[1]
	replacement := strings.ReplaceAll(parts[2], `\/`, "/")
	flags := ""
	if len(parts) > 3 {
		flags = parts[3]
	}

	text := msg.ReplyToMessage.Text
	if text == "" {
		text = msg.ReplyToMessage.Caption
	}
	if text == "" {
		_, err := msg.Reply(b, "❌ Replied message has no text.", nil)
		return err
	}

	// Build regex
	regexPrefix := ""
	if strings.Contains(flags, "i") {
		regexPrefix += "(?i)"
	}
	if strings.Contains(flags, "s") {
		regexPrefix += "(?s)"
	}

	re, err := regexp.Compile(regexPrefix + pattern)
	if err != nil {
		_, err = msg.Reply(b, fmt.Sprintf("❌ Invalid regex: %s", err.Error()), nil)
		return err
	}

	var result string
	if strings.Contains(flags, "g") {
		result = re.ReplaceAllString(text, replacement)
	} else {
		// Replace only first occurrence
		found := false
		result = re.ReplaceAllStringFunc(text, func(match string) string {
			if !found {
				found = true
				return re.ReplaceAllString(match, replacement)
			}
			return match
		})
	}

	if result == text {
		_, err = msg.Reply(b, "ℹ️ No match found.", nil)
		return err
	}

	_, err = msg.Reply(b,
		fmt.Sprintf("<pre>%s</pre>", html.EscapeString(result)),
		&gotgbot.SendMessageOpts{ParseMode: "HTML"},
	)
	return err
}

// splitSed splits "s/pattern/replacement/flags" respecting escaped slashes
func splitSed(text string) []string {
	if len(text) < 2 {
		return nil
	}
	result := []string{"s"}
	current := strings.Builder{}
	i := 2 // skip "s/"
	for i < len(text) {
		if text[i] == '/' && (i == 2 || text[i-1] != '\\') {
			result = append(result, current.String())
			current.Reset()
		} else {
			current.WriteByte(text[i])
		}
		i++
	}
	result = append(result, current.String())
	return result
}
