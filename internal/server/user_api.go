package server

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
)

func CreateAccount(c *gin.Context) {
	// Extracting username and password from request
	var user data.User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		log.Printf("Failed decode given user: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	createdUser, err := validateUserAndCreateAccount(user, c.Request.Header.Get("x-api-key"))
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusCreated, createdUser)
}

func validateUserAndCreateAccount(user data.User, apiKey string) (data.User, error) {
	if user.OnlineID != 0 {
		return data.User{}, errors.New("user id already set")
	}
	if user.Username == "" || user.Password == "" {
		return data.User{}, errors.New("invalid username or password")
	}
	//if user.Username == "admin" {
	//	keyValid, err := authentication.ApiKeyValid(apiKey)
	//	if apiKey == "" || err != nil || keyValid.ValidUntil.Before(time.Now()) {
	//		log.Printf("api key not valid %s - %s", err, apiKey)
	//		return data.User{}, errors.New("invalid api key")
	//	}
	//}
	loginUser, err := database.CreateUserAccountInDatabase(user.Username, user.Password)
	if err != nil {
		return data.User{}, err
	}
	// Don't include hashed password information in answer
	loginUser.Password = "accepted"
	return loginUser, nil
}

func DeleteAccount(c *gin.Context) {
	sId := c.Param("userId")
	id, err := strconv.Atoi(sId)
	if err != nil {
		log.Printf("Failed to parse given userId: %s: %s", sId, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	userId := c.GetInt64("userId")
	if userId != int64(id) {
		log.Printf("Authenticated user is not user that should be deleted!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := database.DeleteUserAccount(int64(userId)); err != nil {
		log.Printf("Failed to delete user account")
		c.AbortWithStatus(http.StatusGone)
		return
	}
	c.Status(http.StatusOK)
}

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
	user, err := database.GetUser(int64(queriedUserId))
	if err != nil {
		log.Printf("Queried user %d does not exist", queriedUserId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, user.ToWireFormat())
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
