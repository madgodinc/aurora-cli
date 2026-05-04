package ui

import "charm.land/lipgloss/v2"

// Aurora pink palette — Hello Kitty vibes
var (
	PinkLight  = lipgloss.Color("#FFB6C1") // light pink
	Pink       = lipgloss.Color("#FF69B4") // hot pink
	PinkBright = lipgloss.Color("#FF1493") // deep pink
	PinkMuted  = lipgloss.Color("#DB7093") // pale violet-red
	Rose       = lipgloss.Color("#FF007F") // rose
	White      = lipgloss.Color("#FFF0F5") // lavender blush
	Gray       = lipgloss.Color("#808080")
	DarkGray   = lipgloss.Color("#404040")
	Green      = lipgloss.Color("#00FF7F") // spring green
	Yellow     = lipgloss.Color("#FFD700") // gold
	Cyan       = lipgloss.Color("#00CED1") // dark turquoise
	Red        = lipgloss.Color("#FF4444")

	// Styles
	BannerStyle = lipgloss.NewStyle().
			Foreground(Pink).
			Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(White).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(PinkMuted)

	PromptStyle = lipgloss.NewStyle().
			Foreground(PinkBright).
			Bold(true)

	AuroraNameStyle = lipgloss.NewStyle().
			Foreground(Pink).
			Bold(true)

	UserNameStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(Gray)

	ToolStyle = lipgloss.NewStyle().
			Foreground(Yellow)

	ToolDoneStyle = lipgloss.NewStyle().
			Foreground(Green)

	ToolErrorStyle = lipgloss.NewStyle().
			Foreground(Red)

	CodeStyle = lipgloss.NewStyle().
			Foreground(Cyan)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(PinkLight).
			Background(lipgloss.Color("#1a1a2e"))

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Pink)

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(PinkMuted)
)
