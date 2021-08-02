package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/githubixx/pinbackup/importer"
)

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.PersistentFlags().StringVar(&redisHost, "redis-host", "localhost", "Redis host name")
	importCmd.PersistentFlags().IntVar(&redisPort, "redis-port", 6379, "Redis port")
	importCmd.PersistentFlags().StringVar(&importDir, "dir", "/tmp", "Directory where pins are stored")
	importCmd.PersistentFlags().StringVar(&userDatabaseName, "database", "user", "Database name where users and pins are stored")

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.AutomaticEnv()
	viper.BindPFlag("redis-host", importCmd.PersistentFlags().Lookup("redis-host"))
	viper.BindPFlag("redis-port", importCmd.PersistentFlags().Lookup("redis-port"))
	viper.BindPFlag("importDir", importCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("userDatabaseName", importCmd.PersistentFlags().Lookup("database"))
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Imports users and boards into Redis database",
	Long:  `Imports users and boards into Redis database. The directory structure must be user/board.`,
	Run: func(cmd *cobra.Command, args []string) {

		ctx := context.Background()

		config := importer.Config{
			ImportDir:        viper.GetString("importDir"),
			UserDatabaseName: viper.GetString("database"),
		}
		if err := importer.StartImport(ctx, &config); err != nil {
			fmt.Println(err)
		}
	},
}
