package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/go-sql-driver/mysql"
	"shop.cloudsheeptech.com/server/configuration"
	"shop.cloudsheeptech.com/server/data"
)

// A small database wrapper allowing to access a MySQL database

// ------------------------------------------------------------
// Configuration File Handling
// ------------------------------------------------------------

var config DBConf
var db *sql.DB

type DBConf struct {
	DBUser string
	DBPass string
	Addr   string
	DBName string
}

func createDefaultConfiguration(confFile string) {
	// This method is only meant to create the file in the right format
	// It is not meant to create a config file holding a working configuration
	conf := DBConf{
		DBUser: "",
		DBPass: "",
		Addr:   "127.0.0.1:3306",
		DBName: "shoppinglist",
	}
	config = conf
	storeConfiguration(confFile)
}

func loadConfig(confFile string) {
	if confFile == "" {
		log.Fatal("Cannot load database configuration")
	}
	content, err := os.ReadFile(confFile)
	if err != nil {
		createDefaultConfiguration(confFile)
		log.Fatalf("Failed to read database configuration file: %s", err)
	}
	var configuration DBConf
	err = json.Unmarshal(content, &configuration)
	if err != nil {
		log.Fatalf("Configuration file '%s' not in correct format: %s", confFile, err)
	}
	config = configuration
	log.Printf("Successfully loaded configuration from '%s'", confFile)
}

func storeConfiguration(confFile string) {
	if confFile == "" {
		log.Fatal("Cannot store configuration file due to empty location")
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("Failed to convert configuration to file format")
	}
	err = os.WriteFile(confFile, encoded, 0660)
	if err != nil {
		log.Fatalf("Failed to store configuration to file: %s", err)
	}
	log.Printf("Stored configuration to file: %s", confFile)
}

// ------------------------------------------------------------

func CheckDatabaseOnline(cfg configuration.Config) {
	if config == (DBConf{}) {
		log.Print("Configuration not initialized")
		loadConfig(cfg.DatabaseConfig)
	}
	if db != nil {
		log.Print("Already connected to database")
		return
	}
	mysqlCfg := mysql.Config{
		User:                 config.DBUser,
		Passwd:               config.DBPass,
		Net:                  "tcp",
		Addr:                 config.Addr,
		DBName:               config.DBName,
		AllowNativePasswords: true,
	}
	var err error
	db, err = sql.Open("mysql", mysqlCfg.FormatDSN())
	if err != nil {
		log.Fatalf("Cannot connect to database: %s", err)
	}
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatalf("Database not responding: %s", pingErr)
	}
	log.Print("Connected to database")
}

// ------------------------------------------------------------
// The data structs and constants for the user handling
// ------------------------------------------------------------

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

const userTable = "shoppers"

func GetUser(id int64) (data.User, error) {
	query := "SELECT * FROM " + userTable + " WHERE id = ?"
	row := db.QueryRow(query, id)
	var user data.User
	if err := row.Scan(&user.ID, &user.Username, &user.Passwd); err == sql.ErrNoRows {
		return data.User{}, err
	}
	return user, nil
}

func CheckUserExists(id int64) error {
	log.Printf("Checking if user exists")
	_, err := GetUser(id)
	return err
}

func CreateUserAccount(username string, passwd string) (data.User, error) {
	log.Printf("Creating new user account: %s", username)
	// Creating the struct we are going to insert first
	userId := random.Int63()
	for {
		err := CheckUserExists(int64(userId))
		if err == nil { // User already exists
			userId = rand.Int63()
			continue
		} else {
			break
		}
	}
	// Hashing the password
	hashedPw, err := argon2id.CreateHash(passwd, argon2id.DefaultParams)
	if err != nil {
		log.Printf("Failed to hash password: %s", err)
		return data.User{}, err
	}
	newUser := data.User{
		ID:       userId,
		Username: username,
		Passwd:   hashedPw,
	}
	// Insert the newly created user into the database
	query := "INSERT INTO " + userTable + " (id, username, passwd) VALUES (?, ?, ?)"
	_, err = db.Exec(query, newUser.ID, newUser.Username, newUser.Passwd)
	if err != nil {
		log.Printf("Failed to insert new user into database: %s", err)
		return data.User{}, err
	}
	return newUser, nil
}

func ModifyUserAccountName(id int64, username string) (data.User, error) {
	log.Printf("Modifying user with ID %d", id)
	user, err := GetUser(id)
	if err != nil {
		log.Printf("Failed to find user with ID %d", id)
		return data.User{}, err
	}
	user.Username = username
	query := "UPDATE " + userTable + " SET username = ? WHERE id = ?"
	_, err = db.Exec(query, user.Username, user.ID)
	if err != nil {
		log.Printf("Failed to update user with ID %d", id)
		return data.User{}, err
	}
	return user, nil
}

