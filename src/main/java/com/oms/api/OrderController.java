package com.oms.api;

import static com.oms.process.ProcessConstants.MSG_CANCEL_ORDER;
import static com.oms.process.ProcessConstants.MSG_CAPTURE_PAYMENT;
import static com.oms.process.ProcessConstants.MSG_SUPPORT_CORRECTION;
import static com.oms.process.ProcessConstants.PROCESS_ID;
import static com.oms.process.ProcessConstants.VAR_CORRECTION_ITEMS;
import static com.oms.process.ProcessConstants.VAR_ORDER;
import static com.oms.process.ProcessConstants.VAR_ORDER_ID;
import static com.oms.process.ProcessConstants.VAR_PAYMENT;

import com.oms.dashboard.DashboardRepository;
import com.oms.dashboard.DashboardStatus;
import com.oms.model.Dtos.CancelWebhookRequest;
import com.oms.model.Dtos.CommerceWebhookRequest;
import com.oms.model.Dtos.CorrectionWebhookRequest;
import com.oms.model.Dtos.ErrorResponse;
import com.oms.model.Dtos.OrderStatusResponse;
import com.oms.model.Dtos.PaymentWebhookRequest;
import com.oms.model.Dtos.StartWorkflowResponse;
import com.oms.model.OrderInput;
import com.oms.model.OrderStatus;
import com.oms.model.PaymentInput;
import io.camunda.zeebe.client.ZeebeClient;
import io.camunda.zeebe.client.api.response.ProcessInstanceEvent;
import java.time.Duration;
import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * HTTP gateway — the Camunda parity of the Temporal {@code api/ApiApp.go}.
 *
 * <pre>
 *   POST /orders            -> start an order-processing process instance
 *   POST /orders/correction -> publish SupportCorrection message
 *   POST /orders/payment    -> publish CapturePayment message
 *   POST /orders/cancel     -> publish CancelOrder message
 *   GET  /orders/status     -> read the dashboard projection
 * </pre>
 *
 * <p>Temporal used synchronous validated Workflow Updates to reject illegal
 * correction/payment attempts with HTTP 409. Zeebe messages are fire-and-forget
 * with no validating handler, so that 409 behaviour is reproduced here at the
 * API layer by validating against the dashboard projection before publishing —
 * the one design accommodation called out in the migration notes.
 */
@RestController
@RequestMapping("/orders")
public class OrderController {

  private static final Logger LOG = LoggerFactory.getLogger(OrderController.class);

  /** Messages buffer in Zeebe until the instance reaches the catch, up to TTL. */
  private static final Duration MESSAGE_TTL = Duration.ofDays(30);

  private final ZeebeClient zeebe;
  private final DashboardRepository dashboardRepo;

  public OrderController(ZeebeClient zeebe, DashboardRepository dashboardRepo) {
    this.zeebe = zeebe;
    this.dashboardRepo = dashboardRepo;
  }

  // ------------------------------------------------------------ POST /orders

  @PostMapping
  public ResponseEntity<?> createOrder(@RequestBody CommerceWebhookRequest req) {
    if (isBlank(req.orderId())) {
      return badRequest("order_id is required");
    }

    // Atomic idempotency claim (parity of ALLOW_DUPLICATE_FAILED_ONLY): the
    // projection's primary key is a per-order lock. Exactly one concurrent
    // create wins the insert and starts the instance; the rest get 409. The
    // claim also seeds ORDER_RECEIVED so an immediate GET /orders/status doesn't
    // race the first update-dashboard job.
    boolean claimed =
        dashboardRepo.insertIfAbsent(
            new DashboardStatus(req.orderId(), OrderStatus.ORDER_RECEIVED.name(), Instant.now()));
    if (!claimed) {
      return conflict("order workflow already exists");
    }

    OrderInput order =
        new OrderInput(
            req.orderId(), req.customerId(), req.items(), req.totalAmount(), req.currency());

    Map<String, Object> variables = new HashMap<>();
    variables.put(VAR_ORDER_ID, req.orderId());
    variables.put(VAR_ORDER, order);

    ProcessInstanceEvent instance;
    try {
      instance =
          zeebe
              .newCreateInstanceCommand()
              .bpmnProcessId(PROCESS_ID)
              .latestVersion()
              .variables(variables)
              .send()
              .join();
    } catch (RuntimeException e) {
      // Release the claim so the order can be retried rather than being wedged
      // behind a claim row that has no running instance.
      LOG.error("failed to start process for order {}; releasing claim", req.orderId(), e);
      safeReleaseClaim(req.orderId());
      return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
          .body(new ErrorResponse("failed to start order workflow"));
    }

    return ResponseEntity.accepted()
        .body(new StartWorkflowResponse(instance.getProcessInstanceKey(), req.orderId()));
  }

