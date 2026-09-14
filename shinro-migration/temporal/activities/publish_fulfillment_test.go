package activities

import (
	"testing"

	"go.temporal.io/sdk/testsuite"

	"shinro-migration/oms-camunda-quest1/shared"
)

func TestPublishFulfillment(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(PublishFulfillment)

	tests := []struct {
		name  string
		input shared.PublishFulfillmentInput
	}{
		{
			name: "publish fulfillment message",
			input: shared.PublishFulfillmentInput{
				Order: shared.OrderInput{
					OrderID:    "order-123",
					CustomerID: "customer-456",
					Items: []shared.OrderItem{
						{ItemID: "item-1", Quantity: 2, UnitPrice: 10.0, SkuID: "SKU-1", BrandCode: "BRAND-1"},
					},
					TotalAmount: 20.0,
					Currency:    "USD",
				},
				Payment: shared.PaymentInput{
					RRN:           "123456789012",
					Amount:        20.0,
					Currency:      "USD",
					PaymentMethod: "CARD",
					PaidAt:        "2024-01-01T10:00:00Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteActivity(PublishFulfillment, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var output shared.PublishFulfillmentOutput
			if err := result.Get(&output); err != nil {
				t.Fatalf("failed to get result: %v", err)
			}
		})
	}
}
