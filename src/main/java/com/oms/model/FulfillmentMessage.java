package com.oms.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

/**
 * Message published to the downstream fulfillment topic by the
 * {@code publish-fulfillment} worker. Job workers are at-least-once (a worker
 * may be re-run after a timeout), so consumers must deduplicate on
 * {@code eventId} — exactly as noted for the Temporal activity.
 */
public record FulfillmentMessage(
    @JsonProperty("event_id") String eventId,
    @JsonProperty("schema_version") int schemaVersion,
    @JsonProperty("customer_id") String customerId,
    @JsonProperty("order_id") String orderId,
    @JsonProperty("payment_details") PaymentInput paymentDetails,
    @JsonProperty("items") List<OrderItem> items) {}
