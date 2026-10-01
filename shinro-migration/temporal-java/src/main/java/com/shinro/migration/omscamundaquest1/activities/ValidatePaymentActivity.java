package com.shinro.migration.omscamundaquest1.activities;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;
import java.util.Map;

@ActivityInterface
public interface ValidatePaymentActivity {
  @ActivityMethod(name = "Validate-payment")
  Map<String, Object> validatePayment(Map<String, Object> variables);
}