package models

import (
	"time"

	"gorm.io/gorm"
)

type Memo struct {
	ID                  string `gorm:"primaryKey"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt `gorm:"index"`
	Title               string         `gorm:"not null"`
	Content             string
	UserID              string `gorm:"index"` // UserIDをstringに変更
	RelatedMemoIDs      []string `gorm:"-"` // DBには保存しない
	RelatedMemoIDsStore string `gorm:"type:text;column:related_memo_ids"` // DB保存用
}
