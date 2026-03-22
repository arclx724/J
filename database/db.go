// RoboKaty — database/db.go

package database

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/robokatybot/robokaty/config"
	"github.com/robokatybot/robokaty/database/models"
)

var DB *gorm.DB

func Init() {
	var err error
	DB, err = gorm.Open(postgres.Open(config.DatabaseURI), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("[ERROR] Database connection failed: %v", err)
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	log.Println("[DB] ✅ Connected!")
	autoMigrate()
}

func autoMigrate() {
	err := DB.AutoMigrate(
		&models.Warn{},
		&models.Note{},
		&models.Welcome{},
		&models.Rules{},
		&models.Lock{},
		&models.Blacklist{},
		&models.Federation{},
		&models.FedBan{},
		&models.FedChat{},
		&models.Afk{},
		&models.Karma{},
		&models.NightMode{},
		&models.UserChat{},
		&models.Approval{},
	)
	if err != nil {
		log.Fatalf("[ERROR] AutoMigrate failed: %v", err)
	}
	log.Println("[DB] ✅ Migrated!")
}
