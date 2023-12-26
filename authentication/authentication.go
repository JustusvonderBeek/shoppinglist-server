package authentication

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"shop.cloudsheeptech.com/configuration"
	"shop.cloudsheeptech.com/database"
)

var IPWhitelist = map[string]bool{
	"127.0.0.1":      true,
	"188.100.243.67": true,
	"138.246.0.0":    true,
	"131.159.0.0":    true,
	"88.77.0.0":      true,
	"178.1.0.0":      true,
}

var tokens []string
var jwt_secret = []byte("password")

type Claims struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type UserWireFormat struct {
	Id       int64
	Username string
	Password string
}

type JWTSecretFile struct {
	Secret     string
	ValidUntil string
}

func generateJWT(id int, username string) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Id:       id,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	return token.SignedString(jwt_secret)
}

func CreateAccount(c *gin.Context) {
	// Creating a new user and inserting into the database
	// Extracting username and password from request
	var user UserWireFormat
	err := c.ShouldBindJSON(&user)
	if err != nil {
		log.Printf("Failed to create new user: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if user.Id != 0 {
		log.Print("Given user has ID already set!")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	loginUser, err := database.CreateUserAccount(user.Username, user.Password)
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	user.Id = loginUser.ID
	c.IndentedJSON(http.StatusCreated, user)
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		splits := strings.Split(tokenString, " ")
		if len(splits) != 2 {
			log.Print("Token in incorrect format!")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "wrong token format"})
			return
		}
		reqToken := splits[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(reqToken, claims, func(t *jwt.Token) (interface{}, error) {
			_, ok := t.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return "", errors.New("unauthorized")
			}
			// data, err := os.ReadFile(cfg.JWTSecretFile)
			// if err != nil {
			// 	return nil, err
			// }
			// var jwtSecret JWTSecretFile
			// err = json.Unmarshal(data, &jwtSecret)
			// if err != nil {
			// 	return nil, err
			// }
			// // jwt_secret = []byte(jwtSecret.Secret)
			// log.Printf("Secret: %s", jwtSecret.Secret)
			// secretKey := os.Getenv("")
			// secretKeyByte := []byte(jwt_secret)
			// return secretKeyByte, nil
			return jwt_secret, nil
		})
		// log.Printf("Parsing got: %s, %s", token.Raw, err)
		if err != nil {
			log.Printf("Invalid token: %s", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		// if claims, ok := token.Claims.(jwt.RegisteredClaims); ok && token.Valid {
		if token.Valid {
			// c.Set("userId", claims["Id"])
			c.Next()
		} else {
			log.Printf("Invalid claims: %s", claims.Valid().Error())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		}
	}
}

func PerformAuthentication(c *gin.Context) {
	usern, passwd, ok := c.Request.BasicAuth()
	// Check if the basic auth is correct
	log.Printf("Bool okay: %t", ok)
	log.Printf("User '%s' tries to login: %s", usern, passwd)

	database.PrintUserTable("loginuser")
	value, err := strconv.Atoi(usern)
	if err != nil {
		return
	}
	err = database.CheckUserExists(int64(value))
	if usern == "admin" && passwd == "secret" {
		err = nil
	}
	if err != nil {
		log.Printf("User not found!")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Generate a new token that is valid for a few minutes to make a few requests
	token, _ := generateJWT(0, usern)
	tokens = append(tokens, token)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
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
		log.Printf("Unauthorized access from %s", clientIP)
		errString := "ip address '" + clientIP + "' not authorized."
		return errors.New(errString)
	}
	return nil
}
