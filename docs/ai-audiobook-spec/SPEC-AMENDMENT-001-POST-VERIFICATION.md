# SPEC AMENDMENT 001 — Post-Verification Corrections

**Status:** REQUIRED FOR IMPLEMENTATION  
**Amendment ID:** `SPEC-AMENDMENT-001`  
**Verification date:** 2026-08-13  
**Base specification:** `spec_final.md` — Version `1.0 FINAL`  
**Purpose:** Khóa các correction/clarification phát hiện trong lần review + verify cuối trước khi bắt đầu migration và implementation.

---

# 1. Precedence / Cách đọc tài liệu

Effective implementation specification của project là:

```text
spec_final.md
+
SPEC-AMENDMENT-001-POST-VERIFICATION.md
```

Tài liệu này **không thay thế toàn bộ** `spec_final.md`.

Nó chỉ supersede các section/decision được ghi rõ bên dưới.

Mọi phần của `spec_final.md` không được nhắc tới trong amendment này vẫn giữ nguyên và tiếp tục authoritative.

Khi implementation, developer/AI agent **MUST read theo thứ tự**:

```text
1. spec_final.md
2. SPEC-AMENDMENT-001-POST-VERIFICATION.md
```

Nếu hai file khác nhau tại một điểm được amendment này nêu rõ:

```text
SPEC-AMENDMENT-001
wins.
```

Không được tự merge theo phỏng đoán.

---

# 2. Verification result

Sau khi review lại Product/Business/Domain/Architecture/Schema:

```text
PRODUCT / BUSINESS / DOMAIN
= giữ nguyên, không reopen.

ARCHITECTURE DIRECTION
= giữ nguyên.

PHYSICAL / OPERATIONAL DETAILS
= áp các correction trong amendment này trước khi viết migration/code tương ứng.
```

Các correction dưới đây **không yêu cầu redesign toàn bộ hệ thống**.

Chúng chủ yếu khóa:

- source of truth;
- durability;
- auth deployment topology;
- Canon immutability;
- queue behavior;
- state-transition edge cases;
- schema relationship ambiguity.

---

# A-001 — Production Auth Domain, Cookie, CORS & CSRF Topology

## Supersedes / clarifies

`spec_final.md`:

- Section 8 — Authentication Architecture
- Section 168 — Production Infrastructure
- các security/CORS statements liên quan.

## Problem

Base spec đã chốt:

```text
Access JWT in-memory
+
rotating opaque Refresh Token
+
HttpOnly Secure cookie
```

nhưng chưa khóa production domain topology.

Nếu frontend dùng trực tiếp vendor domain kiểu:

```text
project.vercel.app
```

và API:

```text
project.onrender.com
```

thì browser coi đây là cross-site deployment.

Không được lấy raw vendor-domain topology đó làm production auth baseline.

## FINAL DECISION

Production MUST use same-site custom subdomains dưới cùng một registrable domain.

Example:

```text
app.example.com
api.example.com
```

Production refresh cookie:

```text
Host: api.example.com

HttpOnly = true
Secure   = true
SameSite = Lax
Domain   = NOT SET
Path     = /
```

Recommended cookie name:

```text
__Host-refresh_token
```

vì cookie này:

- Secure;
- host-only;
- Path `/`;
- không có `Domain=`.

Frontend API requests cần credentials khi gọi refresh/logout endpoints.

CORS production:

```text
Allowed Origin:
https://app.example.com
```

Không dùng wildcard:

```text
Access-Control-Allow-Origin: *
```

với credentialed request.

State-changing cookie-backed auth endpoints MUST validate request Origin và/or CSRF protection appropriate to the endpoint.

Examples:

```text
POST /auth/refresh
POST /auth/logout
POST /auth/logout-all
```

API chỉ trust exact configured frontend origin(s).

## Preview / Hobby

Raw Vercel/Render domains có thể dùng cho dev/preview nhưng không phải production authentication topology.

Preview auth phải được cấu hình riêng nếu cần.

Không được làm production security design phụ thuộc third-party-cookie behavior.

## Additional configuration

