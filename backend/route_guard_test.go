package main

// Every admin route is behind RequireAdmin - proved against the real mux, for
// the whole route table, derived from the source (BOG-55).
//
// The old lamsza suite spot-checked four paths by hand
// (`TestAdminRoutesRequireAdmin`: entries, settings, events, news_feeds). That
// is the kind of list nobody updates: a route added later is unguarded and
// every test stays green. So this file reads the route table out of main.go's
// AST instead, in the style of boot_ddl_test.go, and sweeps all of it.
//
// None of it needs a database. RequireAdmin calls auth.UserFromRequest, which
// returns "no session" before it reaches Postgres when the cookie is absent, so
// the 401 sweep is the one part of this suite that runs in CI.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// route is one mux registration as main.go spells it.
type route struct {
	pattern string
	// handlerSrc is the second argument rendered back to source, so the whole
	// wrapper chain is visible. It is matched as text rather than as one
	// identifier so an extra layer - `audit(admin(h))`, say - still reads as
	// gated instead of silently reading as open.
	handlerSrc string
	adminGated bool
	pos        string
}

// parseRoutes returns every route registered inside newMux(): each
// mux.HandleFunc / mux.Handle call with a literal pattern, and each call of the
// `admin(path, handler)` helper (BOG-48), which wraps the handler in
// ApplyCORS, RequireAdmin and audit.Wrap and registers it itself. It reads the
// source rather than introspecting *http.ServeMux because the standard mux
// does not expose its patterns, and a hand-kept list is the failure mode this
// test exists to remove.
func parseRoutes(t *testing.T) []route {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("parse main.go: %v", err)
	}

	var body *ast.BlockStmt
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "newMux" {
			body = fn.Body
		}
	}
	if body == nil {
		t.Fatal("no func newMux in main.go; the route table moved and this guard is looking at the wrong place")
	}

	var routes []route
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		if fn, ok := call.Fun.(*ast.Ident); ok && fn.Name == "admin" {
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				t.Errorf("%s: admin() route pattern is not a string literal; this guard cannot read it",
					fset.Position(call.Args[0].Pos()))
				return true
			}
			pattern, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Errorf("%s: unquote route pattern: %v", fset.Position(lit.Pos()), err)
				return true
			}
			var rendered bytes.Buffer
			if err := printer.Fprint(&rendered, fset, call); err != nil {
				t.Errorf("%s: render admin() call: %v", fset.Position(call.Pos()), err)
				return true
			}
			routes = append(routes, route{
				pattern:    pattern,
				handlerSrc: rendered.String(),
				adminGated: true,
				pos:        fset.Position(call.Pos()).String(),
			})
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok || recv.Name != "mux" {
			return true
		}
		if sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle" {
			return true
		}
		// The admin() helper's own body registers `route`, its parameter. That
		// call is the mechanism, not a route; the helper's call sites are
		// collected above.
		if id, ok := call.Args[0].(*ast.Ident); ok && id.Name == "route" {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			t.Errorf("%s: route pattern is not a string literal; this guard cannot read it",
				fset.Position(call.Args[0].Pos()))
			return true
		}
		pattern, err := strconv.Unquote(lit.Value)
		if err != nil {
			t.Errorf("%s: unquote route pattern: %v", fset.Position(lit.Pos()), err)
			return true
		}

		var rendered bytes.Buffer
		if err := printer.Fprint(&rendered, fset, call.Args[1]); err != nil {
			t.Errorf("%s: render handler expression: %v", fset.Position(call.Args[1].Pos()), err)
			return true
		}

		r := route{
			pattern:    pattern,
			handlerSrc: rendered.String(),
			pos:        fset.Position(call.Pos()).String(),
		}
		r.adminGated = strings.Contains(r.handlerSrc, "admin(") ||
			strings.Contains(r.handlerSrc, "RequireAdmin")
		routes = append(routes, r)
		return true
	})

	if len(routes) == 0 {
		t.Fatal("found no routes in newMux(); the parse is wrong, not the code")
	}
	return routes
}

// mustBeGated is the set the sweeps below cover: anything that reads as gated
// in the source, plus anything under /api/admin/ whatever its source says. The
// second half matters because a route registered with no guard at all is
// exactly the bug - it must be swept, not skipped for failing the source test.
func mustBeGated(r route) bool {
	return r.adminGated || strings.HasPrefix(r.pattern, "/api/admin/")
}

// TestRouteTableIsReadable fails if the AST sweep stops seeing the table it is
// supposed to guard. Without it, a refactor that this parser cannot follow
// would leave every test below passing over an empty list.
func TestRouteTableIsReadable(t *testing.T) {
	routes := parseRoutes(t)

	var gated, open []string
	for _, r := range routes {
		if r.adminGated {
			gated = append(gated, r.pattern)
		} else {
			open = append(open, r.pattern)
		}
	}
	sort.Strings(gated)
	sort.Strings(open)

	// 35 admin routes as of BOG-55. The floor is a tripwire for a parse that
	// silently stops matching, not a count to keep exact - adding a route is
	// fine, losing two thirds of them is not.
	if len(gated) < 30 {
		t.Fatalf("only %d admin-gated routes found, expected at least 30; the parser has probably stopped matching:\n%s",
			len(gated), strings.Join(gated, "\n"))
	}
	t.Logf("%d admin-gated routes, %d open routes", len(gated), len(open))
	for _, p := range open {
		t.Logf("open: %s", p)
	}
}

