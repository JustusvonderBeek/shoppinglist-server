package database

import (
	"log"
	"testing"

	"shop.cloudsheeptech.com/server/configuration"
	"shop.cloudsheeptech.com/server/data"
)

// ------------------------------------------------------------
// Connect the test to the database: required
// ------------------------------------------------------------

func connectDatabase() {
	cfg := configuration.Config{
		DatabaseConfig: "../resources/db.json",
	}
	CheckDatabaseOnline(cfg)
}

func TestPrinting(t *testing.T) {
	connectDatabase()
	PrintShoppingListTable()
}

// ------------------------------------------------------------
// Testing the user in : user_test
// ------------------------------------------------------------

// ------------------------------------------------------------
// Testing the list handling: list_test
// ------------------------------------------------------------

// TODO: Extracting useful information for application
func TestInsertMapping(t *testing.T) {
	connectDatabase()
	mapping := data.ItemPerList{
		ID:       12,
		ListId:   0,
		ItemId:   1,
		Quantity: 1,
		Checked:  false,
		AddedBy:  1234,
	}
	created, err := InsertItemToList(mapping)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		t.FailNow()
	}
	if created.ID == 0 {
		log.Print("Mapping not correctly inserted")
		t.FailNow()
	}
	getMapping, err := GetItemsInList(mapping.ListId)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 1 {
		log.Printf("The list is longer than expected")
		t.FailNow()
	}
	PrintItemPerListTable()
	log.Print("InsertMapping successfully completed")
	ResetItemPerListTable()
}

func TestDeleteMapping(t *testing.T) {
	connectDatabase()
	mapping := data.ItemPerList{
		ID:       12,
		ListId:   0,
		ItemId:   1,
		Quantity: 1,
		Checked:  false,
		AddedBy:  1234,
	}
	created, err := InsertItemToList(mapping)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		t.FailNow()
	}
	if created.ID == 0 {
		log.Print("Mapping not correctly inserted")
		t.FailNow()
	}
	getMapping, err := GetItemsInList(mapping.ListId)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 1 {
		log.Printf("The list is longer than expected")
		t.FailNow()
	}
	PrintItemPerListTable()
	err = DeleteItemInList(created.ItemId, created.ListId)
	if err != nil {
		log.Printf("Failed to delete mapping")
		t.FailNow()
	}
	getMapping, err = GetItemsInList(mapping.ListId)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 0 {
		log.Printf("The list is longer than expected")
		t.FailNow()
	}
	PrintItemPerListTable()
	log.Print("DeleteMapping successfully completed")
	ResetItemPerListTable()
}

// ------------------------------------------------------------
// Testing items in: item_test
// ------------------------------------------------------------
