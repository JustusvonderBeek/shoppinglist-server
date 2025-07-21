package server

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"time"

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

func pingTest(c *gin.Context) {
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Print("Logged in user is zero")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	status := data.Ping{
		Status:      "success",
		CurrentTime: time.Now(),
	}
	c.JSON(http.StatusOK, status)
}

// ------------------------------------------------------------
// The main function
// ------------------------------------------------------------

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, activeConnections)
}

func SetupRouter(db *sql.DB, config configuration.Config) *gin.Engine {
	if config.Server.Production {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	auth := authentication.NewAuthenticationHandler(db, config)
	router.Use(middleware.CorsMiddleware())
	router.Use(prometheusMiddleware)

	// ------------- Handling Account Creation and Login ---------------

	// TODO: Outsource the handling of users into it's own Service
	// Independent of API version, therefore not in the auth bracket
	router.POST("/v1/users", CreateAccount)
	// Server BASED AUTHENTICATION
	router.POST("/v1/users/login/:userId", auth.Login)

	// ------------- Handling Routes v1 (API version 1) ---------------

	// Add authentication middleware to v1 router
	authorized := router.Group("/v1")
	authorized.Use(auth.AuthMiddleware())
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
		authorized.DELETE("/lists", deleteAllOwnShoppingLists)

		authorized.POST("/share/:listId", shareShoppingList)
		authorized.PUT("/share/:listId", updateShareShoppingList)
		authorized.DELETE("/share/:listId", unshareShoppingList)

		authorized.POST("/recipe", createRecipe)
		authorized.GET("/recipe/:recipeId", getRecipe)
		authorized.GET("/recipe/:recipeId/images", getRecipeImages)
		authorized.GET("recipe/:recipeId/full", getRecipeWithImages)
		authorized.GET("/recipe/full", getOwnAndSharedRecipesWithImages)
		authorized.PUT("/recipe/:recipeId", updateRecipe)
		authorized.DELETE("/recipe/:recipeId", deleteRecipe)

		authorized.POST("recipe/share/:recipeId", createShareRecipe)
		authorized.DELETE("recipe/share/:recipeId", deleteShareRecipe)

		// DEBUG Purpose: TODO: Disable when no longer testing
		authorized.GET("/ping", pingTest)
		authorized.GET("/test/auth", returnUnauth)
		authorized.POST("/test/auth", returnPostTest)
	}

	admin := router.Group("v1/admin")
	admin.Use(auth.AdminAuthenticationMiddleware())
	{
		admin.GET("/users", getAllUsers)
		admin.GET("/lists", getAllLists)
		admin.GET("/recipes", getAllRecipes)
	}

	metrics := router.Group("/v1")
	metrics.Use(auth.AdminAuthWithoutUserMiddleware())
	{
		// Prometheus metrics endpoint, secured by API Key
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	router.GET("/test/unauth", returnUnauth)

	return router
}

func Start(db *sql.DB, config configuration.Config) error {
	router := SetupRouter(db, config)

	serverConfig := config.Server
	tlsConfig := config.TLS

	address := serverConfig.ListenAddr + ":" + serverConfig.ListenPort
	// Only allow TLS
	var err error
	if !tlsConfig.DisableTLS {
		log.Printf("Listening on %s with TLS enabled...", address)
		err = router.RunTLS(address, tlsConfig.CertificateFile, tlsConfig.KeyFile)
	} else {
		log.Printf("Listening on %s without TLS...", address)
		err = router.Run(address)
	}
	return err
}
