package ui

import "charm.land/lipgloss/v2"

var (
	PinkLight  = lipgloss.Color("#FFB6C1")
	Pink       = lipgloss.Color("#FF69B4")
	PinkBright = lipgloss.Color("#FF1493")
	PinkMuted  = lipgloss.Color("#DB7093")
	Rose       = lipgloss.Color("#FF007F")
	White      = lipgloss.Color("#FFF0F5")
	Gray       = lipgloss.Color("#808080")
	DarkGray   = lipgloss.Color("#404040")
	Green      = lipgloss.Color("#00FF7F")
	Yellow     = lipgloss.Color("#FFD700")
	Cyan       = lipgloss.Color("#00CED1")
	Red        = lipgloss.Color("#FF4444")

	BannerStyle     = lipgloss.NewStyle().Foreground(Pink).Bold(true)
	TitleStyle      = lipgloss.NewStyle().Foreground(White).Bold(true)
	SubtitleStyle   = lipgloss.NewStyle().Foreground(PinkMuted)
	PromptStyle     = lipgloss.NewStyle().Foreground(PinkBright).Bold(true)
	AuroraNameStyle = lipgloss.NewStyle().Foreground(Pink).Bold(true)
	UserNameStyle   = lipgloss.NewStyle().Foreground(Green).Bold(true)
	DimStyle        = lipgloss.NewStyle().Foreground(Gray)
	ToolStyle       = lipgloss.NewStyle().Foreground(Yellow)
	ToolDoneStyle   = lipgloss.NewStyle().Foreground(Green)
	ToolErrorStyle  = lipgloss.NewStyle().Foreground(Red)
	CodeStyle       = lipgloss.NewStyle().Foreground(Cyan)
	StatusBarStyle  = lipgloss.NewStyle().Foreground(PinkLight).Background(lipgloss.Color("#1a1a2e"))
	BorderStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Pink)
	SeparatorStyle  = lipgloss.NewStyle().Foreground(PinkMuted)
)

func applyTheme(name string) {
	switch name {
	case "pink": // Default Hello Kitty
		Pink = lipgloss.Color("#FF69B4")
		PinkBright = lipgloss.Color("#FF1493")
		PinkMuted = lipgloss.Color("#DB7093")
		BannerStyle = lipgloss.NewStyle().Foreground(Pink).Bold(true)
		PromptStyle = lipgloss.NewStyle().Foreground(PinkBright).Bold(true)
		AuroraNameStyle = lipgloss.NewStyle().Foreground(Pink).Bold(true)
		StatusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB6C1")).Background(lipgloss.Color("#1a1a2e"))
		SeparatorStyle = lipgloss.NewStyle().Foreground(PinkMuted)

	case "cyber": // Cyberpunk neon
		Pink = lipgloss.Color("#00FFFF")        // cyan
		PinkBright = lipgloss.Color("#FF00FF")   // magenta
		PinkMuted = lipgloss.Color("#7B68EE")    // medium slate blue
		BannerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
		PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Bold(true)
		AuroraNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
		StatusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Background(lipgloss.Color("#0d0d1a"))
		SeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7B68EE"))
		Yellow = lipgloss.Color("#FFFF00")
		Green = lipgloss.Color("#39FF14")

	case "dark": // Minimal dark
		Pink = lipgloss.Color("#888888")
		PinkBright = lipgloss.Color("#AAAAAA")
		PinkMuted = lipgloss.Color("#555555")
		BannerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Bold(true)
		PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Bold(true)
		AuroraNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Bold(true)
		StatusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Background(lipgloss.Color("#111111"))
		SeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	}

	// Update dependent styles
	ToolStyle = lipgloss.NewStyle().Foreground(Yellow)
	ToolDoneStyle = lipgloss.NewStyle().Foreground(Green)
	ToolErrorStyle = lipgloss.NewStyle().Foreground(Red)
	DimStyle = lipgloss.NewStyle().Foreground(Gray)
	UserNameStyle = lipgloss.NewStyle().Foreground(Green).Bold(true)
	CodeStyle = lipgloss.NewStyle().Foreground(Cyan)
	BorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Pink)
}
