package console

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

// Error pretty-prints given message
func Error(message string) {
	fmt.Println(promptui.Styler(promptui.FGRed)("✗"), message)
}

// Info pretty-prints given message
func Info(message string) {
	fmt.Println(promptui.Styler(promptui.FGYellow)("❯"), message)
}

// Success pretty-prints given message
func Success(message string) {
	fmt.Println(promptui.Styler(promptui.FGGreen)("✔"), message)
}
