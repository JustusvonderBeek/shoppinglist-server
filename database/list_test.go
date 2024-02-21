package database

import (
	"log"
	"testing"
	"time"

	"shop.cloudsheeptech.com/server/data"
)

// ------------------------------------------------------------
// Testing data handling
// ------------------------------------------------------------

func createUserDb(name string) (data.User, error) {
	user, err := CreateUserAccountInDatabase("test user", "123")
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		return data.User{}, err
	}
	return user, nil
}

func createListBase(name string, createdBy int64) data.Shoppinglist {
	creator := data.ListCreator{
		ID:   createdBy,
		Name: "List Creator",
	}
	created := time.Now().Local()
	return data.Shoppinglist{
		ListId:     1,
		Name:       "Create List Name",
		CreatedBy:  creator,
		Created:    created,
		LastEdited: created,
		Items:      []data.ItemWire{},
	}
}

func TestCreatingList(t *testing.T) {
	connectDatabase()
	user, err := createUserDb("test user")
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	list := createListBase("list base", user.ID)
	list.CreatedBy.ID = user.ID
	err = CreateOrUpdateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create new list: %s", err)
		t.FailNow()
	}
	getList, err := GetShoppingList(list.ListId, list.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to get newly created shopping list: %s", err)
		t.FailNow()
	}
	if getList.CreatedBy.ID != list.CreatedBy.ID {
		log.Printf("IDs do not match")
		t.FailNow()
	}
	PrintShoppingListTable()
	log.Print("TestCreatingList successfully completed")
	ResetShoppingListTable()
	ResetUserTable()
}

func TestUpdatingList(t *testing.T) {
	connectDatabase()
	user, err := createUserDb("test user")
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	for i := 0; i < 3; i++ {
		list := createListBase("list base 1", user.ID)
		list.CreatedBy.ID = user.ID
		err = CreateOrUpdateShoppingList(list)
		if err != nil {
			log.Printf("Failed to create new list: %s", err)
			t.FailNow()
		}
		getList, err := GetShoppingList(list.ListId, list.CreatedBy.ID)
		if err != nil {
			log.Printf("Failed to get newly created shopping list: %s", err)
			t.FailNow()
		}
		if getList.CreatedBy.ID != list.CreatedBy.ID {
			log.Printf("IDs do not match")
			t.FailNow()
		}
	}
	lists, err := GetShoppingListsFromUserId(user.ID)
	if err != nil {
		log.Printf("Failed to get lists for user: %s", err)
		t.FailNow()
	}
	if len(lists) > 1 {
		log.Printf("Wanted to update the list, not create a new one! Found %d lists instead", len(lists))
		t.FailNow()
	}
	log.Print("TestUpdatingList successfully completed")
	ResetShoppingListTable()
	ResetUserTable()
}

func TestModifyListName(t *testing.T) {
	connectDatabase()
	user, err := createUserDb("test user")
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	list := createListBase("base list", user.ID)
	err = CreateOrUpdateShoppingList(list)
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
	oldName := getList.Name
	updatedName := "New List Name"
	updatedList.Name = updatedName
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
	if getList.Name == oldName {
		log.Print("List names still match after update!")
		t.FailNow()
	}
	if getList.Name != updatedName {
		log.Printf("Name update not correctly stored")
		t.FailNow()
	}
	log.Print("TestModifyListName successfully completed")
	ResetShoppingListTable()
	ResetUserTable()
}

func TestDeletingList(t *testing.T) {
	connectDatabase()
	user, err := createUserDb("test user")
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	list := createListBase("list base", user.ID)
	err = CreateOrUpdateShoppingList(list)
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
	err = DeleteShoppingList(list.ListId, user.ID)
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
