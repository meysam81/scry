package rules

import (
	"context"

	"github.com/meysam81/scry/core/model"
)

// RuleChecker adapts a rules [Engine] to the [audit.Checker] interface.
type RuleChecker struct {
	engine *Engine
}

// NewRuleChecker wraps an Engine as an audit.Checker.
func NewRuleChecker(engine *Engine) *RuleChecker {
	return &RuleChecker{engine: engine}
}

// Name returns the checker identifier.
func (c *RuleChecker) Name() string { return "rules" }

// Check evaluates all CEL rules against the given page.
func (c *RuleChecker) Check(ctx context.Context, page *model.Page) []model.Issue {
	return c.engine.Evaluate(ctx, page)
}