Add conceptual config:

```text
APP_PUBLIC_URL=
API_PUBLIC_URL=
CORS_ALLOWED_ORIGINS=
REFRESH_COOKIE_NAME=__Host-refresh_token
```

`APP_PUBLIC_URL` và `API_PUBLIC_URL` production phải thuộc cùng site baseline đã chốt.

---

# A-002 — Gemini TTS Is a Preferred Adapter, Not an Immutable Dependency

## Supersedes / clarifies

`spec_final.md`:

- Section 76 — Provider Abstraction
- Section 78–80 — Narration / Audio
- Section 190 — Frozen Implementation Decisions.

## Problem

Base spec ghi:

```text
Gemini TTS primary direction
```

Điều này đúng làm current provider preference nhưng không nên được hiểu là:

```text
Gemini TTS
=
permanent architectural dependency
```

External AI/TTS products, model IDs, quotas và availability có thể thay đổi.

## FINAL DECISION

Frozen architecture is:

```text
TTSProvider abstraction
```

not a specific model.

Current preferred adapter:

```text
Gemini TTS
```

Exact model ID:

```text
TTS_MODEL
```

MUST remain runtime/configuration data.

Before production rollout hoặc khi đổi model:

```text
re-verify:
- model availability
- language support
- quota
- pricing
- preview/stable status
- output limits
```

Không migration/domain change chỉ vì exact Gemini TTS model đổi.

## Long-form audio implication

Tiếp tục giữ segmented TTS architecture.

Do not request one entire 20–30+ minute Chapter as one fragile TTS operation.

Pipeline remains:

```text
Narration Revision
→ Segments
→ TTS per Segment
→ durable staging
→ FFmpeg
→ final AudioAsset
```

Provider fallback remains controlled by Workflow/Provider Policy.

---

# A-003 — Render Deployment Profiles: Hobby vs Production

## Supersedes / clarifies

`spec_final.md`:

- Section 168 — Production Infrastructure
- Section 169 — API / Worker Separation.

## Problem

Architecture đã đúng khi tách:

```text
cmd/api
cmd/worker
```

nhưng không được giả định rằng một free Render Background Worker luôn tồn tại.

Free Render Web Service cũng có idle spin-down và ephemeral filesystem.

## FINAL DECISION

Code architecture ALWAYS keeps:

```text
cmd/api
cmd/worker
```

Deployment has two profiles.

## Profile 1 — HOBBY / FREE / DEMO

Allowed:

```text
Render Free Web Service
```

Worker may run embedded/co-located only as a **best-effort hobby mode**.

This profile:

- is not production-grade;
- may spin down;
- may interrupt long background work;
- must rely on persistent Job state;
- must recover after restart;
- must never rely on local filesystem.

Conceptual config:

```text
WORKER_MODE=embedded
```

## Profile 2 — PRODUCTION

Use:

```text
Render Web Service → API
Render Background Worker / equivalent paid worker → Worker
```

Conceptual config:

```text
WORKER_MODE=separate
```

API and Worker still share the same codebase/modules/database but are separate processes/services.

## Invariant

Deployment profile must not change business behavior.

Only execution availability/performance differs.

---

# A-004 — PostgreSQL Queue on Neon Uses Durable Rows + Adaptive Polling

## Supersedes / clarifies

`spec_final.md`:

- Section 70 — PostgreSQL Job Queue
- Section 168–171 — Production/Operations.

## Problem

Neon can scale compute to zero when inactive.

A worker polling unnecessarily often can keep the database active.

Session-level mechanisms such as:

```text
LISTEN / NOTIFY
advisory locks
temporary session state
```

must not be relied on as durable queue truth across connection lifecycle.

## FINAL DECISION

Authoritative queue state is ALWAYS persisted in:

```text
generation_jobs
generation_job_dependencies
generation_job_attempts
```

Worker claiming stays:

```sql
FOR UPDATE SKIP LOCKED
```

`LISTEN / NOTIFY` MAY be added later only as:

```text
latency optimization
```

never as authoritative durability.

## Adaptive polling

