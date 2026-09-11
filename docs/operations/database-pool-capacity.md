# PostgreSQL connection pool capacity

Synaudio API and worker processes share one explicit PostgreSQL pool contract. Production no longer relies on implicit `pgxpool` defaults that scale with CPU or replica count.

## Per-process configuration

Both binaries call `config.LoadDatabasePoolSettings` and `db.NewPool` with the same semantics.

| Variable | Required in production | Purpose |
| --- | --- | --- |
| `DATABASE_POOL_MAX_CONNS` | yes | Maximum open connections for this process |
| `DATABASE_POOL_PROCESS_ROLE` | yes | `api` or `worker`; must match the running binary |
| `DATABASE_POOL_MIN_CONNS` | no | Minimum warm connections (default `0`) |
| `DATABASE_POOL_MAX_CONN_LIFETIME` | no | Connection recycle interval (default `1h`) |
| `DATABASE_POOL_MAX_CONN_IDLE_TIME` | no | Idle connection close interval (default `30m`) |
| `DATABASE_POOL_HEALTH_CHECK_PERIOD` | no | Idle health-check interval (default `1m`) |
| `DATABASE_POOL_ACQUIRE_TIMEOUT` | no | Enforced upper bound for waiting to obtain a pooled connection (default `30s`). A shorter caller deadline/cancellation still wins. The timeout stops after acquisition and does not impose an artificial query-duration limit. |

Development leaves `DATABASE_POOL_MAX_CONNS` unset to use the explicit pgx default of `max(4, runtime.NumCPU())` without requiring budget inputs.

## Total connection budget

Production startup validates the declared PostgreSQL capacity budget before serving traffic.

| Variable | Required in production | Purpose |
| --- | --- | --- |
| `DATABASE_CONNECTION_BUDGET` | yes | Total PostgreSQL connection allowance for the service (for example Neon `max_connections`) |
| `DATABASE_POOL_API_REPLICAS` | yes | Planned steady-state API replica count |
| `DATABASE_POOL_WORKER_REPLICAS` | yes | Planned steady-state worker replica count |
| `DATABASE_POOL_API_MAX_CONNS` | yes | Per-API-replica `DATABASE_POOL_MAX_CONNS` used in budget math |
| `DATABASE_POOL_WORKER_MAX_CONNS` | yes | Per-worker-replica `DATABASE_POOL_MAX_CONNS` used in budget math |
| `DATABASE_POOL_OPERATIONS_RESERVE` | no | Connections reserved for migrations, admin, monitoring (default `5`) |
| `DATABASE_POOL_ROLLOUT_SURGE_FACTOR` | no | Multiplier for simultaneous old/new replicas during rolling deploys (default `2`) |

Validation formula:

```text
rollout_peak =
  DATABASE_POOL_ROLLOUT_SURGE_FACTOR
  * (DATABASE_POOL_API_REPLICAS * DATABASE_POOL_API_MAX_CONNS
     + DATABASE_POOL_WORKER_REPLICAS * DATABASE_POOL_WORKER_MAX_CONNS)
  + DATABASE_POOL_OPERATIONS_RESERVE

rollout_peak must be <= DATABASE_CONNECTION_BUDGET
```

Each process also verifies that its own `DATABASE_POOL_MAX_CONNS` matches the role-specific budget input (`DATABASE_POOL_API_MAX_CONNS` for API, `DATABASE_POOL_WORKER_MAX_CONNS` for worker).

Invalid or over-budget settings fail closed at process startup.

## Deployment coordination (#49)

Deployment owns replica counts and database service sizing. This document owns application pool semantics.

When sizing production:

1. Choose `DATABASE_CONNECTION_BUDGET` from the managed PostgreSQL plan.
2. Reserve `DATABASE_POOL_OPERATIONS_RESERVE` for schema migrations, operator consoles, backups, and monitoring scrapes that connect directly to Postgres.
3. Set per-role `DATABASE_POOL_MAX_CONNS` from expected concurrent request/worker load, not CPU count.
4. Set `DATABASE_POOL_API_REPLICAS` and `DATABASE_POOL_WORKER_REPLICAS` to the steady-state counts from the deployment manifest.
5. Keep `DATABASE_POOL_ROLLOUT_SURGE_FACTOR=2` (or higher for blue/green with longer overlap) so rolling updates that temporarily run old and new replicas together do not exceed the budget.

Example with budget `100`, `2` API replicas at `10` connections, `1` worker replica at `5` connections, reserve `5`, surge factor `2`:

```text
rollout_peak = 2 * (2*10 + 1*5) + 5 = 55 <= 100
```

## Saturation behavior

- Pool size is fixed by `DATABASE_POOL_MAX_CONNS`; Synaudio does not create unbounded connections under load.
- DBTX operations (`Exec`, `Query`, `QueryRow`) and transaction establishment (`Begin`) enforce `DATABASE_POOL_ACQUIRE_TIMEOUT` only while waiting for a connection. Once acquired, the original request/job context controls query execution.
- A shorter request/job deadline or cancellation takes precedence over the pool acquire timeout.
- Query/result wrappers retain the acquired connection until rows are closed/exhausted or `QueryRow.Scan` completes, preserving normal pgx lifecycle semantics.
- Worker backlog sampling also uses the bounded DBTX path so saturation cannot strand that loop indefinitely.
- Pool saturation is observable through bounded Prometheus gauges/counters documented in `observability.md`.
