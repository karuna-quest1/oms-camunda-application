package com.shinro.migration.omscamundaquest1.domain;

import java.time.Instant;

public record DashboardStatus(String orderId, String status, Instant timestamp) {}