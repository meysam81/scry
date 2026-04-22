//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"syscall/js"

	"github.com/meysam81/scry/internal/audit"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/model"
)

func buildRegistry() *audit.Registry {
	l := logger.Nop()
	r := audit.NewRegistry(l)
	// Register the subset of checkers that do not depend on external schema files.
	r.Register(audit.NewSEOChecker())
	r.Register(audit.NewHealthChecker())
	r.Register(audit.NewImageChecker())
	r.Register(audit.NewLinkChecker())
	r.Register(audit.NewPerformanceChecker())
	r.Register(audit.NewStructuredDataChecker())
	r.Register(audit.NewSecurityChecker())
	r.Register(audit.NewAccessibilityChecker())
	r.Register(audit.NewHreflangChecker())
	return r
}

func scryAuditPage(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return map[string]any{"error": "scryAuditPage(pageJSON) requires one argument"}
	}
	raw := args[0].String()

	var page model.Page
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		return map[string]any{"error": "invalid page JSON: " + err.Error()}
	}
	if len(args) >= 2 && args[1].Type() == js.TypeString {
		page.Body = []byte(args[1].String())
	}

	reg := buildRegistry()
	issues := reg.RunAll(context.Background(), []*model.Page{&page})

	out, err := json.Marshal(issues)
	if err != nil {
		return map[string]any{"error": "marshal issues: " + err.Error()}
	}
	return string(out)
}

func main() {
	js.Global().Set("scryAuditPage", js.FuncOf(scryAuditPage))
	// Keep the Go runtime alive so the exported function remains callable.
	select {}
}
