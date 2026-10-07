package audit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The audit trail is only worth having if an admin cannot edit it. "Append-only
// from the app's perspective" is not a convention anyone can be relied on to
// keep, so this test re-derives it from the source on every run: scan every .go
// file in the backend for a statement that writes `admin_audit_log`, and fail
// on anything that is not an INSERT.
//
// Real append-only enforcement - a database role with INSERT and SELECT but no
// UPDATE or DELETE on this table - belongs to the production database and is
// the owner's, not an agent's. This guard is the half that lives in the code.

var (
	auditTableRef = regexp.MustCompile(`admin_audit_log`)
	mutatingStmt  = regexp.MustCompile(`(?is)\b(UPDATE\s+admin_audit_log|DELETE\s+FROM\s+admin_audit_log|TRUNCATE\s+(?:TABLE\s+)?admin_audit_log|DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?admin_audit_log|ALTER\s+TABLE\s+admin_audit_log)\b`)
)

func TestAdminAuditLogIsAppendOnly(t *testing.T) {
	root := backendRoot(t)

	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		// This file names the forbidden statements in order to look for them.
		if filepath.Base(path) == "append_only_test.go" {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !auditTableRef.Match(src) {
			return nil
		}
		for _, m := range mutatingStmt.FindAllString(string(src), -1) {
			rel, _ := filepath.Rel(root, path)
			offenders = append(offenders, rel+": "+strings.Join(strings.Fields(m), " "))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk backend: %v", err)
	}

	for _, o := range offenders {
		t.Errorf("admin_audit_log must be append-only from this app, but found: %s", o)
	}
}

// TestNoAdminRouteCanReachTheLogExceptToRead is the other half: the only
// handler the registry exposes for the log is the read one, and it refuses
// anything but GET.
func TestNoAdminRouteCanReachTheLogExceptToRead(t *testing.T) {
	res, ok := Lookup("/api/admin/audit-log")
	if !ok {
		t.Fatal("/api/admin/audit-log is not declared; the trail is not queryable")
	}
	if len(res.Tables) != 0 {
		t.Errorf("the audit route declares tables %v; it must not snapshot anything", res.Tables)
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		if got := recorderStatus(t, method); got != 405 {
			t.Errorf("%s /api/admin/audit-log returned %d, want 405", method, got)
		}
	}
}

func backendRoot(t *testing.T) string {
	t.Helper()
	// internal/audit -> internal -> backend
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve backend root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "main.go")); err != nil {
		t.Fatalf("backend root %s has no main.go; the guard is not looking at the right tree", root)
	}
	return root
}
