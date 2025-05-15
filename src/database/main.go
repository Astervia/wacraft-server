package database

import (
	"gorm.io/gorm"
)

var DB *gorm.DB

// func Connection() *gorm.DB {
// 	return DB
// }

func init() {
	connect()
}
