package activities

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"go.temporal.io/sdk/temporal"

	"shinro-migration/oms-camunda-application/shared"
)

// ValidateOrder reimplements Camunda job worker "validate-order"
// (CommerceWorkers#validateOrder).
//
// The source handler is a pure, in-memory validation of the commerce
// webhook order:
//
//  1. apply any pending SupportCorrection (delivered as the
//     "correctionItems" variable) by replacing the order's items —
//     parity of WorkflowState.ApplyCorrection in the original Temporal app;
//  2. run the business validation (order_id required, at least one item,
//     total_amount > 0, customer_id required);
//  3. on failure throw the BPMN error INVALID_ORDER (the boundary event
//     drives the correction loop); on success complete with the validated
//     order and clear the consumed correctionItems variable.
//
// No external I/O — validation is idempotent and safe to re-run.
func ValidateOrder(ctx context.Context, in shared.ValidateOrderInput) (shared.ValidateOrderOutput, error) {
	logger := slog.Default().With("activity", "validate-order")

	order := in.Order
	// Apply any pending SupportCorrection (delivered as correctionItems)
	// before re-validating — parity of WorkflowState.ApplyCorrection.
	order = applyCorrection(order, in.CorrectionItems)

	logger.InfoContext(ctx, "validate-order started", "orderId", orderID(order))

	if rejection := validationError(order); rejection != "" {
		logger.WarnContext(ctx, "order validation failed", "rejection", rejection)
		// Non-retryable business failure -> correction loop (not an activity
		// retry). BPMN error INVALID_ORDER -> boundary event Bnd_InvalidOrder.
		return shared.ValidateOrderOutput{}, temporal.NewNonRetryableApplicationError(
			rejection, shared.ErrInvalidOrder, nil)
	}

	logger.InfoContext(ctx, "validate-order completed", "orderId", order.OrderId)
	// Echo the (validated) order back; the consumed correction is cleared by
	// the workflow (correctionItems is nilled on re-entry).
	return shared.ValidateOrderOutput{Order: order}, nil
}

// applyCorrection replaces the order's items with the corrected ones when a
// non-empty correction is pending (CommerceWorkers#applyCorrection).
func applyCorrection(order *shared.OrderInput, correctionItems []shared.OrderItem) *shared.OrderInput {
	if order == nil || len(correctionItems) == 0 {
		return order
	}
	return &shared.OrderInput{
		OrderId:     order.OrderId,
		CustomerId:  order.CustomerId,
		Items:       correctionItems,
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
	}
}

// validationError returns nil when the order is valid, else the rejection
// reason (CommerceWorkers#validationError, same messages).
func validationError(order *shared.OrderInput) string {
	if order == nil || strings.TrimSpace(order.OrderId) == "" {
		return "order_id is required"
	}
	if len(order.Items) == 0 {
		return "order must contain at least one item"
	}
	if order.TotalAmount <= 0 {
		return "invalid total_amount: " + strconv.FormatFloat(order.TotalAmount, 'g', -1, 64)
	}
	if strings.TrimSpace(order.CustomerId) == "" {
		return "customer_id is required"
	}
	return ""
}

func orderID(o *shared.OrderInput) string {
	if o == nil {
		return ""
	}
	return o.OrderId
}
