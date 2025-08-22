package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func AuthMiddleware(redisClient *redis.Client, jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(401, map[string]string{"error": "Missing Authorization header"})
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			exists, err := redisClient.Exists(context.Background(), tokenString).Result()
			if err != nil || exists == 1 {
				return c.JSON(http.StatusUnauthorized, "Invalid token")
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, "Invalid token")
			}

			return next(c)
		}
	}
}		

