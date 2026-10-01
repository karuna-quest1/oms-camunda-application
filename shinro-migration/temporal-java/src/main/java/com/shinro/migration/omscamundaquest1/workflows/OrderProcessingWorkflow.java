package com.shinro.migration.omscamundaquest1.workflows;

import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;
import java.util.Map;

@WorkflowInterface
public interface OrderProcessingWorkflow {
  @WorkflowMethod
  void execute(Map<String, Object> input);
}