Worker polling strategy:

```text
queue has work
→ poll aggressively / immediately continue

queue empty
→ back off progressively

new request / wake hint
→ reset backoff
```

Example implementation range:

```text
busy:       immediate / sub-second loop after each claimed batch
idle:       ~1s → 2s → 5s → 10s → 30s
```

Exact numbers are configuration, not domain rules.

## Invariant

A DB suspend/reconnect must not lose:

- pending jobs;
- job dependency;
- attempts;
- retry schedule;
- cancellation;
- staleness state.

---

# A-005 — Successful TTS Segments Must Be Durably Staged

## Supersedes / clarifies

`spec_final.md`:

- Section 79 — Audio Pipeline
- Section 125 — Retention
- Section 161 — `tts_segments`
- Section 168 — Render filesystem.

## Problem

Base spec correctly says backend filesystem is ephemeral.

However successful TTS segment output cannot live only in:

```text
/tmp
```

until FFmpeg finishes.

Otherwise:

```text
18/20 segments completed
→ worker restart
→ all 18 successful segments lost
```

which violates the already-frozen partial-retry behavior.

## FINAL DECISION

After each TTS segment succeeds:

```text
TTS response
→ validate segment
→ upload segment to private Object Storage staging
→ mark tts_segment SUCCEEDED
```

Staging path example:

```text
staging/tts/{storyId}/{chapterId}/{narrationRevisionId}/{segmentNo}.wav
```

or equivalent ID-based path.

`/tmp` is only:

```text
working copy
```

for:

- provider response buffering;
- FFmpeg input download;
- FFmpeg output;
- checksum/validation.

## Schema correction

Rename conceptual column:

```text
tts_segments.temp_storage_key
```

to:

```text
tts_segments.staging_storage_key
```

because the object is durable enough for retry/recovery, not merely local temp data.

## Cleanup

After final AudioAsset becomes READY and retention window passes:

```text
delete staging segment objects
```

Cancelled/failed abandoned staging objects are removed by lifecycle cleanup.

Recommended staging retention:

```text
7 days
```

configurable.

Pinned/debug runs may retain longer if explicitly configured.

---

# A-006 — `chapters.current_audio_asset_id` Is the Single Active-Audio Source of Truth

## Supersedes

`spec_final.md`:

- Section 148 — `chapters`
- Section 161 — `audio_assets`
- previous partial unique `is_active` design.

## Problem

Base schema currently contains both:

```text
chapters.current_audio_asset_id
```

and:

```text
audio_assets.is_active
```

plus:

```text
UNIQUE(chapter_id) WHERE is_active = true
```

This creates two writable representations of the same fact.

They can drift.

## FINAL DECISION

Single authoritative pointer:

```text
chapters.current_audio_asset_id
```

`audio_assets` MUST NOT contain:

```text
is_active
```

The previous partial unique index:

```text
UNIQUE(chapter_id)
WHERE is_active = true
```

MUST NOT be implemented.

## Corrected `audio_assets`

Conceptually:

```text
id
chapter_id
version_no
source_narration_revision_id
status
storage_key
mime_type
size_bytes
duration_ms
bitrate_kbps
checksum
generation_run_id
created_at
```

Unique:

```text
UNIQUE(chapter_id, version_no)
```

## Promotion

Audio promotion transaction:

```text
verify candidate AudioAsset READY
verify candidate belongs to Chapter
verify source narration is current/valid
UPDATE chapters.current_audio_asset_id = candidate_id
```

Old AudioAsset remains historical.

No active flag mutation required.

---

# A-007 — CanonVersion Rows Are Immutable; Promotion Creates New Versions

## Supersedes / clarifies

`spec_final.md`:

- Section 57 — Canon Lifecycle
- Section 58 — Batch / Provisional Canon
- Section 157 — Canon Tables.

## Problem

Base spec contains both:

```text
canon_branches.type
```

and:

```text
canon_versions.status
```

If implemented carelessly, a developer could mutate:

```text
PROVISIONAL → OFFICIAL
```

on the same row.

That would weaken historical provenance.

