package activities

import (
	"errors"
	"testing"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"shinro-migration/oms-camunda-quest1/shared"
)

func TestValidateOrder(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(ValidateOrder)

	tests := []struct {
		name          string
		input         shared.ValidateOrderInput
		wantErr       bool
		wantErrType   string
		wantErrMsg    string
	}{
		{
			name: "valid order",
			input: shared.ValidateOrderInput{
				Order: shared.OrderInput{
					OrderID:     "order-123",
					CustomerID:  "customer-456",
					Items: []shared.OrderItem{
						{ItemID: "item-1", Quantity: 2, UnitPrice: 10.0},
					},
					TotalAmount: 20.0,
					Currency:    "USD",
				},
			},
			wantErr: false,
		},
		{
			name: "missing order_id",
			input: shared.ValidateOrderInput{
				Order: shared.OrderInput{
					CustomerID:  "customer-456",
					Items:       []shared.OrderItem{{ItemID: "item-1", Quantity: 1, UnitPrice: 10.0}},
					TotalAmount: 10.0,
					Currency:    "USD",
				},
			},
			wantErr:     true,
			wantErrType: "INVALID_ORDER",
			wantErrMsg:  "order_id is required",
		},
		{
			name: "empty items",
			input: shared.ValidateOrderInput{
				Order: shared.OrderInput{
					OrderID:     "order-123",
					CustomerID:  "customer-456",
					Items:       []shared.OrderItem{},
					TotalAmount: 10.0,
					Currency:    "USD",
				},
			},
			wantErr:     true,
			wantErrType: "INVALID_ORDER",
			wantErrMsg:  "order must contain at least one item",
		},
		{
			name: "invalid total_amount",
			input: shared.ValidateOrderInput{
				Order: shared.OrderInput{
					OrderID:     "order-123",
					CustomerID:  "customer-456",
					Items:       []shared.OrderItem{{ItemID: "item-1", Quantity: 1, UnitPrice: 10.0}},
					TotalAmount: -5.0,
					Currency:    "USD",
				},
			},
			wantErr:     true,
			wantErrType: "INVALID_ORDER",
		},
		{
			name: "missing customer_id",
			input: shared.ValidateOrderInput{
				Order: shared.OrderInput{
					OrderID:     "order-123",
					Items:       []shared.OrderItem{{ItemID: "item-1", Quantity: 1, UnitPrice: 10.0}},
					TotalAmount: 10.0,
					Currency:    "USD",
				},
			},
			wantErr:     true,
			wantErrType: "INVALID_ORDER",
			wantErrMsg:  "customer_id is required",
		},
		{
			name: "order with correction items",
			input: shared.ValidateOrderInput{
				Order: shared.OrderInput{
					OrderID:     "order-123",
					CustomerID:  "customer-456",
					Items:       []shared.OrderItem{{ItemID: "bad-item", Quantity: 1, UnitPrice: 10.0}},
					TotalAmount: 20.0,
					Currency:    "USD",
				},
				CorrectionItems: []shared.OrderItem{
					{ItemID: "corrected-item-1", Quantity: 1, UnitPrice: 10.0},
					{ItemID: "corrected-item-2", Quantity: 1, UnitPrice: 10.0},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteActivity(ValidateOrder, tt.input)

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
					if tt.wantErrMsg != "" && appErr.Message() != tt.wantErrMsg {
						t.Errorf("expected error message %s, got %s", tt.wantErrMsg, appErr.Message())
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var output shared.ValidateOrderOutput
			if err := result.Get(&output); err != nil {
				t.Fatalf("failed to get result: %v", err)
			}

			// Verify correction items are applied
			if len(tt.input.CorrectionItems) > 0 {
				if len(output.Order.Items) != len(tt.input.CorrectionItems) {
					t.Errorf("expected %d items after correction, got %d",
						len(tt.input.CorrectionItems), len(output.Order.Items))
				}
			}
		})
	}
}
