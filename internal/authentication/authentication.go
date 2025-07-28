package authentication

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/configuration"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/data"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
)

type AuthenticationHandler struct {
	config       configuration.Config
	tokenHandler *TokenHandler
	db           *sql.DB
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

// ------------------------------------------------------------
// Setup and configuration
// ------------------------------------------------------------

func NewAuthenticationHandler(db *sql.DB, config configuration.Config) *AuthenticationHandler {
	// TODO: Move into database as well
	//err := SetupWhitelistedIPs()
	//if err != nil {
	//	log.Fatalf("Failed to setup whitelisted IPs: %s", err)
	//}
	tokenHandler := NewTokenHandler(db, config.JWT)
	return &AuthenticationHandler{
		config:       config,
		tokenHandler: tokenHandler,
		db:           db,
	}
}

// ------------------------------------------------------------
// Helping methods and auxiliary functions
// ------------------------------------------------------------

func IPWhiteList(whitelist map[string]bool) gin.HandlerFunc {
	f := func(c *gin.Context) {
		// If the IP isn't in the whitelist, forbid the request.
		ip := c.ClientIP()
		// log.Printf("IP: %s", ip)

		re := regexp.MustCompile(`([\d]+).([\d]+).[\d]+.[\d]+`)
		ipRange := re.ReplaceAllString(ip, "$1.$2.0.0")
		// log.Printf("Ip Range: %s", ipRange)
		if !whitelist[ip] && !whitelist[ipRange] {
			log.Printf("Unauthorized access from %s", ip)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "request from IP not allowed"})
			return
		}
		c.Next()
	}
	return f
}

// ------------------------------------------------------------
// Account authentication and login
// ------------------------------------------------------------

