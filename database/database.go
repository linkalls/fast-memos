package database

import (
	"fmt"
	"log"
	"os"
	"time" // time パッケージをインポート

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
			LogLevel:      logger.Info,            // Log level
			Colorful:      true,                   // Enable color
		},
	)

	// DBファイルパスを環境変数から取得（なければデフォルト）
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "memo_app.db"
	}

	// DBファイルのディレクトリが存在しない場合は作成
	if dbPath != ":memory:" { // SQLiteのメモリDBは除外
		dir := ""
		if idx := len(dbPath) - 1 - len("/"); idx >= 0 {
			for i := len(dbPath) - 1; i >= 0; i-- {
				if dbPath[i] == '/' {
					dir = dbPath[:i]
					break
				}
			}
		}
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				panic("Failed to create DB directory: " + err.Error())
			}
		}
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: dbLogger,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database! path=%s error=%v\n", dbPath, err)
		panic(err)
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
