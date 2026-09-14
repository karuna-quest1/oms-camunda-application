package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/sdk/temporal"

	"shinro-migration/oms-camunda-quest1/shared"
)

// ValidatePayment reimplements Camunda job worker "validate-payment".
//
// The original handler is in the findings (workers[].sourceCode); translate its
// business logic, dropping every Camunda engine call (complete/fail/throwError)
// — a Temporal activity just returns its result or an error.
func ValidatePayment(ctx context.Context, in shared.ValidatePaymentInput) (shared.ValidatePaymentOutput, error) {
	// <shingen:body activity:validate-payment>
	payment := in.Payment
	expectedAmount := in.ExpectedAmount

	// Validate payment
	if err := paymentValidationError(payment, expectedAmount); err != nil {
		// BPMN error -> boundary event loops back to the CapturePayment wait.
		return shared.ValidatePaymentOutput{}, temporal.NewNonRetryableApplicationError(
			err.Error(),
			"INVALID_PAYMENT",
			nil,
		)
	}

	return shared.ValidatePaymentOutput{Payment: payment}, nil
	// </shingen:body>
}

// paymentValidationError returns an error when the payment is invalid, else nil.
func paymentValidationError(payment shared.PaymentInput, expectedAmount float64) error {
	if payment.RRN == "" {
		return errors.New("payment RRN is empty")
	}
	if len(payment.RRN) != 12 {
		return fmt.Errorf("invalid RRN length %d; must be 12 digits", len(payment.RRN))
	}
	for _, c := range payment.RRN {
		if c < '0' || c > '9' {
			return errors.New("RRN must contain only digits")
		}
	}
	if payment.Amount <= 0 {
		return fmt.Errorf("invalid payment amount: %f", payment.Amount)
	}
	// Compare money as integer cents; exact double equality is unsafe for
	// amounts that aren't representable in binary floating point.
	if int64(payment.Amount*100) != int64(expectedAmount*100) {
		return fmt.Errorf("payment amount %.2f does not match order total %.2f",
			payment.Amount, expectedAmount)
	}
	return nil
}

var (
	_ = temporal.NewNonRetryableApplicationError
	_ = fmt.Sprintf
	_ = errors.New
	_ = json.Marshal
)
