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
	"github.com/jmoiron/sqlx"

	"github.com/justusvonderbeek/shopping-list-server/internal/configuration"
	"github.com/justusvonderbeek/shopping-list-server/internal/data"
)

// A small database wrapper allowing to access a MySQL database

// ------------------------------------------------------------
// Configuration File Handling
// ------------------------------------------------------------

var config DBConf
var db *sql.DB

type DBConf struct {
	DBUser      string
	DBPwd       string
	Addr        string
	NetworkType string
	DBName      string
}

func createDefaultConfiguration(confFile string) {
	// This method is only meant to create the file in the right format
	// It is not meant to create a config file holding a working configuration
	conf := DBConf{
		DBUser:      "",
		DBPwd:       "",
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
		Passwd:               config.DBPwd,
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

func convertTimeToString(timeToFormat time.Time) string {
	// For some strange reason, the default mechanism to parse UTC time
	// seems to produce different results for the raspi and my local machine
	// Therefore, explicitly define the format here again
	// log.Printf("Time before format: %v", timeToFormat)

	// We need to omit the Z at the end for our database
	formatTime := timeToFormat.Format("2006-01-02T03:04:05+07:00")
	// log.Printf("Converted time: %s", formatTime)
	return formatTime
}

func convertStringToTime(strTime string) time.Time {
	trimmedString := strings.Trim(strTime, "\t")
	parsedTime, err := time.Parse("2006-01-02T03:04:05+07:00", trimmedString)
	if err != nil {
		log.Printf("Failed to parse time: %s", trimmedString)
		return time.Now().UTC()
	}
	return parsedTime
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
	if err := row.Scan(&user.OnlineID, &user.Username, &user.Password, &user.Created, &user.LastLogin); err == sql.ErrNoRows {
		return data.User{}, err
	}
	return user, nil
}

func GetAllUsers() ([]data.User, error) {
	query := "SELECT id,username,created,lastLogin FROM " + userTable
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to query database for users: %s", err)
		return []data.User{}, err
	}
	defer rows.Close()
	// Looping through data, assigning the columns to the given struct
	var users []data.User
	for rows.Next() {
		var user data.User
		if err := rows.Scan(&user.OnlineID, &user.Username, &user.Created, &user.LastLogin); err != nil {
			return []data.User{}, err
		}
		users = append(users, user)
	}
	return users, nil
}

func GetUserInWireFormat(id int64) (data.User, error) {
	user, err := GetUser(id)
	if err != nil {
		return data.User{}, err
	}
	userWire := data.User{
		OnlineID: user.OnlineID,
		Username: user.Username,
	}
	return userWire, nil
}

func GetUserFromMatchingUsername(name string) ([]data.User, error) {
	query := "SELECT id,username,lastLogin FROM " + userTable + " WHERE INSTR(username, '" + name + "') > 0"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to query database for users: %s", err)
		return []data.User{}, err
	}
	defer rows.Close()
	// Looping through data, assigning the columns to the given struct
	var users []data.User
	for rows.Next() {
		var user data.User
		var lastLogin time.Time
		if err := rows.Scan(&user.OnlineID, &user.Username, &lastLogin); err != nil {
			return []data.User{}, err
		}
		// convert the time into a string
		// timeConv := convertTimeToString(lastLogin)
		// user.LastLogin = timeConv
		user.LastLogin = lastLogin
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
	now := time.Now().UTC()
	// created := convertTimeToString(now)
	newUser := data.User{
		OnlineID:  int64(userId),
		Username:  username,
		Password:  hashedPw,
		Created:   now,
		LastLogin: now,
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
	log.Printf("Creating new user %d: %s", newUser.OnlineID, username)
	// Insert the newly created user into the database
	query := "INSERT INTO " + userTable + " (id, username, passwd, created, lastLogin) VALUES (?, ?, ?, ?, ?)"
	_, err = db.Exec(query, newUser.OnlineID, newUser.Username, newUser.Password, newUser.Created, newUser.LastLogin)
	if err != nil {
		log.Printf("Failed to insert new user into database: %s", err)
		return data.User{}, err
	}
	newUser, err = GetUser(newUser.OnlineID)
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
	_, err = db.Exec(query, user.Username, user.OnlineID)
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
	_, err = db.Exec(query, user.Password, user.OnlineID)
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

func GetShoppingListWithId(id int64, createdBy int64) (int, data.List, error) {
	query := "SELECT * FROM " + shoppingListTable + " WHERE listId = ? AND createdBy = ?"
	row := db.QueryRow(query, id, createdBy)
	var dbId int
	var list data.List
	if err := row.Scan(&dbId, &list.ListId, &list.Title, &list.CreatedBy.ID, &list.CreatedAt, &list.LastUpdated); err == sql.ErrNoRows {
		return 0, data.List{}, err
	}
	user, err := GetUser(createdBy)
	if err != nil {
		log.Printf("User not found: %s", err)
		return 0, data.List{}, err
	}
	list.CreatedBy.Name = user.Username
	return dbId, list, nil
}

func GetAllShoppingLists() ([]data.List, error) {
	query := "SELECT * FROM " + shoppingListTable
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to retrieve any list: %s", err)
		return []data.List{}, err
	}
	var lists []data.List
	for rows.Next() {
		var dbId int64
		var list data.List
		if err := rows.Scan(&dbId, &list.ListId, &list.Title, &list.CreatedBy.ID, &list.CreatedAt, &list.LastUpdated, &list.Version); err != nil {
			log.Printf("Failed to query table: %s: %s", shoppingListTable, err)
			return []data.List{}, err
		}
		user, err := GetUser(list.CreatedBy.ID) // TODO: Cache this to reduce the db hits necessary
		if err != nil {
			return []data.List{}, err
		}
		list.CreatedBy.Name = user.Username
		lists = append(lists, list)
	}
	return lists, nil
}

func GetShoppingList(id int64, createdBy int64) (data.List, error) {
	_, list, err := GetShoppingListWithId(id, createdBy)
	return list, err
}

func GetShoppingListsFromUserId(id int64) ([]data.List, error) {
	query := "SELECT * FROM " + shoppingListTable + " WHERE createdBy = ?"
	rows, err := db.Query(query, id)
	if err != nil {
		log.Printf("Failed to retrieve any list for user %d: %s", id, err)
		return []data.List{}, err
	}
	var lists []data.List
	for rows.Next() {
		var dbId int64
		var list data.List
		if err := rows.Scan(&dbId, &list.ListId, &list.Title, &list.CreatedBy.ID, &list.CreatedAt, &list.LastUpdated, &list.Version); err != nil {
			log.Printf("Failed to query table: %s: %s", shoppingListTable, err)
			return []data.List{}, err
		}
		user, err := GetUser(list.CreatedBy.ID) // TODO: Cache this to reduce the db hits necessary
		if err != nil {
			return []data.List{}, err
		}
		list.CreatedBy.Name = user.Username
		lists = append(lists, list)
	}
	return lists, nil
}

func GetShoppingListsFromSharedListIds(sharedLists []data.ListShared) ([]data.List, error) {
	if len(sharedLists) == 0 {
		log.Print("No ids given.")
		return []data.List{}, nil
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
		return []data.List{}, err
	}
	var lists []data.List
	for rows.Next() {
		var dbId int64
		var list data.List
		if err := rows.Scan(&dbId, &list.ListId, &list.Title, &list.CreatedBy.ID, &list.CreatedAt, &list.LastUpdated); err != nil {
			log.Printf("Failed to query table: %s: %s", shoppingListTable, err)
			return []data.List{}, err
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

func checkListCorrect(list data.List) error {
	if list.CreatedBy.ID == 0 {
		return errors.New("invalid field created by")
	}
	if list.Title == "" {
		return errors.New("invalid field name")
	}
	if list.LastUpdated.After(time.Now().UTC().Add(5 * time.Second)) {
		return fmt.Errorf("invalid field last edited. time '%s' is in future. Now: %s", list.LastUpdated, time.Now().GoString())
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

func createOrUpdateShoppingListBase(list data.List) error {
	if err := checkListCorrect(list); err != nil {
		log.Printf("List not in correct format for insertion: %s", err)
		return err
	}
	// Check if list exists and update / insert the values in this case
	query := "INSERT INTO " + shoppingListTable + " (listId, name, createdBy, created, lastEdited, version) VALUES (?, ?, ?, ?, ?, ?)"
	replacing := false
	var result sql.Result
	if databaseListId, _, err := GetShoppingListWithId(list.ListId, list.CreatedBy.ID); err == nil {
		// Replace existing
		replacing = true
		log.Printf("List %d from %d exists. Replacing...", list.ListId, list.CreatedBy.ID)
		query = fmt.Sprintf("UPDATE %s SET name = ?, lastEdited = ?, version = ? WHERE id = ?", shoppingListTable)
		result, err = db.Exec(query, list.Title, list.LastUpdated, list.Version, databaseListId)
		if err != nil {
			return err
		}
	}
	if !replacing {
		var err error
		result, err = db.Exec(query, list.ListId, list.Title, list.CreatedBy.ID, list.CreatedAt, list.LastUpdated, list.Version)
		if err != nil {
			return err
		}
	}
	if _, err := result.LastInsertId(); err != nil {
		return err
	}
	return nil
}

func addOrRemoveItemsInShoppingList(list data.List) ([]int64, []int64, error) {
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

func mapItemsIntoShoppingList(list data.List, itemIds []int64, addedBy []int64) error {
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
		converted := data.ListItem{
			ID:        0,
			ListId:    list.ListId,
			ItemId:    itemIds[i],
			Quantity:  item.Quantity,
			Checked:   item.Checked,
			CreatedBy: list.CreatedBy.ID,
			AddedBy:   addedBy[i],
		}
		if _, err := InsertOrUpdateItemInList(converted); err != nil {
			log.Printf("Failed to add '%s' to list '%s'", item.Name, list.Title)
		}
	}
	return nil
}

func CreateOrUpdateShoppingList(list data.List) error {
	log.Printf("Creating shopping list '%s' with id '%d' from %v", list.Title, list.ListId, list.CreatedBy)
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

// TODO: rework this
func CreateShoppingList(list data.List) error {
	log.Printf("Creating new shopping list '%s'", list.Title)
	// TODO: abort if the list already exists
	// because that is bad usage of this function
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
		SharedWith: []int64{sharedWith},
		Created:    time.Now().Local(),
	}
	result, err := db.Exec(query, shared.ListId, shared.CreatedBy, shared.SharedWith[0], shared.Created)
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
	var mapping data.ListItem
	if err := row.Scan(&dbId, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.CreatedBy, &mapping.AddedBy); err == sql.ErrNoRows {
		return 0, err
	}
	return int64(dbId), nil
}

// Returns the lists in which the item with the given ID is included
func GetListsOfItem(itemId int64) ([]data.ListItem, error) {
	var lists []data.ListItem
	query := "SELECT * FROM " + itemPerListTable + " WHERE itemId = ?"
	rows, err := db.Query(query, itemId)
	if err != nil {
		log.Printf("Failed to query for lists containing item %d: %s", itemId, err)
		return []data.ListItem{}, nil
	}
	for rows.Next() {
		var mapping data.ListItem
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.AddedBy); err != nil {
			log.Printf("Failed to query table: %s: %s", itemPerListTable, err)
			return []data.ListItem{}, err
		}
		lists = append(lists, mapping)
	}
	return lists, nil
}

// Returns the items in a specific list
func GetItemsInList(listId int64, createdBy int64) ([]data.ListItem, error) {
	var list []data.ListItem
	query := "SELECT * FROM " + itemPerListTable + " WHERE listId = ? AND createdBy = ?"
	rows, err := db.Query(query, listId, createdBy)
	if err != nil {
		log.Printf("Failed to query for items contained in list %d: %s", listId, err)
		return []data.ListItem{}, nil
	}
	for rows.Next() {
		var mapping data.ListItem
		if err := rows.Scan(&mapping.ID, &mapping.ListId, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.CreatedBy, &mapping.AddedBy); err != nil {
			log.Printf("Failed to query table: %s: %s", itemPerListTable, err)
			return []data.ListItem{}, err
		}
		list = append(list, mapping)
	}
	return list, nil
}

func InsertOrUpdateItemInList(mapping data.ListItem) (data.ListItem, error) {
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
			return data.ListItem{}, err
		}
		return mapping, nil
	}
	query := "INSERT INTO " + itemPerListTable + " (listId, itemId, quantity, checked, createdBy, addedBy) VALUES (?, ?, ?, ?, ?, ?)"
	result, err := db.Exec(query, mapping.ListId, mapping.ItemId, mapping.Quantity, mapping.Checked, mapping.CreatedBy, mapping.AddedBy)
	if err != nil {
		log.Printf("Failed to insert mapping into database: %s", err)
		return data.ListItem{}, err
	}
	mapping.ID, err = result.LastInsertId()
	if err != nil || mapping.ID == 0 {
		log.Printf("Failed to insert mapping into database: %s", err)
		return data.ListItem{}, err
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
	if err := row.Scan(&item.ItemId, &item.Name, &item.Icon); err != nil {
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
		if err := rows.Scan(&item.ItemId, &item.Name, &item.Icon); err != nil {
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
		if err := rows.Scan(&item.ItemId, &item.Name, &item.Icon); err != nil {
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
	if err := row.Scan(&existingItem.ItemId, &existingItem.Name, &existingItem.Icon); err == nil {
		log.Printf("DEBUG: Item (%s) existed (%d)...", existingItem.Name, existingItem.ItemId)
		return existingItem.ItemId, nil
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
	result, err := db.Exec(query, item.Name, item.ItemId)
	if err != nil {
		log.Printf("Failed to update item name: %s", err)
		return data.Item{}, err
	}
	item.ItemId, err = result.LastInsertId()
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
	result, err := db.Exec(query, item.Icon, item.ItemId)
	if err != nil {
		log.Printf("Failed to update item icon: %s", err)
		return data.Item{}, err
	}
	item.ItemId, err = result.LastInsertId()
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
// Recipes Handling
// ------------------------------------------------------------

var recipeTable = "recipe"

func CreateRecipe(recipe data.Recipe) error {
	log.Printf("Creating new recipe with name '%s'", recipe.Name)
	query := fmt.Sprintf("INSERT INTO %s (recipeId, createdBy, name, createdAt, lastUpdate, version, defaultPortion) VALUES (?, ?, ?, ?, ?, ?, ?)", recipeTable)
	_, err := db.Exec(query, recipe.RecipeId, recipe.CreatedBy.ID, recipe.Name, recipe.CreatedAt, recipe.LastUpdate, recipe.Version, recipe.DefaultPortion)
	if err != nil {
		log.Printf("Failed to insert values into database: %s", err)
		return err
	}
	err = insertDescription(recipe.RecipeId, recipe.CreatedBy.ID, recipe.Description)
	if err != nil {
		log.Printf("Failed to create recipe '%s' because of descriptions: %s", recipe.Name, err)
		return err
	}
	err = insertIngredients(recipe.RecipeId, recipe.CreatedBy.ID, recipe.Ingredients)
	if err != nil {
		log.Printf("Failed to create recipe '%s' because of ingredients: %s", recipe.Name, err)
		return err
	}
	return nil
}

var recipeDescriptionTable = "descriptionPerRecipe"

func insertDescription(recipeId int64, createdBy int64, descriptions []data.RecipeDescription) error {
	log.Printf("Inserting %d recipe descriptions", len(descriptions))
	query := fmt.Sprintf("INSERT INTO %s (recipeId, createdBy, description, descriptionOrder) VALUES (?, ?, ?, ?)", recipeDescriptionTable)
	for _, v := range descriptions {
		_, err := db.Exec(query, recipeId, createdBy, v.Step, v.Order)
		if err != nil {
			log.Printf("Failed to insert %v into table: %s", v, err)
			return err
		}
	}
	return nil
}

var recipeIngredientTable = "ingredientPerRecipe"

func insertIngredients(recipeId int64, createdBy int64, ingredients []data.Ingredient) error {
	log.Printf("Insert %d recipe ingredients", len(ingredients))
	// We need to check if an item exists and reference this item rather than creating a new one
	query := fmt.Sprintf("INSERT INTO %s (recipeId, createdBy, itemId, quantity, quantityType) VALUES (?, ?, ?, ?, ?)", recipeIngredientTable)
	for _, v := range ingredients {
		existingItems, err := GetAllItemsFromName(v.Name)
		if err != nil {
			return err
		}
		itemId := int64(0)
		exists := false
		for _, item := range existingItems {
			if item.Name == v.Name {
				itemId = item.ItemId
				exists = true
				break
			}
		}
		if !exists {
			log.Printf("Item '%s' does not exist yet", v.Name)
			itemId, err = InsertItem(v.Name, v.Icon)
			if err != nil {
				log.Printf("Failed to create new item '%s'", v.Name)
				continue
			}
		}
		_, err = db.Exec(query, recipeId, createdBy, itemId, v.Quantity, v.QuantityType)
		if err != nil {
			log.Printf("Failed to insert %v into table: %s", v, err)
			return err
		}
	}
	return nil
}

func GetIngredientsForRecipe(recipeId int64, createdBy int64) ([]data.Ingredient, error) {
	log.Printf("Trying to retrieve ingredients for recipe %d from %d", recipeId, createdBy)
	query := fmt.Sprintf("SELECT i.name, i.icon, r.quantity, r.quantityType FROM %s r JOIN items i ON r.itemId = i.id WHERE recipeId = ? AND createdBy = ?", recipeIngredientTable)
	dbx := sqlx.NewDb(db, "sql")
	rows, err := dbx.Queryx(query, recipeId, createdBy)
	if err != nil {
		log.Printf("Failed to retrieve ingredients for recipe %d from %d", recipeId, createdBy)
		return []data.Ingredient{}, err
	}
	var ingredients []data.Ingredient
	for rows.Next() {
		var dbIngredient data.Ingredient
		if err := rows.StructScan(&dbIngredient); err != nil {
			log.Printf("Failed to retrieve ingredient for recipe: %s", err)
			return []data.Ingredient{}, err
		}
		ingredients = append(ingredients, dbIngredient)
	}
	return ingredients, nil
}

func GetDescriptionsForRecipe(recipeId int64, createdBy int64) ([]data.RecipeDescription, error) {
	log.Printf("Reading descriptions for recipe %d from %d", recipeId, createdBy)
	query := fmt.Sprintf("SELECT description, descriptionOrder FROM %s WHERE recipeId = ? AND createdBy = ?", recipeDescriptionTable)
	rows, err := db.Query(query, recipeId, createdBy)
	if err != nil {
		log.Printf("Failed to retrieve descriptions for recipe %d from %d", recipeId, createdBy)
		return []data.RecipeDescription{}, err
	}
	var descriptions []data.RecipeDescription
	for rows.Next() {
		var description data.RecipeDescription
		if err := rows.Scan(&description.Step, &description.Order); err != nil {
			log.Printf("Failed to retrieve description for recipe %d from %d", recipeId, createdBy)
			return []data.RecipeDescription{}, err
		}
		descriptions = append(descriptions, description)
	}
	return descriptions, nil
}

func GetRecipe(recipeId int64, createdBy int64) (data.Recipe, error) {
	log.Printf("Retrieving recipe '%d' from '%d'", recipeId, createdBy)
	// dbx := sqlx.NewDb(db, "sql")

	query := fmt.Sprintf("SELECT recipeId, createdBy, name, createdAt, lastUpdate, version, defaultPortion FROM %s WHERE recipeId = ? AND createdBy = ?", recipeTable)
	// row := dbx.QueryRowx(query, recipeId, createdBy)
	row := db.QueryRow(query, recipeId, createdBy)

	var recipe data.Recipe
	// err := row.StructScan(&recipe)
	err := row.Scan(&recipe.RecipeId, &recipe.CreatedBy.ID, &recipe.Name, &recipe.CreatedAt, &recipe.LastUpdate, &recipe.Version, &recipe.DefaultPortion)
	if err != nil {
		log.Printf("Failed to get recipe %d from %d: %s", recipeId, createdBy, err)
		return data.Recipe{}, err
	}
	ingredients, err := GetIngredientsForRecipe(recipeId, createdBy)
	if err != nil {
		log.Printf("Failed to get ingredient for recipe: %s", err)
		return data.Recipe{}, err
	}
	recipe.Ingredients = ingredients

	descriptions, err := GetDescriptionsForRecipe(recipeId, createdBy)
	if err != nil {
		log.Printf("Failed to retrieve recipe %d from %d", recipeId, createdBy)
		return data.Recipe{}, nil
	}
	recipe.Description = descriptions
	return recipe, nil
}

func GetAllRecipes() ([]data.Recipe, error) {
	log.Print("Retrieving all recipes")
	query := fmt.Sprintf("SELECT recipeId, name, createdBy, createdAt, lastUpdate, version, defaultPortion FROM %s", recipeTable)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to retrieve all recipes: %s", err)
		return nil, err
	}
	recipes := make([]data.Recipe, 0)
	for rows.Next() {
		var recipe data.Recipe
		if err := rows.Scan(&recipe.RecipeId, &recipe.Name, &recipe.CreatedBy.ID, &recipe.CreatedAt, &recipe.LastUpdate, &recipe.Version, &recipe.DefaultPortion); err != nil {
			log.Printf("Failed to get data from table: %s: %s", recipeTable, err)
			return nil, err
		}
		ingredients, err := GetIngredientsForRecipe(recipe.RecipeId, recipe.CreatedBy.ID)
		if err != nil {
			log.Printf("Failed to retrieve ingredients for recipe %s: %s", recipe.Name, err)
			recipes = append(recipes, recipe)
			continue
		}
		recipe.Ingredients = ingredients
		descriptions, err := GetDescriptionsForRecipe(recipe.RecipeId, recipe.CreatedBy.ID)
		if err != nil {
			log.Printf("Failed to retrieve descriptions for recipe %s: %s", recipe.Name, err)
			recipes = append(recipes, recipe)
			continue
		}
		recipe.Description = descriptions
		recipes = append(recipes, recipe)
	}
	return recipes, nil
}

func updateIngredients(recipeId int64, createdBy int64, ingredients []data.Ingredient) error {
	err := deleteIngredients(recipeId, createdBy)
	if err != nil {
		return err
	}
	err = insertIngredients(recipeId, createdBy, ingredients)
	return err
}

func updateDescriptions(recipeId int64, createdBy int64, descriptions []data.RecipeDescription) error {
	err := deleteDescriptions(recipeId, createdBy)
	if err != nil {
		return err
	}
	err = insertDescription(recipeId, createdBy, descriptions)
	return err
}

func UpdateRecipe(recipe data.Recipe) error {
	log.Printf("Updating recipe '%s'", recipe.Name)
	existingRecipe, err := GetRecipe(recipe.RecipeId, recipe.CreatedBy.ID)
	if err != nil {
		log.Printf("The recipe to update was not found: %s", err)
		return err
	}
	if existingRecipe.Version >= recipe.Version {
		return errors.New(" recipe to update has the same or lower version than existing recipe")
	}
	updateRecipeVersionQuery := fmt.Sprintf("UPDATE %s SET version = ?, lastUpdate = ? WHERE recipeId = ? AND createdBy = ?", recipeTable)
	_, err = db.Exec(updateRecipeVersionQuery, recipe.Version, time.Now(), recipe.RecipeId, recipe.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to update recipe version: %s", err)
		return err
	}
	if err := updateDescriptions(recipe.RecipeId, recipe.CreatedBy.ID, recipe.Description); err != nil {
		log.Printf("Failed to update descriptions: %s", err)
		return err
	}
	if err := updateIngredients(recipe.RecipeId, recipe.CreatedBy.ID, recipe.Ingredients); err != nil {
		log.Printf("Failed to update ingredients: %s", err)
		return err
	}
	return nil
}

func deleteIngredients(recipeId int64, createdBy int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE recipeId = ? AND createdBy = ?", recipeIngredientTable)
	_, err := db.Exec(query, recipeId, createdBy)
	return err
}

func deleteDescriptions(recipeId int64, createdBy int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE recipeId = ? AND createdBy = ?", recipeDescriptionTable)
	_, err := db.Exec(query, recipeId, createdBy)
	return err
}

func DeleteRecipe(recipeId int64, createdBy int64) error {
	log.Printf("Deleting recipe %d from %d", recipeId, createdBy)
	if err := deleteDescriptions(recipeId, createdBy); err != nil {
		return err
	}
	if err := deleteIngredients(recipeId, createdBy); err != nil {
		return err
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE recipeId = ? AND createdBy = ?", recipeTable)
	_, err := db.Exec(query, recipeId, createdBy)
	if err != nil {
		log.Printf("Failed to delete recipe %d from %d: %s", recipeId, createdBy, err)
		return err
	}
	return nil
}

func ResetRecipeTables() {
	log.Print("RESETTING ALL RECIPES. CANNOT BE REVERTED!")

	query := "DELETE FROM " + recipeTable
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to remove all recipes: %s", err)
		return
	}
	query = "DELETE FROM " + recipeDescriptionTable
	_, err = db.Exec(query)
	if err != nil {
		log.Printf("Failed to remove all recipe descriptions: %s", err)
		return
	}
	query = "DELETE FROM " + recipeIngredientTable
	_, err = db.Exec(query)
	if err != nil {
		log.Printf("Failed to remove all recipe ingredients: %s", err)
		return
	}

	log.Print("RESET RECIPE TABLES")
}

// ------------------------------------------------------------
// Recipe Sharing Handling
// ------------------------------------------------------------

var recipeSharingTable = "sharedRecipe"

func CreateRecipeSharing(recipeId int64, createdBy int64, sharedWith int64) error {
	log.Printf("Creating new sharing for %d of recipe %d from %d", sharedWith, recipeId, createdBy)

	query := fmt.Sprintf("INSERT INTO %s (recipeId, createdBy, sharedWith) VALUES (?, ?, ?)", recipeSharingTable)
	_, err := db.Exec(query, recipeId, createdBy, sharedWith)
	if err != nil {
		log.Printf("Failed to insert sharing for user %d into database: %s", sharedWith, err)
		return err
	}
	return nil
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
		if err := rows.Scan(&user.OnlineID, &user.Username, &user.Password, &user.Created, &user.LastLogin); err != nil {
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
		var list data.List
		if err := rows.Scan(&dbId, &list.ListId, &list.Title, &list.CreatedBy.ID, &list.CreatedAt, &list.LastUpdated); err != nil {
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
		if err := rows.Scan(&item.ItemId, &item.Name, &item.Icon); err != nil {
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
		var mapping data.ListItem
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

// Below is todo:
type Quantity string

const (
	GRAMM  Quantity = "g"
	KILO   Quantity = "kg"
	ML     Quantity = "ml"
	PIECES Quantity = "Stk."
	SPOONS Quantity = "EL"
)

type Recipe struct {
	Title                string
	Subdescription       string
	Difficulty           int
	Duration             int
	TotalDuration        int
	Servings             int
	Ingredients          []string
	IngredientQuantities []string
	Description          []string
	Url                  string
}
