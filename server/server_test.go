package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"shop.cloudsheeptech.com/database"
	"shop.cloudsheeptech.com/server"
	"shop.cloudsheeptech.com/server/authentication"
	"shop.cloudsheeptech.com/server/configuration"
	"shop.cloudsheeptech.com/server/data"
)

// ------------------------------------------------------------
// Data types and config for testing
// ------------------------------------------------------------

const USERNAME = "testuser"
const PASSWORD = "password"
const TESTING_DIR = "testing/resources/"
const JWT_FILE = "jwt.token"
const USER_FILE = "user.json"

var cfg = configuration.Config{
	ListenAddr:     "0.0.0.0",
	ListenPort:     "46152",
	DatabaseConfig: "../resources/db.json",
	TLSCertificate: "../resources/shoppinglist.crt",
	TLSKeyfile:     "../resources/shoppinglist.pem",
	JWTSecretFile:  "../resources/jwtSecret.json",
	JWTTimeout:     20, // 20 minutes; ONLY for testing
}

// ------------------------------------------------------------
// Testing helper functions
// ------------------------------------------------------------

func storeUser(user data.User) error {
	raw, err := json.Marshal(user)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(TESTING_DIR, 0777); err != nil {
		return err
	}
	if err = os.WriteFile(TESTING_DIR+USER_FILE, raw, 0660); err != nil {
		return err
	}
	return nil
}

func readUserFile() (data.User, error) {
	content, err := os.ReadFile(TESTING_DIR + USER_FILE)
	if err != nil {
		return data.User{}, err
	}
	var user data.User
	err = json.Unmarshal(content, &user)
	if err != nil {
		return data.User{}, err
	}
	return user, nil
}

func storeJwtToFile(jwt string) {
	log.Print("Storing JWT token for testing to file")
	err := os.MkdirAll(TESTING_DIR, 0777)
	if err != nil {
		log.Printf("Failed to create directory: %s", err)
		return
	}
	if err := os.WriteFile(TESTING_DIR+JWT_FILE, []byte(jwt), 0660); err != nil {
		log.Printf("Failed to write JWT token to file: %s", err)
	}
}

func readJwtFromFile() (string, error) {
	if content, err := os.ReadFile(TESTING_DIR + JWT_FILE); err != nil {
		return "", err
	} else {
		return string(content), nil
	}
}

// ------------------------------------------------------------
// Database helper + setup functions
// ------------------------------------------------------------

func connectDatabase() {
	cfg := configuration.Config{
		DatabaseConfig: "../resources/db.json",
	}
	database.CheckDatabaseOnline(cfg)
}

func TestSetupTesting(t *testing.T) {
	connectDatabase()
	// Depending on the test add or remove user
	// CreateTestUser(t)
	// DeleteTestUser(t)
}

func CreateTestUser(t *testing.T) {
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
	if err = storeUser(user); err != nil {
		log.Printf("Failed to store user: %s", err)
		t.FailNow()
	}
	log.Print("Test user successfully created")
}

func DeleteTestUser(t *testing.T) {
	log.Print("Deleting test user")
	connectDatabase()
	user, err := readUserFile()
	if err != nil {
		log.Print("Cannot delete nil user")
		t.FailNow()
	}
	err = database.DeleteUserAccount(user.ID)
	if err != nil {
		log.Printf("Failed to delete user: %s", err)
		t.FailNow()
	}
	err = os.Remove(TESTING_DIR + USER_FILE)
	if err != nil {
		log.Printf("Failed to remove user file: %s", err)
		t.FailNow()
	}
	log.Print("User deleted")
}

// ------------------------------------------------------------
// Testing the authentication methods
// ------------------------------------------------------------

func TestLogin(t *testing.T) {
	log.Print("Testing login function")
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	user, err := readUserFile()
	if err != nil {
		log.Printf("Failed to read user: %s", err)
		t.FailNow()
	}

	raw, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to encode user: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(raw)
	req, _ := http.NewRequest("POST", "/auth/login", reader)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var token authentication.Token
	if json.Unmarshal(w.Body.Bytes(), &token) != nil {
		log.Printf("Failed to decode answer into token! %s", err)
		t.FailNow()
	}
	storeJwtToFile(token.Token)
	log.Print("Logged in and stored jwt secret to file")
}

// ------------------------------------------------------------
// Testing the list methods
// ------------------------------------------------------------

func createListOffline(name string, userId int64) (data.Shoppinglist, error) {
	list, err := database.CreateShoppingList(name, userId, time.Now().Format(time.RFC3339))
	if err != nil {
		return data.Shoppinglist{}, err
	}
	if list.ID == 0 || list.Name != name || list.CreatedBy != userId {
		return data.Shoppinglist{}, errors.New("list was incorrectly stored")
	}
	return list, nil
}

