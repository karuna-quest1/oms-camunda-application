package com.oms.dashboard;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.Optional;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

/**
 * JDBC-backed dashboard projection. Uses an update-first, insert-on-miss upsert
 * guarded by the primary key so the same code works on both the embedded H2
 * default and PostgreSQL without dialect-specific {@code ON CONFLICT} /
 * {@code MERGE} syntax — and stays correct under concurrent first writes.
 */
@Repository
public class JdbcDashboardRepository implements DashboardRepository {

  private final JdbcTemplate jdbc;

  public JdbcDashboardRepository(JdbcTemplate jdbc) {
    this.jdbc = jdbc;
  }

  @Override
  public void upsert(DashboardStatus status) {
    Timestamp updatedAt = Timestamp.from(status.updatedAt());
    if (update(status.orderId(), status.status(), updatedAt) == 0) {
      // Row absent: try to insert. If a concurrent writer inserted it first, the
      // primary key rejects our insert (DuplicateKeyException) and we fall back
      // to an update — so the write is never silently lost to a race.
      try {
        insert(status.orderId(), status.status(), updatedAt);
      } catch (DuplicateKeyException e) {
        update(status.orderId(), status.status(), updatedAt);
      }
    }
  }

  @Override
  public boolean insertIfAbsent(DashboardStatus status) {
    try {
      insert(status.orderId(), status.status(), Timestamp.from(status.updatedAt()));
      return true;
    } catch (DuplicateKeyException e) {
      return false;
    }
  }

  @Override
  public void delete(String orderId) {
    jdbc.update("DELETE FROM orders_dashboard WHERE order_id = ?", orderId);
  }

  private int update(String orderId, String status, Timestamp updatedAt) {
    return jdbc.update(
        "UPDATE orders_dashboard SET status = ?, updated_at = ? WHERE order_id = ?",
        status,
        updatedAt,
        orderId);
  }

  private void insert(String orderId, String status, Timestamp updatedAt) {
    jdbc.update(
        "INSERT INTO orders_dashboard (order_id, status, updated_at) VALUES (?, ?, ?)",
        orderId,
        status,
        updatedAt);
  }

  @Override
  public Optional<DashboardStatus> find(String orderId) {
    try {
      DashboardStatus row =
          jdbc.queryForObject(
              "SELECT order_id, status, updated_at FROM orders_dashboard WHERE order_id = ?",
              (rs, n) ->
                  new DashboardStatus(
                      rs.getString("order_id"),
                      rs.getString("status"),
                      toInstant(rs.getTimestamp("updated_at"))),
              orderId);
      return Optional.ofNullable(row);
    } catch (EmptyResultDataAccessException e) {
      return Optional.empty();
    }
  }

  private static Instant toInstant(Timestamp ts) {
    return ts == null ? null : ts.toInstant();
  }
}
