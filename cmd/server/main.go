package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is the application version (set during build).
	Version = "dev"

	// Commit is the git commit hash (set during build).
	Commit = "unknown"

	// BuildDate is the build date (set during build).
	BuildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Package Universe Server",
	Long:  `A Go backend server for the Package Universe project.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
