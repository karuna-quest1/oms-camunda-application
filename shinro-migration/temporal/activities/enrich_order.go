package activities

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"shinro-migration/oms-camunda-application/shared"
)

// EnrichOrder reimplements Camunda job worker "enrich-order"
// (OmsWorkers#enrichOrder).
//
// Enriches order items with PIM metadata (skuId, brandCode) and returns
// the enriched order. Safe to retry because enrichment is deterministic
// on the input:
//
//   - skuId:       "SKU-<orderId>-<n>" (1-based) when the item has none
//   - brandCode:   "BRAND-<itemId>" when the item has none
func EnrichOrder(ctx context.Context, in shared.EnrichOrderInput) (shared.EnrichOrderOutput, error) {
	logger := slog.Default().With("activity", "enrich-order")
	order := in.Order
	logger.InfoContext(ctx, "enrich-order started", "orderId", orderID(order))

	enriched := make([]shared.OrderItem, 0, len(order.Items))
	for i, item := range order.Items {
		sku := item.SkuId
		if strings.TrimSpace(sku) == "" {
			sku = fmt.Sprintf("SKU-%s-%d", order.OrderId, i+1)
		}
		brand := item.BrandCode
		if strings.TrimSpace(brand) == "" {
			brand = "BRAND-" + item.ItemId
		}
		enriched = append(enriched, shared.OrderItem{
			ItemId:     item.ItemId,
			Quantity:   item.Quantity,
			UnitPrice:  item.UnitPrice,
			SkuId:      sku,
			BrandCode:  brand,
		})
	}

	enrichedOrder := &shared.OrderInput{
		OrderId:     order.OrderId,
		CustomerId:  order.CustomerId,
		Items:       enriched,
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
	}

	logger.InfoContext(ctx, "enrich-order completed", "orderId", order.OrderId, "itemCount", len(enriched))
	return shared.EnrichOrderOutput{Order: enrichedOrder}, nil
}
