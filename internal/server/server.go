package server

import (
	"database/sql"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sedobrengocce/TaskMaster/internal/db"
)

type Server struct {
	conn 			*sql.DB
	DB 				*db.Queries
	echo			*echo.Echo
	JWTSecret 		[]byte
	RefreshSecret 	[]byte
}

func NewServer(conn *sql.DB, jwtSecret string, refreshSecret string) *Server {
	return &Server{
		conn: conn,
		DB:   db.New(conn),
		echo: echo.New(),
		JWTSecret: []byte(jwtSecret),
		RefreshSecret: []byte(refreshSecret),
	}
}

func (s *Server) Run() error {
	s.echo.Use(middleware.Logger())  // Logga le richieste HTTP
	s.echo.Use(middleware.Recover()) // Recupera da eventuali panic e li gestisce

	s.RegisterRoutes()

	return s.echo.Start(":3000")
}

