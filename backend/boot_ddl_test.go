package main

// Guard for BOG-39.
//
// The admin backend shares one Postgres database with the main lamsza backend,
// and the main backend owns the schema. An admin restart must therefore never
// change the schema. `db.InitDB()` used to run `DROP TABLE locations_legacy`
// and the `auth` package carried a `Migrate()` that created `users`/`sessions`
// and added a constraint; both are gone.
//
// These tests re-derive the startup path from the source on every run, so they
// fail if DDL — or a migrate helper that could reach DDL through a .sql file —
// is wired back onto it.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ddlPattern matches the statements that change schema. It is deliberately
// broad: a false positive is a comment away, a false negative is a shared
// database being mutated by the wrong process.
var ddlPattern = regexp.MustCompile(`(?is)\b(?:CREATE|ALTER|DROP)\s+(?:OR\s+REPLACE\s+)?(?:UNIQUE\s+)?(?:TABLE|INDEX|SEQUENCE|VIEW|TYPE|SCHEMA|EXTENSION|DATABASE|FUNCTION|TRIGGER)\b|\bADD\s+CONSTRAINT\b|\bDROP\s+CONSTRAINT\b|\bTRUNCATE\b`)

// migrateNamePattern matches helpers whose whole job is to change schema. They
// are flagged by name as well as by content because some of them execute DDL
// read from a .sql file, where no literal in the Go source would match.
var migrateNamePattern = regexp.MustCompile(`^[Mm]igrate`)

type funcKey struct {
	pkg  string
	name string
}

// srcFunc is one function declaration plus the import map of the file it came
// from, which is what makes `pkg.Fn()` resolvable.
type srcFunc struct {
	decl    *ast.FuncDecl
	pkg     string
	imports map[string]string // local name -> package name
}

type violation struct {
	pos  string
	fn   funcKey
	what string
}

// TestNoDDLOnStartupPath walks every function reachable from main() and from
// every package init(), and fails on DDL or a migrate helper.
func TestNoDDLOnStartupPath(t *testing.T) {
	fset := token.NewFileSet()
	index, seeds := indexModule(t, fset)

	if len(index[funcKey{"main", "main"}]) == 0 {
		t.Fatal("could not find func main in package main; the guard is not looking at the right tree")
	}

	var violations []violation
	visited := map[funcKey]bool{}
	queue := seeds

	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if visited[key] {
			continue
		}
		visited[key] = true

		if migrateNamePattern.MatchString(key.name) && len(index[key]) > 0 {
			for _, fn := range index[key] {
				violations = append(violations, violation{
					pos:  fset.Position(fn.decl.Pos()).String(),
					fn:   key,
					what: "migrate helper reachable from the startup path",
				})
			}
		}

		for _, fn := range index[key] {
			if fn.decl.Body == nil {
				continue
			}
			ast.Inspect(fn.decl.Body, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.BasicLit:
					if node.Kind == token.STRING && ddlPattern.MatchString(node.Value) {
						violations = append(violations, violation{
							pos:  fset.Position(node.Pos()).String(),
							fn:   key,
							what: "DDL statement: " + firstLine(node.Value),
						})
					}
				case *ast.CallExpr:
					for _, callee := range calleeKeys(fn, node) {
						queue = append(queue, callee)
					}
				}
				return true
			})
		}
	}

	if len(violations) > 0 {
		t.Errorf("the admin startup path must not change the schema — the main lamsza backend owns it (see docs/ARCHITECTURE.md). Found %d violation(s):", len(violations))
		for _, v := range violations {
			t.Errorf("  %s: reached via %s.%s: %s", v.pos, v.fn.pkg, v.fn.name, v.what)
		}
	}
}

// TestDBPackageHasNoDDL pins the exact regression BOG-39 was filed for, without
// depending on the call-graph walk above.
func TestDBPackageHasNoDDL(t *testing.T) {
	entries, err := os.ReadDir("internal/db")
	if err != nil {
		t.Fatalf("read internal/db: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join("internal/db", e.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		// Comments are allowed to name the statements they forbid; only code is
		// checked, so parse and look at string literals.
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, body, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if ok && lit.Kind == token.STRING && ddlPattern.MatchString(lit.Value) {
				t.Errorf("%s: internal/db must only open the connection, never run DDL: %s",
					fset.Position(lit.Pos()), firstLine(lit.Value))
			}
			return true
		})
	}
}

// indexModule parses every non-test .go file under the backend module and
// returns the function index plus the startup seeds: main.main and every
// init(). Every init() in the module is seeded rather than only the imported
// ones — over-approximating keeps the guard on the safe side.
func indexModule(t *testing.T, fset *token.FileSet) (map[funcKey][]*srcFunc, []funcKey) {
	t.Helper()

	index := map[funcKey][]*srcFunc{}
	var seeds []funcKey

	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "data" || d.Name() == "migrations" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}

		imports := map[string]string{}
		for _, imp := range file.Imports {
			target := strings.Trim(imp.Path.Value, `"`)
			name := target[strings.LastIndex(target, "/")+1:]
			local := name
			if imp.Name != nil {
				local = imp.Name.Name
			}
			imports[local] = name
		}

		pkg := file.Name.Name
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			key := funcKey{pkg, fn.Name.Name}
			index[key] = append(index[key], &srcFunc{decl: fn, pkg: pkg, imports: imports})
			if fn.Name.Name == "init" {
				seeds = append(seeds, key)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("index backend sources: %v", err)
	}

	return index, append(seeds, funcKey{"main", "main"})
}

// calleeKeys resolves a call to the function keys it may reach. A bare name is
// looked up in the caller's own package; `x.Fn()` is a package call when x is a
// known import, and is otherwise treated as a same-package method, which again
// over-approximates on purpose.
func calleeKeys(caller *srcFunc, call *ast.CallExpr) []funcKey {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return []funcKey{{caller.pkg, fun.Name}}
	case *ast.SelectorExpr:
		ident, ok := fun.X.(*ast.Ident)
		if !ok {
			return nil
		}
		if pkg, ok := caller.imports[ident.Name]; ok {
			return []funcKey{{pkg, fun.Sel.Name}}
		}
		return []funcKey{{caller.pkg, fun.Sel.Name}}
	}
	return nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(strings.Trim(s, "`\""))
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 90 {
		s = s[:90] + "…"
	}
	return strings.TrimSpace(s)
}
