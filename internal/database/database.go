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

	"github.com/JustusvonderBeek/shoppinglist-server/internal/configuration"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/util"
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

func loadConfigFile(filename string) ([]byte, error) {
	return util.ReadFileFromRoot(filename)
}

func loadDbConfig(filename string) (DBConf, error) {
	if filename == "" {
		return DBConf{}, errors.New("no database config file given")
	}
	content, err := loadConfigFile(filename)
	if err != nil && os.IsNotExist(err) {
		createDefaultConfiguration(filename)
		return DBConf{}, errors.New("no config file found, created default one but missing entries")
	} else if err != nil {
		return DBConf{}, err
	}
	var conf DBConf
	err = json.Unmarshal(content, &conf)
	if err != nil {
		return DBConf{}, err
	}
	return conf, nil
}

func loadConfig(confFile string) {
	loadedConfig, err := loadDbConfig(confFile)
	if err != nil {
		log.Fatalf("Failed to load DB loadedConfig: %s", err)
	}
	config = loadedConfig
	log.Printf("Successfully loaded loadedConfig from '%s'", confFile)
}

func storeConfiguration(filename string) {
	if filename == "" {
		log.Fatal("Cannot store configuration file due to empty location")
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("Failed to convert configuration to file format")
	}
	storedFilename, _, err := util.WriteFileAtRoot(encoded, filename, false)
	log.Printf("Stored configuration to file: %s", storedFilename)
}

// ------------------------------------------------------------

