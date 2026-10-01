package com.shinro.migration.omscamundaquest1.activities;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.shinro.migration.omscamundaquest1.domain.OrderInput;
import com.shinro.migration.omscamundaquest1.domain.PaymentInput;
import io.temporal.activity.Activity;
import io.temporal.failure.ApplicationFailure;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ValidatePaymentActivityImpl implements ValidatePaymentActivity {
  private static final Logger LOG = LoggerFactory.getLogger(ValidatePaymentActivityImpl.class);
  private final ObjectMapper objectMapper;

  public ValidatePaymentActivityImpl(ObjectMapper objectMapper) {
    this.objectMapper = objectMapper;
  }

  @Override
  public Map<String, Object> validatePayment(Map<String, Object> variables) {
    PaymentInput payment = objectMapper.convertValue(variables.get("payment"), PaymentInput.class);
    OrderInput order = objectMapper.convertValue(variables.get("order"), OrderInput.class);
    
    double expectedAmount = order == null ? 0 : order.totalAmount();

    LOG.info("validate-payment started rrn={}", payment == null ? null : payment.rrn());

    String rejection = validationError(payment, expectedAmount);
    if (rejection != null) {
      LOG.warn("Payment validation failed: {}", rejection);
      
      // BPMN error -> boundary event loops back to the CapturePayment wait
      throw ApplicationFailure.newNonRetryableFailure(rejection, "INVALID_PAYMENT");
    }

    LOG.info("validate-payment completed rrn={}", payment.rrn());
    
    return Map.of("payment", payment);
  }

  /** Returns {@code null} when the payment is valid, else the rejection reason. */
  private static String validationError(PaymentInput payment, double expectedAmount) {
    if (payment == null || payment.rrn() == null || payment.rrn().isEmpty()) {
      return "payment RRN is empty";
    }
    if (payment.rrn().length() != 12) {
      return "invalid RRN length " + payment.rrn().length() + "; must be 12 digits";
    }
    for (int i = 0; i < payment.rrn().length(); i++) {
      char c = payment.rrn().charAt(i);
      if (c < '0' || c > '9') {
        return "RRN must contain only digits";
      }
    }
    if (payment.amount() <= 0) {
      return "invalid payment amount: " + payment.amount();
    }
    // Compare money as integer cents; exact double equality is unsafe for
    // amounts that aren't representable in binary floating point.
    if (Math.round(payment.amount() * 100) != Math.round(expectedAmount * 100)) {
      return String.format(
          "payment amount %.2f does not match order total %.2f", payment.amount(), expectedAmount);
    }
    return null;
  }
}