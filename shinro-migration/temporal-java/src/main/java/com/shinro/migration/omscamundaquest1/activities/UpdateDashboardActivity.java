package com.shinro.migration.omscamundaquest1.activities;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;
import java.util.Map;

@ActivityInterface
public interface UpdateDashboardActivity {
  @ActivityMethod(name = "Update-dashboard")
  void updateDashboard(Map<String, Object> variables, String status);
}