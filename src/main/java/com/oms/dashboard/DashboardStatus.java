package com.oms.dashboard;

import java.time.Instant;

/**
 * A row in the customer-facing dashboard read model. This projection is the
 * durable, query-optimised store that decouples customer status reads from the
 * Zeebe engine — the counterpart of the Temporal app's PostgreSQL
 * {@code orders_dashboard} table.
 */
public record DashboardStatus(String orderId, String status, Instant updatedAt) {}
