// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved
// Mirrors misskaty/vars.py exactly

package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var (
	APIId       int64
	APIHash     string
	BotToken    string
	DatabaseURI string
	LogChannel  int64

	LogGroupId    int64
	DatabaseName  string
	TZ            string
	Port          string
	SupportChat   string
	OwnerID       int64
	Sudo          []int64
	CommandHandler []string
	AutoRestart   bool

	ForwardFromChatId          []int64
	ForwardToChatId            []int64
	ForwardFilters             []string
	BlockFilesWithoutExtension bool
	BlockedExtensions          []string
	MinimumFileSize            string
)

func Load() {
	if err := godotenv.Load("config.env"); err != nil {
		log.Println("[WARN] config.env not found, reading from environment")
	}

	rawAPIId := mustGet("API_ID")
	id, err := strconv.ParseInt(rawAPIId, 10, 64)
	if err != nil {
		log.Fatal("[ERROR] API_ID must be a valid integer")
	}
	APIId = id
	APIHash   = mustGet("API_HASH")
	BotToken  = mustGet("BOT_TOKEN")
	DatabaseURI = mustGet("DATABASE_URI")

	lc, err := strconv.ParseInt(mustGet("LOG_CHANNEL"), 10, 64)
	if err != nil {
		log.Fatal("[ERROR] LOG_CHANNEL must be a valid integer")
	}
	LogChannel = lc

	DatabaseName = getOr("DATABASE_NAME", "RoboKatyDB")
	TZ           = getOr("TZ", "Asia/Kolkata")
	Port         = getOr("PORT", "8080")
	SupportChat  = getOr("SUPPORT_CHAT", "RoboKaty")
	AutoRestart  = getOr("AUTO_RESTART", "false") == "true"

	if raw := os.Getenv("LOG_GROUP_ID"); raw != "" {
		v, _ := strconv.ParseInt(raw, 10, 64)
		LogGroupId = v
	}

	OwnerID = int64(2024984460)
	if raw := os.Getenv("OWNER_ID"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			OwnerID = v
		}
	}

	sudoRaw := getOr("SUDO", strconv.FormatInt(OwnerID, 10))
	for _, s := range strings.Fields(sudoRaw) {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			Sudo = append(Sudo, v)
		}
	}
	if !containsInt64(Sudo, OwnerID) {
		Sudo = append(Sudo, OwnerID)
	}

	CommandHandler = strings.Fields(getOr("COMMAND_HANDLER", "/ . !"))

	for _, s := range strings.Fields(getOr("FORWARD_FROM_CHAT_ID", "")) {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			ForwardFromChatId = append(ForwardFromChatId, v)
		}
	}
	for _, s := range strings.Fields(getOr("FORWARD_TO_CHAT_ID", "")) {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			ForwardToChatId = append(ForwardToChatId, v)
		}
	}
	ForwardFilters             = strings.Fields(getOr("FORWARD_FILTERS", "video document"))
	BlockFilesWithoutExtension = getOr("BLOCK_FILES_WITHOUT_EXTENSIONS", "true") == "true"
	BlockedExtensions          = strings.Fields(getOr("BLOCKED_EXTENSIONS", "html htm json txt php gif png ink torrent url nfo xml xhtml jpg"))
	MinimumFileSize            = os.Getenv("MINIMUM_FILE_SIZE")
}

func IsSudo(userID int64) bool {
	return containsInt64(Sudo, userID) || userID == OwnerID
}

func mustGet(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[ERROR] Required env var %q is missing!", key)
	}
	return v
}

func getOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func containsInt64(slice []int64, val int64) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