func ResetDatabase() {
	DropUserTable()
	ResetSharedListTable()
	ResetItemTable()
	ResetItemPerListTable()
	DropShoppingListTable()
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

const getUserQuery = "SELECT id,username,passwd,created,lastLogin FROM shoppers WHERE id = ?"
const getUserRoleQuery = "SELECT role FROM role WHERE user_id = ?"

func GetUser(id int64) (data.User, error) {
	row := db.QueryRow(getUserQuery, id)
	var user data.User
	if err := row.Scan(&user.OnlineID, &user.Username, &user.Password, &user.Created, &user.LastLogin); errors.Is(err, sql.ErrNoRows) || err != nil {
		return data.User{}, err
	}
	roleRow := db.QueryRow(getUserRoleQuery, id)
	var role data.Role
	if err := roleRow.Scan(&role.Role); err != nil {
		return data.User{}, err
	}
	user.Role = role.ToEnumConstant()
	return user, nil
}

const getAllUserQuery = "SELECT id,username,created,lastLogin FROM shoppers"

func GetAllUsers() ([]data.User, error) {
	rows, err := db.Query(getAllUserQuery)
	if err != nil {
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

const getUserFromUsernameSearchString = "SELECT id,username,created,lastLogin FROM shoppers WHERE INSTR(username, '?') > 0"

func GetUserFromMatchingUsername(name string) ([]data.User, error) {
	rows, err := db.Query(getUserFromUsernameSearchString, name)
	if err != nil {
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
		user.LastLogin = lastLogin
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func userExists(id int64) error {
	_, err := GetUser(id)
	return err
}

/* The reason why we don't simply use AUTO_INCREMENT is so that randomly generated IDs prevent easy guessing */
func createNewUserId() int64 {
	userId := random.Int31()
	for {
		err := userExists(int64(userId))
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
		return data.User{}, errors.New("empty username or password")
	}
	now := time.Now().UTC()
	newUser := data.User{
		OnlineID:  userId,
		Username:  username,
		Password:  hashedPw,
		Role:      data.USER,
		Created:   now,
		LastLogin: now,
	}
	return newUser, nil
}

const createUserQuery = "INSERT INTO shoppers (id,username,passwd,created,lastLogin) VALUES (?, ?, ?, ?, ?)"
const createUserRoleQuery = "INSERT INTO role (user_id,role) VALUES (?, ?)"

func CreateUserAccountInDatabase(username string, passwd string) (data.User, error) {
	newUser, err := createUser(username, passwd)
	if err != nil {
		return data.User{}, err
	}
	log.Printf("Creating new user %d: %s", newUser.OnlineID, username)
	_, err = db.Exec(createUserQuery, newUser.OnlineID, newUser.Username, newUser.Password, newUser.Created, newUser.LastLogin)
	if err != nil {
		return data.User{}, err
	}
	newUser, err = GetUser(newUser.OnlineID)
	if err != nil {
		return data.User{}, err
	}
	// User can have more than a single role -> second table
	log.Printf("trying to add %s", newUser.OnlineID)
	_, err = db.Exec(createUserRoleQuery, newUser.OnlineID, newUser.Role)
	if err != nil {
		return data.User{}, err
	}
	return newUser, nil
}

const updateLoginTimeQuery = "UPDATE shoppers SET lastLogin = CURRENT_TIMESTAMP WHERE id = ?"

func ModifyLastLogin(id int64) (data.User, error) {
	log.Printf("Updating the last login time for %d to now", id)
	err := userExists(id)
	if err != nil {
		return data.User{}, err
	}
	_, err = db.Exec(updateLoginTimeQuery, id)
	if err != nil {
		log.Printf("Failed to update last login for user %d: %s", id, err)
		return data.User{}, err
	}
	user, _ := GetUser(id)
	return user, nil
}

const updateUsernameQuery = "UPDATE shoppers SET username = ? WHERE id = ?"

func ModifyUserAccountName(id int64, newUsername string) (data.User, error) {
	user, err := GetUser(id)
	if err != nil {
		return data.User{}, err
	}
	user.Username = newUsername
	_, err = db.Exec(updateUsernameQuery, user.Username, user.OnlineID)
	if err != nil {
		return data.User{}, err
	}
	return user, nil
}

const updatePasswordQuery = "UPDATE shoppers SET passwd = ? WHERE id = ?"

func ModifyUserAccountPassword(id int64, password string) (data.User, error) {
	user, err := GetUser(id)
	if err != nil {
		return data.User{}, err
	}
	hashedPw, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Printf("Failed to hash given password: %s", err)
		return data.User{}, err
	}
	user.Password = hashedPw
	_, err = db.Exec(updatePasswordQuery, user.Password, user.OnlineID)
	if err != nil {
		return data.User{}, err
	}
	return user, nil
}

const deleteUserQuery = "DELETE FROM shoppers WHERE id = ?"

func DeleteUserAccount(id int64) error {
	_, err := db.Exec(deleteUserQuery, id)
	if err != nil {
		return err
	}
	return nil
}

const dropUserTableQuery = "DELETE FROM shoppers"

func DropUserTable() {
	log.Print("RESETTING ALL USERS. THIS DISABLES LOGIN FOR ALL EXISTING USERS")

	_, err := db.Exec(dropUserTableQuery)
	if err != nil {
		return
	}

	log.Print("USER TABLE DROPPED")
}

// ------------------------------------------------------------
// The shopping list handling
// ------------------------------------------------------------

const getShoppingListQuery = "SELECT * FROM shopping_list WHERE listId = ? AND createdBy = ?"

func GetRawShoppingListWithId(listId int64, createdBy int64) (data.List, error) {
	row := db.QueryRow(getShoppingListQuery, listId, createdBy)
	var list data.List
	if err := row.Scan(&list.ListId, &list.CreatedBy.ID, &list.Title, &list.CreatedAt, &list.LastUpdated, &list.Version); errors.Is(err, sql.ErrNoRows) {
		return data.List{}, err
	}
	user, err := GetUser(createdBy)
	if err != nil {
		log.Printf("List Creator not found: %s", err)
		return data.List{}, err
	}
	list.CreatedBy.Name = user.Username
	return list, nil
}

const getAllShoppingListForUserQuery = "SELECT * FROM shopping_list WHERE createdBy = ?"

func GetRawShoppingListsForUserId(id int64) ([]data.List, error) {
	rows, err := db.Query(getAllShoppingListForUserQuery, id)
	if err != nil {
		return []data.List{}, err
	}
	defer rows.Close()
	user, err := GetUser(id)
	if err != nil {
		return []data.List{}, err
	}
	var lists []data.List
	for rows.Next() {
		var list data.List
		if err := rows.Scan(&list.ListId, &list.CreatedBy.ID, &list.Title, &list.CreatedAt, &list.LastUpdated, &list.Version); err != nil {
			return []data.List{}, err
		}
		list.CreatedBy.Name = user.Username
		lists = append(lists, list)
	}
	return lists, nil
}

const getShoppingListsById = "SELECT * FROM shopping_list WHERE (listId, createdBy) IN ((?, ?))"

func GetRawShoppingListsByIDs(listIds []data.ListPK) ([]data.List, error) {
	rows, err := db.Query(getShoppingListsById, listIds)
	if err != nil {
		return []data.List{}, err
	}
	defer rows.Close()
	var lists []data.List
	for rows.Next() {
		var list data.List
		if err := rows.Scan(&list.ListId, &list.CreatedBy.ID, &list.Title, &list.CreatedAt, &list.LastUpdated, &list.Version); err != nil {
			return []data.List{}, err
		}
		user, err := GetUser(list.CreatedBy.ID)
		if err != nil {
			return []data.List{}, err
		}
		list.CreatedBy.Name = user.Username
		lists = append(lists, list)
	}
	return lists, nil
}

const getAllShoppingListQuery = "SELECT * FROM shopping_list"

func GetAllRawShoppingLists() ([]data.List, error) {
	rows, err := db.Query(getAllShoppingListQuery)
	if err != nil {
		return []data.List{}, err
	}
	defer rows.Close()
	var lists []data.List
	for rows.Next() {
		var list data.List
		if err := rows.Scan(&list.ListId, &list.CreatedBy.ID, &list.Title, &list.CreatedAt, &list.LastUpdated, &list.Version); err != nil {
			return []data.List{}, err
		}
		user, err := GetUser(list.CreatedBy.ID)
		if err != nil {
			return []data.List{}, err
		}
		list.CreatedBy.Name = user.Username
		lists = append(lists, list)
	}
	return lists, nil
}

const getSharedWithShoppingListQuery = "SELECT * FROM shopping_list WHERE (listId, createdBy) IN ((?, ?))"

func GetShoppingListsFromSharedListIds(sharedLists []data.ListShared) ([]data.List, error) {
	if len(sharedLists) == 0 {
		return []data.List{}, errors.New("no shared ids given")
	}
	// Extract the list ids so we can query them
	// Join the IDs followed by the createdBy to make a fitting query
	// log.Printf("Shared list: %v", sharedLists)
	listIds := make([]interface{}, 0)
	for _, shared := range sharedLists {
		listIds = append(listIds, strconv.FormatInt(shared.ListId, 10))
		listIds = append(listIds, strconv.FormatInt(shared.CreatedBy, 10))
	}
	// log.Printf("Searching for %d lists: %v", len(listIds), listIds)
	query := getSharedWithShoppingListQuery
	if len(listIds) > 1 {
		getSharedWithShoppingListQueryInAppendableFormat := strings.TrimSuffix(getSharedWithShoppingListQuery, ")")
		query = getSharedWithShoppingListQueryInAppendableFormat + strings.Repeat(",(?,?)", len(sharedLists)-1) + ")"
	}
	// log.Printf("Query string: %s", query)
	rows, err := db.Query(query, listIds...)
	if err != nil {
		sharedWithId := -1
		if len(sharedLists) > 0 {
			sharedWithId = int(sharedLists[0].SharedWithId)
		}
		log.Printf("Failed to retrieve any shared list for user %d: %s", sharedWithId, err)
		return []data.List{}, err
	}
	defer rows.Close()
	var lists []data.List
	for rows.Next() {
		var list data.List
		if err := rows.Scan(&list.ListId, &list.CreatedBy.ID, &list.Title, &list.CreatedAt, &list.LastUpdated, &list.Version); err != nil {
			return []data.List{}, err
		}
		creatorInfo, err := GetUser(list.CreatedBy.ID)
		if err != nil {
			log.Printf("Cannot find list creator %d and skip: %s", list.CreatedBy.ID, err)
			continue
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

const updateRawShoppingListQuery = "UPDATE shopping_list SET name = ?, lastEdited = CURRENT_TIMESTAMP, version = ? WHERE listId = ? AND createdBy = ?"

func updateRawShoppingList(list data.List) (data.List, error) {
	existingList, err := GetRawShoppingListWithId(list.ListId, list.CreatedBy.ID)
	if err != nil {
		return data.List{}, err
	}
	if err := checkListCorrect(list); err != nil {
		log.Printf("List not in correct format for insertion: %s", err)
		return data.List{}, err
	}
	if existingList.Version >= list.Version {
		return data.List{}, errors.New("newer list exists")
	}
	_, err = db.Exec(updateRawShoppingListQuery, list.Title, list.Version, list.ListId, list.CreatedBy.ID)
	return list, err
}

const createRawShoppingListQuery = "INSERT INTO shopping_list (listId,createdBy,name,created,lastEdited,version) VALUES (?, ?, ?, ?, ?, ?)"

func createRawShoppingList(list data.List) error {
	if err := checkListCorrect(list); err != nil {
		log.Printf("List not in correct format for insertion: %s", err)
		return err
	}
	result, err := db.Exec(createRawShoppingListQuery, list.ListId, list.CreatedBy.ID, list.Title, list.CreatedAt, list.LastUpdated, list.Version)
	if err != nil {
		return err
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
	if err := createRawShoppingList(list); err != nil {
		_, err = updateRawShoppingList(list)
		if err != nil {
			return err
		}
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

const deleteShoppingListQuery = "DELETE FROM shopping_list WHERE listId = ? AND createdBy = ?"

func DeleteShoppingList(id int64, createdBy int64) error {
	_, err := db.Exec(deleteShoppingListQuery, id, createdBy)
	if err != nil {
		return err
	}
	return nil
}

const deleteAllShoppingListFromQuery = "DELETE FROM shopping_list WHERE createdBy = ?"

func DeleteShoppingListFrom(createdBy int64) error {
	_, err := db.Exec(deleteAllShoppingListFromQuery, createdBy)
	if err != nil {
		return err
	}
	return nil
}

const dropShoppingListTableQuery = "DELETE FROM shopping_list"

func DropShoppingListTable() {
	log.Print("DROPPING SHOPPING LIST TABLE. CANNOT BE REVERTED!")

	_, err := db.Exec(dropShoppingListTableQuery)
	if err != nil {
		return
	}

	log.Print("DROPPED SHOPPING TABLE")
}

// ------------------------------------------------------------

const listIsSharedWithUser = "SELECT * FROM shared_list WHERE sharedWithId IN (?, -1)"

func GetListIdsSharedWithUser(userId int64) ([]data.ListPK, error) {
	rows, err := db.Query(listIsSharedWithUser, userId)
	if err != nil {
		return []data.ListPK{}, err
	}
	var list []data.ListPK
	for rows.Next() {
		var shared data.ListPK
		if err := rows.Scan(&shared.ListID, &shared.CreatedBy); err != nil {
			return []data.ListPK{}, err
		}
		list = append(list, shared)
	}
	return list, nil
}

const istListSharedWithUserQuery = "SELECT * FROM shared_list WHERE listId = ? AND createdBy = ? AND sharedWithId = ?"

func IsListSharedWithUser(listId int64, createdBy int64, userId int64) error {
	rows, err := db.Query(istListSharedWithUserQuery, listId, createdBy, userId)
	if err != nil {
		log.Printf("The list %d is not shared with the user %d: %s", listId, userId, err)
		return err
	}
	defer rows.Close()
	counter := 1
	correctUserIdContained := false
	for rows.Next() {
		if counter > 1 {
			log.Printf("Expected only a single shared row but got more than one!")
		}
		var shared data.ListShared
		if err := rows.Scan(&shared.ListId, &shared.CreatedBy, &shared.SharedWithId, &shared.Created); err != nil {
			return err
		}
		if shared.SharedWithId == userId {
			correctUserIdContained = true
		}
		counter += 1
	}
	if counter == 1 || correctUserIdContained != true {
		return errors.New("list is not shared with user")
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
	_, err = GetRawShoppingListWithId(listId, createdBy)
	if err != nil {
		return errors.New("shared list does not exist")
	}
	return nil
}

const createShoppingListSharingForUserQuery = "INSERT INTO shared_list (listId,createdBy,sharedWithId,created) VALUES (?,?,?,CURRENT_TIMESTAMP)"

func CreateOrUpdateSharedList(listId int64, createdBy int64, sharedWith int64) (data.ListShared, error) {
	err := IsListSharedWithUser(listId, createdBy, sharedWith)
	if err == nil {
		log.Printf("Shared of list %d for user %d exists", listId)
		return data.ListShared{ListId: listId, CreatedBy: createdBy, SharedWithId: sharedWith, Created: time.Now()}, nil
	}
	if err := CheckUserAndListExist(listId, createdBy, sharedWith); err != nil {
		log.Printf("User or list does not exist: %s", err)
		return data.ListShared{}, err
	}
	_, err = db.Exec(createShoppingListSharingForUserQuery, listId, createdBy, sharedWith)
	if err != nil {
		log.Printf("Failed to insert sharing into database: %s", err)
		return data.ListShared{}, err
	}
	newShared := data.ListShared{ListId: listId, CreatedBy: createdBy, SharedWithId: sharedWith, Created: time.Now()}
	return newShared, nil
}

const deleteSharingOfShoppingListQuery = "DELETE FROM shared_list WHERE listId = ? AND createdBy = ?"

func DeleteSharingOfList(listId int64, createdBy int64) error {
	_, err := db.Exec(deleteSharingOfShoppingListQuery, listId, createdBy)
	if err != nil {
		log.Printf("Failed to delete sharing of list %d: %s", listId, err)
		return err
	}
	return nil
}

const deleteShoppingListSharingForUserQuery = "DELETE FROM shared_list WHERE listId = ? AND createdBy = ? AND sharedWithId = ?"

func DeleteSharingForUser(listId int64, createdBy int64, userId int64) error {
	_, err := db.Exec(deleteShoppingListSharingForUserQuery, listId, createdBy, userId)
	if err != nil {
		log.Printf("Failed to delete sharing for user %d of list %d: %s", userId, listId, err)
		return err
	}
	return nil
}

const dropShoppingListSharedTable = "DELETE FROM shared_list"

func ResetSharedListTable() {
	log.Print("RESETTING SHARING LIST. CANNOT BE REVERTED!")

	_, err := db.Exec(dropShoppingListSharedTable)
	if err != nil {
		log.Printf("Failed to remove all sharing from table: %s", err)
		return
	}

	log.Print("RESET SHARING TABLE")
}

// ------------------------------------------------------------

const doesItemMappingExistQuery = "SELECT * FROM item_per_list WHERE listId = ? AND createdBy = ? AND itemId = ?"

func IsItemInList(listId int64, createdBy int64, itemId int64) (data.ListItem, error) {
	row := db.QueryRow(doesItemMappingExistQuery, listId, itemId, createdBy)
	var mapping data.ListItem
	if err := row.Scan(&mapping.ListId, &mapping.CreatedBy, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.AddedBy); errors.Is(err, sql.ErrNoRows) {
		return data.ListItem{}, err
	}
	return mapping, nil
}

const getItemsInListQuery = "SELECT it.name,it.icon,map.quantity,map.checked,map.addedBy FROM items_per_list map INNER JOIN items it ON map.itemId = it.id WHERE listId = ? AND createdBy = ?"

func GetItemsInList(listId int64, createdBy int64) ([]data.ItemWire, error) {
	rows, err := db.Query(getItemsInListQuery, listId, createdBy)
	if err != nil {
		log.Printf("Failed to query for items contained in list %d: %s", listId, err)
		return []data.ItemWire{}, nil
	}
	defer rows.Close()
	var list []data.ItemWire
	for rows.Next() {
		var item data.ItemWire
		if err := rows.Scan(&item.Name, &item.Icon, &item.Quantity, &item.Checked, &item.AddedBy); err != nil {
			return []data.ItemWire{}, err
		}
		list = append(list, item)
	}
	return list, nil
}

const updateItemMappingQuery = "UPDATE item_per_list SET quantity = ?, checked = ?, addedBy = ? WHERE listId = ? AND createdBy = ? AND itemId = ?"
const insertItemMappingQuery = "INSERT INTO item_per_list (listId,createdBy,itemId,quantity,checked,addedBy) VALUES (?, ?, ?, ?, ?, ?)"

func InsertOrUpdateItemInList(mapping data.ListItem) (data.ListItem, error) {
	update := false
	existingItemMapping, err := IsItemInList(mapping.ListId, mapping.CreatedBy, mapping.ItemId)
	if err == nil {
		update = true
	}
	if update {
		_, err := db.Exec(updateItemMappingQuery, mapping.Quantity, mapping.Checked, mapping.AddedBy, mapping.ListId, mapping.CreatedBy, existingItemMapping.ItemId)
		if err != nil {
			return data.ListItem{}, err
		}
		return mapping, nil
	}
	_, err = db.Exec(insertItemMappingQuery, mapping.ListId, mapping.CreatedBy, mapping.ItemId, mapping.Quantity, mapping.Checked, mapping.AddedBy)
	if err != nil {
		return data.ListItem{}, err
	}
	return mapping, nil
}

const deleteShoppingListMappingQuery = "DELETE FROM item_per_list WHERE listId = ? AND createdBy = ? AND sharedWithId = ?"

func DeleteItemInList(listId int64, createdBy int64, itemId int64) error {
	_, err := db.Exec(deleteShoppingListMappingQuery, listId, createdBy, itemId)
	if err != nil {
		log.Printf("Failed to delete item %d in list: %s", itemId, err)
		return err
	}
	return nil
}

const deleteAllShoppingListMappingsForListQuery = "DELETE FROM item_per_list WHERE listId = ? AND createdBy = ?"

func DeleteAllItemsInList(listId int64, createdBy int64) error {
	_, err := db.Exec(deleteAllShoppingListMappingsForListQuery, listId, createdBy)
	if err != nil {
		log.Printf("Failed to delete list %d: %s", listId, err)
		return err
	}
	return nil
}

const dropItemPerListTable = "DELETE FROM item_per_list"

func ResetItemPerListTable() {
	log.Print("RESETTING ALL ITEMS PER LIST. CANNOT BE REVERTED!")

	_, err := db.Exec(dropItemPerListTable)
	if err != nil {
		log.Printf("Failed to remove mappings from table: %s", err)
		return
	}

	log.Print("RESET ITEM MAPPING TABLE")
}

// ------------------------------------------------------------
// Item Handling
// ------------------------------------------------------------

const getItemQuery = "SELECT * FROM items WHERE id = ?"

func GetItem(id int64) (data.Item, error) {
	if id < 0 {
		err := errors.New("items with id < 0 do not exist")
		return data.Item{}, err
	}
	row := db.QueryRow(getItemQuery, id)
	var item data.Item
	if err := row.Scan(&item.ItemId, &item.Name, &item.Icon); err != nil {
		return data.Item{}, err
	}
	return item, nil
}

const getAllItemsQuery = "SELECT * FROM items"

func GetAllItems() ([]data.Item, error) {
	rows, err := db.Query(getAllItemsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

const getAllItemsFromNameQuery = "SELECT * FROM items WHERE name LIKE '%?%'"

func GetAllItemsFromName(name string) ([]data.Item, error) {
	rows, err := db.Query(getAllItemsFromNameQuery, name)
	if err != nil {
		log.Printf("Failed to query database for items: %s", err)
		return nil, err
	}
	defer rows.Close()
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

const insertItemQuery = "INSERT INTO items (name, icon) SELECT ?,? WHERE NOT EXISTS (SELECT 1 FROM items WHERE name = ?);"

// TODO: Does this return the row number of the existing item if nothing changed?
func InsertItemStruct(item data.Item) (int64, error) {
	trimmedName := strings.TrimSpace(item.Name)
	trimmedIcon := strings.TrimSpace(item.Icon)
	result, err := db.Exec(insertItemQuery, trimmedName, trimmedIcon, trimmedName)
	if err != nil {
		return -1, err
	}
	id, err := result.RowsAffected()
	if err != nil || id == 0 {
		log.Printf("Failed to insert item into database: %s", err)
		return 0, err
	}
	return id, nil
}

const updateItemNameQuery = "UPDATE items SET name = ?, icon = ? WHERE id = ?"

func ModifyItem(id int64, name string, icon string) (data.Item, error) {
	item, err := GetItem(id)
	if err != nil {
		return data.Item{}, err
	}
	if name != "" {
		item.Name = name
	}
	if icon != "" {
		item.Icon = icon
	}
	result, err := db.Exec(updateItemNameQuery, item.Name, item.Icon, item.ItemId)
	if err != nil {
		return data.Item{}, err
	}
	item.ItemId, err = result.RowsAffected()
	if err != nil {
		log.Printf("Problem with ID during insertion of item: %s", err)
		return data.Item{}, err
	}
	return item, err
}

const deleteItemQuery = "DELETE FROM items WHERE id = ?"

func DeleteItem(id int64) error {
	_, err := db.Exec(deleteItemQuery, id)
	if err != nil {
		return err
	}
	return nil
}

const dropItemTable = "DELETE FROM items"

func ResetItemTable() {
	log.Print("RESETTING ALL ITEMS. CANNOT BE REVERTED!")

	_, err := db.Exec(dropItemTable)
	if err != nil {
		log.Printf("Failed to remove all items from table: %s", err)
		return
	}

	log.Print("RESET ITEMS TABLE")
}

// ------------------------------------------------------------
// Recipes Handling
// ------------------------------------------------------------

const createRawRecipeQuery = "INSERT INTO recipe (recipeId,createdBy,name,createdAt,lastUpdate,version,defaultPortion) VALUES (?,?,?,?,?,?,?)"

func CreateRecipe(recipe data.Recipe) error {
	_, err := db.Exec(createRawRecipeQuery, recipe.RecipeId, recipe.CreatedBy.ID, recipe.Name, recipe.CreatedAt, recipe.LastUpdate, recipe.Version, recipe.DefaultPortion)
	if err != nil {
		log.Printf("Failed to insert values into database: %s", err)
		return err
	}
	err = insertDescriptions(recipe.RecipeId, recipe.CreatedBy.ID, recipe.Description)
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

const insertRecipeDescriptionQuery = "INSERT INTO description_per_recipe (recipeId,createdBy,descriptionOrder,description) VALUE (?,?,?,?)"

func insertDescriptions(recipeId int64, createdBy int64, descriptions []data.RecipeDescription) error {
	log.Printf("Inserting %d recipe descriptions", len(descriptions))
	query := insertRecipeDescriptionQuery
	if len(descriptions) > 1 {
		query = insertRecipeDescriptionQuery + strings.Repeat(",(?,?,?,?)", len(descriptions)-1)
	}
	flattenedParameter := make([]interface{}, 0)
	for _, v := range descriptions {
		flattenedParameter = append(flattenedParameter, recipeId)
		flattenedParameter = append(flattenedParameter, createdBy)
		flattenedParameter = append(flattenedParameter, v.Order)
		flattenedParameter = append(flattenedParameter, v.Step)
	}
	_, err := db.Exec(query, flattenedParameter...)
	return err
}

const insertRecipeIngredientsQuery = "INSERT INTO ingredient_per_recipe (recipeId,createdBy,itemId,quantity,quantityType) VALUES (?,?,?,?,?)"

func insertIngredients(recipeId int64, createdBy int64, ingredients []data.Ingredient) error {
	log.Printf("Insert %d recipe ingredients", len(ingredients))
	// We need to check if an item exists and reference this item rather than creating a new one
	query := insertRecipeIngredientsQuery
	if len(ingredients) > 1 {
		query = insertRecipeIngredientsQuery + strings.Repeat(",(?,?,?,?,?)", len(ingredients)-1)
	}
	flattenedParameter := make([]interface{}, 0)
	for _, v := range ingredients {
		itemId, err := InsertItem(v.Name, v.Icon)
		if err != nil {
			return err
		}
		flattenedParameter = append(flattenedParameter, recipeId)
		flattenedParameter = append(flattenedParameter, createdBy)
		flattenedParameter = append(flattenedParameter, itemId)
		flattenedParameter = append(flattenedParameter, v.Quantity)
		flattenedParameter = append(flattenedParameter, v.QuantityType)
	}
	_, err := db.Exec(query, flattenedParameter...)
	return err
}

const getIngredientsForRecipeQuery = "SELECT it.name,it.icon,map.quantity,map.quantityType FROM ingredient_per_recipe map JOIN items it ON map.itemId = it.id WHERE map.recipeId = ? AND map.createdBy = ?"

func GetIngredientsForRecipe(recipeId int64, createdBy int64) ([]data.Ingredient, error) {
	rows, err := db.Query(getIngredientsForRecipeQuery, recipeId, createdBy)
	if err != nil {
		log.Printf("Failed to retrieve ingredients for recipe %d from %d", recipeId, createdBy)
		return []data.Ingredient{}, err
	}
	defer rows.Close()
	var ingredients []data.Ingredient
	for rows.Next() {
		var ingredient data.Ingredient
		if err := rows.Scan(&ingredient.Name, &ingredient.Icon, &ingredient.Quantity, &ingredient.QuantityType); err != nil {
			return []data.Ingredient{}, err
		}
		ingredients = append(ingredients, ingredient)
	}
	return ingredients, nil
}

const getDescriptionsForRecipeQuery = "SELECT descriptionOrder,description FROM description_per_recipe WHERE recipeId = ? AND createdBy = ?"

func GetDescriptionsForRecipe(recipeId int64, createdBy int64) ([]data.RecipeDescription, error) {
	rows, err := db.Query(getDescriptionsForRecipeQuery, recipeId, createdBy)
	if err != nil {
		return []data.RecipeDescription{}, err
	}
	defer rows.Close()
	var descriptions []data.RecipeDescription
	for rows.Next() {
		var description data.RecipeDescription
		if err := rows.Scan(&description.Order, &description.Step); err != nil {
			return []data.RecipeDescription{}, err
		}
		descriptions = append(descriptions, description)
	}
	return descriptions, nil
}

const getRawRecipeQuery = "SELECT recipeId,createdBy,name,createdAt,lastUpdate,version,defaultPortion FROM recipe WHERE recipeId = ? AND createdBy = ?"

func GetRecipe(recipeId int64, createdBy int64) (data.Recipe, error) {
	row := db.QueryRow(getRawRecipeQuery, recipeId, createdBy)
	var recipe data.Recipe
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

const getAllRawRecipesQuery = "SELECT recipeId,createdBy,name,createdAt,lastUpdate,version,defaultPortion FROM recipe"

func GetAllRecipes() ([]data.Recipe, error) {
	rows, err := db.Query(getAllRawRecipesQuery)
	if err != nil {
		log.Printf("Failed to retrieve all recipes: %s", err)
		return nil, err
	}
	defer rows.Close()
	recipes := make([]data.Recipe, 0)
	for rows.Next() {
		var recipe data.Recipe
		if err := rows.Scan(&recipe.RecipeId, &recipe.CreatedBy.ID, &recipe.Name, &recipe.CreatedAt, &recipe.LastUpdate, &recipe.Version, &recipe.DefaultPortion); err != nil {
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
	err = insertDescriptions(recipeId, createdBy, descriptions)
	return err
}

const updateRawRecipeQuery = "UPDATE recipe SET version = ?, name = ?, lastUpdate = CURRENT_TIMESTAMP WHERE recipeId = ? AND createdBy = ?"

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
	_, err = db.Exec(updateRawRecipeQuery, recipe.Version, recipe.Name, recipe.RecipeId, recipe.CreatedBy.ID)
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

const deleteIngredientsForRecipeQuery = "DELETE FROM ingredient_per_recipe WHERE recipeId = ? AND createdBy = ?"

func deleteIngredients(recipeId int64, createdBy int64) error {
	_, err := db.Exec(deleteIngredientsForRecipeQuery, recipeId, createdBy)
	return err
}

const deleteDescriptionsForRecipeQuery = "DELETE FROM description_per_recipe WHERE recipeId = ? AND createdBy = ?"

func deleteDescriptions(recipeId int64, createdBy int64) error {
	_, err := db.Exec(deleteDescriptionsForRecipeQuery, recipeId, createdBy)
	return err
}

const deleteRawRecipeQuery = "DELETE FROM recipe WHERE recipeId = ? AND createdBy = ?"

func DeleteRecipe(recipeId int64, createdBy int64) error {
	if err := deleteDescriptions(recipeId, createdBy); err != nil {
		return err
	}
	if err := deleteIngredients(recipeId, createdBy); err != nil {
		return err
	}
	_, err := db.Exec(deleteRawRecipeQuery, recipeId, createdBy)
	if err != nil {
		log.Printf("Failed to delete recipe %d from %d: %s", recipeId, createdBy, err)
		return err
	}
	return nil
}

const dropRecipeTable = "DELETE FROM recipe"

func ResetRecipeTables() {
	log.Print("RESETTING ALL RECIPES. CANNOT BE REVERTED!")

	_, err := db.Exec(dropRecipeTable)
	if err != nil {
		log.Printf("Failed to remove all recipes: %s", err)
		return
	}

	log.Print("RESET RECIPE TABLES")
}

// ------------------------------------------------------------
// Recipe Sharing Handling
// ------------------------------------------------------------

const createRecipeSharingQuery = "INSERT INTO shared_recipe (recipeId,createdBy,sharedWith) VALUES (?,?,?)"

func CreateRecipeSharing(recipeId int64, createdBy int64, sharedWith int64) error {
	log.Printf("Creating new sharing for %d of recipe %d from %d", sharedWith, recipeId, createdBy)
	_, err := db.Exec(createRecipeSharingQuery, recipeId, createdBy, sharedWith)
	if err != nil {
		log.Printf("Failed to insert sharing for user %d into database: %s", sharedWith, err)
		return err
	}
	return nil
}

const deleteRecipeSharingForIdQuery = "DELETE FROM shared_recipe WHERE recipeId = ? AND createdBy = ? AND sharedWith = ?"

func DeleteRecipeSharing(recipeId int64, createdBy int64, sharedWith int64) error {
	log.Printf("Deleting sharing for %d of recipe %d from %d", sharedWith, recipeId, createdBy)
	_, err := db.Exec(deleteRecipeSharingForIdQuery, recipeId, createdBy, sharedWith)
	if err != nil {
		log.Printf("Failed to delete sharing for user %d from recipe %d: %s", sharedWith, recipeId, err)
		return err
	}
	return nil
}

const deleteRecipeSharingForAllQuery = "DELETE FROM shared_recipe WHERE recipeId = ? AND createdBy = ?"

func DeleteAllSharingForRecipe(recipeId int64, createdBy int64) error {
	log.Printf("Deleting all sharing for recipe %d from %d", recipeId, createdBy)
	_, err := db.Exec(deleteRecipeSharingForAllQuery, recipeId, createdBy)
	if err != nil {
		log.Printf("Failed to delete all sharing for recipe %d from %d: %s", recipeId, createdBy, err)
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

const printShoppingListTableQuery = "SELECT * FROM shopping_list"

func PrintShoppingListTable() {
	rows, err := db.Query(printShoppingListTableQuery)
	if err != nil {
		return
	}
	defer rows.Close()
	log.Print("------------- Shopping List Table -------------")
	for rows.Next() {
		var list data.List
		if err := rows.Scan(&list.ListId, &list.CreatedBy.ID, &list.Title, &list.CreatedAt, &list.LastUpdated, &list.Version); err != nil {
		}
		log.Printf("%v", list)
	}
	log.Print("---------------------------------------")
}

const printItemTableQuery = "SELECT * FROM items"

func PrintItemTable() {
	rows, err := db.Query(printItemTableQuery)
	if err != nil {
		return
	}
	defer rows.Close()
	log.Print("------------- Item Table -------------")
	for rows.Next() {
		var item data.Item
		if err := rows.Scan(&item.ItemId, &item.Name, &item.Icon); err != nil {
		}
		log.Printf("%v", item)
	}
	log.Print("---------------------------------------")
}

const printItemToShoppingListMappingTableQuery = "SELECT * FROM items_per_list"

func PrintItemPerListTable() {
	rows, err := db.Query(printItemToShoppingListMappingTableQuery)
	if err != nil {
		return
	}
	defer rows.Close()
	log.Print("------------- Item Table -------------")
	for rows.Next() {
		var mapping data.ListItem
		if err := rows.Scan(&mapping.ListId, &mapping.CreatedBy, &mapping.ItemId, &mapping.Quantity, &mapping.Checked, &mapping.AddedBy); err != nil {
		}
		log.Printf("%v", mapping)
	}
	log.Print("---------------------------------------")
}

const printShoppingListSharingTableQuery = "SELECT * FROM shared_list"

func PrintSharingTable() {
	rows, err := db.Query(printShoppingListSharingTableQuery)
	if err != nil {
		return
	}
	defer rows.Close()
	log.Print("------------- Sharing Table -------------")
	for rows.Next() {
		var sharing data.ListShared
		if err := rows.Scan(&sharing.ListId, &sharing.CreatedBy, &sharing.SharedWithId, &sharing.Created); err != nil {
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
