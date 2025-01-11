package database

import (
	"log"
	"testing"
	"time"

	"github.com/alexedwards/argon2id"
)

// ------------------------------------------------------------
// Connect the test to the database: required, found in database_test
// ------------------------------------------------------------

// ------------------------------------------------------------
// Testing user creation and authentication
// ------------------------------------------------------------

func TestCreatingUser(t *testing.T) {
	connectDatabase()
	username := "test user 123 🐧"
	password := "password is secure"
	user, err := createUser(username, password)
	if err != nil {
		log.Printf("User creation failed: %s", err)
		t.FailNow()
	}
	if user.Username != username {
		log.Printf("Usernames do not match")
		t.FailNow()
	}
	if match, err := argon2id.ComparePasswordAndHash(password, user.Password); err != nil || !match {
		log.Printf("Password does not match")
		t.FailNow()
	}
	createdTime := user.Created
	// createdTime, err := time.Parse(time.RFC3339Nano, user.Created)
	// if err != nil {
	// 	log.Printf("Created un-parseable time format!")
	// 	t.FailNow()
	// }
	now := time.Now().Local()
	if now.Before(createdTime) {
		log.Printf("Created time (%s) is after the current time (%s)", createdTime, now)
		t.FailNow()
	}
	if user.Created != user.LastLogin {
		log.Printf("Created time and last login do not match")
		t.FailNow()
	}
	log.Printf("Creation was success")
}

func TestCreatingIncorrectUser(t *testing.T) {
	connectDatabase()
	username := "test user 123 🐧"
	password := "password is secure"
	user, err := createUser("", password)
	if err == nil {
		log.Printf("Expected error but got none")
		t.FailNow()
	}
	if user.OnlineID != 0 || user.Username != "" || user.Password != "" {
		log.Printf("Expected empty user but got: %v", user)
		t.FailNow()
	}
	user, err = createUser(username, "")
	if err == nil {
		log.Printf("Expected error but got none")
		t.FailNow()
	}
	if user.OnlineID != 0 || user.Username != "" || user.Password != "" {
		log.Printf("Expected empty user but got: %v", user)
		t.FailNow()
	}
	log.Printf("Incorrect creation was success")
}

func TestInsertUser(t *testing.T) {
	connectDatabase()
	newUser, _ := createUser("new user", "new secure password")
	_, err := CreateUserAccountInDatabase(newUser.Username, newUser.Password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	log.Print("InsertUser successfully completed")
	PrintUserTable("shoppers")
	ResetUserTable()
}

func TestDeletingUser(t *testing.T) {
	connectDatabase()
	password := "password"
	user, _ := createUser("username", password)
	createdUser, err := CreateUserAccountInDatabase(user.Username, password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	match, err := argon2id.ComparePasswordAndHash(password, createdUser.Password)
	if err != nil {
		log.Printf("Password and hash do not match: %s", err)
		t.FailNow()
	}
	if createdUser.Username != user.Username || !match {
		log.Printf("User not correctly inserted")
		t.FailNow()
	}
	PrintUserTable("shoppers")
	err = DeleteUserAccount(createdUser.OnlineID)
	if err != nil {
		log.Printf("Failed to delete user with id %d from database", createdUser.OnlineID)
		t.FailNow()
	}
	deletedUser, err := GetUser(createdUser.OnlineID)
	if err == nil || deletedUser.OnlineID != 0 {
		log.Print("Could retrieve user from database after deleting!")
		t.FailNow()
	}
	log.Print("DeleteUser successfully completed")
	PrintUserTable("shoppers")
	ResetUserTable()
}

func TestUserLogin(t *testing.T) {
	connectDatabase()
	password := "very secure password"
	user, _ := createUser("test user login", password)
	createdUser, err := CreateUserAccountInDatabase(user.Username, password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	match, err := argon2id.ComparePasswordAndHash(password, createdUser.Password)
	if err != nil {
		log.Printf("Password and hash do not match: %s", err)
		t.FailNow()
	}
	if createdUser.Username != user.Username || !match {
		log.Printf("User not correctly inserted")
		t.FailNow()
	}
	PrintUserTable("shoppers")
	checkLoginUser, err := GetUser(createdUser.OnlineID)
	if err != nil {
		log.Printf("Failed to get newly created user for login check: %s", err)
		t.FailNow()
	}
	match, err = argon2id.ComparePasswordAndHash(password, checkLoginUser.Password)
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
	ResetUserTable()
}

func TestModifyUsername(t *testing.T) {
	connectDatabase()
	password := "very secure password"
	user, _ := createUser("modify username user", password)
	createdUser, err := CreateUserAccountInDatabase(user.Username, password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	checkOldUsername, err := GetUser(createdUser.OnlineID)
	if err != nil {
		log.Printf("Failed to get newly created user for modify check: %s", err)
		t.FailNow()
	}
	if checkOldUsername.Username != user.Username {
		log.Print("Usernames do not match before checking!")
		t.FailNow()
	}
	updatedUsername, err := ModifyUserAccountName(createdUser.OnlineID, user.Username+" - Updated")
	if err != nil {
		log.Printf("Failed to update username: %s", err)
		t.FailNow()
	}
	PrintUserTable("shoppers")
	if updatedUsername.Username == checkOldUsername.Username {
		log.Print("The updated username is still the same!")
		t.FailNow()
	}
	checkNewUsername, err := GetUser(createdUser.OnlineID)
	if err != nil {
		log.Printf("Failed to get updated user: %s", err)
		t.FailNow()
	}
	if checkNewUsername.Username != user.Username+" - Updated" {
		log.Print("Usernames do not match!")
		t.FailNow()
	}
	log.Print("TestModifyUsername successfully completed")
	ResetUserTable()
}

func TestModifyUserPassword(t *testing.T) {
	connectDatabase()
	password := "very secure password"
	user, _ := createUser("modify password user", password)
	createdUser, err := CreateUserAccountInDatabase(user.Username, password)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		t.FailNow()
	}
	checkOldPassword, err := GetUser(createdUser.OnlineID)
	if err != nil {
		log.Printf("Failed to get newly created user for modify check: %s", err)
		t.FailNow()
	}
	if checkOldPassword.Password != createdUser.Password {
		log.Print("Password do not match before update!")
		t.FailNow()
	}
	updatedUser, err := ModifyUserAccountPassword(createdUser.OnlineID, "New Password")
	if err != nil {
		log.Printf("Failed to update password: %s", err)
		t.FailNow()
	}
	PrintUserTable("shoppers")
	match, err := argon2id.ComparePasswordAndHash("New Password", updatedUser.Password)
	if err != nil || !match {
		log.Print("The password was not correctly updated!")
		t.FailNow()
	}
	checkNewPassword, err := GetUser(createdUser.OnlineID)
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
	ResetUserTable()
}
