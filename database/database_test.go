package database

import (
	"log"
	"strings"
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
// Testing data handling
// ------------------------------------------------------------

func TestCreatingList(t *testing.T) {
	connectDatabase()
	creator := data.ListCreator{
		ID:   1337,
		Name: "List Creator",
	}
	list := data.Shoppinglist{
		ListId:     0,
		Name:       "Create List Name",
		CreatedBy:  creator,
		LastEdited: "2024-01-01T12:00:00Z",
		Items:      []data.ItemWire{},
	}
	err := CreateOrUpdateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create new list: %s", err)
		t.FailNow()
	}
	getList, err := GetShoppingList(list.ListId, list.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to get newly created shopping list")
		t.FailNow()
	}
	if getList.CreatedBy != list.CreatedBy {
		log.Printf("IDs do not match")
		t.FailNow()
	}
	PrintShoppingListTable()
	log.Print("TestCreatingList successfully completed")
	ResetShoppingListTable()
}

func TestModifyListName(t *testing.T) {
	connectDatabase()
	creator := data.ListCreator{
		ID:   1337,
		Name: "List Creator",
	}
	list := data.Shoppinglist{
		ListId:     0,
		Name:       "Create List Name",
		CreatedBy:  creator,
		LastEdited: "2024-01-01T12:00:00Z",
		Items:      []data.ItemWire{},
	}
	err := CreateOrUpdateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create new list: %s", err)
		t.FailNow()
	}
	getList, err := GetShoppingList(list.ListId, list.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to get newly created shopping list")
		t.FailNow()
	}
	if getList.ListId != list.ListId {
		log.Printf("IDs do not match")
		t.FailNow()
	}
	updatedList := list
	updatedList.Name = "New List Name"
	err = CreateOrUpdateShoppingList(updatedList)
	if err != nil {
		log.Printf("Failed to modify shopping list name: %s", err)
		t.FailNow()
	}
	getList, err = GetShoppingList(updatedList.ListId, updatedList.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to get list: %s", err)
		t.FailNow()
	}
	if getList.Name == updatedList.Name {
		log.Print("List names still match after update!")
		t.FailNow()
	}
	if getList.Name != "New List Name" {
		log.Printf("Name update not correctly stored")
		t.FailNow()
	}
	log.Print("TestModifyListName successfully completed")
	ResetShoppingListTable()
}

func TestDeletingList(t *testing.T) {
	connectDatabase()
	creator := data.ListCreator{
		ID:   1337,
		Name: "List Creator",
	}
	list := data.Shoppinglist{
		ListId:     0,
		Name:       "Create List Name",
		CreatedBy:  creator,
		LastEdited: "2024-01-01T12:00:00Z",
		Items:      []data.ItemWire{},
	}
	err := CreateOrUpdateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create new list: %s", err)
		t.FailNow()
	}
	PrintShoppingListTable()
	getList, err := GetShoppingList(list.ListId, list.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to get newly created shopping list")
		t.FailNow()
	}
	if getList.ListId != list.ListId {
		log.Printf("IDs do not match")
		t.FailNow()
	}
	err = DeleteShoppingList(list.ListId)
	if err != nil {
		log.Printf("Failed to delete shopping list: %s", err)
		t.FailNow()
	}
	getList, err = GetShoppingList(list.ListId, list.CreatedBy.ID)
	if err == nil || getList.ListId == list.ListId {
		log.Printf("Can get delete list!")
		t.FailNow()
	}
	PrintShoppingListTable()
	log.Print("TestDeletingList successfully completed")
	ResetShoppingListTable()
}

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
// Testing item handling
// ------------------------------------------------------------

func TestGetAllItems(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ID:   12,
		Name: "New Item",
		Icon: "Abc",
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
		ID:   12,
		Name: "New Item A",
		Icon: "Abc",
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
		ID:   12,
		Name: "New Item",
		Icon: "Abc",
	}
	created, err := InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item")
		t.FailNow()
	}
	if created == 0 {
		log.Printf("Item ID (%d) not correct", created)
		t.FailNow()
	}
	PrintItemTable()
	getItem, err := GetItem(created)
	if err != nil {
		log.Printf("Failed to get new item")
		t.FailNow()
	}
	if getItem.ID != item.ID {
		log.Print("Item ID not correct")
		t.FailNow()
	}
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
		ID:   12,
		Name: "Old Item",
		Icon: "Abc",
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
	if getItem.ID != item.ID {
		log.Print("Item ID not correct")
		t.FailNow()
	}
	if getItem.Name != item.Name || getItem.Icon != item.Icon {
		log.Print("Information cannot be retrieved correctly")
		t.FailNow()
	}
	newItem, err := ModifyItemName(created, "New Item")
	if err != nil {
		log.Printf("Failed to modify item name: %s", err)
		t.FailNow()
	}
	if newItem.Name != "New Item" {
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
		ID:   12,
		Name: "Old Item",
		Icon: "Abc",
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
	if getItem.ID != created {
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
		ID:   12,
		Name: "New Item",
		Icon: "Abc",
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
	if getItem.ID != created {
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
	if err == nil || getItem.ID != 0 {
		log.Printf("Can still retrieve item!")
		t.FailNow()
	}
	PrintItemTable()
	log.Print("DeleteItem successfully completed")
	ResetItemTable()
}
