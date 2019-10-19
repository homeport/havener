package main

import (
	"log"

	"github.com/homeport/havener/internal/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	hvnrCmd := cmd.NewHvnrRootCmd()

	err := doc.GenMarkdownTree(hvnrCmd, ".docs/commands/")
	if err != nil {
		log.Fatal(err)
	}
}
