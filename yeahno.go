package yeahno

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
)

type Option[T comparable] struct {
	Key      string
	Value    T
	selected bool

	mcp      bool
	fields   []*Input
	desc     string
	toolName string
}

func NewOption[T comparable](key string, value T) Option[T] {
	return Option[T]{Key: key, Value: value}
}

func NewOptions[T comparable](values ...T) []Option[T] {
	options := make([]Option[T], len(values))
	for i, v := range values {
		options[i] = Option[T]{
			Key:   fmt.Sprint(v),
			Value: v,
		}
	}
	return options
}

func (o Option[T]) Selected(selected bool) Option[T] {
	o.selected = selected
	return o
}

func (o Option[T]) String() string {
	return o.Key
}

func (o Option[T]) MCP(include bool) Option[T] {
	o.mcp = include
	return o
}

func (o Option[T]) Description(desc string) Option[T] {
	o.desc = desc
	return o
}

func (o Option[T]) WithField(field *Input) Option[T] {
	o.fields = append(o.fields, field)
	return o
}

func (o Option[T]) ToolName(name string) Option[T] {
	o.toolName = name
	return o
}

type Select[T comparable] struct {
	title       string
	description string
	options     []Option[T]
	value       *T
	validate    func(T) error
	height      int
	theme       *huh.Theme

	handler    func(ctx context.Context, value T, fields map[string]string) (any, error)
	toolPrefix string
}

func NewSelect[T comparable]() *Select[T] {
	return &Select[T]{
		validate: func(T) error { return nil },
	}
}

func (s *Select[T]) Value(value *T) *Select[T] {
	s.value = value
	return s
}

func (s *Select[T]) Title(title string) *Select[T] {
	s.title = title
	return s
}

func (s *Select[T]) Description(description string) *Select[T] {
	s.description = description
	return s
}

func (s *Select[T]) Options(options ...Option[T]) *Select[T] {
	s.options = options
	return s
}

func (s *Select[T]) Validate(validate func(T) error) *Select[T] {
	s.validate = validate
	return s
}

func (s *Select[T]) Height(height int) *Select[T] {
	s.height = height
	return s
}

func (s *Select[T]) WithTheme(theme *huh.Theme) *Select[T] {
	s.theme = theme
	return s
}

func (s *Select[T]) Handler(h func(ctx context.Context, value T, fields map[string]string) (any, error)) *Select[T] {
	s.handler = h
	return s
}

func (s *Select[T]) ToolPrefix(prefix string) *Select[T] {
	s.toolPrefix = prefix
	return s
}

func (s *Select[T]) Run(ctx context.Context) (any, error) {
	huhOpts := make([]huh.Option[T], len(s.options))
	for i, o := range s.options {
		huhOpts[i] = huh.NewOption(o.Key, o.Value)
		if o.selected {
			huhOpts[i] = huhOpts[i].Selected(true)
		}
	}

	sel := huh.NewSelect[T]().
		Title(s.title).
		Description(s.description).
		Options(huhOpts...).
		Value(s.value)

	if s.validate != nil {
		sel = sel.Validate(s.validate)
	}
	if s.height > 0 {
		sel = sel.Height(s.height)
	}

	form := huh.NewForm(huh.NewGroup(sel))
	if s.theme != nil {
		form = form.WithTheme(s.theme)
	}
	if err := form.Run(); err != nil {
		return nil, err
	}

	var selected *Option[T]
	for i := range s.options {
		if s.value != nil && s.options[i].Value == *s.value {
			selected = &s.options[i]
			break
		}
	}

	fields := make(map[string]string)
	if selected != nil && len(selected.fields) > 0 {
		for _, f := range selected.fields {
			var val string
			input := huh.NewInput().
				Title(f.title).
				Description(f.description).
				Placeholder(f.placeholder).
				Value(&val)

			// Build combined validator for format + custom validation
			input = input.Validate(f.buildValidator())
			if f.charLimit > 0 {
				input = input.CharLimit(f.charLimit)
			}

			inputForm := huh.NewForm(huh.NewGroup(input))
			if s.theme != nil {
				inputForm = inputForm.WithTheme(s.theme)
			}
			if err := inputForm.Run(); err != nil {
				return nil, err
			}
			fields[f.key] = val
		}
	}

	if s.handler != nil && s.value != nil {
		return s.handler(ctx, *s.value, fields)
	}

	if s.value != nil {
		return *s.value, nil
	}
	return nil, nil
}

