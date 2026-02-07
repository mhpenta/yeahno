package yeahno

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type skillTool struct {
	name   string
	desc   string
	fields []skillField
}

type skillField struct {
	key      string
	required bool
	format   string
	title    string
}

type skillSection struct {
	heading string
	body    string
}

type Skill struct {
	name        string
	description string
	title       string
	tools       []skillTool
	workflows   []string
	guidelines  []string
	sections    []skillSection
}

func (s *Select[T]) ToSkill() (*Skill, error) {
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

	name := toKebabCase(s.title)
	if s.toolPrefix != "" {
		name = toKebabCase(s.toolPrefix)
	}

	desc := s.description
	if desc == "" {
		desc = s.title
	}

	sk := &Skill{
		name:  name,
		title: s.title,
	}

	var tools []skillTool
	for _, opt := range mcpOptions {
		toolName := skillToolName(opt, s.toolPrefix)
		optDesc := opt.desc
		if optDesc == "" {
			optDesc = opt.Key
		}

		var fields []skillField
		for _, f := range opt.fields {
			format := f.format
			if format == "" {
				format = "string"
			}
			fields = append(fields, skillField{
				key:      f.resolvedKey(),
				required: f.required,
				format:   format,
				title:    f.resolvedTitle(),
			})
		}

		tools = append(tools, skillTool{name: toolName, desc: optDesc, fields: fields})
	}
	sk.tools = tools

	sk.description = skillDescription(s.title, desc, mcpOptions)

	sk.guidelines = defaultGuidelines(mcpOptions)

	return sk, nil
}

func (sk *Skill) Description(desc string) *Skill {
	sk.description = desc
	return sk
}

func (sk *Skill) Workflow(steps ...string) *Skill {
	sk.workflows = append(sk.workflows, steps...)
	return sk
}

func (sk *Skill) Guideline(items ...string) *Skill {
	sk.guidelines = append(sk.guidelines, items...)
	return sk
}

func (sk *Skill) Section(heading, body string) *Skill {
	sk.sections = append(sk.sections, skillSection{heading: heading, body: body})
	return sk
}

func (sk *Skill) Attach(cmd *cobra.Command) *Skill {
	var showSkill bool
	cmd.PersistentFlags().BoolVar(&showSkill, "agent-skill-md", false, "Print agent skill definition (SKILL.md) to stdout")

	if cmd.RunE == nil && cmd.Run == nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		}
	}

	existing := cmd.PersistentPreRunE
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if showSkill {
			fmt.Fprint(cmd.OutOrStdout(), sk.String())
			os.Exit(0)
		}
		if existing != nil {
			return existing(cmd, args)
		}
		return nil
	}
	return sk
}

func (sk *Skill) String() string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", sk.name))
	b.WriteString(fmt.Sprintf("description: %s\n", sk.description))
	b.WriteString("---\n\n")

	b.WriteString(fmt.Sprintf("# %s\n\n", sk.title))

	b.WriteString("## Available Tools\n\n")
	for _, tool := range sk.tools {
		b.WriteString(fmt.Sprintf("### `%s`\n\n", tool.name))
		b.WriteString(fmt.Sprintf("%s\n\n", tool.desc))

		if len(tool.fields) > 0 {
			b.WriteString("| Field | Required | Format | Description |\n")
			b.WriteString("|-------|----------|--------|-------------|\n")
			for _, f := range tool.fields {
				req := "yes"
				if !f.required {
					req = "no"
				}
				b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n", f.key, req, f.format, f.title))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("## Workflow\n\n")
	if len(sk.workflows) > 0 {
		for i, step := range sk.workflows {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	} else {
		for i, tool := range sk.tools {
			b.WriteString(fmt.Sprintf("%d. **%s** — `%s`\n", i+1, tool.desc, tool.name))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Guidelines\n\n")
	for _, g := range sk.guidelines {
		b.WriteString(fmt.Sprintf("- %s\n", g))
	}
	if len(sk.guidelines) > 0 {
		b.WriteString("\n")
	}

	for _, sec := range sk.sections {
		b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", sec.heading, sec.body))
	}

	return b.String()
}

func skillToolName[T comparable](opt Option[T], prefix string) string {
	toolName := opt.toolName
	if toolName == "" {
		toolName = opt.Key
	}
	toolName = toSnakeCase(toolName)
	if prefix != "" {
		toolName = prefix + "_" + toolName
	}
	return toolName
}

func skillDescription[T comparable](title, desc string, opts []Option[T]) string {
	var actions []string
	for _, opt := range opts {
		d := opt.desc
		if d == "" {
			d = opt.Key
		}
		actions = append(actions, strings.ToLower(d))
	}

	var sb strings.Builder
	sb.WriteString(desc)
	if len(actions) > 0 {
		sb.WriteString(" Use when the user needs to: ")
		for i, a := range actions {
			if i > 0 && i == len(actions)-1 {
				sb.WriteString(", or ")
			} else if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(a)
		}
		sb.WriteString(".")
	}
	return sb.String()
}

func defaultGuidelines[T comparable](opts []Option[T]) []string {
	var guidelines []string

	var formats []string
	var requiredFields []string
	for _, opt := range opts {
		for _, f := range opt.fields {
			fKey := f.resolvedKey()
			if f.required {
				requiredFields = append(requiredFields, fmt.Sprintf("`%s`", fKey))
			}
			if f.format != "" {
				formats = append(formats, fmt.Sprintf("`%s` (%s format)", fKey, f.format))
			}
		}
	}

	if len(formats) > 0 {
		guidelines = append(guidelines, fmt.Sprintf("Validate format before submission: %s", strings.Join(formats, ", ")))
	}
	if len(requiredFields) > 0 {
		guidelines = append(guidelines, fmt.Sprintf("Always provide required fields: %s", strings.Join(requiredFields, ", ")))
	}
	guidelines = append(guidelines, "Handle errors gracefully and inform the user")

	return guidelines
}
