package com.shinro.migration.omscamundaquest1.activities;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;
import java.util.Map;

@ActivityInterface
public interface PublishFulfillmentActivity {
  @ActivityMethod(name = "Publish-fulfillment")
  void publishToFulfillment(Map<String, Object> variables);
}