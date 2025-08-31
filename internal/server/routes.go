package server

func (s *Server) RegisterRoutes() {
	api := s.echo.Group("/api")
	apiPrivate := s.echo.Group("/api")

	apiPrivate.Use(AuthMiddleware(s.Redis, string(s.JWTSecret)))

	// GET
	s.echo.GET("/healthcheck", s.HealthCheckHandler)
	apiPrivate.GET("/projects", s.ListProjects)
	apiPrivate.GET("/projects/:id", s.GetProject)

	// POST
	api.POST("/register", s.RegisterUserHandler)
	api.POST("/login", s.LoginUserHandler)
	api.POST("/refresh_token", s.RefreshTokenHandler)
	api.POST("/logout", s.LogoutUserHandler)
	apiPrivate.POST("/projects", s.CreateProject)

	// PUT
	apiPrivate.PUT("/projects/:id", s.UpdateProject)

	// DELETE
	apiPrivate.DELETE("/projects/:id", s.DeleteProject)

}
