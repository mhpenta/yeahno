package yeahno_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mhpenta/yeahno"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// TestToToolsBasic tests a simple select menu converted to MCP tools.
func TestToToolsBasic(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Choose Action").
		Description("Select an action to perform").
		Options(
			yeahno.NewOption("Start", "start").MCP(true),
			yeahno.NewOption("Stop", "stop").MCP(true),
			yeahno.NewOption("Restart", "restart").MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return fmt.Sprintf("Action: %s", action), nil
		})

	env := setupMCPServerClient(t, menu)
	defer env.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tool name derived from Key "Start" -> "start"
	result, err := env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "start",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	assertTextContent(t, result, "Action: start")

	// Tool name derived from Key "Stop" -> "stop"
	result, err = env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "stop",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	assertTextContent(t, result, "Action: stop")
}

// TestToToolsWithFields tests a select menu with conditional input fields.
func TestToToolsWithFields(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Installation").
		Options(
			yeahno.NewOption("Quick Install", "quick").
				Description("Install with defaults").
				MCP(true),

			yeahno.NewOption("Custom Install", "custom").
				Description("Specify installation path").
				WithField(yeahno.NewInput().Key("path").Title("Installation path")).
				WithField(yeahno.NewInput().Key("components").Title("Components").Required(false)).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			switch action {
			case "quick":
				return "Installed to /usr/local/bin", nil
			case "custom":
				path := fields["path"]
				components := fields["components"]
				if components == "" {
					return fmt.Sprintf("Installed to %s with all components", path), nil
				}
				return fmt.Sprintf("Installed to %s with: %s", path, components), nil
			}
			return nil, fmt.Errorf("unknown action: %s", action)
		})

	env := setupMCPServerClient(t, menu)
	defer env.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tool name from Key "Quick Install" -> "quick_install"
	result, err := env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "quick_install",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	assertTextContent(t, result, "Installed to /usr/local/bin")

	// Tool name from Key "Custom Install" -> "custom_install"
	result, err = env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "custom_install",
		Arguments: json.RawMessage(`{"path": "/opt/myapp", "components": "core,ui"}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	assertTextContent(t, result, "Installed to /opt/myapp with: core,ui")

	result, err = env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "custom_install",
		Arguments: json.RawMessage(`{"path": "/home/user/app"}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	assertTextContent(t, result, "Installed to /home/user/app with all components")
}

// TestToToolsMCPFiltering tests that non-MCP options are filtered out.
func TestToToolsMCPFiltering(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Actions").
		Options(
			yeahno.NewOption("Public Action", "public").MCP(true),
			yeahno.NewOption("Internal Only", "internal"), // Not exposed to MCP
			yeahno.NewOption("Another Public", "public2").MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return fmt.Sprintf("Executed: %s", action), nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	// Should only have 2 tools
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, td := range tools {
		names[td.Tool.Name] = true
	}

	// Tool names from Keys: "Public Action" -> "public_action", "Another Public" -> "another_public"
	if !names["public_action"] {
		t.Error("Expected 'public_action' tool")
	}
	if !names["another_public"] {
		t.Error("Expected 'another_public' tool")
	}
	if names["internal_only"] {
		t.Error("'internal_only' should not be exposed as a tool")
	}
}

// TestToToolsIntValues tests Select with int type values.
func TestToToolsIntValues(t *testing.T) {
	var choice int

	menu := yeahno.NewSelect[int]().
		Title("Priority").
		Options(
			yeahno.NewOption("Low", 1).MCP(true),
			yeahno.NewOption("Medium", 2).MCP(true),
			yeahno.NewOption("High", 3).MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, priority int, fields map[string]string) (any, error) {
			return map[string]int{"priority": priority}, nil
		})

	env := setupMCPServerClientInt(t, menu)
	defer env.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tool name from Key "Medium" -> "medium"
	result, err := env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "medium",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	assertTextContentContains(t, result, `"priority":2`)
}