## FINAL DECISION

**CanonVersion rows are immutable.**

Never mutate an existing CanonVersion from:

```text
PROVISIONAL
```

to:

```text
OFFICIAL
```

## Effective physical model

`canon_branches.type` determines lineage:

```text
OFFICIAL
PROVISIONAL
RETCON
```

`canon_versions` does NOT need a mutable lifecycle `status`.

Recommended correction:

```text
remove canon_versions.status
```

A CanonVersion belongs to one branch and its meaning is derived from branch type.

## Meaning of old logical Canon states

Logical lifecycle remains understandable as:

```text
DRAFT
→ PROVISIONAL
→ OFFICIAL
```

but physical semantics are:

### DRAFT

Uncommitted validated Canon Change Set / Generation artifact.

It is not an Official CanonVersion.

### PROVISIONAL

Immutable CanonVersion created on a `PROVISIONAL` branch.

### OFFICIAL

Immutable CanonVersion created on the `OFFICIAL` branch.

## Promotion

When promoting provisional work:

```text
Provisional CanonVersion P
→ create NEW Official CanonVersion O
```

Official version stores:

```text
source_provisional_version_id = P.id
```

No mutation of P.

## Retcon

Retcon workspace versions live on `RETCON` branch.

Final Retcon Apply creates/promotes new immutable versions into the Official line.

Historical RETCON versions remain historical candidate lineage.

## Story head

Authoritative Official head remains:

```text
stories.current_official_canon_version_id
```

updated transactionally only after successful Official commit/promotion.

---

# A-008 — Complete `Revise Before Publish` Workflow

## Supersedes / clarifies

`spec_final.md`:

- Chapter lifecycle
- Artifact versioning
- API endpoint:
  `POST /admin/chapters/{chapterId}/revise-before-publish`

## Problem

Endpoint exists but base final spec does not fully lock the lifecycle after a Chapter has already become Official Canon but is not yet Published.

## FINAL DECISION

`Revise Before Publish` is allowed only when:

```text
Chapter has Official Canon
AND
Chapter is NOT PUBLISHED
```

Typical source state:

```text
PRODUCTION
or
READY
```

## Workflow

```text
Current Official Content Revision R1
        ↓
Admin chooses Revise Before Publish
        ↓
create new Content Revision R2
        ↓
R2 becomes current candidate
        ↓
Chapter enters CONTENT_REVIEW for R2
        ↓
Continuity / Quality / Safety / Duration review
        ↓
Admin Content Approval R2
        ↓
Memory Extraction R2
        ↓
Canon Reconciliation
        ↓
create NEW Official CanonVersion
        ↓
Chapter → PRODUCTION
        ↓
new Narration
        ↓
new Audio
        ↓
READY
```

## Canon rule

Do NOT overwrite or mutate the prior Official CanonVersion.

The correction creates a new Official CanonVersion.

Because content was never publicly released, this operation is:

```text
Pre-Publish Canon Revision
```

not True Retcon.

## Downstream impact

If later Draft/Provisional Chapters were generated using the previous Official content:

```text
analyze dependency
→ mark affected downstream content STALE
```

Do not silently regenerate.

## Artifact invalidation

Changing Story prose invalidates prior derived:

```text
Narration
Audio
content reviews bound to old revision
```

as already defined by base spec.

---

# A-009 — Unpublish Only Returns to READY When READY Is Still True

## Supersedes / clarifies

`spec_final.md`:

- Section 44 — READY
- Section 105 — Unpublish.

## Problem

Base spec says conceptually:

```text
PUBLISHED → READY
```

But READY has a strong meaning:

```text
publishable right now
```

Therefore a Chapter cannot be forced to READY when its current artifacts no longer satisfy READY gates.

## FINAL DECISION

### Normal voluntary Unpublish

If exact current revision/artifacts remain valid:

```text
PUBLISHED
→ READY
```

This is the normal path.

### Audio/Narration invalidation

If unpublishing occurs because current narration/audio requires repair:

```text
PUBLISHED
→ PRODUCTION
```

because Canon remains Official while production artifacts are repaired.