func ModifyUserAccountPassword(id int64, password string) (data.User, error) {
	log.Printf("Modifying user with ID %d", id)
	user, err := GetUser(id)
	if err != nil {
		log.Printf("Failed to find user with ID %d", id)
		return data.User{}, err
	}
	hashedPw, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Printf("Failed to hash given password: %s", err)
		return data.User{}, err
	}
	user.Passwd = hashedPw
	query := "UPDATE " + userTable + " SET passwd = ? WHERE id = ?"
	_, err = db.Exec(query, user.Passwd, user.ID)
	if err != nil {
		log.Printf("Failed to update user with ID %d", id)
		return data.User{}, err
	}
	return user, nil
}

func DeleteUserAccount(id int64) error {
	_, err := db.Exec("DELETE FROM shoppers WHERE id = ?", id)
	if err != nil {
		log.Printf("Failed to delete user with id %d", id)
		return err
	}
	return nil
}

func ResetUserTable() {
	log.Print("RESETTING ALL USERS. THIS DISABLES LOGIN FOR ALL EXISTING USERS")

	query := "DELETE FROM " + userTable
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to delete user table: %s", err)
		return
	}

	log.Print("RESET LOGIN USER TABLE")
}

// ------------------------------------------------------------
// The shopping list handling
// ------------------------------------------------------------

const shoppingListTable = "shoppinglist"

func GetShoppingList(id int64) (data.Shoppinglist, error) {
	query := "SELECT * FROM " + shoppingListTable + " WHERE id = ?"
	row := db.QueryRow(query, id)
	var list data.Shoppinglist
	if err := row.Scan(&list.ID, &list.Name, &list.CreatedBy); err == sql.ErrNoRows {
		return data.Shoppinglist{}, err
	}
	return list, nil
}

func GetShoppingListFromUserId(id int64) (data.Shoppinglist, error) {
	query := "SELECT * FROM " + shoppingListTable + " WHERE creatorId = ?"
	row := db.QueryRow(query, id)
	var list data.Shoppinglist
	if err := row.Scan(&list.ID, &list.Name, &list.CreatedBy); err == sql.ErrNoRows {
		return data.Shoppinglist{}, err
	}
	return list, nil
}

func CreateShoppingList(name string, createdBy int64) (data.Shoppinglist, error) {
	log.Printf("Creating shopping list %s from %d", name, createdBy)
	list := data.Shoppinglist{
		ID:        0,
		Name:      name,
		CreatedBy: createdBy,
	}
	query := "INSERT INTO " + shoppingListTable + " (id, name, creatorId) VALUES (?, ?, ?)"
	result, err := db.Exec(query, list.ID, list.Name, list.CreatedBy)
	if err != nil {
		log.Printf("Failed to create list into database: %s", err)
		return data.Shoppinglist{}, err
	}
	list.ID, err = result.LastInsertId()
	if err != nil {
		log.Printf("Problem with ID during insertion of shopping list: %s", err)
		return data.Shoppinglist{}, err
	}
	return list, err
}

func ModifyShoppingListName(id int64, name string) (data.Shoppinglist, error) {
	log.Printf("Modifying list %d", id)
	list, err := GetShoppingList(id)
	if err != nil {
		log.Printf("Failed to get list with ID %s", err)
		return data.Shoppinglist{}, err
	}
	list.Name = name
	query := "UPDATE " + shoppingListTable + " SET name = ? WHERE id = ?"
	result, err := db.Exec(query, list.Name, list.ID)
	if err != nil {
		log.Printf("Failed to update list name: %s", err)
		return data.Shoppinglist{}, err
	}
	list.ID, err = result.LastInsertId()
	if err != nil {
		log.Printf("Problem with ID during insertion of shopping list: %s", err)
		return data.Shoppinglist{}, err
	}
	return list, err
}

func DeleteShoppingList(id int64) error {
	query := "DELETE FROM " + shoppingListTable + " WHERE id = ?"
	_, err := db.Exec(query, id)
	if err != nil {
		log.Printf("Failed to delete shopping list with id %d", id)
		return err
	}
	return nil
}

func ResetShoppingListTable() {
	log.Print("RESETTING ALL SHOPPING LISTS. CANNOT BE REVERTED!")

	query := "DELETE FROM " + shoppingListTable
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to delete shopping list table: %s", err)
		return
	}

	log.Print("RESET SHOPPING TABLE")
}

type Item struct {
	ID    int64
	Name  string
	Image string
}

