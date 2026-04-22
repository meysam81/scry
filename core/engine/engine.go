// Package engine is the public orchestrator used by every frontend
// (CLI, WASM, tests). It composes check registries, rule evaluators,
// and schema validators behind a single narrow API so that the business
// logic remains one implementation with many callers.
package engine

import (
	"context"
	"fmt"

	"github.com/meysam81/scry/core/checks"
	"github.com/meysam81/scry/core/model"
	"github.com/meysam81/scry/core/rules"
	"github.com/meysam81/scry/core/schema"
	"github.com/meysam81/scry/internal/logger"
	"gopkg.in/yaml.v3"
)

// Options configures a new Engine. Zero value is valid and produces an
// engine with the default built-in check set and embedded schema registry.
type Options struct {
	// Logger is used for check-time diagnostics. Defaults to a no-op logger.
	Logger logger.Logger

	// SchemaRegistry overrides the embedded Schema.org registry. When nil,
	// the engine loads the registry bundled via go:embed.
	SchemaRegistry *schema.Registry

	// RulesYAML provides optional user-defined CEL rules. Empty string skips
	// rule evaluation entirely.
	RulesYAML string

	// DisableChecks names built-in checkers to skip (matched by Name()).
	// Use ListCheckNames() for valid identifiers.
	DisableChecks []string

	// IncludeDeepStructuredData toggles the JSON-LD Schema.org deep validator.
	// Defaults to true.
	IncludeDeepStructuredData bool
}

// Engine is the shared audit surface. It is safe to reuse across goroutines;
// every method is a pure function of its arguments once the engine is built.
type Engine struct {
	registry *checks.Registry
	rules    *rules.Engine
	log      logger.Logger
}

// New builds an Engine from the provided options. It fails only when the
// rules YAML or schema registry is malformed.
func New(opts Options) (*Engine, error) {
	log := opts.Logger
	// logger.Logger wraps a zerolog.Logger which is not comparable, so we
	// cannot zero-check it. Callers that want silence pass logger.Nop().
	_ = log

	reg := checks.NewRegistry(log)

	skip := make(map[string]bool, len(opts.DisableChecks))
	for _, n := range opts.DisableChecks {
		skip[n] = true
	}

	register := func(c checks.Checker) {
		if !skip[c.Name()] {
			reg.Register(c)
		}
	}
	register(checks.NewSEOChecker())
	register(checks.NewHealthChecker())
	register(checks.NewImageChecker())
	register(checks.NewLinkChecker())
	register(checks.NewPerformanceChecker())
	register(checks.NewStructuredDataChecker())
	register(checks.NewSecurityChecker())
	register(checks.NewAccessibilityChecker())
	register(checks.NewHreflangChecker())
	register(checks.NewExternalLinkChecker())
	register(checks.NewTLSChecker())

	if opts.IncludeDeepStructuredData || opts.SchemaRegistry != nil {
		schemaReg := opts.SchemaRegistry
		if schemaReg == nil {
			loaded, err := schema.LoadEmbedded()
			if err != nil {
				return nil, fmt.Errorf("load embedded schemas: %w", err)
			}
			schemaReg = loaded
		}
		register(checks.NewDeepStructuredDataChecker(schemaReg))
	}

	e := &Engine{registry: reg, log: log}

	if opts.RulesYAML != "" {
		var rf rules.RuleFile
		if err := yaml.Unmarshal([]byte(opts.RulesYAML), &rf); err != nil {
			return nil, fmt.Errorf("parse rules yaml: %w", err)
		}
		rulesEng, err := rules.NewEngine(rf.Rules, log)
		if err != nil {
			return nil, fmt.Errorf("compile rules: %w", err)
		}
		e.rules = rulesEng
	}

	return e, nil
}

// AuditPage runs every registered check against a single page and returns
// every issue found, sorted deterministically by severity then URL.
func (e *Engine) AuditPage(ctx context.Context, page *model.Page) []model.Issue {
	return e.AuditSite(ctx, []*model.Page{page})
}

// AuditSite runs every registered check (page-level and site-level) across
// all pages and returns the aggregated, sorted issues list.
func (e *Engine) AuditSite(ctx context.Context, pages []*model.Page) []model.Issue {
	issues := e.registry.RunAll(ctx, pages)
	if e.rules != nil {
		for _, p := range pages {
			issues = append(issues, e.rules.Evaluate(ctx, p)...)
		}
	}
	return issues
}

// CheckNames returns the names of every checker currently registered, in
// registration order. Useful for building UI filters.
func (e *Engine) CheckNames() []string {
	return e.registry.CheckerNames()
}

// ListAllCheckNames returns every built-in checker's canonical name without
// constructing an engine. Handy for CLI flag validation and UI prep.
func ListAllCheckNames() []string {
	return []string{
		(&checks.SEOChecker{}).Name(),
		(&checks.HealthChecker{}).Name(),
		(&checks.ImageChecker{}).Name(),
		(&checks.LinkChecker{}).Name(),
		(&checks.PerformanceChecker{}).Name(),
		(&checks.StructuredDataChecker{}).Name(),
		(&checks.SecurityChecker{}).Name(),
		(&checks.AccessibilityChecker{}).Name(),
		(&checks.HreflangChecker{}).Name(),
		(&checks.ExternalLinkChecker{}).Name(),
		(&checks.TLSChecker{}).Name(),
	}
}
