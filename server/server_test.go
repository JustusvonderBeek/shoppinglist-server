package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"shop.cloudsheeptech.com/database"
	"shop.cloudsheeptech.com/server"
	"shop.cloudsheeptech.com/server/configuration"
	"shop.cloudsheeptech.com/server/data"
)

// ------------------------------------------------------------
// Handling the user for testing + connect database
// ------------------------------------------------------------

const USERNAME = "testuser"
const PASSWORD = "password"

var USER_ID = 0
var cfg = configuration.Config{
	ListenAddr:     "0.0.0.0",
	ListenPort:     "46152",
	DatabaseConfig: "resources/db.json",
	TLSCertificate: "resources/shoppinglist.crt",
	TLSKeyfile:     "resources/shoppinglist.pem",
	JWTSecretFile:  "resources/jwtSecret.json",
}

func connectDatabase() {
	cfg := configuration.Config{
		DatabaseConfig: "../resources/db.json",
	}
	database.CheckDatabaseOnline(cfg)
}

func TestCreateUser(t *testing.T) {
	log.Print("Creating test user")
	connectDatabase()
	user, err := database.CreateUserAccount(USERNAME, PASSWORD)
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	if user.ID == 0 {
		log.Printf("Failed to create user: %s", "user id == 0")
		t.FailNow()
	}
	USER_ID = int(user.ID)
	log.Print("Test user successfully created")
}

func DeleteTestUser(t *testing.T) {
	log.Print("Deleting test user")
	connectDatabase()
	if USER_ID == 0 {
		log.Print("Internal error: cannot find user ID!")
		t.FailNow()
	}
	err := database.DeleteUserAccount(int64(USER_ID))
	if err != nil {
		log.Printf("Failed to delete user: %s", err)
		t.FailNow()
	}
	log.Print("User deleted")
}

// ------------------------------------------------------------
// Testing the list functions
// ------------------------------------------------------------

func addListForUser(id int64) error {
	log.Printf("Adding list for user: %d", id)
	listName := "test list"
	list, err := database.CreateShoppingList(listName, id)
	if err != nil {
		return err
	}
	if list.Name != listName {
		return errors.New("list name incorrectly stored")
	}
	retrieveList, err := database.GetShoppingList(list.ID)
	if err != nil {
		return err
	}
	if retrieveList.Name != list.Name || retrieveList.LastEdited != list.LastEdited || retrieveList.CreatedBy != list.CreatedBy {
		return errors.New("returned and stored lists differ")
	}
	return nil
}

func TestCreatingList(t *testing.T) {
	log.Print("Testing creating list")
	// Creating with default configuration
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	listName := "test list"
	list := data.Shoppinglist{
		ID:         0,
		Name:       listName,
		CreatedBy:  int64(USER_ID),
		LastEdited: time.Now().Format(time.RFC3339),
	}
	jsonList, err := json.Marshal(list)
	if err != nil {
		log.Printf("Failed to encode list. Error in test")
		t.FailNow()
	}
	reader := bytes.NewReader(jsonList)
	bearer := "Bearer " + ""
	req, _ := http.NewRequest("POST", "/v1/list", reader)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "PONG", w.Body.String())
}

func TestGetAllLists(t *testing.T) {
	log.Printf("Testing getting all lists for user %d", USER_ID)

}
