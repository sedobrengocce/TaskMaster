package server

func (s *Server)RegisterRoutes() {
	api := s.echo.Group("/api")

	// GET
	s.echo.GET("/healthcheck", s.HealthCheckHandler)

	// POST
	api.POST("/register", s.RegisterUserHandler)
	api.POST("/login", s.LoginUserHandler)
	api.POST("/refresh_token", s.RefreshTokenHandler)
	api.POST("/logout", s.LogoutUserHandler)

	// PUT

	// DELETE

}
