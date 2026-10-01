# Worker: validate-payment

- Worker id (manifest `source.workerId`): `worker:oms-camunda-quest1:com.oms.workers.OmsWorkers#validatePayment`
- Activity name: `Validate-payment`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/OmsWorkers.java:63 — `@JobWorker(type = JOB_VALIDATE_PAYMENT, autoComplete = false)`
- autoComplete: false (explicit completion)

## Classification

- Side-effect class: **read-only** (confidence 0.95, provenance llm)
- Summary: Validates payment data against order total and throws a BPMN error if validation fails, otherwise completes with the payment variable
- Rationale: The code slice shows the worker reads VAR_PAYMENT and VAR_ORDER from job variables, deserializes them via ObjectMapper.convertValue, performs in-memory validation (validationError check comparing payment against expectedAmount), logs the result, and either throws a BPMN error or completes the job. No external I/O is visible: ObjectMapper.convertValue is a pure deserialization, validationError is a pure function (not shown but context implies local validation logic), and the only calls are Logger.info/warn (observability, no side effects) and Zeebe engine commands (newThrowErrorCommand/newCompleteCommand). The worker reads data, computes a validation result in memory, and signals the outcome to the engine — no mutations to external systems.

## Migration guidance

- Coverage: **ASSISTED**
- Suggested retry policy: maxAttempts=5
- flag: autoComplete=false + throwError -> typed error branch to user-task

## Variables

Read:

Written:

## Engine API calls (these DISAPPEAR — map to Temporal return patterns)

- `newThrowErrorCommand` (success) at src/main/java/com/oms/workers/OmsWorkers.java:78 — hint: NonRetryableApplicationError(type="")
- `newCompleteCommand` (success) at src/main/java/com/oms/workers/OmsWorkers.java:88 — hint: return output, nil

## Source

Read the handler source with read_file: `src/main/java/com/oms/workers/OmsWorkers.java` (source repo root).
Call graph: ObjectMapper.convertValue, Logger.info, Logger.warn