type Mapping struct {
	ID       int64
	ListId   int64
	ItemId   int64
	Quantity int64
	Checked  bool
}

type UserIdToMapping struct {
	ID          int64
	UserId      int64
	ListId      int64
	ForEveryone bool
}

// ------------------------------------------------------------

func GetItem(id int) (Item, error) {
	if id < 0 {
		err := errors.New("items with id < 0 do not exist")
		return Item{}, err
	}
	var item Item
	row := db.QueryRow("SELECT * FROM items WHERE id = ?", id)
	// Looping through data, assigning the columns to the given struct
	if err := row.Scan(&item.ID, &item.Name, &item.Image); err != nil {
		return Item{}, err
	}
	return item, nil
}

func GetAllItems() ([]Item, error) {
	var items []Item
	rows, err := db.Query("SELECT * FROM items")
	if err != nil {
		log.Printf("Failed to query database for item: %s", err)
		return nil, err
	}
	defer rows.Close()
	// Looping through data, assigning the columns to the given struct
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Image); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Failed to retrieve data from database: %s", err)
		return nil, err
	}
	return items, nil
}

func InsertItem(item Item) (int64, error) {
	result, err := db.Exec("INSERT INTO items (name, image) VALUES (?, ?)", item.Name, item.Image)
	if err != nil {
		log.Printf("Failed to insert item into database: %s", err)
		return -1, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to insert item into database: %s", err)
		return -1, err
	}
	return id, nil
}

// ------------------------------------------------------------

func GetMapping(id int) (Mapping, error) {
	if id < 0 {
		err := errors.New("mapping with id < 0 do not exist")
		return Mapping{}, err
	}
	var mapping Mapping
	row := db.QueryRow("SELECT * FROM shoppinglists WHERE id = ?", id)
	// Looping through data, assigning the columns to the given struct
	if err := row.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity); err != nil {
		return Mapping{}, err
	}
	return mapping, nil
}

func GetMappingWithUserId(id int) ([]Mapping, error) {
	if id < 0 {
		err := errors.New("mapping with user id < 0 do not exist")
		return []Mapping{}, err
	}
	var mappings []Mapping
	rows, err := db.Query("SELECT * FROM listaccess WHERE userId = ?", id)
	if err != nil {
		log.Printf("Failed to query for user id %d", id)
		return []Mapping{}, err
	}
	// Looping through data, assigning the columns to the given struct
	for rows.Next() {
		var mapping Mapping
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked); err != nil {
			return []Mapping{}, err
		}
		mappings = append(mappings, mapping)
	}
	return mappings, nil
}

func GetMappingWithListId(id int) ([]Mapping, error) {
	if id < 0 {
		err := errors.New("mapping with id < 0 do not exist")
		return []Mapping{}, err
	}
	var mappings []Mapping
	rows, err := db.Query("SELECT * FROM shoppinglists WHERE listId = ?", id)
	if err != nil {
		log.Printf("Failed to query for list id %d", id)
		return []Mapping{}, err
	}
	// Looping through data, assigning the columns to the given struct
	for rows.Next() {
		var mapping Mapping
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked); err != nil {
			return []Mapping{}, err
		}
		mappings = append(mappings, mapping)
	}
	return mappings, nil
}

func InsertMapping(mapping Mapping) (int64, error) {
	result, err := db.Exec("INSERT INTO shoppinglists (listId, itemId, quantity) VALUES (?, ?, ?, ?)", mapping.ListId, mapping.ItemId, mapping.Quantity, mapping.Checked)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		return -1, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		return -1, err
	}
	return id, nil
}

// ------------------------------------------------------------
// Debug printout and functionality
// ------------------------------------------------------------

func PrintUserTable(tableName string) {
	rows, err := db.Query("SELECT * FROM shoppers")
	if err != nil {
		log.Printf("Failed to print table %s: %s", tableName, err)
		return
	}
	log.Print("------------- User Table -------------")
	for rows.Next() {
		var user data.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Passwd); err != nil {
			log.Printf("Failed to print table: %s: %s", tableName, err)
		}
		log.Printf("%v", user)
	}
	log.Print("---------------------------------------")
}

func PrintShoppingListTable() {
	query := "SELECT * FROM " + shoppingListTable
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to print table %s: %s", shoppingListTable, err)
		return
	}
	log.Print("------------- Shopping List Table -------------")
	for rows.Next() {
		var list data.Shoppinglist
		if err := rows.Scan(&list.ID, &list.Name, &list.CreatedBy); err != nil {
			log.Printf("Failed to print table: %s: %s", shoppingListTable, err)
		}
		log.Printf("%v", list)
	}
	log.Print("---------------------------------------")
}
