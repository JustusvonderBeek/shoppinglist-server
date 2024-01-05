package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
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
	JWTTimeout:     1200, // 20 minutes; ONLY for testing
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

func TestShowUsers(t *testing.T) {
	connectDatabase()
	database.PrintUserTable("")
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
	// We get the hash back but need to store the password
	user.Password = PASSWORD
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

func loadUserAndSetupFields(id int64, name string, password string) (io.Reader, error) {
	user, err := readUserFile()
	if err != nil {
		return nil, err
	}
	if id != 0 {
		log.Printf("Set id to %d", id)
		user.ID = id
	}
	if name != "" {
		log.Printf("Set name to %s", name)
		user.Username = name
	}
	if password != "" {
		log.Printf("Set password to %s", password)
		user.Password = password
	}
	raw, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(raw)
	return reader, nil
}

// TODO: Fix the whitelisted IP not showing in the test
func TestUserCreation(t *testing.T) {
	log.Print("Testing creating new user")
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	newUser := data.User{
		ID:       0,
		Username: "test creation user",
		Password: "new password",
	}
	rawUser, err := json.Marshal(newUser)
	if err != nil {
		log.Printf("Failed to encode user: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(rawUser)
	r.POST("/auth/create", authentication.CreateAccount)
	c.Request, _ = http.NewRequest("POST", "/auth/create", reader)
	// Set a custom IP address for the request
	c.Request.Host = "192.168.1.33"
	c.Request.Header.Set("X-Real-IP", "192.168.1.33")
	router.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusCreated, w.Code)

	var answeredUser data.User
	if err = json.Unmarshal(w.Body.Bytes(), &answeredUser); err != nil {
		log.Printf("Did not receive a user as answer!")
		t.FailNow()
	}
	assert.NotEqual(t, 0, answeredUser.ID)
	assert.Equal(t, answeredUser.Username, newUser.Username)
	assert.Equal(t, "accepted", newUser.Password)

	w = httptest.NewRecorder()
	newUser.ID = answeredUser.ID
	rawUser, err = json.Marshal(newUser)
	if err != nil {
		log.Printf("Failed to encode user: %s", err)
		t.FailNow()
	}
	reader = bytes.NewReader(rawUser)
	req, _ := http.NewRequest("POST", "/auth/login", reader)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogin(t *testing.T) {
	log.Print("Testing login function")
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	reader, err := loadUserAndSetupFields(0, "", "")
	if err != nil {
		log.Printf("Failed to load user: %s", err)
		t.FailNow()
	}
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

func TestLoginIncorrectUsername(t *testing.T) {
	log.Print("Testing login with wrong username")
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	unknownUserName := "not known"

	reader, err := loadUserAndSetupFields(0, unknownUserName, "")
	if err != nil {
		log.Printf("Failed to load and setup user: %s", err)
		t.FailNow()
	}
	req, _ := http.NewRequest("POST", "/auth/login", reader)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginIncorrectPassword(t *testing.T) {
	log.Print("Testing login with wrong password")
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	unknownPassword := "empty"
	reader, err := loadUserAndSetupFields(0, "", unknownPassword)
	if err != nil {
		log.Printf("Failed to load and setup user: %s", err)
		t.FailNow()
	}
	req, _ := http.NewRequest("POST", "/auth/login", reader)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginIncorrectId(t *testing.T) {
	log.Print("Testing login with wrong password")
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	unknownUserId := 12345
	reader, err := loadUserAndSetupFields(int64(unknownUserId), "", "")
	if err != nil {
		log.Printf("Failed to load and setup user: %s", err)
		t.FailNow()
	}
	req, _ := http.NewRequest("POST", "/auth/login", reader)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthenticationTimeoutedToken(t *testing.T) {
	log.Print("Testing login with token that timed out")
	connectDatabase()

	testConfiguration := cfg
	testConfiguration.JWTTimeout = 1

	router := server.SetupRouter(testConfiguration)
	w := httptest.NewRecorder()

	reader, err := loadUserAndSetupFields(0, "", "")
	if err != nil {
		log.Printf("Failed to load and setup user: %s", err)
		t.FailNow()
	}
	req, _ := http.NewRequest("POST", "/auth/login", reader)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var token authentication.Token
	if json.Unmarshal(w.Body.Bytes(), &token) != nil {
		log.Printf("Failed to decode answer into token! %s", err)
		t.FailNow()
	}

	tkn, err := jwt.Parse(token.Token, func(t *jwt.Token) (interface{}, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("unauthorized")
		}
		pwd, _ := os.Getwd()
		finalJWTFile := filepath.Join(pwd, cfg.JWTSecretFile)
		data, err := os.ReadFile(finalJWTFile)
		if err != nil {
			log.Print("Failed to find JWT secret file")
			return nil, err
		}
		var jwtSecret authentication.JWTSecretFile
		err = json.Unmarshal(data, &jwtSecret)
		if err != nil {
			log.Print("JWT secret file is in incorrect format")
			return nil, err
		}
		log.Print("Parsed and loaded JWT secret file")
		secretByteKey := []byte(jwtSecret.Secret)
		return secretByteKey, nil
	})
	if err != nil {
		log.Printf("Failed to parse token: %s", err)
		t.FailNow()
	}
	log.Printf("Token is still valid: %v", tkn.Claims)
	// Now we wait and try to access the debug resource with our invalid token
	log.Printf("Waiting for token to time out: %s", "")
	time.Sleep(time.Second * 2)

	w = httptest.NewRecorder()
	// Adding the authentication token
	req, _ = http.NewRequest("GET", "/v1/test/auth", reader)
	bearer := "Bearer " + token.Token
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthentcationWrongTokenSignature(t *testing.T) {
	log.Print("Testing login with token that is invalid (wrong signature) wrong username, wrong id)")
	connectDatabase()

	user, err := readUserFile()
	if err != nil {
		log.Printf("Failed to read user: %s", err)
		t.FailNow()
	}

	expirationTime := time.Now().Add(1 * time.Minute)
	wrongUsername := authentication.Claims{
		Id:       int(user.ID),
		Username: user.Username + "invalid",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	ownToken := jwt.NewWithClaims(jwt.SigningMethodHS512, wrongUsername)
	content, err := os.ReadFile(cfg.JWTSecretFile)
	if err != nil {
		log.Printf("Cannot read token secret file! %s", err)
		t.FailNow()
	}
	var jwtSecretFile authentication.JWTSecretFile
	err = json.Unmarshal(content, &jwtSecretFile)
	if err != nil {
		log.Printf("The given jwt secret file is in incorrect format! %s", err)
		t.FailNow()
	}
	signedToken, err := ownToken.SignedString([]byte(jwtSecretFile.Secret))
	if err != nil {
		log.Printf("Failed to sign token: %s", err)
		t.FailNow()
	}

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	reader, err := loadUserAndSetupFields(0, "", "")
	if err != nil {
		log.Printf("Failed to load and setup user: %s", err)
		t.FailNow()
	}
	req, _ := http.NewRequest("GET", "/v1/test/auth", reader)
	bearer := "Bearer " + signedToken
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuthenticationModifiedToken(t *testing.T) {
	log.Print("Testing login with token that was modified")
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	reader, err := loadUserAndSetupFields(0, "", "")
	if err != nil {
		log.Printf("Failed to load and setup user: %s", err)
		t.FailNow()
	}
	req, _ := http.NewRequest("POST", "/auth/login", reader)
	router.ServeHTTP(w, req)

	var token authentication.Token
	if json.Unmarshal(w.Body.Bytes(), &token) != nil {
		log.Printf("Failed to decode answer into token! %s", err)
		t.FailNow()
	}
	// log.Printf("Answered Token: %s", token.Token)
	// Modify the token
	modifiedToken := strings.ReplaceAll(token.Token, "U", "u")

	w = httptest.NewRecorder()
	// Adding the authentication token
	req, _ = http.NewRequest("GET", "/v1/test/auth", reader)
	bearer := "Bearer " + modifiedToken
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUnissuedToken(t *testing.T) {
	log.Print("Testing login with unissued token")
	connectDatabase()

	user, err := readUserFile()
	if err != nil {
		log.Printf("Failed to read user: %s", err)
		t.FailNow()
	}

	expirationTime := time.Now().Add(1 * time.Minute)
	userToken := authentication.Claims{
		Id:       int(user.ID),
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	ownToken := jwt.NewWithClaims(jwt.SigningMethodHS512, userToken)
	content, err := os.ReadFile(cfg.JWTSecretFile)
	if err != nil {
		log.Printf("Cannot read token secret file! %s", err)
		t.FailNow()
	}
	var jwtSecretFile authentication.JWTSecretFile
	err = json.Unmarshal(content, &jwtSecretFile)
	if err != nil {
		log.Printf("The given jwt secret file is in incorrect format! %s", err)
		t.FailNow()
	}
	signedToken, err := ownToken.SignedString([]byte(jwtSecretFile.Secret))
	if err != nil {
		log.Printf("Failed to sign token: %s", err)
		t.FailNow()
	}

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	reader, err := loadUserAndSetupFields(0, "", "")
	if err != nil {
		log.Printf("Failed to load and setup user: %s", err)
		t.FailNow()
	}
	req, _ := http.NewRequest("GET", "/v1/test/auth", reader)
	bearer := "Bearer " + signedToken
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
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

	var allLists []data.Shoppinglist
	if json.Unmarshal(w.Body.Bytes(), &allLists) != nil {
		log.Printf("Failed to parse server answer. Expected lists JSON: %s", err)
		t.FailNow()
	}

	assert.Equal(t, 4, len(allLists))
	for i := 0; i < 4; i++ {
		assert.Equal(t, offlineList[i].CreatedBy, allLists[i].CreatedBy)
		assert.Equal(t, offlineList[i].LastEdited, allLists[i].LastEdited)
		assert.Equal(t, offlineList[i].Name, allLists[i].Name)
		assert.Equal(t, offlineList[i].ID, allLists[i].ID)
	}

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
	database.ResetSharedListTable()
}

func TestRemoveList(t *testing.T) {
	// TODO:
}

func TestCreateSharing(t *testing.T) {
	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}
	log.Printf("Testing creating sharing for user %d", user.ID)
	connectDatabase()

	// Creating two own lists and share one with a random user
	sharedWithUserId := 12345
	var offlineList []data.Shoppinglist
	for i := 0; i < 2; i++ {
		list, err := createListOffline("own list "+strconv.Itoa(i+1), user.ID)
		if err != nil {
			log.Printf("Failed to create list: %s", err)
			t.FailNow()
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
	sharedWith := data.ListShared{
		ID:         0,
		ListId:     offlineList[0].ID,
		SharedWith: int64(sharedWithUserId),
	}
	encodedShared, err := json.Marshal(sharedWith)
	if err != nil {
		log.Printf("Failed to encoded data: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(encodedShared)
	req, _ := http.NewRequest("POST", "/v1/share/"+strconv.Itoa(int(offlineList[0].ID)), reader)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	sharedDb, err := database.GetSharedListForUserId(int64(sharedWithUserId))

	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(sharedDb))
	assert.Equal(t, sharedWith.ListId, sharedDb[0].ListId)
	assert.Equal(t, sharedWith.SharedWith, sharedDb[0].SharedWith)

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
	database.ResetSharedListTable()
}

func TestCreateSharingOfUnownedList(t *testing.T) {
	// TODO: Testing that no one can create sharing for list that is not owned by himself
	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}
	log.Printf("Testing creating sharing for user %d", user.ID)
	connectDatabase()

	// Creating a list that WE DO NOT OWN
	list, err := createListOffline("unowned list 1", user.ID+1)
	if err != nil {
		log.Printf("Failed to create list: %s", err)
		t.FailNow()
	}

	// Now trying if we can share the list via the API
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	sharedWithUserId := 1234
	sharedWith := data.ListShared{
		ID:         0,
		ListId:     list.ID,
		SharedWith: int64(sharedWithUserId),
	}
	encodedShared, err := json.Marshal(sharedWith)
	if err != nil {
		log.Printf("Failed to encoded data: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(encodedShared)
	req, _ := http.NewRequest("POST", "/v1/share/"+strconv.Itoa(int(list.ID)), reader)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	database.PrintSharingTable()
	database.ResetShoppingListTable()
	database.ResetSharedListTable()
}

func TestPushingOwnItem(t *testing.T) {
	log.Print("Testing sharing one own item")
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	item := data.Item{
		ID:   0,
		Name: "new item",
		Icon: "unknown",
	}
	encodedItem, err := json.Marshal(item)
	if err != nil {
		log.Printf("Failed to encoded item: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(encodedItem)
	req, _ := http.NewRequest("POST", "/v1/item", reader)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	database.PrintItemTable()
	database.ResetItemTable()
}

func TestPushingMultipleItems(t *testing.T) {
	log.Print("Testing sharing of all own items")
	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}
	log.Printf("Testing creating sharing for user %d", user.ID)
	connectDatabase()

	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	item := data.Item{
		ID:   0,
		Name: "new item",
		Icon: "unknown",
	}
	var multipleItems []data.Item
	for i := 0; i < 3; i++ {
		multipleItems = append(multipleItems, item)
	}
	encodedItem, err := json.Marshal(multipleItems)
	if err != nil {
		log.Printf("Failed to encoded items: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(encodedItem)
	req, _ := http.NewRequest("POST", "/v1/items", reader)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	database.PrintItemTable()
	database.ResetItemTable()
}
