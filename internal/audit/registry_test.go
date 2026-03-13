package audit

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/model"
)

type mockChecker struct {
	name   string
	issues []model.Issue
	calls  atomic.Int64
}

func (m *mockChecker) Name() string { return m.name }
func (m *mockChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	m.calls.Add(1)
	result := make([]model.Issue, len(m.issues))
	for i, iss := range m.issues {
		iss.URL = page.URL
		result[i] = iss
	}
	return result
}

type mockSiteChecker struct {
	mockChecker
	siteIssues []model.Issue
	siteCalls  atomic.Int64
}

func (m *mockSiteChecker) CheckSite(_ context.Context, _ []*model.Page) []model.Issue {
	m.siteCalls.Add(1)
	return m.siteIssues
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry(logger.Nop())
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(r.checkers) != 0 {
		t.Fatalf("expected 0 checkers, got %d", len(r.checkers))
	}
}

func TestRegister(t *testing.T) {
	r := NewRegistry(logger.Nop())
	r.Register(&mockChecker{name: "test"})
	if len(r.checkers) != 1 {
		t.Fatalf("expected 1 checker, got %d", len(r.checkers))
	}
}

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry(logger.Nop())
	if len(r.checkers) != 12 {
		t.Fatalf("expected 12 checkers, got %d", len(r.checkers))
	}
}

func TestRunAll_MultipleCheckersAndPages(t *testing.T) {
	r := NewRegistry(logger.Nop())

	c1 := &mockChecker{
		name: "c1",
		issues: []model.Issue{
			{CheckName: "c1/a", Severity: model.SeverityWarning, Message: "warn"},
		},
	}
	c2 := &mockChecker{
		name: "c2",
		issues: []model.Issue{
			{CheckName: "c2/a", Severity: model.SeverityCritical, Message: "crit"},
		},
	}
	r.Register(c1)
	r.Register(c2)

	pages := []*model.Page{
		{URL: "https://a.com", ContentType: "text/html"},
		{URL: "https://b.com", ContentType: "text/html"},
	}

	issues := r.RunAll(context.Background(), pages)

	// 2 checkers * 2 pages = 4 issues.
	if len(issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(issues))
	}

	// Each checker should be called once per page.
	if c1.calls.Load() != 2 {
		t.Fatalf("expected c1 called 2 times, got %d", c1.calls.Load())
	}
	if c2.calls.Load() != 2 {
		t.Fatalf("expected c2 called 2 times, got %d", c2.calls.Load())
	}

	// Sorted: critical first.
	if issues[0].Severity != model.SeverityCritical {
		t.Fatalf("expected first issue to be critical, got %s", issues[0].Severity)
	}
}

func TestRunAll_SiteChecker(t *testing.T) {
	r := NewRegistry(logger.Nop())

	sc := &mockSiteChecker{
		mockChecker: mockChecker{name: "site"},
		siteIssues: []model.Issue{
			{CheckName: "site/global", Severity: model.SeverityInfo, URL: "https://a.com", Message: "info"},
		},
	}
	r.Register(sc)

	pages := []*model.Page{
		{URL: "https://a.com", ContentType: "text/html"},
	}

	issues := r.RunAll(context.Background(), pages)
	if sc.siteCalls.Load() != 1 {
		t.Fatalf("expected CheckSite called once, got %d", sc.siteCalls.Load())
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
}

func TestRunAll_Sorting(t *testing.T) {
	r := NewRegistry(logger.Nop())

	c := &mockChecker{name: "mixed"}
	r.Register(c)

	// We'll manually test sorting by creating a registry that returns known issues.
	r2 := NewRegistry(logger.Nop())
	infoChecker := &mockChecker{
		name: "info",
		issues: []model.Issue{
			{CheckName: "z/info", Severity: model.SeverityInfo, Message: "info"},
		},
	}
	critChecker := &mockChecker{
		name: "crit",
		issues: []model.Issue{
			{CheckName: "a/crit", Severity: model.SeverityCritical, Message: "crit"},
		},
	}
	warnChecker := &mockChecker{
		name: "warn",
		issues: []model.Issue{
			{CheckName: "m/warn", Severity: model.SeverityWarning, Message: "warn"},
		},
	}
	r2.Register(infoChecker)
	r2.Register(critChecker)
	r2.Register(warnChecker)

	pages := []*model.Page{
		{URL: "https://example.com", ContentType: "text/html"},
	}

	issues := r2.RunAll(context.Background(), pages)
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}
	if issues[0].Severity != model.SeverityCritical {
		t.Errorf("expected first issue critical, got %s", issues[0].Severity)
	}
	if issues[1].Severity != model.SeverityWarning {
		t.Errorf("expected second issue warning, got %s", issues[1].Severity)
	}
	if issues[2].Severity != model.SeverityInfo {
		t.Errorf("expected third issue info, got %s", issues[2].Severity)
	}

	_ = c
}

func TestRunAll_EmptyPages(t *testing.T) {
	r := DefaultRegistry(logger.Nop())
	issues := r.RunAll(context.Background(), nil)
	// Some site-wide checkers may still report issues on nil pages
	// (e.g. security.txt check). Just verify no panic occurs.
	_ = issues
}
