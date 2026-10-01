# Worker: validate-order

- Worker id (manifest `source.workerId`): `worker:oms-camunda-quest1:com.oms.workers.CommerceWorkers#validateOrder`
- Activity name: `Validate-order`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/CommerceWorkers.java:50 — `@JobWorker(type = JOB_VALIDATE_ORDER, autoComplete = false, maxJobsActive = 150)`
- autoComplete: false (explicit completion)

## Classification

- Side-effect class: **read-only** (confidence 0.95, provenance llm)
- Summary: Validates order data by applying corrections and checking business rules, throwing a business error or completing with validated order variables
- Rationale: The code slice shows only in-memory operations: deserializing variables via ObjectMapper.convertValue, calling applyCorrection (a pure transformation), running validationError (a rule check), and logging. No external I/O is present—no HTTP calls, database writes, or message sends. The worker either throws a business error (newThrowErrorCommand) or completes with updated variables (newCompleteCommand), both of which are Zeebe control-plane operations, not side effects to external systems. The comment 'Mock commerce validation is idempotent' and the absence of any downstream I/O confirm this is a pure validation step. Since it performs no mutation and no external call, it is 'read-only'. The orderId from the deserialized order is the natural business key for this operation, suitable as a suggestedIdempotencyKey if any future change introduced side effects.

## Migration guidance

- Coverage: **ASSISTED**
- Suggested retry policy: maxAttempts=2147483647
- flag: autoComplete=false + throwError -> typed error branch to user-task

## Variables

Read:

Written:

## Engine API calls (these DISAPPEAR — map to Temporal return patterns)

- `newThrowErrorCommand` (success) at src/main/java/com/oms/workers/CommerceWorkers.java:65 — hint: NonRetryableApplicationError(type="")
- `newCompleteCommand` (success) at src/main/java/com/oms/workers/CommerceWorkers.java:80 — hint: return output, nil

## Source

Read the handler source with read_file: `src/main/java/com/oms/workers/CommerceWorkers.java` (source repo root).
Call graph: ObjectMapper.convertValue, Logger.info, Logger.warn
