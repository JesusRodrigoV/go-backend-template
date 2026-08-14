package guards

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func JWTGuard(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Falta el token de autenticación",
				})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Formato inválido. Use 'Bearer <token>'",
				})
			}
			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "Método de firma inválido")
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Token inválido o expirado",
				})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if ok && token.Valid {
				if userID, ok := claims["user_id"].(string); ok {
					c.Set("user_id", userID)
				} else {
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "Token corrupto (sin user_id)",
					})
				}
			}

			return next(c)
		}
	}
}
