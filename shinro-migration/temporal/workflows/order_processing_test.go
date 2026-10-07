package workflows

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"shinro-migration/oms-camunda-application/activities"
	"shinro-migration/oms-camunda-application/shared"
)

// newTestEnv builds a workflow test environment. (In this SDK build the
// testsuite constructors are methods on the WorkflowTestSuite.)
func newTestEnv() *testsuite.TestWorkflowEnvironment {
	return (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
}

// This cached SDK build (v1.34.0) matches activity calls against the mock
// WITHOUT prepending the context argument to stored expectations, so every
// OnActivity below registers a mock.Anything context placeholder first.

// testOrder / testPayment are the canonical fixtures: the payment amount
// matches the order total, and the RRN is a valid 12-digit number.
var (
	testOrder = &shared.OrderInput{
		OrderId:     "ord-1",
		CustomerId:  "cust-1",
		Items:       []shared.OrderItem{{ItemId: "i1", Quantity: 2, UnitPrice: 10.0}},
		TotalAmount: 20.0,
		Currency:    "USD",
	}
	testPayment = &shared.PaymentInput{Rrn: "123456789012", Amount: 20.0, Currency: "USD"}
)

// enrichedOrder is what EnrichOrder returns for testOrder: SKU-i1 gets the
// deterministic "SKU-ord-1-1" and the brand "BRAND-i1".
var enrichedOrder = &shared.OrderInput{
	OrderId:     "ord-1",
	CustomerId:  "cust-1",
	Items:       []shared.OrderItem{{ItemId: "i1", Quantity: 2, UnitPrice: 10.0, SkuId: "SKU-ord-1-1", BrandCode: "BRAND-i1"}},
	TotalAmount: 20.0,
	Currency:    "USD",
}

func dash(status shared.OrderStatus) shared.UpdateDashboardInput {
	return shared.UpdateDashboardInput{OrderId: "ord-1", Status: status}
}

// mockDashboard registers an UpdateDashboard expectation for one status.
func mockDashboard(env *testsuite.TestWorkflowEnvironment, status shared.OrderStatus) {
	env.OnActivity(activities.UpdateDashboard, mock.Anything, dash(status)).
		Return(shared.UpdateDashboardOutput{Applied: true}, nil)
}

// registerHappyTail mocks the activity sequence after a successful
// ValidatePayment: PAYMENT_CAPTURED, enrich, publish (with the validated
// payment — the catch-event payload must reach the next activity), FULFILLED.
func registerHappyTail(env *testsuite.TestWorkflowEnvironment) {
	mockDashboard(env, shared.StatusPaymentCaptured)
	env.OnActivity(activities.EnrichOrder, mock.Anything,
		shared.EnrichOrderInput{Order: testOrder}).
		Return(shared.EnrichOrderOutput{Order: enrichedOrder}, nil)
	env.OnActivity(activities.PublishFulfillment, mock.Anything,
		shared.PublishFulfillmentInput{Order: enrichedOrder, Payment: testPayment}).
		Return(shared.PublishFulfillmentOutput{EventId: "fulfillment:ord-1:v1"}, nil)
	mockDashboard(env, shared.StatusFulfilled)
}

func runWorkflow(env *testsuite.TestWorkflowEnvironment, in shared.OrderProcessingInput) (shared.OrderProcessingResult, error) {
	var result shared.OrderProcessingResult
	env.ExecuteWorkflow(OrderProcessingWorkflow, in)
	err := env.GetWorkflowError()
	if !env.IsWorkflowCompleted() {
		// The environment timed out (default 1s) — the workflow did not
		// reach a terminal state in mock time.
		err = temporal.NewNonRetryableApplicationError("workflow did not complete (test timeout)", "TEST_TIMEOUT", nil)
	}
	if err == nil {
		_ = env.GetWorkflowResult(&result)
	}
	return result, err
}

func TestOrderProcessingHappyPath(t *testing.T) {
	env := newTestEnv()
	env.SetTestTimeout(10 * time.Second)

	in := shared.OrderProcessingInput{OrderId: "ord-1", Order: testOrder}

	mockDashboard(env, shared.StatusOrderReceived)
	env.OnActivity(activities.ValidateOrder, mock.Anything,
		shared.ValidateOrderInput{Order: testOrder, CorrectionItems: nil}).
		Return(shared.ValidateOrderOutput{Order: testOrder}, nil)
	mockDashboard(env, shared.StatusPaymentPending)
	env.OnActivity(activities.ValidatePayment, mock.Anything,
		shared.ValidatePaymentInput{Order: testOrder, Payment: testPayment}).
		Return(shared.ValidatePaymentOutput{Payment: testPayment}, nil)
	registerHappyTail(env)

	// Catch_Payment: Message_CapturePayment arrives while the workflow waits.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalCapturePayment,
			shared.CapturePaymentSignal{OrderId: "ord-1", Payment: testPayment})
	}, time.Millisecond)

	result, err := runWorkflow(env, in)
	if err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	if result.Outcome != "FULFILLED" {
		t.Fatalf("outcome = %q, want FULFILLED", result.Outcome)
	}
	if result.Order == nil || result.Order.OrderId != "ord-1" ||
		len(result.Order.Items) != 1 || result.Order.Items[0].SkuId != "SKU-ord-1-1" ||
		result.Order.Items[0].BrandCode != "BRAND-i1" {
		t.Fatalf("result order = %+v, want the enriched order", result.Order)
	}
	if result.Payment == nil || result.Payment.Rrn != testPayment.Rrn {
		t.Fatalf("result payment = %+v, want RRN %s", result.Payment, testPayment.Rrn)
	}
	env.AssertExpectations(t)
}

