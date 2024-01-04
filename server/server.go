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
// Handling of items
// ------------------------------------------------------------

func getAllItems(c *gin.Context) {
	log.Printf("Trying to access all items")
	items, err := database.GetAllItems()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusOK, items)
	log.Printf("Send %d items back to requester", len(items))
}

func getItem(c *gin.Context) {
	sId := c.Param("id")
	id, err := strconv.Atoi(sId)
	if err != nil {
		log.Printf("Failed to parse given item id: %s", sId)
		log.Printf("Err: %s", err)
		return
	}
	log.Printf("Trying to access item: %d", id)
	item, err := database.GetItem(int64(id))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.IndentedJSON(http.StatusOK, item)
	log.Print("Send item back to requester")
}

func addItem(c *gin.Context) {
	var item data.Item
	if err := c.BindJSON(&item); err != nil {
		log.Printf("Item is in incorrect format: %v", c.Request.Body)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	item, err := database.InsertItemStruct(item)
	if err != nil || item.ID < 0 {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusCreated, item)
	log.Printf("Inserted item under ID %d", item.ID)
}

func addMultipleItems(c *gin.Context) {
	var items []data.Item
	if err := c.BindJSON(&items); err != nil {
		log.Printf("Items not in correct format: %v", c.Request.Body)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	var insertedItems []data.Item
	for _, item := range items {
		// TODO: Implement check that no item with the same name is inserted twice
		insertedItem, err := database.InsertItem(item.Name, item.Icon)
		if err != nil {
			log.Printf("Failed to insert item: %s", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		insertedItems = append(insertedItems, insertedItem)
	}
	// This answer is relevant for the client regarding the Item ID
	c.JSON(http.StatusCreated, insertedItems)
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
	mapping, err := database.GetShoppingList(int64(id))
	if err != nil {
		log.Printf("Failed to get mapping for id %d: %s", id, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// TODO: Transform this mapping to a concrete item list
	// raw, err := json.Marshal(mapping)
	// if err != nil {
	// 	log.Printf("Failed to convert raw data: %s", err)
	// 	c.AbortWithStatus(http.StatusInternalServerError)
	// 	return
	// }
	c.IndentedJSON(http.StatusOK, mapping)
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
	shoplist, err := database.CreateShoppingList(list.Name, list.CreatedBy, list.LastEdited)
	if err != nil {
		log.Printf("Failed to insert list into server: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// raw, err := json.Marshal(shoplist)
	// if err != nil {
	// 	log.Printf("Failed to convert list to JSON! %s", err)
	// 	c.AbortWithStatus(http.StatusInternalServerError)
	// 	return
	// }
	c.IndentedJSON(http.StatusCreated, shoplist)
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
	var shared data.ListShared
	if err = c.BindJSON(&shared); err != nil {
		log.Printf("Failed to bind given data: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Check if the user owns this list?
	list, err := database.GetShoppingList(shared.ListId)
	if err != nil {
		log.Printf("Failed to retrieve list: %s", err)
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
	if list.ID != int64(id) {
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
	c.IndentedJSON(http.StatusOK, gin.H{"status": "testing content"})
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
		// Handling the lists itself
		authorized.GET("/lists/:userId", getShoppingListsForUser) // Includes OWN and SHARED lists
		// authorized.GET("/list/:id", getShoppingList)

		authorized.POST("/list", postShoppingList)

		// Handling the items
		authorized.GET("/items", getAllItems)
		// authorized.GET("/items/:id", getItem)

		authorized.POST("/item", addItem)
		authorized.POST("/items", addMultipleItems)

		// Handling sharing a list
		authorized.POST("/share/:id", shareList)

		// DEBUG Purpose: TODO: Disable when no longer testing
		authorized.GET("/test/auth", returnUnauth)
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
