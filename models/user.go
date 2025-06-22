package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Username  string         `gorm:"uniqueIndex;not null"`
	Password  string         `gorm:"not null"`
	Memos     []Memo         // ユーザーが所有するメモ (リレーション)
}
