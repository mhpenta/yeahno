# yeahno

<p>
<img width="525" alt="yeahno Logo" src="https://github-production-user-asset-6210df.s3.amazonaws.com/183146177/538075071-1aeec958-e8cd-4376-a876-21e161af3023.jpg">
</p>

> [!WARNING]
> This project is experimental. API may change.

Define a form once. Run it as a TUI or expose it as MCP tools. For administrative bots and humans alike!

Built on [huh](https://github.com/charmbracelet/huh) and [Go's official MCP SDK](https://github.com/modelcontextprotocol/go-sdk).

## Install

```bash
go get github.com/mhpenta/yeahno
```

## Usage

```go
var choice string

menu := yeahno.NewSelect[string]().
    Title("Site Manager").
    ToolPrefix("site").  // Tools will be "site_add", "site_list"
    Options(
        yeahno.NewOption("Add", "add").
            Description("Add a new site").
            WithField(yeahno.NewInput().Key("domain").Format("domain")).
            MCP(true),

        yeahno.NewOption("List", "list").
            Description("List all sites").
            MCP(true),

        yeahno.NewOption("Admin", "admin"),  // TUI only
    ).
    Value(&choice).
    Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
        // Handle both TUI and MCP calls here
        return nil, nil
    })

// Run as TUI
result, err := menu.Run(ctx)

// Or register as MCP tools
menu.RegisterTools(server)
```

Each MCP-enabled option becomes its own tool with a clean schema containing only that option's fields.

## API

### Select Methods

| Method | Description |
|--------|-------------|
| `.ToolPrefix(prefix)` | Prefix for all tool names (e.g., "site" → "site_add") |
| `.Handler(fn)` | Shared handler for TUI and MCP |
| `.ToTools()` | Generate `[]ToolDef` (tool + handler pairs) |
| `.RegisterTools(server)` | Register all tools with MCP server |

### Option Methods

| Method | Description |
|--------|-------------|
| `.MCP(true)` | Expose option as an MCP tool |
| `.Description(text)` | Tool description |
| `.ToolName(name)` | Override default tool name |
| `.WithField(input)` | Attach input field to this option |

### Input Methods

| Method | Description |
|--------|-------------|
| `.Key(key)` | Field key in schema |
| `.Title(text)` | Field description |
| `.Required(bool)` | Mark field as required (default: true) |
| `.Format(format)` | Validation format: "domain", "uri" |
| `.Validate(fn)` | Custom validation function |

Options not marked with `.MCP(true)` are hidden from LLMs but available in TUI.

## Tips for MCP Tools

### Write Clear Descriptions

The `.Description()` text becomes the MCP tool's description that LLMs see when deciding which tool to use. Write descriptions that:

- **Explain what the tool does** clearly and concisely
- **Specify output format** (e.g., "Returns JSON data" vs "Returns Markdown text")  
- **Mention where data is stored** if applicable (e.g., "Saved to the users table")
- **Distinguish similar tools** - if you have multiple tools that sound similar, make their differences explicit

```go
// ❌ Bad: Vague, doesn't help LLM distinguish from similar tools
yeahno.NewOption("Create Report", "create_report").
    Description("Create a report")

// ✅ Good: Clear output format, storage location, and purpose
yeahno.NewOption("Create Report", "create_report").
    Description("Generate a Markdown research report stored in the reports table. Output is human-readable narrative text.")
```

### Avoid `fmt.Print` in Handlers

> [!CAUTION]
> **Never use `fmt.Print`, `fmt.Println`, or `fmt.Printf` in MCP tool handlers when running in stdio mode.**

MCP stdio transport uses stdout for JSON-RPC communication. Any text written to stdout (including debug prints) will corrupt the JSON stream and cause errors like:

```
ERROR calling "tools/call": invalid character '=' looking for beginning of value
```

Use `slog` or another logger configured to write to stderr instead:

```go
// ❌ Bad: Corrupts MCP stdio communication
fmt.Println("Processing request...")

// ✅ Good: Logs to stderr, not stdout
slog.Info("Processing request...")
```

## Example

```go
menu := yeahno.NewSelect[string]().
    Title("Company").
    ToolPrefix("company").
    Options(
        yeahno.NewOption("Add & Monitor", "add_monitor").
            Description("Add a company to monitor").
            WithField(yeahno.NewInput().Key("domain").Format("domain")).
            WithField(yeahno.NewInput().Key("name").Title("Company name")).
            WithField(yeahno.NewInput().Key("ticker").Required(false)).
            MCP(true),

        yeahno.NewOption("List", "list").
            Description("List monitored companies").
            MCP(true),

        yeahno.NewOption("Cancel", "cancel"),  // TUI only
    ).
    Handler(myHandler)

// Generates 2 tools:
// - company_add_monitor (with domain, name, ticker fields)
// - company_list (no fields)
tools, _ := menu.ToTools()
```

## Supported Types

| huh | yeahno |
|-----|--------|
| `NewSelect[T]()` | Yes |
| `NewInput()` | Yes |
| `NewText()` | Yes |
| `NewConfirm()` | Yes |
| `NewMultiSelect[T]()` | Yes |

## License

MIT
