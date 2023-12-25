package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
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

var tokens []string
var jwt_secret = []byte("password")

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func generateJWT() (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Username: "username",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(jwt_secret)

}

func Start(cfg configuration.Config) error {
	gin.SetMode(gin.DebugMode)

	router := gin.Default()
	router.GET("/items", getAllItems)
	router.GET("/items/:id", getItem)

	router.POST("/items", addItem)

	router.GET("/lists/:userId", getShoppingListsForUser)
	router.GET("/list/:id", getShoppingList)

	// Testing the authentication
	// NO AUTHENTICATION
	// router.GET("/resource", func(c *gin.Context) {
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"data": "resource data",
	// 	})
	// })
	// BASIC AUTHENTICATION
	// router.GET("/resource", gin.BasicAuth(gin.Accounts{
	// 	"admin": "secret",
	// }), func(c *gin.Context) {

	// 	c.JSON(http.StatusOK, gin.H{
	// 		"data": "resource data",
	// 	})
	// })
	// TOKEN BASED AUTHENTICATION
	// router.POST("/login", gin.BasicAuth(gin.Accounts{
	// 	"admin": "secret",
	// }), func(c *gin.Context) {
	// 	token := rand.Uint64()
	// 	stringToken := fmt.Sprintf("%x", token)
	// 	tokens = append(tokens, stringToken)

	// 	c.JSON(http.StatusOK, gin.H{
	// 		"token": stringToken,
	// 	})
	// })
	// router.GET("/resource", func(c *gin.Context) {
	// 	bearerToken := c.Request.Header.Get("Authorization")
	// 	log.Printf("Trying to log in with token %s", bearerToken)
	// 	reqToken := strings.Split(bearerToken, " ")[1]
	// 	for _, token := range tokens {
	// 		if token == reqToken {
	// 			c.JSON(http.StatusOK, gin.H{
	// 				"data": "resource data",
	// 			})
	// 			return
	// 		}
	// 	}
	// 	c.JSON(http.StatusUnauthorized, gin.H{
	// 		"message": "unauthorized",
	// 	})
	// })
	// JWT BASED AUTHENTICATION
	router.POST("/login", gin.BasicAuth(gin.Accounts{
		"admin": "secret",
	}), func(c *gin.Context) {
		token, _ := generateJWT()
		tokens = append(tokens, token)

		c.JSON(http.StatusOK, gin.H{
			"token": token,
		})
	})
	router.GET("/resource", func(c *gin.Context) {
		bearerToken := c.Request.Header.Get("Authorization")
		reqToken := strings.Split(bearerToken, " ")[1]
		claims := &Claims{}
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
	})

	address := cfg.ListenAddr + ":" + cfg.ListenPort
	// router.Run(address)
	router.RunTLS(address, cfg.TLSCertificate, cfg.TLSKeyfile)

	return nil
}
