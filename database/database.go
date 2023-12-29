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
	if err := row.Scan(&user.ID, &user.Username, &user.Password); err == sql.ErrNoRows {
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
	userId := random.Int31()
	for {
		err := CheckUserExists(int64(userId))
		if err == nil { // User already exists
			userId = rand.Int31()
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
		ID:       int64(userId),
		Username: username,
		Password: hashedPw,
	}
	log.Printf("Inserting %v", newUser)
	// Insert the newly created user into the database
	query := "INSERT INTO " + userTable + " (id, username, passwd) VALUES (?, ?, ?)"
	_, err = db.Exec(query, newUser.ID, newUser.Username, newUser.Password)
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
	user.Password = hashedPw
	query := "UPDATE " + userTable + " SET passwd = ? WHERE id = ?"
	_, err = db.Exec(query, user.Password, user.ID)
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
	DeleteAllSharingForUser(id)
	return DeleteShoppingListFrom(id)
}

func ResetUserTable() {
	log.Print("RESETTING ALL USERS. THIS DISABLES LOGIN FOR ALL EXISTING USERS")

	query := "DELETE FROM " + userTable
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to reset user table: %s", err)
		return
	}

	log.Print("RESET USER TABLE")
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
	query := "INSERT INTO " + shoppingListTable + " (name, creatorId) VALUES (?, ?)"
	result, err := db.Exec(query, list.Name, list.CreatedBy)
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
		log.Printf("Failed to get list with ID %d", id)
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

func DeleteShoppingListFrom(userId int64) error {
	query := "DELETE FROM " + shoppingListTable + " WHERE creatorId = ?"
	_, err := db.Exec(query, userId)
	if err != nil {
		log.Printf("Failed to delete shopping lists for user %d: %s", userId, err)
		return err
	}
	return nil
}

func ResetShoppingListTable() {
	log.Print("RESETTING ALL SHOPPING LISTS. CANNOT BE REVERTED!")

	query := "DELETE FROM " + shoppingListTable
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to reset shopping list table: %s", err)
		return
	}

	log.Print("RESET SHOPPING TABLE")
}

// ------------------------------------------------------------

const sharedListTable = "sharedList"

func GetSharedListFromListId(listId int64) ([]data.ListShared, error) {
	query := "SELECT * FROM " + sharedListTable + " WHERE listId = ?"
	rows, err := db.Query(query, listId)
	if err != nil {
		log.Printf("Failed to query for users that get shared list %d: %s", listId, err)
		return []data.ListShared{}, nil
	}
	var list []data.ListShared
	for rows.Next() {
		var shared data.ListShared
		if err := rows.Scan(&shared.ID, &shared.ListId, &shared.SharedWith); err != nil {
			log.Printf("Failed to query table: %s: %s", sharedListTable, err)
			return []data.ListShared{}, err
		}
		list = append(list, shared)
	}
	return list, nil
}

func GetSharedListForUserId(userId int64) ([]data.ListShared, error) {
	query := "SELECT * FROM " + sharedListTable + " WHERE sharedWithId = ?"
	rows, err := db.Query(query, userId)
	if err != nil {
		log.Printf("Failed to query for lists that are shared with the user %d: %s", userId, err)
		return []data.ListShared{}, nil
	}
	var list []data.ListShared
	for rows.Next() {
		var shared data.ListShared
		if err := rows.Scan(&shared.ID, &shared.ListId, &shared.SharedWith); err != nil {
			log.Printf("Failed to query table: %s: %s", sharedListTable, err)
			return []data.ListShared{}, err
		}
		list = append(list, shared)
	}
	return list, nil
}

func CreateSharedList(listId int64, sharedWith int64) (data.ListShared, error) {
	query := "INSERT INTO " + sharedListTable + " (listId, sharedWithId) VALUES (?, ?)"
	shared := data.ListShared{
		ID:         0,
		ListId:     listId,
		SharedWith: sharedWith,
	}
	result, err := db.Exec(query, shared.ListId, shared.SharedWith)
	if err != nil {
		log.Printf("Failed to insert sharing into database: %s", err)
		return data.ListShared{}, err
	}
	shared.ID, err = result.LastInsertId()
	if err != nil || shared.ID == 0 {
		log.Printf("Failed to insert mapping into database: %s", err)
		return data.ListShared{}, err
	}
	return shared, nil
}

func DeleteSharingOfList(listId int64) error {
	query := "DELETE FROM " + sharedListTable + " WHERE listId = ?"
	_, err := db.Exec(query, listId)
	if err != nil {
		log.Printf("Failed to delete sharing of list %d: %s", listId, err)
		return err
	}
	return nil
}

