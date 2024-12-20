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
// Handling of user information
// ------------------------------------------------------------

func updateUserinfo(c *gin.Context) {
	var user data.User
	err := c.BindJSON(&user)
	if err != nil {
		log.Printf("Failed to parse updated user information: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Check if the user we want to modify is in fact the user that called our service
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User not authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if userId != user.OnlineID {
		log.Printf("User %d cannot modify user %d", userId, user.OnlineID)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// User already found in our database. Simply update this stuff
	user, err = database.ModifyUserAccountName(user.OnlineID, user.Username)
	if err != nil {
		log.Printf("User %d to update not found: %s", user.OnlineID, err)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, user)
}

func getUserInfos(c *gin.Context) {
	sUserId := c.Param("userId")
	queriedUserId, err := strconv.Atoi(sUserId)
	if err != nil {
		log.Printf("Failed to parse given item id: %s", sUserId)
		log.Printf("Err: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Make sure that the format of the user only includes name and other
	// non critical information, especially passwords
	user, err := database.GetUserInWireFormat(int64(queriedUserId))
	if err != nil {
		log.Printf("Queried user %d does not exist", queriedUserId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, user)
}

func getMatchingUsers(c *gin.Context) {
	// Expecting the searched username in the URL as query parameter
	// like: users/name?username=xxx
	queryUsername := c.Query("username")
	if queryUsername == "" {
		log.Printf("Username query not found or empty!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	users, err := database.GetUserFromMatchingUsername(queryUsername)
	if err != nil {
		log.Printf("Failed to retrieve matching users: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// Remove the user itself
	finalUsers := make([]data.ListCreator, 0)
	for _, user := range users {
		if user.OnlineID != int64(userId) {
			listCreator := data.ListCreator{
				ID:   user.OnlineID,
				Name: user.Username,
			}
			finalUsers = append(finalUsers, listCreator)
		}
	}
	if len(finalUsers) == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	log.Printf("Found %d matching users", len(users))
	c.JSON(http.StatusOK, finalUsers)
}

// ------------------------------------------------------------
// Handling of lists
// ------------------------------------------------------------

func postShoppingList(c *gin.Context) {
	var list data.List
	err := c.BindJSON(&list)
	if err != nil {
		log.Printf("Failed to convert given data to shopping list: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// Check if the requesting user is the owner or the list is shared
	if userId != list.CreatedBy.ID || list.CreatedBy.ID == 0 {
		log.Printf("The logged in user %d and the createdBy %d are not equal", userId, list.CreatedBy.ID)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	err = database.CreateShoppingList(list)
	if err != nil {
		log.Printf("Failed to create list: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// No more information is gained through an answer because the client
	// dictates the ID and the server stores the info combined with the userId
	c.Status(http.StatusCreated)
}

func putShoppingList(c *gin.Context) {
	// Check the contained listId and createdBy
	strListId := c.Param("listId")
	if strListId == "" {
		log.Printf("required listId not found or empty")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	listId, err := strconv.Atoi(strListId)
	if err != nil {
		log.Printf("failed to convert given listId to integer: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Check if the list was created by the requesting user or might be shared
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	createdBy := int(userId)
	strCreatedBy := c.Query("createdBy")
	if strCreatedBy != "" {
		// The list was not created by the requesting user!
		// Check if the list was shared with this user, otherwise that
		// would be an error
		createdBy, err = strconv.Atoi(strCreatedBy)
		if err != nil {
			log.Printf("given createdBy parameter is not an integer: %s", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if err = database.IsListSharedWithUser(int64(listId), int64(createdBy), int64(userId)); err != nil {
			log.Printf("list %d is not shared with user %d", listId, userId)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
	// Now, either the requesting user created the list or the list was shared with
	// the user that wants to update it
	var list data.List
	err = c.BindJSON(&list)
	if err != nil {
		log.Printf("Failed to convert given data to shopping list: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err = database.CreateOrUpdateShoppingList(list); err != nil {
		log.Printf("failed to update listId %d from user %d", listId, userId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
}

func getShoppingList(c *gin.Context) {
	strListId := c.Param("listId")
	if strListId == "" {
		log.Printf("listId parameter not found or empty")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	listId, err := strconv.Atoi(strListId)
	if err != nil {
		log.Printf("Failed to parse given listId: %s", strListId)
		log.Printf("Err: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	createdBy := int(userId)
	// Check if the list was created by the user itself or by another user
	strCreatedBy := c.Query("createdBy")
	if strCreatedBy != "" {
		createdBy, err = strconv.Atoi(strCreatedBy)
		if err != nil {
			log.Printf("given createdBy query parameter is no integer")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}
	listIsFromCreator := createdBy == int(userId)
	if !listIsFromCreator {
		// Check if the user actually has access to this list
		if err := database.IsListSharedWithUser(int64(listId), int64(createdBy), int64(userId)); err != nil {
			log.Printf("User %d is not owner of list %d but list is not shared", userId, listId)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
	list, err := database.GetShoppingList(int64(listId), int64(createdBy))
	if err != nil {
		log.Printf("Failed to get mapping for id %d: %s", listId, err)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	itemsInList, err := database.GetItemsInList(list.ListId, int64(createdBy))
	if err != nil {
		log.Printf("Failed to get item in list: %s", err)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	for _, item := range itemsInList {
		dbItem, err := database.GetItem(item.ItemId)
		if err != nil {
			log.Printf("Failed to find item '%d' in database", item.ItemId)
			continue
		}
		listItem := data.ItemWire{
			Name:     dbItem.Name,
			Icon:     dbItem.Icon,
			Quantity: item.Quantity,
			Checked:  item.Checked,
			AddedBy:  item.AddedBy,
		}
		list.Items = append(list.Items, listItem)
	}
	c.JSON(http.StatusOK, list)
}

func getAllShoppingListsForUser(c *gin.Context) {
	// User MUST be authenticated so it does exist and is allowed to make the request
	// Check for the lists of the user itself first
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User not authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	ownLists, err := database.GetShoppingListsFromUserId(int64(userId))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	log.Printf("Got %d lists for user", len(ownLists))

	// Asking the database for all the lists that are shared with the current user
	sharedListIds, err := database.GetSharedListForUserId(int64(userId))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	log.Printf("Got %d shared lists for user", len(sharedListIds))
	// Get full information for the shared lists
	sharedLists, err := database.GetShoppingListsFromSharedListIds(sharedListIds)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	ownLists = append(ownLists, sharedLists...)
	// Asking DB to get the items in this list
	for i, list := range ownLists {
		itemsPerList, err := database.GetItemsInList(list.ListId, list.CreatedBy.ID)
		if err != nil {
			log.Printf("Failed to get items for list %d: %s", list.ListId, err)
			ownLists[i].Items = []data.ItemWire{}
			continue
		}
		if len(itemsPerList) == 0 {
			ownLists[i].Items = []data.ItemWire{}
			continue
		}
		// Unpack and convert into wire format
		for _, item := range itemsPerList {
			dbItem, err := database.GetItem(item.ItemId)
			if err != nil {
				log.Printf("Failed to get information for item %d in list", i)
				continue
			}
			wireItem := data.ItemWire{
				Name:     dbItem.Name,
				Icon:     dbItem.Icon,
				Quantity: item.Quantity,
				Checked:  item.Checked,
				AddedBy:  item.AddedBy,
			}
			ownLists[i].Items = append(ownLists[i].Items, wireItem)
		}
	}
	c.JSON(http.StatusOK, ownLists)
}

func removeShoppingList(c *gin.Context) {
	strListId := c.Param("listId")
	if strListId == "" {
		log.Printf("Expected listId parameter but did not get anything")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	listId, err := strconv.Atoi(strListId)
	if err != nil {
		log.Printf("Failed to parse given listId: %s", strListId)
		log.Printf("Err: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// User can only delete the own lists, therefore check only if the list
	// is owned by the user
	list, err := database.GetShoppingList(int64(listId), int64(userId))
	if err != nil {
		log.Printf("Failed to get mapping for listId %d: %s", listId, err)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	// Can this really happen? What has to go wrong for the method above to return
	// a list with a different createdBy ?
	if list.CreatedBy.ID != userId {
		log.Printf("Cannot delete list: User %d did not create list %d", userId, list.ListId)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if err := database.DeleteShoppingList(int64(listId), int64(userId)); err != nil {
		log.Printf("Failed to delete list %d", listId)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	log.Printf("Delete list %d", listId)
	c.Status(http.StatusOK)
}

// ------------------------------------------------------------
// Handling of sharing
// ------------------------------------------------------------

func shareList(c *gin.Context) {
	strListId := c.Param("listId")
	if strListId == "" {
		log.Printf("listId parameter not found or empty")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	listId, err := strconv.Atoi(strListId)
	if err != nil {
		log.Printf("Failed to parse given list id: %s: %s", strListId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	var shared data.ListSharedWire
	if err = c.BindJSON(&shared); err != nil {
		log.Printf("Failed to bind given data: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Abort if the user does not own the list
	list, err := database.GetShoppingList(int64(listId), int64(userId))
	if err != nil {
		log.Printf("listId %d for given user %d not found: %s", listId, userId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if list.ListId != int64(listId) {
		log.Printf("IDs do not match!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// Should never happen, but who knows
	if list.CreatedBy.ID != int64(userId) {
		log.Printf("listId %d was not createdBy %d", listId, userId)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	var listShared data.ListShared
	for _, sharedWith := range shared.SharedWith {
		listShared, err = database.CreateOrUpdateSharedList(int64(listId), userId, sharedWith)
		if err != nil {
			log.Printf("Failed to create sharing: %s", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}
	c.JSON(http.StatusCreated, listShared)
}

func unshareList(c *gin.Context) {
	strListId := c.Param("listId")
	listId, err := strconv.Atoi(strListId)
	if err != nil {
		log.Printf("Failed to parse given list id: %s: %s", strListId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// Check if the user owns the list that should be unshared
	list, err := database.GetShoppingList(int64(listId), int64(userId))
	if err != nil {
		log.Printf("listId %d for given user %d not found: %s", listId, userId, err)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	// Delete requests cannot have a body, therefore simply delete all sharing
	// if only specific should be deleted, the PUT method can be used

	// var listUnshare data.ListSharedWire
	// if err := c.BindJSON(&listUnshare); err != nil {
	// 	log.Print("Failed to deserialize share object")
	// 	c.AbortWithStatus(http.StatusBadRequest)
	// 	return
	// }
	// should not happen, unless my implementation above is bogus, so could be :)
	if list.CreatedBy.ID != int64(userId) {
		log.Printf("User ID (%d) does not match created ID (%d)", userId, list.CreatedBy.ID)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if err = database.DeleteSharingOfList(int64(listId), userId); err != nil {
		log.Printf("failed to delete sharing of list %d for user %d", listId, userId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusOK)
}

func updateSharing(c *gin.Context) {
	strListId := c.Param("listId")
	listId, err := strconv.Atoi(strListId)
	if err != nil {
		log.Printf("Failed to parse given list id: %s: %s", strListId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId == 0 {
		log.Printf("User is not correctly authenticated")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// Check if the user owns the list that should be unshared
	list, err := database.GetShoppingList(int64(listId), int64(userId))
	if err != nil {
		log.Printf("listId %d for given user %d not found: %s", listId, userId, err)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	var updatedListShare data.ListSharedWire
	if err := c.BindJSON(&updatedListShare); err != nil {
		log.Print("Failed to deserialize share object")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// should not happen, unless my implementation above is bogus, so could be :)
	if list.CreatedBy.ID != int64(userId) {
		log.Printf("User ID (%d) does not match created ID (%d)", userId, list.CreatedBy.ID)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if err = database.DeleteSharingOfList(int64(listId), userId); err != nil {
		log.Printf("failed to delete sharing of list %d for user %d", listId, userId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	for _, shareWithId := range updatedListShare.SharedWith {
		if _, err := database.CreateOrUpdateSharedList(int64(listId), userId, shareWithId); err != nil {
			log.Printf("Failed to create sharing %s", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}
	c.Status(http.StatusOK)
}

// ------------------------------------------------------------
// Handling of recipes
// ------------------------------------------------------------

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
	if recipeId != int(recipeToUpdate.RecipeId) || userId != recipeToUpdate.CreatedBy {
		log.Printf("User is not allowed to update the recipe")
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

// ------------------------------------------------------------
// Handling of recipe sharing
// ------------------------------------------------------------

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
	if recipe.CreatedBy != userId {
		log.Printf("User %d is not allowed to share recipe %d created by %d", userId, recipeId, recipe.CreatedBy)
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

func updateShareRecipe(c *gin.Context) {
	log.Print("Updating recipe")

}

func deleteShareRecipe(c *gin.Context) {
	log.Print("Deleting recipe")
}

// ------------------------------------------------------------
// Admin Functionality
// ------------------------------------------------------------

func getAllUsers(c *gin.Context) {
	users, err := database.GetAllUsers()
	if err != nil {
		log.Printf("Failed to get all users: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.IndentedJSON(http.StatusOK, users)
}

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
	gin.SetMode(gin.DebugMode)

	router := gin.Default()
	authentication.Setup(cfg)

	// ------------- Handling Account Creation and Login ---------------

	// Independent of API version, therefore not in the auth bracket
	router.POST("/v1/users", authentication.CreateAccount)
	// JWT BASED AUTHENTICATION
	router.POST("/v1/users/:userId/login", authentication.Login)

	// ------------- Handling Routes v1 (API version 1) ---------------

	// Add authentication middleware to v1 router
	authorized := router.Group("/v1")
	authorized.Use(authentication.AuthenticationMiddleware())
	{
		// The structure is similar to the order of operations: create, update, get, delete

		// Notice: The creation of a new user does not require an account and authorization
		// therefore, it is handled in the unauthorized part above
		authorized.PUT("/users/:userId", updateUserinfo)
		authorized.GET("/users/:userId", getUserInfos)
		authorized.DELETE("/users/:userId", authentication.DeleteAccount)

		authorized.GET("/users/name", getMatchingUsers) // Includes search query parameter

		authorized.POST("/lists", postShoppingList)
		authorized.PUT("/lists/:listId", putShoppingList) // Includes createBy parameter
		authorized.GET("/lists/:listId", getShoppingList) // Includes search query parameter
		authorized.GET("/lists", getAllShoppingListsForUser)
		authorized.DELETE("/lists/:listId", removeShoppingList)

		authorized.POST("/share/:listId", shareList)
		authorized.PUT("/share/:listId", updateSharing)
		authorized.DELETE("/share/:listId", unshareList)

		authorized.POST("/recipe", createRecipe)
		authorized.GET("/recipe/:recipeId", getRecipe)
		authorized.PUT("/recipe/:recipeId", updateRecipe)
		authorized.DELETE("/recipe/:recipeId", deleteRecipe)

		authorized.POST("recipe/share/:recipeId", createShareRecipe)
		authorized.PUT("recipe/share/:recipeId", updateShareRecipe)
		authorized.DELETE("recipe/share/:recipeId", deleteShareRecipe)

		// DEBUG Purpose: TODO: Disable when no longer testing
		authorized.GET("/test/auth", returnUnauth)
		authorized.POST("/test/auth", returnPostTest)
	}

	authorized.Use(authentication.AdminAuthenticationMiddleware())
	{
	}

	router.GET("/admin/users", getAllUsers)
	router.GET("/test/unauth", returnUnauth)

	return router
}

func Start(cfg configuration.Config) error {
	router := SetupRouter(cfg)

	// -------------------------------------------

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