  // ------------------------------------------------- POST /orders/correction

  @PostMapping("/correction")
  public ResponseEntity<?> correction(@RequestBody CorrectionWebhookRequest req) {
    if (isBlank(req.orderId())) {
      return badRequest("order_id is required");
    }

    OrderStatus status = currentStatus(req.orderId());
    if (status == null) {
      return notFound("order not found");
    }
    if (isTerminal(status)) {
      return conflict("workflow is already in a terminal state");
    }
    if (status != OrderStatus.AWAITING_CORRECTION) {
      return conflict("correction not allowed in current state");
    }

    publish(MSG_SUPPORT_CORRECTION, req.orderId(), Map.of(VAR_CORRECTION_ITEMS, req.items()));
    return ResponseEntity.noContent().build();
  }

  // ---------------------------------------------------- POST /orders/payment

  @PostMapping("/payment")
  public ResponseEntity<?> payment(@RequestBody PaymentWebhookRequest req) {
    if (isBlank(req.orderId())) {
      return badRequest("order_id is required");
    }
    if (isBlank(req.rrn())) {
      return badRequest("rrn is required");
    }

    OrderStatus status = currentStatus(req.orderId());
    if (status == null) {
      return notFound("order not found");
    }
    if (isTerminal(status)) {
      return conflict("workflow is already in a terminal state");
    }
    if (status == OrderStatus.PAYMENT_CAPTURED) {
      return conflict("payment already captured");
    }

    PaymentInput payment =
        new PaymentInput(
            req.rrn(), req.amount(), req.currency(), req.paymentMethod(), req.paidAt());
    publish(MSG_CAPTURE_PAYMENT, req.orderId(), Map.of(VAR_PAYMENT, payment));
    return ResponseEntity.noContent().build();
  }

  // ----------------------------------------------------- POST /orders/cancel

  @PostMapping("/cancel")
  public ResponseEntity<?> cancel(@RequestBody CancelWebhookRequest req) {
    if (isBlank(req.orderId())) {
      return badRequest("order_id is required");
    }
    publish(MSG_CANCEL_ORDER, req.orderId(), Map.of());
    return ResponseEntity.noContent().build();
  }

  // ------------------------------------------------------ GET /orders/status

  @GetMapping("/status")
  public ResponseEntity<?> status(@RequestParam("order_id") String orderId) {
    if (isBlank(orderId)) {
      return badRequest("order_id query parameter is required");
    }
    return dashboardRepo
        .find(orderId)
        .<ResponseEntity<?>>map(
            row ->
                ResponseEntity.ok(
                    new OrderStatusResponse(row.orderId(), OrderStatus.fromString(row.status()))))
        .orElseGet(() -> notFound("order not found"));
  }

  // --------------------------------------------------------------- internals

  private OrderStatus currentStatus(String orderId) {
    return dashboardRepo
        .find(orderId)
        .map(row -> OrderStatus.fromString(row.status()))
        .orElse(null);
  }

  private static boolean isTerminal(OrderStatus status) {
    return switch (status) {
      case CANCELLED, EXPIRED, FULFILLED, FAILED_PAYMENT -> true;
      default -> false;
    };
  }

  private void publish(String messageName, String orderId, Map<String, Object> variables) {
    zeebe
        .newPublishMessageCommand()
        .messageName(messageName)
        .correlationKey(orderId)
        .timeToLive(MESSAGE_TTL)
        .variables(variables)
        .send()
        .join();
  }

  private void safeReleaseClaim(String orderId) {
    try {
      dashboardRepo.delete(orderId);
    } catch (RuntimeException e) {
      LOG.warn("failed to release idempotency claim for order {}", orderId, e);
    }
  }

  private static boolean isBlank(String s) {
    return s == null || s.isBlank();
  }

  private static ResponseEntity<ErrorResponse> badRequest(String msg) {
    return ResponseEntity.badRequest().body(new ErrorResponse(msg));
  }

  private static ResponseEntity<ErrorResponse> conflict(String msg) {
    return ResponseEntity.status(HttpStatus.CONFLICT).body(new ErrorResponse(msg));
  }

  private static ResponseEntity<ErrorResponse> notFound(String msg) {
    return ResponseEntity.status(HttpStatus.NOT_FOUND).body(new ErrorResponse(msg));
  }
}
