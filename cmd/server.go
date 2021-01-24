package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rs/zerolog/log"

	"github.com/githubixx/pinbackup/board"
	redisPool "github.com/githubixx/pinbackup/redis"

	"net/http"
	"strings"
	"time"
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.PersistentFlags().StringVar(&redisHost, "redis-host", "localhost", "Redis host name")
	serverCmd.PersistentFlags().IntVar(&redisPort, "redis-port", 6379, "Redis port")

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.AutomaticEnv()
	viper.BindPFlag("redis-host", serverCmd.PersistentFlags().Lookup("redis-host"))
	viper.BindPFlag("redis-port", serverCmd.PersistentFlags().Lookup("redis-port"))
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Receives board to scrape and put it into a queue",
	Long:  `Receives board to scrape and put it into a queue to be consumed by the scraper process.`,
	Run: func(cmd *cobra.Command, args []string) {
		port := "3333"
		wait := time.Second * 15

		router := board.Routes()

		srv := &http.Server{
			Addr: "0.0.0.0:" + port,
			// Good practice to set timeouts to avoid Slowloris attacks.
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      router,
		}

		log.Debug().
			Str("method", "server/init()").
			Msgf("Redis host: %s:%d", viper.GetString("redis-host"), viper.GetInt("redis-port"))

		redisPool.InitPool(viper.GetString("redis-host"), viper.GetInt("redis-port"))

		// Run our server in a goroutine so that it doesn't block.
		go func() {
			log.Debug().Msgf("Starting HTTP server: %s", port)
			if err := srv.ListenAndServe(); err != nil {
				log.Fatal().Err(err)
			}
		}()

		c := make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
		// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
		signal.Notify(c, os.Interrupt)

		// Block until we receive our signal.
		<-c

		// Create a deadline to wait for.
		ctx, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel()
		// Doesn't block if no connections, but will otherwise wait
		// until the timeout deadline.
		log.Info().
			Str("method", "server/init()").
			Msg("Shutting HTTP server down")

		srv.Shutdown(ctx)
		os.Exit(0)
	},
}
