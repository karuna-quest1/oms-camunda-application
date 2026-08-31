package com.oms.dashboard;

import java.util.Optional;

/** Data access for the dashboard read model. */
public interface DashboardRepository {

  /** Idempotent upsert keyed by order id (safe under at-least-once delivery). */
  void upsert(DashboardStatus status);

  /**
   * Atomically inserts the row only if the order id is not already present.
   * Returns {@code true} if this call created the row, {@code false} if a row
   * for that order already existed. Used as an idempotency claim on order
   * creation — the primary key doubles as a per-order lock.
   */
  boolean insertIfAbsent(DashboardStatus status);

  /** Removes the projection row for an order (used to release a failed claim). */
  void delete(String orderId);

  /** Returns the latest projected status for an order, if one exists. */
  Optional<DashboardStatus> find(String orderId);
}
