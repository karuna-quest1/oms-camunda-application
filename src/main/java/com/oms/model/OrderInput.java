package com.oms.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

/**
 * Canonical order payload carried as the {@code order} process variable, and
 * the shape the {@code validate-order} / {@code enrich-order} workers read and
 * write back. Equivalent to {@code models.OrderInput} in the Temporal app.
 *
 * <p>Note: unlike Temporal, PII (customer email / name) is deliberately kept
 * out of the process. Only the {@code customerId} reference travels through
 * Zeebe; contact details live solely in the dashboard projection.
 */
public record OrderInput(
    @JsonProperty("order_id") String orderId,
    @JsonProperty("customer_id") String customerId,
    @JsonProperty("items") List<OrderItem> items,
    @JsonProperty("total_amount") double totalAmount,
    @JsonProperty("currency") String currency) {}