// TestToToolsValidation tests that missing required fields return errors.
func TestToToolsValidation(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Config").
		Options(
			yeahno.NewOption("Set Value", "set").
				WithField(yeahno.NewInput().Key("key").Title("Key").Required(true)).
				WithField(yeahno.NewInput().Key("value").Title("Value").Required(true)).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return fmt.Sprintf("Set %s=%s", fields["key"], fields["value"]), nil
		})

	env := setupMCPServerClient(t, menu)
	defer env.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tool name from Key "Set Value" -> "set_value"
	result, err := env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "set_value",
		Arguments: json.RawMessage(`{"key": "foo"}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !result.IsError {
		t.Error("Expected IsError=true for missing required field")
	}
	assertTextContentContains(t, result, "missing required field")

	result, err = env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "set_value",
		Arguments: json.RawMessage(`{"key": "foo", "value": "bar"}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Errorf("Expected success but got error: %v", getTextContent(result))
	}
	assertTextContent(t, result, "Set foo=bar")
}

// TestToToolsNoHandler tests that ToTools returns error without a handler.
func TestToToolsNoHandler(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("No Handler").
		Options(
			yeahno.NewOption("Do Something", "do").MCP(true),
		).
		Value(&choice)

	_, err := menu.ToTools()
	if err == nil {
		t.Error("Expected error for missing handler")
	}
	if err != nil && !containsString(err.Error(), "no handler configured") {
		t.Errorf("Expected 'no handler configured' error, got: %v", err)
	}
}

// TestToToolsHandlerError tests that handler errors are properly returned.
func TestToToolsHandlerError(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Error Test").
		Options(
			yeahno.NewOption("Fail", "fail").MCP(true),
			yeahno.NewOption("Succeed", "succeed").MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			if action == "fail" {
				return nil, fmt.Errorf("intentional failure")
			}
			return "success", nil
		})

	env := setupMCPServerClient(t, menu)
	defer env.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tool name from Key "Fail" -> "fail"
	result, err := env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fail",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !result.IsError {
		t.Error("Expected IsError=true for handler error")
	}
	assertTextContentContains(t, result, "tool execution failed")

	// Tool name from Key "Succeed" -> "succeed"
	result, err = env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "succeed",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Errorf("Expected success but got error: %v", getTextContent(result))
	}
}

