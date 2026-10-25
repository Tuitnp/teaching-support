# Grading job lifecycle and recovery

R12 policy approved on 2026-10-25: **30 minutes per whole-job attempt**, with three
automatic retries after the first attempt and the existing Asynq backoff. A job
is pending between attempts, processing while running, completed after all work
succeeds, and failed after retries are exhausted. Zero attached submissions is a
completed no-op. This does not add cancellation or change grading permissions.

## Identity and writes

The payload is `{job_id, assignment_id, teacher_id, force_regrade}`. `job_id` is
also the deterministic Asynq task ID. Each explicit new start receives a new job
UUID; the existing unique assignment/teacher pair still keeps only the current
run. Restarting a completed job preserves the existing forced-regrade behavior;
starting after failure retains the existing normal-fill behavior. Automatic
retries retain the original payload, including its force flag.

Starting a run locks the assignment and reserves the job inside a DB transaction.
The queue call has a five-second context limit; a rejected enqueue rolls back the
reservation, preserving any previous terminal job. This holds a DB lock across a
bounded queue call, accepting brief contention to avoid an outbox/schema change.
Queue acceptance and PostgreSQL commit are still separate operations: if acceptance
is unacknowledged or commit fails, an orphan task may remain in Redis. It cannot
grade unless its exact job UUID also committed. It retries/archives harmlessly.
A worker that races ahead of the reservation's commit can retry its missing-row
lookup. An uncertain commit is resolved by the persisted UUID, never by deleting
grades or replaying writes against a different run.

Each attempt records a fresh microsecond `started_at` token. Progress, cleanup and
AI writes require both the current job UUID and token. AI writes check them under
the assignment/job locks before the existing conditional score/feedback transaction.
DB reads/writes, storage and provider work receive the attempt context. Expired or
superseded attempts cannot commit a late AI result or change a newer attempt's
state. Database cleanup uses a separate bounded context after cancellation.

Normal retries skip committed grades. Approved forced retries may regenerate and
replace AI grades already saved by that run, so provider calls and AI scores can
repeat/change. Existing AI feedback IDs are retained, previous output survives a
failed replacement, and teacher/TA/unknown/unattributed grades remain protected.
This is not a promise of exactly-once provider execution.

## Observing and recovering state

The worker records each normal outcome and handles early failures/panics, including
no-submission cases and finalization errors. Asynq's timeout handler can record an
expired attempt even before provider cleanup returns. Worker death or a DB outage
can still prevent immediate status persistence.

Job-list reads and new-start requests reconcile active DB rows against Redis:
queued/retry/scheduled means pending, archived means failed, and completed means
completed. A missing task makes the reservation failed/restartable. Redis lookup
errors return an error and preserve the row; an outage is not evidence of failure.
Updates compare the observed status/start token to protect newer attempts. This
is reconciliation on observation, not a new background polling service. Completed
Redis tasks are retained for 24 hours to support this check; terminal DB rows do
not need Redis to remain readable.

Legacy payloads lacking `job_id` are rejected without grading or changing another
run. Their old reservations become restartable when read/reconciled. A future
deployment must stop old workers and coordinate API/worker versions; mixed old/new
workers would bypass the new identity guard. Review legacy queued work before
restarting grading. No deployment, queue purge, existing DB mutation or migration
was performed during development.

## Evidence and remaining limits

PostgreSQL-backed HTTP/repository tests inject queue-acknowledgement loss, a real
deferred commit failure, concurrent starts, and queue states/outages. Worker tests
cover retry exhaustion, early failures, partial normal/forced saves, zero work,
duplicate completion, cancellation during provider work and blocked SQL, stale
attempts/job IDs, and late timeout cleanup. The enqueue adapter checks real Asynq
option values. See [quality-baseline.md](quality-baseline.md) for commands/results.

Redis was unavailable locally. Actual broker scheduling, lease recovery and
process restarts still need the disposable Redis integration check tracked under
Q1. No paid provider, real student data or production service was used. Feedback
publication, cancellation, historical duplicates, and retained-file cleanup remain
separate work. Template/library multi-step writes are outside the assignment
transactions implemented in R12b.
