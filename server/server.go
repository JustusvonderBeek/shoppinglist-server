package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"shop.cloudsheeptech.com/database"
	"shop.cloudsheeptech.com/server/authentication"
	"shop.cloudsheeptech.com/server/configuration"
	"shop.cloudsheeptech.com/server/data"
)

// ------------------------------------------------------------
// Handling of user information
// ------------------------------------------------------------

func updateUserinfo(c *gin.Context) {
	var user data.User
	err := c.BindJSON(&user)
	if err != nil {
		log.Printf("Failed to parse updated user information: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// User already found in our database. Simply update this stuff
	user, err = database.ModifyUserAccountName(user.ID, user.Username)
	if err != nil {
		log.Printf("Failed to insert user into database: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, user)
}

func getUserInfos(c *gin.Context) {
	sUserId := c.Param("userId")
	queriedUserId, err := strconv.Atoi(sUserId)
	if err != nil {
		log.Printf("Failed to parse given item id: %s", sUserId)
		log.Printf("Err: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	user, err := database.GetUserInWireFormat(int64(queriedUserId))
	if err != nil {
		log.Printf("Queried user %d does not exist", queriedUserId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, user)
}

func getMatchingUsers(c *gin.Context) {
	sUsername := c.Param("name")
	users, err := database.GetUserFromMatchingUsername(sUsername)
	if err != nil {
		log.Printf("Failed to retrieve matching users: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	log.Printf("Found %d matching users", len(users))
	c.JSON(http.StatusOK, users)
}

// ------------------------------------------------------------
// Handling of lists
// ------------------------------------------------------------

func getShoppingListsForUser(c *gin.Context) {
	sUserId := c.Param("userId")
	id, err := strconv.Atoi(sUserId)
	if err != nil {
		log.Printf("Failed to parse given item id: %s", sUserId)
		log.Printf("Err: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// User MUST be authenticated so it does exist and is allowed to make the request
	// Check for the lists of the user itself first
	lists, err := database.GetShoppingListsFromUserId(int64(id))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	log.Printf("Got %d lists for user itself", len(lists))

	// Asking the database for all the lists that are shared with the current user
	sharedInfo, err := database.GetSharedListForUserId(int64(id))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	log.Printf("Got %d shared lists for user", len(sharedInfo))
	// Get full information for the shared lists
	sharedLists, err := database.GetShoppingListsFromSharedListIds(sharedInfo)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	lists = append(lists, sharedLists...)
	// Asking DB to get the items in this list
	for i, list := range lists {
		itemsPerList, err := database.GetItemsInList(list.ListId, list.CreatedBy.ID)
		if err != nil {
			log.Printf("Failed to get items for list %d: %s", list.ListId, err)
			lists[i].Items = []data.ItemWire{}
			continue
		}
		if len(itemsPerList) == 0 {
			lists[i].Items = []data.ItemWire{}
			continue
		}
		// Unpack and convert into wire format
		for _, item := range itemsPerList {
			dbItem, err := database.GetItem(item.ItemId)
			if err != nil {
				log.Printf("Failed to get information for item %d in list", i)
				continue
			}
			wireItem := data.ItemWire{
				Name:     dbItem.Name,
				Icon:     dbItem.Icon,
				Quantity: item.Quantity,
				Checked:  item.Checked,
				AddedBy:  item.AddedBy,
			}
			lists[i].Items = append(lists[i].Items, wireItem)
		}
	}

	c.JSON(http.StatusOK, lists)
}

func getShoppingList(c *gin.Context) {
	sId := c.Param("id")
	id, err := strconv.Atoi(sId)
	if err != nil {
		log.Printf("Failed to parse given item id: %s", sId)
		log.Printf("Err: %s", err)
		return
	}
	stored, exists := c.Get("userId")
	if !exists {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userId, ok := stored.(int)
	if !ok {
		log.Print("Internal server error: stored value is not correct")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	list, err := database.GetShoppingList(int64(id), int64(userId))
	if err != nil {
		log.Printf("Failed to get mapping for id %d: %s", id, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	itemsInList, err := database.GetItemsInList(list.ListId, list.CreatedBy.ID)
	if err != nil {
		log.Printf("Failed to get item in list: %s", err)
		c.JSON(http.StatusInternalServerError, list)
		return
	}
	for _, item := range itemsInList {
		dbItem, err := database.GetItem(item.ItemId)
		if err != nil {
			log.Printf("Failed to find item '%d' in database", item.ItemId)
			continue
		}
		listItem := data.ItemWire{
			Name:     dbItem.Name,
			Icon:     dbItem.Icon,
			Quantity: item.Quantity,
			Checked:  item.Checked,
		}
		list.Items = append(list.Items, listItem)
	}
	c.JSON(http.StatusOK, list)
}

func postShoppingList(c *gin.Context) {
	var list data.Shoppinglist
	err := c.BindJSON(&list)
	if err != nil {
		log.Printf("Failed to convert given data to shopping list: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	stored, exists := c.Get("userId")
	if !exists {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userId, ok := stored.(int)
	if !ok {
		log.Print("Internal server error: stored value is not correct")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if userId != int(list.CreatedBy.ID) || list.CreatedBy.ID == 0 {
		log.Print("The logged in user and the created by are not equal")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	err = database.CreateOrUpdateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create or update list: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// No more information is gained through the same data
	c.Status(http.StatusCreated)
}

func removeShoppingList(c *gin.Context) {
	sId := c.Param("id")
	listId, err := strconv.Atoi(sId)
	if err != nil {
		log.Printf("Failed to parse given listId: %s", sId)
		log.Printf("Err: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	stored, exists := c.Get("userId")
	if !exists {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userId, ok := stored.(int)
	if !ok {
		log.Print("Internal server error: stored value is not correct")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	list, err := database.GetShoppingList(int64(listId), int64(userId))
	if err != nil {
		log.Printf("Failed to get mapping for listId %d: %s", listId, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if int(list.CreatedBy.ID) != userId {
		log.Printf("Cannot delete list: User %d did not create list %d", userId, list.ListId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := database.DeleteShoppingList(int64(listId), int64(userId)); err != nil {
		log.Printf("Failed to delete list %d", listId)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	log.Printf("Delete list %d", listId)
	c.Status(http.StatusOK)
}

// ------------------------------------------------------------
// Handling of sharing
// ------------------------------------------------------------

func shareList(c *gin.Context) {
	sId := c.Param("id")
	id, err := strconv.Atoi(sId)
	if err != nil {
		log.Printf("Failed to parse given list id: %s: %s", sId, err)
		return
	}
	stored, exists := c.Get("userId")
	if !exists {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userId, ok := stored.(int)
	if !ok {
		log.Print("Internal server error: stored value is not correct")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	var shared data.ListSharedWire
	if err = c.BindJSON(&shared); err != nil {
		log.Printf("Failed to bind given data: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Check if the user owns this list?
	list, err := database.GetShoppingList(shared.ListId, int64(userId))
	if err != nil {
		log.Printf("Failed to retrieve list: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if list.ListId != int64(id) {
		log.Printf("IDs do not match!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if list.CreatedBy.ID != int64(userId) {
		log.Printf("User ID (%d) does not match created ID (%d)", userId, list.CreatedBy.ID)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	listShared, err := database.CreateOrUpdateSharedList(shared.ListId, shared.CreatedBy, shared.SharedWith)
	if err != nil {
		log.Printf("Failed to create sharing: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusCreated, listShared)
}

func unshareList(c *gin.Context) {
	sId := c.Param("id")
	id, err := strconv.Atoi(sId)
	if err != nil {
		log.Printf("Failed to parse given list id: %s: %s", sId, err)
		return
	}
	stored, exists := c.Get("userId")
	if !exists {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userId, ok := stored.(int)
	if !ok {
		log.Print("Internal server error: stored value is not correct")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	var unshare data.ListSharedWire
	if err := c.BindJSON(&unshare); err != nil {
		log.Print("Failed to deserialize share object")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if unshare.ListId != int64(id) {
		log.Print("Given list and unshare list do not match")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Check if the user owns this list?
	list, err := database.GetShoppingList(unshare.ListId, int64(userId))
	if err != nil {
		log.Printf("Failed to retrieve list: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if list.CreatedBy.ID != int64(userId) {
		log.Printf("User ID (%d) does not match created ID (%d)", userId, list.CreatedBy.ID)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := database.DeleteSharingForUser(unshare.ListId, unshare.CreatedBy, unshare.SharedWith); err != nil {
		log.Printf("Failed to delete sharing %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

// ------------------------------------------------------------
// Debug functionality
// ------------------------------------------------------------

func returnUnauth(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"status": "testing-content"})
}

func returnPostTest(c *gin.Context) {
	var item data.Item
	err := c.BindJSON(&item)
	if err != nil {
		log.Print("Require item to be send for testing")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"status": "post-successful"})
}

// ------------------------------------------------------------
// The main function
// ------------------------------------------------------------

func SetupRouter(cfg configuration.Config) *gin.Engine {
	gin.SetMode(gin.DebugMode)

	router := gin.Default()
	authentication.Setup(cfg)

	// ------------- Handling Account Creation and Login ---------------

	// Independent of API version, therefore not in the auth bracket
	router.POST("/auth/create", authentication.CreateAccount)
	// JWT BASED AUTHENTICATION
	router.POST("/auth/login", authentication.Login)

	// ------------- Handling Routes v1 (API version 1) ---------------

	// Add authentication middleware to v1 router
	authorized := router.Group("/v1")
	authorized.Use(authentication.AuthenticationMiddleware(cfg))
	{
		// Taking care of users which are registered but want to update their info
		authorized.PUT("/userinfo/:userId", updateUserinfo)
		authorized.GET("/userinfo/:userId", getUserInfos)
		authorized.GET("/users/:name", getMatchingUsers)
		authorized.DELETE("/users/:userId", authentication.DeleteAccount)

		// Handling the lists itself
		authorized.GET("/lists/:userId", getShoppingListsForUser) // Includes OWN and SHARED lists
		authorized.GET("/list/:id", getShoppingList)
		authorized.DELETE("/list/:id", removeShoppingList)

		// Includes both adding a new list and updating an existing list
		authorized.POST("/list", postShoppingList)

		// Handling the items per list
		// authorized.POST("/list/items", postItemsInList)

		// Handling sharing a list
		authorized.POST("/share/:id", shareList)
		authorized.DELETE("/share/:id", unshareList)

		// DEBUG Purpose: TODO: Disable when no longer testing
		authorized.GET("/test/auth", returnUnauth)
		authorized.POST("/test/auth", returnPostTest)
	}

	router.GET("/test/unauth", returnUnauth)

	return router
}

func Start(cfg configuration.Config) error {
	router := SetupRouter(cfg)

	// -------------------------------------------

	address := cfg.ListenAddr + ":" + cfg.ListenPort
	// Only allow TLS
	err := router.RunTLS(address, cfg.TLSCertificate, cfg.TLSKeyfile)
	return err
}
