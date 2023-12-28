package database_test

import (
	"crypto/sha1"
	"encoding/base64"
	"log"
	"strconv"
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

// ------------------------------------------------------------
// Testing user creation and authentication
// ------------------------------------------------------------

func TestInsertUser(t *testing.T) {
	connectDatabase()
	user := data.User{
		ID:       12,
		Username: strconv.Itoa(32),
		Passwd:   "Biene Maja",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Passwd)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	match, err := argon2id.ComparePasswordAndHash(user.Passwd, createdUser.Passwd)
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
		Passwd:   "Biene Maja",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Passwd)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	match, err := argon2id.ComparePasswordAndHash(user.Passwd, createdUser.Passwd)
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
		Passwd:   "Secure Password 123",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Passwd)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	match, err := argon2id.ComparePasswordAndHash(user.Passwd, createdUser.Passwd)
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
	match, err = argon2id.ComparePasswordAndHash(user.Passwd, checkLoginUser.Passwd)
	if err != nil {
		log.Printf("Failed to compare password and hash: %s", err)
		t.FailNow()
	}
	if !match {
		log.Print("Password and hash do not match even though they should!")
		t.FailNow()
	}
	match, _ = argon2id.ComparePasswordAndHash("Secure Password 12", checkLoginUser.Passwd)
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
		Passwd:   "Biene Maja",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Passwd)
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
		Passwd:   "Old Password",
	}
	createdUser, err := database.CreateUserAccount(user.Username, user.Passwd)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	checkOldPassword, err := database.GetUser(createdUser.ID)
	if err != nil {
		log.Printf("Failed to get newly created user for modify check: %s", err)
		t.FailNow()
	}
	if checkOldPassword.Passwd != createdUser.Passwd {
		log.Print("Password do not match before update!")
		t.FailNow()
	}
	updatedUser, err := database.ModifyUserAccountPassword(createdUser.ID, "New Password")
	if err != nil {
		log.Printf("Failed to update password: %s", err)
		t.FailNow()
	}
	database.PrintUserTable("shoppers")
	match, err := argon2id.ComparePasswordAndHash("New Password", updatedUser.Passwd)
	if err != nil || !match {
		log.Print("The password was not correctly updated!")
		t.FailNow()
	}
	checkNewPassword, err := database.GetUser(createdUser.ID)
	if err != nil {
		log.Printf("Failed to get updated user: %s", err)
		t.FailNow()
	}
	match, err = argon2id.ComparePasswordAndHash("New Password", checkNewPassword.Passwd)
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
	loginUser, err := database.GetUser(int64(userId))
	if err != nil {
		log.Printf("Failed to retrieve user with id %d", userId)
		t.FailNow()
	}
	// passwd := "schlechtes wetter"
	hasher := sha1.New()
	// hasher.Write([]byte(passwd + loginUser.Salt))
	hashedPwd := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	if loginUser.Passwd != hashedPwd {
		log.Print("Given password does not match the stored password!")
		t.FailNow()
	}
	log.Print("User correctly stored and retrieved")
}
