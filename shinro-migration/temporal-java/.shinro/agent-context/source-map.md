# Source map

Every entity → where it lives in the source repository.

- worker `validate-order` → src/main/java/com/oms/workers/CommerceWorkers.java:50
- worker `enrich-order` → src/main/java/com/oms/workers/OmsWorkers.java:124
- worker `publish-fulfillment` → src/main/java/com/oms/workers/OmsWorkers.java:198
- worker `update-dashboard` → src/main/java/com/oms/workers/OmsWorkers.java:169
- worker `validate-payment` → src/main/java/com/oms/workers/OmsWorkers.java:63
- process `order-processing` → `src/main/resources/models/order-processing.bpmn`

## Entry points that start processes

- rest → starts `` (POST /orders) at src/main/java/com/oms/api/OrderController.java:106
- message → starts `` at src/main/java/com/oms/api/OrderController.java:222