func (a *AuthenticationHandler) Login(c *gin.Context) {
	// Decided to only use the JSON in the body for authentication as everything else is redundant
	// TODO: Even prevent login if header with credentials is set
	// c.GetHeader("Authorization")

	var user data.User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		log.Printf("Login does not contain user information: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// database.PrintUserTable("shoppers")
	// TODO: Move this functionality out of the code and into the data; with a permission column
	specialUser := user.Username
	if specialUser == "admin" {
		log.Print("Admin user logging in, changing login process for debug purposes")
		if user.Password != "12345" {
			log.Print("Incorrect password for user 'Admin'")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		token, err := a.tokenHandler.GenerateNewJWTToken(user.OnlineID, specialUser)
		if err != nil {
			log.Printf("Failed to generate JWT token: %s", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		wireToken := Token{
			Token: token,
		}
		c.JSON(http.StatusOK, wireToken)
		return
	}
	dbUser, err := database.GetUser(user.OnlineID)
	if err != nil {
		log.Printf("User not found!")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// Username and ID must match
	if dbUser.OnlineID != user.OnlineID || dbUser.Username != user.Username {
		log.Print("The stored user does not match the user trying to log in!")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// Check if the given password matches the one stored
	match, err := argon2id.ComparePasswordAndHash(user.Password, dbUser.Password)
	if err != nil {
		log.Printf("Failed to compare password and hash: %s", err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if !match {
		log.Printf("The given password is incorrect for user %d", user.OnlineID)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Generate a new token that is valid for a few minutes to make a few requests
	token, err := a.tokenHandler.GenerateNewJWTToken(user.OnlineID, user.Username)
	if err != nil {
		log.Printf("Failed to generate JWT token: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	log.Print("User found and token generated")
	_, err = database.ModifyLastLogin(user.OnlineID)
	if err != nil {
		log.Printf("Failed to modify last login: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	wireToken := Token{
		Token: token,
	}
	c.JSON(http.StatusOK, wireToken)
}

func (a *AuthenticationHandler) basicTokenAuthenticationFunction(c *gin.Context) {
	origin := c.ClientIP()
	remote := c.RemoteIP()
	log.Printf("Authenticate client from origin: %s, Remote: %s", origin, remote)

	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		log.Print("No token found! Abort")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no token"})
		return
	}
	splits := strings.Split(tokenString, " ")
	var reqToken string
	if len(splits) != 2 {
		if strings.HasPrefix(splits[0], "Authorization") {
			log.Printf("Token in incorrect format! '%s'", tokenString)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "wrong token format"})
			return
		}
		reqToken = splits[0]
	} else {
		reqToken = splits[1]
	}
	claims := Claims{}
	token, err := jwt.ParseWithClaims(reqToken, &claims, func(t *jwt.Token) (interface{}, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("unauthorized")
		}

		if time.Now().After(a.config.JWT.ValidUntil) {
			log.Print("The given secret under 'JWT.Secret' is no longer valid! Please renew the secret in config.json")
			return nil, errors.New("token no longer valid")
		}
		secretKeyByte := []byte(a.config.JWT.Secret)
		return secretKeyByte, nil
	})
	if err != nil {
		log.Printf("Error during token parsing: %s", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	// Checking if user in this form exists
	parsedClaims, ok := token.Claims.(*Claims)
	if !ok {
		log.Print("Received token claims are in incorrect format!")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	user, err := database.GetUser(parsedClaims.Id)
	if err != nil {
		log.Printf("User for id %d not found!", parsedClaims.Id)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if user.Username != parsedClaims.Username {
		log.Print("The stored user and claimed token user do not match")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	// Check if the token was issued
	if err = a.tokenHandler.IsTokenValid(user.OnlineID, token.Raw); err != nil {
		log.Printf("Error with token: %s", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if token.Valid {
		c.Set("userId", claims.Id)
		c.Next()
	} else {
		log.Printf("Invalid claims: %v", claims)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	}
}

func (a *AuthenticationHandler) debugAuthentication(c *gin.Context) {
	c.Next()
}

func (a *AuthenticationHandler) AuthMiddleware() gin.HandlerFunc {
	return a.basicTokenAuthenticationFunction
}

func (a *AuthenticationHandler) DebugAuthenticationMiddleware() gin.HandlerFunc {
	return a.debugAuthentication
}

func (a *AuthenticationHandler) AdminAuthWithoutUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKeyString := c.GetHeader("x-api-key")
		if apiKeyString == "" {
			log.Print("No token found! Abort")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no API token"})
			return
		}
		// Validate the token to be correct
		apiKeyClaims, err := a.ApiKeyValid(apiKeyString)
		if err != nil {
			log.Printf("API Key not valid: %s", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}
		// Validate the user
		if apiKeyClaims.Admin != true {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		c.Next()
	}
}

func (a *AuthenticationHandler) AdminAuthenticationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKeyString := c.GetHeader("x-api-key")
		if apiKeyString == "" {
			log.Print("No token found! Abort")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no API token"})
			return
		}
		// Validate the token to be correct
		apiKeyClaims, err := a.ApiKeyValid(apiKeyString)
		if err != nil {
			log.Printf("API Key not valid: %s", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}
		// Validate the user
		if apiKeyClaims.Admin != true {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		a.basicTokenAuthenticationFunction(c)
	}
}

type ApiKey struct {
	Key        string    `json:"key"`
	ValidUntil time.Time `json:"validUntil"`
	Admin      bool      `json:"admin"`
	jwt.RegisteredClaims
}

type ApiKeySecret struct {
	Secret     string    `json:"secret"`
	ValidUntil time.Time `json:"validUntil"`
}

func (a *AuthenticationHandler) parseApiKeyToClaims(apiKey string) (ApiKey, error) {
	apiKey = strings.TrimSpace(apiKey)
	claims := ApiKey{}
	_, err := jwt.ParseWithClaims(apiKey, &claims, func(t *jwt.Token) (interface{}, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("invalid signing method")
		}
		if time.Now().After(a.config.API.ValidUntil) {
			log.Print("The given secret is no longer valid! Please renew the secret")
			return nil, errors.New("token no longer valid")
		}
		secretKeyByte := []byte(a.config.API.Key)
		return secretKeyByte, nil
	})
	if err != nil {
		return ApiKey{}, err
	}
	return claims, nil
}

func (a *AuthenticationHandler) ApiKeyValid(apiKey string) (ApiKey, error) {
	httpRequestClaims, err := a.parseApiKeyToClaims(apiKey)
	if err != nil {
		return ApiKey{}, err
	}
	if time.Now().After(httpRequestClaims.ValidUntil) {
		return ApiKey{}, errors.New("api key no longer valid")
	}
	if httpRequestClaims.Admin != true {
		return ApiKey{}, errors.New("invalid user rights")
	}
	if httpRequestClaims.Key != a.config.API.Key {
		log.Print("Claimed key does not match secret")
		return ApiKey{}, errors.New("invalid secret")
	}
	log.Printf("API Key is valid")
	return httpRequestClaims, nil
}