type Input struct {
	title       string
	description string
	placeholder string
	value       *string
	validate    func(string) error
	charLimit   int
	theme       *huh.Theme

	key      string
	required bool
	format   string // JSON Schema format hint (e.g., "uri", "domain")
}

func NewInput() *Input {
	return &Input{
		required: true,
	}
}

func (i *Input) Value(value *string) *Input {
	i.value = value
	return i
}

func (i *Input) Title(title string) *Input {
	i.title = title
	return i
}

func (i *Input) Description(description string) *Input {
	i.description = description
	return i
}

func (i *Input) Placeholder(placeholder string) *Input {
	i.placeholder = placeholder
	return i
}

func (i *Input) Validate(validate func(string) error) *Input {
	i.validate = validate
	return i
}

func (i *Input) CharLimit(limit int) *Input {
	i.charLimit = limit
	return i
}

func (i *Input) WithTheme(theme *huh.Theme) *Input {
	i.theme = theme
	return i
}

func (i *Input) Key(key string) *Input {
	i.key = key
	return i
}

func (i *Input) Required(required bool) *Input {
	i.required = required
	return i
}

func (i *Input) Format(format string) *Input {
	i.format = format
	return i
}

func (i *Input) buildValidator() func(string) error {
	return func(s string) error {
		// Run format validation if specified
		if i.format != "" {
			if err := ValidateFormat(i.format, s); err != nil {
				return err
			}
		}
		// Run custom validation if specified
		if i.validate != nil {
			if err := i.validate(s); err != nil {
				return err
			}
		}
		return nil
	}
}

func (i *Input) Run() error {
	input := huh.NewInput().
		Title(i.title).
		Description(i.description).
		Placeholder(i.placeholder).
		Value(i.value)

	if i.validate != nil {
		input = input.Validate(i.validate)
	}
	if i.charLimit > 0 {
		input = input.CharLimit(i.charLimit)
	}

	form := huh.NewForm(huh.NewGroup(input))
	if i.theme != nil {
		form = form.WithTheme(i.theme)
	}
	return form.Run()
}

type Confirm struct {
	title       string
	description string
	affirmative string
	negative    string
	value       *bool
	validate    func(bool) error
	theme       *huh.Theme

	key string
}

func NewConfirm() *Confirm {
	return &Confirm{
		affirmative: "Yes",
		negative:    "No",
	}
}

func (c *Confirm) Value(value *bool) *Confirm {
	c.value = value
	return c
}

func (c *Confirm) Title(title string) *Confirm {
	c.title = title
	return c
}

func (c *Confirm) Description(description string) *Confirm {
	c.description = description
	return c
}

func (c *Confirm) Affirmative(text string) *Confirm {
	c.affirmative = text
	return c
}

func (c *Confirm) Negative(text string) *Confirm {
	c.negative = text
	return c
}

func (c *Confirm) Validate(validate func(bool) error) *Confirm {
	c.validate = validate
	return c
}

func (c *Confirm) WithTheme(theme *huh.Theme) *Confirm {
	c.theme = theme
	return c
}

func (c *Confirm) Key(key string) *Confirm {
	c.key = key
	return c
}

func (c *Confirm) Run() error {
	confirm := huh.NewConfirm().
		Title(c.title).
		Description(c.description).
		Affirmative(c.affirmative).
		Negative(c.negative).
		Value(c.value)

	if c.validate != nil {
		confirm = confirm.Validate(c.validate)
	}

	form := huh.NewForm(huh.NewGroup(confirm))
	if c.theme != nil {
		form = form.WithTheme(c.theme)
	}
	return form.Run()
}

