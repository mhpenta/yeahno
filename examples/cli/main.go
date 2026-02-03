package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/mhpenta/yeahno"
	"github.com/spf13/cobra"
)

// Demonstrates CLI generation from yeahno menus.
//
// Run with:
//
//	go run ./examples/cli list-tasks
//	go run ./examples/cli add-task --title "Buy milk" --priority high
//	go run ./examples/cli complete-task --id 123
//
// Or run TUI mode:
//
//	go run ./examples/cli tui
func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	// Build the shared menu definition
	menu := buildMenu()

	// Create root CLI command
	rootCmd := &cobra.Command{
		Use:   "tasks",
		Short: "Task management CLI",
		Long:  "A task manager demonstrating yeahno CLI generation.",
	}

	// Add TUI subcommand for interactive mode
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

	// Register CLI commands from the same menu
	if err := menu.RegisterCLI(rootCmd); err != nil {
		return err
	}

	// Use yeahno's default theme (or customize with .WithPrimary(), etc.)
	theme := yeahno.DefaultTheme()

	// Use fang with yeahno theme
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
				Description("TUI only - configure preferences"),

			yeahno.NewOption("Exit", "exit"),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			switch action {
			case "list":
				return []string{"Buy milk", "Fix bug", "Write docs"}, nil
			case "add":
				priority := fields["priority"]
				if priority == "" {
					priority = "normal"
				}
				return fmt.Sprintf("Added task: %s (priority: %s)", fields["title"], priority), nil
			case "complete":
				return fmt.Sprintf("Completed task #%s", fields["id"]), nil
			case "settings":
				return "Opening settings...", nil
			case "exit":
				return "Goodbye!", nil
			}
			return nil, fmt.Errorf("unknown action: %s", action)
		})
}