After valid audio exists:

```text
PRODUCTION
→ READY
```

### Published prose / Canon must change

This is not a normal Unpublish+Edit workflow.

Use:

```text
True Retcon / Canon Revision
```

The publication must be withdrawn according to sequential rules while Retcon repair proceeds.

Unpublish must never become a loophole to bypass Published Canon protection.

## Sequential rule remains

Unpublishing Chapter N cannot leave N+1..M public.

Affected later Chapters are unpublished sequentially as already specified.

If later Chapters themselves remain fully valid:

```text
they may move to READY
```

while the target Chapter is repaired.

---

# A-010 — Final Schema & Semantic Clarifications

This section collects smaller corrections that MUST be applied before writing the relevant migrations.

## A-010.1 Planning Phase Applies to FINITE and OPEN_ENDED

FINAL interpretation:

```text
planning_phase
=
ONGOING
CLOSING
FINAL_ARC
COMPLETED
```

is available to both:

```text
FINITE
OPEN_ENDED
```

Do not create a second planning-phase model for FINITE.

---

## A-010.2 `public_rating` and `public_warnings` Are Public Projections

`stories.public_rating` and:

```text
stories.public_warnings
```

are not an independent raw source of content truth.

They are:

```text
current public projection / denormalized catalog state
```

managed by Governance/Publication application service.

Inputs include:

- current Published Chapter classifications;
- required Story-level Content Profile constraints;
- explicit valid Admin classification decisions.

Do not expose a generic edit that can make public metadata contradict current Published Chapter classification.

Publish/unpublish/reclassification updates the projection transactionally where appropriate.

---

## A-010.3 `public_since` Means First Public Launch

Define:

```text
stories.public_since
=
first time Story successfully transitioned PRIVATE → PUBLIC
```

It is set once.

Do NOT reset it when:

```text
PUBLIC → PRIVATE → PUBLIC
```

again.

`last_published_at` remains the timestamp used for:

```text
Recently Updated
```

and changes when new public Chapter release occurs.

---

## A-010.4 Bootstrap First Admin Uses One-Time Setup, Never Plaintext Seed Password

The platform requires a controlled bootstrap path.

Add command concept:

```text
cmd/bootstrap-admin
```

or equivalent explicit administrative command.

Bootstrap input:

```text
email
```

Bootstrap MUST NOT persist or ship a plaintext default password.

Recommended flow:

```text
bootstrap command
→ create one-time setup token
→ store token HASH only
→ print setup URL/token once
→ Admin chooses password
→ Admin configures TOTP
→ setup token invalidated
```

Until MFA is configured:

```text
Admin privileged capability remains blocked
```

except bootstrap-security completion actions.

The Last Active Admin Guard applies after bootstrap completes.

---

## A-010.5 Circular Current-Version Foreign Keys Must Be Migration-Safe

Examples:

```text
stories.current_story_bible_version_id
stories.current_ending_plan_version_id
stories.current_content_profile_version_id
stories.current_official_canon_version_id

chapters.current_plan_revision_id
chapters.current_content_revision_id
chapters.current_narration_revision_id
chapters.current_audio_asset_id
```

These pointers may be:

```text
NULL
```

during initial/root creation when the referenced version does not exist yet.

Migration strategy:

```text
1. create base identity tables
2. create version/artifact tables
3. add pointer FK constraints after both sides exist
```

or equivalent migration-safe ordering.

Do not permanently disable referential integrity to avoid circular creation.

---

## A-010.6 ContextSnapshot Relationship Is One-Way; Multiple Snapshots per Run Are Allowed

Base schema contains:

```text
generation_runs.context_snapshot_id
```

and:

```text
context_snapshots.run_id
```

FINAL DECISION:

Remove:

```text
generation_runs.context_snapshot_id
```

Keep:

```text
context_snapshots.run_id
```

A GenerationRun MAY have multiple ContextSnapshots.

Add conceptual fields:

```text
context_snapshots.job_id NULL
context_snapshots.purpose
```

Examples:

