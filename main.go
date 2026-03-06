// Package main is the entry point for the scry CLI.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/cmd/check"
	"github.com/meysam81/scry/cmd/crawl"
	"github.com/meysam81/scry/cmd/lighthouse"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Force exit on second signal.
	go func() {
		<-ctx.Done()
		// First signal received, context cancelled, graceful shutdown in progress.
		// If a second signal arrives, force exit.
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		os.Exit(1)
	}()

	root := &cli.Command{
		Name:  "scry",
		Usage: "A fast, developer-friendly website auditing tool.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Value:   "info",
				Usage:   "log level: debug|info|warn|error",
			},
			&cli.StringFlag{
				Name:  "log-format",
				Value: "pretty",
				Usage: "log format: pretty|json",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "terminal",
				Usage:   "output format(s), comma-separated",
			},
			&cli.StringFlag{
				Name:  "output-file",
				Value: "",
				Usage: "path for non-terminal output",
			},
			&cli.StringFlag{
				Name:  "fail-on",
				Value: "",
				Usage: "severity threshold for non-zero exit",
			},
			&cli.StringFlag{
				Name:  "config",
				Value: "",
				Usage: "path to scry.yml config file",
			},
		},
		Before: setupLogger,
		Commands: []*cli.Command{
			crawl.Command(),
			check.Command(),
			lighthouse.Command(),
		},
	}

	if err := root.Run(ctx, os.Args); err != nil {
		log.Fatal().Err(err).Msg("fatal error")
	}
}

// setupLogger configures zerolog based on the --log-level and --log-format flags.
func setupLogger(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	format := cmd.String("log-format")
	level := cmd.String("log-level")

	var logger zerolog.Logger

	switch format {
	case "json":
		logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	default:
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	}

	switch level {
	case "debug":
		logger = logger.Level(zerolog.DebugLevel)
	case "warn":
		logger = logger.Level(zerolog.WarnLevel)
	case "error":
		logger = logger.Level(zerolog.ErrorLevel)
	default:
		logger = logger.Level(zerolog.InfoLevel)
	}

	log.Logger = logger

	return ctx, nil
}
