package com.shinro.migration.omscamundaquest1.activities;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.shinro.migration.omscamundaquest1.domain.OrderInput;
import com.shinro.migration.omscamundaquest1.domain.OrderItem;
import io.temporal.activity.Activity;
import io.temporal.failure.ApplicationFailure;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ValidateOrderActivityImpl implements ValidateOrderActivity {
  private static final Logger LOG = LoggerFactory.getLogger(ValidateOrderActivityImpl.class);
  private final ObjectMapper objectMapper;

  public ValidateOrderActivityImpl(ObjectMapper objectMapper) {
    this.objectMapper = objectMapper;
  }

  @Override
  public Map<String, Object> validateOrder(Map<String, Object> variables) {
    OrderInput order = objectMapper.convertValue(variables.get("order"), OrderInput.class);

    // Apply any pending SupportCorrection (delivered as correctionItems) before
    // re-validating — the parity of WorkflowState.ApplyCorrection.
    Object correctionItemsRaw = variables.get("correctionItems");
    if (correctionItemsRaw != null) {
      order = applyCorrection(order, correctionItemsRaw);
    }

    LOG.info("validate-order started orderId={}", order == null ? null : order.orderId());

    String rejection = validationError(order);
    if (rejection != null) {
      LOG.warn("Order validation failed: {}", rejection);
      
      // Non-retryable business failure -> boundary event loops back to correction wait
      throw ApplicationFailure.newNonRetryableFailure(rejection, "INVALID_ORDER");
    }

    // Mock commerce validation is idempotent; echo the (validated) order back
    LOG.info("validate-order completed orderId={}", order.orderId());
    Map<String, Object> out = new HashMap<>();
    out.put("order", order);
    out.put("correctionItems", null);
    return out;
  }

  /** Replaces the order's items with corrected ones when a correction is pending. */
  private OrderInput applyCorrection(OrderInput order, Object correctionItemsRaw) {
    if (order == null || correctionItemsRaw == null) {
      return order;
    }
    
    // For simplicity in this migration we assume it's a list
    List<OrderItem> items = (List<OrderItem>) correctionItemsRaw;
    if (items == null || items.isEmpty()) {
      return order;
    }
    
    return new OrderInput(
        order.orderId(), 
        order.customerId(), 
        items, 
        order.totalAmount(), 
        order.currency());
  }

  /** Returns {@code null} when the order is valid, else the rejection reason. */
  private static String validationError(OrderInput order) {
    if (order == null || order.orderId() == null || order.orderId().isBlank()) {
      return "order_id is required";
    }
    if (order.items() == null || order.items().isEmpty()) {
      return "order must contain at least one item";
    }
    if (order.totalAmount() <= 0) {
      return "invalid total_amount: " + order.totalAmount();
    }
    if (order.customerId() == null || order.customerId().isBlank()) {
      return "customer_id is required";
    }
    return null;
  }
}