package styles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles contains all the styles used in the application
type Styles struct {
	AppName,
	CliArgs,
	Comment,
	CyclingChars,
	ErrorHeader,
	ErrorDetails,
	ErrPadding,
	Flag,
	FlagComma,
	FlagDesc,
	InlineCode,
	Link,
	Pipe,
	Quote,
	ConversationList,
	SHA1,
	Timeago lipgloss.Style
}

// MakeStyles creates a new set of styles
func MakeStyles(r *lipgloss.Renderer) (s Styles) {
	const horizontalEdgePadding = 2
	s.AppName = r.NewStyle().Bold(true)
	s.CliArgs = r.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))
	s.Comment = r.NewStyle().Foreground(lipgloss.Color("#565f89"))
	s.CyclingChars = r.NewStyle().Foreground(lipgloss.Color("#bb9af7"))
	s.ErrorHeader = r.NewStyle().Foreground(lipgloss.Color("#c0caf5")).Background(lipgloss.Color("#f7768e")).Bold(true).Padding(0, 1).SetString("ERROR")
	s.ErrorDetails = s.Comment
	s.ErrPadding = r.NewStyle().Padding(0, horizontalEdgePadding)
	s.Flag = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#41a6b5", Dark: "#7dcfff"}).Bold(true)
	s.FlagComma = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#89ddff", Dark: "#89ddff"}).SetString(",")
	s.FlagDesc = s.Comment
	s.InlineCode = r.NewStyle().Foreground(lipgloss.Color("#f7768e")).Background(lipgloss.Color("#1a1b26")).Padding(0, 1)
	s.Link = r.NewStyle().Foreground(lipgloss.Color("#7aa2f7")).Underline(true)
	s.Quote = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#bb9af7", Dark: "#bb9af7"})
	s.Pipe = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#7dcfff", Dark: "#7dcfff"})
	s.ConversationList = r.NewStyle().Padding(0, 1)
	s.SHA1 = s.Flag
	s.Timeago = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#565f89", Dark: "#565f89"})
	return s
}

// Action Messages
const defaultAction = "WROTE"

// OutputHeader is the style for the output header
var OutputHeader = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#a9b1d6")). // Foreground color from tokyonight-storm theme
	Background(lipgloss.Color("#9d7cd8")). // Purple color from tokyonight-storm theme
	Bold(true).
	Padding(0, 1).
	MarginRight(1)

// PrintConfirmation prints a confirmation message
func PrintConfirmation(action, content string) {
	if action == "" {
		action = defaultAction
	}
	OutputHeader = OutputHeader.SetString(strings.ToUpper(action))
	fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, OutputHeader.String(), content))
}
