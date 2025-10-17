package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "commiter",
	Short: "AI-powered commit message generator",
	Long:  `A CLI tool that generates commit messages using AI based on your staged git changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// This will be replaced with Bubbletea TUI
		return runTUI()
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize with OpenRouter API key",
	Long:  `Set up the API key and default configurations for commiter.`,
	Run: func(cmd *cobra.Command, args []string) {
		runInit()
	},
}

var simpleCommitCmd = &cobra.Command{
	Use:   "simple-commit",
	Short: "Generate and commit with a simple message",
	Long:  `Automatically generates a short commit message and commits the staged changes.`,
	Run: func(cmd *cobra.Command, args []string) {
		message := generateCommitMessage(true)
		if strings.HasPrefix(message, "Error") || strings.HasPrefix(message, "No") || strings.HasPrefix(message, "API") {
			fmt.Println(message)
			os.Exit(1)
		}
		result := performCommit(message, true)
		fmt.Println(result)
	},
}

var detailedCommitCmd = &cobra.Command{
	Use:   "detailed-commit",
	Short: "Generate and commit with a detailed message",
	Long:  `Automatically generates a detailed commit message with descriptions and commits the staged changes.`,
	Run: func(cmd *cobra.Command, args []string) {
		message := generateCommitMessage(false)
		if strings.HasPrefix(message, "Error") || strings.HasPrefix(message, "No") || strings.HasPrefix(message, "API") {
			fmt.Println(message)
			os.Exit(1)
		}
		result := performCommit(message, false)
		fmt.Println(result)
	},
}

var stashCmd = &cobra.Command{
	Use:   "stash",
	Short: "Generate message and stash changes",
	Long:  `Automatically generates a stash message and stashes the current changes.`,
	Run: func(cmd *cobra.Command, args []string) {
		message := generateStashMessage()
		if strings.HasPrefix(message, "Error") || strings.HasPrefix(message, "No") || strings.HasPrefix(message, "API") {
			fmt.Println(message)
			os.Exit(1)
		}
		result := performStash(message)
		fmt.Println(result)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(simpleCommitCmd)
	rootCmd.AddCommand(detailedCommitCmd)
	rootCmd.AddCommand(stashCmd)
}