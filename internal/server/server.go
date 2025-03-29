package server

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/authentication"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/configuration"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/middleware"
)

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
	if cfg.Production {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	authentication.Setup(cfg)
	router.Use(middleware.CorsMiddleware())

	// ------------- Handling Account Creation and Login ---------------

	// TODO: Outsource the handling of users into it's own Service
	// Independent of API version, therefore not in the auth bracket
	router.POST("/v1/users", CreateAccount)
	// JWT BASED AUTHENTICATION
	router.POST("/v1/users/login/:userId", authentication.Login)

	// ------------- Handling Routes v1 (API version 1) ---------------

	// Add authentication middleware to v1 router
	authorized := router.Group("/v1")
	authorized.Use(authentication.AuthMiddleware())
	{
		// The structure is similar to the order of operations: create, update, get, delete

		// Notice: The creation of a new user does not require an account and authorization
		// therefore, it is handled in the unauthorized part above
		authorized.PUT("/users/:userId", updateUserinfo)
		authorized.GET("/users/:userId", getUserInfos)
		authorized.DELETE("/users/:userId", DeleteAccount)

		authorized.GET("/users/name", getMatchingUsers) // Includes search query parameter

		authorized.POST("/lists", createShoppingList)
		authorized.PUT("/lists/:listId", updateShoppingList) // Includes createBy parameter
		authorized.GET("/lists/:listId", getShoppingList)    // Includes search query parameter
		authorized.GET("/lists", getAllShoppingListsForUser)
		authorized.DELETE("/lists/:listId", deleteShoppingList)

		authorized.POST("/share/:listId", shareShoppingList)
		authorized.PUT("/share/:listId", updateShareShoppingList)
		authorized.DELETE("/share/:listId", unshareShoppingList)

		authorized.POST("/recipe", createRecipe)
		authorized.GET("/recipe/:recipeId", getRecipe)
		authorized.PUT("/recipe/:recipeId", updateRecipe)
		authorized.DELETE("/recipe/:recipeId", deleteRecipe)

		authorized.POST("recipe/share/:recipeId", createShareRecipe)
		authorized.DELETE("recipe/share/:recipeId", deleteShareRecipe)

		// DEBUG Purpose: TODO: Disable when no longer testing
		authorized.GET("/test/auth", returnUnauth)
		authorized.POST("/test/auth", returnPostTest)
	}

	admin := router.Group("v1/admin")
	admin.Use(authentication.AdminAuthenticationMiddleware())
	{
		admin.GET("/users", getAllUsers)
		admin.GET("/lists", getAllLists)
		admin.GET("/recipes", getAllRecipes)
	}

	router.GET("/test/unauth", returnUnauth)

	return router
}

func Start(cfg configuration.Config) error {
	router := SetupRouter(cfg)

	address := cfg.ListenAddr + ":" + cfg.ListenPort
	// Only allow TLS
	var err error
	if !cfg.DisableTLS {
		err = router.RunTLS(address, cfg.TLSCertificate, cfg.TLSKeyfile)
	} else {
		log.Printf("Disabling TLS...")
		err = router.Run(address)
	}
	return err
}
