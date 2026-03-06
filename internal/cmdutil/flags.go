package cmdutil

import (
	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/internal/config"
)

// ApplyGlobalOverrides applies the global CLI flags (output, output-file, fail-on)
// from the parent command to the config, only when explicitly set.
func ApplyGlobalOverrides(cmd *cli.Command, cfg *config.Config) {
	if cmd.IsSet("output") {
		cfg.OutputFormat = cmd.String("output")
	}
	if cmd.IsSet("output-file") {
		cfg.OutputFile = cmd.String("output-file")
	}
	if cmd.IsSet("fail-on") {
		cfg.FailOn = cmd.String("fail-on")
	}
}
