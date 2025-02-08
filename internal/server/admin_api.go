package server

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
)

func getAllUsers(c *gin.Context) {
	users, err := database.GetAllUsers()
	if err != nil {
		log.Printf("Failed to get all users: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.IndentedJSON(http.StatusOK, users)
}

func getAllLists(c *gin.Context) {
	lists, err := database.GetAllRawShoppingLists()
	if err != nil {
		log.Printf("Failed to get all lists: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, lists)
}

func getAllRecipes(c *gin.Context) {
	recipes, err := database.GetAllRecipes()
	if err != nil {
		log.Printf("Failed to get all recipes: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, recipes)
}