type Text struct {
	title       string
	description string
	placeholder string
	value       *string
	validate    func(string) error
	charLimit   int
	lines       int
	theme       *huh.Theme

	key      string
	required bool
}

func NewText() *Text {
	return &Text{
		required: true,
		lines:    3,
	}
}

func (t *Text) Value(value *string) *Text {
	t.value = value
	return t
}

func (t *Text) Title(title string) *Text {
	t.title = title
	return t
}

func (t *Text) Description(description string) *Text {
	t.description = description
	return t
}

func (t *Text) Placeholder(placeholder string) *Text {
	t.placeholder = placeholder
	return t
}

func (t *Text) Validate(validate func(string) error) *Text {
	t.validate = validate
	return t
}

func (t *Text) CharLimit(limit int) *Text {
	t.charLimit = limit
	return t
}

func (t *Text) Lines(lines int) *Text {
	t.lines = lines
	return t
}

func (t *Text) WithTheme(theme *huh.Theme) *Text {
	t.theme = theme
	return t
}

func (t *Text) Key(key string) *Text {
	t.key = key
	return t
}

func (t *Text) Required(required bool) *Text {
	t.required = required
	return t
}

func (t *Text) Run() error {
	text := huh.NewText().
		Title(t.title).
		Description(t.description).
		Placeholder(t.placeholder).
		Value(t.value)

	if t.validate != nil {
		text = text.Validate(t.validate)
	}
	if t.charLimit > 0 {
		text = text.CharLimit(t.charLimit)
	}
	if t.lines > 0 {
		text = text.Lines(t.lines)
	}

	form := huh.NewForm(huh.NewGroup(text))
	if t.theme != nil {
		form = form.WithTheme(t.theme)
	}
	return form.Run()
}

type MultiSelect[T comparable] struct {
	title       string
	description string
	options     []Option[T]
	value       *[]T
	validate    func([]T) error
	limit       int
	height      int
	theme       *huh.Theme

	key string
}

func NewMultiSelect[T comparable]() *MultiSelect[T] {
	return &MultiSelect[T]{
		validate: func([]T) error { return nil },
	}
}

func (m *MultiSelect[T]) Value(value *[]T) *MultiSelect[T] {
	m.value = value
	return m
}

func (m *MultiSelect[T]) Title(title string) *MultiSelect[T] {
	m.title = title
	return m
}

func (m *MultiSelect[T]) Description(description string) *MultiSelect[T] {
	m.description = description
	return m
}

func (m *MultiSelect[T]) Options(options ...Option[T]) *MultiSelect[T] {
	m.options = options
	return m
}

func (m *MultiSelect[T]) Validate(validate func([]T) error) *MultiSelect[T] {
	m.validate = validate
	return m
}

func (m *MultiSelect[T]) Limit(limit int) *MultiSelect[T] {
	m.limit = limit
	return m
}

func (m *MultiSelect[T]) Height(height int) *MultiSelect[T] {
	m.height = height
	return m
}

func (m *MultiSelect[T]) WithTheme(theme *huh.Theme) *MultiSelect[T] {
	m.theme = theme
	return m
}

func (m *MultiSelect[T]) Key(key string) *MultiSelect[T] {
	m.key = key
	return m
}

func (m *MultiSelect[T]) Run() error {
	huhOpts := make([]huh.Option[T], len(m.options))
	for i, o := range m.options {
		huhOpts[i] = huh.NewOption(o.Key, o.Value)
		if o.selected {
			huhOpts[i] = huhOpts[i].Selected(true)
		}
	}

	ms := huh.NewMultiSelect[T]().
		Title(m.title).
		Description(m.description).
		Options(huhOpts...).
		Value(m.value)

	if m.validate != nil {
		ms = ms.Validate(m.validate)
	}
	if m.limit > 0 {
		ms = ms.Limit(m.limit)
	}
	if m.height > 0 {
		ms = ms.Height(m.height)
	}

	form := huh.NewForm(huh.NewGroup(ms))
	if m.theme != nil {
		form = form.WithTheme(m.theme)
	}
	return form.Run()
}
