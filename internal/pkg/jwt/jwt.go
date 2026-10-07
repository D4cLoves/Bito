package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenID      string `json:"token_id,omitempty"`
}

type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	TokenType string    `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// GenerateTokenPair создает пару access и refresh токенов с уникальным JTI
func (tm *TokenManager) GenerateTokenPair(userID uuid.UUID, username string) (*TokenPair, error) {
	access, err := tm.GenerateAccessToken(userID, username)
	if err != nil {
		return nil, fmt.Errorf("generate access: %w", err)
	}

	refresh, tokenID, err := tm.GenerateRefreshToken(userID, username)
	if err != nil {
		return nil, fmt.Errorf("generate refresh: %w", err)
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenID:      tokenID,
	}, nil
}

// GenerateAccessToken создает короткоживущий Access Token (15 минут)
func (tm *TokenManager) GenerateAccessToken(userID uuid.UUID, username string) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tm.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken создает долгоживущий Refresh Token (7 дней) с уникальным JTI (tokenID)
func (tm *TokenManager) GenerateRefreshToken(userID uuid.UUID, username string) (string, string, error) {
	now := time.Now()
	tokenID := uuid.NewString()

	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tm.secret)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return tokenString, tokenID, nil
}

// ValidateAccessToken проверяет валидность access токена
func (tm *TokenManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := tm.parseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ValidateRefreshToken проверяет валидность refresh токена
func (tm *TokenManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := tm.parseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (tm *TokenManager) parseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	parsedToken, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return tm.secret, nil
		},
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	if !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
