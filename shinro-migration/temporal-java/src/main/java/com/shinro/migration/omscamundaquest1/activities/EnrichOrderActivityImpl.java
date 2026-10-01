package com.shinro.migration.omscamundaquest1.activities;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.shinro.migration.omscamundaquest1.domain.OrderInput;
import com.shinro.migration.omscamundaquest1.domain.OrderItem;
import io.temporal.activity.Activity;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class EnrichOrderActivityImpl implements EnrichOrderActivity {
  private static final Logger LOG = LoggerFactory.getLogger(EnrichOrderActivityImpl.class);
  private final ObjectMapper objectMapper;

  public EnrichOrderActivityImpl(ObjectMapper objectMapper) {
    this.objectMapper = objectMapper;
  }

  @Override
  public Map<String, Object> enrichOrder(Map<String, Object> variables) {
    OrderInput order = objectMapper.convertValue(variables.get("order"), OrderInput.class);
    LOG.info("enrich-order started orderId={}", order.orderId());

    List<OrderItem> enriched = new ArrayList<>();
    List<OrderItem> items = order.items() == null ? List.of() : order.items();
    for (int i = 0; i < items.size(); i++) {
      OrderItem item = items.get(i);
      String sku =
          item.skuId() == null || item.skuId().isBlank()
              ? "SKU-" + order.orderId() + "-" + (i + 1)
              : item.skuId();
      String brand =
          item.brandCode() == null || item.brandCode().isBlank()
              ? "BRAND-" + item.itemId()
              : item.brandCode();
      enriched.add(
          new OrderItem(item.itemId(), item.quantity(), item.unitPrice(), sku, brand));
    }

    OrderInput enrichedOrder =
        new OrderInput(
            order.orderId(),
            order.customerId(),
            enriched,
            order.totalAmount(),
            order.currency());
    LOG.info("enrich-order completed orderId={} itemCount={}", order.orderId(), enriched.size());
    return Map.of("order", enrichedOrder);
  }
}