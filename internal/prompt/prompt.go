package prompt

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

// Abstracts interactive prompting so that process functions can be tested without a TTY.
type Prompter interface {
	// Asks the user a yes/no question.
	//   - Returns true if confirmed, false if declined.
	//   - Returns an error only on interrupt (^C).
	Confirm(label string) (bool, error)

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

func (p *InteractivePrompter) Confirm(label string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
		Default:   "y",
	}
	_, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			return false, fmt.Errorf("interrupted")
		}
		return false, nil
	}
	return true, nil
}

func (p *InteractivePrompter) Input(label, defaultValue string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
	}
	result, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			return "", fmt.Errorf("interrupted")
		}
		return "", nil
	}
	return result, nil
}

// Automatically confirms everything and accepts defaults.
type AutoPrompter struct{}

// Returns a Prompter that auto-confirms and accepts all defaults.
func NewAuto() Prompter {
	return &AutoPrompter{}
}

func (p *AutoPrompter) Confirm(_ string) (bool, error) {
	return true, nil
}

func (p *AutoPrompter) Input(_, defaultValue string) (string, error) {
	return defaultValue, nil
}
