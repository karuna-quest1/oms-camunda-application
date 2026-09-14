package activities

import (
	"testing"

	"go.temporal.io/sdk/testsuite"

	"shinro-migration/oms-camunda-quest1/shared"
)

func TestUpdateDashboard(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(UpdateDashboard)

	tests := []struct {
		name    string
		input   shared.UpdateDashboardInput
		wantErr bool
	}{
		{
			name: "valid status ORDER_RECEIVED",
			input: shared.UpdateDashboardInput{
				OrderID: "order-123",
				Status:  "ORDER_RECEIVED",
			},
			wantErr: false,
		},
		{
			name: "valid status PAYMENT_PENDING",
			input: shared.UpdateDashboardInput{
				OrderID: "order-123",
				Status:  "PAYMENT_PENDING",
			},
			wantErr: false,
		},
		{
			name: "valid status FULFILLED",
			input: shared.UpdateDashboardInput{
				OrderID: "order-123",
				Status:  "FULFILLED",
			},
			wantErr: false,
		},
		{
			name: "invalid status",
			input: shared.UpdateDashboardInput{
				OrderID: "order-123",
				Status:  "INVALID_STATUS",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteActivity(UpdateDashboard, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var output shared.UpdateDashboardOutput
			if err := result.Get(&output); err != nil {
				t.Fatalf("failed to get result: %v", err)
			}
		})
	}
}
