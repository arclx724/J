// * @author        Fake Aaru <arclx724@gmail.com>
// * @date          2026-Mar-22
// * @projectName   RoboKatyBot
// * Copyright ©SlayWithRose All rights reserved

package models

import "time"

type Warn struct {
	ID     uint  `gorm:"primaryKey;autoIncrement"`
	ChatID int64 `gorm:"index;not null"`
	UserID int64 `gorm:"index;not null"`
	Count  int   `gorm:"default:0"`
}

type Note struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	ChatID    int64  `gorm:"index;not null"`
	Name      string `gorm:"index;not null"`
	Content   string `gorm:"type:text"`
	FileID    string
	MediaType string
}


type Welcome struct {
	ChatID         int64  `gorm:"primaryKey"`
	WelcomeEnabled bool   `gorm:"default:true"`
	GoodbyeEnabled bool   `gorm:"default:false"`
	WelcomeText    string `gorm:"type:text"`
	GoodbyeText    string `gorm:"type:text"`
	WelcomeFileID  string
	GoodbyeFileID  string
	CleanWelcome   bool `gorm:"default:false"`
}

type Rules struct {
	ChatID      int64  `gorm:"primaryKey"`
	Rules       string `gorm:"type:text"`
	PrivateMode bool   `gorm:"default:false"`
	BtnName     string `gorm:"default:'Rules'"`
}

type Lock struct {
	ChatID    int64 `gorm:"primaryKey"`
	Sticker   bool  `gorm:"default:false"`
	Audio     bool  `gorm:"default:false"`
	Voice     bool  `gorm:"default:false"`
	Document  bool  `gorm:"default:false"`
	Video     bool  `gorm:"default:false"`
	VideoNote bool  `gorm:"default:false"`
	Contact   bool  `gorm:"default:false"`
	Photo     bool  `gorm:"default:false"`
	Gif       bool  `gorm:"default:false"`
	Url       bool  `gorm:"default:false"`
	Bots      bool  `gorm:"default:false"`
	Forward   bool  `gorm:"default:false"`
	Polls     bool  `gorm:"default:false"`
}

type Blacklist struct {
	ID      uint   `gorm:"primaryKey;autoIncrement"`
	ChatID  int64  `gorm:"index;not null"`
	Trigger string `gorm:"index;not null"`
	Action  string `gorm:"default:'warn'"`
}

type Federation struct {
	FedID     string    `gorm:"primaryKey"`
	FedName   string    `gorm:"not null"`
	OwnerID   int64     `gorm:"not null;index"`
	FedAdmins string    `gorm:"type:text"`
	CreatedAt time.Time
}

type FedBan struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`
	FedID    string    `gorm:"index;not null"`
	UserID   int64     `gorm:"index;not null"`
	Reason   string    `gorm:"type:text"`
	BannedAt time.Time `gorm:"autoCreateTime"`
}

type FedChat struct {
	ID     uint   `gorm:"primaryKey;autoIncrement"`
	FedID  string `gorm:"index;not null"`
	ChatID int64  `gorm:"index;not null"`
}

type Afk struct {
	UserID  int64     `gorm:"primaryKey"`
	IsAfk   bool      `gorm:"default:false"`
	Reason  string    `gorm:"type:text"`
	AfkTime time.Time
}

type Karma struct {
	ID     uint  `gorm:"primaryKey;autoIncrement"`
	ChatID int64 `gorm:"index;not null"`
	UserID int64 `gorm:"index;not null"`
	Karma  int   `gorm:"default:0"`
}

type NightMode struct {
	ChatID    int64  `gorm:"primaryKey"`
	Enabled   bool   `gorm:"default:false"`
	StartTime string `gorm:"default:'22:00'"`
	EndTime   string `gorm:"default:'06:00'"`
}

type UserChat struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	UserID   int64  `gorm:"index;not null"`
	Username string
	ChatID   int64 `gorm:"index"`
}

type Approval struct {
	ID     uint  `gorm:"primaryKey;autoIncrement"`
	ChatID int64 `gorm:"index;not null"`
	UserID int64 `gorm:"index;not null"`
}
