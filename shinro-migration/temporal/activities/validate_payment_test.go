package activities

import (
	"errors"
	"testing"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"shinro-migration/oms-camunda-quest1/shared"
)

func TestValidatePayment(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(ValidatePayment)

	tests := []struct {
		name        string
		input       shared.ValidatePaymentInput
		wantErr     bool
		wantErrType string
	}{
		{
			name: "valid payment",
			input: shared.ValidatePaymentInput{
				Payment: shared.PaymentInput{
					RRN:           "123456789012",
					Amount:        100.00,
					Currency:      "USD",
					PaymentMethod: "CARD",
					PaidAt:        "2024-01-01T10:00:00Z",
				},
				ExpectedAmount: 100.00,
			},
			wantErr: false,
		},
		{
			name: "empty RRN",
			input: shared.ValidatePaymentInput{
				Payment: shared.PaymentInput{
					RRN:           "",
					Amount:        100.00,
					Currency:      "USD",
					PaymentMethod: "CARD",
				},
				ExpectedAmount: 100.00,
			},
			wantErr:     true,
			wantErrType: "INVALID_PAYMENT",
		},
		{
			name: "invalid RRN length",
			input: shared.ValidatePaymentInput{
				Payment: shared.PaymentInput{
					RRN:           "12345",
					Amount:        100.00,
					Currency:      "USD",
					PaymentMethod: "CARD",
				},
				ExpectedAmount: 100.00,
			},
			wantErr:     true,
			wantErrType: "INVALID_PAYMENT",
		},
		{
			name: "non-digit RRN",
			input: shared.ValidatePaymentInput{
				Payment: shared.PaymentInput{
					RRN:           "12345678901A",
					Amount:        100.00,
					Currency:      "USD",
					PaymentMethod: "CARD",
				},
				ExpectedAmount: 100.00,
			},
			wantErr:     true,
			wantErrType: "INVALID_PAYMENT",
		},
		{
			name: "negative amount",
			input: shared.ValidatePaymentInput{
				Payment: shared.PaymentInput{
					RRN:           "123456789012",
					Amount:        -10.00,
					Currency:      "USD",
					PaymentMethod: "CARD",
				},
				ExpectedAmount: 100.00,
			},
			wantErr:     true,
			wantErrType: "INVALID_PAYMENT",
		},
		{
			name: "amount mismatch",
			input: shared.ValidatePaymentInput{
				Payment: shared.PaymentInput{
					RRN:           "123456789012",
					Amount:        50.00,
					Currency:      "USD",
					PaymentMethod: "CARD",
				},
				ExpectedAmount: 100.00,
			},
			wantErr:     true,
			wantErrType: "INVALID_PAYMENT",
		},
		{
			name: "floating point comparison",
			input: shared.ValidatePaymentInput{
				Payment: shared.PaymentInput{
					RRN:           "123456789012",
					Amount:        99.99,
					Currency:      "USD",
					PaymentMethod: "CARD",
				},
				ExpectedAmount: 99.99,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteActivity(ValidatePayment, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !temporal.IsApplicationError(err) {
					t.Fatalf("expected ApplicationError, got %T", err)
				}
				var appErr *temporal.ApplicationError
				if errors.As(err, &appErr) {
					if tt.wantErrType != "" && appErr.Type() != tt.wantErrType {
						t.Errorf("expected error type %s, got %s", tt.wantErrType, appErr.Type())
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var output shared.ValidatePaymentOutput
			if err := result.Get(&output); err != nil {
				t.Fatalf("failed to get result: %v", err)
			}

			if output.Payment.RRN != tt.input.Payment.RRN {
				t.Errorf("expected RRN %s, got %s", tt.input.Payment.RRN, output.Payment.RRN)
			}
		})
	}
}
