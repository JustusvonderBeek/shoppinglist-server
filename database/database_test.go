package database_test

import (
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/alexedwards/argon2id"
	"shop.cloudsheeptech.com/database"
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
	database.CheckDatabaseOnline(cfg)
}

func TestPrinting(t *testing.T) {
	connectDatabase()
	database.PrintShoppingListTable()
}

// ------------------------------------------------------------
// Testing user creation and authentication
// ------------------------------------------------------------

func TestInsertUser(t *testing.T) {
	connectDatabase()
	user := data.User{
		ID:       12,
		Username: strconv.Itoa(32),
		Password: "Biene Maja",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	match, err := argon2id.ComparePasswordAndHash(user.Password, createdUser.Password)
	if err != nil {
		log.Printf("Password and hash do not match: %s", err)
		t.FailNow()
	}
	if createdUser.Username != user.Username || !match {
		log.Printf("User not correctly inserted")
		t.FailNow()
	}
	log.Print("InsertUser successfully completed")
	database.PrintUserTable("shoppers")
	database.ResetUserTable()
}

func TestDeletingUser(t *testing.T) {
	connectDatabase()
	user := data.User{
		ID:       0,
		Username: "Delete User Test",
		Password: "Biene Maja",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	match, err := argon2id.ComparePasswordAndHash(user.Password, createdUser.Password)
	if err != nil {
		log.Printf("Password and hash do not match: %s", err)
		t.FailNow()
	}
	if createdUser.Username != user.Username || !match {
		log.Printf("User not correctly inserted")
		t.FailNow()
	}
	database.PrintUserTable("shoppers")
	err = database.DeleteUserAccount(createdUser.ID)
	if err != nil {
		log.Printf("Failed to delete user with id %d from database", createdUser.ID)
		t.FailNow()
	}
	deletedUser, err := database.GetUser(createdUser.ID)
	if err == nil || deletedUser.ID != 0 {
		log.Print("Could retrieve user from database after deleting!")
		t.FailNow()
	}
	log.Print("DeleteUser successfully completed")
	database.PrintUserTable("shoppers")
	database.ResetUserTable()
}

func TestUserLogin(t *testing.T) {
	connectDatabase()
	user := data.User{
		ID:       12,
		Username: "Test User Login",
		Password: "Secure Password 123",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	match, err := argon2id.ComparePasswordAndHash(user.Password, createdUser.Password)
	if err != nil {
		log.Printf("Password and hash do not match: %s", err)
		t.FailNow()
	}
	if createdUser.Username != user.Username || !match {
		log.Printf("User not correctly inserted")
		t.FailNow()
	}
	database.PrintUserTable("shoppers")
	checkLoginUser, err := database.GetUser(createdUser.ID)
	if err != nil {
		log.Printf("Failed to get newly created user for login check: %s", err)
		t.FailNow()
	}
	match, err = argon2id.ComparePasswordAndHash(user.Password, checkLoginUser.Password)
	if err != nil {
		log.Printf("Failed to compare password and hash: %s", err)
		t.FailNow()
	}
	if !match {
		log.Print("Password and hash do not match even though they should!")
		t.FailNow()
	}
	match, _ = argon2id.ComparePasswordAndHash("Secure Password 12", checkLoginUser.Password)
	if match {
		log.Print("Passwords match even though they should not!")
		t.FailNow()
	}
	log.Print("TestLoginUser successfully completed")
	database.ResetUserTable()
}

func TestModifyUsername(t *testing.T) {
	connectDatabase()
	user := data.User{
		ID:       12,
		Username: "Test User modify",
		Password: "Biene Maja",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	checkOldUsername, err := database.GetUser(createdUser.ID)
	if err != nil {
		log.Printf("Failed to get newly created user for modify check: %s", err)
		t.FailNow()
	}
	if checkOldUsername.Username != user.Username {
		log.Print("Usernames do not match before checking!")
		t.FailNow()
	}
	updatedUsername, err := database.ModifyUserAccountName(createdUser.ID, user.Username+" - Updated")
	if err != nil {
		log.Printf("Failed to update username: %s", err)
		t.FailNow()
	}
	database.PrintUserTable("shoppers")
	if updatedUsername.Username == checkOldUsername.Username {
		log.Print("The updated username is still the same!")
		t.FailNow()
	}
	checkNewUsername, err := database.GetUser(createdUser.ID)
	if err != nil {
		log.Printf("Failed to get updated user: %s", err)
		t.FailNow()
	}
	if checkNewUsername.Username != user.Username+" - Updated" {
		log.Print("Usernames do not match!")
		t.FailNow()
	}
	log.Print("TestModifyUsername successfully completed")
	database.ResetUserTable()
}

func TestModifyUserPassword(t *testing.T) {
	connectDatabase()
	user := data.User{
		ID:       12,
		Username: "Test User modify",
		Password: "Old Password",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	checkOldPassword, err := database.GetUser(createdUser.ID)
	if err != nil {
		log.Printf("Failed to get newly created user for modify check: %s", err)
		t.FailNow()
	}
	if checkOldPassword.Password != createdUser.Password {
		log.Print("Password do not match before update!")
		t.FailNow()
	}
	updatedUser, err := database.ModifyUserAccountPassword(createdUser.ID, "New Password")
	if err != nil {
		log.Printf("Failed to update password: %s", err)
		t.FailNow()
	}
	database.PrintUserTable("shoppers")
	match, err := argon2id.ComparePasswordAndHash("New Password", updatedUser.Password)
	if err != nil || !match {
		log.Print("The password was not correctly updated!")
		t.FailNow()
	}
	checkNewPassword, err := database.GetUser(createdUser.ID)
	if err != nil {
		log.Printf("Failed to get updated user: %s", err)
		t.FailNow()
	}
	match, err = argon2id.ComparePasswordAndHash("New Password", checkNewPassword.Password)
	if err != nil || !match {
		log.Print("The password was not correctly updated!")
		t.FailNow()
	}
	log.Print("TestModifyUserPassword successfully completed")
	database.ResetUserTable()
}

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
	err := database.CreateOrUpdateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create new list: %s", err)
		t.FailNow()
	}
	getList, err := database.GetShoppingList(list.ListId, list.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to get newly created shopping list")
		t.FailNow()
	}
	if getList.CreatedBy != list.CreatedBy {
		log.Printf("IDs do not match")
		t.FailNow()
	}
	database.PrintShoppingListTable()
	log.Print("TestCreatingList successfully completed")
	database.ResetShoppingListTable()
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
	err := database.CreateOrUpdateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create new list: %s", err)
		t.FailNow()
	}
	getList, err := database.GetShoppingList(list.ListId, list.CreatedBy.ID)
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
	err = database.CreateOrUpdateShoppingList(updatedList)
	if err != nil {
		log.Printf("Failed to modify shopping list name: %s", err)
		t.FailNow()
	}
	getList, err = database.GetShoppingList(updatedList.ListId, updatedList.CreatedBy.ID)
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
	database.ResetShoppingListTable()
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
	err := database.CreateOrUpdateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create new list: %s", err)
		t.FailNow()
	}
	database.PrintShoppingListTable()
	getList, err := database.GetShoppingList(list.ListId, list.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to get newly created shopping list")
		t.FailNow()
	}
	if getList.ListId != list.ListId {
		log.Printf("IDs do not match")
		t.FailNow()
	}
	err = database.DeleteShoppingList(list.ListId)
	if err != nil {
		log.Printf("Failed to delete shopping list: %s", err)
		t.FailNow()
	}
	getList, err = database.GetShoppingList(list.ListId, list.CreatedBy.ID)
	if err == nil || getList.ListId == list.ListId {
		log.Printf("Can get delete list!")
		t.FailNow()
	}
	database.PrintShoppingListTable()
	log.Print("TestDeletingList successfully completed")
	database.ResetShoppingListTable()
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
	created, err := database.InsertItemToList(mapping)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		t.FailNow()
	}
	if created.ID == 0 {
		log.Print("Mapping not correctly inserted")
		t.FailNow()
	}
	getMapping, err := database.GetItemsInList(mapping.ListId)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 1 {
		log.Printf("The list is longer than expected")
		t.FailNow()
	}
	database.PrintItemPerListTable()
	log.Print("InsertMapping successfully completed")
	database.ResetItemPerListTable()
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
	created, err := database.InsertItemToList(mapping)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		t.FailNow()
	}
	if created.ID == 0 {
		log.Print("Mapping not correctly inserted")
		t.FailNow()
	}
	getMapping, err := database.GetItemsInList(mapping.ListId)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 1 {
		log.Printf("The list is longer than expected")
		t.FailNow()
	}
	database.PrintItemPerListTable()
	err = database.DeleteItemInList(created.ItemId, created.ListId)
	if err != nil {
		log.Printf("Failed to delete mapping")
		t.FailNow()
	}
	getMapping, err = database.GetItemsInList(mapping.ListId)
	if err != nil {
		log.Printf("The mapping or item for the mapping cannot be found")
		t.FailNow()
	}
	if len(getMapping) != 0 {
		log.Printf("The list is longer than expected")
		t.FailNow()
	}
	database.PrintItemPerListTable()
	log.Print("DeleteMapping successfully completed")
	database.ResetItemPerListTable()
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
	_, err := database.InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item for testing")
		t.FailNow()
	}
	items, err := database.GetAllItems()
	if err != nil {
		log.Print("Failed to get all items from database")
		t.FailNow()
	}
	if len(items) != 1 {
		log.Printf("The number of all items (%d) does not match the expected (1)!", len(items))
		database.ResetItemTable()
		t.FailNow()
	}
	log.Printf("All items: %v", items)
	log.Print("GetAllItems successfully completed")
	database.ResetItemTable()
}

