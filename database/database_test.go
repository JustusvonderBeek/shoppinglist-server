package database_test

import (
	"log"
	"testing"

	"shop.cloudsheeptech.com/configuration"
	"shop.cloudsheeptech.com/database"
)

func connectDatabase() {
	cfg := configuration.Config{
		DatabaseConfig: "../db.json",
	}
	database.CheckDatabaseOnline(cfg)
}

func TestGetAllItems(t *testing.T) {
	connectDatabase()
	items, err := database.GetAllItems()
	if err != nil {
		log.Print("Failed to get all items from database")
		t.FailNow()
	}
	log.Printf("All items: %v", items)
}

func TestInsertItem(t *testing.T) {
	connectDatabase()
	item := database.Item{
		ID:    12,
		Name:  "New Item",
		Image: "Abc",
	}
	id, err := database.InsertItem(item)
	if err != nil {
		log.Printf("Failed to insert item into database: %s", err)
		t.FailNow()
	}
	if id < 0 {
		log.Printf("Item not correctly inserted: %s", err)
		t.FailNow()
	}
	log.Print("InsertItem successfully completed")
}

func TestInsertUser(t *testing.T) {
	connectDatabase()
	user := database.User{
		ID:        4,
		Name:      "New Item",
		FavRecipe: 3,
	}
	id, err := database.InsertUser(user)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	if id < 0 {
		log.Printf("User not correctly inserted: %s", err)
		t.FailNow()
	}
	log.Print("InsertUser successfully completed")
}

func TestInsertMapping(t *testing.T) {
	connectDatabase()
	mapping := database.Mapping{
		ID:       12,
		ListId:   0,
		ItemId:   1,
		Quantity: 1,
	}
	id, err := database.InsertMapping(mapping)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		t.FailNow()
	}
	if id < 0 {
		log.Printf("Mapping not correctly inserted: %s", err)
		t.FailNow()
	}
	log.Print("InsertMapping successfully completed")
}
