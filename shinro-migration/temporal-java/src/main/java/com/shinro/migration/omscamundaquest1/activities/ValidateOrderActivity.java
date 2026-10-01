package com.shinro.migration.omscamundaquest1.activities;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;
import java.util.Map;

@ActivityInterface
public interface ValidateOrderActivity {
  @ActivityMethod(name = "Validate-order")
  Map<String, Object> validateOrder(Map<String, Object> variables);
}