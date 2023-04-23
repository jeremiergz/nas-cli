package service

import (
	"fmt"
	"io"

	"github.com/cheggaaa/pb/v3/termutil"
	"github.com/manifoldco/promptui"
)

type ConsoleService struct {
	w io.Writer
}

func NewConsoleService(w io.Writer) *ConsoleService {
	service := &ConsoleService{w}

	return service
}

// Pretty-prints given error message
func (s *ConsoleService) Error(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGRed)("✗"), message)
}

// Retrieves the terminal width to use when printing on console
func (s *ConsoleService) GetTerminalWidth() int {
	termWidth, err := termutil.TerminalWidth()
	defaultWidth := 100
	if err != nil {
		termWidth = defaultWidth
	}

	return termWidth
}

// Pretty-prints given info message
func (s *ConsoleService) Info(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGYellow)("❯"), message)
}

// Pretty-prints given success message
func (s *ConsoleService) Success(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGGreen)("✔"), message)
}