func createListSharing(listId int64, userId int64) (data.ListShared, error) {
	sharing, err := database.CreateSharedList(listId, userId)
	if err != nil {
		return data.ListShared{}, err
	}
	if sharing.ID == 0 || sharing.ListId != listId || sharing.SharedWith != userId {
		return data.ListShared{}, errors.New("sharing was incorrectly stored")
	}
	return sharing, nil
}

func TestCreatingList(t *testing.T) {
	log.Print("Testing creating list")
	connectDatabase()
	// Creating with default configuration
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	user, err := readUserFile()
	if err != nil {
		log.Printf("Failed to read user from file: %s", err)
		t.FailNow()
	}
	listName := "test list"
	list := data.Shoppinglist{
		ID:         0,
		Name:       listName,
		CreatedBy:  user.ID,
		LastEdited: time.Now().Format(time.RFC3339),
	}
	jsonList, err := json.Marshal(list)
	if err != nil {
		log.Printf("Failed to encode list. Error in test")
		t.FailNow()
	}
	reader := bytes.NewReader(jsonList)
	// Load authentication token
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	req, _ := http.NewRequest("POST", "/v1/list", reader)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	// Parsing answer and expect everything the same except ID
	var onlineList data.Shoppinglist
	if err = json.Unmarshal(w.Body.Bytes(), &onlineList); err != nil {
		log.Printf("Expected JSON list, but parsing failed: %s", err)
		t.FailNow()
	}

	assert.NotEqual(t, 0, onlineList.ID)
	assert.Equal(t, list.Name, onlineList.Name)
	assert.Equal(t, list.CreatedBy, onlineList.CreatedBy)
	assert.Equal(t, list.LastEdited, onlineList.LastEdited)

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
}

func TestGetAllOwnLists(t *testing.T) {
	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}
	log.Printf("Testing getting all own lists for user %d", user.ID)
	connectDatabase()

	// Add two lists for our user behind the curtains
	var offlineList []data.Shoppinglist
	for i := 0; i < 2; i++ {
		if list, err := createListOffline("own list "+strconv.Itoa(i+1), user.ID); err != nil {
			log.Printf("Failed to create list: %s", err)
		} else {
			offlineList = append(offlineList, list)
		}
	}

	// Now trying if we can get both lists via the API
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	req, _ := http.NewRequest("GET", "/v1/lists/"+strconv.Itoa(int(user.ID)), nil)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var allOwnLists []data.Shoppinglist
	if json.Unmarshal(w.Body.Bytes(), &allOwnLists) != nil {
		log.Printf("Failed to parse server answer. Expected lists JSON: %s", err)
		t.FailNow()
	}

	assert.Equal(t, 2, len(allOwnLists))
	for i := 0; i < 2; i++ {
		assert.Equal(t, user.ID, allOwnLists[i].CreatedBy)
		assert.Equal(t, offlineList[i].LastEdited, allOwnLists[i].LastEdited)
		assert.Equal(t, offlineList[i].Name, allOwnLists[i].Name)
		assert.Equal(t, offlineList[i].ID, allOwnLists[i].ID)
	}

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
}

func TestGetAllLists(t *testing.T) {
	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}
	log.Printf("Testing getting all lists for user %d", user.ID)
	connectDatabase()

	// Creating two own lists
	var offlineList []data.Shoppinglist
	for i := 0; i < 2; i++ {
		if list, err := createListOffline("own list "+strconv.Itoa(i+1), user.ID); err != nil {
			log.Printf("Failed to create list: %s", err)
			t.FailNow()
		} else {
			offlineList = append(offlineList, list)
		}
	}
	// Creating two shared lists from two different IDs
	for i := 0; i < 2; i++ {
		list, err := createListOffline("shared list from "+strconv.Itoa(i+1), int64(i+1))
		if err != nil {
			log.Printf("Failed to created shared list: %s", err)
			t.FailNow()
		}
		offlineList = append(offlineList, list)
		// Create the sharing
		if _, err = createListSharing(list.ID, user.ID); err != nil {
			log.Printf("Failed to create sharing: %s", err)
			t.FailNow()
		}
	}

	// Now trying if we can get both lists via the API
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	req, _ := http.NewRequest("GET", "/v1/lists/"+strconv.Itoa(int(user.ID)), nil)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var allOwnLists []data.Shoppinglist
	if json.Unmarshal(w.Body.Bytes(), &allOwnLists) != nil {
		log.Printf("Failed to parse server answer. Expected lists JSON: %s", err)
		t.FailNow()
	}

	assert.Equal(t, 4, len(allOwnLists))
	for i := 0; i < 4; i++ {
		assert.Equal(t, user.ID, allOwnLists[i].CreatedBy)
		assert.Equal(t, offlineList[i].LastEdited, allOwnLists[i].LastEdited)
		assert.Equal(t, offlineList[i].Name, allOwnLists[i].Name)
		assert.Equal(t, offlineList[i].ID, allOwnLists[i].ID)
	}

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
}