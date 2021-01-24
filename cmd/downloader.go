package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"fmt"
	"strings"

	"github.com/githubixx/pinbackup/downloader"
)

func init() {
	rootCmd.AddCommand(downloaderCmd)

	downloaderCmd.PersistentFlags().StringVar(&redisHost, "redis-host", "localhost", "Redis host name")
	downloaderCmd.PersistentFlags().IntVar(&redisPort, "redis-port", 6379, "Redis port")
	downloaderCmd.PersistentFlags().StringVar(&storageType, "storage-type", "fs", "Currently only fs (filesystem) supported")
	downloaderCmd.PersistentFlags().StringVar(&fsDownloadPath, "fs-download-path", "/tmp", "Directory to store pins")
	downloaderCmd.PersistentFlags().StringVar(&downloadQueue, "download-queue", "download", "Redis queue for pictures to download")

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.AutomaticEnv()
	viper.BindPFlag("redis-host", downloaderCmd.PersistentFlags().Lookup("redis-host"))
	viper.BindPFlag("redis-port", downloaderCmd.PersistentFlags().Lookup("redis-port"))
	viper.BindPFlag("storage-type", downloaderCmd.PersistentFlags().Lookup("storage-type"))
	viper.BindPFlag("fs-download-path", downloaderCmd.PersistentFlags().Lookup("fs-download-path"))
	viper.BindPFlag("download-queue", downloaderCmd.PersistentFlags().Lookup("download-queue"))
}

var downloaderCmd = &cobra.Command{
	Use:   "downloader",
	Short: "Pulls image URLs from queue and fetches pins",
	Long:  `Pulls image URLs from queue and fetches pins.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := downloader.Config{
			RedisHost:      viper.GetString("redis-host"),
			RedisPort:      viper.GetInt("redis-port"),
			StorageType:    viper.GetString("storage-type"),
			FsDownloadPath: viper.GetString("fs-download-path"),
			DownloadQueue:  viper.GetString("download-queue"),
		}
		if err := downloader.StartProcessQueue(&config); err != nil {
			fmt.Println(err)
		}
	},
}
