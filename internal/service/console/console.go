package console

import (
	"fmt"
	"io"

	"github.com/cheggaaa/pb/v3/termutil"
	"github.com/manifoldco/promptui"
)

type Service struct {
	w io.Writer
}

func New(w io.Writer) *Service {
	return &Service{w}
}

// Pretty-prints given error message
func (s *Service) Error(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGRed)("✗"), message)
}

// Retrieves the terminal width to use when printing on console
func (s *Service) GetTerminalWidth() int {
	termWidth, err := termutil.TerminalWidth()
	defaultWidth := 100
	if err != nil {
		termWidth = defaultWidth
	}

	return termWidth
}

// Pretty-prints given info message
func (s *Service) Info(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGYellow)("❯"), message)
}

// Pretty-prints given success message
func (s *Service) Success(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGGreen)("✔"), message)
}
