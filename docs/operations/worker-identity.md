# Worker runtime identity

Every production Synaudio worker process must receive an explicit `WORKER_ID`. The value is used consistently for generation-job claim ownership plus worker audit/log attribution, so concurrently running replicas must never share it.

## Production

Inject a distinct, stable-for-process-lifetime value for every worker replica, for example an orchestrator pod/container identity. `WORKER_ID` is limited to 1-64 characters and accepts ASCII letters, digits, `-`, `_`, `.`, and `:`. Missing or invalid production identity fails startup before job polling begins.

During a rolling update, old and new worker replicas must receive different IDs. Do not configure a shared static value such as `worker-1` across replicas.

## Development

When `APP_ENV=development` and `WORKER_ID` is omitted, the worker generates a process-local `dev-<random>` identity. This preserves zero-setup local development without creating a shared constant ownership identity.

## Ownership boundary

This document covers runtime identity only. Lease duration, stale reclaim and signal/shutdown recovery remain governed by parent task #60 and must not infer ownership safety merely from the worker ID.