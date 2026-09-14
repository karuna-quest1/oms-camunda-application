package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/sdk/temporal"

	"shinro-migration/oms-camunda-quest1/shared"
)

// EnrichOrder reimplements Camunda job worker "enrich-order".
//
// The original handler is in the findings (workers[].sourceCode); translate its
// business logic, dropping every Camunda engine call (complete/fail/throwError)
// — a Temporal activity just returns its result or an error.
func EnrichOrder(ctx context.Context, in shared.EnrichOrderInput) (shared.EnrichOrderOutput, error) {
	// <shingen:body activity:enrich-order>
	order := in.Order

	// Enrich order items with PIM metadata (sku_id, brand_code)
	enriched := make([]shared.OrderItem, 0, len(order.Items))
	for i, item := range order.Items {
		sku := item.SkuID
		if sku == "" {
			sku = fmt.Sprintf("SKU-%s-%d", order.OrderID, i+1)
		}
		brand := item.BrandCode
		if brand == "" {
			brand = fmt.Sprintf("BRAND-%s", item.ItemID)
		}
		enriched = append(enriched, shared.OrderItem{
			ItemID:    item.ItemID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			SkuID:     sku,
			BrandCode: brand,
		})
	}

	enrichedOrder := shared.OrderInput{
		OrderID:     order.OrderID,
		CustomerID:  order.CustomerID,
		Items:       enriched,
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
	}

	return shared.EnrichOrderOutput{Order: enrichedOrder}, nil
	// </shingen:body>
}

var (
	_ = temporal.NewNonRetryableApplicationError
	_ = fmt.Sprintf
	_ = errors.New
	_ = json.Marshal
)
