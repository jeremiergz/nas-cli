package console

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

// Success pretty-prints given message
func Success(message string) {
	fmt.Println(promptui.Styler(promptui.FGGreen)("✔"), message)
}
