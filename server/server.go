package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"shop.cloudsheeptech.com/authentication"
	"shop.cloudsheeptech.com/configuration"
	"shop.cloudsheeptech.com/database"
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
	item, err := database.GetItem(id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.IndentedJSON(http.StatusOK, item)
	log.Print("Send item back to requester")
}

func addItem(c *gin.Context) {
	var item database.Item
	if err := c.BindJSON(&item); err != nil {
		log.Printf("Item is in incorrect format: %v", c.Request.Body)
		return
	}
	id, err := database.InsertItem(item)
	if err != nil || id < 0 {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusAccepted, id)
	log.Printf("Inserted item under ID %d", id)
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
	listIds, err := database.GetMappingWithUserId(id)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	raw, err := json.Marshal(listIds)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
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
	mapping, err := database.GetMappingWithListId(id)
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

var jwt_secret = []byte("password")

func authorize(c *gin.Context) {
	bearerToken := c.Request.Header.Get("Authorization")
	if bearerToken == "" {
		log.Print("No JWT token found")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	reqToken := strings.Split(bearerToken, " ")[1]
	claims := &authentication.Claims{}
	tkn, err := jwt.ParseWithClaims(reqToken, claims, func(token *jwt.Token) (interface{}, error) {
		return jwt_secret, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "bad request",
		})
		return
	}
	if !tkn.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": "resource data",
	})
}

func Start(cfg configuration.Config) error {
	gin.SetMode(gin.DebugMode)

	router := gin.Default()

	// User account handling and creation
	router.POST("/create/account", authentication.CreateAccount)
	// JWT BASED AUTHENTICATION
	router.POST("/login", authentication.PerformAuthentication)
	// router.Use(authentication.AuthenticationMiddleware())

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
	}

	router.GET("/resource", authorize)
	router.GET("/test", getAllItems)

	// -------------------------------------------

	address := cfg.ListenAddr + ":" + cfg.ListenPort
	// Only allow TLS
	router.RunTLS(address, cfg.TLSCertificate, cfg.TLSKeyfile)

	return nil
}
