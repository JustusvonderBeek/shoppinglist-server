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
	mapping, err := database.GetShoppingList(int64(id), int64(userId))
	if err != nil {
		log.Printf("Failed to get mapping for id %d: %s", id, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, mapping)
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
	if userId != int(list.CreatedBy) || list.CreatedBy == 0 {
		log.Print("The logged in user and the created by are not equal")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	err = database.CreateShoppingList(list)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// No more information is gained through the same data
	c.Status(http.StatusCreated)
}

// TODO:
func removeShoppingList(c *gin.Context) {
	log.Print("Not implemented")
}

// func postItemsInList(c *gin.Context) {
// 	stored, exists := c.Get("userId")
// 	if !exists {
// 		log.Printf("User is not correctly authenticated")
// 		c.AbortWithStatus(http.StatusUnauthorized)
// 		return
// 	}
// 	userId, ok := stored.(int)
// 	if !ok {
// 		log.Print("Internal server error: stored value is not correct")
// 		c.AbortWithStatus(http.StatusInternalServerError)
// 		return
// 	}
// 	var mappings []data.ItemPerList
// 	if err := c.BindJSON(&mappings); err != nil {
// 		log.Printf("Failed to decode items in list: %s", err)
// 		c.AbortWithStatus(http.StatusBadRequest)
// 		return
// 	}
// 	listId := 0
// 	// var dbMappings []data.ItemPerList
// 	for i, item := range mappings {
// 		// Check if the list is always the same
// 		if i == 0 {
// 			listId = int(item.ListId)
// 			shoppinglist, err := database.GetShoppingList(item.ListId)
// 			if err != nil {
// 				log.Printf("The list %d does not exist!", listId)
// 				c.AbortWithStatus(http.StatusBadRequest)
// 				return
// 			}
// 			if shoppinglist.CreatedBy != int64(userId) {
// 				log.Print("The user does not own this list!")
// 				c.AbortWithStatus(http.StatusBadRequest)
// 				return
// 			}
// 		}
// 		if item.ListId != int64(listId) {
// 			log.Print("Not all items are in the same list!")
// 			c.AbortWithStatus(http.StatusBadRequest)
// 			return
// 		}
// 		_, err := database.GetItem(item.ItemId)
// 		if err != nil {
// 			log.Printf("Item %d does not exist!", item.ItemId)
// 			c.AbortWithStatus(http.StatusBadRequest)
// 			return
// 		}
// 		_, err = database.InsertItemToList(item)
// 		if err != nil {
// 			log.Printf("Failed to insert mapping: %s", err)
// 			c.AbortWithStatus(http.StatusInternalServerError)
// 			return
// 		}
// 		// dbMappings = append(dbMappings, dbMapping)
// 	}
// 	// We don't need to update any mappings because the id itself is never used
// 	c.JSON(http.StatusCreated, gin.H{"status": "created"})
// }

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
	var shared data.ListShared
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
	if list.CreatedBy != int64(userId) {
		log.Printf("User ID (%d) does not match created ID (%d)", userId, list.CreatedBy)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	listShared, err := database.CreateSharedList(shared.ListId, shared.SharedWith)
	if err != nil {
		log.Printf("Failed to create sharing: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, listShared)
}

// TODO:
func unshareList() {
	log.Print("Not implemented!")

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
		authorized.POST("/update/userinfo", updateUserinfo)

		// Handling the lists itself
		authorized.GET("/lists/:userId", getShoppingListsForUser) // Includes OWN and SHARED lists
		authorized.GET("/list/:id", getShoppingList)

		// Includes both adding a new list and updating an existing list
		authorized.POST("/list", postShoppingList)

		// Handling the items per list
		// authorized.POST("/list/items", postItemsInList)

		// Handling sharing a list
		authorized.POST("/share/:id", shareList)

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
