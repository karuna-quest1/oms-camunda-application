package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"shinro-migration/oms-camunda-application/shared"
)

// DashboardWriter is the Go counterpart of com.oms.dashboard.DashboardRepository —
// the JDBC projection backed by JdbcDashboardRepository (update-first,
// insert-on-miss upsert on the orders_dashboard table, keyed by orderId).
type DashboardWriter interface {
	// Upsert writes (or overwrites) the dashboard row for orderId. The write
	// is idempotent: it is keyed by orderId, so re-running the activity on the
	// same order overwrites the same row with the latest status and timestamp
	// — the downstream must dedupe on orderId (its primary key) for retries to
	// be safe.
	Upsert(orderId string, status shared.OrderStatus, updatedAt time.Time) error
}

// DashboardWriterImpl is the default DashboardWriter. The source
// application's write target is a JDBC repository (JdbcDashboardRepository
// over the orders_dashboard table) that needs a database/DSN, so the
// default (no database configured) fails loudly instead of pretending the
// projection was written. Wire a real implementation (e.g.
// NewSQLDashboardWriter) in cmd/worker.
type DashboardWriterImpl struct{}

// Upsert implements DashboardWriter — NOT_IMPLEMENTED by default: a green
// build must not hide an unwired dashboard write.
func (DashboardWriterImpl) Upsert(orderId string, status shared.OrderStatus, updatedAt time.Time) error {
	return fmt.Errorf("NOT_IMPLEMENTED: DashboardWriter.Upsert for order %s (%s) — wire a real JDBC/dashboard implementation in cmd/worker", orderId, status)
}

// SQLDashboardWriter is the Go port of JdbcDashboardRepository#upsert: an
// update-first, insert-on-miss upsert guarded by the order_id primary key,
// so the same code works on H2 and PostgreSQL without dialect-specific
// ON CONFLICT / MERGE syntax and stays correct under concurrent first
// writes (a concurrent first write makes the insert hit the primary key;
// the fallback update then applies the status so the write is never
// silently lost to the race). The write is idempotent: it is keyed by
// orderId, so re-running the activity on the same order overwrites the same
// row with the latest status and timestamp — the downstream dedupes on
// orderId (its primary key) for retries to be safe.
type SQLDashboardWriter struct {
	db *sql.DB
}

// NewSQLDashboardWriter builds the dashboard writer over the given
// database (the orders_dashboard table must exist: CREATE TABLE
// orders_dashboard (order_id TEXT PRIMARY KEY, status TEXT NOT NULL,
// updated_at TIMESTAMP NOT NULL)).
func NewSQLDashboardWriter(db *sql.DB) *SQLDashboardWriter {
	return &SQLDashboardWriter{db: db}
}

// Upsert implements DashboardWriter.
func (w *SQLDashboardWriter) Upsert(orderId string, status shared.OrderStatus, updatedAt time.Time) error {
	res, err := w.db.Exec(
		"UPDATE orders_dashboard SET status = ?, updated_at = ? WHERE order_id = ?",
		string(status), updatedAt, orderId)
	if err != nil {
		return fmt.Errorf("dashboard update %s: %w", orderId, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Row absent: try to insert. If a concurrent writer inserted it
		// first, the primary key rejects the insert and we fall back to
		// an update — so the write is never silently lost to a race.
		_, err := w.db.Exec(
			"INSERT INTO orders_dashboard (order_id, status, updated_at) VALUES (?, ?, ?)",
			orderId, string(status), updatedAt)
		if err != nil {
			if isDuplicateKeyError(err) {
				_, uerr := w.db.Exec(
					"UPDATE orders_dashboard SET status = ?, updated_at = ? WHERE order_id = ?",
					string(status), updatedAt, orderId)
				return uerr
			}
			return fmt.Errorf("dashboard insert %s: %w", orderId, err)
		}
	}
	return nil
}

// isDuplicateKeyError reports whether err is a unique/primary-key
// violation (parity of Spring's DuplicateKeyException). database/sql
// errors are driver-specific, so match the common wire codes: MySQL/
// MariaDB 1062, PostgreSQL 23505, H2 23505 (SQLState unique violation).
func isDuplicateKeyError(err error) bool {
	byNumber := false
	var numErr sqlErrorNumber
	if errors.As(err, &numErr) {
		byNumber = numErr.Number() == 1062 || numErr.Number() == 23505
	}
	byState := false
	var sqlErr sqlErrorState
	if errors.As(err, &sqlErr) {
		byState = sqlErr.SQLState() == "23505"
	}
	return byNumber || byState
}

// sqlErrorNumber / sqlErrorState narrow the driver-specific error
// interfaces without pinning a driver.
type (
	sqlErrorNumber interface{ error; Number() int }
	sqlErrorState  interface{ error; SQLState() string }
)

// dashboardWriter is the writer used by UpdateDashboard; replace via SetDashboardWriter.
var dashboardWriter DashboardWriter = DashboardWriterImpl{}

// SetDashboardWriter installs a real DashboardWriter before the worker runs.
func SetDashboardWriter(w DashboardWriter) {
	if w != nil {
		dashboardWriter = w
	}
}

// UpdateDashboard reimplements Camunda job worker "update-dashboard"
// (OmsWorkers#updateDashboard).
//
// Upserts the customer dashboard read model. The target status is carried
// on the service task's "status" job header in BPMN, so a single activity
// serves every dashboard step in the model.
//
// As in the source app, this update is best-effort / non-fatal: a
// projection write failure is logged and the activity still succeeds so a
// dashboard outage can never abort the order. An *unsupported status* is
// a programming error, though: the source rethrew the IllegalArgumentException
// (ValidationFailure parity), and we return a plain (retryable) error for it.
func UpdateDashboard(ctx context.Context, in shared.UpdateDashboardInput) (shared.UpdateDashboardOutput, error) {
	logger := slog.Default().With("activity", "update-dashboard")

	if !in.Status.Valid() {
		// Matches ValidationFailure on an unsupported dashboard status
		// (Java OrderStatus.fromString throws IllegalArgumentException).
		logger.ErrorContext(ctx, "unsupported dashboard status", "status", in.Status)
		return shared.UpdateDashboardOutput{}, fmt.Errorf("unsupported dashboard status: %s", in.Status)
	}

	logger.InfoContext(ctx, "update-dashboard", "orderId", in.OrderId, "status", in.Status)
	if err := dashboardWriter.Upsert(in.OrderId, in.Status, time.Now()); err != nil {
		// Best-effort: a projection write failure must never abort the order.
		// The orderId is the idempotency key — safe to re-run.
		logger.WarnContext(ctx, "dashboard update failed; continuing",
			"orderId", in.OrderId, "status", in.Status, "error", err)
	}
	return shared.UpdateDashboardOutput{Applied: true}, nil
}
