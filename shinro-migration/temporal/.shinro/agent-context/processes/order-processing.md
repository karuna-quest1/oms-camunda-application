# Process: order-processing — Order Processing

- File: `src/main/resources/models/order-processing.bpmn`
- Executable: yes
- Description: BPMN process "Order Processing" (id: order-processing) — executable.
Contains 23 flow node(s) and 18 sequence flow(s): 2 boundaryEvent, 3 endEvent, 2 intermediateCatchEvent, 11 serviceTask, 3 startEvent, 2 subProcess.
Service job types: enrich-order, publish-fulfillment, update-dashboard, validate-order, validate-payment.
Correlates message(s): Message_CancelOrder, Message_CapturePayment, Message_SupportCorrection.

Flow:
  Start: none event "Order received"
  Service task "Dashboard: ORDER_RECEIVED" (job type "update-dashboard")
  Service task "Validate order (Commerce API)" (job type "validate-order")
  Service task "Dashboard: PAYMENT_PENDING" (job type "update-dashboard")
  Catch message event "Await CapturePayment" (message "Message_CapturePayment")
  Service task "Validate payment RRN" (job type "validate-payment")
  Service task "Dashboard: PAYMENT_CAPTURED" (job type "update-dashboard")
  Service task "Enrich with PIM" (job type "enrich-order")
  Service task "Publish to fulfillment" (job type "publish-fulfillment")
  Service task "Dashboard: FULFILLED" (job type "update-dashboard")
  End: none event "Order fulfilled"
  Start: message event "CancelOrder" (message "Message_CancelOrder")
  Service task "Dashboard: CANCELLED" (job type "update-dashboard")
  End: terminate event "Cancelled"
  Start: timer event "30-day TTL" (timeDuration=PT720H)
  Service task "Dashboard: EXPIRED" (job type "update-dashboard")
  End: terminate event "Expired"

Boundary events (interrupts / error handling):
- error event "INVALID_ORDER" (error "INVALID_ORDER") on "Validate order (Commerce API)" routes to Dashboard: AWAITING_CORRECTION
- error event "INVALID_PAYMENT" (error "INVALID_PAYMENT") on "Validate payment RRN" routes to Await CapturePayment
- Messages: Message_CancelOrder, Message_CapturePayment, Message_SupportCorrection

## Elements

- `StartEvent_1` **startEvent** "Order received" classificationKey=`startEvent.none`
- `Task_UpdateReceived` **serviceTask** "Dashboard: ORDER_RECEIVED" jobType=`update-dashboard` classificationKey=`serviceTask`
- `Task_ValidateOrder` **serviceTask** "Validate order (Commerce API)" jobType=`validate-order` classificationKey=`serviceTask`
- `Task_UpdatePaymentPending` **serviceTask** "Dashboard: PAYMENT_PENDING" jobType=`update-dashboard` classificationKey=`serviceTask`
- `Catch_Payment` **intermediateCatchEvent** "Await CapturePayment" event=message:Message_CapturePayment
- `Task_ValidatePayment` **serviceTask** "Validate payment RRN" jobType=`validate-payment` classificationKey=`serviceTask`
- `Task_UpdateCaptured` **serviceTask** "Dashboard: PAYMENT_CAPTURED" jobType=`update-dashboard` classificationKey=`serviceTask`
- `Task_Enrich` **serviceTask** "Enrich with PIM" jobType=`enrich-order` classificationKey=`serviceTask`
- `Task_Publish` **serviceTask** "Publish to fulfillment" jobType=`publish-fulfillment` classificationKey=`serviceTask`
- `Task_UpdateFulfilled` **serviceTask** "Dashboard: FULFILLED" jobType=`update-dashboard` classificationKey=`serviceTask`
- `EndEvent_Done` **endEvent** "Order fulfilled" classificationKey=`endEvent.none`
- `Bnd_InvalidOrder` **boundaryEvent** "INVALID_ORDER" event=error:INVALID_ORDER attachedTo=`Task_ValidateOrder`
- `Task_AwaitCorrectionStatus` **serviceTask** "Dashboard: AWAITING_CORRECTION" jobType=`update-dashboard` classificationKey=`serviceTask`
- `Catch_Correction` **intermediateCatchEvent** "Await SupportCorrection" event=message:Message_SupportCorrection
- `Bnd_InvalidPayment` **boundaryEvent** "INVALID_PAYMENT" event=error:INVALID_PAYMENT attachedTo=`Task_ValidatePayment`
- `SubProcess_Cancel` **subProcess** "Cancellation"
- `Start_Cancel` **startEvent** "CancelOrder" event=message:Message_CancelOrder
- `Task_UpdateCancelled` **serviceTask** "Dashboard: CANCELLED" jobType=`update-dashboard` classificationKey=`serviceTask`
- `End_Cancelled` **endEvent** "Cancelled" event=terminate classificationKey=`endEvent.terminate`
- `SubProcess_TTL` **subProcess** "Order TTL (30 days)"
- `Start_TTL` **startEvent** "30-day TTL" event=timer timer=`timeDuration=PT720H`
- `Task_UpdateExpired` **serviceTask** "Dashboard: EXPIRED" jobType=`update-dashboard` classificationKey=`serviceTask`
- `End_Expired` **endEvent** "Expired" event=terminate classificationKey=`endEvent.terminate`

## Sequence flows

- `StartEvent_1` → `Task_UpdateReceived`
- `Task_UpdateReceived` → `Task_ValidateOrder`
- `Task_ValidateOrder` → `Task_UpdatePaymentPending`
- `Task_UpdatePaymentPending` → `Catch_Payment`
- `Catch_Payment` → `Task_ValidatePayment`
- `Task_ValidatePayment` → `Task_UpdateCaptured`
- `Task_UpdateCaptured` → `Task_Enrich`
- `Task_Enrich` → `Task_Publish`
- `Task_Publish` → `Task_UpdateFulfilled`
- `Task_UpdateFulfilled` → `EndEvent_Done`
- `Bnd_InvalidOrder` → `Task_AwaitCorrectionStatus`
- `Task_AwaitCorrectionStatus` → `Catch_Correction`
- `Catch_Correction` → `Task_ValidateOrder`
- `Bnd_InvalidPayment` → `Catch_Payment`
- `Start_Cancel` → `Task_UpdateCancelled`
- `Task_UpdateCancelled` → `End_Cancelled`
- `Start_TTL` → `Task_UpdateExpired`
- `Task_UpdateExpired` → `End_Expired`

## Job types → workers

- `enrich-order` → worker `worker:oms-camunda-application:com.oms.workers.OmsWorkers#enrichOrder` at src/main/java/com/oms/workers/OmsWorkers.java:124 — see `workers/enrich-order.md`
- `publish-fulfillment` → worker `worker:oms-camunda-application:com.oms.workers.OmsWorkers#publishToFulfillment` at src/main/java/com/oms/workers/OmsWorkers.java:198 — see `workers/publish-fulfillment.md`
- `update-dashboard` → worker `worker:oms-camunda-application:com.oms.workers.OmsWorkers#updateDashboard` at src/main/java/com/oms/workers/OmsWorkers.java:169 — see `workers/update-dashboard.md`
- `validate-order` → worker `worker:oms-camunda-application:com.oms.workers.CommerceWorkers#validateOrder` at src/main/java/com/oms/workers/CommerceWorkers.java:50 — see `workers/validate-order.md`
- `validate-payment` → worker `worker:oms-camunda-application:com.oms.workers.OmsWorkers#validatePayment` at src/main/java/com/oms/workers/OmsWorkers.java:63 — see `workers/validate-payment.md`
