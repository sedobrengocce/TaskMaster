package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to TaskMaster",
	RunE: func(cmd *cobra.Command, args []string) error {
		var email string
		fmt.Print("Email: ")
		fmt.Fscanln(os.Stdin, &email)

		fmt.Print("Password: ")
		passBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password := string(passBytes)

		api := newClient()
		body, err := api.Post("/api/login", map[string]string{
			"email":    email,
			"password": password,
		})
		if err != nil {
			return err
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("unexpected response: %w", err)
		}

		token, ok := resp["jwt"].(string)
		if !ok || token == "" {
			return fmt.Errorf("no token in response")
		}

		if err := saveTokenFn(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println("Login successful!")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from TaskMaster",
	RunE: func(cmd *cobra.Command, args []string) error {
		api := newAuthClient()
		_, err := api.Post("/api/logout", nil)
		if err != nil {
			// Remove token locally even on error
			removeTokenFn()
			fmt.Println("Logged out (local token cleared).")
			return nil
		}

		removeTokenFn()
		fmt.Println("Logged out successfully.")
		return nil
	},
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new account",
	RunE: func(cmd *cobra.Command, args []string) error {
		var email string
		fmt.Print("Email: ")
		fmt.Fscanln(os.Stdin, &email)

		fmt.Print("Password: ")
		passBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password := string(passBytes)

		if len(password) < 8 {
			return fmt.Errorf("password must be at least 8 characters")
		}

		api := newClient()
		_, err = api.Post("/api/register", map[string]string{
			"email":    email,
			"password": password,
		})
		if err != nil {
			return err
		}

		fmt.Println("Registration successful! Run: taskmaster login")
		return nil
	},
}

