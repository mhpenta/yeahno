package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/mhpenta/yeahno"
	"github.com/spf13/cobra"
)

// Demonstrates a menu with handler and MCP-enabled options.
//
// Run CLI:
//
//	go run ./examples/repair list-tasks
//	go run ./examples/repair add-task --title "Fix bug"
//	go run ./examples/repair complete-task --id 123
//
// Print agent skill definition:
//
//	go run ./examples/repair --agent-skill-md
//
// Run TUI:
//
//	go run ./examples/repair tui
func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	menu := buildMenu()

	rootCmd := &cobra.Command{
		Use:   "repair",
		Short: "Task management tool",
		Long:  "A task manager demonstrating yeahno TUI, CLI, and MCP generation.",
	}

	// TUI mode
	rootCmd.AddCommand(&cobra.Command{
		Use:   "tui",
		Short: "Run interactive TUI mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := menu.Run(context.Background())
			if err != nil {
				return err
			}
			fmt.Printf("Result: %v\n", result)
			return nil
		},
	})

	// Register CLI commands from menu
	if err := menu.RegisterCLI(rootCmd); err != nil {
		return err
	}

	skill, err := menu.ToSkill()
	if err != nil {
		return err
	}
	skill.
		Workflow(
			"List existing tasks with `task_list-tasks` to show current state",
			"Add new tasks with `task_add-task` — priority defaults to normal",
			"Complete tasks with `task_complete-task` using the task ID from the list",
		).
		Guideline("Priority values: low, normal, high, urgent").
		Section("Error Handling", "If a task ID is not found, call task_list-tasks first to get valid IDs.").
		Attach(rootCmd)

	theme := yeahno.DefaultTheme()
	return fang.Execute(context.Background(), rootCmd,
		fang.WithVersion("1.0.0"),
		fang.WithColorSchemeFunc(theme.FangColorScheme()),
	)
}

func buildMenu() *yeahno.Select[string] {
	var choice string

	return yeahno.NewSelect[string]().
		Title("Task Manager").
		Description("Select an action").
		ToolPrefix("task").
		Options(
			yeahno.NewOption("List tasks", "list").
				ToolName("list-tasks").
				Description("Show all tasks").
				MCP(true),

			yeahno.NewOption("Add task", "add").
				ToolName("add-task").
				Description("Create a new task").
				WithField(yeahno.NewInput().Key("title").Title("Task title")).
				WithField(yeahno.NewInput().Key("priority").Title("Priority").Required(false)).
				MCP(true),

			yeahno.NewOption("Complete task", "complete").
				ToolName("complete-task").
				Description("Mark task as done").
				WithField(yeahno.NewInput().Key("id").Title("Task ID")).
				MCP(true),

			yeahno.NewOption("Settings", "settings").
				Description("TUI only"),

			yeahno.NewOption("Exit", "exit"),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			switch action {
			case "list":
				return []string{"Buy milk", "Fix bug", "Write docs"}, nil
			case "add":
				return fmt.Sprintf("Added: %s", fields["title"]), nil
			case "complete":
				return fmt.Sprintf("Completed task %s", fields["id"]), nil
			case "settings":
				return "Opening settings...", nil
			case "exit":
				return "Goodbye!", nil
			}
			return nil, fmt.Errorf("unknown: %s", action)
		})
}
