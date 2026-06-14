package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/vasti/gdc-backend-test/internal/model"
	"github.com/vasti/gdc-backend-test/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and authorization logic.
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// Register creates a new user account and returns the user with a signed JWT token.
func (s *AuthService) Register(ctx context.Context, email, password, name string, teamID *uuid.UUID) (*model.User, string, error) {
	// Validate required fields
	if email == "" {
		return nil, "", model.ErrValidation("email is required", nil)
	}
	if password == "" {
		return nil, "", model.ErrValidation("password is required", nil)
	}
	if name == "" {
		return nil, "", model.ErrValidation("name is required", nil)
	}

	// Check email uniqueness
	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", model.ErrInternal("failed to check email uniqueness", err)
	}
	if existing != nil {
		return nil, "", model.ErrConflict("email already registered", nil)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", model.ErrInternal("failed to hash password", err)
	}

	// Create user
	user := &model.User{
		Email:    email,
		Password: string(hashedPassword),
		Name:     name,
		TeamID:   teamID,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", model.ErrInternal("failed to create user", err)
	}

	// Sign JWT
	token, err := s.signJWT(user.ID)
	if err != nil {
		return nil, "", model.ErrInternal("failed to sign token", err)
	}

	return user, token, nil
}

// Login authenticates a user by email and password, returning a JWT token.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", model.ErrInternal("failed to find user", err)
	}
	if user == nil {
		return "", model.ErrUnauthorized("invalid email or password", nil)
	}

	// Compare password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", model.ErrUnauthorized("invalid email or password", nil)
	}

	// Sign JWT
	token, err := s.signJWT(user.ID)
	if err != nil {
		return "", model.ErrInternal("failed to sign token", err)
	}

	return token, nil
}

// ValidateTokenAndGetUserID parses a JWT token and extracts the user ID.
func (s *AuthService) ValidateTokenAndGetUserID(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return uuid.Nil, model.ErrUnauthorized("invalid token", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, model.ErrUnauthorized("invalid token claims", nil)
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, model.ErrUnauthorized("invalid token subject", nil)
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, model.ErrUnauthorized("invalid user id in token", err)
	}

	return userID, nil
}

// signJWT creates a signed JWT token for the given user ID with a 24-hour expiry.
func (s *AuthService) signJWT(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
