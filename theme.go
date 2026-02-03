package yeahno

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
)

// Theme defines colors for yeahno CLI output.
// Use NewTheme() or DefaultTheme() to create one.
type Theme struct {
	// Primary is the main accent color (titles, commands)
	Primary color.Color
	// Secondary is for less prominent elements (descriptions, comments)
	Secondary color.Color
	// Muted is for dimmed/inactive elements
	Muted color.Color
	// Surface is for backgrounds (e.g., code blocks, usage boxes)
	Surface color.Color
	// SurfaceLight is for backgrounds on dark terminals (defaults to slightly lighter Surface)
	SurfaceLight color.Color
	// Error is for error states
	Error color.Color
}

// DefaultTheme returns the yeahno default color theme.
func DefaultTheme() *Theme {
	return &Theme{
		Primary:      lipgloss.Color("#ff7f11"), // Vivid Tangerine
		Secondary:    lipgloss.Color("#e2e8ce"), // Beige
		Muted:        lipgloss.Color("#acbfa4"), // Ash Grey
		Surface:      lipgloss.Color("#262626"), // Carbon Black
		SurfaceLight: lipgloss.Color("#363636"), // Slightly lighter for dark terminals
		Error:        lipgloss.Color("#ff1b1c"), // Red
	}
}

// NewTheme creates a theme with the given colors.
// Any nil colors will use defaults.
func NewTheme(primary, secondary, muted, surface, surfaceLight, errorColor color.Color) *Theme {
	t := DefaultTheme()
	if primary != nil {
		t.Primary = primary
	}
	if secondary != nil {
		t.Secondary = secondary
	}
	if muted != nil {
		t.Muted = muted
	}
	if surface != nil {
		t.Surface = surface
	}
	if surfaceLight != nil {
		t.SurfaceLight = surfaceLight
	}
	if errorColor != nil {
		t.Error = errorColor
	}
	return t
}

// WithPrimary returns a copy of the theme with a new primary color.
func (t *Theme) WithPrimary(c color.Color) *Theme {
	t2 := *t
	t2.Primary = c
	return &t2
}

// WithSecondary returns a copy of the theme with a new secondary color.
func (t *Theme) WithSecondary(c color.Color) *Theme {
	t2 := *t
	t2.Secondary = c
	return &t2
}

// WithMuted returns a copy of the theme with a new muted color.
func (t *Theme) WithMuted(c color.Color) *Theme {
	t2 := *t
	t2.Muted = c
	return &t2
}

// WithSurface returns a copy of the theme with a new surface color.
func (t *Theme) WithSurface(c color.Color) *Theme {
	t2 := *t
	t2.Surface = c
	return &t2
}

// WithSurfaceLight returns a copy of the theme with a new surface light color.
func (t *Theme) WithSurfaceLight(c color.Color) *Theme {
	t2 := *t
	t2.SurfaceLight = c
	return &t2
}

// WithError returns a copy of the theme with a new error color.
func (t *Theme) WithError(c color.Color) *Theme {
	t2 := *t
	t2.Error = c
	return &t2
}

// FangColorScheme converts the theme to a fang.ColorScheme.
// Use this with fang.WithColorSchemeFunc().
func (t *Theme) FangColorScheme() func(lipgloss.LightDarkFunc) fang.ColorScheme {
	return func(ld lipgloss.LightDarkFunc) fang.ColorScheme {
		return fang.ColorScheme{
			Base:           ld(t.Secondary, t.Muted),
			Title:          t.Primary,
			Description:    t.Muted,
			Codeblock:      ld(t.Surface, t.SurfaceLight), // Background for usage box
			Program:        t.Primary,
			DimmedArgument: t.Muted,
			Comment:        t.Muted,
			Flag:           t.Secondary,
			FlagDefault:    t.Muted,
			Command:        t.Primary,
			QuotedString:   t.Muted,
			Argument:       t.Secondary,
			Help:           t.Muted,
			Dash:           t.Muted,
			ErrorHeader:    [2]color.Color{t.Secondary, t.Error},
			ErrorDetails:   t.Secondary,
		}
	}
}