func TestOrderProcessingInvalidOrderCorrectionLoop(t *testing.T) {
	env := newTestEnv()
	env.SetTestTimeout(10 * time.Second)

	// First validation: the source order fails business validation
	// (empty items). The corrected items fix it.
	badOrder := &shared.OrderInput{
		OrderId:     "ord-1",
		CustomerId:  "cust-1",
		Items:       nil,
		TotalAmount: 20.0,
		Currency:    "USD",
	}
	correctedItems := []shared.OrderItem{{ItemId: "i1", Quantity: 2, UnitPrice: 10.0}}

	mockDashboard(env, shared.StatusOrderReceived)

	// Task_ValidateOrder #1 -> BPMN error INVALID_ORDER (non-retryable).
	env.OnActivity(activities.ValidateOrder, mock.Anything,
		shared.ValidateOrderInput{Order: badOrder, CorrectionItems: nil}).
		Return(shared.ValidateOrderOutput{}, temporal.NewNonRetryableApplicationError(
			"order must contain at least one item", shared.ErrInvalidOrder, nil))

	// Bnd_InvalidOrder -> Task_AwaitCorrectionStatus.
	mockDashboard(env, shared.StatusAwaitingCorrection)

	// Task_ValidateOrder #2 — the correction reaches the next validation
	// (the catch-event payload must not be dropped).
	env.OnActivity(activities.ValidateOrder, mock.Anything,
		shared.ValidateOrderInput{Order: badOrder, CorrectionItems: correctedItems}).
		Return(shared.ValidateOrderOutput{Order: testOrder}, nil)

	mockDashboard(env, shared.StatusPaymentPending)
	env.OnActivity(activities.ValidatePayment, mock.Anything,
		shared.ValidatePaymentInput{Order: testOrder, Payment: testPayment}).
		Return(shared.ValidatePaymentOutput{Payment: testPayment}, nil)
	registerHappyTail(env)

	// Catch_Correction: Message_SupportCorrection, then Catch_Payment.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalSupportCorrection,
			shared.SupportCorrectionSignal{OrderId: "ord-1", Items: correctedItems})
	}, time.Millisecond)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalCapturePayment,
			shared.CapturePaymentSignal{OrderId: "ord-1", Payment: testPayment})
	}, 2*time.Millisecond)

	result, err := runWorkflow(env,
		shared.OrderProcessingInput{OrderId: "ord-1", Order: badOrder})
	if err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	if result.Outcome != "FULFILLED" {
		t.Fatalf("outcome = %q, want FULFILLED", result.Outcome)
	}
	env.AssertExpectations(t)
}

