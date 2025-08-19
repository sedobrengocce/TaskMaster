package server

import (
	"net/http"
	"github.com/labstack/echo/v4"
)

// HealthCheckHandler gestisce la richiesta per l'endpoint di health check.
func HealthCheckHandler(c echo.Context) error {
	// Definiamo una struttura semplice per la risposta JSON
	response := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Status:  "ok",
		Message: "Server is up and running",
	}

	// Restituiamo una risposta JSON con codice di stato 200 OK
	return c.JSON(http.StatusOK, response)
}
