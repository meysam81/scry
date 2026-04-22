// Package rules provides a CEL-based rule engine for custom audit checks.
//
// Rules are defined in YAML files and compiled once into efficient CEL programs.
// Each rule specifies a boolean condition that is evaluated against a crawled page;
// when the condition is true, the engine produces an [model.Issue].
package rules

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/core/model"
	"gopkg.in/yaml.v3"
)

// Rule describes a single user-defined audit rule.
type Rule struct {
	Name      string `yaml:"name"`
	Severity  string `yaml:"severity"`  // critical, warning, info
	Match     string `yaml:"match"`     // URL glob pattern; empty or "*" matches everything
	Condition string `yaml:"condition"` // CEL boolean expression
	Message   string `yaml:"message"`   // human-readable issue message
}

// RuleFile is the top-level YAML structure for a rules file.
type RuleFile struct {
	Rules []Rule `yaml:"rules"`
}

// Engine holds pre-compiled CEL rules ready for evaluation.
type Engine struct {
	rules []compiledRule
	log   logger.Logger
}

// compiledRule pairs a source Rule with its compiled CEL program and URL matcher.
type compiledRule struct {
	rule    Rule
	program cel.Program
	match   func(string) bool
}

// LoadRuleFile reads and parses a YAML rule file from disk.
func LoadRuleFile(path string) (*RuleFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading rule file: %w", err)
	}

	var rf RuleFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("parsing rule file %s: %w", path, err)
	}
	return &rf, nil
}

// NewEngine compiles the given rules into CEL programs and returns an Engine.
// An error is returned if any rule has an invalid CEL expression or glob pattern.
func NewEngine(rules []Rule, l logger.Logger) (*Engine, error) {
	env, err := newCELEnv()
	if err != nil {
		return nil, fmt.Errorf("creating CEL environment: %w", err)
	}

	compiled := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		prog, err := compileRule(env, r)
		if err != nil {
			return nil, fmt.Errorf("compiling rule %q: %w", r.Name, err)
		}

		matcher, err := buildMatcher(r.Match)
		if err != nil {
			return nil, fmt.Errorf("compiling match pattern for rule %q: %w", r.Name, err)
		}

		compiled = append(compiled, compiledRule{
			rule:    r,
			program: prog,
			match:   matcher,
		})
	}

	return &Engine{rules: compiled, log: l}, nil
}

// Evaluate runs all compiled rules against the given page and returns any issues.
func (e *Engine) Evaluate(_ context.Context, page *model.Page) []model.Issue {
	activation := pageToMap(page)
	input := map[string]any{"page": activation}

	var issues []model.Issue
	for _, cr := range e.rules {
		if !cr.match(page.URL) {
			continue
		}

		out, _, err := cr.program.Eval(input)
		if err != nil {
			// Evaluation errors are treated as non-matching; the rule
			// simply does not fire. This keeps the engine resilient to
			// edge cases in page data.
			e.log.Debug().Err(err).Str("rule", cr.rule.Name).Str("url", page.URL).Msg("CEL eval error")
			continue
		}

		if fired, ok := out.Value().(bool); ok && fired {
			issues = append(issues, model.Issue{
				CheckName: cr.rule.Name,
				Severity:  model.SeverityFromString(cr.rule.Severity),
				Message:   cr.rule.Message,
				URL:       page.URL,
			})
		}
	}
	return issues
}

// RuleCount returns the number of compiled rules in the engine.
func (e *Engine) RuleCount() int {
	return len(e.rules)
}

// --- internal helpers --------------------------------------------------------

// newCELEnv builds the CEL environment with a single top-level "page" variable.
func newCELEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("page", cel.MapType(cel.StringType, cel.DynType)),
		ext.Strings(),
	)
}

// compileRule parses, checks, and compiles a single CEL expression.
func compileRule(env *cel.Env, r Rule) (cel.Program, error) {
	ast, iss := env.Parse(r.Condition)
	if iss.Err() != nil {
		return nil, fmt.Errorf("parse: %w", iss.Err())
	}

	checked, iss := env.Check(ast)
	if iss.Err() != nil {
		return nil, fmt.Errorf("type-check: %w", iss.Err())
	}

	prog, err := env.Program(checked)
	if err != nil {
		return nil, fmt.Errorf("program: %w", err)
	}
	return prog, nil
}

// buildMatcher returns a function that reports whether a URL matches the rule's
// glob pattern. An empty or "*" pattern matches all URLs.
func buildMatcher(pattern string) (func(string) bool, error) {
	if pattern == "" || pattern == "*" {
		return func(string) bool { return true }, nil
	}

	// Validate the pattern.
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, err
	}

	return func(url string) bool {
		ok, err := filepath.Match(pattern, url)
		return err == nil && ok
	}, nil
}

// pageToMap converts a Page into the flat map exposed to CEL expressions.
func pageToMap(page *model.Page) map[string]any {
	return map[string]any{
		"url":               page.URL,
		"status_code":       int64(page.StatusCode),
		"content_type":      page.ContentType,
		"headers":           flattenHeaders(page.Headers),
		"body":              string(page.Body),
		"links":             stringSliceToAny(page.Links),
		"assets":            stringSliceToAny(page.Assets),
		"depth":             int64(page.Depth),
		"fetch_duration_ms": int64(page.FetchDuration / time.Millisecond),
		"in_sitemap":        page.InSitemap,
	}
}

// flattenHeaders converts http.Header to map[string]string with lowercased keys
// and the first value for each header.
func flattenHeaders(h http.Header) map[string]string {
	if h == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			out[strings.ToLower(k)] = v[0]
		}
	}
	return out
}

// stringSliceToAny converts []string to []any so CEL can treat it as a list.
func stringSliceToAny(ss []string) []any {
	if ss == nil {
		return []any{}
	}
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
