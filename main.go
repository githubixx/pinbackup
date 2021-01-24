package main

import (
	"github.com/githubixx/pinbackup/cmd"

	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	cmd.Execute()
}
