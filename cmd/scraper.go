package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"fmt"
	"strings"

	"github.com/githubixx/pinbackup/scraper"
)

func init() {
	rootCmd.AddCommand(scraperCmd)

	scraperCmd.PersistentFlags().StringVar(&redisHost, "redis-host", "localhost", "Redis host name")
	scraperCmd.PersistentFlags().IntVar(&redisPort, "redis-port", 6379, "Redis port")
	scraperCmd.PersistentFlags().StringVar(&loginName, "login-name", "", "Pinterest login name")
	scraperCmd.PersistentFlags().StringVar(&loginPassword, "login-password", "", "Pinterest login password")
	scraperCmd.PersistentFlags().StringVar(&selectorPreviewPins, "selector-preview-pins", "", "CSS selector for preview pins")
	scraperCmd.PersistentFlags().StringVar(&boardsQueue, "boards-queue", "boards", "Redis queue for boards to download")
	scraperCmd.PersistentFlags().StringVar(&downloadQueue, "download-queue", "download", "Redis queue for pictures to download")
	scraperCmd.PersistentFlags().StringVar(&chromeWsDebuggerHost, "chrome-ws-debugger-host", "localhost", "Chrome WebSocket debugger host")

	scraperCmd.MarkFlagRequired("loginName")
	scraperCmd.MarkFlagRequired("loginPassword")

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.AutomaticEnv()
	viper.BindPFlag("redis-host", scraperCmd.PersistentFlags().Lookup("redis-host"))
	viper.BindPFlag("redis-port", scraperCmd.PersistentFlags().Lookup("redis-port"))
	viper.BindPFlag("login-name", scraperCmd.PersistentFlags().Lookup("login-name"))
	viper.BindPFlag("login-password", scraperCmd.PersistentFlags().Lookup("login-password"))
	viper.BindPFlag("selector-preview-pins", scraperCmd.PersistentFlags().Lookup("selector-preview-pins"))
	viper.BindPFlag("boards-queue", scraperCmd.PersistentFlags().Lookup("boards-queue"))
	viper.BindPFlag("download-queue", scraperCmd.PersistentFlags().Lookup("download-queue"))
	viper.BindPFlag("chrome-ws-debugger-host", scraperCmd.PersistentFlags().Lookup("chrome-ws-debugger-host"))
}

var scraperCmd = &cobra.Command{
	Use:   "scraper",
	Short: "Pulls scrape requests from queue and fetches URLs of pins",
	Long:  `Pulls scrape requests from queue and fetches URLs of pins.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := scraper.Config{
			RedisHost:            viper.GetString("redis-host"),
			RedisPort:            viper.GetInt("redis-port"),
			LoginName:            viper.GetString("login-name"),
			LoginPassword:        viper.GetString("login-password"),
			SelectorPreviewPins:  viper.GetString("selector-preview-pins"),
			BoardsQueue:          viper.GetString("boards-queue"),
			DownloadQueue:        viper.GetString("download-queue"),
			ChromeWsDebuggerHost: viper.GetString("chrome-ws-debugger-host"),
		}
		if err := scraper.StartProcessQueue(&config); err != nil {
			fmt.Println(err)
		}
	},
}
