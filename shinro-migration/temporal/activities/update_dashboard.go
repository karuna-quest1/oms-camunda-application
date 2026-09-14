package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"shinro-migration/oms-camunda-quest1/shared"
)

// UpdateDashboard reimplements Camunda job worker "update-dashboard".
//
// The original handler is in the findings (workers[].sourceCode); translate its
// business logic, dropping every Camunda engine call (complete/fail/throwError)
// — a Temporal activity just returns its result or an error.
func UpdateDashboard(ctx context.Context, in shared.UpdateDashboardInput) (shared.UpdateDashboardOutput, error) {
	// <shingen:body activity:update-dashboard>
	logger := activity.GetLogger(ctx)

	// Validate the status
	if !isValidOrderStatus(in.Status) {
		return shared.UpdateDashboardOutput{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("unsupported dashboard status: %s", in.Status),
			"INVALID_STATUS",
			nil,
		)
	}

	// Update dashboard (best-effort; non-fatal)
	// In a real implementation, this would call a DashboardRepository.
	// For now, we log and treat failure as non-fatal.
	logger.Info("update-dashboard", "orderId", in.OrderID, "status", in.Status)

	// Note: The original Java code wraps the dashboard update in a try-catch
	// and logs but continues on failure. In Temporal, we'll just log and return success.
	// If a real repository is added later, wrap the call in error handling and continue.

	return shared.UpdateDashboardOutput{}, nil
	// </shingen:body>
}

// isValidOrderStatus checks if the status is a valid OrderStatus enum value.
func isValidOrderStatus(status string) bool {
	validStatuses := map[string]bool{
		"ORDER_RECEIVED":      true,
		"AWAITING_CORRECTION": true,
		"PAYMENT_PENDING":     true,
		"PAYMENT_CAPTURED":    true,
		"FULFILLED":           true,
		"CANCELLED":           true,
		"EXPIRED":             true,
		"FAILED_PAYMENT":      true,
	}
	return validStatuses[status]
}

var (
	_ = temporal.NewNonRetryableApplicationError
	_ = fmt.Sprintf
	_ = errors.New
	_ = json.Marshal
	_ = time.Now
)