```text
WRITER
CONTINUITY_REVIEW
QUALITY_REVIEW
SAFETY_REVIEW
MEMORY_EXTRACTION
NARRATION
```

A snapshot represents the exact context for a specific AI operation/stage.

Historical ContextSnapshots remain immutable.

---

## A-010.7 Dependency-Terminal Jobs Must Not Remain PENDING Forever

If a Job dependency reaches terminal state:

```text
FAILED
CANCELLED
STALE
```

and the workflow has no explicit alternate/fallback edge, dependent Jobs MUST be resolved.

Default behavior:

```text
dependent Job
→ CANCELLED
```

with reason:

```text
DEPENDENCY_FAILED
DEPENDENCY_CANCELLED
DEPENDENCY_STALE
```

as appropriate.

Do not leave dependent jobs indefinitely:

```text
PENDING
```

Example:

```text
Chapter 103 Writer FAILED
→ Chapter 104 jobs CANCELLED / DEPENDENCY_FAILED
→ Chapter 105 jobs CANCELLED / DEPENDENCY_FAILED
```

This preserves the frozen batch dependency rule.

---

# 3. Corrected Physical Schema Delta

Before migrations, apply at minimum:

## Remove from `audio_assets`

```text
is_active
```

Remove:

```text
UNIQUE(chapter_id)
WHERE is_active = true
```

Keep/add:

```text
UNIQUE(chapter_id, version_no)
```

Authoritative current audio:

```text
chapters.current_audio_asset_id
```

## Rename on `tts_segments`

```text
temp_storage_key
→ staging_storage_key
```

## Remove from `generation_runs`

```text
context_snapshot_id
```

## Add/clarify on `context_snapshots`

```text
run_id NOT NULL
job_id NULL
purpose NOT NULL
```

Multiple ContextSnapshots per GenerationRun are valid.

## Canon

Recommended:

```text
remove canon_versions.status
```

Use immutable CanonVersion rows + `canon_branches.type`.

Promotion creates new CanonVersion rows.

---

# 4. Corrected Runtime Invariants

Implementation MUST satisfy:

```text
Production auth uses same-site custom-domain topology.

Refresh-session security does not depend on raw third-party vendor domains.

TTSProvider abstraction is frozen; Gemini TTS is current preferred adapter.

Render free deployment is hobby/demo only.

PostgreSQL job rows are authoritative.

LISTEN/NOTIFY is optional acceleration only.

Queue polling backs off when idle.

Successful TTS segments survive worker restart through durable staging.

Local /tmp is never the only copy of a successful expensive TTS segment.

chapters.current_audio_asset_id is the only active-audio pointer.

CanonVersion rows are immutable.

Provisional → Official promotion creates new Official rows.

Revise Before Publish creates a new revision and new Official CanonVersion.

Unpublish returns to READY only when READY invariants remain true.

Planning phases apply to FINITE and OPEN_ENDED.

Story public rating/warnings are controlled projections.

public_since is first public launch time and does not reset.

First Admin bootstrap never uses a shipped/plaintext default password.

Circular current-version references are migration-safe.

A GenerationRun can have multiple ContextSnapshots.

Failed/cancelled/stale dependencies resolve dependent jobs.
```

---

# 5. Effective Deployment Baseline After Amendment

## Local

```text
Vue
Go API
Go Worker
PostgreSQL
MinIO
Mock or Real AI
Mock or Real TTS
```

Successful staged TTS media is persisted to MinIO.

## Hobby / Demo cloud

```text
Vercel frontend
Render Free Web Service
Neon
Cloudflare R2
```

Embedded worker is best-effort only.

Not production-grade.

## Production

```text
app.example.com       → Vercel
api.example.com       → Render Web Service
Worker                → Render Background Worker / equivalent
PostgreSQL            → Neon
Object Storage        → Cloudflare R2 private
```

Audio remains direct signed R2 access.

---

# 6. Effective Canon Promotion Example

```text
Official O100
        ↓
Provisional P101
        ↓
Provisional P102
```

Promote P101:

