package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
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
	DBUser      string
	DBPass      string
	Addr        string
	NetworkType string
	DBName      string
}

func createDefaultConfiguration(confFile string) {
	// This method is only meant to create the file in the right format
	// It is not meant to create a config file holding a working configuration
	conf := DBConf{
		DBUser:      "",
		DBPass:      "",
		Addr:        "127.0.0.1:3306",
		NetworkType: "tcp",
		DBName:      "shoppinglist",
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

func ResetDatabase() {
	ResetUserTable()
	ResetSharedListTable()
	ResetItemTable()
	ResetItemPerListTable()
	ResetShoppingListTable()
}

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
		Net:                  config.NetworkType,
		Addr:                 config.Addr,
		DBName:               config.DBName,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		ParseTime:            true,
	}
	var err error
	configString := mysqlCfg.FormatDSN()
	// log.Printf("Config string: %s", configString)
	db, err = sql.Open("mysql", configString)
	if err != nil {
		log.Fatalf("Cannot connect to database: %s", err)
	}
	_, pingErr := db.Exec("select 42")
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
	if err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Created, &user.LastLogin); err == sql.ErrNoRows {
		return data.User{}, err
	}
	return user, nil
}

func GetUserInWireFormat(id int64) (data.UserWire, error) {
	user, err := GetUser(id)
	if err != nil {
		return data.UserWire{}, err
	}
	userWire := data.UserWire{
		ID:       user.ID,
		Username: user.Username,
	}
	return userWire, nil
}

func GetUserFromMatchingUsername(name string) ([]data.ListCreator, error) {
	query := "SELECT * FROM " + userTable + " WHERE INSTR(username, '" + name + "') > 0"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to query database for users: %s", err)
		return []data.ListCreator{}, err
	}
	defer rows.Close()
	// Looping through data, assigning the columns to the given struct
	var users []data.ListCreator
	for rows.Next() {
		var user data.ListCreator
		var password string
		var created time.Time
		var lastLogin time.Time
		if err := rows.Scan(&user.ID, &user.Name, &password, &created, &lastLogin); err != nil {
			return []data.ListCreator{}, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Failed to retrieve users from database: %s", err)
		return nil, err
	}
	return users, nil
}

func CheckUserExists(id int64) error {
	log.Printf("Checking if user exists")
	_, err := GetUser(id)
	return err
}

func createNewUserId() int64 {
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
	return int64(userId)
}

func createUser(username string, passwd string) (data.User, error) {
	userId := createNewUserId()
	hashedPw, err := argon2id.CreateHash(passwd, argon2id.DefaultParams)
	if err != nil {
		return data.User{}, err
	}
	if username == "" || passwd == "" {
		return data.User{}, errors.New("invalid username or password")
	}
	// created := time.Now().Local().Format(time.RFC3339)
	created := time.Now().UTC().Format(time.DateTime)
	newUser := data.User{
		ID:        int64(userId),
		Username:  username,
		Password:  hashedPw,
		Created:   created,
		LastLogin: created,
	}
	return newUser, nil
}

func CreateUserAccountInDatabase(username string, passwd string) (data.User, error) {
	log.Printf("Creating new user account: %s", username)
	// Creating the struct we are going to insert first
	newUser, err := createUser(username, passwd)
	if err != nil {
		log.Printf("Failed to create new user: %s", err)
		return data.User{}, err
	}
	// log.Printf("Inserting new user: %v", newUser)
	log.Printf("Creating new user %d: %s", newUser.ID, username)
	// Insert the newly created user into the database
	query := "INSERT INTO " + userTable + " (id, username, passwd, created, lastLogin) VALUES (?, ?, ?, ?, ?)"
	_, err = db.Exec(query, newUser.ID, newUser.Username, newUser.Password, newUser.Created, newUser.LastLogin)
	if err != nil {
		log.Printf("Failed to insert new user into database: %s", err)
		return data.User{}, err
	}
	newUser, err = GetUser(newUser.ID)
	if err != nil {
		log.Printf("Failed to create new user: %s", err)
		return data.User{}, err
	}
	// log.Printf("Debug: %v", newUser)
	return newUser, nil
}

