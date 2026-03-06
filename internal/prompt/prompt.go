package prompt

import (
	"fmt"

	"github.com/pterm/pterm"
)

// Abstracts interactive prompting so that process functions can be tested without a TTY.
type Prompter interface {
	// Asks the user a yes/no question.
	//   - Returns true if confirmed, false if declined.
	//   - Returns an error only on interrupt (^C).
	Confirm(label string, defaultValue bool) (bool, error)

	// Asks the user for a text value with a default.
	//   - Returns the entered string (or defaultValue if accepted as-is).
	//   - Returns an error only on interrupt (^C).
	Input(label, defaultValue string) (string, error)
}

// Uses promptui for real terminal interaction.
type InteractivePrompter struct{}

// Returns a Prompter that interacts with the user via the terminal.
func NewInteractive() Prompter {
	return &InteractivePrompter{}
}

func (p *InteractivePrompter) Confirm(label string, defaultValue bool) (bool, error) {
	isInterrupted := false
	result, err := pterm.DefaultInteractiveConfirm.
		WithConfirmText("y").
		WithDefaultText(label).
		WithDefaultValue(defaultValue).
		WithOnInterruptFunc(func() {
			isInterrupted = true
		}).
		Show()
	if isInterrupted {
		pterm.Println()
		return false, fmt.Errorf("interrupted")
	}
	if err != nil {
		return false, fmt.Errorf("failed to handle confirm: %w", err)
	}
	if !result {
		return false, nil
	}
	return true, nil
}

func (p *InteractivePrompter) Input(label, defaultValue string) (string, error) {
	isInterrupted := false
	result, err := pterm.DefaultInteractiveTextInput.
		WithDefaultText(label).
		WithDefaultValue(defaultValue).
		WithOnInterruptFunc(func() {
			isInterrupted = true
		}).
		Show()
	if isInterrupted {
		return "", fmt.Errorf("interrupted")
	}
	if err != nil {
		return "", fmt.Errorf("failed to handle input: %w", err)
	}
	return result, nil
}

// Automatically confirms everything and accepts defaults.
type AutoPrompter struct{}

// Returns a Prompter that auto-confirms and accepts all defaults.
func NewAuto() Prompter {
	return &AutoPrompter{}
}

func (p *AutoPrompter) Confirm(_ string, _ bool) (bool, error) {
	return true, nil
}

func (p *AutoPrompter) Input(_, defaultValue string) (string, error) {
	return defaultValue, nil
}