func TestGetAllItemsFromName(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ID:   12,
		Name: "New Item A",
		Icon: "Abc",
	}
	_, err := database.InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item for testing")
		t.FailNow()
	}
	database.PrintItemTable()
	items, err := database.GetAllItemsFromName(strings.Split(item.Name, " ")[0])
	if err != nil {
		log.Print("Failed to get items from database")
		t.FailNow()
	}
	if len(items) != 1 {
		log.Printf("The number of all items (%d) does not match the expected (1)!", len(items))
		database.ResetItemTable()
		t.FailNow()
	}
	log.Printf("All items: %v", items)
	items, err = database.GetAllItemsFromName("Not contained")
	if err != nil {
		log.Print("Failed to get items from database")
		t.FailNow()
	}
	if len(items) != 0 {
		log.Printf("The number of all items (%d) does not match the expected (0)!", len(items))
		database.ResetItemTable()
		t.FailNow()
	}
	log.Printf("All items: %v", items)
	// Testing a SQL injection attack
	item.Name = "') > 0; INSERT INTO items (name, icon) VALUES ('abc', 'abc'); --"
	items, err = database.GetAllItemsFromName(item.Name)
	if err == nil {
		log.Print("Executed injection attack!")
		t.FailNow()
	}
	if len(items) != 0 {
		log.Print("Got items for query")
		t.FailNow()
	}
	database.PrintItemTable()
	log.Print("GetAllItems successfully completed")
	database.ResetItemTable()
}

