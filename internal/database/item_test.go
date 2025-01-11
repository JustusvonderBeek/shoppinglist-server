package database

import (
	"log"
	"strings"
	"testing"

	"github.com/justusvonderbeek/shopping-list-server/internal/data"
)

// ------------------------------------------------------------
// Testing item handling
// ------------------------------------------------------------

func TestGetAllItems(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ItemId: 12,
		Name:   "New Item",
		Icon:   "Abc",
	}
	_, err := InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item for testing")
		t.FailNow()
	}
	items, err := GetAllItems()
	if err != nil {
		log.Print("Failed to get all items from database")
		t.FailNow()
	}
	if len(items) != 1 {
		log.Printf("The number of all items (%d) does not match the expected (1)!", len(items))
		ResetItemTable()
		t.FailNow()
	}
	log.Printf("All items: %v", items)
	log.Print("GetAllItems successfully completed")
	ResetItemTable()
}

func TestGetAllItemsFromName(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ItemId: 12,
		Name:   "New Item A",
		Icon:   "Abc",
	}
	_, err := InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item for testing")
		t.FailNow()
	}
	PrintItemTable()
	items, err := GetAllItemsFromName(strings.Split(item.Name, " ")[0])
	if err != nil {
		log.Print("Failed to get items from database")
		t.FailNow()
	}
	if len(items) != 1 {
		log.Printf("The number of all items (%d) does not match the expected (1)!", len(items))
		ResetItemTable()
		t.FailNow()
	}
	log.Printf("All items: %v", items)
	items, err = GetAllItemsFromName("Not contained")
	if err != nil {
		log.Print("Failed to get items from database")
		t.FailNow()
	}
	if len(items) != 0 {
		log.Printf("The number of all items (%d) does not match the expected (0)!", len(items))
		ResetItemTable()
		t.FailNow()
	}
	log.Printf("All items: %v", items)
	// Testing a SQL injection attack
	item.Name = "') > 0; INSERT INTO items (name, icon) VALUES ('abc', 'abc'); --"
	items, err = GetAllItemsFromName(item.Name)
	if err == nil {
		log.Print("Executed injection attack!")
		t.FailNow()
	}
	if len(items) != 0 {
		log.Print("Got items for query")
		t.FailNow()
	}
	PrintItemTable()
	log.Print("GetAllItems successfully completed")
	ResetItemTable()
}

func TestInsertItem(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ItemId: 12,
		Name:   "New Item",
		Icon:   "Abc",
	}
	created, err := InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item: %s", err)
		t.FailNow()
	}
	if created == 0 {
		log.Printf("Item ID (%d) not correct but zero", created)
		t.FailNow()
	}
	PrintItemTable()
	getItem, err := GetItem(created)
	if err != nil {
		log.Printf("Failed to get new item")
		t.FailNow()
	}
	// Item IDs must not match because we can reuse the existing items
	// if getItem.ID != item.ID {
	// 	log.Print("Item ID not correct")
	// 	t.FailNow()
	// }
	if getItem.Name != item.Name || getItem.Icon != item.Icon {
		log.Print("Information cannot be retrieved correctly")
		t.FailNow()
	}
	log.Print("InsertItem successfully completed")
	ResetItemTable()
}

func TestModifyItemName(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ItemId: 12,
		Name:   "Old Item",
		Icon:   "Abc",
	}
	created, err := InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item")
		t.FailNow()
	}
	getItem, err := GetItem(created)
	if err != nil {
		log.Printf("Failed to get new item")
		t.FailNow()
	}
	// Item IDs must not match because we can reuse the existing items
	// if getItem.ID != item.ID {
	// 	log.Print("Item ID not correct")
	// 	t.FailNow()
	// }
	if getItem.Name != item.Name || getItem.Icon != item.Icon {
		log.Print("Information cannot be retrieved correctly")
		t.FailNow()
	}
	updatedName := "New Item"
	newItem, err := ModifyItemName(created, updatedName)
	if err != nil {
		log.Printf("Failed to modify item name: %s", err)
		t.FailNow()
	}
	if newItem.Name != updatedName {
		log.Print("Name information not correctly stored")
		t.FailNow()
	}
	PrintItemTable()
	log.Print("ModifyItemName successfully completed")
	ResetItemTable()
}

func TestModifyItemIcon(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ItemId: 12,
		Name:   "Old Item",
		Icon:   "Abc",
	}
	created, err := InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item")
		t.FailNow()
	}
	getItem, err := GetItem(created)
	if err != nil {
		log.Printf("Failed to get new item")
		t.FailNow()
	}
	if getItem.ItemId != created {
		log.Print("Item ID not correct")
		t.FailNow()
	}
	if getItem.Name != item.Name || getItem.Icon != item.Icon {
		log.Print("Information cannot be retrieved correctly")
		t.FailNow()
	}
	newItem, err := ModifyItemIcon(created, "New Icon")
	if err != nil {
		log.Printf("Failed to modify item icon: %s", err)
		t.FailNow()
	}
	if newItem.Icon != "New Icon" {
		log.Print("Icon information not correctly stored")
		t.FailNow()
	}
	PrintItemTable()
	log.Print("ModifyItemIcon successfully completed")
	ResetItemTable()
}

func TestDeleteItem(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ItemId: 12,
		Name:   "New Item",
		Icon:   "Abc",
	}
	created, err := InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item")
		t.FailNow()
	}
	PrintItemTable()
	getItem, err := GetItem(created)
	if err != nil {
		log.Printf("Failed to get new item")
		t.FailNow()
	}
	if getItem.ItemId != created {
		log.Print("Item ID not correct")
		t.FailNow()
	}
	if getItem.Name != item.Name || getItem.Icon != item.Icon {
		log.Print("Information cannot be retrieved correctly")
		t.FailNow()
	}
	err = DeleteItem(created)
	if err != nil {
		log.Printf("Failed to delete item: %s", err)
		t.FailNow()
	}
	getItem, err = GetItem(created)
	if err == nil || getItem.ItemId != 0 {
		log.Printf("Can still retrieve item!")
		t.FailNow()
	}
	PrintItemTable()
	log.Print("DeleteItem successfully completed")
	ResetItemTable()
}
