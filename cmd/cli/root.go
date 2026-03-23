package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sedobrengocce/TaskMaster/cmd/cli/commands"
)

var rootCmd = &cobra.Command{
	Use:   "taskmaster",
	Short: "TaskMaster CLI - manage projects and tasks",
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("server", "", "server URL (default: http://localhost:3000)")
	viper.BindPFlag("server_url", rootCmd.PersistentFlags().Lookup("server"))
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	configDir := filepath.Join(home, ".taskmaster")
	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.SetDefault("server_url", "http://localhost:3000")
	viper.SetDefault("week_start", "monday")
	viper.ReadInConfig()
}

func getServerURL() string {
	return viper.GetString("server_url")
}

func getToken() string {
	return viper.GetString("token")
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".taskmaster", "config.yaml")
}

func saveToken(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".taskmaster")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	viper.Set("token", token)
	return viper.WriteConfigAs(configPath())
}

func removeToken() error {
	viper.Set("token", "")
	return viper.WriteConfigAs(configPath())
}

func getUserIDFromToken() (int32, error) {
	token := getToken()
	if token == "" {
		return 0, fmt.Errorf("not logged in, please run: taskmaster login")
	}

	// JWT has 3 segments separated by '.'; decode the payload (middle segment)
	parts := splitJWT(token)
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid token payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, fmt.Errorf("invalid token claims: %w", err)
	}

	sub, ok := claims["sub"]
	if !ok {
		return 0, fmt.Errorf("token missing subject claim")
	}

	switch v := sub.(type) {
	case float64:
		return int32(v), nil
	case string:
		id, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid subject in token: %w", err)
		}
		return int32(id), nil
	default:
		return 0, fmt.Errorf("unexpected subject type in token")
	}
}

func splitJWT(token string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}

func Execute() {
	// Wire helpers into commands package
	commands.GetServerURL = getServerURL
	commands.GetToken = getToken
	commands.SaveToken = saveToken
	commands.RemoveToken = removeToken
	commands.GetUserID = getUserIDFromToken
	commands.RegisterCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