func TestOrderProcessingInvalidPaymentLoopsToCapturePayment(t *testing.T) {
	env := newTestEnv()
	env.SetTestTimeout(10 * time.Second)

	badPayment := &shared.PaymentInput{Rrn: "123", Amount: 20.0, Currency: "USD"}

	mockDashboard(env, shared.StatusOrderReceived)
	env.OnActivity(activities.ValidateOrder, mock.Anything,
		shared.ValidateOrderInput{Order: testOrder, CorrectionItems: nil}).
		Return(shared.ValidateOrderOutput{Order: testOrder}, nil)
	mockDashboard(env, shared.StatusPaymentPending)

	// Task_ValidatePayment #1 -> BPMN error INVALID_PAYMENT (non-retryable)
	// -> Bnd_InvalidPayment loops back to Catch_Payment (no AWAITING
	// dashboard step in between — only a fresh CapturePayment).
	env.OnActivity(activities.ValidatePayment, mock.Anything,
		shared.ValidatePaymentInput{Order: testOrder, Payment: badPayment}).
		Return(shared.ValidatePaymentOutput{}, temporal.NewNonRetryableApplicationError(
			"invalid RRN length 3; must be 12 digits", shared.ErrInvalidPayment, nil))
	env.OnActivity(activities.ValidatePayment, mock.Anything,
		shared.ValidatePaymentInput{Order: testOrder, Payment: testPayment}).
		Return(shared.ValidatePaymentOutput{Payment: testPayment}, nil)
	registerHappyTail(env)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalCapturePayment,
			shared.CapturePaymentSignal{OrderId: "ord-1", Payment: badPayment})
	}, time.Millisecond)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalCapturePayment,
			shared.CapturePaymentSignal{OrderId: "ord-1", Payment: testPayment})
	}, 2*time.Millisecond)

	result, err := runWorkflow(env,
		shared.OrderProcessingInput{OrderId: "ord-1", Order: testOrder})
	if err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	if result.Outcome != "FULFILLED" {
		t.Fatalf("outcome = %q, want FULFILLED", result.Outcome)
	}
	if result.Payment == nil || result.Payment.Rrn != testPayment.Rrn {
		t.Fatalf("result payment = %+v, want the good payment", result.Payment)
	}
	env.AssertExpectations(t)
}

// TestOrderProcessingCancelSubprocess exercises the interrupting message
// event subprocess "Cancellation": Message_CancelOrder while the workflow is
// parked at Catch_Payment -> Dashboard CANCELLED -> terminate end event.
func TestOrderProcessingCancelSubprocess(t *testing.T) {
	env := newTestEnv()
	env.SetTestTimeout(10 * time.Second)

	// Shorten the 720h TTL below the test timeout: at the full 720h the
	// mock clock's timer fast-forward coincides with the test deadline
	// and a ScheduleToClose failure masks the CANCELLED terminate. With a
	// 5s TTL (below the 10s test timeout) the timer fires as a normal
	// timer event, and the test still asserts the cancel subprocess wins
	// the race (EXPIRED must NOT be the outcome).
	origTTL := orderTTL
	orderTTL = 5 * time.Second
	defer func() { orderTTL = origTTL }()

	mockDashboard(env, shared.StatusOrderReceived)
	env.OnActivity(activities.ValidateOrder, mock.Anything,
		shared.ValidateOrderInput{Order: testOrder, CorrectionItems: nil}).
		Return(shared.ValidateOrderOutput{Order: testOrder}, nil)
	mockDashboard(env, shared.StatusPaymentPending)
	mockDashboard(env, shared.StatusCancelled)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalCancelOrder,
			shared.CancelOrderSignal{OrderId: "ord-1", Reason: "customer request"})
	}, time.Millisecond)

	var result shared.OrderProcessingResult
	env.ExecuteWorkflow(OrderProcessingWorkflow,
		shared.OrderProcessingInput{OrderId: "ord-1", Order: testOrder})
	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("expected the TERMINATED typed error from the terminate end event")
	}
	if !isBpmnError(err, "TERMINATED") {
		t.Fatalf("workflow error = %v (%T), want the TERMINATED application error", err, err)
	}
	if !strings.Contains(err.Error(), "CANCELLED") {
		t.Fatalf("workflow error = %v, want the CANCELLED terminate (cancel must beat TTL)", err)
	}
	_ = result
	env.AssertExpectations(t)
}

