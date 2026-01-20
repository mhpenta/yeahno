# yeahno

<p>
<img width="525" alt="yeahno Logo" src="https://github.com/user-attachments/assets/1aeec958-e8cd-4376-a876-21e161af3023">
</p>

> [!WARNING]
> This project is experimental. API may change.

Define a form once. Run it as a TUI or expose it as MCP tools.

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
