package middleware

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	emw "github.com/labstack/echo/v5/middleware"
	"github.com/vasti/gdc-backend-test/internal/model"
)

func bearerExtractor(c *echo.Context) ([]string, emw.ExtractorSource, error) {
	auth := c.Request().Header.Get(echo.HeaderAuthorization)
	if auth == "" {
		return nil, "", echo.ErrUnauthorized
	}
	// Strip "Bearer " prefix if present
	if len(auth) > 7 && auth[:7] == "Bearer " {
		auth = auth[7:]
	}
	return []string{auth}, emw.ExtractorSourceHeader, nil
}

func AuthMiddleware(jwtSecret string) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey:       []byte(jwtSecret),
		TokenLookupFuncs: []emw.ValuesExtractor{bearerExtractor},
		ContextKey:       "user",
		ErrorHandler: func(c *echo.Context, err error) error {
			return model.ErrUnauthorized("invalid or missing token", err)
		},
		SuccessHandler: func(c *echo.Context) error {
			token, ok := c.Get("user").(*jwt.Token)
			if !ok {
				return nil
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return nil
			}
			sub, err := claims.GetSubject()
			if err != nil || sub == "" {
				return nil
			}
			parsedUUID, err := uuid.Parse(sub)
			if err != nil {
				return nil
			}
			c.Set("user_id", parsedUUID)
			return nil
		},
	})
}

func GetUserID(c *echo.Context) uuid.UUID {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}
