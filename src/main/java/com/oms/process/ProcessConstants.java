package com.oms.process;

/**
 * Shared identifiers that couple the Java code to the BPMN model
 * ({@code models/order-processing.bpmn}). Keeping them in one place means the
 * message names, job types, error codes and variable names used by workers, the
 * REST gateway and the model can never silently drift apart.
 */
public final class ProcessConstants {

  private ProcessConstants() {}

  /** BPMN process id (the {@code <bpmn:process id="...">}). */
  public static final String PROCESS_ID = "order-processing";

  // --- Messages (Temporal signals/updates -> Zeebe messages, key = orderId) ---
  public static final String MSG_SUPPORT_CORRECTION = "SupportCorrection";
  public static final String MSG_CAPTURE_PAYMENT = "CapturePayment";
  public static final String MSG_CANCEL_ORDER = "CancelOrder";

  // --- Job types (Temporal activities -> Zeebe service tasks) ---
  public static final String JOB_VALIDATE_ORDER = "validate-order";
  public static final String JOB_VALIDATE_PAYMENT = "validate-payment";
  public static final String JOB_ENRICH_ORDER = "enrich-order";
  public static final String JOB_UPDATE_DASHBOARD = "update-dashboard";
  public static final String JOB_PUBLISH_FULFILLMENT = "publish-fulfillment";

  // --- BPMN error codes (thrown by workers, caught by boundary events) ---
  public static final String ERR_INVALID_ORDER = "INVALID_ORDER";
  public static final String ERR_INVALID_PAYMENT = "INVALID_PAYMENT";

  // --- Process variable names ---
  public static final String VAR_ORDER_ID = "orderId";
  public static final String VAR_ORDER = "order";
  public static final String VAR_PAYMENT = "payment";
  /** Corrected line items delivered by a SupportCorrection message; applied to
   * {@code order} by validate-order on re-entry, then cleared. */
  public static final String VAR_CORRECTION_ITEMS = "correctionItems";

  // --- Task header carrying the target status for update-dashboard tasks ---
  public static final String HEADER_STATUS = "status";

  /**
   * Maximum order lifetime. Enforced by the interrupting timer start event in
   * the TTL event subprocess ({@code PT720H}); kept here for reference/tests.
   */
  public static final String ORDER_TTL_ISO = "PT720H"; // 30 days
}
