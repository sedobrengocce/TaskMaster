package commands

import (
	"github.com/spf13/cobra"

	"github.com/sedobrengocce/TaskMaster/cmd/cli/client"
)

// Function vars set by main to access root-level helpers.
var (
	GetServerURL func() string
	GetToken     func() string
	SaveToken    func(string) error
	RemoveToken  func() error
	GetUserID    func() (int32, error)
)

// Convenience aliases used by command files.
var (
	saveTokenFn   = func(t string) error { return SaveToken(t) }
	removeTokenFn = func() error { return RemoveToken() }
	getUserIDFn   = func() (int32, error) { return GetUserID() }
)

func newClient() *client.APIClient {
	return client.New(GetServerURL(), "")
}

func newAuthClient() *client.APIClient {
	return client.New(GetServerURL(), GetToken())
}

// RegisterCommands adds all subcommands to the root command.
func RegisterCommands(rootCmd *cobra.Command) {
	// auth
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(registerCmd)

	// projects
	rootCmd.AddCommand(projectsCmd)

	// tasks
	rootCmd.AddCommand(tasksCmd)

	// weekly
	rootCmd.AddCommand(weeklyCmd)
}