// TestEveryAdminPathIsGated is the static half: a route under /api/admin/ that
// was registered without the admin() wrapper. The sweep below would also catch
// it, but this names the line in main.go.
func TestEveryAdminPathIsGated(t *testing.T) {
	for _, r := range parseRoutes(t) {
		if !strings.HasPrefix(r.pattern, "/api/admin/") {
			continue
		}
		if !r.adminGated {
			t.Errorf("%s: %s is registered as %s, which does not go through admin()/auth.RequireAdmin; every /api/admin/ route must",
				r.pos, r.pattern, r.handlerSrc)
		}
	}
}

// TestAdminRoutesRejectAnonymous serves every admin route with no session and
// requires 401. No database: RequireAdmin refuses a request with no cookie
// before it queries admin_sessions.
func TestAdminRoutesRejectAnonymous(t *testing.T) {
	for _, r := range parseRoutes(t) {
		if !mustBeGated(r) {
			continue
		}
		t.Run(r.pattern, func(t *testing.T) {
			// GET for every route, including the POST-only ones: RequireAdmin
			// runs before the handler's method check, so an unauthenticated GET
			// must be 401 and never 405. A 405 here would mean the handler ran.
			rr := anon(t, http.MethodGet, r.pattern, nil)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("anonymous GET %s: expected 401, got %d; body: %s",
					r.pattern, rr.Code, strings.TrimSpace(rr.Body.String()))
			}
		})
	}
}

// TestAdminRoutesRejectNonAdminSession is the 403 half: a real, live session in
// admin_sessions whose account is not on ADMIN_GOOGLE_EMAILS. Needs the shared
// database, because the point is that the session is genuinely valid.
func TestAdminRoutesRejectNonAdminSession(t *testing.T) {
	requireDB(t)
	cookie := nonAdminCookie(t)

	for _, r := range parseRoutes(t) {
		if !mustBeGated(r) {
			continue
		}
		t.Run(r.pattern, func(t *testing.T) {
			rr := serve(t, http.MethodGet, r.pattern, nil, cookie)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("non-admin GET %s: expected 403, got %d; body: %s",
					r.pattern, rr.Code, strings.TrimSpace(rr.Body.String()))
			}
		})
	}
}

// TestExpiredAdminSessionIsRejected pins the expiry half of the session query.
// A session store that ignored expires_at would keep every past admin signed in
// for good, and no CRUD test would notice.
func TestExpiredAdminSessionIsRejected(t *testing.T) {
	requireDB(t)
	cookie := expiredAdminCookie(t)

	rr := serve(t, http.MethodGet, "/api/admin/users", nil, cookie)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expired admin session on /api/admin/users: expected 401, got %d; body: %s",
			rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

// TestPublicRoutesNeedNoSession keeps the guard honest in the other direction:
// if admin() were applied to everything, the sweeps above would pass while the
// admin SPA could no longer load its own config or health check.
func TestPublicRoutesNeedNoSession(t *testing.T) {
	requireDB(t)
	for _, path := range []string{"/api/health", "/api/config/public"} {
		rr := anon(t, http.MethodGet, path, nil)
		if rr.Code != http.StatusOK {
			t.Errorf("anonymous GET %s: expected 200, got %d; body: %s",
				path, rr.Code, strings.TrimSpace(rr.Body.String()))
		}
	}

	// /api/auth/me is the one unauthenticated 401 that is correct: it reports
	// who is signed in, and nobody is.
	rr := anon(t, http.MethodGet, "/api/auth/me", nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("anonymous GET /api/auth/me: expected 401, got %d", rr.Code)
	}
}

// TestCORSRefusesForeignOrigin guards the wrapper the admin routes share with
// the public ones. ApplyCORS sits outside RequireAdmin, so it answers first.
func TestCORSRefusesForeignOrigin(t *testing.T) {
	req := newRequest(t, http.MethodOptions, "/api/admin/entries", nil)
	req.Header.Set("Origin", "https://evil.test")
	rr := record(req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("preflight from a foreign origin: got %d, want 403", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("foreign origin was reflected back as %q", got)
	}
}

// TestCORSAllowsAdminOrigin is the matching positive: the preflight succeeds
// before RequireAdmin, which is what lets the SPA send a credentialed request
// at all.
func TestCORSAllowsAdminOrigin(t *testing.T) {
	rr := anon(t, http.MethodOptions, "/api/admin/entries", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("OPTIONS /api/admin/entries from %s: expected 200, got %d", testOrigin, rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, testOrigin)
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing Access-Control-Allow-Methods header")
	}
}
