package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/authentication"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/configuration"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/server"
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
	// database.PrintUserTable("")
	database.PrintShoppingListTable()
	database.PrintItemPerListTable()
	database.PrintItemTable()
}

func TestResetUserDatabase(t *testing.T) {
	connectDatabase()
	database.PrintUserTable("")
	database.ResetUserTable()
	database.PrintUserTable("")
}

func CreateTestUser(t *testing.T) {
	log.Print("Creating test user")
	connectDatabase()
	user, err := database.CreateUserAccountInDatabase(USERNAME, PASSWORD)
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		t.FailNow()
	}
	if user.OnlineID == 0 {
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
	err = database.DeleteUserAccount(user.OnlineID)
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
		user.OnlineID = id
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

	// router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	newUser := data.User{
		OnlineID: 0,
		Username: "test creation user",
		Password: "new password",
	}
	rawUser, err := json.Marshal(newUser)
	if err != nil {
		log.Printf("Failed to encode user: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(rawUser)
	authentication.Setup(cfg)
	r.POST("/auth/create", authentication.CreateAccount)

	c.Request, _ = http.NewRequest("POST", "/auth/create", reader)
	// Set a custom IP address for the request
	c.Request.RemoteAddr = "192.168.1.33:41111"
	c.Request.Header.Set("X-Real-Ip", "192.168.1.33:41111")
	log.Printf("Client IP: %s", c.ClientIP())
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusCreated, w.Code)

	var answeredUser data.User
	if err = json.Unmarshal(w.Body.Bytes(), &answeredUser); err != nil {
		log.Printf("Did not receive a user as answer!")
		t.FailNow()
	}
	assert.NotEqual(t, 0, answeredUser.OnlineID)
	assert.Equal(t, answeredUser.Username, newUser.Username)
	assert.Equal(t, "accepted", answeredUser.Password)

	w = httptest.NewRecorder()
	newUser.OnlineID = answeredUser.OnlineID
	rawUser, err = json.Marshal(newUser)
	if err != nil {
		log.Printf("Failed to encode user: %s", err)
		t.FailNow()
	}
	reader = bytes.NewReader(rawUser)
	r.POST("/auth/login", authentication.Login)
	req, _ := http.NewRequest("POST", "/auth/login", reader)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func login(t *testing.T) {
	// Expecting an offline user for this test
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	reader, err := loadUserAndSetupFields(0, "", "")
	if err != nil {
		log.Printf("Failed to load user: %s", err)
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
	storeJwtToFile(token.Token)
	log.Print("Logged in and stored jwt secret to file")
}

func TestLogin(t *testing.T) {
	log.Print("Testing login function")
	connectDatabase()
	// Creating an offline user for this test
	CreateTestUser(t)
	login(t)
	DeleteTestUser(t)
}

func TestLoginIncorrectUsername(t *testing.T) {
	log.Print("Testing login with wrong username")
	connectDatabase()
	CreateTestUser(t)

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
	DeleteTestUser(t)
}

func TestLoginIncorrectPassword(t *testing.T) {
	log.Print("Testing login with wrong password")
	connectDatabase()
	CreateTestUser(t)

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
	DeleteTestUser(t)
}

func TestLoginIncorrectId(t *testing.T) {
	log.Print("Testing login with wrong password")
	connectDatabase()
	CreateTestUser(t)

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
	DeleteTestUser(t)
}

func TestAuthenticationTimeoutedToken(t *testing.T) {
	log.Print("Testing login with token that timed out")
	connectDatabase()
	CreateTestUser(t)

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
	DeleteTestUser(t)
}

func TestAuthentcationWrongTokenSignature(t *testing.T) {
	log.Print("Testing login with token that is invalid (wrong signature) wrong username, wrong id)")
	connectDatabase()
	CreateTestUser(t)

	user, err := readUserFile()
	if err != nil {
		log.Printf("Failed to read user: %s", err)
		t.FailNow()
	}

	expirationTime := time.Now().Add(1 * time.Minute)
	wrongUsername := authentication.Claims{
		Id:       int(user.OnlineID),
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
	DeleteTestUser(t)
}

func TestAuthenticationModifiedToken(t *testing.T) {
	log.Print("Testing login with token that was modified")
	connectDatabase()
	CreateTestUser(t)

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
	DeleteTestUser(t)
}

func TestUnissuedToken(t *testing.T) {
	log.Print("Testing login with unissued token")
	connectDatabase()
	CreateTestUser(t)

	user, err := readUserFile()
	if err != nil {
		log.Printf("Failed to read user: %s", err)
		t.FailNow()
	}

	expirationTime := time.Now().Add(1 * time.Minute)
	userToken := authentication.Claims{
		Id:       int(user.OnlineID),
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
	DeleteTestUser(t)
}

// ------------------------------------------------------------
// Testing the list methods
// ------------------------------------------------------------

func createListOffline(name string, userId int64, items []data.ItemWire) (data.List, error) {
	creator := data.ListCreator{
		ID:   userId,
		Name: USERNAME,
	}
	timeNow := time.Now().UTC()
	list := data.List{
		ListId:     rand.Int63(),
		Name:       name,
		CreatedBy:  creator,
		Created:    timeNow,
		LastEdited: timeNow,
		Items:      items,
	}
	err := database.CreateOrUpdateShoppingList(list)
	if err != nil {
		return data.List{}, err
	}
	if list.ListId == 0 || list.Name != name || list.CreatedBy.ID != userId {
		return data.List{}, errors.New("list was incorrectly stored")
	}
	return list, nil
}

func createItemsWire(name string, numberOfItems int) []data.ItemWire {
	items := make([]data.ItemWire, 0)
	for i := 0; i < numberOfItems; i++ {
		item := data.ItemWire{
			Name:     fmt.Sprintf("%s %d", name, i+1),
			Icon:     "ic_icon",
			Quantity: int64(i + 1),
			Checked:  i%2 == 0,
		}
		items = append(items, item)
	}
	return items
}

func createListSharing(listId int64, createdBy int64, userId int64) (data.ListShared, error) {
	sharing, err := database.CreateOrUpdateSharedList(listId, createdBy, userId)
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
	CreateTestUser(t)
	login(t)

	// Creating with default configuration
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	user, err := readUserFile()
	if err != nil {
		log.Printf("Failed to read user from file: %s", err)
		t.FailNow()
	}
	listName := "test list"
	creator := data.ListCreator{
		ID:   user.OnlineID,
		Name: user.Username,
	}
	timeNow := time.Now().UTC()
	list := data.List{
		ListId:     rand.Int63(),
		Name:       listName,
		CreatedBy:  creator,
		Created:    timeNow,
		LastEdited: timeNow,
		Items: []data.ItemWire{
			{
				Name:     "Item",
				Icon:     "ic_item",
				Quantity: 1,
				Checked:  false,
			},
		},
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

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
	database.ResetItemTable()
	database.ResetItemPerListTable()
	// Should already delete all mappings
	DeleteTestUser(t)
}

func roundTime(t time.Time) time.Time {
	return t.Round(time.Duration(time.Second))
}

func TestGetAllOwnLists(t *testing.T) {
	log.Print("Testing if all own lists can be obtained")
	connectDatabase()
	CreateTestUser(t)

	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	// Add two lists for our user behind the curtains
	var offlineList []data.List
	for i := 0; i < 2; i++ {
		if list, err := createListOffline("own list "+strconv.Itoa(i+1), user.OnlineID, []data.ItemWire{}); err != nil {
			log.Printf("Failed to create list: %s", err)
		} else {
			// TODO: Add items for this list
			offlineList = append(offlineList, list)
		}
	}

	// Now trying if we can get both lists via the API
	login(t)
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/lists/%d", user.OnlineID), nil)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var allOwnLists []data.List
	if json.Unmarshal(w.Body.Bytes(), &allOwnLists) != nil {
		log.Printf("Failed to parse server answer. Expected lists JSON: %s", err)
		t.FailNow()
	}

	assert.Equal(t, 2, len(allOwnLists))
	for i := 0; i < 2; i++ {
		assert.Equal(t, user.OnlineID, allOwnLists[i].CreatedBy.ID)
		assert.Equal(t, user.Username, allOwnLists[i].CreatedBy.Name)
		assert.Equal(t, roundTime(offlineList[i].LastEdited).Format(time.RFC3339), roundTime(allOwnLists[i].LastEdited).Format(time.RFC3339))
		assert.Equal(t, roundTime(offlineList[i].Created).Format(time.RFC3339), roundTime(allOwnLists[i].Created).Format(time.RFC3339))
		assert.Equal(t, offlineList[i].Name, allOwnLists[i].Name)
		assert.Equal(t, offlineList[i].ListId, allOwnLists[i].ListId)
	}

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
	DeleteTestUser(t)
}

func TestGetAllLists(t *testing.T) {
	log.Print("Testing if all lists can be obtained")
	connectDatabase()

	CreateTestUser(t)
	sharedByUser1, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}
	CreateTestUser(t)
	sharedByUser2, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	CreateTestUser(t)
	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	// Creating two own lists
	var offlineList []data.List
	for i := 0; i < 2; i++ {
		if list, err := createListOffline("own list "+strconv.Itoa(i+1), user.OnlineID, []data.ItemWire{}); err != nil {
			log.Printf("Failed to create list: %s", err)
			t.FailNow()
		} else {
			offlineList = append(offlineList, list)
		}
	}
	// Creating two shared lists from two different IDs
	for i := 0; i < 2; i++ {
		sharedByUser := sharedByUser1
		if i > 1 {
			sharedByUser = sharedByUser2
		}
		list, err := createListOffline("shared list from "+strconv.Itoa(i+1), sharedByUser.OnlineID, []data.ItemWire{})
		if err != nil {
			log.Printf("Failed to created shared list: %s", err)
			t.FailNow()
		}
		offlineList = append(offlineList, list)
		// Create the sharing
		if _, err = createListSharing(list.ListId, list.CreatedBy.ID, user.OnlineID); err != nil {
			log.Printf("Failed to create sharing: %s", err)
			t.FailNow()
		}
	}

	// Now trying if we can get both lists via the API
	login(t)
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/lists/%d", user.OnlineID), nil)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var allLists []data.List
	if json.Unmarshal(w.Body.Bytes(), &allLists) != nil {
		log.Printf("Failed to parse server answer. Expected lists JSON: %s", err)
		t.FailNow()
	}

	assert.Equal(t, 4, len(allLists))
	for i := 0; i < 4; i++ {
		assert.Equal(t, offlineList[i].CreatedBy, allLists[i].CreatedBy)
		assert.Equal(t, roundTime(offlineList[i].LastEdited).Format(time.RFC3339), roundTime(allLists[i].LastEdited).Format(time.RFC3339))
		assert.Equal(t, roundTime(offlineList[i].Created).Format(time.RFC3339), roundTime(allLists[i].Created).Format(time.RFC3339))
		assert.Equal(t, offlineList[i].Name, allLists[i].Name)
		assert.Equal(t, offlineList[i].ListId, allLists[i].ListId)
	}

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
	database.ResetSharedListTable()
	DeleteTestUser(t)
}

func TestGetAllListsWithItems(t *testing.T) {
	log.Print("Testing if all lists can be obtained")
	connectDatabase()

	CreateTestUser(t)
	sharedByUser1, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}
	CreateTestUser(t)
	sharedByUser2, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	CreateTestUser(t)
	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	// Creating two own lists
	var offlineList []data.List
	for i := 0; i < 2; i++ {
		items := createItemsWire("Item", i+1)
		if list, err := createListOffline("own list "+strconv.Itoa(i+1), user.OnlineID, items); err != nil {
			log.Printf("Failed to create list: %s", err)
			t.FailNow()
		} else {
			offlineList = append(offlineList, list)
		}
	}
	// Creating two shared lists from two different IDs
	for i := 0; i < 2; i++ {
		sharedByUser := sharedByUser1
		if i > 1 {
			sharedByUser = sharedByUser2
		}
		items := createItemsWire("Shared Item", i+1)
		list, err := createListOffline("shared list from "+strconv.Itoa(i+1), sharedByUser.OnlineID, items)
		if err != nil {
			log.Printf("Failed to created shared list: %s", err)
			t.FailNow()
		}
		offlineList = append(offlineList, list)
		// Create the sharing
		if _, err = createListSharing(list.ListId, list.CreatedBy.ID, user.OnlineID); err != nil {
			log.Printf("Failed to create sharing: %s", err)
			t.FailNow()
		}
	}
	// Now trying if we can get both lists via the API
	login(t)
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()
	token, err := readJwtFromFile()
	if err != nil {
		log.Printf("Failed to read JWT file: %s", err)
		t.FailNow()
	}
	bearer := "Bearer " + token
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/lists/%d", user.OnlineID), nil)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var allLists []data.List
	if json.Unmarshal(w.Body.Bytes(), &allLists) != nil {
		log.Printf("Failed to parse server answer. Expected lists JSON: %s", err)
		t.FailNow()
	}

	assert.Equal(t, 4, len(allLists))
	for i := 0; i < 4; i++ {
		assert.Equal(t, offlineList[i].CreatedBy, allLists[i].CreatedBy)
		assert.Equal(t, roundTime(offlineList[i].LastEdited).Format(time.RFC3339), roundTime(allLists[i].LastEdited).Format(time.RFC3339))
		assert.Equal(t, roundTime(offlineList[i].Created).Format(time.RFC3339), roundTime(allLists[i].Created).Format(time.RFC3339))
		assert.Equal(t, offlineList[i].Name, allLists[i].Name)
		assert.Equal(t, offlineList[i].ListId, allLists[i].ListId)
		assert.Equal(t, offlineList[i].Items, allLists[i].Items)
		log.Printf("All Lists: %v", allLists[i].Items)
	}

	database.PrintShoppingListTable()
	database.PrintItemTable()
	database.PrintItemPerListTable()
	database.ResetShoppingListTable()
	database.ResetSharedListTable()
	database.ResetItemPerListTable()
	database.ResetItemTable()
	DeleteTestUser(t)
}

func TestRemoveList(t *testing.T) {
	log.Print("Testing if all lists can be removed")
	connectDatabase()
	CreateTestUser(t)
	login(t)

	// Creating with default configuration
	router := server.SetupRouter(cfg)
	w := httptest.NewRecorder()

	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}
	listName := "test list"
	creator := data.ListCreator{
		ID:   user.OnlineID,
		Name: user.Username,
	}
	timeNow := time.Now().UTC()
	list := data.List{
		ListId:     rand.Int63(),
		Name:       listName,
		CreatedBy:  creator,
		Created:    timeNow,
		LastEdited: timeNow,
		Items: []data.ItemWire{
			{
				Name:     "Item",
				Icon:     "ic_item",
				Quantity: 1,
				Checked:  false,
			},
		},
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

	// Now delete this list
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/v1/list/%d", list.ListId), nil)
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check if the list was really deleted
	_, err = database.GetShoppingList(list.ListId, list.CreatedBy.ID)
	assert.NotNil(t, err)

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
	database.ResetItemTable()
	database.ResetItemPerListTable()
	DeleteTestUser(t)
}

func TestCreateSharingWithoutSharedUser(t *testing.T) {
	log.Print("Testing if all lists can be obtained")
	connectDatabase()
	CreateTestUser(t)

	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	// Creating two own lists and share one with a random user
	sharedWithUserId := 12345
	var offlineList []data.List
	for i := 0; i < 2; i++ {
		list, err := createListOffline("own list "+strconv.Itoa(i+1), user.OnlineID, []data.ItemWire{})
		if err != nil {
			log.Printf("Failed to create list: %s", err)
			t.FailNow()
		} else {
			offlineList = append(offlineList, list)
		}
	}

	// Now trying if we can get both lists via the API
	login(t)
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
		ListId:     offlineList[0].ListId,
		CreatedBy:  user.OnlineID,
		SharedWith: int64(sharedWithUserId),
		Created:    time.Now().UTC(),
	}
	encodedShared, err := json.Marshal(sharedWith)
	if err != nil {
		log.Printf("Failed to encoded data: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(encodedShared)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/share/%d", offlineList[0].ListId), reader)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	// we did not create the shared with user and expect this to fail therefore
	assert.Equal(t, http.StatusBadRequest, w.Code)

	database.PrintShoppingListTable()
	database.ResetShoppingListTable()
	database.ResetSharedListTable()
	database.ResetItemPerListTable()
	database.ResetItemTable()
	DeleteTestUser(t)
}

func TestCreateSharing(t *testing.T) {
	log.Print("Testing if all lists can be obtained")
	connectDatabase()

	CreateTestUser(t)
	sharedWithUser, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	CreateTestUser(t)
	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	// Creating two own lists and share one with a random user
	sharedWithUserId := sharedWithUser.OnlineID
	var offlineList []data.List
	for i := 0; i < 2; i++ {
		list, err := createListOffline("own list "+strconv.Itoa(i+1), user.OnlineID, []data.ItemWire{})
		if err != nil {
			log.Printf("Failed to create list: %s", err)
			t.FailNow()
		} else {
			offlineList = append(offlineList, list)
		}
	}

	// Now trying if we can get both lists via the API
	login(t)
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
		ListId:     offlineList[0].ListId,
		CreatedBy:  user.OnlineID,
		SharedWith: int64(sharedWithUserId),
		Created:    time.Now().UTC(),
	}
	encodedShared, err := json.Marshal(sharedWith)
	if err != nil {
		log.Printf("Failed to encoded data: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(encodedShared)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/share/%d", offlineList[0].ListId), reader)
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
	DeleteTestUser(t)
}

func TestCreateSharingOfUnownedList(t *testing.T) {
	log.Print("Testing if all lists can be obtained")
	connectDatabase()
	CreateTestUser(t)

	user, err := readUserFile()
	if err != nil {
		log.Printf("Cannot read user file: %s", err)
		t.FailNow()
	}

	// Creating a list that WE DO NOT OWN
	list, err := createListOffline("unowned list 1", user.OnlineID+1, []data.ItemWire{})
	if err != nil {
		log.Printf("Failed to create list: %s", err)
		t.FailNow()
	}

	// Now trying if we can share the list via the API
	login(t)
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
		ListId:     list.ListId,
		CreatedBy:  user.OnlineID,
		SharedWith: int64(sharedWithUserId),
		Created:    time.Now().UTC(),
	}
	encodedShared, err := json.Marshal(sharedWith)
	if err != nil {
		log.Printf("Failed to encoded data: %s", err)
		t.FailNow()
	}
	reader := bytes.NewReader(encodedShared)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/share/%d", list.ListId), reader)
	// Adding the authentication
	req.Header.Add("Authorization", bearer)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	database.PrintSharingTable()
	database.ResetShoppingListTable()
	database.ResetSharedListTable()
}
