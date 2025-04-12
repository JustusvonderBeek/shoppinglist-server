package server

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
)

func createRecipe(c *gin.Context) {
	_, err := c.MultipartForm()
	if err != nil {
		log.Printf("Wrong request format: No multipart form data found!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	recipeInfo := c.PostForm("object")
	if recipeInfo == "" {
		log.Printf("Wrong request format: No recipe info found!")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	var recipeToCreate data.Recipe
	if err := json.Unmarshal([]byte(recipeInfo), &recipeToCreate); err != nil {
		log.Printf("Failed to parse recipe to create: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	log.Printf("Creating new recipe '%d' with name '%s'", recipeToCreate.RecipeId, recipeToCreate.Name)
	// The user can specify his own id, therefore we don't return anything
	if err := database.CreateRecipe(recipeToCreate); err != nil {
		log.Printf("Failed to create recipe: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	recipePk := data.RecipePK{
		RecipeId:  recipeToCreate.RecipeId,
		CreatedBy: recipeToCreate.CreatedBy.ID,
	}
	if err := database.StoreImagesForRecipe(c, recipePk); err != nil {
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

func getRecipeImages(c *gin.Context) {
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
	imageFilepaths, err := database.GetImageNamesForRecipe(int64(recipeId), userId)
	if err != nil {
		log.Printf("Failed to load images for recipe %d from %d: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	imageData, err := database.GetImagesFromFilepaths("recipe", imageFilepaths)
	if err != nil {
		log.Printf("Failed to load images for recipe %d from %d: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Flatten the given image data
	flattendedImages := make([]byte, 0)
	for _, image := range imageData {
		flattendedImages = append(flattendedImages, image...)
	}
	c.Data(http.StatusOK, "application/media", flattendedImages)
}

func getRecipeWithImages(c *gin.Context) {
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
	imageFilePaths, err := database.GetImageNamesForRecipe(int64(recipe.RecipeId), userId)
	if err != nil {
		log.Printf("Failed to load images for recipe %d from %d: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	imageData, err := database.GetImagesFromFilepaths("images/recipes", imageFilePaths)
	if err != nil {
		log.Printf("Failed to load images for recipe %d from %d: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Since the library doesn't support multipart responses, we need to write or own handler
	randomBoundary := "12345555111123123123132asdkjashfdlkasf"
	c.Writer.Header().Set("Content-Type", "multipart/mixed; boundary"+randomBoundary)
	c.Writer.WriteHeader(http.StatusOK)

	mw := multipart.NewWriter(c.Writer)
	err = mw.SetBoundary(randomBoundary)
	if err != nil {
		log.Printf("Failed to set boundary for recipe %d from %d: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "application/json")
	header.Add("Content-Disposition", "form-data; name=\"object\"")
	object, err := mw.CreatePart(header)
	if err != nil {
		log.Printf("Failed to create object for recipe %d from %d: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	rawRecipe, err := json.Marshal(recipe)
	if err != nil {
		log.Printf("Failed to serialize recipe for recipe %d from %d: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	written, err := object.Write(rawRecipe)
	if err != nil || written != len(rawRecipe) {
		log.Printf("Failed to write object for recipe %d from %d: %s", recipeId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	for i, elem := range imageData {
		contentHeader := make(textproto.MIMEHeader)
		filename := imageFilePaths[i]
		contentHeader.Set("Content-Type", mime.TypeByExtension(filepath.Ext(filename)))
		contentHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		content, err := mw.CreatePart(contentHeader)
		if err != nil {
			log.Printf("Failed to create image for recipe %d from %d: %s", recipeId, userId, err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		_, err = content.Write(elem)
		if err != nil {
			log.Printf("Failed to write image for recipe %d from %d: %s", recipeId, userId, err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}

	err = mw.Close()
	if err != nil {
		log.Printf("Failed to close content writer for recipe %d from %d: %s", recipeId, userId, err)
	}
	// Since we wrote directly into the stream, we are done here
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
	filepaths, err := database.GetImageNamesForRecipe(int64(recipeId), userId)
	if err != nil {
		log.Printf("Failed to load image names for recipe: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	log.Printf("Deleting %d image(s) stored for recipe", len(filepaths))
	err = database.DeleteImagesFromFilepaths("recipes", filepaths)
	if err != nil {
		log.Printf("Failed to delete image names for recipe: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := database.DeleteRecipe(int64(recipeId), userId); err != nil {
		log.Printf("Failed to delete recipe %d from %d: %s", recipeId, userId, err)
		c.Copy().AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
}

// ----------------------------------------
// Recipe sharing
// ----------------------------------------

func createShareRecipe(c *gin.Context) {
	log.Print("Creating recipe sharing")
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
	recipe, err := database.GetRecipe(int64(recipeId), userId)
	if err != nil {
		log.Printf("Recipe to share does not exist")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if recipe.CreatedBy.ID != userId {
		log.Printf("User %d is not allowed to share recipe %d created by %d", userId, recipeId, recipe.CreatedBy.ID)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	strSharedWithId, exists := c.GetQuery("sharedWith")
	if !exists {
		log.Printf("Query parameter sharedWith not found")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	sharedWith, err := strconv.Atoi(strSharedWithId)
	if err != nil {
		log.Printf("Failed to parse sharedWith parameter: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	_, err = database.GetUser(int64(sharedWith))
	if err != nil {
		log.Printf("User %d to share with does not exist", sharedWith)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := database.CreateRecipeSharing(int64(recipeId), userId, int64(sharedWith)); err != nil {
		log.Printf("Failed to create recipe sharing: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusCreated)
}

func deleteShareRecipe(c *gin.Context) {
	log.Print("Deleting recipe sharing")
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
	recipe, err := database.GetRecipe(int64(recipeId), userId)
	if err != nil {
		log.Printf("Recipe to share does not exist")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if recipe.CreatedBy.ID != userId {
		log.Printf("User %d is not allowed to delete sharing of recipe %d created by %d", userId, recipeId, recipe.CreatedBy.ID)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	strSharedWithId, exists := c.GetQuery("sharedWith")
	if !exists {
		if recipe.CreatedBy.ID != userId {
			log.Printf("Query parameter sharedWith not found, and only creator can remove all sharings")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if err := database.DeleteAllSharingForRecipe(int64(recipeId), userId); err != nil {
			log.Printf("Failed to delete all recipe sharings: %s", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	} else {
		sharedWithId, err := strconv.Atoi(strSharedWithId)
		if err != nil {
			log.Printf("Failed to parse sharedWith parameter: %s", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if err := database.DeleteRecipeSharing(int64(recipeId), userId, int64(sharedWithId)); err != nil {
			log.Printf("Failed to delete recipe sharing: %s", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}
	c.Status(http.StatusOK)
}
