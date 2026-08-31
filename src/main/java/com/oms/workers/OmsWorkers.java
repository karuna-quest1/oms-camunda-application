package com.oms.workers;

import static com.oms.process.ProcessConstants.ERR_INVALID_PAYMENT;
import static com.oms.process.ProcessConstants.HEADER_STATUS;
import static com.oms.process.ProcessConstants.JOB_ENRICH_ORDER;
import static com.oms.process.ProcessConstants.JOB_PUBLISH_FULFILLMENT;
import static com.oms.process.ProcessConstants.JOB_UPDATE_DASHBOARD;
import static com.oms.process.ProcessConstants.JOB_VALIDATE_PAYMENT;
import static com.oms.process.ProcessConstants.VAR_ORDER;
import static com.oms.process.ProcessConstants.VAR_ORDER_ID;
import static com.oms.process.ProcessConstants.VAR_PAYMENT;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.oms.dashboard.DashboardRepository;
import com.oms.dashboard.DashboardStatus;
import com.oms.model.FulfillmentMessage;
import com.oms.model.OrderInput;
import com.oms.model.OrderItem;
import com.oms.model.OrderStatus;
import com.oms.model.PaymentInput;
import io.camunda.zeebe.client.api.response.ActivatedJob;
import io.camunda.zeebe.client.api.worker.JobClient;
import io.camunda.zeebe.spring.client.annotation.JobWorker;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

/**
 * The OMS-queue activities from the Temporal app, re-expressed as Zeebe job
 * workers: {@code validate-payment}, {@code enrich-order},
 * {@code update-dashboard} and {@code publish-fulfillment}.
 *
 * <p>Every worker is idempotent and safe to re-run, because Zeebe (like
 * Temporal) delivers jobs at least once.
 */
@Component
public class OmsWorkers {

  private static final Logger LOG = LoggerFactory.getLogger(OmsWorkers.class);

  private final ObjectMapper objectMapper;
  private final DashboardRepository dashboardRepo;

  public OmsWorkers(ObjectMapper objectMapper, DashboardRepository dashboardRepo) {
    this.objectMapper = objectMapper;
    this.dashboardRepo = dashboardRepo;
  }

  // ------------------------------------------------------------------ payment

  /**
   * Validates the payment RRN and amount. Business failures raise the BPMN error
   * {@code INVALID_PAYMENT}, caught by the boundary event that loops back to the
   * {@code CapturePayment} wait — the parity of the Temporal payment retry loop
   * that cleared the invalid payment and awaited a fresh {@code CapturePayment}
   * signal. Transient/system failures instead fall through as job failures and
   * are retried up to the task's configured retry count (5).
   */
  @JobWorker(type = JOB_VALIDATE_PAYMENT, autoComplete = false)
  public void validatePayment(JobClient client, ActivatedJob job) {
    Map<String, Object> vars = job.getVariablesAsMap();
    PaymentInput payment = objectMapper.convertValue(vars.get(VAR_PAYMENT), PaymentInput.class);
    OrderInput order = objectMapper.convertValue(vars.get(VAR_ORDER), OrderInput.class);
    double expectedAmount = order == null ? 0 : order.totalAmount();

    LOG.info("validate-payment started rrn={}", payment == null ? null : payment.rrn());

    String rejection = validationError(payment, expectedAmount);
    if (rejection != null) {
      LOG.warn("Payment validation failed: {}", rejection);
      // BPMN error -> boundary event loops back to the CapturePayment wait. No
      // need to clear the payment variable: the next CapturePayment message
      // overwrites it (mirrors the Temporal loop awaiting a fresh signal).
      client
          .newThrowErrorCommand(job)
          .errorCode(ERR_INVALID_PAYMENT)
          .errorMessage(rejection)
          .send()
          .join();
      return;
    }

    LOG.info("validate-payment completed rrn={}", payment.rrn());
    client.newCompleteCommand(job).variables(Map.of(VAR_PAYMENT, payment)).send().join();
  }

  /** Returns {@code null} when the payment is valid, else the rejection reason. */
  private static String validationError(PaymentInput payment, double expectedAmount) {
    if (payment == null || payment.rrn() == null || payment.rrn().isEmpty()) {
      return "payment RRN is empty";
    }
    if (payment.rrn().length() != 12) {
      return "invalid RRN length " + payment.rrn().length() + "; must be 12 digits";
    }
    for (int i = 0; i < payment.rrn().length(); i++) {
      char c = payment.rrn().charAt(i);
      if (c < '0' || c > '9') {
        return "RRN must contain only digits";
      }
    }
    if (payment.amount() <= 0) {
      return "invalid payment amount: " + payment.amount();
    }
    // Compare money as integer cents; exact double equality is unsafe for
    // amounts that aren't representable in binary floating point.
    if (Math.round(payment.amount() * 100) != Math.round(expectedAmount * 100)) {
      return String.format(
          "payment amount %.2f does not match order total %.2f", payment.amount(), expectedAmount);
    }
    return null;
  }

