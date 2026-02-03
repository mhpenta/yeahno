package yeahno

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ToCLI generates Cobra commands from the Select menu.
// Each MCP-enabled option becomes a subcommand.
// Fields become flags on the subcommand.
func (s *Select[T]) ToCLI() (*cobra.Command, error) {
	if s.handler == nil {
		return nil, fmt.Errorf("no handler configured")
	}

	// Build root command from select metadata
	rootName := toSnakeCase(s.title)
	if s.toolPrefix != "" {
		rootName = s.toolPrefix
	}

	root := &cobra.Command{
		Use:   rootName,
		Short: s.description,
	}

	// Determine which options to include
	var cliOptions []Option[T]
	for _, o := range s.options {
		if o.mcp {
			cliOptions = append(cliOptions, o)
		}
	}
	if len(cliOptions) == 0 {
		cliOptions = s.options
	}

	// Create subcommand for each option
	for _, opt := range cliOptions {
		cmd := s.buildSubcommand(opt)
		root.AddCommand(cmd)
	}

	return root, nil
}

// ToSubcommands generates Cobra subcommands without a root wrapper.
// Use this to attach commands directly to an existing Cobra root.
func (s *Select[T]) ToSubcommands() ([]*cobra.Command, error) {
	if s.handler == nil {
		return nil, fmt.Errorf("no handler configured")
	}

	var cliOptions []Option[T]
	for _, o := range s.options {
		if o.mcp {
			cliOptions = append(cliOptions, o)
		}
	}
	if len(cliOptions) == 0 {
		cliOptions = s.options
	}

	var cmds []*cobra.Command
	for _, opt := range cliOptions {
		cmds = append(cmds, s.buildSubcommand(opt))
	}

	return cmds, nil
}

func (s *Select[T]) buildSubcommand(opt Option[T]) *cobra.Command {
	cmdName := opt.toolName
	if cmdName == "" {
		cmdName = opt.Key
	}
	cmdName = toKebabCase(cmdName)

	desc := opt.desc
	if desc == "" {
		desc = opt.Key
	}

	// Build usage string with required flags shown inline
	// Show first 2 required flags, then "(+N more)" if there are more
	const maxShownFlags = 2
	usageParts := []string{cmdName}
	var requiredFlags []string
	for _, f := range opt.fields {
		if f.required {
			fKey := f.key
			if fKey == "" {
				fKey = toSnakeCase(f.title)
			}
			flagName := toKebabCase(fKey)
			requiredFlags = append(requiredFlags, fmt.Sprintf("--%s <value>", flagName))
		}
	}
	
	if len(requiredFlags) <= maxShownFlags {
		usageParts = append(usageParts, requiredFlags...)
	} else {
		usageParts = append(usageParts, requiredFlags[:maxShownFlags]...)
		usageParts = append(usageParts, fmt.Sprintf("(+%d more)", len(requiredFlags)-maxShownFlags))
	}
	
	// Add [flags] if there are optional flags
	hasOptional := false
	for _, f := range opt.fields {
		if !f.required {
			hasOptional = true
			break
		}
	}
	if hasOptional {
		usageParts = append(usageParts, "[flags]")
	}
	useString := strings.Join(usageParts, " ")

	// Track flag values
	flagValues := make(map[string]*string)

	cmd := &cobra.Command{
		Use:   useString,
		Short: desc,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Collect field values from flags
			fields := make(map[string]string)
			for _, f := range opt.fields {
				fKey := f.key
				if fKey == "" {
					fKey = toSnakeCase(f.title)
				}

				if val, ok := flagValues[fKey]; ok && val != nil && *val != "" {
					// Validate format if specified
					if f.format != "" {
						if err := ValidateFormat(f.format, *val); err != nil {
							return fmt.Errorf("invalid %s: %w", fKey, err)
						}
					}
					// Run custom validation
					if f.validate != nil {
						if err := f.validate(*val); err != nil {
							return fmt.Errorf("invalid %s: %w", fKey, err)
						}
					}
					fields[fKey] = *val
				} else if f.required {
					return fmt.Errorf("required flag --%s not provided", toKebabCase(fKey))
				}
			}

			// Call handler
			result, err := s.handler(cmd.Context(), opt.Value, fields)
			if err != nil {
				return err
			}

			// Output result
			output := formatCLIOutput(result)
			fmt.Fprintln(cmd.OutOrStdout(), output)
			return nil
		},
	}

	// Add flags for each field
	for _, f := range opt.fields {
		fKey := f.key
		if fKey == "" {
			fKey = toSnakeCase(f.title)
		}

		flagName := toKebabCase(fKey)
		flagDesc := f.title
		if f.description != "" {
			flagDesc = f.description
		}
		if f.required {
			flagDesc += " (required)"
		}

		val := new(string)
		flagValues[fKey] = val
		cmd.Flags().StringVar(val, flagName, "", flagDesc)

		if f.required {
			cmd.MarkFlagRequired(flagName)
		}
	}

	return cmd
}

// toKebabCase converts a string to kebab-case for CLI flag/command names.
func toKebabCase(s string) string {
	s = strings.ToLower(s)
	s = snakeCasePattern.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// formatCLIOutput formats handler results for CLI output.
func formatCLIOutput(result any) string {
	switch v := result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case []string:
		return strings.Join(v, "\n")
	default:
		// For complex types, output as JSON
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", result)
		}
		return string(data)
	}
}

// RegisterCLI adds all generated subcommands to an existing Cobra command.
func (s *Select[T]) RegisterCLI(parent *cobra.Command) error {
	cmds, err := s.ToSubcommands()
	if err != nil {
		return err
	}
	for _, cmd := range cmds {
		parent.AddCommand(cmd)
	}
	return nil
}

// CLI returns a single Cobra command tree. Alias for ToCLI.
func (s *Select[T]) CLI() (*cobra.Command, error) {
	return s.ToCLI()
}
