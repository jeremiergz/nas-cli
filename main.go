package main

import (
	"fmt"
	"os"

	"github.com/jeremiergz/nas-cli/cmd"
)

func main() {
	if err := cmd.Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
