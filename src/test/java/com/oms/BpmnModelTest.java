package com.oms;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.oms.process.ProcessConstants;
import io.camunda.zeebe.model.bpmn.Bpmn;
import io.camunda.zeebe.model.bpmn.BpmnModelInstance;
import io.camunda.zeebe.model.bpmn.instance.Process;
import io.camunda.zeebe.model.bpmn.instance.ServiceTask;
import io.camunda.zeebe.model.bpmn.instance.zeebe.ZeebeTaskDefinition;
import java.io.InputStream;
import java.util.HashSet;
import java.util.Set;
import org.junit.jupiter.api.Test;

/**
 * Validates the deployed BPMN with the Zeebe model API: the file parses, the
 * process id matches {@link ProcessConstants}, and every service task's job
 * type is one the workers actually register. This catches model/worker drift
 * without needing a running engine.
 */
class BpmnModelTest {

  private BpmnModelInstance model() {
    InputStream in =
        getClass().getClassLoader().getResourceAsStream("models/order-processing.bpmn");
    assertNotNull(in, "BPMN resource must be on the classpath");
    return Bpmn.readModelFromStream(in);
  }

  @Test
  void parsesAndValidates() {
    BpmnModelInstance instance = model();
    Bpmn.validateModel(instance); // throws on structural violations

    Process process = instance.getModelElementById(ProcessConstants.PROCESS_ID);
    assertNotNull(process, "process id must be " + ProcessConstants.PROCESS_ID);
    assertTrue(process.isExecutable());
  }

  @Test
  void everyServiceTaskUsesAKnownJobType() {
    Set<String> known =
        Set.of(
            ProcessConstants.JOB_VALIDATE_ORDER,
            ProcessConstants.JOB_VALIDATE_PAYMENT,
            ProcessConstants.JOB_ENRICH_ORDER,
            ProcessConstants.JOB_UPDATE_DASHBOARD,
            ProcessConstants.JOB_PUBLISH_FULFILLMENT);

    Set<String> seen = new HashSet<>();
    for (ServiceTask task : model().getModelElementsByType(ServiceTask.class)) {
      ZeebeTaskDefinition def =
          task.getSingleExtensionElement(ZeebeTaskDefinition.class);
      assertNotNull(def, "service task " + task.getId() + " needs a zeebe:taskDefinition");
      assertTrue(
          known.contains(def.getType()),
          "unknown job type '" + def.getType() + "' on task " + task.getId());
      seen.add(def.getType());
    }
    assertEquals(known, seen, "every registered worker type should appear in the model");
  }
}
