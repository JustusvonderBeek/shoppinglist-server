package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
)

func createShoppingList(c *gin.Context) {
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

func updateShoppingList(c *gin.Context) {
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

func deleteShoppingList(c *gin.Context) {
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

func shareShoppingList(c *gin.Context) {
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

func unshareShoppingList(c *gin.Context) {
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

func updateShareShoppingList(c *gin.Context) {
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
