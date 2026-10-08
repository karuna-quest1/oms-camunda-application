package activities

import (
	"context"
	"fmt"
	"log/slog"

	"shinro-migration/oms-camunda-application/shared"
)

// FulfillmentPublisher publishes a FulfillmentMessage to the downstream
// fulfillment topic ("order-fulfillment"). The source worker does not contain
// a broker client — it logs the publish (mocked, "exactly like the Temporal
// activity"), and the message contract requires consumers to deduplicate on
// EventId because delivery is at-least-once.
type FulfillmentPublisher interface {
	// Publish sends msg to the fulfillment topic. The publisher must be safe
	// to re-run: msg.EventId ("fulfillment:<orderId>:v1") is the idempotency
	// key — the downstream consumer must dedupe on it for retries to be safe.
	Publish(ctx context.Context, msg *shared.FulfillmentMessage) error
}

// FulfillmentPublisherImpl is the default publisher. It mirrors the source
// worker's mocked publish: it constructs the wire message and logs it. It is
// NOT a broker client — wire a real Kafka/Pub-Sub publisher here (keeping the
// EventId as the dedup key) for production traffic.
type FulfillmentPublisherImpl struct{}

// Publish implements FulfillmentPublisher (mock parity with the source worker).
func (FulfillmentPublisherImpl) Publish(ctx context.Context, msg *shared.FulfillmentMessage) error {
	slog.Default().InfoContext(ctx,
		"publish-fulfillment: published to order-fulfillment topic",
		"eventId", msg.EventId,
		"orderId", msg.OrderId,
		"customerId", msg.CustomerId,
		"itemCount", len(msg.Items),
		"paymentRRN", rrnString(msg.Payment),
	)
	return nil
}

// fulfillmentPublisher is the publisher used by PublishFulfillment; replace
// via SetFulfillmentPublisher.
var fulfillmentPublisher FulfillmentPublisher = FulfillmentPublisherImpl{}

// SetFulfillmentPublisher installs a real publisher before the worker runs.
func SetFulfillmentPublisher(p FulfillmentPublisher) {
	if p != nil {
		fulfillmentPublisher = p
	}
}

// PublishFulfillment reimplements Camunda job worker "publish-fulfillment"
// (OmsWorkers#publishToFulfillment).
//
// Builds the FulfillmentMessage (eventId "fulfillment:<orderId>:v1",
// schemaVersion 1, payment details, enriched items) and publishes it.
// Idempotent: the event id is derived from the order id, so retries
// publish the same logical event.
func PublishFulfillment(ctx context.Context, in shared.PublishFulfillmentInput) (shared.PublishFulfillmentOutput, error) {
	logger := slog.Default().With("activity", "publish-fulfillment")
	order := in.Order
	payment := in.Payment

	// Idempotency key: consumers must dedupe on this eventId.
	msg := &shared.FulfillmentMessage{
		EventId:       fmt.Sprintf("fulfillment:%s:v1", order.OrderId),
		SchemaVersion: 1,
		CustomerId:    order.CustomerId,
		OrderId:       order.OrderId,
		Payment:       payment,
		Items:         order.Items,
	}

	if err := fulfillmentPublisher.Publish(ctx, msg); err != nil {
		// Transient publish failure -> retryable error (activity is retried
		// per its RetryPolicy; consumers dedupe on eventId).
		return shared.PublishFulfillmentOutput{}, fmt.Errorf("publish fulfillment event %s: %w", msg.EventId, err)
	}

	logger.InfoContext(ctx, "publish-fulfillment completed",
		"eventId", msg.EventId, "orderId", msg.OrderId)
	return shared.PublishFulfillmentOutput{EventId: msg.EventId}, nil
}
