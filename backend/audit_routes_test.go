package main

// Guard for BOG-48.
//
// "Every mutating admin route writes one audit record" only holds if every
// admin route goes through the one wrapper that writes it. A reviewer cannot
// hold that by reading thirty handlers across twelve packages, and the next
// route is always the one that gets forgotten - so this test re-derives it from
// main.go on every `go test` run, the way boot_ddl_test.go does for DDL.
//
// Three things are checked, all without a database:
//
//  1. Every `/api/admin/…` path in main.go is registered through the `admin(…)`
//     helper, not through a bare mux.HandleFunc/mux.Handle.
//  2. Every route the helper registers has a resource declared in
//     internal/audit/resources.go.
//  3. The registry has no entry for a route main.go does not register, so a
//     deleted route leaves no stale declaration behind.
//
// Together with internal/audit's own tests - which assert exactly one record
// per route per write method - route coverage is a property of the wiring.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"

	"backend/internal/audit"
)

const adminPrefix = "/api/admin/"

// parseMain reads main.go and returns the routes registered through the
// `admin(…)` helper and the /api/admin paths registered any other way.
func parseMain(t *testing.T) (viaHelper []string, direct []string) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse main.go: %v", err)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		path, ok := stringLit(call.Args[0])
		if !ok || !strings.HasPrefix(path, adminPrefix) {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			if fn.Name == "admin" {
				viaHelper = append(viaHelper, path)
				return true
			}
			direct = append(direct, path+" (via "+fn.Name+")")
		case *ast.SelectorExpr:
			direct = append(direct, path+" (via "+exprName(fn)+")")
		}
		return true
	})
	sort.Strings(viaHelper)
	sort.Strings(direct)
	return viaHelper, direct
}

func TestEveryAdminRouteGoesThroughTheAuditWrapper(t *testing.T) {
	viaHelper, direct := parseMain(t)

	if len(viaHelper) == 0 {
		t.Fatal("found no admin(…) registrations in main.go; the guard is not reading the right file")
	}
	for _, d := range direct {
		t.Errorf("admin route registered without the audit wrapper: %s - register it with admin(path, handler)", d)
	}
}

func TestEveryRegisteredAdminRouteHasAnAuditResource(t *testing.T) {
	viaHelper, _ := parseMain(t)
	for _, route := range viaHelper {
		if _, ok := audit.Lookup(route); !ok {
			t.Errorf("route %s has no resource in internal/audit/resources.go; audit.Wrap would panic at startup", route)
		}
	}
}

func TestAuditRegistryHasNoStaleRoutes(t *testing.T) {
	viaHelper, _ := parseMain(t)
	registered := map[string]bool{}
	for _, r := range viaHelper {
		registered[r] = true
	}
	for _, route := range audit.Routes() {
		if !registered[route] {
			t.Errorf("internal/audit/resources.go declares %s, which main.go no longer registers", route)
		}
	}
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

func exprName(e *ast.SelectorExpr) string {
	if x, ok := e.X.(*ast.Ident); ok {
		return x.Name + "." + e.Sel.Name
	}
	return e.Sel.Name
}
