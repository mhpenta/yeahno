# yeahno

> [!WARNING]
> This project is experimental. API may change.

Define a form once. Run it as a TUI, CLI, or MCP tools. For administrative bots and humans alike!

Built on [huh](https://github.com/charmbracelet/huh), [Cobra](https://github.com/spf13/cobra), and [Go's official MCP SDK](https://github.com/modelcontextprotocol/go-sdk).

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
            Description("Register a new site for monitoring. Saves to the sites table.").
            WithField(yeahno.NewInput().Key("domain").Format("domain")).
            MCP(true),

        yeahno.NewOption("List", "list").
            Description("List all monitored sites. Returns a formatted text list.").
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

// Or generate CLI commands (uses Cobra + fang for styled output)
rootCmd := &cobra.Command{Use: "myapp", Short: "My app"}
menu.RegisterCLI(rootCmd)
fang.Execute(ctx, rootCmd, fang.WithColorSchemeFunc(yeahno.DefaultTheme().FangColorScheme()))

// Or register as MCP tools
menu.RegisterTools(server)
```

Each MCP-enabled option becomes its own tool (or CLI subcommand) with a clean schema containing only that option's fields.

## API

### Select Methods

| Method | Description |
|--------|-------------|
| `.ToolPrefix(prefix)` | Prefix for all tool names (e.g., "site" → "site_add") |
| `.Handler(fn)` | Shared handler for TUI, CLI, and MCP |
| `.ToTools()` | Generate `[]ToolDef` (tool + handler pairs) |
| `.RegisterTools(server)` | Register all tools with MCP server |
| `.RegisterCLI(cmd)` | Register all subcommands with Cobra command |
| `.CLI()` | Generate standalone Cobra command tree |

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

Options not marked with `.MCP(true)` are hidden from LLMs and CLI but available in TUI.

## CLI

The CLI is generated from the same menu definition. Required flags are shown inline in help output:

```
$ myapp --help

  COMMANDS

    add-task --title <value> [--flags]    Create a new task
    complete-task --id <value>            Mark task as done
    list-tasks                            Show all tasks
```

### Theming

yeahno includes a default theme, or customize with your own colors:

```go
// Use default theme
theme := yeahno.DefaultTheme()

// Or customize
theme := yeahno.DefaultTheme().
    WithPrimary(lipgloss.Color("#00ff00")).
    WithError(lipgloss.Color("#ff0000"))

// Pass to fang
fang.Execute(ctx, rootCmd, fang.WithColorSchemeFunc(theme.FangColorScheme()))
```

| Theme Field | Usage |
|-------------|-------|
| `Primary` | Titles, commands |
| `Secondary` | Flags, arguments |
| `Muted` | Descriptions, dimmed text |
| `Surface` | Code block background (light terminal) |
| `SurfaceLight` | Code block background (dark terminal) |
| `Error` | Error header |

## Skills

Generate a `SKILL.md` agent skill definition from your menu. Skills describe available tools, workflows, and guidelines so LLMs know how to use your CLI/MCP server.

```go
skill, _ := menu.ToSkill()
skill.
    Workflow(
        "List existing tasks with `task_list-tasks`",
        "Add new tasks with `task_add-task`",
        "Complete tasks with `task_complete-task` using the task ID",
    ).
    Guideline("Priority values: low, normal, high, urgent").
    Section("Error Handling", "If a task ID is not found, call task_list-tasks first.").
    Attach(rootCmd)
```

Print the skill definition:

```bash
myapp --agent-skill-md
```

### Skill Methods

| Method | Description |
|--------|-------------|
| `.ToSkill()` | Generate a `Skill` from the menu (MCP-enabled options only) |
| `.Workflow(steps...)` | Define ordered workflow steps |
| `.Guideline(items...)` | Add usage guidelines |
| `.Section(heading, body)` | Add custom sections |
| `.Attach(cmd)` | Add `--agent-skill-md` flag to a Cobra command |
| `.String()` | Render the skill as Markdown with YAML frontmatter |

## Tips for MCP Tools

### Write Clear Descriptions

The `.Description()` text becomes the MCP tool's description that LLMs see when deciding which tool to use. Write descriptions that:

- **Explain what the tool does** clearly and concisely
- **Specify output format** (e.g., "Returns JSON data" vs "Returns Markdown text")  
- **Mention where data is stored** if applicable (e.g., "Saved to the users table")
- **Distinguish similar tools** - if you have multiple tools that sound similar, make their differences explicit

```go
// ❌ Bad: Vague, doesn't help LLM distinguish from similar tools
yeahno.NewOption("Create Job", "create_job").
    Description("Create a job")

// ✅ Good: Clear output format, storage location, and purpose
yeahno.NewOption("Create Job", "create_job").
    Description("Create a new repair job ticket. Saves to the jobs table and returns the job ID.")
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
            Description("Add a company and begin monitoring for changes. Saves to companies table.").
            WithField(yeahno.NewInput().Key("domain").Format("domain")).
            WithField(yeahno.NewInput().Key("name").Title("Company name")).
            WithField(yeahno.NewInput().Key("ticker").Required(false)).
            MCP(true),

        yeahno.NewOption("List", "list").
            Description("List all monitored companies. Returns a formatted text list.").
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
