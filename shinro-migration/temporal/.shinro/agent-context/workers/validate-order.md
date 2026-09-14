# Worker: validate-order

- Activity name: `Validate-order`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/CommerceWorkers.java:50 — `@JobWorker(type = JOB_VALIDATE_ORDER, autoComplete = false, maxJobsActive = 150)`
- autoComplete: false (explicit completion)

## Classification

- Side-effect class: **read-only** (confidence 0.95, provenance llm)
- Summary: Validates order structure and applies pending corrections, throwing a business error if validation fails or completing with the validated order
- Rationale: The code slice shows only in-memory operations: deserializing variables via ObjectMapper.convertValue, calling applyCorrection (local transformation), executing validationError (local check), and logging. No external I/O calls appear in the delegated call graph (only ObjectMapper, Logger). The handler returns either newThrowErrorCommand or newCompleteCommand based on validation outcome, but these are Zeebe engine API calls (workflow control), not side effects on external systems. The comment 'Mock commerce validation is idempotent' confirms no real external mutation occurs. This is a pure validation activity with no observable external state change.

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