```text
P101 remains immutable
        ↓
create Official O101
source_provisional_version_id = P101
        ↓
stories.current_official_canon_version_id = O101
```

Never:

```sql
UPDATE canon_versions
SET status = 'OFFICIAL'
WHERE id = :provisional_id;
```

---

# 7. Effective Audio Promotion Example

Current:

```text
chapters.current_audio_asset_id = AudioV1
```

Generate candidate:

```text
AudioV2 status = PROCESSING
→ quality gate
→ AudioV2 status = READY
```

Promotion transaction:

```text
validate AudioV2
validate source narration/content lineage
UPDATE chapters.current_audio_asset_id = AudioV2
```

AudioV1 remains historical.

---

# 8. Effective Revise-Before-Publish Example

Current:

```text
Chapter 25 = READY
Content Revision 7
Official Canon O25
Narration N3
Audio A2
```

Admin finds prose issue before public release:

```text
Revise Before Publish
→ Content Revision 8
→ CONTENT_REVIEW
→ reviews
→ approval
→ Memory Extraction
→ new Official CanonVersion
→ PRODUCTION
→ Narration N4
→ Audio A3
→ READY
```

Affected downstream provisional/draft candidates:

```text
dependency analysis
→ STALE if affected
```

No True Retcon because the revised Chapter was not Published.

---

# 9. Verification Basis — Official External Docs

External provider/platform facts below were re-verified on 2026-08-13.

## Google Gemini TTS

```text
https://ai.google.dev/gemini-api/docs/speech-generation
https://ai.google.dev/gemini-api/docs/models
```

Gemini TTS remains suitable as the current preferred adapter, while exact model/version remains configuration.

## Render

```text
https://render.com/docs/free
https://render.com/docs/background-workers
```

Free Web Services have free-instance limitations; production Worker remains a separate paid/equivalent worker target.

## Neon

```text
https://neon.com/docs/introduction/scale-to-zero
https://neon.com/docs/reference/compatibility
```

Durable PostgreSQL rows remain queue truth; session-only state is non-authoritative.

## Cloudflare R2

```text
https://developers.cloudflare.com/r2/api/s3/presigned-urls/
https://developers.cloudflare.com/r2/buckets/cors/
```

Private R2 + authorized presigned browser access remains the selected media architecture.

---

# 10. What Is NOT Changed

This amendment does NOT reopen:

```text
GUEST / USER / ADMIN V1

Admin != God Mode

Story platform ownership

StoryGenerationPolicy immutability

StoryWorkflowSettings mutability

Story/Chapter lifecycle intent

Content Approval revision binding

Memory Extraction after final approval

No Auto-Publish V1

Official and Published sequence rules

StoryFact / PlotThread / Story Memory direction

ContextSnapshot immutability

Retry != Regenerate

Rewrite != Regenerate

Sequential Batch Generation

CreativeDecision model

Retcon requirement for Published canonical changes

Narration/Audio repair != Retcon when Story Content is unchanged

GENERAL / TEEN / MATURE baseline

Audit append-only

Modular Monolith

Go + PostgreSQL + pgx/sqlc

Vue 3 + Vite

MinIO local / R2 production

Neon direction

Vercel frontend / Render backend direction

OpenAPI contract

Implementation phase roadmap
```

---

# 11. Implementation Readiness

After applying:

```text
spec_final.md
+
SPEC-AMENDMENT-001-POST-VERIFICATION.md
```

the effective spec is:

```text
FINAL EFFECTIVE SPECIFICATION
READY FOR IMPLEMENTATION
```

No additional Product/Business discovery round is required before Phase 0 / Phase 1.

Future changes follow the Change Control process already defined in `spec_final.md`.

---

# 12. First Implementation Action

Before writing the first real migration:

```text
1. Keep this amendment beside spec_final.md.
2. Mark both files as mandatory AI/dev context.
3. Apply schema deltas from Section 3.
4. Use corrected Canon/audio/context ownership.
5. Use same-site production auth topology.
6. Implement durable TTS staging.
7. Then proceed with Phase 0 / Phase 1.
```

**This amendment closes the post-verification issues identified before implementation.**