// TestToToolsDescription tests that option descriptions are used for tool descriptions.
func TestToToolsDescription(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Described Actions").
		Options(
			yeahno.NewOption("Action One", "one").
				Description("This is the first action").
				MCP(true),
			yeahno.NewOption("Action Two", "two").
				Description("This is the second action").
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	// Tool names from Keys: "Action One" -> "action_one", "Action Two" -> "action_two"
	for _, td := range tools {
		if td.Tool.Name == "action_one" && td.Tool.Description != "This is the first action" {
			t.Errorf("Expected description for 'action_one', got: %s", td.Tool.Description)
		}
		if td.Tool.Name == "action_two" && td.Tool.Description != "This is the second action" {
			t.Errorf("Expected description for 'action_two', got: %s", td.Tool.Description)
		}
	}
}

// TestToToolsFormatValidation tests that format validation works in MCP mode.
func TestToToolsFormatValidation(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Add Website").
		Options(
			yeahno.NewOption("Add", "add").
				WithField(yeahno.NewInput().Key("domain").Title("Domain").Format("domain")).
				WithField(yeahno.NewInput().Key("url").Title("URL").Format("uri")).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return fmt.Sprintf("Added: domain=%s, url=%s", fields["domain"], fields["url"]), nil
		})

	env := setupMCPServerClient(t, menu)
	defer env.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tool name from Key "Add" -> "add"
	result, err := env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add",
		Arguments: json.RawMessage(`{"domain": "example.com", "url": "https://example.com/page"}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Errorf("Expected success but got error: %v", getTextContent(result))
	}
	assertTextContentContains(t, result, "Added:")

	result, err = env.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add",
		Arguments: json.RawMessage(`{"domain": "localhost", "url": "https://example.com"}`),
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !result.IsError {
		t.Error("Expected error for invalid domain (no TLD)")
	}
	assertTextContentContains(t, result, "invalid")
}

// TestToToolsFormatSchema tests that format hints appear in the schema.
func TestToToolsFormatSchema(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Add Website").
		Options(
			yeahno.NewOption("Add", "add").
				WithField(yeahno.NewInput().Key("domain").Title("Domain").Format("domain")).
				WithField(yeahno.NewInput().Key("url").Title("URL").Format("uri")).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return "ok", nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	schema, ok := tools[0].Tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatal("InputSchema is not a map")
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties is not present or wrong type")
	}

	domain, ok := props["domain"].(map[string]any)
	if !ok {
		t.Fatal("domain property not found")
	}
	if domain["format"] != "hostname" {
		t.Errorf("Expected domain format 'hostname', got %v", domain["format"])
	}

	url, ok := props["url"].(map[string]any)
	if !ok {
		t.Fatal("url property not found")
	}
	if url["format"] != "uri" {
		t.Errorf("Expected url format 'uri', got %v", url["format"])
	}
}

// TestValidateFormat tests the ValidateFormat function directly.
func TestValidateFormat(t *testing.T) {
	tests := []struct {
		format  string
		value   string
		wantErr bool
	}{
		{"domain", "example.com", false},
		{"domain", "sub.example.com", false},
		{"domain", "https://example.com", false},
		{"domain", "example.com/path", false},
		{"domain", "localhost", true},
		{"domain", "example .com", true},
		{"domain", "", false},
		{"uri", "https://example.com", false},
		{"uri", "http://example.com/path", false},
		{"uri", "example.com", false},
		{"uri", "", false},
		{"uri", "ftp://evil.com", true},
		{"uri", "javascript:alert(1)", true},
		{"uri", "file:///etc/passwd", true},
		{"domain", "evil.com; rm -rf /", true},
		{"domain", "evil.com\nHost: attacker.com", true},
		{"domain", "../../../etc.passwd", true},
		{"unknown", "anything", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s", tt.format, tt.value), func(t *testing.T) {
			err := yeahno.ValidateFormat(tt.format, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat(%q, %q) error = %v, wantErr %v", tt.format, tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestRegisterTools tests that RegisterTools registers all tools with the server.
func TestRegisterTools(t *testing.T) {
	var choice1, choice2 string

	menu1 := yeahno.NewSelect[string]().
		Title("Menu One").
		ToolPrefix("menu1").
		Options(yeahno.NewOption("Action A", "a").MCP(true)).
		Value(&choice1).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	menu2 := yeahno.NewSelect[string]().
		Title("Menu Two").
		ToolPrefix("menu2").
		Options(yeahno.NewOption("Action B", "b").MCP(true)).
		Value(&choice2).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	if err := menu1.RegisterTools(mcpServer); err != nil {
		t.Fatalf("RegisterTools failed: %v", err)
	}
	if err := menu2.RegisterTools(mcpServer); err != nil {
		t.Fatalf("RegisterTools failed: %v", err)
	}

	ts := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   ts.URL,
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(tools.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools.Tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}

	// Tool names: "Action A" -> "action_a", "Action B" -> "action_b" with prefixes
	if !toolNames["menu1_action_a"] {
		t.Error("Expected 'menu1_action_a' in tool list")
	}
	if !toolNames["menu2_action_b"] {
		t.Error("Expected 'menu2_action_b' in tool list")
	}
}

// TestToToolsCount verifies correct number of tools for MCP-enabled options.
func TestToToolsCount(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Site").
		Options(
			yeahno.NewOption("Add", "add").MCP(true),
			yeahno.NewOption("List", "list").MCP(true),
			yeahno.NewOption("Internal", "internal"), // not exposed
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

// TestToToolsSchema verifies each tool has only its fields, no "action" field.
func TestToToolsSchema(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Site").
		Options(
			yeahno.NewOption("Add", "add").
				WithField(yeahno.NewInput().Key("domain").Title("Domain")).
				MCP(true),
			yeahno.NewOption("List", "list").MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	// Tool names from Keys: "Add" -> "add", "List" -> "list"
	var addTool *yeahno.ToolDef
	var listTool *yeahno.ToolDef
	for i, td := range tools {
		if td.Tool.Name == "add" {
			addTool = &tools[i]
		}
		if td.Tool.Name == "list" {
			listTool = &tools[i]
		}
	}

	if addTool == nil {
		t.Fatal("Expected 'add' tool")
	}
	if listTool == nil {
		t.Fatal("Expected 'list' tool")
	}

	addSchema, ok := addTool.Tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatal("add tool InputSchema is not a map")
	}
	addProps, ok := addSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("add tool properties is not present")
	}
	if _, hasAction := addProps["action"]; hasAction {
		t.Error("add tool should not have 'action' property")
	}
	if _, hasDomain := addProps["domain"]; !hasDomain {
		t.Error("add tool should have 'domain' property")
	}

	listSchema, ok := listTool.Tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatal("list tool InputSchema is not a map")
	}
	listProps, ok := listSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("list tool properties is not present")
	}
	if len(listProps) != 0 {
		t.Errorf("list tool should have empty properties, got %d", len(listProps))
	}
}

// TestToToolsHandlerRouting verifies each handler calls correct action value.
func TestToToolsHandlerRouting(t *testing.T) {
	var choice string
	var calledWith string

	menu := yeahno.NewSelect[string]().
		Title("Site").
		Options(
			yeahno.NewOption("Add", "add").MCP(true),
			yeahno.NewOption("List", "list").MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			calledWith = action
			return fmt.Sprintf("action: %s", action), nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	// Verify handler receives VALUE, not tool name
	for _, td := range tools {
		calledWith = ""
		result, err := td.Handler(context.Background(), &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name:      td.Tool.Name,
				Arguments: json.RawMessage(`{}`),
			},
		})
		if err != nil {
			t.Fatalf("handler failed: %v", err)
		}
		if result.IsError {
			t.Errorf("Expected success for tool %s", td.Tool.Name)
		}
		// Handler should receive the Value ("add", "list"), not the Key/tool name
		if td.Tool.Name == "add" && calledWith != "add" {
			t.Errorf("Expected handler called with 'add', got %q", calledWith)
		}
		if td.Tool.Name == "list" && calledWith != "list" {
			t.Errorf("Expected handler called with 'list', got %q", calledWith)
		}
	}
}

// TestToToolsFieldValidation verifies required fields + format validation per-tool.
func TestToToolsFieldValidation(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Site").
		Options(
			yeahno.NewOption("Add", "add").
				WithField(yeahno.NewInput().Key("domain").Title("Domain").Format("domain")).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return fmt.Sprintf("added: %s", fields["domain"]), nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	handler := tools[0].Handler

	// Tool name from Key "Add" -> "add"
	result, _ := handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "add",
			Arguments: json.RawMessage(`{}`),
		},
	})
	if !result.IsError {
		t.Error("Expected error for missing required field")
	}

	result, _ = handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "add",
			Arguments: json.RawMessage(`{"domain": "localhost"}`),
		},
	})
	if !result.IsError {
		t.Error("Expected error for invalid domain format")
	}

	result, _ = handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "add",
			Arguments: json.RawMessage(`{"domain": "example.com"}`),
		},
	})
	if result.IsError {
		t.Errorf("Expected success for valid domain: %v", getTextContent(result))
	}
}

// TestToToolsToolNameOverride verifies .ToolName("custom") works.
func TestToToolsToolNameOverride(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Site").
		Options(
			yeahno.NewOption("Add Site", "add").ToolName("add_website").MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	// ToolName override takes precedence over Key
	if tools[0].Tool.Name != "add_website" {
		t.Errorf("Expected tool name 'add_website', got '%s'", tools[0].Tool.Name)
	}
}

// TestToToolsPrefix verifies .ToolPrefix("site") prepends to all tools.
func TestToToolsPrefix(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Site").
		ToolPrefix("site").
		Options(
			yeahno.NewOption("Add", "add").MCP(true),
			yeahno.NewOption("List", "list").MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	// Tool names: "Add" -> "site_add", "List" -> "site_list"
	expectedNames := map[string]bool{"site_add": true, "site_list": true}
	for _, td := range tools {
		if !expectedNames[td.Tool.Name] {
			t.Errorf("Unexpected tool name: %s", td.Tool.Name)
		}
	}
}

// TestToToolsRequiredInSchema verifies required fields appear in schema's required array.
func TestToToolsRequiredInSchema(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Site").
		Options(
			yeahno.NewOption("Add", "add").
				WithField(yeahno.NewInput().Key("domain").Title("Domain").Required(true)).
				WithField(yeahno.NewInput().Key("notes").Title("Notes").Required(false)).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	schema, ok := tools[0].Tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatal("InputSchema is not a map")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required is not present or wrong type")
	}

	if len(required) != 1 {
		t.Errorf("Expected 1 required field, got %d", len(required))
	}
	if len(required) > 0 && required[0] != "domain" {
		t.Errorf("Expected 'domain' in required, got %v", required)
	}
}

// TestToToolsAddCompanyMenu tests a real-world-like menu structure using ToTools.
func TestToToolsAddCompanyMenu(t *testing.T) {
	var choice string

	var handlerCalled bool
	var receivedAction string
	var receivedFields map[string]string

	menu := yeahno.NewSelect[string]().
		Title("Add Company").
		Description("Add a new company to crawl and index").
		ToolPrefix("company").
		Options(
			yeahno.NewOption("Add & Monitor", "add_monitor").
				Description("Add company and monitor for future document changes").
				WithField(yeahno.NewInput().Key("domain").Title("Domain (e.g., example.com)").Format("domain")).
				WithField(yeahno.NewInput().Key("company_name").Title("Company Name")).
				WithField(yeahno.NewInput().Key("ticker").Title("Ticker Symbol (optional)").Required(false)).
				MCP(true),

			yeahno.NewOption("Cancel", "cancel"),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			handlerCalled = true
			receivedAction = action
			receivedFields = fields

			switch action {
			case "add_monitor":
				result := map[string]any{
					"status":       "started",
					"domain":       fields["domain"],
					"company_name": fields["company_name"],
				}
				if fields["ticker"] != "" {
					result["ticker"] = fields["ticker"]
				}
				return result, nil
			case "cancel":
				return "Cancelled", nil
			}
			return nil, fmt.Errorf("unknown action: %s", action)
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	td := tools[0]

	// Tool name from Key "Add & Monitor" -> "add_monitor" with prefix -> "company_add_monitor"
	if td.Tool.Name != "company_add_monitor" {
		t.Errorf("Expected tool name 'company_add_monitor', got '%s'", td.Tool.Name)
	}

	handlerCalled = false
	result, err := td.Handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "company_add_monitor",
			Arguments: json.RawMessage(`{"domain": "example.com", "company_name": "Example Corp", "ticker": "EXMP"}`),
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if result.IsError {
		t.Errorf("Expected success, got error: %v", getTextContent(result))
	}
	if !handlerCalled {
		t.Error("Handler was not called")
	}
	// Handler receives VALUE "add_monitor", not tool name
	if receivedAction != "add_monitor" {
		t.Errorf("Expected action 'add_monitor', got '%s'", receivedAction)
	}
	if receivedFields["domain"] != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", receivedFields["domain"])
	}

	result, _ = td.Handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "company_add_monitor",
			Arguments: json.RawMessage(`{"domain": "example.com"}`),
		},
	})
	if !result.IsError {
		t.Error("Expected error for missing company_name")
	}

	result, _ = td.Handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "company_add_monitor",
			Arguments: json.RawMessage(`{"domain": "localhost", "company_name": "Test"}`),
		},
	})
	if !result.IsError {
		t.Error("Expected error for invalid domain (no TLD)")
	}
}

// Helper functions

type testEnv struct {
	server  *httptest.Server
	session *mcp.ClientSession
}

func (e *testEnv) Close() {
	if e.session != nil {
		e.session.Close()
	}
	if e.server != nil {
		e.server.Close()
	}
}

func setupMCPServerClient(t *testing.T, menu *yeahno.Select[string]) *testEnv {
	t.Helper()

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	if err := menu.RegisterTools(mcpServer); err != nil {
		t.Fatalf("RegisterTools failed: %v", err)
	}

	ts := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil))

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   ts.URL,
		MaxRetries: -1,
	}, nil)
	if err != nil {
		ts.Close()
		t.Fatalf("Connect failed: %v", err)
	}

	return &testEnv{server: ts, session: session}
}

func setupMCPServerClientInt(t *testing.T, menu *yeahno.Select[int]) *testEnv {
	t.Helper()

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	if err := menu.RegisterTools(mcpServer); err != nil {
		t.Fatalf("RegisterTools failed: %v", err)
	}

	ts := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil))

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   ts.URL,
		MaxRetries: -1,
	}, nil)
	if err != nil {
		ts.Close()
		t.Fatalf("Connect failed: %v", err)
	}

	return &testEnv{server: ts, session: session}
}

func assertTextContent(t *testing.T, result *mcp.CallToolResult, expected string) {
	t.Helper()
	actual := getTextContent(result)
	if actual != expected {
		t.Errorf("Expected content %q, got %q", expected, actual)
	}
}

func assertTextContentContains(t *testing.T, result *mcp.CallToolResult, substring string) {
	t.Helper()
	actual := getTextContent(result)
	if actual == "" {
		t.Errorf("Expected content containing %q, but got empty content", substring)
		return
	}
	if !containsString(actual, substring) {
		t.Errorf("Expected content to contain %q, got %q", substring, actual)
	}
}

func getTextContent(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestToToolsFieldLengthLimit(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Length Test").
		Options(
			yeahno.NewOption("Echo", "echo").
				WithField(yeahno.NewInput().Key("msg").Title("Message").CharLimit(10)).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return fields["msg"], nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	handler := tools[0].Handler

	result, _ := handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "echo",
			Arguments: json.RawMessage(`{"msg": "short"}`),
		},
	})
	if result.IsError {
		t.Errorf("Expected success for short message: %v", getTextContent(result))
	}

	result, _ = handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "echo",
			Arguments: json.RawMessage(`{"msg": "` + strings.Repeat("a", 11) + `"}`),
		},
	})
	if !result.IsError {
		t.Error("Expected error for message exceeding charLimit")
	}
	assertTextContentContains(t, result, "exceeds maximum length")
}

func TestToToolsErrorSanitization(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Error Sanitize").
		Options(
			yeahno.NewOption("Fail", "fail").MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return nil, fmt.Errorf("secret db password: hunter2 at /internal/path/db.go:42")
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	result, _ := tools[0].Handler(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "fail",
			Arguments: json.RawMessage(`{}`),
		},
	})
	if !result.IsError {
		t.Error("Expected error")
	}
	text := getTextContent(result)
	if strings.Contains(text, "hunter2") || strings.Contains(text, "/internal/path") {
		t.Errorf("Error message leaks internal details: %s", text)
	}
	assertTextContent(t, result, "tool execution failed")
}

func TestToToolsCharLimitInSchema(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Schema Limit").
		Options(
			yeahno.NewOption("Echo", "echo").
				WithField(yeahno.NewInput().Key("msg").Title("Message").CharLimit(50)).
				WithField(yeahno.NewInput().Key("note").Title("Note")).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return fields["msg"], nil
		})

	tools, err := menu.ToTools()
	if err != nil {
		t.Fatalf("ToTools failed: %v", err)
	}

	schema, ok := tools[0].Tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatal("InputSchema is not a map")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not present")
	}

	msg, ok := props["msg"].(map[string]any)
	if !ok {
		t.Fatal("msg property not found")
	}
	if msg["maxLength"] != float64(50) {
		t.Errorf("Expected msg maxLength=50, got %v", msg["maxLength"])
	}

	note, ok := props["note"].(map[string]any)
	if !ok {
		t.Fatal("note property not found")
	}
	if _, hasMax := note["maxLength"]; hasMax {
		t.Error("note should not have maxLength when CharLimit is not set")
	}
}

func TestToSkill(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Site Manager").
		Description("Manage monitored websites").
		ToolPrefix("site").
		Options(
			yeahno.NewOption("Add Site", "add").
				Description("Register a new site for monitoring").
				WithField(yeahno.NewInput().Key("domain").Title("Domain").Format("domain")).
				WithField(yeahno.NewInput().Key("notes").Title("Notes").Required(false)).
				MCP(true),

			yeahno.NewOption("List Sites", "list").
				Description("Show all monitored sites").
				MCP(true),

			yeahno.NewOption("Admin", "admin"),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	sk, err := menu.ToSkill()
	if err != nil {
		t.Fatalf("ToSkill failed: %v", err)
	}

	skill := sk.String()

	checks := []string{
		"---\nname: site\n",
		"description: Manage monitored websites Use when the user needs to:",
		"register a new site for monitoring",
		"show all monitored sites",
		"# Site Manager",
		"### `site_add_site`",
		"Register a new site for monitoring",
		"### `site_list_sites`",
		"| `domain` | yes | domain | Domain |",
		"| `notes` | no | string | Notes |",
		"## Workflow",
		"## Guidelines",
		"`domain` (domain format)",
		"Always provide required fields: `domain`",
	}

	for _, check := range checks {
		if !strings.Contains(skill, check) {
			t.Errorf("Skill missing expected content: %q\n\nGot:\n%s", check, skill)
		}
	}

	if strings.Contains(skill, "admin") || strings.Contains(skill, "Admin") {
		t.Error("Skill should not contain non-MCP options")
	}
}

func TestToSkillNoHandler(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Test").
		Options(yeahno.NewOption("A", "a").MCP(true)).
		Value(&choice)

	_, err := menu.ToSkill()
	if err == nil {
		t.Error("Expected error for missing handler")
	}
}

func TestSkillBuilder(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Task Manager").
		Description("Manage repair tasks").
		ToolPrefix("task").
		Options(
			yeahno.NewOption("Add Task", "add").
				Description("Create a new task").
				WithField(yeahno.NewInput().Key("title").Title("Title").Required(true)).
				WithField(yeahno.NewInput().Key("priority").Title("Priority")).
				MCP(true),
			yeahno.NewOption("List Tasks", "list").
				Description("Show all tasks").
				MCP(true),
			yeahno.NewOption("Complete Task", "complete").
				Description("Mark a task as done").
				WithField(yeahno.NewInput().Key("task_id").Title("Task ID").Required(true)).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	sk, err := menu.ToSkill()
	if err != nil {
		t.Fatalf("ToSkill failed: %v", err)
	}

	t.Run("description override", func(t *testing.T) {
		sk.Description("Full task lifecycle management for repair tickets")
		out := sk.String()
		if !strings.Contains(out, "description: Full task lifecycle management for repair tickets") {
			t.Errorf("Description override not applied:\n%s", out)
		}
		if strings.Contains(out, "description: Manage repair tasks Use when") {
			t.Error("Old auto-generated description should be replaced")
		}
	})

	t.Run("custom workflow", func(t *testing.T) {
		sk.Workflow(
			"List existing tasks with `task_list_tasks`",
			"Add new tasks with `task_add_task`",
			"Complete tasks with `task_complete_task`",
		)
		out := sk.String()
		if !strings.Contains(out, "1. List existing tasks with `task_list_tasks`") {
			t.Errorf("Custom workflow step 1 missing:\n%s", out)
		}
		if !strings.Contains(out, "3. Complete tasks with `task_complete_task`") {
			t.Errorf("Custom workflow step 3 missing:\n%s", out)
		}
		if strings.Contains(out, "**Create a new task** — `task_add_task`") {
			t.Error("Default workflow should be replaced when custom workflow is set")
		}
	})

	t.Run("guideline append", func(t *testing.T) {
		sk.Guideline("Priority values: low, normal, high, urgent")
		out := sk.String()
		if !strings.Contains(out, "- Priority values: low, normal, high, urgent") {
			t.Errorf("Custom guideline missing:\n%s", out)
		}
		if !strings.Contains(out, "Always provide required fields:") {
			t.Error("Default guidelines should be preserved")
		}
	})

	t.Run("custom section", func(t *testing.T) {
		sk.Section("Error Handling", "If task ID is not found, call task_list_tasks first.")
		out := sk.String()
		if !strings.Contains(out, "## Error Handling") {
			t.Errorf("Custom section heading missing:\n%s", out)
		}
		if !strings.Contains(out, "If task ID is not found, call task_list_tasks first.") {
			t.Errorf("Custom section body missing:\n%s", out)
		}
	})

	t.Run("chaining returns self", func(t *testing.T) {
		sk2, _ := menu.ToSkill()
		result := sk2.
			Description("test").
			Workflow("step one").
			Guideline("rule one").
			Section("Extra", "content")
		if result != sk2 {
			t.Error("Chainable methods should return same pointer")
		}
	})
}

func TestAttachAgentSkillMd(t *testing.T) {
	var choice string

	menu := yeahno.NewSelect[string]().
		Title("Site Manager").
		Description("Manage monitored websites").
		ToolPrefix("site").
		Options(
			yeahno.NewOption("Add Site", "add").
				Description("Register a new site for monitoring").
				WithField(yeahno.NewInput().Key("domain").Title("Domain").Format("domain")).
				MCP(true),
			yeahno.NewOption("List Sites", "list").
				Description("Show all monitored sites").
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			return action, nil
		})

	sk, err := menu.ToSkill()
	if err != nil {
		t.Fatalf("ToSkill failed: %v", err)
	}

	rootCmd := &cobra.Command{
		Use:   "site",
		Short: "Site manager",
	}
	sk.Attach(rootCmd)

	t.Run("flag is registered", func(t *testing.T) {
		f := rootCmd.PersistentFlags().Lookup("agent-skill-md")
		if f == nil {
			t.Fatal("--agent-skill-md flag not registered")
		}
		if f.Usage != "Print agent skill definition (SKILL.md) to stdout" {
			t.Errorf("Unexpected flag usage: %s", f.Usage)
		}
	})

	t.Run("without flag runs normally", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:   "site",
			Short: "Site manager",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprint(cmd.OutOrStdout(), "normal output")
				return nil
			},
		}
		sk.Attach(cmd)

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "normal output") {
			t.Errorf("Expected normal output, got: %s", buf.String())
		}
	})

	t.Run("attach returns self for chaining", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		result := sk.Attach(cmd)
		if result != sk {
			t.Error("Attach should return same pointer")
		}
	})
}
