package authentication

import (
	"database/sql"
	"errors"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/configuration"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenHandler struct {
	db     *sql.DB
	config configuration.AuthConfig
}

func NewTokenHandler(db *sql.DB, config configuration.AuthConfig) *TokenHandler {
	return &TokenHandler{
		db:     db,
		config: config,
	}
}

// ------------------------------------------------------------
// The authentication and login data structures
// ------------------------------------------------------------

type Claims struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
	jwt.RegisteredClaims
}

type Token struct {
	Token string `json:"token"`
}

type TokenData struct {
	UserId     int64
	Token      string
	ValidUntil time.Time
}

// ------------------------------------------------------------

func (t *TokenHandler) GenerateNewJWTToken(id int64, username string) (string, error) {
	// Give enough time for a few requests
	notBefore := time.Now()
	expiresAt := notBefore.Add(time.Duration(t.config.KeyTimeoutMs) * time.Millisecond)
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
	signedToken, err := newToken.SignedString([]byte(t.config.Secret))
	if err != nil {
		return "", err
	}
	err = t.storeToken(signedToken, id, expiresAt, true)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

const insertTokenQuery = "INSERT INTO token (userId, token, validUntil) VALUES (?, ?, ?)"

func (t *TokenHandler) storeToken(token string, userId int64, validUntil time.Time, overwrite bool) error {
	if token == "" {
		return errors.New("empty token")
	}
	if overwrite {
		err := t.clearExistingTokensForUser(userId)
		if err != nil {
			return err
		}
	}
	_, err := t.db.Exec(insertTokenQuery, userId, token, validUntil)
	if err != nil {
		return err
	}
	log.Printf("Updated token for user %d valid until %s", userId, validUntil)
	return nil
}

const clearUserTokensQuery = "DELETE FROM token WHERE userId = ?"

func (t *TokenHandler) clearExistingTokensForUser(userId int64) error {
	res, err := t.db.Exec(clearUserTokensQuery, userId)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	log.Printf("Removed %d tokens for user %d", rowsAffected, userId)
	return nil
}

const removeAllExpiredTokens = "DELETE FROM token WHERE validUntil < ?"

func (t *TokenHandler) removeInvalidTokens() error {
	res, err := t.db.Exec(removeAllExpiredTokens, time.Now())
	if err != nil {
		return err
	}
	affectedRows, _ := res.RowsAffected()
	log.Printf("Removed %d rows because the token was invalid", affectedRows)
	return nil
}

const selectUserTokenQuery = "SELECT userId, token, validUntil FROM token WHERE userId = ? ORDER BY validUntil DESC"

func (t *TokenHandler) IsTokenValid(token string) error {
	rows, err := t.db.Query(selectUserTokenQuery, token)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("invalid token")
	}
	var tokenData TokenData
	for rows.Next() {
		if err := rows.Scan(&tokenData.UserId, &tokenData.Token, &tokenData.ValidUntil); err != nil {
			return err
		}
		// If there is more than a single entry skip rest
		break
	}
	if tokenData.ValidUntil.Before(time.Now()) {
		return errors.New("invalid token")
	}
	return nil
}
