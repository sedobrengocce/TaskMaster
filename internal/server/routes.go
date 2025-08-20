package server

func (s *Server)RegisterRoutes() {
	api := s.echo.Group("/api/")

	// GET
	api.GET("/healthcheck", s.HealthCheckHandler)

	// POST
	api.POST("/register", s.RegisterUserHandler)

	// PUT

	// DELETE

}
