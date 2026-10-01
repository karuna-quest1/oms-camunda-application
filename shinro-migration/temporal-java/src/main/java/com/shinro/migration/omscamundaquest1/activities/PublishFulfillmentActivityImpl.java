package com.shinro.migration.omscamundaquest1.activities;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.shinro.migration.omscamundaquest1.domain.FulfillmentMessage;
import com.shinro.migration.omscamundaquest1.domain.OrderInput;
import com.shinro.migration.omscamundaquest1.domain.PaymentInput;
import io.temporal.activity.Activity;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class PublishFulfillmentActivityImpl implements PublishFulfillmentActivity {
  private static final Logger LOG = LoggerFactory.getLogger(PublishFulfillmentActivityImpl.class);
  private final ObjectMapper objectMapper;

  public PublishFulfillmentActivityImpl(ObjectMapper objectMapper) {
    this.objectMapper = objectMapper;
  }

  @Override
  public void publishToFulfillment(Map<String, Object> variables) {
    OrderInput order = objectMapper.convertValue(variables.get("order"), OrderInput.class);
    PaymentInput payment = objectMapper.convertValue(variables.get("payment"), PaymentInput.class);

    FulfillmentMessage msg = new FulfillmentMessage(
        "fulfillment:" + order.orderId() + ":v1",
        1,
        order.customerId(),
        order.orderId(),
        payment,
        order.items());

    LOG.info(
        "publish-fulfillment: published to order-fulfillment topic eventId={} orderId={} customerId={} itemCount={} paymentRRN={}",
        msg.eventId(),
        msg.orderId(),
        msg.customerId(),
        msg.items() == null ? 0 : msg.items().size(),
        payment == null ? null : payment.rrn());
  }
}