func DeleteSharingForUser(listId int64, userId int64) error {
	query := "DELETE FROM " + sharedListTable + " WHERE listId = ? AND sharedWithId = ?"
	_, err := db.Exec(query, listId, userId)
	if err != nil {
		log.Printf("Failed to delete sharing for user %d of list %d: %s", userId, listId, err)
		return err
	}
	return nil
}

func DeleteAllSharingForUser(userId int64) error {
	query := "DELETE FROM " + sharedListTable + " WHERE sharedWithId = ?"
	_, err := db.Exec(query, userId)
	if err != nil {
		log.Printf("Failed to delete sharing for user %d: %s", userId, err)
		return err
	}
	return nil
}

func ResetSharedList() {
	log.Print("RESETTING SHARING LIST. CANNOT BE REVERTED!")

	query := "DELETE FROM " + sharedListTable
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to remove all sharing from table: %s", err)
		return
	}

	log.Print("RESET SHARING TABLE")
}

// ------------------------------------------------------------

const itemPerListTable = "itemsPerList"

// Returns the lists in which the item with the given ID is included
func GetListsOfItem(itemId int64) ([]data.ItemPerList, error) {
	var lists []data.ItemPerList
	query := "SELECT * FROM " + itemPerListTable + " WHERE itemId = ?"
	rows, err := db.Query(query, itemId)
	if err != nil {
		log.Printf("Failed to query for lists containing item %d: %s", itemId, err)
		return []data.ItemPerList{}, nil
	}
	for rows.Next() {
		var mapping data.ItemPerList
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.AddedBy); err != nil {
			log.Printf("Failed to query table: %s: %s", itemPerListTable, err)
			return []data.ItemPerList{}, err
		}
		lists = append(lists, mapping)
	}
	return lists, nil
}

// Returns the items in a specific list
func GetItemsInList(listId int64) ([]data.ItemPerList, error) {
	var list []data.ItemPerList
	query := "SELECT * FROM " + itemPerListTable + " WHERE listId = ?"
	rows, err := db.Query(query, listId)
	if err != nil {
		log.Printf("Failed to query for items contained in list %d: %s", listId, err)
		return []data.ItemPerList{}, nil
	}
	for rows.Next() {
		var mapping data.ItemPerList
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.AddedBy); err != nil {
			log.Printf("Failed to query table: %s: %s", itemPerListTable, err)
			return []data.ItemPerList{}, err
		}
		list = append(list, mapping)
	}
	return list, nil
}

func InsertItemToList(mapping data.ItemPerList) (data.ItemPerList, error) {
	query := "INSERT INTO " + itemPerListTable + " (listId, itemId, quantity, checked, addedBy) VALUES (?, ?, ?, ?, ?)"
	result, err := db.Exec(query, mapping.ListId, mapping.ItemId, mapping.Quantity, mapping.Checked, mapping.AddedBy)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		return data.ItemPerList{}, err
	}
	mapping.ID, err = result.LastInsertId()
	if err != nil || mapping.ID == 0 {
		log.Printf("Failed to insert mapping into database: %s", err)
		return data.ItemPerList{}, err
	}
	return mapping, nil
}

func DeleteItemInList(itemId int64, listId int64) error {
	query := "DELETE FROM " + itemPerListTable + " WHERE itemId = ? AND listId = ?"
	_, err := db.Exec(query, itemId, listId)
	if err != nil {
		log.Printf("Failed to delete item %d in list: %s", itemId, err)
		return err
	}
	return nil
}

// This method should only be used if an item is deleted
func DeleteItemInAllLists(itemId int64) error {
	query := "DELETE FROM " + itemPerListTable + " WHERE itemId = ?"
	_, err := db.Exec(query, itemId)
	if err != nil {
		log.Printf("Failed to delete item %d from all lists: %s", itemId, err)
		return err
	}
	return nil
}

func DeleteAllItemsInList(listId int64) error {
	query := "DELETE FROM " + itemPerListTable + " WHERE listId = ?"
	_, err := db.Exec(query, listId)
	if err != nil {
		log.Printf("Failed to delete list %d: %s", listId, err)
		return err
	}
	return nil
}

func ResetItemPerListTable() {
	log.Print("RESETTING ALL ITEMS PER LIST. CANNOT BE REVERTED!")

	query := "DELETE FROM " + itemPerListTable
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to remove mappings from table: %s", err)
		return
	}

	log.Print("RESET ITEM MAPPING TABLE")
}

// ------------------------------------------------------------
// Item Handling
// ------------------------------------------------------------

const itemTable = "items"

func GetItem(id int64) (data.Item, error) {
	if id < 0 {
		err := errors.New("items with id < 0 do not exist")
		return data.Item{}, err
	}
	query := "SELECT * FROM " + itemTable + " WHERE id = ?"
	var item data.Item
	row := db.QueryRow(query, id)
	// Looping through data, assigning the columns to the given struct
	if err := row.Scan(&item.ID, &item.Name, &item.Icon); err != nil {
		return data.Item{}, err
	}
	return item, nil
}

