package db

import (
	"fmt"
	"sync/atomic"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDBCounter int64

func CreateTestDB() *gorm.DB {
	// Use a unique database name for each test to avoid concurrency issues
	counter := atomic.AddInt64(&testDBCounter, 1)
	dbName := fmt.Sprintf("file:test_%d.db?mode=memory&cache=shared", counter)

	db, err := gorm.Open(sqlite.Open(dbName))
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&User{}, &Task{}, &TaskComment{}, &WebhookSubscription{}, &Repository{})

	return db
}
