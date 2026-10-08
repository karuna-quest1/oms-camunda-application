package activities

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"go.temporal.io/sdk/temporal"

	"shinro-migration/oms-camunda-application/shared"
)

// ValidatePayment reimplements Camunda job worker "validate-payment"
// (OmsWorkers#validatePayment).
//
// Business failures (bad RRN, wrong amount) raise the BPMN error
// INVALID_PAYMENT, caught by the boundary event that loops back to the
// CapturePayment wait — the parity of the original Temporal payment retry
// loop that awaited a fresh CapturePayment signal. Transient/system
// failures return plain (retryable) errors and are retried up to the
// task's configured retry count (5).
func ValidatePayment(ctx context.Context, in shared.ValidatePaymentInput) (shared.ValidatePaymentOutput, error) {
	logger := slog.Default().With("activity", "validate-payment")
	payment := in.Payment
	expectedAmount := in.Order.TotalAmountOrZero()

	logger.InfoContext(ctx, "validate-payment started", "rrn", rrnString(payment))

	if rejection := paymentValidationError(payment, expectedAmount); rejection != "" {
		logger.WarnContext(ctx, "payment validation failed", "rejection", rejection)
		// BPMN error -> boundary event loops back to the CapturePayment wait.
		// No need to clear the payment variable: the next CapturePayment
		// message overwrites it (mirrors the Temporal loop awaiting a fresh
		// signal).
		return shared.ValidatePaymentOutput{}, temporal.NewNonRetryableApplicationError(
			rejection, shared.ErrInvalidPayment, nil)
	}

	logger.InfoContext(ctx, "validate-payment completed", "rrn", payment.Rrn)
	// Complete with the payment variable (echoed unchanged).
	return shared.ValidatePaymentOutput{Payment: payment}, nil
}

// paymentValidationError returns "" when the payment is valid, else the
// rejection reason (OmsWorkers#validationError, same messages).
func paymentValidationError(payment *shared.PaymentInput, expectedAmount float64) string {
	if payment == nil || payment.Rrn == "" {
		return "payment RRN is empty"
	}
	if len(payment.Rrn) != 12 {
		return fmt.Sprintf("invalid RRN length %d; must be 12 digits", len(payment.Rrn))
	}
	for i := 0; i < len(payment.Rrn); i++ {
		c := payment.Rrn[i]
		if c < '0' || c > '9' {
			return "RRN must contain only digits"
		}
	}
	if payment.Amount <= 0 {
		return fmt.Sprintf("invalid payment amount: %v", payment.Amount)
	}
	// Compare money as integer cents; exact float equality is unsafe for
	// amounts that aren't representable in binary floating point.
	if math.Round(payment.Amount*100) != math.Round(expectedAmount*100) {
		return fmt.Sprintf("payment amount %.2f does not match order total %.2f", payment.Amount, expectedAmount)
	}
	return ""
}

func rrnString(p *shared.PaymentInput) string {
	if p == nil {
		return ""
	}
	return p.Rrn
}
