package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"shinro-migration/oms-camunda-quest1/shared"
)

// FulfillmentMessage is the message published to the downstream fulfillment topic.
// Consumers must deduplicate on eventId (at-least-once delivery).
type FulfillmentMessage struct {
	EventID       string             `json:"event_id"`
	SchemaVersion int                `json:"schema_version"`
	CustomerID    string             `json:"customer_id"`
	OrderID       string             `json:"order_id"`
	PaymentDetails shared.PaymentInput `json:"payment_details"`
	Items         []shared.OrderItem  `json:"items"`
}

// PublishFulfillment reimplements Camunda job worker "publish-fulfillment".
//
// The original handler is in the findings (workers[].sourceCode); translate its
// business logic, dropping every Camunda engine call (complete/fail/throwError)
// — a Temporal activity just returns its result or an error.
func PublishFulfillment(ctx context.Context, in shared.PublishFulfillmentInput) (shared.PublishFulfillmentOutput, error) {
	// <shingen:body activity:publish-fulfillment>
	logger := activity.GetLogger(ctx)

	msg := FulfillmentMessage{
		EventID:       fmt.Sprintf("fulfillment:%s:v1", in.Order.OrderID),
		SchemaVersion: 1,
		CustomerID:    in.Order.CustomerID,
		OrderID:       in.Order.OrderID,
		PaymentDetails: in.Payment,
		Items:         in.Order.Items,
	}

	// In a real implementation, this would publish to a message broker.
	// For now, we log the message (matching the Java implementation).
	logger.Info("publish-fulfillment: published to order-fulfillment topic",
		"eventId", msg.EventID,
		"orderId", msg.OrderID,
		"customerId", msg.CustomerID,
		"itemCount", len(msg.Items),
		"paymentRRN", in.Payment.RRN)

	// Note: Consumers must deduplicate on eventId to handle at-least-once delivery.

	return shared.PublishFulfillmentOutput{}, nil
	// </shingen:body>
}

var (
	_ = temporal.NewNonRetryableApplicationError
	_ = fmt.Sprintf
	_ = errors.New
	_ = json.Marshal
)
