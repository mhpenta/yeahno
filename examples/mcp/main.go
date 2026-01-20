package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/mhpenta/yeahno"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Demonstrates MCP server integration with yeahno menus.
// Run with: go run ./examples/mcp
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build menu and generate tools
	menu := buildMenu()
	tools, err := menu.ToTools()
	if err != nil {
		return err
	}

	fmt.Printf("Generated %d tools:\n", len(tools))
	for _, td := range tools {
		fmt.Printf("  - %s\n", td.Tool.Name)
	}
	fmt.Println()

	// Start MCP server
	server := mcp.NewServer(&mcp.Implementation{Name: "demo", Version: "1.0"}, nil)
	if err := menu.RegisterTools(server); err != nil {
		return err
	}

	ts := httptest.NewServer(mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return server }, nil))
	defer ts.Close()

	// Connect client
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: ts.URL, MaxRetries: -1,
	}, nil)
	if err != nil {
		return err
	}
	defer session.Close()

	// Test tool calls
	tests := []struct {
		tool string
		args map[string]any
	}{
		{"note_list_notes", nil},
		{"note_add_note", map[string]any{"title": "Test note", "body": "Hello world"}},
		{"note_get_note", map[string]any{"id": "123"}},
	}

	for _, tc := range tests {
		argsJSON, _ := json.Marshal(tc.args)
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      tc.tool,
			Arguments: json.RawMessage(argsJSON),
		})
		if err != nil {
			fmt.Printf("%s: error %v\n", tc.tool, err)
			continue
		}
		if content, ok := result.Content[0].(*mcp.TextContent); ok {
			fmt.Printf("%s: %s\n", tc.tool, content.Text)
		}
	}

	return nil
}

func buildMenu() *yeahno.Select[string] {
	var choice string

	return yeahno.NewSelect[string]().
		Title("Notes").
		ToolPrefix("note").
		Options(
			yeahno.NewOption("List notes", "list").MCP(true),

			yeahno.NewOption("Add note", "add").
				WithField(yeahno.NewInput().Key("title").Title("Title")).
				WithField(yeahno.NewInput().Key("body").Title("Body").Required(false)).
				MCP(true),

			yeahno.NewOption("Get note", "get").
				WithField(yeahno.NewInput().Key("id").Title("Note ID")).
				MCP(true),

			yeahno.NewOption("Exit", "exit"),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			switch action {
			case "list":
				return []string{"Meeting notes", "Shopping list"}, nil
			case "add":
				return fmt.Sprintf("Created: %s", fields["title"]), nil
			case "get":
				return map[string]string{"id": fields["id"], "title": "Sample", "body": "Content"}, nil
			case "exit":
				return "Goodbye!", nil
			}
			return nil, fmt.Errorf("unknown: %s", action)
		})
}
