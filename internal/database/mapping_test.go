package database

import (
	"log"
	"testing"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
)

// ------------------------------------------------------------
// Testing the list handling: list_test
// ------------------------------------------------------------

func createDefaultMapping() data.ListItem {
	return data.ListItem{
		ID:        12,
		ListId:    1,
		ItemId:    1,
		Quantity:  1,
		Checked:   false,
		CreatedBy: 1234,
		AddedBy:   1234,
	}
}

func TestInsertMapping(t *testing.T) {
	connectDatabase()
	mapping := createDefaultMapping()
	created, err := InsertOrUpdateItemInList(mapping)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		t.FailNow()
	}
	if created.ID == 0 {
		log.Print("Mapping not correctly inserted")
		t.FailNow()
	}
	getMapping, err := GetItemsInList(mapping.ListId, 0)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 1 {
		log.Printf("The list is longer (%d) than expected", len(getMapping))
		t.FailNow()
	}
	onlyMapping := getMapping[0]
	if onlyMapping.ItemId != mapping.ItemId || onlyMapping.Quantity != 1 || onlyMapping.Checked != mapping.Checked || onlyMapping.CreatedBy != mapping.CreatedBy || onlyMapping.AddedBy != mapping.AddedBy {
		log.Printf("Wrongly inserted. Attributes do not match")
		t.FailNow()
	}
	PrintItemPerListTable()
	log.Print("InsertMapping successfully completed")
	ResetItemPerListTable()
}

func TestInsertDoubleMapping(t *testing.T) {
	connectDatabase()
	mapping := createDefaultMapping()
	for i := 0; i < 3; i++ {
		created, err := InsertOrUpdateItemInList(mapping)
		if err != nil {
			log.Printf("Failed to insert mapping into database: %s", err)
			t.FailNow()
		}
		if created.ID == 0 {
			log.Print("Mapping not correctly inserted")
			t.FailNow()
		}
		getMapping, err := GetItemsInList(mapping.ListId, created.ID)
		if err != nil {
			log.Printf("The mapping or item for the mapping cannot be found")
			t.FailNow()
		}
		if len(getMapping) != 1 {
			log.Printf("The list is longer (%d) than expected", len(getMapping))
			t.FailNow()
		}
		onlyMapping := getMapping[0]
		if onlyMapping.ItemId != mapping.ItemId || onlyMapping.Quantity != 1 || onlyMapping.Checked != mapping.Checked || onlyMapping.CreatedBy != mapping.CreatedBy || onlyMapping.AddedBy != mapping.AddedBy {
			log.Printf("Wrongly inserted. Attributes do not match")
			t.FailNow()
		}
	}
	allMappings, err := GetItemsInList(mapping.ListId, mapping.CreatedBy)
	if err != nil {
		log.Printf("Failed to get items but there should be 1: %s", err)
		t.FailNow()
	}
	if len(allMappings) != 1 {
		log.Printf("Found more than a single mapping which is incorrect!")
		t.FailNow()
	}
	PrintItemPerListTable()
	log.Print("InsertMapping successfully completed")
	ResetItemPerListTable()
}

func TestUpdatingMapping(t *testing.T) {
	connectDatabase()
	mapping := createDefaultMapping()
	created, err := InsertOrUpdateItemInList(mapping)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		t.FailNow()
	}
	if created.ID == 0 {
		log.Print("Mapping not correctly inserted")
		t.FailNow()
	}
	getMapping, err := GetItemsInList(mapping.ListId, mapping.CreatedBy)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 1 {
		log.Printf("The list is longer (%d) than expected", len(getMapping))
		t.FailNow()
	}
	onlyMapping := getMapping[0]
	if onlyMapping.ItemId != mapping.ItemId || onlyMapping.Quantity != mapping.Quantity || onlyMapping.Checked != mapping.Checked || onlyMapping.CreatedBy != mapping.CreatedBy || onlyMapping.AddedBy != mapping.AddedBy {
		log.Printf("Wrongly inserted. Attributes do not match")
		t.FailNow()
	}
	// Update the mapping
	mapping.Checked = !mapping.Checked
	mapping.Quantity = mapping.Quantity + 1
	mapping.AddedBy = 12345
	_, err = InsertOrUpdateItemInList(mapping)
	if err != nil {
		log.Printf("Failed to update mapping into database: %s", err)
		t.FailNow()
	}
	updatedMapping, err := GetItemsInList(mapping.ListId, mapping.CreatedBy)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(updatedMapping) != 1 {
		log.Printf("The list is longer (%d) than expected", len(updatedMapping))
		t.FailNow()
	}
	onlyMapping = updatedMapping[0]
	if onlyMapping.ItemId != mapping.ItemId || onlyMapping.Quantity != mapping.Quantity || onlyMapping.Checked != mapping.Checked || onlyMapping.CreatedBy != mapping.CreatedBy || onlyMapping.AddedBy != mapping.AddedBy {
		log.Printf("Wrongly updated. Attributes do not match")
		t.FailNow()
	}
	PrintItemPerListTable()
	log.Print("InsertMapping successfully completed")
	ResetItemPerListTable()
}

func TestDeleteMapping(t *testing.T) {
	connectDatabase()
	mapping := createDefaultMapping()
	created, err := InsertOrUpdateItemInList(mapping)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		t.FailNow()
	}
	if created.ID == 0 {
		log.Print("Mapping not correctly inserted")
		t.FailNow()
	}
	getMapping, err := GetItemsInList(mapping.ListId, mapping.CreatedBy)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 1 {
		log.Printf("The list is longer than expected")
		t.FailNow()
	}
	PrintItemPerListTable()
	err = DeleteItemInList(created.ItemId, created.ListId, created.CreatedBy)
	if err != nil {
		log.Printf("Failed to delete mapping")
		t.FailNow()
	}
	getMapping, err = GetItemsInList(mapping.ListId, mapping.CreatedBy)
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
