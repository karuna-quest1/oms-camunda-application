package com.shinro.migration.omscamundaquest1.activities;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;
import java.util.Map;

@ActivityInterface
public interface EnrichOrderActivity {
  @ActivityMethod(name = "Enrich-order")
  Map<String, Object> enrichOrder(Map<String, Object> variables);
}