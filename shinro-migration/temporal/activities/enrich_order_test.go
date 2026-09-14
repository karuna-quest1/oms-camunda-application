package activities

import (
	"testing"

	"go.temporal.io/sdk/testsuite"

	"shinro-migration/oms-camunda-quest1/shared"
)

func TestEnrichOrder(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(EnrichOrder)

	tests := []struct {
		name  string
		input shared.EnrichOrderInput
	}{
		{
			name: "enrich order with empty sku and brand",
			input: shared.EnrichOrderInput{
				Order: shared.OrderInput{
					OrderID:    "order-123",
					CustomerID: "customer-456",
					Items: []shared.OrderItem{
						{ItemID: "item-1", Quantity: 2, UnitPrice: 10.0},
						{ItemID: "item-2", Quantity: 1, UnitPrice: 20.0},
					},
					TotalAmount: 40.0,
					Currency:    "USD",
				},
			},
		},
		{
			name: "enrich order with existing sku and brand",
			input: shared.EnrichOrderInput{
				Order: shared.OrderInput{
					OrderID:    "order-456",
					CustomerID: "customer-789",
					Items: []shared.OrderItem{
						{ItemID: "item-3", Quantity: 1, UnitPrice: 15.0, SkuID: "SKU-EXISTING", BrandCode: "BRAND-EXISTING"},
					},
					TotalAmount: 15.0,
					Currency:    "USD",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteActivity(EnrichOrder, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var output shared.EnrichOrderOutput
			if err := result.Get(&output); err != nil {
				t.Fatalf("failed to get result: %v", err)
			}

			if len(output.Order.Items) != len(tt.input.Order.Items) {
				t.Errorf("expected %d items, got %d", len(tt.input.Order.Items), len(output.Order.Items))
			}

			for i, item := range output.Order.Items {
				if item.SkuID == "" {
					t.Errorf("item %d: SkuID should not be empty", i)
				}
				if item.BrandCode == "" {
					t.Errorf("item %d: BrandCode should not be empty", i)
				}

				// If input had existing values, they should be preserved
				if tt.input.Order.Items[i].SkuID != "" {
					if item.SkuID != tt.input.Order.Items[i].SkuID {
						t.Errorf("item %d: expected SkuID %s, got %s",
							i, tt.input.Order.Items[i].SkuID, item.SkuID)
					}
				}
				if tt.input.Order.Items[i].BrandCode != "" {
					if item.BrandCode != tt.input.Order.Items[i].BrandCode {
						t.Errorf("item %d: expected BrandCode %s, got %s",
							i, tt.input.Order.Items[i].BrandCode, item.BrandCode)
					}
				}
			}
		})
	}
}
