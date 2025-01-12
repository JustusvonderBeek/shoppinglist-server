package server

import (
	"github.com/JustusvonderBeek/shoppinglist-server/internal/configuration"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func connectDatabase() {
	cfg := configuration.Config{
		DatabaseConfig: "../../resources/db.json",
	}
	database.CheckDatabaseOnline(cfg)
}

func TestUserCreationWithCorrectData(t *testing.T) {
	connectDatabase()
	newUser := data.User{
		OnlineID: 0,
		Username: "test creation user",
		Password: "new password",
	}
	createdUser, err := validateUserAndCreateAccount(newUser, "")
	if err != nil {
		log.Printf("Creating user failed: %s", err)
		t.FailNow()
	}

	assert.Equal(t, newUser.Username, createdUser.Username)
	assert.NotEqual(t, 0, createdUser.OnlineID)
	assert.NotEqual(t, newUser.Password, createdUser.Password)
}

func TestUserCreationWithWrongData(t *testing.T) {
	connectDatabase()
	newUser := data.User{
		OnlineID: 12,
		Username: "test creation user",
		Password: "",
	}
	_, err := validateUserAndCreateAccount(newUser, "")
	if err == nil {
		log.Printf("Creating user did not fail with malicious data")
		t.FailNow()
	}
	assert.NotNil(t, err)
}

func TestCreatingAdminUser(t *testing.T) {
	connectDatabase()
	newUser := data.User{
		OnlineID: 0,
		Username: "admin",
		Password: "admin_password",
	}
	_, err := validateUserAndCreateAccount(newUser, "")
	if err == nil {
		log.Printf("Creating admin user without API key did not fail")
		t.FailNow()
	}
	assert.NotNil(t, err)
}
