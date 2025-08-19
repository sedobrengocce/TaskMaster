package server

import "github.com/labstack/echo/v4"

// RegisterRoutes registra tutte le route per l'applicazione.
func RegisterRoutes(e *echo.Echo) {
	// Definiamo una route GET per /healthcheck che userà HealthCheckHandler
	e.GET("/healthcheck", HealthCheckHandler)

	// ...qui verranno aggiunte le altre route in futuro
}