// TestOrderProcessingTTLSubprocess exercises the interrupting timer event
// subprocess "Order TTL (30 days)": no CapturePayment arrives, the timer
// fires -> Dashboard EXPIRED -> terminate end event. The mock clock in the
// test environment fast-forwards the (shortened) TTL timer.
func TestOrderProcessingTTLSubprocess(t *testing.T) {
	env := newTestEnv()
	env.SetTestTimeout(10 * time.Second)

	origTTL := orderTTL
	orderTTL = time.Hour // shortened so the mock clock reaches it quickly
	defer func() { orderTTL = origTTL }()

	mockDashboard(env, shared.StatusOrderReceived)
	env.OnActivity(activities.ValidateOrder, mock.Anything,
		shared.ValidateOrderInput{Order: testOrder, CorrectionItems: nil}).
		Return(shared.ValidateOrderOutput{Order: testOrder}, nil)
	mockDashboard(env, shared.StatusPaymentPending)
	mockDashboard(env, shared.StatusExpired)

	// No CapturePayment is delivered: the 30-day TTL must win.
	env.ExecuteWorkflow(OrderProcessingWorkflow,
		shared.OrderProcessingInput{OrderId: "ord-1", Order: testOrder})
	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("expected the TERMINATED typed error from the terminate end event")
	}
	if !isBpmnError(err, "TERMINATED") {
		t.Fatalf("workflow error = %v (%T), want the TERMINATED application error", err, err)
	}
	if !strings.Contains(err.Error(), "EXPIRED") {
		t.Fatalf("workflow error = %v, want the EXPIRED terminate", err)
	}
	env.AssertExpectations(t)
}

// TestOrderProcessingValidateOrderTransientFailure asserts that a
// non-BPMN (retryable) failure from ValidateOrder is NOT swallowed by the
// correction loop — the boundary event only matches the INVALID_ORDER code,
// so a transient error propagates and eventually fails the workflow.
func TestOrderProcessingValidateOrderTransientFailure(t *testing.T) {
	env := newTestEnv()
	env.SetTestTimeout(10 * time.Second)

	mockDashboard(env, shared.StatusOrderReceived)
	env.OnActivity(activities.ValidateOrder, mock.Anything,
		shared.ValidateOrderInput{Order: testOrder, CorrectionItems: nil}).
		Return(shared.ValidateOrderOutput{}, temporal.NewApplicationError(
			"commerce API timeout", "TRANSIENT", nil))

	var result shared.OrderProcessingResult
	env.ExecuteWorkflow(OrderProcessingWorkflow,
		shared.OrderProcessingInput{OrderId: "ord-1", Order: testOrder})
	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("expected the workflow to fail on the transient validation error")
	}
	if isBpmnError(err, shared.ErrInvalidOrder) {
		t.Fatal("a TRANSIENT error must not be treated as the INVALID_ORDER boundary")
	}
	_ = result
}
