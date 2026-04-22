//go:build js && wasm

// WASM entry point for the scry browser extension. Exposes a narrow,
// JSON-over-string surface that mirrors core/engine so the extension
// calls into the same audit pipeline the CLI uses.
package main

import (
	"context"
	"encoding/json"
	"syscall/js"

	"github.com/meysam81/scry/core/engine"
	"github.com/meysam81/scry/core/model"
)

var defaultEngine *engine.Engine

func init() {
	e, err := engine.New(engine.Options{IncludeDeepStructuredData: true})
	if err != nil {
		// Fall back to a schema-less engine if the embedded registry is
		// somehow corrupt. A degraded audit beats a dead extension.
		e, _ = engine.New(engine.Options{IncludeDeepStructuredData: false})
	}
	defaultEngine = e
}

// resp is the canonical envelope every exported function returns.
// The extension validates { ok, data?, error? } with Zod.
type resp struct {
	OK    bool `json:"ok"`
	Data  any  `json:"data,omitempty"`
	Error any  `json:"error,omitempty"`
}

func ok(data any) string {
	b, _ := json.Marshal(resp{OK: true, Data: data})
	return string(b)
}

func fail(msg string) string {
	b, _ := json.Marshal(resp{OK: false, Error: msg})
	return string(b)
}

// auditPageInput is the JSON contract with the extension. Body is sent
// separately because model.Page excludes Body from JSON marshaling.
type auditPageInput struct {
	Page model.Page `json:"page"`
	Body string     `json:"body"`
}

func auditPage(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return fail("scryAuditPage expects one string argument (JSON)")
	}
	var in auditPageInput
	if err := json.Unmarshal([]byte(args[0].String()), &in); err != nil {
		return fail("invalid input JSON: " + err.Error())
	}
	if in.Body != "" {
		in.Page.Body = []byte(in.Body)
	}
	issues := defaultEngine.AuditPage(context.Background(), &in.Page)
	if issues == nil {
		issues = []model.Issue{}
	}
	return ok(map[string]any{
		"issues": issues,
		"url":    in.Page.URL,
	})
}

func listChecks(_ js.Value, _ []js.Value) any {
	return ok(map[string]any{"checks": engine.ListAllCheckNames()})
}

func version(_ js.Value, _ []js.Value) any {
	return ok(map[string]any{
		"engine": "scry-wasm",
		"api":    1,
	})
}

func main() {
	js.Global().Set("scryAuditPage", js.FuncOf(auditPage))
	js.Global().Set("scryListChecks", js.FuncOf(listChecks))
	js.Global().Set("scryVersion", js.FuncOf(version))
	select {}
}
