package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strconv"

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
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	var recipeToCreate data.Recipe
	if err := json.Unmarshal([]byte(recipeInfo), &recipeToCreate); err != nil {
		log.Printf("Failed to parse recipe to create: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	log.Printf("Creating new recipe '%d' with name '%s'", recipeToCreate.RecipeId, recipeToCreate.Name)
	if err := IsUserAllowedToHandleData(recipeToCreate, userId, false); err != nil {
		log.Printf("Failed to create recipe '%d': %s", recipeToCreate.RecipeId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
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

func loadRecipeWithImages(recipeId int64, createdBy int64) (data.Recipe, [][]byte, []string, error) {
	log.Printf("Loading recipe %d from user %d", recipeId, createdBy)
	recipe, err := database.GetRecipe(int64(recipeId), createdBy)
	if err != nil {
		return data.Recipe{}, nil, []string{}, err
	}
	imageFilePaths, err := database.GetImageNamesForRecipe(recipeId, createdBy)
	if err != nil {
		return data.Recipe{}, nil, []string{}, err
	}
	imageData, err := database.GetImagesFromFilepaths("images/recipes", imageFilePaths)
	if err != nil {
		return data.Recipe{}, nil, []string{}, err
	}
	return recipe, imageData, imageFilePaths, nil
}

func sendResponseInMultiformData(writer gin.ResponseWriter, recipes []data.Recipe, images [][][]byte, imageFilePaths [][]string) error {
	writer.Header().Set("Content-Type", "multipart/mixed")
	writer.WriteHeader(http.StatusOK)
	mw := multipart.NewWriter(writer)
	for i, recipe := range recipes {
		// Layout is first the object content followed by binary data
		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", "application/json")
		recipeName := fmt.Sprintf("object-%d", i)
		contentDisposition := fmt.Sprintf("form-data; name=\"%s\"", recipeName)
		header.Add("Content-Disposition", contentDisposition)
		object, err := mw.CreatePart(header)
		if err != nil {
			log.Printf("Failed to create object for recipe %d from %s: %s", recipe.RecipeId, recipe.CreatedBy.Name, err)
			return err
		}
		rawRecipe, err := json.Marshal(recipe)
		if err != nil {
			log.Printf("Failed to serialize recipe %d from %s: %s", recipe.RecipeId, recipe.CreatedBy.Name, err)
			return err
		}
		written, err := object.Write(rawRecipe)
		if err != nil || written != len(rawRecipe) {
			log.Printf("Failed to write recipe %d from %s to response: %s", recipe.RecipeId, recipe.CreatedBy.Name, err)
			return err
		}

		recipeImageData := images[i]
		recipeImageFilePaths := imageFilePaths[i]
		for imageIndex, imageData := range recipeImageData {
			contentHeader := make(textproto.MIMEHeader)
			filename := recipeImageFilePaths[imageIndex]
			if filepath.Ext(filename) == "" {
				errorMessage := fmt.Sprintf("unset image file extension for image %d in recipe %d from %d", imageIndex, recipe.RecipeId, recipe.CreatedBy.ID)
				return errors.New(errorMessage)
			}
			contentHeader.Set("Content-Type", mime.TypeByExtension(filepath.Ext(filename)))
			contentHeader.Set("Content-Disposition", fmt.Sprintf("attachment; name=content, filename=\"%s\"", filename))
			content, err := mw.CreatePart(contentHeader)
			if err != nil {
				log.Printf("Failed to create image for recipe %d from %d: %s", recipe.RecipeId, recipe.CreatedBy.ID, err)
				return err
			}
			_, err = content.Write(imageData)
			if err != nil {
				log.Printf("Failed to write image for recipe %d from %d: %s", recipe.RecipeId, recipe.CreatedBy.ID, err)
				return err
			}
		}
	}
	err := mw.Close()
	if err != nil {
		log.Printf("Failed to close content writer: %s", err)
	}
	return nil
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
	createdBy := c.Query("createdBy")
	if createdBy != "" {
		log.Printf("CreatedBy is set, using query parameter userId")
		parsedCreatedBy, err := strconv.Atoi(createdBy)
		if err != nil {
			log.Printf("Failed to parse query parameter parameter %s: %s", "createdBy", err)
		}
		userId = int64(parsedCreatedBy)
	}
	recipe, imageData, imageFilePaths, err := loadRecipeWithImages(int64(recipeId), userId)
	if err != nil {
		log.Printf("Failed to load recipe %d: %s", recipeId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	wrappedImageData := make([][][]byte, 1)
	wrappedImageData[0] = imageData
	wrappedImagePaths := make([][]string, 1)
	wrappedImagePaths[0] = imageFilePaths
	err = sendResponseInMultiformData(c.Writer, []data.Recipe{recipe}, wrappedImageData, wrappedImagePaths)
	if err != nil {
		log.Printf("Failed to send response to recipe %d: %s", recipeId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Since we wrote directly into the stream, we are done here
}

func getOwnAndSharedRecipesWithImages(c *gin.Context) {
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User not authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	ownRecipeIds, err := database.GetRecipeForUserId(userId)
	if err != nil {
		log.Printf("Failed to get recipes for user %d: %s", userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	sharedWithRecipeIds, sharedWithRecipeCreatedBy, err := database.GetRecipeIdsSharedWithUserId(userId)
	if err != nil {
		log.Printf("Failed to get shared recipes for user %d: %s", userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	combinedRecipeIds := make([]int64, 0)
	combinedRecipeIds = append(combinedRecipeIds, ownRecipeIds...)
	combinedRecipeIds = append(combinedRecipeIds, sharedWithRecipeIds...)
	allRawRecipes := make([]data.Recipe, 0)
	allRawImages := make([][][]byte, 0)
	allRawImageFilePaths := make([][]string, 0)
	for index, recipeId := range combinedRecipeIds {
		recipeCreatedBy := userId
		if index >= len(ownRecipeIds) {
			recipeCreatedBy = sharedWithRecipeCreatedBy[index-len(ownRecipeIds)]
		}
		recipe, imageData, imageFilePaths, err := loadRecipeWithImages(recipeId, recipeCreatedBy)
		if err != nil {
			log.Printf("Failed to load recipe %d: %s", recipeId, err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		recipeCreator, err := database.GetUser(recipeCreatedBy)
		if err != nil {
			log.Printf("Failed to get recipe creator for recipe %d: %s", recipeId, err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		recipe.CreatedBy.Name = recipeCreator.Username
		allRawRecipes = append(allRawRecipes, recipe)
		allRawImages = append(allRawImages, imageData)
		allRawImageFilePaths = append(allRawImageFilePaths, imageFilePaths)
	}
	log.Printf("Got total of %d recipes for user %d", len(allRawRecipes), userId)
	err = sendResponseInMultiformData(c.Writer, allRawRecipes, allRawImages, allRawImageFilePaths)
	if err != nil {
		log.Printf("Failed to send response for user %d: %s", userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	log.Printf("Successfully send %d recipes for user %d", len(allRawRecipes), userId)
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
	recipeInfo := c.PostForm("object")
	if recipeInfo == "" {
		log.Printf("Wrong request format: No recipe info found!")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	var recipeToUpdate data.Recipe
	if err := json.Unmarshal([]byte(recipeInfo), &recipeToUpdate); err != nil {
		log.Printf("Failed to parse recipe to update: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if recipeId != int(recipeToUpdate.RecipeId) {
		log.Printf("RecipeId to update in body %d and header %d do not match", recipeId, recipeToUpdate.CreatedBy.ID)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err = IsUserAllowedToHandleData(recipeToUpdate, userId, true); err != nil {
		log.Printf("User is not allowed to update recipe: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := database.UpdateRecipe(recipeToUpdate); err != nil {
		log.Printf("Failed to update recipe: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	recipePk := data.RecipePK{
		RecipeId:  recipeToUpdate.RecipeId,
		CreatedBy: recipeToUpdate.CreatedBy.ID,
	}
	if err := database.UpdateAndReplaceImagesForRecipe(c, recipePk); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
}

func IsUserAllowedToHandleData(recipe data.Recipe, userId int64, update bool) error {
	if userId == recipe.CreatedBy.ID {
		return nil
	}
	if !update {
		log.Printf("UserId %d does not match recipe createdBy %d", userId, recipe.CreatedBy.ID)
		return errors.New("user creating recipe different from user in recipe")
	}
	if err := database.IsRecipeSharedWithUser(userId, recipe.RecipeId, recipe.CreatedBy.ID); err != nil {
		log.Printf("Recipe %d from %d to update is not shared with user %d", recipe.RecipeId, recipe.CreatedBy.ID, userId)
		return errors.New("recipe is not shared with user")
	}
	return nil
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
	recipeToDelete, err := database.GetRecipe(int64(recipeId), userId)
	if err != nil {
		log.Printf("Failed to get recipe %d: %s", recipeId, err)
		c.AbortWithStatus(http.StatusOK)
		return
	}
	if err := IsUserAllowedToHandleData(recipeToDelete, userId, false); err != nil {
		log.Printf("User is not allowed to delete recipe %d: %s", recipeId, err)
		c.AbortWithStatus(http.StatusBadRequest)
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

func isUserAllowedToUnshare(recipeId int64, createdBy int64, userId int64) error {
	if createdBy == userId {
		return nil
	}
	err := database.IsRecipeSharedWithUser(userId, recipeId, createdBy)
	return err
}

func deleteShareRecipe(c *gin.Context) {
	log.Print("Deleting recipe sharing")
	strRecipeId := c.Param("recipeId")
	convertedRecipeId, err := strconv.Atoi(strRecipeId)
	if err != nil {
		log.Printf("Failed to parse given list id: %s: %s", strRecipeId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	recipeId := int64(convertedRecipeId)
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	createdBy := userId
	strCreatedBy, createdByExists := c.GetQuery("createdBy")
	if createdByExists && strCreatedBy != "" {
		convertedCreatedBy, err := strconv.Atoi(strCreatedBy)
		if err != nil {
			log.Printf("Given query parameter createdBy %s in wrong format: %s", strCreatedBy, err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		createdBy = int64(convertedCreatedBy)
	}

	if err := isUserAllowedToUnshare(recipeId, createdBy, userId); err != nil {
		log.Printf("User %d is no allowed to handle recipe %d created by %d", userId, recipeId, createdBy)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	recipe, err := database.GetRecipe(recipeId, createdBy)
	if err != nil {
		log.Printf("Recipe to unshare does not exist")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	strSharedWithId, exists := c.GetQuery("sharedWith")
	if !exists {
		if recipe.CreatedBy.ID != userId {
			log.Printf("Query parameter sharedWith not found, and only creator can remove all sharings")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if err := database.DeleteAllSharingForRecipe(recipeId, userId); err != nil {
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
		if err := database.DeleteRecipeSharing(recipeId, createdBy, int64(sharedWithId)); err != nil {
			log.Printf("Failed to delete recipe sharing: %s", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}
	c.Status(http.StatusOK)
}
