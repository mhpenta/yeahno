package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mhpenta/yeahno"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	menu := buildMenu()

	mux := http.NewServeMux()
	if err := menu.RegisterHTTP(mux); err != nil {
		return err
	}

	addr := ":8080"
	fmt.Println(banner(addr))
	return http.ListenAndServe(addr, mux)
}

func buildMenu() *yeahno.Select[string] {
	var choice string

	return yeahno.NewSelect[string]().
		Title("Notes").
		Description("Manage notes with create, read, and list operations").
		ToolPrefix("note").
		Options(
			yeahno.NewOption("List notes", "list").
				Description("List all notes").
				MCP(true),

			yeahno.NewOption("Add note", "add").
				Description("Create a new note").
				WithField(yeahno.NewInput().Key("title").Title("Title")).
				WithField(yeahno.NewInput().Key("body").Title("Body").Required(false)).
				MCP(true),

			yeahno.NewOption("Get note", "get").
				Description("Retrieve a note by ID").
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

func banner(addr string) string {
	var b strings.Builder
	b.WriteString("yeahno TAP server on " + addr + "\n\n")
	b.WriteString("  GET  /tools              → index\n")
	b.WriteString("  GET  /tools/{name}       → documentation\n")
	b.WriteString("  POST /tools/{name}/run   → invoke\n")
	b.WriteString("\nTry:\n")
	b.WriteString("  curl localhost:8080/tools\n")
	b.WriteString("  curl localhost:8080/tools/note_add\n")
	b.WriteString("  curl -X POST localhost:8080/tools/note_add/run -d '{\"title\":\"hello\"}'")
	return b.String()
}