func ModifyLastLogin(id int64) (data.User, error) {
	log.Printf("Updating the last login time to now")
	_, err := GetUser(id)
	if err != nil {
		log.Printf("User %d not found", id)
		return data.User{}, err
	}
	query := fmt.Sprintf("UPDATE %s SET lastLogin = current_timestamp() WHERE id = ?", userTable)
	_, err = db.Exec(query, id)
	if err != nil {
		log.Printf("Failed to update last login: %s", err)
		return data.User{}, err
	}
	user, _ := GetUser(id)
	return user, nil
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
	query := "DELETE FROM " + userTable + " WHERE id = ?"
	_, err := db.Exec(query, id)
	if err != nil {
		log.Printf("Failed to delete user with id %d: %s", id, err)
		return err
	}
	err = DeleteAllSharingForUser(id)
	if err != nil {
		return err
	}
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

func GetShoppingListWithId(id int64, createdBy int64) (int, data.Shoppinglist, error) {
	query := "SELECT * FROM " + shoppingListTable + " WHERE listId = ? AND createdBy = ?"
	row := db.QueryRow(query, id, createdBy)
	var dbId int
	var list data.Shoppinglist
	if err := row.Scan(&dbId, &list.ListId, &list.Name, &list.CreatedBy.ID, &list.Created, &list.LastEdited); err == sql.ErrNoRows {
		return 0, data.Shoppinglist{}, err
	}
	user, err := GetUser(createdBy)
	if err != nil {
		log.Printf("User not found: %s", err)
		return 0, data.Shoppinglist{}, err
	}
	list.CreatedBy.Name = user.Username
	return dbId, list, nil
}

func GetShoppingList(id int64, createdBy int64) (data.Shoppinglist, error) {
	_, list, err := GetShoppingListWithId(id, createdBy)
	return list, err
}

func GetShoppingListsFromUserId(id int64) ([]data.Shoppinglist, error) {
	query := "SELECT * FROM " + shoppingListTable + " WHERE createdBy = ?"
	rows, err := db.Query(query, id)
	if err != nil {
		log.Printf("Failed to retrieve any list for user %d: %s", id, err)
		return []data.Shoppinglist{}, err
	}
	var lists []data.Shoppinglist
	for rows.Next() {
		var dbId int64
		var list data.Shoppinglist
		if err := rows.Scan(&dbId, &list.ListId, &list.Name, &list.CreatedBy.ID, &list.Created, &list.LastEdited); err != nil {
			log.Printf("Failed to query table: %s: %s", shoppingListTable, err)
			return []data.Shoppinglist{}, err
		}
		user, err := GetUser(list.CreatedBy.ID)
		if err != nil {
			return []data.Shoppinglist{}, err
		}
		list.CreatedBy.Name = user.Username
		lists = append(lists, list)
	}
	return lists, nil
}

func GetShoppingListsFromSharedListIds(sharedLists []data.ListShared) ([]data.Shoppinglist, error) {
	if len(sharedLists) == 0 {
		log.Print("No ids given.")
		return []data.Shoppinglist{}, nil
	}
	// Extract the list ids so we can query them
	// Join the IDs followed by the createdBy to make a fitting query
	// log.Printf("Shared list: %v", sharedLists)
	listIds := make([]interface{}, 0)
	for _, shared := range sharedLists {
		listIds = append(listIds, strconv.FormatInt(shared.ListId, 10))
		listIds = append(listIds, strconv.FormatInt(shared.CreatedBy, 10))
		// listIds = append(listIds, int(shared.ListId))
	}
	// log.Printf("Searching for %d lists: %v", len(listIds), listIds)
	query := "SELECT * FROM " + shoppingListTable + " WHERE (listId, createdBy) IN ((?,?)" + strings.Repeat(",(?,?)", len(sharedLists)-1) + ")"
	// log.Printf("Query string: %s", query)
	rows, err := db.Query(query, listIds...)
	if err != nil {
		sharedWithId := -1
		if len(sharedLists) > 0 {
			sharedWithId = int(sharedLists[0].ID)
		}
		log.Printf("Failed to retrieve any shared list for user %d: %s", sharedWithId, err)
		return []data.Shoppinglist{}, err
	}
	var lists []data.Shoppinglist
	for rows.Next() {
		var dbId int64
		var list data.Shoppinglist
		if err := rows.Scan(&dbId, &list.ListId, &list.Name, &list.CreatedBy.ID, &list.Created, &list.LastEdited); err != nil {
			log.Printf("Failed to query table: %s: %s", shoppingListTable, err)
			return []data.Shoppinglist{}, err
		}
		creatorInfo, err := GetUserInWireFormat(list.CreatedBy.ID)
		if err != nil {
			log.Printf("Failed to find info of user that created the list")
		}
		list.CreatedBy.Name = creatorInfo.Username
		lists = append(lists, list)
	}
	return lists, nil
}

func checkListCorrect(list data.Shoppinglist) error {
	if list.CreatedBy.ID == 0 {
		return errors.New("invalid field created by")
	}
	if list.Name == "" {
		return errors.New("invalid field name")
	}
	if list.LastEdited.After(time.Now().UTC().Add(5 * time.Second)) {
		return fmt.Errorf("invalid field last edited. time '%s' is in future. Now: %s", list.LastEdited, time.Now().GoString())
	}
	return nil
}

func checkItemCorrect(item data.ItemWire) (data.Item, error) {
	if item.Name == "" {
		return data.Item{}, errors.New("invalid field name: is empty")
	}
	if item.Quantity <= 0 {
		return data.Item{}, errors.New("invalid field quantity: <= 0")
	}
	converted := data.Item{
		Name: item.Name,
		Icon: item.Icon,
	}
	return converted, nil
}

func createOrUpdateShoppingListBase(list data.Shoppinglist) error {
	if err := checkListCorrect(list); err != nil {
		log.Printf("List not in correct format for insertion: %s", err)
		return err
	}
	// Check if list exists and update / insert the values in this case
	query := "INSERT INTO " + shoppingListTable + " (listId, name, createdBy, created, lastEdited) VALUES (?, ?, ?, ?, ?)"
	replacing := false
	var result sql.Result
	if databaseListId, _, err := GetShoppingListWithId(list.ListId, list.CreatedBy.ID); err == nil {
		// Replace existing
		replacing = true
		log.Printf("List %d from %d exists. Replacing...", list.ListId, list.CreatedBy.ID)
		query = fmt.Sprintf("UPDATE %s SET name = ?, lastEdited = ? WHERE id = ?", shoppingListTable)
		result, err = db.Exec(query, list.Name, list.LastEdited, databaseListId)
		if err != nil {
			return err
		}
	}
	if !replacing {
		var err error
		result, err = db.Exec(query, list.ListId, list.Name, list.CreatedBy.ID, list.Created, list.LastEdited)
		if err != nil {
			return err
		}
	}
	if _, err := result.LastInsertId(); err != nil {
		return err
	}
	return nil
}

func addOrRemoveItemsInShoppingList(list data.Shoppinglist) ([]int64, []int64, error) {
	log.Printf("Adding (%d) items in shopping list to database", len(list.Items))
	var itemIds []int64
	var addedBy []int64
	for _, item := range list.Items {
		conv, err := checkItemCorrect(item)
		if err != nil {
			log.Printf("Failed to insert item '%s': %s", item.Name, err)
			return []int64{}, []int64{}, err
		}
		itemId, err := InsertItemStruct(conv)
		if err != nil {
			log.Printf("Failed to insert item '%s': %s", conv.Name, err)
			return []int64{}, []int64{}, err
		}
		itemIds = append(itemIds, itemId)
		addedBy = append(addedBy, item.AddedBy)
	}
	return itemIds, addedBy, nil
}

func mapItemsIntoShoppingList(list data.Shoppinglist, itemIds []int64, addedBy []int64) error {
	log.Printf("Adding (%d) items to shopping list", len(list.Items))
	if len(list.Items) == 0 || len(itemIds) == 0 {
		return nil
	}
	if len(list.Items) != len(itemIds) {
		return errors.New("length of items and ids does not match")
	}
	if err := DeleteAllItemsInList(list.ListId, list.CreatedBy.ID); err != nil {
		log.Printf("Failed to remove items from list %d for update: %s", list.ListId, err)
		return err
	}
	for i, item := range list.Items {
		converted := data.ItemPerList{
			ID:        0,
			ListId:    list.ListId,
			ItemId:    itemIds[i],
			Quantity:  item.Quantity,
			Checked:   item.Checked,
			CreatedBy: list.CreatedBy.ID,
			AddedBy:   addedBy[i],
		}
		if _, err := InsertOrUpdateItemInList(converted); err != nil {
			log.Printf("Failed to add '%s' to list '%s'", item.Name, list.Name)
		}
	}
	return nil
}

func CreateOrUpdateShoppingList(list data.Shoppinglist) error {
	log.Printf("Creating shopping list '%s' with id '%d' from %v", list.Name, list.ListId, list.CreatedBy)
	if err := createOrUpdateShoppingListBase(list); err != nil {
		return err
	}
	itemIds, addedBy, err := addOrRemoveItemsInShoppingList(list)
	if err != nil {
		return err
	}
	if err := mapItemsIntoShoppingList(list, itemIds, addedBy); err != nil {
		return err
	}
	return nil
}

func DeleteShoppingList(id int64, createdBy int64) error {
	query := "DELETE FROM " + shoppingListTable + " WHERE listId = ? AND createdBy = ?"
	_, err := db.Exec(query, id, createdBy)
	if err != nil {
		log.Printf("Failed to delete shopping list with id %d", id)
		return err
	}
	if err := DeleteSharingOfList(id, createdBy); err != nil {
		log.Printf("Failed to delete sharing of list %d", id)
		return err
	}
	return DeleteAllItemsInList(id, createdBy)
}

func DeleteShoppingListFrom(userId int64) error {
	query := "DELETE FROM " + shoppingListTable + " WHERE createdBy = ?"
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

func GetSharedListFromUserAndListId(listId int64, createdBy int64, sharedWith int64) ([]data.ListShared, error) {
	query := "SELECT * FROM " + sharedListTable + " WHERE listId = ? AND createdBy = ? AND sharedWithId = ?"
	rows, err := db.Query(query, listId, createdBy, sharedWith)
	if err != nil {
		return []data.ListShared{}, err
	}
	var sharedLists []data.ListShared
	for rows.Next() {
		var shared data.ListShared
		if err := rows.Scan(&shared.ID, &shared.ListId, &shared.CreatedBy, &shared.SharedWith, &shared.Created); err == sql.ErrNoRows {
			return []data.ListShared{}, err
		}
		sharedLists = append(sharedLists, shared)
	}
	return sharedLists, nil
}

func GetSharedListFromListId(listId int64) ([]data.ListShared, error) {
	query := "SELECT * FROM " + sharedListTable + " WHERE listId = ?"
	rows, err := db.Query(query, listId)
	if err != nil {
		log.Printf("Failed to query for users that get shared list %d: %s", listId, err)
		return []data.ListShared{}, err
	}
	var list []data.ListShared
	for rows.Next() {
		var shared data.ListShared
		if err := rows.Scan(&shared.ID, &shared.ListId, &shared.CreatedBy, &shared.SharedWith, &shared.Created); err != nil {
			log.Printf("Failed to query table: %s: %s", sharedListTable, err)
			return []data.ListShared{}, err
		}
		list = append(list, shared)
	}
	return list, nil
}

func GetSharedListForUserId(userId int64) ([]data.ListShared, error) {
	query := "SELECT * FROM " + sharedListTable + " WHERE sharedWithId IN (?, -1)"
	rows, err := db.Query(query, userId)
	if err != nil {
		log.Printf("Failed to query for lists that are shared with the user %d: %s", userId, err)
		return []data.ListShared{}, nil
	}
	var list []data.ListShared
	for rows.Next() {
		var shared data.ListShared
		if err := rows.Scan(&shared.ID, &shared.ListId, &shared.CreatedBy, &shared.SharedWith, &shared.Created); err != nil {
			log.Printf("Failed to query table: %s: %s", sharedListTable, err)
			return []data.ListShared{}, err
		}
		list = append(list, shared)
	}
	return list, nil
}

func IsListSharedWithUser(listId int64, createdBy int64, userId int64) error {
	query := fmt.Sprintf("SELECT * FROM %s WHERE listId = ? and createdBy = ? AND (sharedWithId = -1 OR sharedWithId = ?)", sharedListTable)
	rows, err := db.Query(query, listId, createdBy, userId)
	if err != nil {
		log.Printf("The list %d is not shared with the user %d: %s", listId, userId, err)
		return err
	}
	counter := 1
	for rows.Next() {
		if counter > 1 {
			log.Printf("Expected only a single shared row but got more than one!")
		}
		var shared data.ListShared
		if err := rows.Scan(&shared.ID, &shared.ListId, &shared.CreatedBy, &shared.SharedWith, &shared.Created); err != nil {
			log.Printf("Failed to query table: %s: %s", sharedListTable, err)
			return err
		}
		counter += 1
	}
	return nil
}

func CheckUserAndListExist(listId int64, createdBy int64, sharedWith int64) error {
	_, err := GetUser(createdBy)
	if err != nil {
		return errors.New("list owner does not exist")
	}
	_, err = GetUser(sharedWith)
	if err != nil {
		return errors.New("shared with user does not exist")
	}
	_, err = GetShoppingList(listId, createdBy)
	if err != nil {
		return errors.New("shared list does not exist")
	}
	return nil
}

func CreateOrUpdateSharedList(listId int64, createdBy int64, sharedWith int64) (data.ListShared, error) {
	sharedExists, err := GetSharedListFromUserAndListId(listId, createdBy, sharedWith)
	if err == nil && len(sharedExists) > 0 {
		log.Printf("Shared of list %d for user %d exists", listId, sharedExists[0].SharedWith)
		return sharedExists[0], nil
	}
	if err := CheckUserAndListExist(listId, createdBy, sharedWith); err != nil {
		log.Printf("User or list does not exist: %s", err)
		return data.ListShared{}, err
	}
	query := "INSERT INTO " + sharedListTable + " (listId, createdBy, sharedWithId, created) VALUES (?, ?, ?, ?)"
	shared := data.ListShared{
		ID:         0,
		ListId:     listId,
		CreatedBy:  createdBy,
		SharedWith: sharedWith,
		Created:    time.Now().Local(),
	}
	result, err := db.Exec(query, shared.ListId, shared.CreatedBy, shared.SharedWith, shared.Created)
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

func DeleteSharingOfList(listId int64, createdBy int64) error {
	query := "DELETE FROM " + sharedListTable + " WHERE listId = ? AND createdBy = ?"
	_, err := db.Exec(query, listId, createdBy)
	if err != nil {
		log.Printf("Failed to delete sharing of list %d: %s", listId, err)
		return err
	}
	return nil
}

func DeleteSharingForUser(listId int64, createdBy int64, userId int64) error {
	query := "DELETE FROM " + sharedListTable + " WHERE listId = ? AND createdBy = ? AND sharedWithId = ?"
	_, err := db.Exec(query, listId, createdBy, userId)
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

func ResetSharedListTable() {
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

func IsItemInList(listId int64, createdBy int64, itemId int64) (int64, error) {
	query := "SELECT * FROM " + itemPerListTable + " WHERE listId = ? AND itemId = ? AND createdBy = ?"
	row := db.QueryRow(query, listId, itemId, createdBy)
	var dbId int
	var mapping data.ItemPerList
	if err := row.Scan(&dbId, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.CreatedBy, &mapping.AddedBy); err == sql.ErrNoRows {
		return 0, err
	}
	return int64(dbId), nil
}

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
func GetItemsInList(listId int64, createdBy int64) ([]data.ItemPerList, error) {
	var list []data.ItemPerList
	query := "SELECT * FROM " + itemPerListTable + " WHERE listId = ? AND createdBy = ?"
	rows, err := db.Query(query, listId, createdBy)
	if err != nil {
		log.Printf("Failed to query for items contained in list %d: %s", listId, err)
		return []data.ItemPerList{}, nil
	}
	for rows.Next() {
		var mapping data.ItemPerList
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.CreatedBy, &mapping.AddedBy); err != nil {
			log.Printf("Failed to query table: %s: %s", itemPerListTable, err)
			return []data.ItemPerList{}, err
		}
		list = append(list, mapping)
	}
	return list, nil
}

func InsertOrUpdateItemInList(mapping data.ItemPerList) (data.ItemPerList, error) {
	update := false
	itemId, err := IsItemInList(mapping.ListId, mapping.CreatedBy, mapping.ItemId)
	if err == nil {
		update = true
	}
	if update {
		query := fmt.Sprintf("UPDATE %s SET quantity = ?, checked = ?, addedBy = ? WHERE id = ?", itemPerListTable)
		_, err := db.Exec(query, mapping.Quantity, mapping.Checked, mapping.AddedBy, itemId)
		if err != nil {
			log.Printf("Failed to insert mapping into database: %s", err)
			return data.ItemPerList{}, err
		}
		return mapping, nil
	}
	query := "INSERT INTO " + itemPerListTable + " (listId, itemId, quantity, checked, createdBy, addedBy) VALUES (?, ?, ?, ?, ?, ?)"
	result, err := db.Exec(query, mapping.ListId, mapping.ItemId, mapping.Quantity, mapping.Checked, mapping.CreatedBy, mapping.AddedBy)
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

func DeleteItemInList(itemId int64, listId int64, createdBy int64) error {
	query := "DELETE FROM " + itemPerListTable + " WHERE itemId = ? AND listId = ? AND createdBy = ?"
	_, err := db.Exec(query, itemId, listId, createdBy)
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

func DeleteAllItemsInList(listId int64, createdBy int64) error {
	query := "DELETE FROM " + itemPerListTable + " WHERE listId = ? AND createdBy = ?"
	_, err := db.Exec(query, listId, createdBy)
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

func InsertItem(name string, icon string) (int64, error) {
	item := data.Item{
		Name: name,
		Icon: icon,
	}
	return InsertItemStruct(item)
}

func InsertItemStruct(item data.Item) (int64, error) {
	// log.Printf("DEBUG: checking if item needs to be inserted or does exist")
	selectQuery := "SELECT * FROM " + itemTable + " WHERE name = ?"
	row := db.QueryRow(selectQuery, strings.TrimSpace(item.Name))
	var existingItem data.Item
	if err := row.Scan(&existingItem.ID, &existingItem.Name, &existingItem.Icon); err == nil {
		log.Printf("DEBUG: Item (%s) existed (%d)...", existingItem.Name, existingItem.ID)
		return existingItem.ID, nil
	}
	query := "INSERT INTO " + itemTable + " (name, icon) VALUES (?, ?)"
	result, err := db.Exec(query, strings.TrimSpace(item.Name), strings.TrimSpace(item.Icon))
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil || id == 0 {
		log.Printf("Failed to insert item into database: %s", err)
		return 0, err
	}
	return id, nil
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
		if err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Created, &user.LastLogin); err != nil {
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
		var dbId int64
		var list data.Shoppinglist
		if err := rows.Scan(&dbId, &list.ListId, &list.Name, &list.CreatedBy.ID, &list.Created, &list.LastEdited); err != nil {
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
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.CreatedBy, &mapping.AddedBy); err != nil {
			log.Printf("Failed to print table: %s: %s", itemPerListTable, err)
		}
		log.Printf("%v", mapping)
	}
	log.Print("---------------------------------------")
}

func PrintSharingTable() {
	query := "SELECT * FROM " + sharedListTable
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to print table %s: %s", sharedListTable, err)
		return
	}
	defer rows.Close()
	log.Print("------------- Sharing Table -------------")
	for rows.Next() {
		var sharing data.ListShared
		if err := rows.Scan(&sharing.ID, &sharing.ListId, &sharing.CreatedBy, &sharing.SharedWith, &sharing.Created); err != nil {
			log.Printf("Failed to print table: %s: %s", itemPerListTable, err)
		}
		log.Printf("%v", sharing)
	}
	log.Print("---------------------------------------")
}
