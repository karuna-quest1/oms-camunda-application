package com.oms.model;

/**
 * Lifecycle states of an order, mirrored 1:1 from the Temporal implementation
 * ({@code models.OrderStatus}). These strings are what the dashboard read model
 * and the {@code GET /orders/status} response expose to customers.
 */
public enum OrderStatus {
  ORDER_RECEIVED,
  AWAITING_CORRECTION,
  PAYMENT_PENDING,
  PAYMENT_CAPTURED,
  FULFILLED,
  CANCELLED,
  EXPIRED,
  FAILED_PAYMENT;

  public static OrderStatus fromString(String s) {
    return OrderStatus.valueOf(s);
  }
}
