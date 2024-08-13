package authentication

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
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

type IpWhiteList struct {
	IPs []string `json:"ips"`
}

var config configuration.Config
var tokens []string
var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

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

type Token struct {
	Token string `json:"token"`
}

// ------------------------------------------------------------
// Setup and configuration
// ------------------------------------------------------------

func Setup(cfg configuration.Config) {
	config = cfg
	if ips, err := readIpWhitelistFromFile(); err != nil {
		log.Printf("Failed to read ips from disk: %s", err)
	} else if len(ips) > 0 {
		IPWhitelist = ips
	}
	if tkns, err := readTokensFromDisk(); err != nil {
		log.Printf("Failed to read tokens from disk: %s", err)
	} else {
		validTokens, err := removeInvalidTokens(tkns)
		if err != nil {
			log.Printf("Tokens invalid")
			return
		}
		tokens = validTokens
		storeTokensToDisk(true)
	}
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

func storeTokensToDisk(overwrite bool) error {
	// Dont overwrite if already existing
	finalTokenPath := filepath.Join(basepath, "../../resources/tokens.txt")

	exists := true
	if _, err := os.Stat(finalTokenPath); errors.Is(err, os.ErrNotExist) {
		exists = false
	}
	fileMode := os.O_CREATE | os.O_WRONLY
	if overwrite {
		fileMode = fileMode | os.O_TRUNC
	} else {
		fileMode = fileMode | os.O_APPEND
	}
	file, err := os.OpenFile(finalTokenPath, fileMode, 0660)
	if err != nil {
		return err
	}
	for i, token := range tokens {
		if i == 0 && !exists { // Only use this mode for the very first token written to the file
			_, err := file.Write([]byte(token))
			if err != nil {
				return err
			}
			continue
		}
		_, err := file.Write([]byte("," + token))
		if err != nil {
			return err
		}
	}
	return nil
}

func readTokensFromDisk() ([]string, error) {
	finalTokenPath := filepath.Join(basepath, "../../resources/tokens.txt")
	content, err := os.ReadFile(finalTokenPath)
	if err != nil {
		return nil, err
	}
	readTokens := strings.Split(string(content), ",")
	log.Printf("Read %d tokens from disk", len(readTokens))
	return readTokens, nil
}

func removeInvalidTokens(tokens []string) ([]string, error) {
	claims := Claims{}
	var validTokens []string
	for _, token := range tokens {
		_, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
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
			if err = json.Unmarshal(data, &jwtSecret); err != nil {
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
		if err != nil {
			// log.Printf("Token no longer valid? %s", err)
			continue
		}
		validTokens = append(validTokens, token)
	}
	log.Printf("Removed %d tokens", len(tokens)-len(validTokens))
	return validTokens, nil
}

func generateJWT(id int, username string) (string, error) {
	// Give enough time for a few requests
	expirationTime := time.Now().Add(time.Duration(config.JWTTimeout) * time.Second)
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

func checkJWTTokenIssued(token string) error {
	storedTokens, err := readTokensFromDisk()
	if err != nil {
		return errors.New("no stored tokens found")
	}
	allTokens := append(storedTokens, tokens...)
	for _, storedTkn := range allTokens {
		if storedTkn == token {
			return nil
		}
	}
	return errors.New("token not found")
}

func readIpWhitelistFromFile() (map[string]bool, error) {
	finalTokenPath := filepath.Join(basepath, "../../resources/whitelisted_ips.json")
	content, err := os.ReadFile(finalTokenPath)
	if err != nil {
		return nil, err
	}
	var ips IpWhiteList
	if err = json.Unmarshal(content, &ips); err != nil {
		return nil, err
	}
	log.Printf("Found IP Whitelist with %d IPs", len(ips.IPs))
	whiteIps := make(map[string]bool, len(ips.IPs))
	for _, ip := range ips.IPs {
		whiteIps[ip] = true
	}
	return whiteIps, nil
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "request from IP not allowed"})
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
	if user.OnlineID != 0 {
		log.Print("Given user has ID already set!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if user.Username == "" || user.Password == "" {
		log.Print("Username or password not set!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	loginUser, err := database.CreateUserAccountInDatabase(user.Username, user.Password)
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// Dont include hashed password information in answer
	loginUser.Password = "accepted"
	c.JSON(http.StatusCreated, loginUser)
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

// ------------------------------------------------------------
// Account authentication and login
// ------------------------------------------------------------

func Login(c *gin.Context) {
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
	dbUser, err := database.GetUser(int64(user.OnlineID))
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
	token, err := generateJWT(int(user.OnlineID), user.Username)
	if err != nil {
		log.Printf("Failed to generate JWT token: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	tokens = append(tokens, token)
	// Store the token to disk, in case of failure and persistency
	if err = storeTokensToDisk(false); err != nil {
		log.Printf("Failed to store tokens to disk: %s", err)
	}

	log.Print("User found and token generated")
	database.ModifyLastLogin(user.OnlineID)

	// log.Printf("Sending token: %s", token)

	wireToken := Token{
		Token: token,
	}
	c.JSON(http.StatusOK, wireToken)
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

		// TODO: Add the API token-
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
		claims := Claims{}
		token, err := jwt.ParseWithClaims(reqToken, &claims, func(t *jwt.Token) (interface{}, error) {
			_, ok := t.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, errors.New("unauthorized")
			}
			// parsedClaim, ok := t.Claims.(Claims)
			// if !ok {
			// 	log.Print("Token in invalid format")
			// 	return nil, errors.New("token in invalid format")
			// }
			// log.Printf("Token is issued for: %d", parsedClaim.Id)
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
		// Checking if user in this form exists
		// TODO: Find a way to extract the custom information from the token
		parsedClaims, ok := token.Claims.(*Claims)
		if !ok {
			log.Print("Received token claims are in incorrect format!")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		user, err := database.GetUser(int64(parsedClaims.Id))
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
		if err = checkJWTTokenIssued(reqToken); err != nil {
			log.Printf("Error with token: %s", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if token.Valid {
			c.Set("userId", int64(claims.Id))
			c.Next()
		} else {
			log.Printf("Invalid claims: %s", claims.Valid().Error())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		}
	}
}
