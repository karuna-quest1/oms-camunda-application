# Worker: validate-payment

- Activity name: `Validate-payment`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/OmsWorkers.java:63 — `@JobWorker(type = JOB_VALIDATE_PAYMENT, autoComplete = false)`
- autoComplete: false (explicit completion)

## Classification

- Side-effect class: **read-only** (confidence 0.95, provenance llm)
- Summary: Validates payment data against order total and throws a BPMN error if validation fails, otherwise completes with the payment variable.
- Rationale: The code slice shows only in-memory operations: ObjectMapper.convertValue reads variables from the job, validationError performs a pure comparison (payment amount vs. expectedAmount), and Logger.info/warn are observability calls with no external mutation. The worker either throws a BPMN error (newThrowErrorCommand) or completes with the same payment variable it read (newCompleteCommand). No external I/O, no database write, no HTTP call—only reads and conditional logic. This is a pure validation step with no side effects.

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
