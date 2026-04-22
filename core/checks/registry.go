// Package audit provides checkers that analyse crawled pages and report issues.
package checks

import (
	"context"
	"runtime"
	"sort"
	"sync"

	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/core/model"
	"github.com/meysam81/scry/core/schema"
)

// Checker runs checks against a single crawled page.
type Checker interface {
	Name() string
	Check(ctx context.Context, page *model.Page) []model.Issue
}

// SiteChecker extends Checker with site-wide analysis.
type SiteChecker interface {
	Checker
	CheckSite(ctx context.Context, pages []*model.Page) []model.Issue
}

// Registry holds all registered checkers and runs them.
type Registry struct {
	checkers []Checker
	log      logger.Logger
}

// NewRegistry returns an empty Registry.
func NewRegistry(l logger.Logger) *Registry {
	return &Registry{log: l}
}

// Register adds a checker to the registry.
func (r *Registry) Register(c Checker) {
	r.checkers = append(r.checkers, c)
}

// CheckerNames returns the canonical Name() of every registered checker, in
// registration order.
func (r *Registry) CheckerNames() []string {
	out := make([]string, len(r.checkers))
	for i, c := range r.checkers {
		out[i] = c.Name()
	}
	return out
}

// DefaultRegistry creates a Registry with all built-in checkers registered.
// If schemaPath is empty, it falls back to the default local schema path.
func DefaultRegistry(l logger.Logger, schemaPath string) *Registry {
	r := NewRegistry(l)
	r.Register(NewSEOChecker())
	r.Register(NewHealthChecker())
	r.Register(NewImageChecker())
	r.Register(NewLinkChecker())
	r.Register(NewPerformanceChecker())
	r.Register(NewStructuredDataChecker())
	r.Register(NewSecurityChecker())
	r.Register(NewAccessibilityChecker())
	r.Register(NewHreflangChecker())
	r.Register(NewExternalLinkChecker())
	r.Register(NewTLSChecker())
	if schemaPath == "" {
		schemaPath = schema.LocalSchemaPath()
	}
	r.Register(NewDeepStructuredDataChecker(schema.Load(schemaPath)))
	return r
}

// RunAll runs every registered checker against all pages and returns sorted issues.
//
// Per-page checks are executed concurrently using a worker pool sized to
// runtime.NumCPU(). After all page checks complete, any SiteChecker's
// CheckSite method is called. Results are sorted by severity (critical first),
// then URL, then CheckName.
func (r *Registry) RunAll(ctx context.Context, pages []*model.Page) []model.Issue {
	setAuditLogger(r.log)

	var (
		mu     sync.Mutex
		issues []model.Issue
	)

	workers := max(runtime.NumCPU(), 1)

	type job struct {
		checker Checker
		page    *model.Page
	}

	jobs := make(chan job, len(pages)*len(r.checkers))
	var wg sync.WaitGroup

	// Start workers.
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if ctx.Err() != nil {
					return
				}
				found := j.checker.Check(ctx, j.page)
				if len(found) > 0 {
					mu.Lock()
					issues = append(issues, found...)
					mu.Unlock()
				}
			}
		}()
	}

	// Enqueue jobs.
	for _, c := range r.checkers {
		for _, p := range pages {
			jobs <- job{checker: c, page: p}
		}
	}
	close(jobs)
	wg.Wait()

	// Run site-wide checks.
	for _, c := range r.checkers {
		if sc, ok := c.(SiteChecker); ok {
			found := sc.CheckSite(ctx, pages)
			issues = append(issues, found...)
		}
	}

	// Free cached HTML documents after all checks complete.
	clearDocCache()

	// Sort: severity desc → URL asc → CheckName asc.
	sort.Slice(issues, func(i, j int) bool {
		si, sj := issues[i].Severity.Level(), issues[j].Severity.Level()
		if si != sj {
			return si > sj
		}
		if issues[i].URL != issues[j].URL {
			return issues[i].URL < issues[j].URL
		}
		return issues[i].CheckName < issues[j].CheckName
	})

	return issues
}
