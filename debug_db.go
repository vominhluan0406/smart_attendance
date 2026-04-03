package main

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("data/smart_attendance.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables)
	fmt.Println("Tables:", tables)

	var results []map[string]interface{}
	db.Table("branches").Find(&results)
	fmt.Println("Branches count:", len(results))
	for _, r := range results {
		fmt.Printf("Branch: %v | Methods: %v\n", r["name"], r["allowed_methods"])
	}
}
