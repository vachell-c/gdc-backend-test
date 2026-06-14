package unit_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vasti/gdc-backend-test/internal/repository"
	"github.com/vasti/gdc-backend-test/internal/service"
)

// Helper to create an AuthService without needing a real DB pool.
// The service will have a nil UserRepository, so only operations that
// don't hit the DB (validation, JWT) will work.
func newTestAuthService(jwtSecret string) *service.AuthService {
	return service.NewAuthService(nil, jwtSecret)
}

// Helper to create an AuthService with a given repo (nil allowed for testing).
func newTestAuthServiceWithRepo(repo *repository.UserRepository, jwtSecret string) *service.AuthService {
	return service.NewAuthService(repo, jwtSecret)
}

// --- Register validation tests (no DB needed) ---

func TestRegister_EmptyEmail(t *testing.T) {
	svc := newTestAuthService("test-secret")
	_, _, err := svc.Register(nil, "", "password123", "Test User", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ERR_VALIDATION")
	assert.Contains(t, err.Error(), "email is required")
}

func TestRegister_EmptyPassword(t *testing.T) {
	svc := newTestAuthService("test-secret")
	_, _, err := svc.Register(nil, "test@example.com", "", "Test User", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ERR_VALIDATION")
	assert.Contains(t, err.Error(), "password is required")
}

func TestRegister_EmptyName(t *testing.T) {
	svc := newTestAuthService("test-secret")
	_, _, err := svc.Register(nil, "test@example.com", "password123", "", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ERR_VALIDATION")
	assert.Contains(t, err.Error(), "name is required")
}

// --- JWT sign + validate round-trip ---

func TestJWT_SignAndValidate(t *testing.T) {
	svc := newTestAuthService("my-jwt-secret-key-12345")
	userID := uuid.New()

	// Build a token using the same algorithm as service.signJWT
	tokenStr := signTestJWT(userID, "my-jwt-secret-key-12345")

	// Now validate the token
	extractedID, err := svc.ValidateTokenAndGetUserID(tokenStr)
	assert.NoError(t, err)
	assert.Equal(t, userID, extractedID)
}

func TestJWT_ValidateInvalidToken(t *testing.T) {
	svc := newTestAuthService("my-jwt-secret-key-12345")

	_, err := svc.ValidateTokenAndGetUserID("invalid-token-string")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ERR_UNAUTHORIZED")
}

func TestJWT_ValidateWrongSecret(t *testing.T) {
	svc := newTestAuthService("my-jwt-secret-key-12345")

	// Sign with a different secret
	userID := uuid.New()
	tokenStr := signTestJWT(userID, "different-secret")

	_, err := svc.ValidateTokenAndGetUserID(tokenStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ERR_UNAUTHORIZED")
}

func TestJWT_ValidateTamperedToken(t *testing.T) {
	svc := newTestAuthService("my-jwt-secret-key-12345")

	userID := uuid.New()
	tokenStr := signTestJWT(userID, "my-jwt-secret-key-12345")

	// Tamper with the token by changing a character near the end
	tampered := tokenStr[:len(tokenStr)-5] + "XXXXX"

	_, err := svc.ValidateTokenAndGetUserID(tampered)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ERR_UNAUTHORIZED")
}

func TestJWT_ValidateEmptyToken(t *testing.T) {
	svc := newTestAuthService("my-jwt-secret-key-12345")

	_, err := svc.ValidateTokenAndGetUserID("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ERR_UNAUTHORIZED")
}

// --- Concurrent JWT validation test ---

func TestJWT_ConcurrentValidation(t *testing.T) {
	svc := newTestAuthService("concurrent-secret")
	userID := uuid.New()
	tokenStr := signTestJWT(userID, "concurrent-secret")

	const goroutines = 20
	errs := make(chan error, goroutines)

	for range goroutines {
		go func() {
			id, err := svc.ValidateTokenAndGetUserID(tokenStr)
			if err != nil {
				errs <- err
				return
			}
			if id != userID {
				errs <- assert.AnError
				return
			}
			errs <- nil
		}()
	}

	for range goroutines {
		err := <-errs
		assert.NoError(t, err)
	}
}

// --- Test that Register calls find by email (negative test with nil repo) ---

func TestRegister_NilRepoPanics(t *testing.T) {
	// With a nil *repository.UserRepository, calling s.userRepo.FindByEmail
	// causes a nil pointer dereference panic.
	// This test verifies that the code flow reaches the DB call.
	svc := newTestAuthServiceWithRepo(nil, "test-secret")

	assert.Panics(t, func() {
		_, _, _ = svc.Register(nil, "existing@example.com", "password123", "Test User", nil)
	}, "Register with nil repo should panic when calling FindByEmail")
}

// signTestJWT creates a JWT token matching the format produced by
// AuthService.signJWT (HS256, sub=userID, exp=24h, iat=now).
func signTestJWT(userID uuid.UUID, secret string) string {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		panic("failed to sign test JWT: " + err.Error())
	}
	return signed
}
