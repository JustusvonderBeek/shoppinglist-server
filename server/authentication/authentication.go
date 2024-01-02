package authentication

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"shop.cloudsheeptech.com/database"
	"shop.cloudsheeptech.com/server/configuration"
	"shop.cloudsheeptech.com/server/data"
)

var IPWhitelist = map[string]bool{
	"127.0.0.1":      true,
	"188.100.243.67": true,
	"138.246.0.0":    true,
	"131.159.0.0":    true,
	"88.77.0.0":      true,
	"178.1.0.0":      true,
}

var config configuration.Config
var tokens []string

// ------------------------------------------------------------
// The authentication and login data structures
// ------------------------------------------------------------

type Claims struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JWTSecretFile struct {
	Secret     string
	ValidUntil time.Time
}

// ------------------------------------------------------------
// Setup and configuration
// ------------------------------------------------------------

func Setup(cfg configuration.Config) {
	config = cfg
}

// ------------------------------------------------------------
// Helping methods and auxiliary functions
// ------------------------------------------------------------

func writeDefaultJWTSecretFile() {
	secretFile := JWTSecretFile{
		Secret:     "<enter secret here>",
		ValidUntil: time.Now().AddDate(0, 3, 0), // Adding 3 months as duration
		// ValidUntil: time.Now(), // For testing purposes
	}
	raw, err := json.MarshalIndent(secretFile, "", "\t")
	if err != nil {
		log.Printf("Failed to convert JWT file struct to raw data: %s", err)
		return
	}
	pwd, _ := os.Getwd()
	finalJWTFile := filepath.Join(pwd, config.JWTSecretFile)
	err = os.WriteFile(finalJWTFile, raw, 0760)
	if err != nil {
		log.Printf("Failed to store JWT secret to file: %s", err)
	}
	log.Printf("Stored default JWT secret file in: %s", finalJWTFile)
}

func generateJWT(id int, username string) (string, error) {
	// Give enough time for a few requests
	expirationTime := time.Now().Add(3 * time.Minute)
	claims := &Claims{
		Id:       id,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	// Load the secret from the file system and sign the token
	pwd, _ := os.Getwd()
	finalJWTFile := filepath.Join(pwd, config.JWTSecretFile)
	content, err := os.ReadFile(finalJWTFile)
	if err != nil {
		log.Printf("Error during token generation. Cannot read token secret file! %s", err)
		writeDefaultJWTSecretFile()
		return "", err
	}
	var jwtSecretFile JWTSecretFile
	err = json.Unmarshal(content, &jwtSecretFile)
	if err != nil {
		log.Printf("The given jwt secret file is in incorrect format! %s", err)
		return "", err
	}
	if time.Now().After(jwtSecretFile.ValidUntil) {
		log.Print("The given secret is no longer valid! Please renew the secret")
		return "", errors.New("token no longer valid")
	}
	return token.SignedString([]byte(jwtSecretFile.Secret))
}

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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "fuck you in the ass blyad"})
			return
		}
		c.Next()
	}
	return f
}

func CheckIPWhitelisted(clientIP string, whitelist map[string]bool) error {
	// If the IP isn't in the whitelist, forbid the request.
	// log.Printf("IP: %s", clientIP)

	re := regexp.MustCompile(`([\d]+).([\d]+).[\d]+.[\d]+`)
	ipRange := re.ReplaceAllString(clientIP, "$1.$2.0.0")
	// log.Printf("Ip Range: %s", ipRange)
	if !whitelist[clientIP] && !whitelist[ipRange] {
		log.Printf("Unauthorized access from IP '%s'", clientIP)
		errString := "ip address '" + clientIP + "' not authorized."
		return errors.New(errString)
	}
	return nil
}

// ------------------------------------------------------------
// Account creation
// ------------------------------------------------------------

func CreateAccount(c *gin.Context) {
	// Creating a new user and inserting into the database
	// First checking if the creation is from a whitelisted IP address
	origin := c.ClientIP()
	err := CheckIPWhitelisted(origin, IPWhitelist)
	if err != nil {
		log.Printf("Request origin %s not from a whitelisted IP address!", origin)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	// Extracting username and password from request
	var user data.User
	err = c.ShouldBindJSON(&user)
	if err != nil {
		log.Printf("Failed decode given user: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if user.ID != 0 {
		log.Print("Given user has ID already set!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if user.Username == "" || user.Password == "" {
		log.Print("Username or password not set!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	loginUser, err := database.CreateUserAccount(user.Username, user.Password)
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// Dont include hashed password information in answer
	loginUser.Password = "accepted"
	c.IndentedJSON(http.StatusCreated, loginUser)
}

// ------------------------------------------------------------
// Account authentication and login
// ------------------------------------------------------------

func Login(c *gin.Context) {
	// Decided to only use the JSON in the body for authentication as everything else is redundant
	var user data.User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		log.Printf("Login does not contain user information: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// database.PrintUserTable("shoppers")
	err = database.CheckUserExists(int64(user.ID))
	if err != nil {
		log.Printf("User not found!")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// Generate a new token that is valid for a few minutes to make a few requests
	token, err := generateJWT(int(user.ID), user.Username)
	if err != nil {
		log.Printf("Failed to generate JWT token: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	tokens = append(tokens, token)
	log.Print("User found and token generated")
	// log.Printf("Sending token: %s", token)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

func AuthenticationMiddleware(cfg configuration.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// body, _ := io.ReadAll(c.Request.Body)
		// header := c.Request.Header
		origin := c.ClientIP()
		remote := c.RemoteIP()
		// log.Printf("Request body: %s", body)
		// log.Printf("Request header: %s", header)
		log.Printf("Origin: %s, Remote: %s", origin, remote)

		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			log.Print("No token found! Abort")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no token"})
			return
		}
		splits := strings.Split(tokenString, " ")
		if len(splits) != 2 {
			log.Printf("Token in incorrect format! '%s'", tokenString)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "wrong token format"})
			return
		}
		reqToken := splits[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(reqToken, claims, func(t *jwt.Token) (interface{}, error) {
			_, ok := t.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, errors.New("unauthorized")
			}
			pwd, _ := os.Getwd()
			finalJWTFile := filepath.Join(pwd, config.JWTSecretFile)
			data, err := os.ReadFile(finalJWTFile)
			if err != nil {
				log.Print("Failed to find JWT secret file")
				return nil, err
			}
			var jwtSecret JWTSecretFile
			err = json.Unmarshal(data, &jwtSecret)
			if err != nil {
				log.Print("JWT secret file is in incorrect format")
				return nil, err
			}
			if time.Now().After(jwtSecret.ValidUntil) {
				log.Print("The given secret is no longer valid! Please renew the secret")
				return nil, errors.New("token no longer valid")
			}
			secretKeyByte := []byte(jwtSecret.Secret)
			return secretKeyByte, nil
		})
		// log.Printf("Parsing got: %s, %s", token.Raw, err)
		if err != nil {
			log.Printf("Error during token parsing: %s", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		// if claims, ok := token.Claims.(jwt.RegisteredClaims); ok && token.Valid {
		// TODO: Update checking token validity, include if we generated the token and if user exists
		if token.Valid {
			// c.Set("userId", claims["Id"])
			c.Next()
		} else {
			log.Printf("Invalid claims: %s", claims.Valid().Error())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		}
	}
}
