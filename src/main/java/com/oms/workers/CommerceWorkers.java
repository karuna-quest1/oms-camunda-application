package com.oms.workers;

import static com.oms.process.ProcessConstants.ERR_INVALID_ORDER;
import static com.oms.process.ProcessConstants.JOB_VALIDATE_ORDER;
import static com.oms.process.ProcessConstants.VAR_CORRECTION_ITEMS;
import static com.oms.process.ProcessConstants.VAR_ORDER;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.oms.model.OrderInput;
import com.oms.model.OrderItem;
import io.camunda.zeebe.client.api.response.ActivatedJob;
import io.camunda.zeebe.client.api.worker.JobClient;
import io.camunda.zeebe.spring.client.annotation.JobWorker;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

/**
 * Commerce-platform worker — the Camunda equivalent of the Temporal
 * {@code ValidateOrderAPI} activity, which ran on a dedicated {@code
 * COMMERCE_QUEUE} capped at 150 RPS.
 *
 * <p>Zeebe has no per-task-queue rate limiter like Temporal's {@code
 * TaskQueueActivitiesPerSecond}. The closest lever is bounding how many jobs a
 * worker activates concurrently ({@code maxJobsActive}) and running a fixed
 * number of replicas; a true RPS cap would need a rate-limiting gateway in
 * {@code validateOrder} itself. This is called out in the README's mapping
 * table as the one behavioural gap versus Temporal.
 *
 * <p>Business validation failures are surfaced as the BPMN error {@code
 * INVALID_ORDER}, caught by the boundary event that drives the correction loop
 * — mirroring the Temporal NonRetryable {@code ValidationFailure} that parked
 * the workflow in {@code AWAITING_CORRECTION}.
 */
@Component
public class CommerceWorkers {

  private static final Logger LOG = LoggerFactory.getLogger(CommerceWorkers.class);

  private final ObjectMapper objectMapper;

  public CommerceWorkers(ObjectMapper objectMapper) {
    this.objectMapper = objectMapper;
  }

  @JobWorker(type = JOB_VALIDATE_ORDER, autoComplete = false, maxJobsActive = 150)
  public void validateOrder(JobClient client, ActivatedJob job) {
    Map<String, Object> vars = job.getVariablesAsMap();
    OrderInput order = objectMapper.convertValue(vars.get(VAR_ORDER), OrderInput.class);

    // Apply any pending SupportCorrection (delivered as correctionItems) before
    // re-validating — the parity of WorkflowState.ApplyCorrection.
    order = applyCorrection(order, vars.get(VAR_CORRECTION_ITEMS));

    LOG.info("validate-order started orderId={}", order == null ? null : order.orderId());

    String rejection = validationError(order);
    if (rejection != null) {
      LOG.warn("Order validation failed: {}", rejection);
      // Non-retryable business failure -> correction loop (not a job retry).
      client
          .newThrowErrorCommand(job)
          .errorCode(ERR_INVALID_ORDER)
          .errorMessage(rejection)
          .send()
          .join();
      return;
    }

    // Mock commerce validation is idempotent; echo the (validated) order back
    // and clear the consumed correction.
    LOG.info("validate-order completed orderId={}", order.orderId());
    Map<String, Object> out = new HashMap<>();
    out.put(VAR_ORDER, order);
    out.put(VAR_CORRECTION_ITEMS, null);
    client.newCompleteCommand(job).variables(out).send().join();
  }

  /** Replaces the order's items with corrected ones when a correction is pending. */
  private OrderInput applyCorrection(OrderInput order, Object correctionItemsRaw) {
    if (order == null || correctionItemsRaw == null) {
      return order;
    }
    List<OrderItem> items =
        objectMapper.convertValue(correctionItemsRaw, new TypeReference<List<OrderItem>>() {});
    if (items == null || items.isEmpty()) {
      return order;
    }
    return new OrderInput(
        order.orderId(), order.customerId(), items, order.totalAmount(), order.currency());
  }

  /** Returns {@code null} when the order is valid, else the rejection reason. */
  private static String validationError(OrderInput order) {
    if (order == null || order.orderId() == null || order.orderId().isBlank()) {
      return "order_id is required";
    }
    if (order.items() == null || order.items().isEmpty()) {
      return "order must contain at least one item";
    }
    if (order.totalAmount() <= 0) {
      return "invalid total_amount: " + order.totalAmount();
    }
    if (order.customerId() == null || order.customerId().isBlank()) {
      return "customer_id is required";
    }
    return null;
  }
}
