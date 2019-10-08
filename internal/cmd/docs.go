package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

var (
	flagDocsMarkdownOutputDir string
)

// docsMarkdownCmd represents the markdown command
var docsMarkdownCmd = &cobra.Command{
	Use:   "markdown",
	Short: "Generates markdown documentation for havener.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		flagDocsMarkdownOutputDir = viper.GetString("output-dir")

		if flagDocsMarkdownOutputDir, err = absolutePath(flagDocsMarkdownOutputDir); err != nil {
			return err
		}
		return doc.GenMarkdownTree(rootCmd, flagDocsMarkdownOutputDir)
	},
}

func init() {
	rootCmd.AddCommand(docsMarkdownCmd)
	docsMarkdownCmd.PersistentFlags().StringP("output-dir", "o", "./docs", "Specifies a directory where markdown documents will be generated.")
	viper.BindPFlags(docsMarkdownCmd.PersistentFlags())
}

func absolutePath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("error getting absolute path: %s: %v", path, err)
	}
	return path, nil
}
