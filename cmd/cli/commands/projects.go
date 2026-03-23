package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type Project struct {
	ID        int32  `json:"ID"`
	Name      string `json:"Name"`
	ColorHex  string `json:"ColorHex"`
	UserID    int32  `json:"UserID"`
	CreatedAt string `json:"CreatedAt"`
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := getUserIDFn()
		if err != nil {
			return err
		}

		api := newAuthClient()
		body, err := api.Get(fmt.Sprintf("/api/projects?user_id=%d", userID))
		if err != nil {
			return err
		}

		var projects []Project
		if err := json.Unmarshal(body, &projects); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCOLOR\tCREATED")
		for _, p := range projects {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.ID, p.Name, p.ColorHex, p.CreatedAt)
		}
		w.Flush()
		return nil
	},
}

var projectsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := getUserIDFn()
		if err != nil {
			return err
		}

		color, _ := cmd.Flags().GetString("color")

		api := newAuthClient()
		payload := map[string]interface{}{
			"name":    args[0],
			"user_id": userID,
		}
		if color != "" {
			payload["color_hex"] = color
		}
		body, err := api.Post("/api/projects", payload)
		if err != nil {
			return err
		}

		var project Project
		if err := json.Unmarshal(body, &project); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Project created (ID: %d)\n", project.ID)
		return nil
	},
}

var projectsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := newAuthClient()
		_, err := api.Delete("/api/projects/" + args[0])
		if err != nil {
			return err
		}
		fmt.Println("Project deleted.")
		return nil
	},
}

var projectsShareCmd = &cobra.Command{
	Use:   "share <project_id> <user_id>",
	Short: "Share a project with a user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			return fmt.Errorf("invalid user_id: %w", err)
		}

		api := newAuthClient()
		_, err = api.Post("/api/projects/"+args[0]+"/share", map[string]interface{}{
			"id": int32(userID),
		})
		if err != nil {
			return err
		}
		fmt.Println("Project shared.")
		return nil
	},
}

var projectsUnshareCmd = &cobra.Command{
	Use:   "unshare <project_id> <user_id>",
	Short: "Stop sharing a project with a user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			return fmt.Errorf("invalid user_id: %w", err)
		}

		api := newAuthClient()
		_, err = api.DeleteWithBody("/api/projects/"+args[0]+"/share", map[string]interface{}{
			"id": int32(userID),
		})
		if err != nil {
			return err
		}
		fmt.Println("Project unshared.")
		return nil
	},
}

func init() {
	projectsCreateCmd.Flags().String("color", "", "project color hex (e.g. #FF0000)")

	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsCreateCmd)
	projectsCmd.AddCommand(projectsDeleteCmd)
	projectsCmd.AddCommand(projectsShareCmd)
	projectsCmd.AddCommand(projectsUnshareCmd)
}
