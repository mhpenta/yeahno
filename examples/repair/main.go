package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mhpenta/yeahno"
)

// Demonstrates a menu with handler and MCP-enabled options.
// Run with: go run ./examples/repair
func main() {
	menu := buildMenu()

	result, err := menu.Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Result: %v\n", result)
}

func buildMenu() *yeahno.Select[string] {
	var choice string

	return yeahno.NewSelect[string]().
		Title("Task Manager").
		Description("Select an action").
		ToolPrefix("task").
		Options(
			yeahno.NewOption("List tasks", "list").
				Description("Show all tasks").
				MCP(true),

			yeahno.NewOption("Add task", "add").
				Description("Create a new task").
				WithField(yeahno.NewInput().Key("title").Title("Task title")).
				WithField(yeahno.NewInput().Key("priority").Title("Priority").Required(false)).
				MCP(true),

			yeahno.NewOption("Complete task", "complete").
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
