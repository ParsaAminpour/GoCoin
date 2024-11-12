package main

// Testing database functionality in memory as tmp_db to aviod main database manipulation
import (
	"testing"

	_ "github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func CreateTestMemDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil

}

func TestFetchUser(t *testing.T) {

}
