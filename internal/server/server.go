package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Run avvia il server web.
func Run() error {
	// Crea una nuova istanza di Echo
	e := echo.New()

	// Middleware di base
	e.Use(middleware.Logger())  // Logga le richieste HTTP
	e.Use(middleware.Recover()) // Recupera da eventuali panic e li gestisce

	// Registra le route dell'applicazione
	RegisterRoutes(e)

	// Avvia il server sulla porta 8080
	// Questo corrisponde alla porta esposta nel Dockerfile
	return e.Start(":3000")
}
