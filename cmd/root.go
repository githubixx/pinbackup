package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	redisHost            string
	redisPort            int
	loginName            string
	loginPassword        string
	storageType          string
	fsDownloadPath       string
	selectorPreviewPins  string
	boardsQueue          string
	downloadQueue        string
	chromeWsDebuggerHost string
)

var rootCmd = &cobra.Command{
	Use:   "pinbackup",
	Short: "Backup your boards at Pinterest",
	Long:  `Backup your boards at Pinterest.`,
}

// Execute this command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
