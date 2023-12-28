package database_test

import (
	"crypto/sha1"
	"encoding/base64"
	"log"
	"strconv"
	"testing"

	"shop.cloudsheeptech.com/database"
	"shop.cloudsheeptech.com/server/configuration"
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

// ------------------------------------------------------------
// Testing user creation and authentication
// ------------------------------------------------------------

func TestInsertUser(t *testing.T) {
	connectDatabase()
	user := database.User{
		ID:       4,
		Username: strconv.Itoa(4),
		Passwd:   "Biene Maja",
		Salt:     "1234",
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

// ------------------------------------------------------------
// Testing data handling
// ------------------------------------------------------------

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

func TestGetMapping(t *testing.T) {
	connectDatabase()
	id := 0
	mapping, err := database.GetMappingWithListId(id)
	if err != nil {
		log.Printf("Failed to get mapping for id %d", id)
		t.FailNow()
	}
	if len(mapping) == 0 {
		t.FailNow()
	}
	log.Print("GetMapping successfully completed")
}

func TestCreatingUser(t *testing.T) {
	connectDatabase()
	log.Print("Trying to create new user")
	log.Print("Old User Table")
	database.PrintUserTable("loginuser")
	user, err := database.CreateUserAccount("testuser", "schlechtes wetter")
	if err != nil {
		log.Printf("Failed to create new user: %s", err)
		t.FailNow()
	}
	log.Printf("Created user: %v", user)
	database.PrintUserTable("loginuser")
	log.Print("Successfully created new user")
}

func TestCheckingUserLogin(t *testing.T) {
	log.Print("Trying to check if user can login")
	connectDatabase()
	userId := 5953928440124292227
	loginUser, err := database.GetLoginUser(int64(userId))
	if err != nil {
		log.Printf("Failed to retrieve user with id %d", userId)
		t.FailNow()
	}
	passwd := "schlechtes wetter"
	hasher := sha1.New()
	hasher.Write([]byte(passwd + loginUser.Salt))
	hashedPwd := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	if loginUser.Passwd != hashedPwd {
		log.Print("Given password does not match the stored password!")
		t.FailNow()
	}
	log.Print("User correctly stored and retrieved")
}