func TestInsertItem(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ID:   12,
		Name: "New Item",
		Icon: "Abc",
	}
	created, err := database.InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item")
		t.FailNow()
	}
	if created == 0 {
		log.Printf("Item ID (%d) not correct", created)
		t.FailNow()
	}
	database.PrintItemTable()
	getItem, err := database.GetItem(created)
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
	database.ResetItemTable()
}

func TestModifyItemName(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ID:   12,
		Name: "Old Item",
		Icon: "Abc",
	}
	created, err := database.InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item")
		t.FailNow()
	}
	getItem, err := database.GetItem(created)
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
	newItem, err := database.ModifyItemName(created, "New Item")
	if err != nil {
		log.Printf("Failed to modify item name: %s", err)
		t.FailNow()
	}
	if newItem.Name != "New Item" {
		log.Print("Name information not correctly stored")
		t.FailNow()
	}
	database.PrintItemTable()
	log.Print("ModifyItemName successfully completed")
	database.ResetItemTable()
}

func TestModifyItemIcon(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ID:   12,
		Name: "Old Item",
		Icon: "Abc",
	}
	created, err := database.InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item")
		t.FailNow()
	}
	getItem, err := database.GetItem(created)
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
	newItem, err := database.ModifyItemIcon(created, "New Icon")
	if err != nil {
		log.Printf("Failed to modify item icon: %s", err)
		t.FailNow()
	}
	if newItem.Icon != "New Icon" {
		log.Print("Icon information not correctly stored")
		t.FailNow()
	}
	database.PrintItemTable()
	log.Print("ModifyItemIcon successfully completed")
	database.ResetItemTable()
}

func TestDeleteItem(t *testing.T) {
	connectDatabase()
	item := data.Item{
		ID:   12,
		Name: "New Item",
		Icon: "Abc",
	}
	created, err := database.InsertItemStruct(item)
	if err != nil {
		log.Printf("Failed to create new item")
		t.FailNow()
	}
	database.PrintItemTable()
	getItem, err := database.GetItem(created)
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
	err = database.DeleteItem(created)
	if err != nil {
		log.Printf("Failed to delete item: %s", err)
		t.FailNow()
	}
	getItem, err = database.GetItem(created)
	if err == nil || getItem.ID != 0 {
		log.Printf("Can still retrieve item!")
		t.FailNow()
	}
	database.PrintItemTable()
	log.Print("DeleteItem successfully completed")
	database.ResetItemTable()
}
