package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/vasti/gdc-backend-test/internal/model"
	"github.com/vasti/gdc-backend-test/internal/service"
)

// AuthHandler handles authentication HTTP requests.
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authSvc: authSvc,
	}
}

// registerRequest is the JSON body for POST /register.
type registerRequest struct {
	Email    string     `json:"email"`
	Password string     `json:"password"`
	Name     string     `json:"name"`
	TeamID   *uuid.UUID `json:"team_id,omitempty"`
}

// loginRequest is the JSON body for POST /login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates a new user account.
func (h *AuthHandler) Register(c *echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return model.ErrValidation("invalid request body", err)
	}

	user, token, err := h.authSvc.Register(c.Request().Context(), req.Email, req.Password, req.Name, req.TeamID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// Login authenticates a user and returns a JWT token.
func (h *AuthHandler) Login(c *echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return model.ErrValidation("invalid request body", err)
	}

	token, err := h.authSvc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"token": token,
	})
}