  // ------------------------------------------------------------------- enrich

  /**
   * Enriches order items with PIM metadata (sku_id, brand_code) and writes the
   * enriched {@code order} back. Equivalent to {@code EnrichWithPIM}; safe to
   * retry because enrichment is deterministic on the input.
   */
  @JobWorker(type = JOB_ENRICH_ORDER)
  public Map<String, Object> enrichOrder(ActivatedJob job) {
    OrderInput order =
        objectMapper.convertValue(job.getVariablesAsMap().get(VAR_ORDER), OrderInput.class);
    LOG.info("enrich-order started orderId={}", order.orderId());

    List<OrderItem> enriched = new ArrayList<>();
    List<OrderItem> items = order.items() == null ? List.of() : order.items();
    for (int i = 0; i < items.size(); i++) {
      OrderItem item = items.get(i);
      String sku =
          item.skuId() == null || item.skuId().isBlank()
              ? "SKU-" + order.orderId() + "-" + (i + 1)
              : item.skuId();
      String brand =
          item.brandCode() == null || item.brandCode().isBlank()
              ? "BRAND-" + item.itemId()
              : item.brandCode();
      enriched.add(
          new OrderItem(item.itemId(), item.quantity(), item.unitPrice(), sku, brand));
    }

    OrderInput enrichedOrder =
        new OrderInput(
            order.orderId(),
            order.customerId(),
            enriched,
            order.totalAmount(),
            order.currency());
    LOG.info("enrich-order completed orderId={} itemCount={}", order.orderId(), enriched.size());
    return Map.of(VAR_ORDER, enrichedOrder);
  }

  // ---------------------------------------------------------------- dashboard

  /**
   * Upserts the customer dashboard read model. The target status is carried on
   * the service task's {@code status} job header, so a single worker serves
   * every dashboard step in the model.
   *
   * <p>As in the Temporal app, this update is best-effort/non-fatal: a
   * projection write failure is logged and the job still completes so it can
   * never abort the order. (Temporal warned and continued; here we complete the
   * job rather than let it incident.)
   */
  @JobWorker(type = JOB_UPDATE_DASHBOARD)
  public void updateDashboard(ActivatedJob job) {
    String orderId = String.valueOf(job.getVariablesAsMap().get(VAR_ORDER_ID));
    String statusHeader = job.getCustomHeaders().get(HEADER_STATUS);

    OrderStatus status;
    try {
      status = OrderStatus.fromString(statusHeader);
    } catch (RuntimeException e) {
      // Matches ValidationFailure on an unsupported dashboard status.
      LOG.error("Unsupported dashboard status header: {}", statusHeader);
      throw e;
    }

    LOG.info("update-dashboard orderId={} status={}", orderId, status);
    try {
      dashboardRepo.upsert(new DashboardStatus(orderId, status.name(), Instant.now()));
    } catch (RuntimeException e) {
      LOG.warn("dashboard update failed; continuing orderId={} status={}", orderId, status, e);
    }
  }

  // -------------------------------------------------------------- fulfillment

  /**
   * Publishes the fulfillment message. Equivalent to {@code
   * PublishToFulfillment}; consumers deduplicate on {@code eventId}. Mocked as a
   * log line here, exactly like the Temporal activity.
   */
  @JobWorker(type = JOB_PUBLISH_FULFILLMENT)
  public void publishToFulfillment(ActivatedJob job) {
    Map<String, Object> vars = job.getVariablesAsMap();
    OrderInput order = objectMapper.convertValue(vars.get(VAR_ORDER), OrderInput.class);
    PaymentInput payment = objectMapper.convertValue(vars.get(VAR_PAYMENT), PaymentInput.class);

    FulfillmentMessage msg =
        new FulfillmentMessage(
            "fulfillment:" + order.orderId() + ":v1",
            1,
            order.customerId(),
            order.orderId(),
            payment,
            order.items());

    LOG.info(
        "publish-fulfillment: published to order-fulfillment topic eventId={} orderId={} customerId={} itemCount={} paymentRRN={}",
        msg.eventId(),
        msg.orderId(),
        msg.customerId(),
        msg.items() == null ? 0 : msg.items().size(),
        payment == null ? null : payment.rrn());
  }
}
