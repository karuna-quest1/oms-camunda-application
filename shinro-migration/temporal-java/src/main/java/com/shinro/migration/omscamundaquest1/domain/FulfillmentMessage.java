package com.shinro.migration.omscamundaquest1.domain;

import java.util.List;

public record FulfillmentMessage(
    String eventId,
    int version,
    String customerId,
    String orderId,
    PaymentInput payment,
    List<OrderItem> items) {}