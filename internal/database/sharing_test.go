package database

import (
	"log"
	"testing"
	"time"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
)

func createDefaultSharing() data.ListShared {
	return data.ListShared{
		ID:           0,
		ListId:       1,
		CreatedBy:    1234,
		SharedWithId: []int64{2222},
		Created:      time.Now().Local(),
	}
}

func TestCreateSharing(t *testing.T) {
	connectDatabase()
	// Creating a user
	user, err := CreateUserAccountInDatabase("test", "bla")
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	sharedUser, err := CreateUserAccountInDatabase("shared user", "bla")
	if err != nil {
		log.Printf("Failed to create shared user: %s", err)
		t.FailNow()
	}
	listBase := createListBase("test", user.OnlineID)
	err = CreateOrUpdateShoppingList(listBase)
	if err != nil {
		log.Printf("Failed to create list for sharing: %s", err)
		t.FailNow()
	}
	shared := createDefaultSharing()
	shared.CreatedBy = user.OnlineID
	shared.SharedWithId = []int64{sharedUser.OnlineID}
	sharedWith, err := CreateOrUpdateSharedList(shared.ListId, shared.CreatedBy, shared.SharedWithId[0])
	if err != nil {
		log.Printf("Failed to create list sharing")
		t.FailNow()
	}
	if shared.ListId != sharedWith.ListId || shared.CreatedBy != sharedWith.CreatedBy || shared.SharedWithId[0] != sharedWith.SharedWithId[0] {
		log.Printf("Incorrectly inserted")
		t.FailNow()
	}
	getSharing, err := GetSharedListFromListId(shared.ListId)
	if err != nil {
		log.Printf("Expected sharing but got none: %s", err)
		t.FailNow()
	}
	if len(getSharing) != 1 {
		log.Printf("Expected only single sharing but got more (%d)", len(getSharing))
		t.FailNow()
	}
	onlySharing := getSharing[0]
	log.Printf("onlySharing: %v", onlySharing)
	if onlySharing.ListId != shared.ListId || onlySharing.CreatedBy != shared.CreatedBy || onlySharing.SharedWithId[0] != shared.SharedWithId[0] {
		log.Printf("Incorrectly inserted")
		t.FailNow()
	}
	PrintSharingTable()
	log.Printf("TestCreateSharing successful")
	ResetSharedListTable()
}

func TestCreateSharingWithoutUser(t *testing.T) {
	connectDatabase()
	shared := createDefaultSharing()
	if _, err := CreateOrUpdateSharedList(shared.ListId, shared.CreatedBy, shared.SharedWithId[0]); err == nil {
		log.Printf("Should fail because of non-existing user")
		t.FailNow()
	}
	if lists, err := GetSharedListFromListId(shared.ListId); err == nil && len(lists) > 0 {
		log.Printf("Expected no sharing but got some")
		t.FailNow()
	}
	PrintSharingTable()
	log.Printf("TestCreateSharing successful")
	ResetSharedListTable()
}

func TestCreatingMultipleSharings(t *testing.T) {
	connectDatabase()
	// Creating a user
	user, err := CreateUserAccountInDatabase("test", "bla")
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	sharedUser, err := CreateUserAccountInDatabase("shared user", "bla")
	if err != nil {
		log.Printf("Failed to create shared user: %s", err)
		t.FailNow()
	}
	listBase := createListBase("test", user.OnlineID)
	err = CreateOrUpdateShoppingList(listBase)
	if err != nil {
		log.Printf("Failed to create list for sharing: %s", err)
		t.FailNow()
	}
	shared := createDefaultSharing()
	shared.CreatedBy = user.OnlineID
	sharingList := make([]int64, 0)
	sharingList = append(sharingList, sharedUser.OnlineID)
	shared.SharedWithId = sharingList
	for i := 0; i < 3; i++ {
		sharedWith, err := CreateOrUpdateSharedList(shared.ListId, shared.CreatedBy, shared.SharedWithId[0])
		if err != nil {
			log.Printf("Failed to create list sharing")
			t.FailNow()
		}
		if shared.ListId != sharedWith.ListId || shared.CreatedBy != sharedWith.CreatedBy || shared.SharedWithId[0] != sharedWith.SharedWithId[0] {
			log.Printf("Incorrectly inserted")
			t.FailNow()
		}
		getSharing, err := GetSharedListFromListId(shared.ListId)
		if err != nil {
			log.Printf("Expected sharing but got none: %s", err)
			t.FailNow()
		}
		if len(getSharing) != 1 {
			log.Printf("Expected only single sharing but got more (%d)", len(getSharing))
			t.FailNow()
		}
		onlySharing := getSharing[0]
		if onlySharing.ListId != shared.ListId || onlySharing.CreatedBy != shared.CreatedBy || onlySharing.SharedWithId[0] != shared.SharedWithId[0] {
			log.Printf("Incorrectly inserted")
			t.FailNow()
		}
	}
	sharings, err := GetSharedListFromListId(shared.ListId)
	if err != nil {
		log.Printf("Failed to get shared list")
		t.FailNow()
	}
	if len(sharings) != 1 {
		log.Printf("Expected only single sharing but got %d", len(sharings))
		t.FailNow()
	}
	PrintSharingTable()
	log.Printf("TestCreateMapping successful")
	ResetSharedListTable()
}

func TestDeleteSharing(t *testing.T) {
	connectDatabase()
	// Creating a user
	user, err := CreateUserAccountInDatabase("test", "bla")
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	sharedUser, err := CreateUserAccountInDatabase("shared user", "bla")
	if err != nil {
		log.Printf("Failed to create shared user: %s", err)
		t.FailNow()
	}
	listBase := createListBase("test", user.OnlineID)
	err = CreateOrUpdateShoppingList(listBase)
	if err != nil {
		log.Printf("Failed to create list for sharing: %s", err)
		t.FailNow()
	}
	shared := createDefaultSharing()
	shared.CreatedBy = user.OnlineID
	sharingList := make([]int64, 0)
	sharingList = append(sharingList, sharedUser.OnlineID)
	shared.SharedWithId = sharingList
	_, err = CreateOrUpdateSharedList(shared.ListId, shared.CreatedBy, shared.SharedWithId[0])
	if err != nil {
		log.Printf("Failed to create list sharing")
		t.FailNow()
	}
	err = DeleteSharingForUser(shared.ListId, shared.CreatedBy, shared.SharedWithId[0])
	if err != nil {
		log.Printf("Failed to delete sharing: %s", err)
		t.FailNow()
	}
	getSharing, err := GetSharedListFromListId(shared.ListId)
	if err != nil {
		log.Printf("Expected no error but got some: %s", err)
		t.FailNow()
	}
	if len(getSharing) != 0 {
		log.Printf("Expected only single sharing but got more (%d)", len(getSharing))
		t.FailNow()
	}
	PrintSharingTable()
	log.Printf("TestCreateMapping successful")
	ResetSharedListTable()
}
