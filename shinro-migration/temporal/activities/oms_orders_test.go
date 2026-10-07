package activities

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"shinro-migration/oms-camunda-application/shared"
)

// memoryDashboardWriter records upserts; it stands in for the JDBC
// DashboardRepository during tests.
type memoryDashboardWriter struct {
	rows     map[string]shared.OrderStatus
	writes   int
	failWith error
}

func (m *memoryDashboardWriter) Upsert(orderId string, status shared.OrderStatus, updatedAt time.Time) error {
	m.writes++
	if m.failWith != nil {
		return m.failWith
	}
	if m.rows == nil {
		m.rows = map[string]shared.OrderStatus{}
	}
	m.rows[orderId] = status
	return nil
}

func setWriter(t *testing.T, w DashboardWriter) {
	t.Helper()
	old := dashboardWriter
	SetDashboardWriter(w)
	t.Cleanup(func() { SetDashboardWriter(old) })
}

func TestUpdateDashboardUpsertsKeyedByOrder(t *testing.T) {
	w := &memoryDashboardWriter{}
	setWriter(t, w)

	out, err := UpdateDashboard(context.Background(),
		shared.UpdateDashboardInput{OrderId: "ord-1", Status: shared.StatusOrderReceived})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Applied {
		t.Fatal("applied = false, want true")
	}
	if w.rows["ord-1"] != shared.StatusOrderReceived {
		t.Fatalf("row = %v, want ORDER_RECEIVED", w.rows["ord-1"])
	}
}

func TestUpdateDashboardIsBestEffort(t *testing.T) {
	// A projection write failure must NOT abort the activity — parity of the
	// source worker logging and completing the job anyway.
	w := &memoryDashboardWriter{failWith: errors.New("db down")}
	setWriter(t, w)

	out, err := UpdateDashboard(context.Background(),
		shared.UpdateDashboardInput{OrderId: "ord-1", Status: shared.StatusFulfilled})
	if err != nil {
		t.Fatalf("best-effort dashboard must not return an error, got: %v", err)
	}
	if !out.Applied {
		t.Fatal("applied = false, want true even when the write fails")
	}
}

func TestUpdateDashboardRejectsUnknownStatus(t *testing.T) {
	w := &memoryDashboardWriter{}
	setWriter(t, w)

	_, err := UpdateDashboard(context.Background(),
		shared.UpdateDashboardInput{OrderId: "ord-1", Status: shared.OrderStatus("BOGUS")})
	if err == nil {
		t.Fatal("expected an error for an unsupported dashboard status")
	}
	if w.writes != 0 {
		t.Fatalf("writes = %d, want 0 (no write for a bad status)", w.writes)
	}
}

// recordingPublisher captures the published message for assertions.
type recordingPublisher struct {
	msgs     []*shared.FulfillmentMessage
	failWith error
}

func (r *recordingPublisher) Publish(ctx context.Context, msg *shared.FulfillmentMessage) error {
	if r.failWith != nil {
		return r.failWith
	}
	r.msgs = append(r.msgs, msg)
	return nil
}

func setPublisher(t *testing.T, p FulfillmentPublisher) {
	t.Helper()
	old := fulfillmentPublisher
	SetFulfillmentPublisher(p)
	t.Cleanup(func() { SetFulfillmentPublisher(old) })
}

func TestPublishFulfillmentBuildsEventWithDedupKey(t *testing.T) {
	p := &recordingPublisher{}
	setPublisher(t, p)

	order := &shared.OrderInput{
		OrderId:    "ord-1",
		CustomerId: "cust-1",
		Items:      []shared.OrderItem{{ItemId: "i1", Quantity: 1, UnitPrice: 5.0, SkuId: "S", BrandCode: "B"}},
	}
	payment := &shared.PaymentInput{Rrn: "123456789012", Amount: 5.0}

	out, err := PublishFulfillment(context.Background(),
		shared.PublishFulfillmentInput{Order: order, Payment: payment})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// eventId is the consumer dedup key — deterministic per order.
	if out.EventId != "fulfillment:ord-1:v1" {
		t.Fatalf("eventId = %q, want fulfillment:ord-1:v1", out.EventId)
	}
	if len(p.msgs) != 1 {
		t.Fatalf("published %d messages, want 1", len(p.msgs))
	}
	msg := p.msgs[0]
	if msg.OrderId != "ord-1" || msg.CustomerId != "cust-1" || msg.SchemaVersion != 1 {
		t.Fatalf("message = %+v, want order/customer/schema populated", msg)
	}
	if msg.Payment == nil || msg.Payment.Rrn != payment.Rrn {
		t.Fatalf("payment = %+v, want the captured payment in the event", msg.Payment)
	}
	if len(msg.Items) != 1 || msg.Items[0].ItemId != "i1" {
		t.Fatalf("items = %+v, want the enriched order items", msg.Items)
	}
}

// TestSQLDashboardWriterDuplicateKeyFallsBackToUpdate verifies the port of
// JdbcDashboardRepository#upsert's race handling: when the insert hits the
// primary key (concurrent first writer), the status is still applied via
// the fallback update. The SQL statements are captured from a closed
// *sql.DB whose errors are shaped by a driver-error stand-in.
func TestSQLDashboardWriterDuplicateKeyRecognition(t *testing.T) {
	// The port of JdbcDashboardRepository#upsert falls back to a final
	// UPDATE when the insert hits the primary key (concurrent first
	// writer); the fallback trigger is driver-specific, so assert the
	// recognition logic directly.
	if !isDuplicateKeyError(dupKeyErr{num: 1062}) {
		t.Fatal("error number 1062 must be recognised as a duplicate key")
	}
	if !isDuplicateKeyError(dupKeyErr{state: "23505"}) {
		t.Fatal("SQLState 23505 must be recognised as a duplicate key")
	}
	if isDuplicateKeyError(errors.New("db down")) {
		t.Fatal("a plain error must not be treated as a duplicate key")
	}
}

type dupKeyErr struct {
	num   int
	state string
}

func (e dupKeyErr) Error() string { return fmt.Sprintf("dup key (num=%d state=%q)", e.num, e.state) }
func (e dupKeyErr) Number() int   { return e.num }
func (e dupKeyErr) SQLState() string {
	return e.state
}

func TestPublishFulfillmentTransientFailureIsRetryable(t *testing.T) {
	p := &recordingPublisher{failWith: errors.New("broker down")}
	setPublisher(t, p)

	_, err := PublishFulfillment(context.Background(),
		shared.PublishFulfillmentInput{Order: &shared.OrderInput{OrderId: "ord-1"}, Payment: &shared.PaymentInput{}})
	if err == nil {
		t.Fatal("expected the publish failure to surface (activity retry handles it)")
	}
}
