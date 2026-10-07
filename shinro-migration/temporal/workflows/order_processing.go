package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"shinro-migration/oms-camunda-application/activities"
	"shinro-migration/oms-camunda-application/shared"
)

// orderTTL is the 30-day order lifetime (BPMN Start_TTL timer,
// timeDuration=PT720H). A package variable so conformance tests can
// shorten it; production workers keep the default.
var orderTTL = shared.OrderTTL

// ErrTerminated is the typed workflow error that carries a BPMN terminate
// end event (End_Cancelled / End_Expired): the dashboard step for the
// terminal state was applied, and the workflow instance ends — parity of
// the Zeebe terminate event aborting the whole instance.
func ErrTerminated(outcome string) error {
	return temporal.NewNonRetryableApplicationError("order-processing terminated: "+outcome, "TERMINATED", nil)
}

// OrderProcessingWorkflow reimplements Camunda process "order-processing"
// (models/order-processing.bpmn) as a Temporal workflow.
//
// BPMN → Temporal mapping:
//
//   - service tasks (update-dashboard / validate-order / validate-payment /
//     enrich-order / publish-fulfillment) → activities; the retry policy is
//     MaximumAttempts: 5 on validate-payment only (per the worker's
//     migration guidance) — every other service task keeps the SDK default
//     retry policy;
//   - message catch events (Catch_Payment, Catch_Correction) → signal
//     channels on the same message names, correlated by workflow id
//     (parity of the zeebe correlationKey=orderId — the workflow id is
//     derived from the order id by the starter);
//   - boundary error events → typed non-retryable application errors:
//     Bnd_InvalidOrder (code INVALID_ORDER) on Task_ValidateOrder drives
//     the correction loop; Bnd_InvalidPayment (code INVALID_PAYMENT) on
//     Task_ValidatePayment loops back to the Catch_Payment wait;
//   - interrupting message event subprocess "Cancellation"
//     (Message_CancelOrder → Dashboard CANCELLED → terminate) and
//     interrupting timer event subprocess "Order TTL (30 days)"
//     (PT720H → Dashboard EXPIRED → terminate) → a workflow.Go goroutine
//     per trigger records the delivery; the main flow races every catch
//     point against the armed interrupts, cancel winning the race (the
//     interrupting subprocesses share the process-level scope). An
//     interrupt applies its dashboard step, then ends the workflow with
//     the TERMINATED error (terminate end event).
//
// Race primitive on this SDK build (no workflow.Select; a blocking
// Receive inside an Await condition was probed to defeat the
// event-driven re-evaluation the flag race needs): each message catch
// event is served by one long-lived workflow.Go goroutine that loops
// blocking Receives into a slot (latest delivery + generation counter);
// the main flow suspends in an Await on the generation counters and the
// interrupt flags only — no blocking calls in the condition — and
// consumes the slot after the wake.
func OrderProcessingWorkflow(ctx workflow.Context, in shared.OrderProcessingInput) (shared.OrderProcessingResult, error) {
	logger := workflow.GetLogger(ctx)

	// Base activity options: a generous start-to-close bound. Only
	// validate-payment gets a cap — its migration guidance names
	// maxAttempts=5; the other service tasks keep the SDK default retry
	// policy.
	base := workflow.ActivityOptions{StartToCloseTimeout: 5 * time.Minute}
	validatePayment := base
	validatePayment.RetryPolicy = &temporal.RetryPolicy{MaximumAttempts: 5}
	ctx = workflow.WithActivityOptions(ctx, base)

	orderID := in.OrderId
	order := in.Order

	// --- interrupting subprocess "Cancellation" (Message_CancelOrder) ---
	cancelSig := (*shared.CancelOrderSignal)(nil)
	workflow.Go(ctx, func(gctx workflow.Context) {
		var sig shared.CancelOrderSignal
		workflow.GetSignalChannel(gctx, shared.SignalCancelOrder).Receive(gctx, &sig)
		s := sig
		cancelSig = &s
	})

	// --- interrupting timer subprocess "Order TTL (30 days)" (PT720H) ---
	// workflow.NewTimer is deterministic; the future completes 720h after
	// the start event on every replay.
	ttlFired := false
	workflow.Go(ctx, func(gctx workflow.Context) {
		if err := workflow.NewTimer(gctx, orderTTL).Get(gctx, nil); err != nil {
			return
		}
		ttlFired = true
	})

	// Catch_Payment delivery slot: one goroutine loops Receives on the
	// Message_CapturePayment channel so the (re-entered after an invalid
	// payment) wait keeps working across loop iterations.
	capSlot := (*shared.CapturePaymentSignal)(nil)
	capGen := 0
	workflow.Go(ctx, func(gctx workflow.Context) {
		ch := workflow.GetSignalChannel(gctx, shared.SignalCapturePayment)
		for {
			var p shared.CapturePaymentSignal
			ch.Receive(gctx, &p)
			q := p
			capSlot = &q
			capGen++
		}
	})

	// Catch_Correction delivery slot (same pattern).
	corrSlot := (*shared.SupportCorrectionSignal)(nil)
	corrGen := 0
	workflow.Go(ctx, func(gctx workflow.Context) {
		ch := workflow.GetSignalChannel(gctx, shared.SignalSupportCorrection)
		for {
			var p shared.SupportCorrectionSignal
			ch.Receive(gctx, &p)
			q := p
			corrSlot = &q
			corrGen++
		}
	})

	// handleTTL runs the TTL subprocess body: Dashboard EXPIRED, then the
	// terminate end event (End_Expired).
	handleTTL := func() error {
		logger.Info("order TTL (30 days) elapsed")
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard,
			shared.UpdateDashboardInput{OrderId: orderID, Status: shared.StatusExpired}).Get(ctx, nil); err != nil {
			return err
		}
		// End_Expired: terminate end event.
		return ErrTerminated("EXPIRED")
	}

	// handleCancel runs the Cancellation subprocess body: Dashboard
	// CANCELLED, then the terminate end event (End_Cancelled).
	handleCancel := func(sig *shared.CancelOrderSignal) error {
		logger.Info("cancel order signal received reason=", sig.Reason)
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard,
			shared.UpdateDashboardInput{OrderId: orderID, Status: shared.StatusCancelled}).Get(ctx, nil); err != nil {
			return err
		}
		// End_Cancelled: terminate end event.
		return ErrTerminated("CANCELLED")
	}

	// armedInterrupt names the interrupt that has already arrived, the
	// cancel signal winning the race when both are set — matching the
	// interrupting subprocesses' shared process-level scope (and the
	// test-environment's event ordering, where a long TTL timer can be
	// fast-forwarded ahead of the cancel signal inside one workflow task).
	// Returns "" when no interrupt has fired.
	armedInterrupt := func() string {
		if cancelSig != nil {
			return "cancel"
		}
		if ttlFired {
			return "ttl"
		}
		return ""
	}

	// checkInterrupts handles interrupts that arrived while the main flow
	// was inside an activity or a catch wait.
	checkInterrupts := func() (bool, error) {
		switch armedInterrupt() {
		case "cancel":
			return true, handleCancel(cancelSig)
		case "ttl":
			return true, handleTTL()
		}
		return false, nil
	}

	payment := shared.PaymentInput{} // zero value until CapturePayment arrives
	var correctionItems []shared.OrderItem // pending SupportCorrection, cleared once consumed
	corrConsumed := 0
	capConsumed := 0

	// Task_UpdateReceived: Dashboard: ORDER_RECEIVED
	if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard,
		shared.UpdateDashboardInput{OrderId: orderID, Status: shared.StatusOrderReceived}).Get(ctx, nil); err != nil {
		return shared.OrderProcessingResult{}, err
	}
	if done, err := checkInterrupts(); done {
		return shared.OrderProcessingResult{}, err
	}

	for {
		// Task_ValidateOrder: Validate order (Commerce API). The activity
		// applies correctionItems itself (parity of applyCorrection) and
		// throws BPMN error INVALID_ORDER on business failure.
		var validateOut shared.ValidateOrderOutput
		err := workflow.ExecuteActivity(ctx, activities.ValidateOrder,
			shared.ValidateOrderInput{Order: order, CorrectionItems: correctionItems}).Get(ctx, &validateOut)
		if err != nil {
			if isBpmnError(err, shared.ErrInvalidOrder) {
				logger.Info("order validation failed; awaiting correction reason=", err.Error())
				// Task_AwaitCorrectionStatus: Dashboard: AWAITING_CORRECTION
				if derr := workflow.ExecuteActivity(ctx, activities.UpdateDashboard,
					shared.UpdateDashboardInput{OrderId: orderID, Status: shared.StatusAwaitingCorrection}).Get(ctx, nil); derr != nil {
					return shared.OrderProcessingResult{}, derr
				}
				// Catch_Correction: await Message_SupportCorrection,
				// racing the interrupting subprocesses.
				workflow.Await(ctx, func() bool {
					return corrGen > corrConsumed || armedInterrupt() != ""
				})
				if done, ierr := checkInterrupts(); done {
					// Drop the stray correction delivery; the
					// subprocess wins.
					return shared.OrderProcessingResult{}, ierr
				}
				// Consume the correction (the catch-event payload must
				// reach the next validation, never be dropped).
				correctionItems = corrSlot.Items
				corrConsumed = corrGen
				// Re-enter Task_ValidateOrder (loop); the correction is
				// consumed and cleared by the next validation.
				continue
			}
			return shared.OrderProcessingResult{}, err
		}
		order = validateOut.Order
		// The consumed correction is cleared (next validation passes nil —
		// parity of newCompleteCommand writing correctionItems=null).
		correctionItems = nil

		// Task_UpdatePaymentPending: Dashboard: PAYMENT_PENDING
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard,
			shared.UpdateDashboardInput{OrderId: orderID, Status: shared.StatusPaymentPending}).Get(ctx, nil); err != nil {
			return shared.OrderProcessingResult{}, err
		}
		if done, err := checkInterrupts(); done {
			return shared.OrderProcessingResult{}, err
		}

		// Catch_Payment: await Message_CapturePayment, racing the
		// interrupting Cancellation subprocess and the 30-day TTL timer.
		// (Also the landing point of the Bnd_InvalidPayment boundary below.)
		workflow.Await(ctx, func() bool {
			return capGen > capConsumed || armedInterrupt() != ""
		})
		if done, ierr := checkInterrupts(); done {
			// Drop any stray payment delivery; the subprocess wins.
			return shared.OrderProcessingResult{}, ierr
		}
		// Consume the delivery (catch-event payload reaches the next
		// activity).
		capConsumed = capGen
		var captured shared.PaymentInput
		if capSlot.Payment != nil {
			captured = *capSlot.Payment
		} else {
			// A CapturePayment delivery without a payment body would
			// nil-panic the RRN check; treat it as an empty payment so
			// the validate-payment boundary (INVALID_PAYMENT) rejects
			// it instead.
		}

		// Task_ValidatePayment: Validate payment RRN — boundary
		// Bnd_InvalidPayment (error INVALID_PAYMENT) loops back to the
		// Catch_Payment wait; the next CapturePayment message overwrites
		// the stale payment.
		var payOut shared.ValidatePaymentOutput
		err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, validatePayment),
			activities.ValidatePayment,
			shared.ValidatePaymentInput{Order: order, Payment: &captured}).Get(ctx, &payOut)
		if err != nil {
			if isBpmnError(err, shared.ErrInvalidPayment) {
				logger.Info("payment validation failed; awaiting new payment reason=", err.Error())
				// Bnd_InvalidPayment → Flow_bnd_catchpay → Catch_Payment.
				if done, ierr := checkInterrupts(); done {
					return shared.OrderProcessingResult{}, ierr
				}
				continue
			}
			return shared.OrderProcessingResult{}, err
		}
		payment = *payOut.Payment
		break
	}

	// Task_UpdateCaptured: Dashboard: PAYMENT_CAPTURED
	if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard,
		shared.UpdateDashboardInput{OrderId: orderID, Status: shared.StatusPaymentCaptured}).Get(ctx, nil); err != nil {
		return shared.OrderProcessingResult{}, err
	}
	if done, err := checkInterrupts(); done {
		return shared.OrderProcessingResult{}, err
	}

	// Task_Enrich: Enrich with PIM
	var enrichOut shared.EnrichOrderOutput
	if err := workflow.ExecuteActivity(ctx, activities.EnrichOrder,
		shared.EnrichOrderInput{Order: order}).Get(ctx, &enrichOut); err != nil {
		return shared.OrderProcessingResult{}, err
	}
	order = enrichOut.Order

	// Task_Publish: Publish to fulfillment
	if err := workflow.ExecuteActivity(ctx, activities.PublishFulfillment,
		shared.PublishFulfillmentInput{Order: order, Payment: &payment}).Get(ctx, nil); err != nil {
		return shared.OrderProcessingResult{}, err
	}

	// Task_UpdateFulfilled: Dashboard: FULFILLED
	if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard,
		shared.UpdateDashboardInput{OrderId: orderID, Status: shared.StatusFulfilled}).Get(ctx, nil); err != nil {
		return shared.OrderProcessingResult{}, err
	}

	// EndEvent_Done: Order fulfilled
	return shared.OrderProcessingResult{
		OrderId: orderID,
		Outcome: "FULFILLED",
		Order:   order,
		Payment: &payment,
	}, nil
}

// appErrorCode walks the error chain (this SDK build wraps activity
// failures in *internal.WorkflowExecutionError before they reach the
// workflow) and returns the Type() of the first *temporal.ApplicationError
// found, or "" when none is present.
func appErrorCode(err error) string {
	for err != nil {
		if ae, ok := err.(*temporal.ApplicationError); ok {
			return ae.Type()
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return ""
		}
		err = u.Unwrap()
	}
	return ""
}

// isBpmnError reports whether err is an application error carrying the given
// BPMN error code (the Go equivalent of a Zeebe boundary error event matched
// on errorCode).
func isBpmnError(err error, code string) bool {
	return appErrorCode(err) == code
}
