package yeahno

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxFieldLength = 10000

type ToolDef struct {
	Tool    *mcp.Tool
	Handler mcp.ToolHandler
}

type formatValidator struct {
	schemaFormat string
	validate     func(string) error
}

var hostnamePattern = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

var formatValidators = map[string]formatValidator{
	"uri": {
		schemaFormat: "uri",
		validate: func(s string) error {
			s = strings.TrimSpace(s)
			if s == "" {
				return nil
			}
			parsed, err := url.Parse(s)
			if err != nil {
				return fmt.Errorf("invalid URI format")
			}
			if parsed.Scheme == "" {
				s = "https://" + s
				parsed, err = url.Parse(s)
				if err != nil {
					return fmt.Errorf("invalid URI format")
				}
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				return fmt.Errorf("invalid URI scheme: only http and https are allowed")
			}
			if parsed.Host == "" {
				return fmt.Errorf("invalid URI format")
			}
			return nil
		},
	},
	"domain": {
		schemaFormat: "hostname",
		validate: func(s string) error {
			s = strings.TrimSpace(s)
			if s == "" {
				return nil
			}
			s = strings.TrimPrefix(s, "https://")
			s = strings.TrimPrefix(s, "http://")
			if idx := strings.Index(s, "/"); idx != -1 {
				s = s[:idx]
			}
			if !hostnamePattern.MatchString(s) {
				return fmt.Errorf("invalid domain format")
			}
			return nil
		},
	},
}

func ValidateFormat(format, value string) error {
	if fv, ok := formatValidators[format]; ok {
		return fv.validate(value)
	}
	return nil
}

var (
	snakeCasePattern = regexp.MustCompile(`[^a-z0-9]+`)
)

func toSnakeCase(s string) string {
	s = strings.ToLower(s)
	s = snakeCasePattern.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}

func intPtr(i int) *int { return &i }

func resultToString(result any) string {
	switch v := result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		data, _ := json.Marshal(result)
		return string(data)
	}
}

func (s *Select[T]) ToTools() ([]ToolDef, error) {
	if s.handler == nil {
		return nil, fmt.Errorf("no handler configured")
	}

	var mcpOptions []Option[T]
	for _, o := range s.options {
		if o.mcp {
			mcpOptions = append(mcpOptions, o)
		}
	}

	if len(mcpOptions) == 0 {
		mcpOptions = s.options
	}

	var tools []ToolDef
	for _, opt := range mcpOptions {
		toolName := opt.toolName
		if toolName == "" {
			toolName = opt.Key
		}
		toolName = toSnakeCase(toolName)

		if s.toolPrefix != "" {
			toolName = s.toolPrefix + "_" + toolName
		}

		desc := opt.desc
		if desc == "" {
			desc = opt.Key
		}

		properties := make(map[string]*jsonschema.Schema)
		var propertyOrder []string
		var required []string

		for _, f := range opt.fields {
			fKey := f.key
			if fKey == "" {
				fKey = toSnakeCase(f.title)
			}

			schema := &jsonschema.Schema{Type: "string"}
			if f.title != "" {
				schema.Description = f.title
			}
			if f.charLimit > 0 {
				schema.MaxLength = intPtr(f.charLimit)
			}
			if f.format != "" {
				if fv, ok := formatValidators[f.format]; ok {
					schema.Format = fv.schemaFormat
				}
			}
			properties[fKey] = schema
			propertyOrder = append(propertyOrder, fKey)

			if f.required {
				required = append(required, fKey)
			}
		}

		jschema := &jsonschema.Schema{
			Type:          "object",
			Properties:    properties,
			PropertyOrder: propertyOrder,
		}
		if len(required) > 0 {
			jschema.Required = required
		}

		schemaBytes, err := json.Marshal(jschema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema for tool %s: %w", toolName, err)
		}
		var schemaMap map[string]any
		if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema for tool %s: %w", toolName, err)
		}
		
		if len(required) > 0 {
			schemaMap["required"] = required
		}

		tool := &mcp.Tool{
			Name:        toolName,
			Description: desc,
			InputSchema: schemaMap,
		}

		handler := s.makeToolHandler(opt)
		tools = append(tools, ToolDef{Tool: tool, Handler: handler})
	}

	return tools, nil
}

func (s *Select[T]) makeToolHandler(opt Option[T]) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input map[string]any
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to parse arguments: %v", err)}},
				IsError: true,
			}, nil
		}

		fields := make(map[string]string)
		for _, f := range opt.fields {
			fKey := f.key
			if fKey == "" {
				fKey = toSnakeCase(f.title)
			}

			if val, ok := input[fKey].(string); ok {
				limit := maxFieldLength
				if f.charLimit > 0 {
					limit = f.charLimit
				}
				if len(val) > limit {
					return &mcp.CallToolResult{
						Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("%s exceeds maximum length of %d", fKey, limit)}},
						IsError: true,
					}, nil
				}
				if f.format != "" {
					if fv, ok := formatValidators[f.format]; ok {
						if err := fv.validate(val); err != nil {
							return &mcp.CallToolResult{
								Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("invalid %s: %v", fKey, err)}},
								IsError: true,
							}, nil
						}
					}
				}
				if f.validate != nil {
					if err := f.validate(val); err != nil {
						return &mcp.CallToolResult{
							Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("invalid %s: %v", fKey, err)}},
							IsError: true,
						}, nil
					}
				}
				fields[fKey] = val
			} else if f.required {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("missing required field: %s", fKey)}},
					IsError: true,
				}, nil
			}
		}

		result, err := s.handler(ctx, opt.Value, fields)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "tool execution failed"}},
				IsError: true,
			}, nil
		}

		text := resultToString(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil
	}
}

func (s *Select[T]) RegisterTools(server *mcp.Server) error {
	tools, err := s.ToTools()
	if err != nil {
		return err
	}
	for _, td := range tools {
		server.AddTool(td.Tool, td.Handler)
	}
	return nil
}
