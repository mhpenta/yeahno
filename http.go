package yeahno

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mhpenta/tap-go/server"
)

type httpTool struct {
	name        string
	description string
	parameters  map[string]any
	handler     func(ctx context.Context, args json.RawMessage) (any, error)
}

func (s *Select[T]) toHTTPTools() ([]httpTool, error) {
	if s.handler == nil {
		return nil, fmt.Errorf("no handler configured")
	}

	var opts []Option[T]
	for _, o := range s.options {
		if o.mcp {
			opts = append(opts, o)
		}
	}
	if len(opts) == 0 {
		opts = s.options
	}

	var tools []httpTool
	for _, opt := range opts {
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

		properties := make(map[string]any)
		var required []string
		var propertyOrder []string

		for _, f := range opt.fields {
			fKey := f.key
			if fKey == "" {
				fKey = toSnakeCase(f.title)
			}

			prop := map[string]any{"type": "string"}
			if f.title != "" {
				prop["description"] = f.title
			}
			if f.charLimit > 0 {
				prop["maxLength"] = f.charLimit
			}
			if f.format != "" {
				if fv, ok := formatValidators[f.format]; ok {
					prop["format"] = fv.schemaFormat
				}
			}

			properties[fKey] = prop
			propertyOrder = append(propertyOrder, fKey)
			if f.required {
				required = append(required, fKey)
			}
		}

		params := map[string]any{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			params["required"] = required
		}

		opt := opt
		handler := s.makeHTTPHandler(opt)

		tools = append(tools, httpTool{
			name:        toolName,
			description: desc,
			parameters:  params,
			handler:     handler,
		})
	}

	return tools, nil
}

func (s *Select[T]) makeHTTPHandler(opt Option[T]) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var input map[string]any
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
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
					return nil, fmt.Errorf("%s exceeds maximum length of %d", fKey, limit)
				}
				if f.format != "" {
					if fv, ok := formatValidators[f.format]; ok {
						if err := fv.validate(val); err != nil {
							return nil, fmt.Errorf("invalid %s: %v", fKey, err)
						}
					}
				}
				if f.validate != nil {
					if err := f.validate(val); err != nil {
						return nil, fmt.Errorf("invalid %s: %v", fKey, err)
					}
				}
				fields[fKey] = val
			} else if f.required {
				return nil, fmt.Errorf("missing required field: %s", fKey)
			}
		}

		return s.handler(ctx, opt.Value, fields)
	}
}

func (s *Select[T]) RegisterTAP(mux *http.ServeMux) error {
	tools, err := s.toHTTPTools()
	if err != nil {
		return err
	}

	srv := server.New(s.tapDescription())
	for i := range tools {
		t := tools[i]
		srv.AddTool(&server.Tool{
			Name:        t.name,
			Description: t.description,
			Parameters:  t.parameters,
			Handler:     t.handler,
		})
	}

	srv.Register(mux, nil)

	return nil
}

func (s *Select[T]) RegisterHTTP(mux *http.ServeMux) error {
	return s.RegisterTAP(mux)
}

func (s *Select[T]) tapDescription() string {
	if s.title != "" && s.description != "" {
		return fmt.Sprintf("%s - %s", s.title, s.description)
	}
	if s.title != "" {
		return s.title
	}
	return s.description
}
