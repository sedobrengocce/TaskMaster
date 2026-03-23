package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type WeeklyViewResponse struct {
	WeekStart string       `json:"week_start"`
	WeekEnd   string       `json:"week_end"`
	Tasks     []WeeklyTask `json:"tasks"`
}

type WeeklyTask struct {
	ID       int32          `json:"id"`
	Title    string         `json:"title"`
	TaskType string         `json:"task_type"`
	Days     []WeeklyTaskDay `json:"days"`
}

type WeeklyTaskDay struct {
	Date      string `json:"date"`
	Weekday   string `json:"weekday"`
	Completed bool   `json:"completed"`
	Scheduled bool   `json:"scheduled"`
}

var weeklyCmd = &cobra.Command{
	Use:   "weekly",
	Short: "Show weekly task view",
	RunE: func(cmd *cobra.Command, args []string) error {
		week, _ := cmd.Flags().GetString("week")

		api := newAuthClient()
		path := "/api/weekly"
		if week != "" {
			path += "?week=" + week
		}

		body, err := api.Get(path)
		if err != nil {
			return err
		}

		var resp WeeklyViewResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Week: %s → %s\n\n", resp.WeekStart, resp.WeekEnd)

		if len(resp.Tasks) == 0 {
			fmt.Println("No tasks for this week.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "Task\tMon\tTue\tWed\tThu\tFri\tSat\tSun\n")

		for _, task := range resp.Tasks {
			fmt.Fprintf(w, "%s", task.Title)
			for _, day := range task.Days {
				if day.Completed {
					fmt.Fprintf(w, "\t ✓ ")
				} else if day.Scheduled {
					fmt.Fprintf(w, "\t x ")
				} else {
					fmt.Fprintf(w, "\t   ")
				}
			}
			fmt.Fprintln(w)
		}
		w.Flush()
		return nil
	},
}

func init() {
	weeklyCmd.Flags().String("week", "", "week start date (YYYY-MM-DD), defaults to current week")
}