func GetAllItems() ([]data.Item, error) {
	query := "SELECT * FROM " + itemTable
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to query database for items: %s", err)
		return nil, err
	}
	defer rows.Close()
	// Looping through data, assigning the columns to the given struct
	var items []data.Item
	for rows.Next() {
		var item data.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Icon); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Failed to retrieve items from database: %s", err)
		return nil, err
	}
	return items, nil
}

func GetAllItemsFromName(name string) ([]data.Item, error) {
	query := "SELECT * FROM " + itemTable + " WHERE INSTR(name, '" + name + "') > 0"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to query database for items: %s", err)
		return nil, err
	}
	defer rows.Close()
	// Looping through data, assigning the columns to the given struct
	var items []data.Item
	for rows.Next() {
		var item data.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Icon); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Failed to retrieve items from database: %s", err)
		return nil, err
	}
	return items, nil
}

func InsertItem(name string, icon string) (data.Item, error) {
	item := data.Item{
		ID:   0,
		Name: name,
		Icon: icon,
	}
	return InsertItemStruct(item)
}

func InsertItemStruct(item data.Item) (data.Item, error) {
	query := "INSERT INTO " + itemTable + " (name, icon) VALUES (?, ?)"
	result, err := db.Exec(query, item.Name, item.Icon)
	if err != nil {
		log.Printf("Failed to insert item into database: %s", err)
		return data.Item{}, err
	}
	item.ID, err = result.LastInsertId()
	if err != nil || item.ID == 0 {
		log.Printf("Failed to insert item into database: %s", err)
		return data.Item{}, err
	}
	return item, nil
}

func ModifyItemName(id int64, name string) (data.Item, error) {
	item, err := GetItem(id)
	if err != nil {
		log.Printf("Failed to get item with ID %d", id)
		return data.Item{}, err
	}
	item.Name = name
	query := "UPDATE " + itemTable + " SET name = ? WHERE id = ?"
	result, err := db.Exec(query, item.Name, item.ID)
	if err != nil {
		log.Printf("Failed to update item name: %s", err)
		return data.Item{}, err
	}
	item.ID, err = result.LastInsertId()
	if err != nil {
		log.Printf("Problem with ID during insertion of item: %s", err)
		return data.Item{}, err
	}
	return item, err
}

func ModifyItemIcon(id int64, icon string) (data.Item, error) {
	item, err := GetItem(id)
	if err != nil {
		log.Printf("Failed to get item with ID %d", id)
		return data.Item{}, err
	}
	item.Icon = icon
	query := "UPDATE " + itemTable + " SET icon = ? WHERE id = ?"
	result, err := db.Exec(query, item.Icon, item.ID)
	if err != nil {
		log.Printf("Failed to update item icon: %s", err)
		return data.Item{}, err
	}
	item.ID, err = result.LastInsertId()
	if err != nil {
		log.Printf("Problem with ID during insertion of item: %s", err)
		return data.Item{}, err
	}
	return item, err
}

func DeleteItem(id int64) error {
	query := "DELETE FROM " + itemTable + " WHERE id = ?"
	_, err := db.Exec(query, id)
	if err != nil {
		log.Printf("Failed to delete item %d: %s", id, err)
		return err
	}
	return DeleteItemInAllLists(id)
}

func ResetItemTable() {
	log.Print("RESETTING ALL ITEMS. CANNOT BE REVERTED!")

	query := "DELETE FROM " + itemTable
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to remove all items from table: %s", err)
		return
	}

	log.Print("RESET ITEMS TABLE")
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
	defer rows.Close()
	log.Print("------------- User Table -------------")
	for rows.Next() {
		var user data.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Password); err != nil {
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
	defer rows.Close()
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

func PrintItemTable() {
	query := "SELECT * FROM " + itemTable
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to print table %s: %s", shoppingListTable, err)
		return
	}
	defer rows.Close()
	log.Print("------------- Item Table -------------")
	for rows.Next() {
		var item data.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Icon); err != nil {
			log.Printf("Failed to print table: %s: %s", shoppingListTable, err)
		}
		log.Printf("%v", item)
	}
	log.Print("---------------------------------------")
}

func PrintItemPerListTable() {
	query := "SELECT * FROM " + itemPerListTable
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to print table %s: %s", itemPerListTable, err)
		return
	}
	defer rows.Close()
	log.Print("------------- Item Table -------------")
	for rows.Next() {
		var mapping data.ItemPerList
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.AddedBy); err != nil {
			log.Printf("Failed to print table: %s: %s", itemPerListTable, err)
		}
		log.Printf("%v", mapping)
	}
	log.Print("---------------------------------------")
}
