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

	// Project Sharing
	apiPrivate.POST("/projects/:id/share", s.SharedProjectHandler)
	apiPrivate.DELETE("/projects/:id/share", s.UnshareProjectHandler)

	// Tasks
	apiPrivate.POST("/tasks", s.CreateTaskHandler)
	apiPrivate.GET("/tasks", s.ListTasksByUserHandler)
	apiPrivate.GET("/projects/:id/tasks", s.ListTasksByProjectHandler)
	apiPrivate.PUT("/tasks/:id", s.UpdateTaskHandler)
	apiPrivate.DELETE("/tasks/:id", s.DeleteTaskHandler)

	// Task Sharing
	apiPrivate.POST("/tasks/:id/share", s.ShareTaskHandler)
	apiPrivate.DELETE("/tasks/:id/share", s.UnshareTaskHandler)

	// Task Completions
	apiPrivate.POST("/tasks/:id/complete", s.CompleteTaskHandler)
	apiPrivate.DELETE("/tasks/:id/complete", s.UncompleteTaskHandler)
	apiPrivate.GET("/tasks/:id/completions", s.GetTaskCompletionsHandler)
}
