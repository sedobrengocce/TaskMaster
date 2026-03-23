package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Task struct {
	ID              int32  `json:"ID"`
	ProjectID       *int32 `json:"ProjectID"`
	Title           string `json:"Title"`
	Description     string `json:"Description"`
	TaskType        string `json:"TaskType"`
	Priority        *int32 `json:"Priority"`
	CreatedByUserID int32  `json:"CreatedByUserID"`
	CreatedAt       string `json:"CreatedAt"`
}

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Manage tasks",
}

var tasksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetInt32("project")

		api := newAuthClient()
		var body []byte
		var err error

		priority, _ := cmd.Flags().GetInt32("priority")

		showAll, _ := cmd.Flags().GetBool("all")

		if projectID > 0 {
			body, err = api.Get(fmt.Sprintf("/api/projects/%d/tasks", projectID))
		} else {
			userID, uerr := getUserIDFn()
			if uerr != nil {
				return uerr
			}
			url := fmt.Sprintf("/api/tasks?user_id=%d", userID)
			if !showAll {
				url += "&exclude_scheduled=true"
			}
			if cmd.Flags().Changed("priority") {
				url += fmt.Sprintf("&priority=%d", priority)
			}
			body, err = api.Get(url)
		}
		if err != nil {
			return err
		}

		var tasks []Task
		if err := json.Unmarshal(body, &tasks); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tTYPE\tPRIORITY\tPROJECT")
		for _, t := range tasks {
			priority := "-"
			if t.Priority != nil {
				priority = fmt.Sprintf("%d", *t.Priority)
			}
			projectStr := "-"
			if t.ProjectID != nil {
				projectStr = fmt.Sprintf("%d", *t.ProjectID)
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", t.ID, t.Title, t.TaskType, priority, projectStr)
		}
		w.Flush()
		return nil
	},
}

var tasksCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := getUserIDFn()
		if err != nil {
			return err
		}

		taskType, _ := cmd.Flags().GetString("type")
		projectID, _ := cmd.Flags().GetInt32("project")
		priority, _ := cmd.Flags().GetInt32("priority")

		payload := map[string]interface{}{
			"title":              args[0],
			"task_type":          taskType,
			"user_id": userID,
		}
		if projectID > 0 {
			payload["project_id"] = projectID
		}
		if cmd.Flags().Changed("priority") {
			payload["priority"] = priority
		}

		api := newAuthClient()
		body, err := api.Post("/api/tasks", payload)
		if err != nil {
			return err
		}

		var task Task
		if err := json.Unmarshal(body, &task); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Task created (ID: %d)\n", task.ID)
		return nil
	},
}

var tasksCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Mark a task as complete",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := getUserIDFn()
		if err != nil {
			return err
		}

		api := newAuthClient()
		_, err = api.Post("/api/tasks/"+args[0]+"/complete", map[string]interface{}{
			"user_id": userID,
		})
		if err != nil {
			return err
		}
		fmt.Println("Task marked as complete.")
		return nil
	},
}

var tasksUncompleteCmd = &cobra.Command{
	Use:   "uncomplete <id>",
	Short: "Mark a task as not complete",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := getUserIDFn()
		if err != nil {
			return err
		}

		api := newAuthClient()
		_, err = api.Delete(fmt.Sprintf("/api/tasks/%s/complete?user_id=%d", args[0], userID))
		if err != nil {
			return err
		}
		fmt.Println("Task marked as not complete.")
		return nil
	},
}

var tasksDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := newAuthClient()
		_, err := api.Delete("/api/tasks/" + args[0])
		if err != nil {
			return err
		}
		fmt.Println("Task deleted.")
		return nil
	},
}

var tasksSetProjectCmd = &cobra.Command{
	Use:   "set-project <task_id> <project_id>",
	Short: "Assign a task to a project",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		projectID, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}
		pid := int32(projectID)

		api := newAuthClient()
		_, err = api.Put("/api/tasks/"+taskID+"/project", map[string]interface{}{
			"project_id": pid,
		})
		if err != nil {
			return err
		}
		fmt.Println("Task assigned to project.")
		return nil
	},
}

var tasksUnsetProjectCmd = &cobra.Command{
	Use:   "unset-project <task_id>",
	Short: "Remove a task from its project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := newAuthClient()
		_, err := api.Put("/api/tasks/"+args[0]+"/project", map[string]interface{}{
			"project_id": nil,
		})
		if err != nil {
			return err
		}
		fmt.Println("Task removed from project.")
		return nil
	},
}

var tasksScheduleCmd = &cobra.Command{
	Use:   "schedule <task_id> <weekday 1-7 | now>",
	Short: "Schedule a task for a weekday (1-7) or today (now)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var date string
		if strings.ToLower(args[1]) == "now" {
			date = time.Now().Format("2006-01-02")
		} else {
			weekdayNum, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid weekday number or 'now': %w", err)
			}
			date, err = nextDateForWeekday(weekdayNum)
			if err != nil {
				return err
			}
		}

		api := newAuthClient()
		_, err := api.Post("/api/tasks/"+args[0]+"/schedule", map[string]interface{}{
			"date": date,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Task scheduled for %s.\n", date)
		return nil
	},
}

var tasksUnscheduleCmd = &cobra.Command{
	Use:   "unschedule <task_id> <weekday 1-7>",
	Short: "Unschedule a task from the next occurrence of a weekday (1-7)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		weekdayNum, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid weekday number: %w", err)
		}
		date, err := nextDateForWeekday(weekdayNum)
		if err != nil {
			return err
		}

		api := newAuthClient()
		_, err = api.DeleteWithBody("/api/tasks/"+args[0]+"/schedule", map[string]interface{}{
			"date": date,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Task unscheduled from %s.\n", date)
		return nil
	},
}

func nextDateForWeekday(weekdayNum int) (string, error) {
	if weekdayNum < 1 || weekdayNum > 7 {
		return "", fmt.Errorf("weekday must be between 1 and 7")
	}

	weekStart := viper.GetString("week_start")
	var startDay time.Weekday
	if strings.ToLower(weekStart) == "sunday" {
		startDay = time.Sunday
	} else {
		startDay = time.Monday
	}

	targetWeekday := time.Weekday((int(startDay) + weekdayNum - 1) % 7)

	today := time.Now()
	daysAhead := (int(targetWeekday) - int(today.Weekday()) + 7) % 7
	if daysAhead == 0 {
		daysAhead = 7
	}
	target := today.AddDate(0, 0, daysAhead)
	return target.Format("2006-01-02"), nil
}

func init() {
	tasksListCmd.Flags().Int32("project", 0, "filter by project ID")
	tasksListCmd.Flags().Int32("priority", 0, "filter by priority")
	tasksListCmd.Flags().Bool("all", false, "show all tasks including scheduled ones")

	tasksCreateCmd.Flags().String("type", "single", "task type (single or repetitive)")
	tasksCreateCmd.Flags().Int32("project", 0, "project ID")
	tasksCreateCmd.Flags().Int32("priority", 0, "task priority")

	tasksCmd.AddCommand(tasksListCmd)
	tasksCmd.AddCommand(tasksCreateCmd)
	tasksCmd.AddCommand(tasksCompleteCmd)
	tasksCmd.AddCommand(tasksUncompleteCmd)
	tasksCmd.AddCommand(tasksDeleteCmd)
	tasksCmd.AddCommand(tasksSetProjectCmd)
	tasksCmd.AddCommand(tasksUnsetProjectCmd)
	tasksCmd.AddCommand(tasksScheduleCmd)
	tasksCmd.AddCommand(tasksUnscheduleCmd)
}
