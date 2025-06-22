package database

import (
	"fmt"
	"log"
	"time" // time パッケージをインポート
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/linkalls/fast-memos/models" // モデルのパスを修正
)

var DB *gorm.DB

func ConnectDatabase() {
	dbLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: 200 * time.Millisecond, // Slow SQL threshold を time.Duration に変更
			LogLevel:      logger.Info, // Log level
			Colorful:      true,        // Enable color
		},
	)

	var err error
	DB, err = gorm.Open(sqlite.Open("memo_app.db"), &gorm.Config{
		Logger: dbLogger,
	})

	if err != nil {
		panic("Failed to connect to database!")
	}

	fmt.Println("Database connection successfully opened")

	// AutoMigrate a User and Memo table
	err = DB.AutoMigrate(&models.User{}, &models.Memo{})
	if err != nil {
		fmt.Println("Failed to migrate database")
		panic(err)
	}

	fmt.Println("Database Migrated")
}
