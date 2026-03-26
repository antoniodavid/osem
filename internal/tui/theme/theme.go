package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name string

	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	Background       lipgloss.Color
	Surface          lipgloss.Color
	SurfaceHighlight lipgloss.Color

	Text       lipgloss.Color
	TextMuted  lipgloss.Color
	TextBright lipgloss.Color

	Border      lipgloss.Color
	BorderFocus lipgloss.Color

	Error   lipgloss.Color
	Success lipgloss.Color
	Warning lipgloss.Color
	Info    lipgloss.Color

	Bookmark lipgloss.Color
	Active   lipgloss.Color
}

func Get(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["default"]
}

func Names() []string {
	return []string{"default", "dracula", "catppuccin", "tokyo"}
}

var themes = map[string]Theme{
	"default": {
		Name:             "Default",
		Primary:          lipgloss.Color("36"),
		Secondary:        lipgloss.Color("37"),
		Accent:           lipgloss.Color("45"),
		Background:       lipgloss.Color("235"),
		Surface:          lipgloss.Color("237"),
		SurfaceHighlight: lipgloss.Color("238"),
		Text:             lipgloss.Color("252"),
		TextMuted:        lipgloss.Color("246"),
		TextBright:       lipgloss.Color("255"),
		Border:           lipgloss.Color("238"),
		BorderFocus:      lipgloss.Color("36"),
		Error:            lipgloss.Color("196"),
		Success:          lipgloss.Color("82"),
		Warning:          lipgloss.Color("214"),
		Info:             lipgloss.Color("39"),
		Bookmark:         lipgloss.Color("220"),
		Active:           lipgloss.Color("82"),
	},
	"dracula": {
		Name:             "Dracula",
		Primary:          lipgloss.Color("141"),
		Secondary:        lipgloss.Color("189"),
		Accent:           lipgloss.Color("213"),
		Background:       lipgloss.Color("235"),
		Surface:          lipgloss.Color("236"),
		SurfaceHighlight: lipgloss.Color("237"),
		Text:             lipgloss.Color("188"),
		TextMuted:        lipgloss.Color("246"),
		TextBright:       lipgloss.Color("255"),
		Border:           lipgloss.Color("238"),
		BorderFocus:      lipgloss.Color("141"),
		Error:            lipgloss.Color("203"),
		Success:          lipgloss.Color("120"),
		Warning:          lipgloss.Color("228"),
		Info:             lipgloss.Color("117"),
		Bookmark:         lipgloss.Color("228"),
		Active:           lipgloss.Color("120"),
	},
	"catppuccin": {
		Name:             "Catppuccin",
		Primary:          lipgloss.Color("183"),
		Secondary:        lipgloss.Color("189"),
		Accent:           lipgloss.Color("117"),
		Background:       lipgloss.Color("235"),
		Surface:          lipgloss.Color("237"),
		SurfaceHighlight: lipgloss.Color("238"),
		Text:             lipgloss.Color("189"),
		TextMuted:        lipgloss.Color("246"),
		TextBright:       lipgloss.Color("255"),
		Border:           lipgloss.Color("60"),
		BorderFocus:      lipgloss.Color("183"),
		Error:            lipgloss.Color("210"),
		Success:          lipgloss.Color("158"),
		Warning:          lipgloss.Color("222"),
		Info:             lipgloss.Color("153"),
		Bookmark:         lipgloss.Color("222"),
		Active:           lipgloss.Color("158"),
	},
	"tokyo": {
		Name:             "Tokyo Night",
		Primary:          lipgloss.Color("189"),
		Secondary:        lipgloss.Color("183"),
		Accent:           lipgloss.Color("117"),
		Background:       lipgloss.Color("235"),
		Surface:          lipgloss.Color("237"),
		SurfaceHighlight: lipgloss.Color("238"),
		Text:             lipgloss.Color("189"),
		TextMuted:        lipgloss.Color("246"),
		TextBright:       lipgloss.Color("255"),
		Border:           lipgloss.Color("60"),
		BorderFocus:      lipgloss.Color("189"),
		Error:            lipgloss.Color("210"),
		Success:          lipgloss.Color("158"),
		Warning:          lipgloss.Color("222"),
		Info:             lipgloss.Color("153"),
		Bookmark:         lipgloss.Color("222"),
		Active:           lipgloss.Color("158"),
	},
}
