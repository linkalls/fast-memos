package models

import "gorm.io/gorm"

type Memo struct {
	gorm.Model
	Title   string `gorm:"not null"`
	Content string
	UserID  uint   // このメモを所有するユーザーのID
}
