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
// The data structs and constants for the user handling
// ------------------------------------------------------------

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

const userTable = "shoppers"

func GetUser(id int64) (data.User, error) {
	query := "SELECT * FROM " + userTable + " WHERE id = ?"
	row := db.QueryRow(query, id)
	var user data.User
	if err := row.Scan(&user.ID, &user.Username, &user.Passwd, &user.Salt); err == sql.ErrNoRows {
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
		Salt:     "",
	}
	// Insert the newly created user into the database
	query := "INSERT INTO " + userTable + " (id, username, passwd, salt) VALUES (?, ?, ?, ?)"
	_, err = db.Exec(query, newUser.ID, newUser.Username, newUser.Passwd, newUser.Salt)
	if err != nil {
		log.Printf("Failed to insert new user into database: %s", err)
		return data.User{}, err
	}
	return newUser, nil
}

func DeleteUserAccount(id int64) error {
	_, err := db.Exec("DELET FROM shoppers WHERE id = ?", id)
	if err != nil {
		log.Printf("Failed to delete user with id %d", id)
		return err
	}
	return nil
}

func ResetUserTable() {
	log.Print("RESETTING ALL USERS. THIS DISABLES LOGIN FOR ALL EXISTING USERS")

	_, err := db.Exec("DELETE FROM shoppers")
	if err != nil {
		log.Printf("Failed to delete user table: %s", err)
		return
	}

	log.Print("RESET LOGIN USER TABLE")
}

// ------------------------------------------------------------
// The data structs for the queries
// ------------------------------------------------------------

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

// func GetUser(id int) (User, error) {
// 	if id < 0 {
// 		err := errors.New("users with id < 0 do not exist")
// 		return User{}, err
// 	}
// 	var user User
// 	row := db.QueryRow("SELECT * FROM shoppers WHERE id = ?", id)
// 	// Looping through data, assigning the columns to the given struct
// 	if err := row.Scan(&user.ID, &user.Name, &user.FavRecipe); err != nil {
// 		return User{}, err
// 	}
// 	return user, nil
// }

// func InsertUser(user User) (int64, error) {
// 	result, err := db.Exec("INSERT INTO shoppers (name, favRecipe) VALUES (?, ?)", user.Name, user.FavRecipe)
// 	if err != nil {
// 		log.Printf("Failed to insert user into database: %s", err)
// 		return -1, err
// 	}
// 	id, err := result.LastInsertId()
// 	if err != nil {
// 		log.Printf("Failed to insert user into database: %s", err)
// 		return -1, err
// 	}
// 	return id, nil
// }

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
	rows, err := db.Query("SELECT * FROM loginuser")
	if err != nil {
		log.Printf("Failed to print table %s: %s", tableName, err)
		return
	}
	log.Print("------------- User Table -------------")
	for rows.Next() {
		var user data.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Passwd, &user.Salt); err != nil {
			log.Printf("Failed to print table: %s: %s", tableName, err)
		}
		log.Printf("%v", user)
	}
	log.Print("---------------------------------------")
}
