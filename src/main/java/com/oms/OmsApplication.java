package com.oms;

import io.camunda.client.annotation.Deployment;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * OMS on Camunda 8.
 *
 * <p>Single deployable that hosts both the REST gateway ({@code com.oms.api})
 * and the Zeebe job workers ({@code com.oms.workers}). This mirrors the
 * Temporal application, which ran an API process and a worker process; here a
 * Spring Boot process embeds both the {@code CamundaClient} used by the gateway
 * and the annotation-driven {@code @JobWorker} beans.
 *
 * <p>{@code @Deployment} auto-deploys the BPMN model to Zeebe on startup.
 */
@SpringBootApplication
@Deployment(resources = "classpath:models/order-processing.bpmn")
public class OmsApplication {
  public static void main(String[] args) {
    SpringApplication.run(OmsApplication.class, args);
  }
}
