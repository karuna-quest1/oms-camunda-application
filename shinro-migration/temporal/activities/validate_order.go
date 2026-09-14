package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/sdk/temporal"

	"shinro-migration/oms-camunda-quest1/shared"
)

// ValidateOrder reimplements Camunda job worker "validate-order".
//
// The original handler is in the findings (workers[].sourceCode); translate its
// business logic, dropping every Camunda engine call (complete/fail/throwError)
// — a Temporal activity just returns its result or an error.
func ValidateOrder(ctx context.Context, in shared.ValidateOrderInput) (shared.ValidateOrderOutput, error) {
	// <shingen:body activity:validate-order>
	order := in.Order

	// Apply any pending SupportCorrection (delivered as correctionItems) before
	// re-validating — the parity of the Camunda applyCorrection logic.
	if len(in.CorrectionItems) > 0 {
		order.Items = in.CorrectionItems
	}

	// Validate order
	if err := validationError(order); err != nil {
		// Non-retryable business failure -> correction loop (not a job retry).
		return shared.ValidateOrderOutput{}, temporal.NewNonRetryableApplicationError(
			err.Error(),
			"INVALID_ORDER",
			nil,
		)
	}

	// Return the validated order
	return shared.ValidateOrderOutput{Order: order}, nil
	// </shingen:body>
}

// validationError returns an error when the order is invalid, else nil.
func validationError(order shared.OrderInput) error {
	if order.OrderID == "" {
		return errors.New("order_id is required")
	}
	if len(order.Items) == 0 {
		return errors.New("order must contain at least one item")
	}
	if order.TotalAmount <= 0 {
		return fmt.Errorf("invalid total_amount: %f", order.TotalAmount)
	}
	if order.CustomerID == "" {
		return errors.New("customer_id is required")
	}
	return nil
}

var (
	_ = temporal.NewNonRetryableApplicationError
	_ = fmt.Sprintf
	_ = errors.New
	_ = json.Marshal
)
