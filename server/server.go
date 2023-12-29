package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"shop.cloudsheeptech.com/database"
	"shop.cloudsheeptech.com/server/authentication"
	"shop.cloudsheeptech.com/server/configuration"
	"shop.cloudsheeptech.com/server/data"
)

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
		return
	}
	item, err := database.InsertItemStruct(item)
	if err != nil || item.ID < 0 {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusAccepted, item)
	log.Printf("Inserted item under ID %d", item.ID)
}

// TODO: Implement actual mapping of user id to list
func getShoppingListsForUser(c *gin.Context) {
	sUserId := c.Param("userId")
	id, err := strconv.Atoi(sUserId)
	if err != nil {
		log.Printf("Failed to parse given item id: %s", sUserId)
		log.Printf("Err: %s", err)
		return
	}
	listIds, err := database.GetShoppingListFromUserId(int64(id))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	raw, err := json.Marshal(listIds)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	// TODO: add also the different lists to the data that is sent back
	c.IndentedJSON(http.StatusOK, raw)
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
	raw, err := json.Marshal(mapping)
	if err != nil {
		log.Printf("Failed to convert raw data: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusOK, raw)
}

// ------------------------------------------------------------
// Debug functionality
// ------------------------------------------------------------

func returnUnauth(c *gin.Context) {
	item := data.Item{}
	c.IndentedJSON(http.StatusOK, item)
}

// ------------------------------------------------------------
// The main function
// ------------------------------------------------------------

func Start(cfg configuration.Config) error {
	gin.SetMode(gin.DebugMode)

	router := gin.Default()
	authentication.Setup(cfg)

	// ------------- Handling Account Creation and Login ---------------

	router.POST("/auth/create", authentication.CreateAccount)
	// JWT BASED AUTHENTICATION
	router.POST("/auth/login", authentication.Login)

	// ------------- Handling Routes v1 ---------------

	// Add authentication middleware to v1 router
	authorized := router.Group("/v1")
	authorized.Use(authentication.AuthenticationMiddleware(cfg))
	{
		authorized.GET("/items", getAllItems)
		authorized.GET("/items/:id", getItem)

		authorized.POST("/items", addItem)

		authorized.GET("/lists/:userId", getShoppingListsForUser)
		authorized.GET("/list/:id", getShoppingList)

		authorized.GET("/test/auth", returnUnauth)
	}

	router.GET("/test/unauth", returnUnauth)

	// -------------------------------------------

	address := cfg.ListenAddr + ":" + cfg.ListenPort
	// Only allow TLS
	router.RunTLS(address, cfg.TLSCertificate, cfg.TLSKeyfile)

	return nil
}
