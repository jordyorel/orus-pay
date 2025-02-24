package utils

import (
	"errors"
	"os"
	"strconv"
	"time"

	"orus/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateTokens generates an access token and a refresh token for the given user claims.
// The JWT secret is expected to be set in the environment variable JWT_SECRET.
func GenerateTokens(claims *models.UserClaims) (accessToken string, refreshToken string, err error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", "", errors.New("JWT_SECRET not configured")
	}

	now := time.Now()

	// Create access token claims using models.UserClaims; include all necessary fields.
	accessClaims := models.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "orus-api",
			Subject:   strconv.FormatUint(uint64(claims.UserID), 10),
		},
		UserID:       claims.UserID,
		Email:        claims.Email,
		Role:         claims.Role,
		Permissions:  claims.Permissions,
		TokenVersion: claims.TokenVersion,
	}
	accessJwt := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessJwt.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", err
	}

	// Create refresh token claims. (Include needed fields; you might skip some if desired.)
	refreshClaims := models.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "orus-api",
			Subject:   strconv.FormatUint(uint64(claims.UserID), 10),
		},
		UserID:       claims.UserID,
		Email:        claims.Email,
		Role:         claims.Role,
		TokenVersion: claims.TokenVersion,
	}
	refreshJwt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshJwt.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// ParseToken parses and validates a JWT token string.
// It returns the token if valid, or an error if something is wrong.
func ParseToken(tokenStr string) (*jwt.Token, *models.UserClaims, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, nil, errors.New("JWT_SECRET not configured")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, nil, err
	}

	claims, ok := token.Claims.(*models.UserClaims)
	if !ok || !token.Valid {
		return nil, nil, errors.New("invalid token claims")
	}

	return token, claims, nil
}
