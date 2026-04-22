// Package update implements the update-schemas subcommand for scry.
package update

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/core/schema"
	"github.com/meysam81/scry/internal/logger"
)

var flagURL string

// Command returns the cli.Command for the update-schemas subcommand.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "update-schemas",
		Usage: "Update Schema.org type definitions from the latest release.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "url",
				Value:       schema.DefaultUpdateURL,
				Usage:       "URL to fetch schema definitions from",
				Sources:     cli.EnvVars("SCRY_UPDATE_URL"),
				Destination: &flagURL,
			},
		},
		Action: runUpdate,
	}
}

func runUpdate(ctx context.Context, _ *cli.Command) error {
	l := logger.FromContext(ctx)

	dest := schema.LocalSchemaPath()
	if dest == "" {
		return fmt.Errorf("cannot determine local schema path (no home directory)")
	}

	l.Info().Str("url", flagURL).Str("dest", dest).Msg("fetching schema update")

	version, err := schema.FetchLatest(l, flagURL, dest)
	if err != nil {
		return fmt.Errorf("update schemas: %w", err)
	}

	l.Info().Str("version", version).Str("path", dest).Msg("schema update complete")
	fmt.Printf("Schema definitions updated to version %s\nSaved to: %s\n", version, dest)

	return nil
}
