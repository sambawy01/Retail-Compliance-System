// Package auth provides JWT authentication using RS256.
package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT payload for Watch Dog.
type Claims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Service manages JWT signing and validation.
type Service struct {
	privateKey any
	publicKey  any
}

// New loads the RSA private and public keys from the given file paths.
func New(privateKeyPath, publicKeyPath string) (*Service, error) {
	privData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("auth: read private key: %w", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privData)
	if err != nil {
		return nil, fmt.Errorf("auth: parse private key: %w", err)
	}

	pubData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("auth: read public key: %w", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubData)
	if err != nil {
		return nil, fmt.Errorf("auth: parse public key: %w", err)
	}

	return &Service{privateKey: privKey, publicKey: pubKey}, nil
}

// NewFromBytes creates a Service from raw PEM key bytes.
func NewFromBytes(privPEM, pubPEM []byte) (*Service, error) {
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		return nil, fmt.Errorf("auth: parse private key: %w", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubPEM)
	if err != nil {
		return nil, fmt.Errorf("auth: parse public key: %w", err)
	}
	return &Service{privateKey: privKey, publicKey: pubKey}, nil
}

// NewFromBase64 creates a Service from base64-encoded PEM keys.
func NewFromBase64(privB64, pubB64 string) (*Service, error) {
	privPEM, err := base64.StdEncoding.DecodeString(privB64)
	if err != nil {
		return nil, fmt.Errorf("auth: decode private key: %w", err)
	}
	pubPEM, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil {
		return nil, fmt.Errorf("auth: decode public key: %w", err)
	}
	return NewFromBytes(privPEM, pubPEM)
}

// GenerateToken creates a signed JWT for the given user with the default 24h expiry.
func (s *Service) GenerateToken(userID, orgID, role string) (string, error) {
	return s.GenerateTokenWithExpiry(userID, orgID, role, 24*time.Hour)
}

// GenerateTokenWithExpiry creates a signed JWT with a custom TTL.
func (s *Service) GenerateTokenWithExpiry(userID, orgID, role string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "watch-dog",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("auth: sign token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT string.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("auth: unexpected signing method")
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: validate token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("auth: invalid token claims")
	}
	return claims, nil
}