package authentication

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/util"
)

// ------------------------------------------------------------
// The authentication and login data structures
// ------------------------------------------------------------

type Claims struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
	jwt.RegisteredClaims
}

type JWTSecretFile struct {
	Secret     string
	ValidUntil time.Time
}

type Token struct {
	Token string `json:"token"`
}

var tokens []string

func SetupTokenHandler() error {
	loadedTokens, err := setup()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			loadedTokens = []string{}
		} else {
			return err
		}
	}
	tokens = loadedTokens
	return nil
}

func GenerateNewJWTToken(id int, username string) (string, error) {
	// Give enough time for a few requests
	notBefore := time.Now()
	expiresAt := notBefore.Add(time.Duration(config.JwtTimeout) * time.Second)
	claims := &Claims{
		Id:       id,
		Username: username,
		Admin:    false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "shopping-list-server",
			NotBefore: jwt.NewNumericDate(notBefore),
		},
	}
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	secretFileReader := util.ConfigReader{Filename: config.JwtSecretFile}
	secret, err := loadSecretFromDisk(&secretFileReader)
	if err != nil {
		return "", err
	}
	signedToken, err := newToken.SignedString([]byte(secret.Secret))
	if err != nil {
		return "", err
	}
	updatedTokens := append(tokens, signedToken)
	tokens = updatedTokens
	err = storeTokensToDisk(updatedTokens, "resources/tokens.txt", false)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func setup() ([]string, error) {
	loadedTokens, err := readTokensFromDisk()
	if err != nil {
		return nil, err
	}
	validTokens, err := removeInvalidTokens(loadedTokens)
	if err != nil {
		return nil, err
	}
	err = storeTokensToDisk(validTokens, "resources/tokens.txt", true)
	if err != nil {
		return nil, err
	}
	return validTokens, nil
}

func readTokensFromDisk() ([]string, error) {
	content, err := util.ReadFileFromRoot("resources/tokens.txt")
	if err != nil {
		return nil, err
	}
	readTokens := strings.Split(string(content), ",")
	log.Printf("Read %d tokens from disk", len(readTokens))
	return readTokens, nil
}

func storeTokensToDisk(tokens []string, filename string, overwrite bool) error {
	// Dont overwrite if already existing
	joinedStrings := strings.Join(tokens, ",")
	if overwrite {
		writtenFilepath, _, err := util.OverwriteFileAtRoot([]byte(joinedStrings), filename)
		log.Printf("Stored %d tokens to file: %s", len(joinedStrings), writtenFilepath)
		return err
	}
	writtenFilepath, _, err := util.WriteFileAtRoot([]byte(joinedStrings), filename, false)
	log.Printf("Stored %d tokens to file: %s", len(joinedStrings), writtenFilepath)
	return err
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
			finalJWTFile := filepath.Join(pwd, config.JwtSecretFile)
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

func loadSecretFromDisk(reader util.IReader) (JWTSecretFile, error) {
	content, err := reader.ReadConfig()
	if err != nil && os.IsNotExist(err) {
		jwtSecretFile := createNewJWTSecret()
		err = storeJWTSecret(jwtSecretFile)
		if err != nil {
			return JWTSecretFile{}, err
		}
		log.Fatalf("Wrote default JWT secret file. Please set the secret under: '%s'", config.JwtSecretFile)
	} else if err != nil {
		return JWTSecretFile{}, err
	}
	var jwtSecret JWTSecretFile
	err = json.Unmarshal(content, &jwtSecret)
	if err != nil {
		return JWTSecretFile{}, err
	}
	if time.Now().After(jwtSecret.ValidUntil) {
		log.Fatalf("The JWT Secret lost validity at: %s", jwtSecret.ValidUntil)
	}
	return jwtSecret, nil
}

func createNewJWTSecret() JWTSecretFile {
	secretFile := JWTSecretFile{
		Secret:     "<enter secret here>",
		ValidUntil: time.Now().AddDate(0, 3, 0), // Adding 3 months as duration
	}
	return secretFile
}

func storeJWTSecret(jwtSecret JWTSecretFile) error {
	rawJwtSecretFile, err := json.Marshal(jwtSecret)
	if err != nil {
		return err
	}
	writtenFilepath, _, err := util.OverwriteFileAtRoot(rawJwtSecretFile, config.JwtSecretFile)
	log.Printf("Stored JWT secret file at: %s", writtenFilepath)
	return err
}

func IsTokenValid(token string) error {
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
	return errors.New("token was not issued by this server")
}
