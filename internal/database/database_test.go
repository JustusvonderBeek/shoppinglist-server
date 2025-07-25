package database

import (
	"testing"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/configuration"
)

// ------------------------------------------------------------
// Connect the test to the database: required
// ------------------------------------------------------------

func connectDatabase() {
	cfg := configuration.Config{
		Database: "../resources/db.json",
	}
	CheckDatabaseOnline(cfg)
}

func TestPrinting(t *testing.T) {
	connectDatabase()
	PrintShoppingListTable()
}

// ------------------------------------------------------------
// Testing the user in : user_test
// ------------------------------------------------------------

// ------------------------------------------------------------
// Testing the list handling: list_test
// ------------------------------------------------------------

// ------------------------------------------------------------
// Testing the mapping handling: mapping_test
// ------------------------------------------------------------

// ------------------------------------------------------------
// Testing items in: item_test
// ------------------------------------------------------------
