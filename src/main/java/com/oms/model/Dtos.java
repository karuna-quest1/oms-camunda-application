package com.oms.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

/**
 * Inbound webhook request bodies and outbound response envelopes for the REST
 * gateway. Grouped here as nested records because each is a thin transport
 * shape (the Temporal app kept these in {@code models.go}).
 */
public final class Dtos {

  private Dtos() {}

  /** {@code POST /orders} body — a commerce-system order webhook. */
  public record CommerceWebhookRequest(
      @JsonProperty("order_id") String orderId,
      @JsonProperty("customer_id") String customerId,
      @JsonProperty("items") List<OrderItem> items,
      @JsonProperty("total_amount") double totalAmount,
      @JsonProperty("currency") String currency) {}

  /** {@code POST /orders/payment} body — a payment-gateway webhook. */
  public record PaymentWebhookRequest(
      @JsonProperty("order_id") String orderId,
      @JsonProperty("rrn") String rrn,
      @JsonProperty("amount") double amount,
      @JsonProperty("currency") String currency,
      @JsonProperty("payment_method") String paymentMethod,
      @JsonProperty("paid_at") String paidAt) {}

  /** {@code POST /orders/correction} body — a support-corrected order. */
  public record CorrectionWebhookRequest(
      @JsonProperty("order_id") String orderId,
      @JsonProperty("items") List<OrderItem> items) {}

  /** {@code POST /orders/cancel} body. */
  public record CancelWebhookRequest(
      @JsonProperty("order_id") String orderId, @JsonProperty("reason") String reason) {}

  /** {@code POST /orders} success response. */
  public record StartWorkflowResponse(
      @JsonProperty("process_instance_key") long processInstanceKey,
      @JsonProperty("order_id") String orderId) {}

  /** {@code GET /orders/status} response, served from the dashboard projection. */
  public record OrderStatusResponse(
      @JsonProperty("order_id") String orderId, @JsonProperty("status") OrderStatus status) {}

  /** Standard error envelope. */
  public record ErrorResponse(@JsonProperty("error") String error) {}
}
