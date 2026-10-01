package com.shinro.migration.omscamundaquest1;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.shinro.migration.omscamundaquest1.activities.*;
import com.shinro.migration.omscamundaquest1.activities.UpdateDashboardActivityImpl.DashboardRepository;
import com.shinro.migration.omscamundaquest1.domain.DashboardStatus;
import com.shinro.migration.omscamundaquest1.workflows.OrderProcessingWorkflow;
import com.shinro.migration.omscamundaquest1.workflows.OrderProcessingWorkflowImpl;
import io.temporal.client.WorkflowClient;
import io.temporal.serviceclient.WorkflowServiceStubs;
import io.temporal.worker.Worker;
import io.temporal.worker.WorkerFactory;

public class TemporalWorker {

  public static void main(String[] args) {
    // Create the workflow service stubs
    WorkflowServiceStubs service = WorkflowServiceStubs.newInstance();
    
    // Set up client to connect to the server
    WorkflowClient client = WorkflowClient.newInstance(service);
    
    // Create a worker factory
    WorkerFactory factory = WorkerFactory.newInstance(client);
    
    // Create Worker for our workflow and activities
    Worker worker = factory.newWorker("order-processing-task-queue");
    
    ObjectMapper objectMapper = new ObjectMapper();
    
    // Register Activities
    worker.registerActivitiesImplementations(
        new ValidateOrderActivityImpl(objectMapper),
        new EnrichOrderActivityImpl(objectMapper),
        new UpdateDashboardActivityImpl(objectMapper, new DashboardRepository() {
          @Override
          public void upsert(DashboardStatus status) {
            // Stub implementation for migration
          }
        }),
        new ValidatePaymentActivityImpl(objectMapper),
        new PublishFulfillmentActivityImpl(objectMapper)
    );
    
    // Register Workflow
    worker.registerWorkflowImplementationTypes(OrderProcessingWorkflowImpl.class);
    
    // Start the worker
    factory.start();
    
    System.out.println("Temporal worker started on task queue: order-processing-task-queue");
  }
}