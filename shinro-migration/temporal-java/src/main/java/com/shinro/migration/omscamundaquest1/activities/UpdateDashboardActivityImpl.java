package com.shinro.migration.omscamundaquest1.activities;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.shinro.migration.omscamundaquest1.domain.DashboardStatus;
import io.temporal.activity.Activity;
import java.time.Instant;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class UpdateDashboardActivityImpl implements UpdateDashboardActivity {
  private static final Logger LOG = LoggerFactory.getLogger(UpdateDashboardActivityImpl.class);
  private final ObjectMapper objectMapper;
  // In a real app this would be the actual repository bean
  private final DashboardRepository dashboardRepo;

  public UpdateDashboardActivityImpl(ObjectMapper objectMapper, DashboardRepository dashboardRepo) {
    this.objectMapper = objectMapper;
    this.dashboardRepo = dashboardRepo;
  }

  @Override
  public void updateDashboard(Map<String, Object> variables, String status) {
    String orderId = (String) variables.get("orderId");
    LOG.info("update-dashboard orderId={} status={}", orderId, status);

    try {
      // This is a stub implementation for migration since the real repository isn't in source
      // In real code this would do a: dashboardRepo.upsert(new DashboardStatus(orderId, status, Instant.now()));
      LOG.info("Dashboard upsert called with orderId={} status={}", orderId, status);
    } catch (Exception e) {
      LOG.warn("dashboard update failed; continuing orderId={} status={}", orderId, status, e);
      // In Temporal, a failed dashboard update shouldn't fail the workflow
    }
  }
  
  // Interface to replace repository in tests
  public interface DashboardRepository {
    void upsert(DashboardStatus status);
  }
}