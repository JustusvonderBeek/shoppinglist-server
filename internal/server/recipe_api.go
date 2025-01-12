package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
)

func createShareRecipe(c *gin.Context) {
	strRecipeId := c.Param("recipeId")
	recipeId, err := strconv.Atoi(strRecipeId)
	if err != nil {
		log.Printf("Failed to parse given list id: %s: %s", strRecipeId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// Ignore the 'id' field in the struct
	var shared data.RecipeShared
	if err = c.BindJSON(&shared); err != nil {
		log.Printf("Failed to bind given data: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	recipe, err := database.GetRecipe(int64(recipeId), userId)
	if err != nil {
		log.Printf("Recipe to share does not exist")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if recipe.CreatedBy.ID != userId {
		log.Printf("User %d is not allowed to share recipe %d created by %d", userId, recipeId, recipe.CreatedBy.ID)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if err := database.CreateRecipeSharing(int64(recipeId), userId, shared.SharedWith); err != nil {
		log.Printf("Failed to create recipe sharing: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusCreated)
}

func createRecipe(c *gin.Context) {
	log.Print("Creating new recipe")
	var recipeToCreate data.Recipe
	if err := c.BindJSON(&recipeToCreate); err != nil {
		log.Printf("Failed to parse recipe to create: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// The user can specify his own id, therefore we don't return anything
	if err := database.CreateRecipe(recipeToCreate); err != nil {
		log.Printf("Failed to create recipe: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusCreated)
}

func getRecipe(c *gin.Context) {
	log.Print("Reading recipe")
	strRecipeId := c.Param("recipeId")
	recipeId, err := strconv.Atoi(strRecipeId)
	if err != nil {
		log.Printf("Failed to parse recipeId parameter: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	recipe, err := database.GetRecipe(int64(recipeId), userId)
	if err != nil {
		log.Printf("Failed to read recipe %d from %d from database: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, recipe)
}

func updateRecipe(c *gin.Context) {
	log.Print("Updating recipe")
	strRecipeId := c.Param("recipeId")
	recipeId, err := strconv.Atoi(strRecipeId)
	if err != nil {
		log.Printf("Failed to parse recipeId parameter: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	var recipeToUpdate data.Recipe
	if err := c.BindJSON(&recipeToUpdate); err != nil {
		log.Printf("Failed to parse update recipe body: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// TODO: Include updating a shared recipe
	if recipeId != int(recipeToUpdate.RecipeId) || userId != recipeToUpdate.CreatedBy.ID {
		log.Printf("User %d is not allowed to update the recipe from %d", userId, recipeToUpdate.CreatedBy.ID)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if err := database.UpdateRecipe(recipeToUpdate); err != nil {
		log.Printf("Failed to update recipe: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
}

func deleteRecipe(c *gin.Context) {
	log.Print("Deleting recipe")
	strRecipeId := c.Param("recipeId")
	recipeId, err := strconv.Atoi(strRecipeId)
	if err != nil {
		log.Printf("Failed to parse recipeId parameter: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// TODO: Include deleting a shared recipe
	if err := database.DeleteRecipe(int64(recipeId), userId); err != nil {
		log.Printf("Failed to delete recipe %d from %d: %s", recipeId, userId, err)
		c.Copy().AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
}

func updateShareRecipe(c *gin.Context) {
	log.Print("Updating recipe")

}

func deleteShareRecipe(c *gin.Context) {
	log.Print("Deleting recipe")
}
