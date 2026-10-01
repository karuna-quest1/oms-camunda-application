package com.shinro.migration.omscamundaquest1.workflows;

import io.temporal.activity.ActivityOptions;
import io.temporal.failure.ApplicationFailure;
import io.temporal.workflow.*;
import java.time.Duration;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import com.shinro.migration.omscamundaquest1.activities.ValidateOrderActivity;
import com.shinro.migration.omscamundaquest1.activities.EnrichOrderActivity;
import com.shinro.migration.omscamundaquest1.activities.PublishFulfillmentActivity;
import com.shinro.migration.omscamundaquest1.activities.UpdateDashboardActivity;
import com.shinro.migration.omscamundaquest1.activities.ValidatePaymentActivity;

public class OrderProcessingWorkflowImpl implements OrderProcessingWorkflow {

  private final ValidateOrderActivity validateOrderActivity = 
      Workflow.newActivityStub(ValidateOrderActivity.class, ActivityOptions.newBuilder()
          .setStartToCloseTimeout(Duration.ofSeconds(30))
          .build());
  
  private final EnrichOrderActivity enrichOrderActivity = 
      Workflow.newActivityStub(EnrichOrderActivity.class, ActivityOptions.newBuilder()
          .setStartToCloseTimeout(Duration.ofSeconds(30))
          .build());
  
  private final PublishFulfillmentActivity publishFulfillmentActivity = 
      Workflow.newActivityStub(PublishFulfillmentActivity.class, ActivityOptions.newBuilder()
          .setStartToCloseTimeout(Duration.ofSeconds(30))
          .build());
  
  private final UpdateDashboardActivity updateDashboardActivity = 
      Workflow.newActivityStub(UpdateDashboardActivity.class, ActivityOptions.newBuilder()
          .setStartToCloseTimeout(Duration.ofSeconds(60))
          .build());
  
  private final ValidatePaymentActivity validatePaymentActivity = 
      Workflow.newActivityStub(ValidatePaymentActivity.class, ActivityOptions.newBuilder()
          .setStartToCloseTimeout(Duration.ofSeconds(30))
          .build());

  @Override
  public void execute(Map<String, Object> input) {
    // Start by logging ORDER_RECEIVED to dashboard
    updateDashboardActivity.updateDashboard(input, "ORDER_RECEIVED");
    
    // Validate order and handle INVALID_ORDER boundary error
    Map<String, Object> validatedOrder = null;
    try {
      validatedOrder = validateOrderActivity.validateOrder(input);
    } catch (ApplicationFailure e) {
      if ("INVALID_ORDER".equals(e.getType())) {
        // INVALID_ORDER error -> Dashboard: AWAITING_CORRECTION
        updateDashboardActivity.updateDashboard(input, "AWAITING_CORRECTION");
        // Wait for correction message
        Workflow.await(() -> false);
        // Process would continue after signal is received (simplified)
        return;
      } else {
        throw e;
      }
    }
    
    // Set PAYMENT_PENDING status
    updateDashboardActivity.updateDashboard(input, "PAYMENT_PENDING");
    
    // Wait for payment capture message (this is a simplified implementation)
    boolean paymentReceived = false;
    while (!paymentReceived) {
      try {
        // Check if payment signal was received - in real workflow this would wait properly
        Workflow.await(() -> false);
//        } catch (Exception e) {
//          // Just continue to retry 
//        }
        
        Map<String, Object> paymentData = input;
        if (paymentData.containsKey("payment")) {
          paymentData = validatePaymentActivity.validatePayment(paymentData);
          paymentReceived = true;
        } else {
          updateDashboardActivity.updateDashboard(input, "PAYMENT_PENDING");
        }
      } catch (ApplicationFailure e) {
        if ("INVALID_PAYMENT".equals(e.getType())) {
          // INVALID_PAYMENT error -> wait for CapturePayment again
          updateDashboardActivity.updateDashboard(input, "PAYMENT_PENDING");
          // Continue loop to wait for signal in a more realistic way
          Workflow.await(() -> false);
        } else {
          throw e;
        }
      }
    }
    
    // Set PAYMENT_CAPTURED status
    updateDashboardActivity.updateDashboard(input, "PAYMENT_CAPTURED");
    
    // Enrich order with PIM
    Map<String, Object> enrichedOrder = enrichOrderActivity.enrichOrder(validatedOrder);
    
    // Set FULFILLED status
    updateDashboardActivity.updateDashboard(input, "FULFILLED");
    
    // Publish to fulfillment
    publishFulfillmentActivity.publishToFulfillment(enrichedOrder);
  }
}