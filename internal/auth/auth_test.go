package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// generateTestKeys creates a fresh RSA key pair for use in tests.
func generateTestKeys(t *testing.T) (privPEM, pubPEM []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privPEM, pubPEM
}

func mustGeneratePrivPEM(t *testing.T) []byte {
	t.Helper()
	priv, _ := generateTestKeys(t)
	return priv
}

func mustGeneratePubPEM(t *testing.T) []byte {
	t.Helper()
	_, pub := generateTestKeys(t)
	return pub
}

func TestNewFromBytes(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	svc, err := NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewFromBase64(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	svc, err := NewFromBase64(
		base64.StdEncoding.EncodeToString(privPEM),
		base64.StdEncoding.EncodeToString(pubPEM),
	)
	if err != nil {
		t.Fatalf("NewFromBase64: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewFromBase64_InvalidEncoding(t *testing.T) {
	tests := []struct {
		name    string
		privB64 string
		pubB64  string
		wantErr string
	}{
		{"bad private key", "!!!notbase64!!!", base64.StdEncoding.EncodeToString(mustGeneratePubPEM(t)), "decode private key"},
		{"bad public key", base64.StdEncoding.EncodeToString(mustGeneratePrivPEM(t)), "!!!notbase64!!!", "decode public key"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewFromBase64(tc.privB64, tc.pubB64)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestNewFromBytes_InvalidKeys(t *testing.T) {
	validPriv := mustGeneratePrivPEM(t)
	validPub := mustGeneratePubPEM(t)
	tests := []struct {
		name    string
		privPEM []byte
		pubPEM  []byte
		wantErr string
	}{
		{"bad private key", []byte("not a pem"), validPub, "parse private key"},
		{"bad public key", validPriv, []byte("not a pem"), "parse public key"},
		{"empty private key", []byte{}, validPub, "parse private key"},
		{"empty public key", validPriv, []byte{}, "parse public key"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewFromBytes(tc.privPEM, tc.pubPEM)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	svc, err := NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}

	tests := []struct {
		name   string
		userID string
		orgID  string
		role   string
	}{
		{"standard user", "user-123", "org-456", "admin"},
		{"empty user", "", "org-456", "viewer"},
		{"empty org", "user-123", "", "admin"},
		{"empty role", "user-123", "org-456", ""},
		{"uuid-like values", "550e8400-e29b-41d4-a716-446655440000", "org-uuid-123", "operator"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := svc.GenerateToken(tc.userID, tc.orgID, tc.role)
			if err != nil {
				t.Fatalf("GenerateToken: %v", err)
			}
			if token == "" {
				t.Fatal("expected non-empty token")
			}

			claims, err := svc.ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken: %v", err)
			}
			if claims.UserID != tc.userID {
				t.Errorf("UserID: got %q, want %q", claims.UserID, tc.userID)
			}
			if claims.OrgID != tc.orgID {
				t.Errorf("OrgID: got %q, want %q", claims.OrgID, tc.orgID)
			}
			if claims.Role != tc.role {
				t.Errorf("Role: got %q, want %q", claims.Role, tc.role)
			}
			if claims.Issuer != "watch-dog" {
				t.Errorf("Issuer: got %q, want %q", claims.Issuer, "watch-dog")
			}
		})
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	svc, err := NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr string
	}{
		{"empty token", "", "validate token"},
		{"garbage", "not.a.jwt", "validate token"},
		{"random string", "abcdefghijklmnopqrstuvwxyz", "validate token"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.ValidateToken(tc.token)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestValidateToken_WrongPublicKey(t *testing.T) {
	privPEM1, _ := generateTestKeys(t)
	_, pubPEM2 := generateTestKeys(t)
	svc, err := NewFromBytes(privPEM1, pubPEM2)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	// Generate a token with a different private key
	privPEM2, _ := generateTestKeys(t)
	svc2, err := NewFromBytes(privPEM2, pubPEM2)
	if err != nil {
		t.Fatalf("NewFromBytes svc2: %v", err)
	}
	token, err := svc2.GenerateToken("user", "org", "role")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	// Validate with svc which has a different public key than svc2's private key
	_, err = svc.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for token signed by different key")
	}
}

func TestValidateToken_RejectsHS256(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	svc, err := NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	// Create an HS256 token (should be rejected since we expect RSA)
	claims := Claims{
		UserID: "user",
		OrgID:  "org",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "watch-dog",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	_, err = svc.ValidateToken(signed)
	if err == nil {
		t.Fatal("expected error for HS256 token")
	}
	if !strings.Contains(err.Error(), "validate token") {
		t.Errorf("expected validate token error, got %q", err.Error())
	}
}

func TestGenerateToken_SigningMethod(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	svc, err := NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	token, err := svc.GenerateToken("user", "org", "role")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	// Parse without claims validation to check the header
	parsed, _, err := jwt.NewParser().ParseUnverified(token, &Claims{})
	if err != nil {
		t.Fatalf("ParseUnverified: %v", err)
	}
	if parsed.Method != jwt.SigningMethodRS256 {
		t.Errorf("expected RS256, got %v", parsed.Method)
	}
}

func TestTokenExpiry(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	svc, err := NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	token, err := svc.GenerateToken("user", "org", "role")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
	// Token should expire in ~24 hours
	expiryDiff := claims.ExpiresAt.Time.Sub(time.Now())
	if expiryDiff < 23*time.Hour || expiryDiff > 25*time.Hour {
		t.Errorf("expected ~24h expiry, got %v", expiryDiff)
	}
}

func TestNewFromFile(t *testing.T) {
	// Test with non-existent files
	_, err := New("/nonexistent/private.pem", "/nonexistent/public.pem")
	if err == nil {
		t.Fatal("expected error for non-existent files")
	}
	if !strings.Contains(err.Error(), "read private key") {
		t.Errorf("expected read private key error, got %q", err.Error())
	}
}
