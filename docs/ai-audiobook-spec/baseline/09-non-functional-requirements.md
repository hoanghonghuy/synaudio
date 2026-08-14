# 09 — Non-Functional Requirements

## 1. Performance

Metadata API target ban đầu:

```text
p95 < 500 ms
```

Generation là async/job-based. Audio không proxy qua API.

---

## 2. Reliability

Generation phải:

```text
retryable
idempotent where applicable
stateful
auditable
recoverable
```

---

## 3. Canon integrity

```text
Draft cannot mutate Official Canon
Memory extraction only after approved content
Canon commit is transactional
Major decision requires decision gate
Retcon requires impact analysis
```

---

## 4. Story continuity protection

```text
Context Builder
Story Writer
Continuity Reviewer
Fact Conflict Detection
Creative Decision Gate
Admin Review
Memory Extractor
Canon Commit
```

---

## 5. Cost control

External call log nên có provider, model, input/output size, duration, estimated cost và generation run.

Mock provider tồn tại để dev không tốn quota.

---

## 6. Security / environment safety

Development mặc định không truy cập production DB/storage.

Production validate và có thể reject mock provider hoặc unsafe debug config.

---

## 7. Auditability

Phải trace được:

```text
who
when
what
story/chapter
generation run
canon version
source/provenance
```

Audit append-only.

---

## 8. Maintainability

Cho phép thay:

```text
Gemini → provider khác
MinIO → R2
Postgres job queue → task queue
in-process Memory → remote service
```

mà không rewrite domain.

---

## 9. Story quality requirements

Default Story Creation Contract:

```text
minimum chapter audio duration = 20 minutes
target chapter audio duration = 30 minutes
```

Không dùng filler hoặc giả giảm tốc audio để đạt duration.

---

## 10. Retcon safety

Published Canon change là dangerous operation. Hệ thống phải warn, analyze dependency, tạo impact report/repair plan, mark stale và yêu cầu Admin confirm.

---

## 11. Time convention

```text
UTC TIMESTAMPTZ
```
