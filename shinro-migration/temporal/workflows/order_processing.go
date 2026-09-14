package workflows

import (
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"shinro-migration/oms-camunda-quest1/activities"
	"shinro-migration/oms-camunda-quest1/shared"
)

// OrderProcessingWorkflow reimplements Camunda process "order-processing" as a Temporal workflow.
// Message "Message_CancelOrder" becomes a Temporal signal of the same name.
// Message "Message_CapturePayment" becomes a Temporal signal of the same name.
// Message "Message_SupportCorrection" becomes a Temporal signal of the same name.
// Timer Start_TTL (timeDuration=PT720H) becomes workflow.Sleep / workflow.NewTimer.
func OrderProcessingWorkflow(ctx workflow.Context, in shared.OrderProcessingInput) (shared.OrderProcessingResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// <shingen:body workflow:order-processing>
	logger := workflow.GetLogger(ctx)

	// Create order input from workflow input
	order := shared.OrderInput{
		OrderID:     in.OrderID,
		CustomerID:  in.CustomerID,
		Items:       in.Items,
		TotalAmount: in.TotalAmount,
		Currency:    in.Currency,
	}

	// State variables
	var payment shared.PaymentInput
	var correctionItems []shared.OrderItem
	orderStatus := "PENDING"

	// Set up signal channels
	cancelCh := workflow.GetSignalChannel(ctx, "Message_CancelOrder")
	captureCh := workflow.GetSignalChannel(ctx, "Message_CapturePayment")
	correctionCh := workflow.GetSignalChannel(ctx, "Message_SupportCorrection")

	// Set up 30-day TTL timer (PT720H)
	ttlTimer := workflow.NewTimer(ctx, 720*time.Hour)

	// Use a selector to handle the main flow, signals, and timer concurrently
	selector := workflow.NewSelector(ctx)

	// Track completion
	workflowComplete := false
	var workflowErr error

	// Main flow goroutine
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("workflow panicked", "panic", r)
				workflowErr = fmt.Errorf("workflow panicked: %v", r)
			}
			workflowComplete = true
		}()

		// Dashboard: ORDER_RECEIVED
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard, shared.UpdateDashboardInput{
			OrderID: order.OrderID,
			Status:  "ORDER_RECEIVED",
		}).Get(ctx, nil); err != nil {
			logger.Warn("dashboard update failed (non-fatal)", "status", "ORDER_RECEIVED", "error", err)
		}

		// Validate order with correction loop
		validateOrderLoop := func() error {
			for {
				validateOut := shared.ValidateOrderOutput{}
				err := workflow.ExecuteActivity(ctx, activities.ValidateOrder, shared.ValidateOrderInput{
					Order:           order,
					CorrectionItems: correctionItems,
				}).Get(ctx, &validateOut)

				if err != nil {
					var appErr *temporal.ApplicationError
					if errors.As(err, &appErr) && appErr.Type() == "INVALID_ORDER" {
						// Boundary event: INVALID_ORDER -> await correction
						logger.Info("order validation failed, awaiting correction", "error", appErr.Message())
						
						// Dashboard: AWAITING_CORRECTION
						if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard, shared.UpdateDashboardInput{
							OrderID: order.OrderID,
							Status:  "AWAITING_CORRECTION",
						}).Get(ctx, nil); err != nil {
							logger.Warn("dashboard update failed (non-fatal)", "status", "AWAITING_CORRECTION", "error", err)
						}

						// Wait for SupportCorrection signal
						var correctionSignal shared.SupportCorrectionSignal
						correctionCh.Receive(ctx, &correctionSignal)
						correctionItems = correctionSignal.CorrectionItems
						logger.Info("received correction, retrying validation", "itemCount", len(correctionItems))
						continue // Retry validation
					}
					return err
				}

				// Validation succeeded
				order = validateOut.Order
				correctionItems = nil // Clear correction
				break
			}
			return nil
		}

		if err := validateOrderLoop(); err != nil {
			workflowErr = err
			return
		}

		// Dashboard: PAYMENT_PENDING
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard, shared.UpdateDashboardInput{
			OrderID: order.OrderID,
			Status:  "PAYMENT_PENDING",
		}).Get(ctx, nil); err != nil {
			logger.Warn("dashboard update failed (non-fatal)", "status", "PAYMENT_PENDING", "error", err)
		}

		// Wait for CapturePayment signal with payment validation loop
		capturePaymentLoop := func() error {
			for {
				// Wait for payment signal
				var captureSignal shared.CapturePaymentSignal
				captureCh.Receive(ctx, &captureSignal)
				payment = captureSignal.Payment
				logger.Info("received payment", "rrn", payment.RRN)

				// Validate payment
				validatePayOut := shared.ValidatePaymentOutput{}
				err := workflow.ExecuteActivity(ctx, activities.ValidatePayment, shared.ValidatePaymentInput{
					Payment:        payment,
					ExpectedAmount: order.TotalAmount,
				}).Get(ctx, &validatePayOut)

				if err != nil {
					var appErr *temporal.ApplicationError
					if errors.As(err, &appErr) && appErr.Type() == "INVALID_PAYMENT" {
						// Boundary event: INVALID_PAYMENT -> loop back to await payment
						logger.Info("payment validation failed, awaiting new payment", "error", appErr.Message())
						continue // Loop back to await payment
					}
					return err
				}

				// Payment validation succeeded
				payment = validatePayOut.Payment
				break
			}
			return nil
		}

		if err := capturePaymentLoop(); err != nil {
			workflowErr = err
			return
		}

		// Dashboard: PAYMENT_CAPTURED
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard, shared.UpdateDashboardInput{
			OrderID: order.OrderID,
			Status:  "PAYMENT_CAPTURED",
		}).Get(ctx, nil); err != nil {
			logger.Warn("dashboard update failed (non-fatal)", "status", "PAYMENT_CAPTURED", "error", err)
		}

		// Enrich with PIM
		enrichOut := shared.EnrichOrderOutput{}
		if err := workflow.ExecuteActivity(ctx, activities.EnrichOrder, shared.EnrichOrderInput{
			Order: order,
		}).Get(ctx, &enrichOut); err != nil {
			workflowErr = err
			return
		}
		order = enrichOut.Order

		// Publish to fulfillment
		if err := workflow.ExecuteActivity(ctx, activities.PublishFulfillment, shared.PublishFulfillmentInput{
			Order:   order,
			Payment: payment,
		}).Get(ctx, nil); err != nil {
			workflowErr = err
			return
		}

		// Dashboard: FULFILLED
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard, shared.UpdateDashboardInput{
			OrderID: order.OrderID,
			Status:  "FULFILLED",
		}).Get(ctx, nil); err != nil {
			logger.Warn("dashboard update failed (non-fatal)", "status", "FULFILLED", "error", err)
		}

		orderStatus = "FULFILLED"
	})

	// Handle cancel signal
	selector.AddReceive(cancelCh, func(c workflow.ReceiveChannel, more bool) {
		if workflowComplete {
			return
		}
		var cancelSignal shared.CancelOrderSignal
		c.Receive(ctx, &cancelSignal)
		logger.Info("received cancel signal")

		// Dashboard: CANCELLED
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard, shared.UpdateDashboardInput{
			OrderID: order.OrderID,
			Status:  "CANCELLED",
		}).Get(ctx, nil); err != nil {
			logger.Warn("dashboard update failed (non-fatal)", "status", "CANCELLED", "error", err)
		}

		orderStatus = "CANCELLED"
		workflowComplete = true
	})

	// Handle TTL timer
	selector.AddFuture(ttlTimer, func(f workflow.Future) {
		if workflowComplete {
			return
		}
		logger.Info("order TTL expired (30 days)")

		// Dashboard: EXPIRED
		if err := workflow.ExecuteActivity(ctx, activities.UpdateDashboard, shared.UpdateDashboardInput{
			OrderID: order.OrderID,
			Status:  "EXPIRED",
		}).Get(ctx, nil); err != nil {
			logger.Warn("dashboard update failed (non-fatal)", "status", "EXPIRED", "error", err)
		}

		orderStatus = "EXPIRED"
		workflowComplete = true
	})

	// Run selector until workflow completes
	for !workflowComplete {
		selector.Select(ctx)
	}

	if workflowErr != nil {
		return shared.OrderProcessingResult{}, workflowErr
	}

	return shared.OrderProcessingResult{
		Status:  orderStatus,
		OrderID: order.OrderID,
	}, nil
	// </shingen:body>
}

var (
	_ = temporal.NewNonRetryableApplicationError
	_ = activities.ValidateOrder
	_ = activities.EnrichOrder
	_ = activities.PublishFulfillment
	_ = activities.UpdateDashboard
	_ = activities.ValidatePayment
)
