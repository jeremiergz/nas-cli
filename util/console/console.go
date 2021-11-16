package console

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

// Pretty-prints given error message
func Error(message string) {
	fmt.Println(promptui.Styler(promptui.FGRed)("✗"), message)
}

// Pretty-prints given info message
func Info(message string) {
	fmt.Println(promptui.Styler(promptui.FGYellow)("❯"), message)
}

// Pretty-prints given success message
func Success(message string) {
	fmt.Println(promptui.Styler(promptui.FGGreen)("✔"), message)
}
