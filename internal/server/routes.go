package server

func (s *Server)RegisterRoutes() {
	api := s.echo.Group("/api")

	// GET
	s.echo.GET("/healthcheck", s.HealthCheckHandler)

	// POST
	api.POST("/register", s.RegisterUserHandler)

	// PUT

	// DELETE

}
