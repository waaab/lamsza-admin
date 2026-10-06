package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// recorderStatus calls the audit-log read handler directly, without a database,
// so only the method gate is under test.
func recorderStatus(t *testing.T, method string) int {
	t.Helper()
	w := httptest.NewRecorder()
	HandleAdminAuditLog(w, httptest.NewRequest(method, "/api/admin/audit-log", nil))
	return w.Code
}

func TestAuditLogReadRejectsNonGET(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if got := recorderStatus(t, method); got != http.StatusMethodNotAllowed {
			t.Errorf("%s returned %d, want 405", method, got)
		}
	}
}
