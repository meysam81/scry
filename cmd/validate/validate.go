// Package validate implements the validate subcommand for scry.
package validate

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/internal/config"
)

// Command returns the cli.Command for the validate subcommand.
func Command() *cli.Command {
	return &cli.Command{
		Name:   "validate",
		Usage:  "Validate configuration and show resolved values.",
		Action: runValidate,
	}
}

func runValidate(_ context.Context, cmd *cli.Command) error {
	cfg, err := config.LoadWithFile(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetStyle(table.StyleLight)
	tw.AppendHeader(table.Row{"Setting", "Value"})

	tw.AppendRow(table.Row{"Max Depth", cfg.MaxDepth})
	tw.AppendRow(table.Row{"Max Pages", cfg.MaxPages})
	tw.AppendRow(table.Row{"Concurrency", cfg.Concurrency})
	tw.AppendRow(table.Row{"Request Timeout", cfg.RequestTimeout})
	tw.AppendRow(table.Row{"Rate Limit", cfg.RateLimit})
	tw.AppendRow(table.Row{"User Agent", cfg.UserAgent})
	tw.AppendRow(table.Row{"Respect Robots", cfg.RespectRobots})
	tw.AppendRow(table.Row{"Output Format", cfg.OutputFormat})
	tw.AppendRow(table.Row{"Output File", cfg.OutputFile})
	tw.AppendRow(table.Row{"Fail On", cfg.FailOn})
	tw.AppendRow(table.Row{"Browser Mode", cfg.BrowserMode})
	tw.AppendRow(table.Row{"Browserless URL", cfg.BrowserlessURL})
	tw.AppendRow(table.Row{"Lighthouse Enabled", cfg.LighthouseEnabled})
	tw.AppendRow(table.Row{"Lighthouse Mode", cfg.LighthouseMode})
	tw.AppendRow(table.Row{"PSI Strategy", cfg.PSIStrategy})
	tw.AppendRow(table.Row{"Log Level", cfg.LogLevel})
	tw.AppendRow(table.Row{"Log Format", cfg.LogFormat})

	if len(cfg.IncludePatterns) > 0 {
		tw.AppendRow(table.Row{"Include Patterns", strings.Join(cfg.IncludePatterns, ", ")})
	}
	if len(cfg.ExcludePatterns) > 0 {
		tw.AppendRow(table.Row{"Exclude Patterns", strings.Join(cfg.ExcludePatterns, ", ")})
	}

	tw.Render()

	if _, err := os.Stdout.WriteString("\nConfiguration is valid.\n"); err != nil {
		return fmt.Errorf("write validation result: %w", err)
	}

	return nil
}
