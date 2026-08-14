# AI Audiobook Platform — FINAL SYSTEM SPECIFICATION

**Version:** 1.0 FINAL
**Status:** FROZEN BASELINE FOR IMPLEMENTATION
**Purpose:** Single source of truth cho Product, Business Rules, Domain, Architecture, Data Model, API, Security, Worker/Jobs, Infrastructure và Implementation Roadmap.

> Tài liệu này là baseline để bắt đầu triển khai.
> Các consolidated/baseline document cũ giữ giá trị lịch sử, nhưng nếu có wording khác nhau thì **FINAL SYSTEM SPECIFICATION này được ưu tiên**.

---

# 1. Product Goal

AI Audiobook Platform là nền tảng truyện/audiobook tiếng Việt trong đó AI hỗ trợ toàn bộ content-production pipeline:

```text
Story Idea
→ Planning
→ Writing
→ Review
→ Canon / Memory
→ Narration
→ TTS
→ Audio
→ Publish
→ Listener
```

Hệ thống phải phục vụ được:

* Story ngắn khoảng 10–50 Chapters.
* Story finite khoảng 100–300+ Chapters.
* Story open-ended có thể chạy 500+ Chapters.
* Listener có thể **Read** hoặc **Listen**.
* Admin vận hành Story như một content-production pipeline thay vì CRUD đơn giản.

V1 Content Origin:

```text
AI_GENERATED
```

Future-compatible:

```text
HUMAN_WRITTEN
AI_ASSISTED
LICENSED / PUBLIC_DOMAIN
```

Working product name trong code/spec:

```text
AI Audiobook Platform
```

Brand name cuối không phải architectural blocker.

---

# 2. Core Invariants

Các rule sau là invariant và không được implementation tùy tiện phá:

```text
Admin != God Mode.

Draft cannot mutate Official Canon.

Published canonical content cannot be normally edited.

StoryGenerationPolicy is immutable after Story creation.

StoryWorkflowSettings is mutable.

Canonical Memory Extraction only uses final approved Content Revision.

Official Chapter sequence must be contiguous.

Published Chapter sequence must be contiguous.

AI output is untrusted until validated.

Generation status != Chapter status.

Retry != Regenerate.

Rewrite != Regenerate.

Stale results cannot overwrite current state.

Cancelled Runs cannot late-apply provider results.

Audio is generated before publication and stored.

Playback never regenerates TTS.

Go API does not proxy large audio files.

Audit is append-only.

Historical ContextSnapshots are immutable.

V1 never auto-publishes.

Story belongs to the platform in V1.

created_by is provenance, not Story ownership.

ARCHIVED / SUPERSEDED / INVALIDATED are business states,
not generic delete flags.

V1 starts as Modular Monolith, not Microservices.
```

---

# 3. Product Roles

V1 only has:

```text
GUEST
USER
ADMIN
```

## 3.1 Guest

Can:

* browse/search public Stories;
* read Story metadata;
* list Published Chapters;
* read Published Chapter text;
* listen to Published audio;
* retain local browser listening progress.

Guest is **not a persistent database User**.

## 3.2 User

Includes Guest capabilities plus:

* Profile;
* Favorite Stories;
* server-backed Listening Progress;
* Continue Listening;
* multi-device progress;
* session/security management;
* account deactivation/deletion request.

## 3.3 Admin

Can operate:

* Story creation/lifecycle;
* Story Bible;
* Characters;
* Arcs;
* Ending;
* Chapter Plans;
* Generation;
* Reviews;
* Content Approval;
* Creative Decisions;
* Canon/Memory;
* Narration;
* TTS/Audio;
* Publish/Unpublish;
* Canon Data Repair;
* Retcon;
* Audit;
* user/admin security management.

Admin can still use normal listener functionality.

---

# 4. Identity Model

One person uses one Account/Identity.

Do **not** create independent:

```text
users
admins
```

account systems.

Admin is an authenticated identity with elevated roles/permissions.

---

# 5. Account Lifecycle

```text
ACTIVE
SUSPENDED
DEACTIVATED
```

## ACTIVE

Normal authenticated use.

## SUSPENDED

* authenticated operations denied;
* existing Favorite/Progress preserved;
* audit preserved;
* user may still consume public content anonymously as Guest.

## DEACTIVATED

* authenticated access disabled;
* user data enters account lifecycle/deletion process;
* historical audit/provenance is not blindly deleted.

---

# 6. Authentication Baseline

V1 supports:

```text
EMAIL + PASSWORD
```

Email is the primary login identifier.

`display_name` is not a login identifier.

Email uniqueness is enforced after normalization.

Public signup always creates normal User.

Client cannot request:

```text
role = ADMIN
```

during registration.

---

# 7. Email Verification

Normal User may still listen/read/favorite/save progress before email verification because Guest already has public access.

Admin capability requires:

```text
ACTIVE account
+
verified email
+
required MFA
```

---

# 8. Authentication Architecture

Implementation baseline:

```text
Short-lived Access JWT
+
Rotating opaque Refresh Session
```

Rules:

* access token stored only in frontend memory;
* do not store access token in localStorage;
* refresh token stored in `HttpOnly + Secure` cookie;
* refresh token is random opaque data;
* backend stores only refresh-token hash;
* refresh token rotates on refresh;
* logout revokes current session;
* logout-all revokes all sessions.

Default configurable values:

```text
Access token      ~15 minutes
Absolute session  ~30 days
Idle session      ~14 days
Recent Auth       ~10 minutes
```

These are configuration defaults, not immutable product rules.

Privileged Admin requests re-check current account/session/permissions so suspension/revocation takes effect promptly.

---

# 9. Password Security

Password hash:

```text
Argon2id
```

Password Reset:

```text
Forgot Password
→ one-time recovery credential
→ new password
→ recovery credential invalidated
→ existing sessions revoked by default
```

Public forgot-password response must not clearly expose whether an email exists.

---

# 10. MFA

Normal User:

```text
optional
```

Admin:

```text
required
```

V1:

```text
TOTP + Recovery Codes
```

TOTP secret encrypted at rest.

Recovery codes stored only as hashes.

---

# 11. Recent Authentication

Required for critical operations such as:

```text
Retcon Apply

Grant Admin

Revoke Admin

Suspend Admin

Deactivate Admin

Change privileged MFA/security state

Future destructive production purge
```

Normal Chapter Publish does not require re-authentication every time.

---

# 12. Last Active Admin Guard

Platform must never reach:

```text
0 active privileged Admins
```

If Admin A is the last Admin, A cannot:

* revoke own Admin role;
* suspend/deactivate themselves;
* remove required privileged authentication;

until another valid Admin exists.

---

# 13. Authorization Model

Roles are bundles of granular permissions.

V1 roles remain:

```text
GUEST
USER
ADMIN
```

Permission vocabulary is granular to enable future:

```text
EDITOR
PUBLISHER
OPERATOR
```

without redesigning authorization.

Authorization order:

```text
Identity valid?
↓
Account ACTIVE?
↓
Authentication assurance sufficient?
↓
Permission exists?
↓
Resource scope allowed?
↓
Business invariant allows operation now?
↓
Execute
```

Authentication says:

> Who are you?

Authorization says:

> What may you attempt?

Business rules say:

> Is the operation valid now?

---

# 14. Core Admin Permissions

Story:

```text
STORY_CREATE
STORY_METADATA_EDIT
STORY_ACTIVATE
STORY_ARCHIVE
STORY_RESTORE
STORY_VISIBILITY_MANAGE
STORY_WORKFLOW_SETTINGS_MANAGE
```

Planning:

```text
STORY_BIBLE_VIEW
STORY_BIBLE_MANAGE
CHARACTER_VIEW
CHARACTER_MANAGE
ARC_VIEW
ARC_MANAGE
ENDING_PLAN_VIEW
ENDING_PLAN_MANAGE
CHAPTER_PLAN_VIEW
CHAPTER_PLAN_MANAGE
STORY_FACT_VIEW
PLOT_THREAD_VIEW
```

Chapter:

```text
CHAPTER_CREATE
CHAPTER_GENERATE
CHAPTER_EDIT_DRAFT
CHAPTER_REVIEW
CHAPTER_APPROVE_CONTENT
CHAPTER_REVISE_PRE_PUBLISH
CHAPTER_PUBLISH
CHAPTER_UNPUBLISH
CHAPTER_ARCHIVE
```

Generation:

```text
GENERATION_VIEW
GENERATION_START
GENERATION_RETRY
GENERATION_REGENERATE
GENERATION_REWRITE
GENERATION_CANCEL
```

Canon:

```text
CANON_VIEW
CANON_HISTORY_VIEW
STORY_MEMORY_VIEW
CANON_DATA_REPAIR_REQUEST
CANON_DATA_REPAIR_APPLY
```

Creative Decisions:

```text
CREATIVE_DECISION_VIEW
CREATIVE_DECISION_RESOLVE
CREATIVE_DECISION_POSTPONE
CREATIVE_DECISION_REJECT
```

Retcon:

```text
RETCON_VIEW
RETCON_REQUEST
RETCON_APPROVE
RETCON_APPLY
RETCON_CANCEL
```

Audio:

```text
NARRATION_VIEW
NARRATION_GENERATE
NARRATION_EDIT
NARRATION_APPROVE
AUDIO_GENERATE
AUDIO_RETRY
AUDIO_REVIEW
AUDIO_ACTIVATE_VERSION
```

Platform:

```text
AUDIT_VIEW
USER_STATUS_MANAGE
ADMIN_ROLE_GRANT
ADMIN_ROLE_REVOKE
ADMIN_STATUS_MANAGE
SECURITY_EVENT_VIEW
```

There is intentionally no:

```text
CANON_DIRECT_EDIT
```

for Published Official history.

---

# 15. Story Ownership

V1:

```text
Story belongs to PLATFORM.
```

Not:

```text
Story belongs to creating Admin.
```

Therefore:

```text
created_by = provenance/audit
```

not authorization ownership.

Admin Story permissions are platform-scoped in V1.

---

# 16. Story State Model

Story Status:

```text
DRAFT
ACTIVE
COMPLETED
ARCHIVED
```

Story Visibility:

```text
PRIVATE
PUBLIC
```

Planning Mode:

```text
FINITE
OPEN_ENDED
```

Planning Phase:

```text
ONGOING
CLOSING
FINAL_ARC
COMPLETED
```

These are independent axes.

Example:

```text
status        = ACTIVE
visibility    = PRIVATE
planningMode  = OPEN_ENDED
planningPhase = ONGOING
```

is valid.

---

# 17. Story DRAFT

Story foundation is being prepared.

Admin may:

* edit Story metadata;
* generate/edit Story Bible;
* generate/edit Characters;
* generate/edit Ending;
* generate/edit initial Arcs;
* adjust mutable Workflow Settings.

Usually PRIVATE.

---

# 18. Story Activation

Transition:

```text
DRAFT
→ Activation Gate
→ ACTIVE
```

Minimum Activation Gate:

* StoryGenerationPolicy exists;
* current Story Bible exists;
* current Ending Plan exists;
* at least one valid initial Arc exists;
* required main Characters exist;
* Planning Mode configured;
* current StoryContentProfile exists.

Missing dependency rejects activation with actionable reason.

Activation is explicit Admin action.

Story Architect generation does not automatically activate Story.

---

# 19. Story ACTIVE

Story foundation is ready for Chapter production.

ACTIVE does not mean PUBLIC.

A Story can remain PRIVATE during internal production.

---

# 20. Story COMPLETED

Requires:

```text
Story Completion Review
+
Admin decision
```

Normal:

```text
Generate Next Chapter
```

is disabled.

Prefer:

```text
Create Sequel
```

instead of reopening.

Completed Story can remain:

```text
PUBLIC
```

and continue to support reading/listening/audio repairs.

---

# 21. Story ARCHIVED

Archive is a management state, not delete.

Archived Story:

* no normal generation;
* no new normal Canon editing;
* no normal new publishing;
* may remain PUBLIC.

Restore returns to previous valid operational state.

Example:

```text
COMPLETED
→ ARCHIVED
→ Restore
→ COMPLETED
```

not automatically ACTIVE.

---

# 22. Story Deletion

Normal deletion only for trivial DRAFT Story with no meaningful:

* Official Canon;
* Published Chapters;
* listening data;
* production history.

Established Story uses Archive.

Future production purge is a separate destructive workflow.

---

# 23. Story Visibility

```text
PRIVATE ↔ PUBLIC
```

PRIVATE hides Story from public catalog.

PRIVATE does not modify Chapter statuses.

Therefore:

```text
Story PRIVATE
Chapter 1 PUBLISHED
Chapter 2 PUBLISHED
```

is valid and still invisible publicly.

---

# 24. Story Public Gate

Story may become PUBLIC only if:

* Status ACTIVE or COMPLETED;
* at least one PUBLISHED Chapter;
* valid title;
* valid description;
* valid cover;
* at least one Genre;
* valid public Content Rating;
* required Content Warnings;
* public metadata safety gate passes;
* no platform blocker.

Publishing Chapter 1 does **not** automatically make Story PUBLIC.

---

# 25. StoryGenerationPolicy

Immutable Story creation contract.

Resolution:

```text
Code Default
→ ENV Override
→ Create Story Advanced Override
→ Immutable Story Snapshot
```

Fields include:

* minimum Chapter audio duration;
* target Chapter audio duration;
* content origin;
* language;
* narration language;
* policy version.

Defaults:

```text
Minimum audio duration = 20 minutes
Target audio duration  = 30 minutes
```

ENV changes only affect future Stories.

No normal Update endpoint.

---

# 26. StoryWorkflowSettings

Mutable.

Includes:

* batch generation size;
* preferred Text AI provider/model;
* preferred TTS provider;
* preferred narrator voice;
* auto AI review;
* pause before TTS;
* planning horizon;
* creative autonomy;
* provider fallback policy.

Batch size:

```text
1 / 3 / 5 / 10 / custom
```

Creative autonomy baseline:

```text
BALANCED
```

Possible future values:

```text
CONSERVATIVE
BALANCED
EXPRESSIVE
```

Changing workflow settings affects future work only.

Historical GenerationRuns retain actual config/provider/model used.

---

# 27. StoryContentProfile

Separate from GenerationPolicy and WorkflowSettings.

It is:

```text
VERSIONED
CONTROLLED
```

Contains Story-specific content boundaries such as:

* maturity target;
* allowed themes;
* disallowed themes;
* violence/gore level;
* language limits;
* romance/sensual limits;
* Story-specific content constraints.

A Story Content Profile cannot weaken global Platform hard restrictions.

Profile revisions do not rewrite historical Runs or Chapters automatically.

---

# 28. Story Planning Modes

## FINITE

AI analyzes Story Idea and proposes scope options such as:

```text
Option A:
120–150 Chapters
6 Arcs

Option B:
180–220 Chapters
9 Arcs

Option C:
250–300 Chapters
12 Arcs

Custom
```

Chapter count remains planning range rather than hard lock.

## OPEN_ENDED

No locked total count.

Still requires:

* long-term destination;
* current planned Ending;
* current Arc;
* next direction.

Not:

```text
AI writes forever without destination.
```

---

# 29. Planning Phases for Open-ended Story

```text
ONGOING
→ CLOSING
→ FINAL_ARC
→ COMPLETED
```

In FINAL_ARC:

* avoid unnecessary major new threads;
* major new thread requires warning/CreativeDecision;
* focus on resolving promised arcs/threads/character journeys.

---

# 30. Story Canon Mental Model

```text
CANON
├── Story Bible
├── Ending Plan
├── Story Arcs
├── Character Profiles
├── Character Current States
├── Story Facts
├── Plot Threads
├── Approved Creative Decisions
└── World State
```

Full Chapter prose archive is always retained.

---

# 31. Story Bible

First-class and versioned.

Contains relatively stable:

* premise;
* world;
* rules;
* tone;
* main plot;
* constraints;
* writing rules.

It is not a dump of all dynamic Story memory.

Semantic changes after canonical production create a new version and require impact validation.

World Rule changes may require CreativeDecision or Retcon depending on impact/history.

---

# 32. Characters

Character is first-class.

Static Profile:

* canonical name;
* aliases;
* appearance;
* personality;
* background;
* motivation;
* fear;
* strength;
* weakness;
* speech style.

Dynamic CharacterState:

* location;
* health/condition;
* emotion;
* inventory;
* knowledge;
* relationships;
* goals;
* abilities;
* alive/dead/missing.

Character state is versioned against Canon.

---

# 33. StoryArc

First-class.

Contains:

* objective;
* conflict;
* key events;
* revelations;
* ending conditions;
* character development;
* expected Chapter range.

Current Arc is detailed.

Far future arcs may remain high-level.

---

# 34. Ending Plan

Always exists, including OPEN_ENDED Stories.

Versioned.

OPEN_ENDED means flexible path, not no ending.

Ending change is controlled by:

```text
CreativeDecision
→ Impact Analysis
→ Admin Approval
→ New Ending Version
```

---

# 35. ChapterPlan

First-class planning artifact before prose.

Contains:

* objective;
* opening;
* scenes;
* relevant Characters;
* required Facts;
* planned Facts;
* PlotThread actions;
* cliffhanger/ending;
* duration budget.

Near planning horizon is detailed.

Far future is abstract.

---

# 36. Chapter State Machine

```text
DRAFT
→ CONTENT_REVIEW
→ PRODUCTION
→ READY
→ PUBLISHED
→ ARCHIVED
```

---

# 37. Chapter DRAFT

Content not yet accepted into canonical production.

Allowed:

* edit plan;
* regenerate;
* rewrite;
* Admin edit;
* AI write;
* title/scene changes.

Cannot mutate Official Canon.

---

# 38. Chapter CONTENT_REVIEW

Means:

> Current content has not completed successful Official Canon Commit.

Typical flow:

```text
Writer
→ Duration Analysis
→ Continuity Review
→ Quality Review
→ Safety Review
→ Rewrite
→ Admin Review
```

Admin actions:

* Approve Content;
* Edit;
* Rewrite with Feedback;
* Regenerate;
* Return to Planning;
* Reject.

If content is approved but Memory Extraction/Canon Validation is retrying, Chapter can remain CONTENT_REVIEW.

---

# 39. Content Revision

Every meaningful Chapter prose change creates a revision.

Examples:

```text
Revision 5 AI generated
Revision 6 Admin edited
Revision 7 AI rewrite
```

Important rule:

```text
Approval applies to exact revision.
```

Approval of Revision 6 does not approve Revision 7.

---

# 40. Content Approval

Admin approval means:

> The exact prose revision is accepted.

It does **not** mean:

* Published;
* audio ready;
* Chapter permanently approved forever.

Flow after approval:

```text
Content Approval
→ Memory Extraction
→ Canon Change Set
→ Canon Validation
→ Official Canon Commit
```

---

# 41. Memory Extraction Rule

Canonical Memory Extraction only runs on:

```text
Final Admin-approved Content Revision
```

Not on:

* draft;
* old review candidate;
* pre-admin AI output.

---

# 42. Canon Commit Gate

Requires:

* approved content revision;
* Memory Extraction succeeded;
* Canon Change Set valid;
* base Canon still valid;
* no unresolved blocking CreativeDecision;
* previous required Chapter is Official;
* Canon sequence remains contiguous;
* no hard Canon conflict;
* Content Safety is acceptable.

---

# 43. Chapter PRODUCTION

After successful Official Canon Commit:

```text
CONTENT_REVIEW
→ PRODUCTION
```

Canonical prose is committed.

Production now covers:

* Narration;
* Narration Review;
* TTS;
* FFmpeg;
* Audio Quality.

Technical job failure does not create Chapter statuses such as:

```text
TTS_FAILED
```

Chapter remains PRODUCTION while job state describes failure.

---

# 44. Chapter READY

Means:

> Chapter is publishable right now.

Requires:

* current Content Revision approved;
* Official Canon committed;
* current Narration valid;
* current active AudioAsset READY;
* all HARD gates pass;
* every OVERRIDABLE violation resolved or explicitly overridden.

READY is intentionally strong.

---

# 45. Chapter PUBLISHED

Only explicit Admin Publish can transition:

```text
READY → PUBLISHED
```

No Auto-Publish in V1.

Listener sees Chapter only when:

```text
Chapter = PUBLISHED
AND
Story = PUBLIC
```

---

# 46. Chapter Numbering

Normal canonical Chapter numbers are contiguous.

Unique:

```text
(story_id, chapter_number)
```

Rules:

```text
Chapter N Official
requires
Chapter N-1 Official

Chapter N Published
requires
Chapter N-1 Published
```

except Chapter 1.

Official/Published Chapter number becomes immutable.

ChapterPlans may exist ahead of Canon.

---

# 47. Artifact Versioning

Version/revision history must exist for:

* Story Bible;
* Ending Plan;
* Arc;
* Character Profile;
* Character State;
* Chapter Plan;
* Chapter Content;
* Narration;
* Audio;
* Story Content Profile;
* Canon.

Never silently overwrite historical artifacts.

---

# 48. Artifact Lineage

Derived artifacts must know exact source.

Example:

```text
ChapterPlan v3
      ↓
Content Revision 8
      ↓
Narration Revision 4
      ↓
Audio Version 3
```

If Content Revision changes:

```text
old Narration = STALE
old Audio     = STALE
```

as appropriate.

---

# 49. Full AI Pipeline

```text
Story Idea
→ Story Architect
→ Story Bible + Characters + Ending + high-level Arcs
→ Arc Planner
→ Chapter Planner
→ Story Writer
→ Content Gap / Duration Analyzer
→ Continuity Reviewer
→ Writing Quality Reviewer
→ Content Safety / Classification Reviewer
→ Rewriter
→ Admin Content Review
→ Memory Extractor
→ Canon Validation
→ Official Canon Commit
→ Narration Director
→ Narration Review
→ TTS
→ Audio Quality Gate
→ READY
→ Admin Publish
```

Reviewers report issues.

Rewriter produces new candidate.

AI Review does not silently mutate content.

Some reviewers may run in parallel against the same frozen revision/context.

---

# 50. Duration Policy

Default:

```text
Minimum = 20 minutes
Target  = 30 minutes
```

Target is not an exact mandatory length.

Validation levels:

```text
Chapter Plan estimate
→ text estimate
→ narration estimate
→ actual TTS duration
```

Actual audio duration is final truth.

Examples:

```text
31m
PASS

24m
PASS + advisory below target

19m48s
OVERRIDABLE BLOCK
```

Under-min override requires:

* Admin;
* reason;
* audit.

No filler rule:

> Never pad duration through meaningless repetition, artificial slow narration, or irrelevant scenes.

---

# 51. Story Memory Engine

Model conversational memory is never Canon truth.

Application-controlled hierarchy:

```text
L0 Canon Constitution
L1 Current State
L2 Active Narrative
L3 Summaries
L4 Recent Detailed Context
L5 Historical Retrieval
```

## L0

* Story Bible;
* World Rules;
* Ending;
* major constraints.

## L1

* current Arc;
* Character state;
* relationship/world state.

## L2

* active Facts;
* active PlotThreads;
* goals;
* mysteries;
* setups/payoffs.

## L3

* Chapter summary;
* Arc summary;
* Story-so-far.

## L4

Recent detailed Chapters/context.

## L5

Historical retrieval.

---

# 52. Memory Retrieval

V1:

```text
PostgreSQL relational retrieval
```

No pgvector dependency required.

Future:

```text
pgvector semantic candidate retrieval
```

Semantic similarity may identify relevant history but never determine canonical truth.

---

# 53. Context Builder

Every important GenerationRun receives a Context Pack including only relevant data:

* StoryGenerationPolicy;
* StoryContentProfile;
* current Story Bible;
* Ending;
* current Arc;
* ChapterPlan;
* relevant Characters;
* CharacterStates;
* Facts;
* PlotThreads;
* recent context;
* historical retrieval;
* selected Creative Decisions;
* Admin-specific generation instruction.

---

# 54. ContextSnapshot

Immutable historical record.

Stores references/version information such as:

* Canon version;
* Story Bible version;
* Ending version;
* Arc version;
* Content Profile version;
* CharacterState versions;
* Facts;
* PlotThreads;
* historical retrieval hits;
* prompt version;
* workflow version;
* provider;
* model;
* Admin instruction.

Retcon must never rewrite old ContextSnapshot.

---

# 55. StoryFact

Statuses:

```text
ACTIVE
SUPERSEDED
INVALIDATED
```

Facts preserve provenance:

* source Chapter;
* content revision;
* GenerationRun;
* Canon version.

Do not semantic-delete old Facts.

---

# 56. PlotThread

Statuses:

```text
OPEN
ADVANCING
RESOLVED
ABANDONED
```

Planner decides which thread to:

* advance;
* resolve;
* intentionally leave untouched;
* open.

Major thread inactivity may create warnings.

---

# 57. Canon Lifecycle

```text
DRAFT
→ PROVISIONAL
→ OFFICIAL
```

`PUBLISHED` is not Canon status.

Normal Canon Commit:

```text
Approved Content
→ Memory Extractor
→ Validated Change Set
→ Canon Commit
```

Official Canon sequence is contiguous.

---

# 58. Batch Generation & Provisional Canon

Batch size is mutable.

Generation order remains sequential.

Example:

```text
Official Canon v100
→ Chapter 101
→ Provisional v101
→ Chapter 102
→ Provisional v102
→ Chapter 103
```

Dependent Chapters are not parallelized.

If Chapter 103 fails:

```text
101 stays valid
102 stays valid
103 blocked
104+ do not start
```

Batch is not a giant transaction.

Admin may:

* retry 103;
* edit 103;
* regenerate 103;
* cancel remaining batch.

If earlier Chapter changes, affected downstream candidates become STALE.

Provisional Canon never serves public listener requests.

---

# 59. Creative Decision Severity

```text
MINOR
SIGNIFICANT
MAJOR
CRITICAL
```

MINOR:

AI usually handles autonomously.

SIGNIFICANT:

may be autonomous if Creative Contract allows.

MAJOR / CRITICAL:

require controlled decision.

System rules can escalate severity even if AI incorrectly labels it minor.

Example:

```text
main character death
→ CRITICAL
```

---

# 60. Creative Decision Status

```text
PROPOSED
→ ANALYZING
→ WAITING_FOR_ADMIN
→ SELECTED
→ APPLIED
```

Other:

```text
REJECTED
POSTPONED
SUPERSEDED
CANCELLED
```

Important:

```text
SELECTED != APPLIED
```

A future selected decision does not mutate current Canon state.

---

# 61. Creative Decision Blocking

```text
NON_BLOCKING

BLOCK_BEFORE_CHAPTER

BLOCK_BEFORE_CANON_COMMIT

BLOCK_IMMEDIATELY
```

---

# 62. Creative Decision Options

AI options contain:

* summary;
* what happens;
* benefits;
* risks;
* Canon impact;
* Character impact;
* Arc impact;
* Ending impact;
* new opportunities;
* threads opened/resolved;
* future complexity.

AI may recommend an option but cannot auto-select MAJOR/CRITICAL in V1.

Admin always has Custom option for major decisions.

Custom flow:

```text
Admin Intent
→ Impact Analysis
→ Canon Conflict Check
→ Future Planning Analysis
→ Admin Confirmation
```

---

# 63. Selected Future Decision

Example:

```text
Lan will die in Arc 5 finale.
```

Current Canon still says:

```text
Lan = alive
```

until the actual event occurs and Canon is committed.

Selected decisions may become planning constraints such as:

```text
Lan must survive until Arc 5 finale.
```

Before application, decision must revalidate against current Canon.

---

# 64. Applied Creative Decision

Applied historical decision cannot be edited.

If not Published yet:

```text
controlled pre-publish revision
```

may be used.

If Published:

```text
Retcon
```

is required to change it.

---

# 65. GenerationRun

Represents a high-level workflow intent.

Possible Run types:

```text
STORY_ARCHITECTURE
ARC_PLANNING
CHAPTER_PLANNING
CHAPTER_GENERATION
CHAPTER_REVIEW
CHAPTER_REWRITE
MEMORY_EXTRACTION
NARRATION_GENERATION
AUDIO_GENERATION
RETCON_ANALYSIS
ARC_COMPLETION_REVIEW
STORY_COMPLETION_REVIEW
```

Statuses:

```text
PENDING
RUNNING
WAITING
SUCCEEDED
FAILED
CANCELLED
STALE
```

---

# 66. GenerationRun WAITING

WAITING means system is not broken.

Possible reasons:

```text
ADMIN_REVIEW
CREATIVE_DECISION
DURATION_OVERRIDE
ARC_DECISION
DEPENDENCY
PROVIDER_RECOVERY
```

Human WAITING never becomes FAILED merely due to elapsed time.

---

# 67. GenerationJob

Smaller execution unit inside GenerationRun.

Statuses:

```text
PENDING
RUNNING
SUCCEEDED
FAILED
CANCELLED
STALE
```

One Job may have multiple attempts.

---

# 68. Failure Classification

```text
TRANSIENT
PERMANENT
VALIDATION
STALE_INPUT
POLICY_BLOCK
```

Policy block origin:

```text
PLATFORM
STORY_PROFILE
PROVIDER
```

---

# 69. Retry Policy

Retry is finite.

Recommended operational defaults:

### Text AI transient

```text
3 attempts
~2s → ~5s → ~15s + jitter
```

Honor provider `Retry-After` where available.

### Malformed structured AI output

```text
1 repair attempt
→ 1 full regenerate
→ FAIL
```

### TTS segment

```text
3 attempts
```

### Object-storage upload

```text
5 attempts
```

### FFmpeg recoverable failure

```text
2 attempts
```

### Permanent credential/config failure

```text
No automatic retry
```

### Platform policy block

```text
No retry
No provider fallback
No Admin override
```

Provider policy refusal may use approved fallback only when Platform and Story policy both allow content.

---

# 70. PostgreSQL Job Queue

V1 uses PostgreSQL-backed jobs.

Worker claims jobs transactionally using:

```sql
FOR UPDATE SKIP LOCKED
```

Jobs have concepts:

```text
available_at
priority
locked_by
lock_expires_at
attempt_count
max_attempts
```

Recommended defaults:

```text
Worker heartbeat       ~30 seconds
Lease                  ~5 minutes
Text AI timeout        ~10 minutes
TTS segment timeout    ~5 minutes
FFmpeg timeout         ~15 minutes
Storage upload timeout ~10 minutes
```

All configurable.

Worker death must not leave Job RUNNING forever.

Expired lease makes Job reclaimable.

---

# 71. Idempotency

High-cost/high-impact actions accept:

```text
Idempotency-Key
```

Examples:

* start generation;
* retry;
* audio generation;
* approval;
* publish;
* unpublish;
* Retcon apply.

Compatible active duplicate generation should be reused/rejected rather than silently creating competing Runs.

---

# 72. Retry vs Regenerate vs Rewrite

## Retry

Same logical intended result after technical failure.

## Regenerate

Create a new candidate result.

## Rewrite

Modify existing candidate according to issues/feedback.

These must remain separate in UI/API/domain vocabulary.

---

# 73. Cancellation

Cancel:

* stops future jobs;
* best-effort interrupts running provider request;
* late result cannot auto-apply.

Before Canon Commit:

```text
Official Canon unchanged.
```

After Canon Commit:

```text
Canon remains Official.
```

Cancelling TTS does not rollback canonical prose.

---

# 74. Staleness

Important jobs bind input versions:

* Content Revision;
* Canon Version;
* Story Bible Version;
* Ending Version;
* Arc Version;
* Content Profile Version;
* Prompt Version;
* Workflow Version.

If current state changes before result applies:

```text
Result = STALE
```

Never overwrite current state.

Stale output may be retained temporarily for comparison/debug.

---

# 75. AI Output Validation

Never:

```text
AI Response
→ DB Apply
```

Always:

```text
AI Response
→ Parse
→ Schema Validation
→ Semantic Validation
→ Business Validation
→ Accept / Reject
```

Example Memory Extractor validation:

* Character belongs to Story?
* reference exists?
* Canon version current?
* state value valid?
* world rule violated?
* immutable rule violated?

---

# 76. Provider Abstraction

Provider SDK does not leak into domain logic.

Interfaces conceptually:

```text
TextAIProvider
TTSProvider
ObjectStorage
EmailProvider
```

Text AI default direction:

```text
Gemini provider
```

but actual model ID is ENV/configured, not compile-time hardcoded.

TTS primary direction:

```text
Gemini TTS
```

with provider abstraction/fallback.

Actual provider/model/voice used is stored per Job/Attempt/Artifact.

---

# 77. Mock Providers

Development supports:

```text
AI_MODE=mock
TTS_MODE=mock
```

Mock mode must be deterministic enough to test:

* Story flow;
* jobs;
* Canon;
* admin UI;
* audio pipeline;
* publication;
* player;

without external quota.

Production rejects mock mode.

---

# 78. Narration

Story Content and Narration Script are separate.

V1 uses one main narrator per Story.

Tone may vary by scene/dialogue.

Narrator gender is **not hardcoded**.

Platform maintains allowlisted voice profiles.

StoryWorkflowSettings may choose one.

Do not clone/imitate a specific real person's voice without appropriate rights.

---

# 79. Audio Pipeline

```text
Approved Canonical Content
→ Narration Script
→ Segmenter
→ TTS Segments
→ FFmpeg
→ Audio Quality
→ Object Storage
→ AudioAsset READY
```

FFmpeg:

* concatenate;
* normalize;
* encode;
* validate duration.

Baseline final format:

```text
MP3
64–96 kbps
```

TTS uses segment-level retry.

---

# 80. Audio Versioning

Regenerating audio creates a new version.

Never overwrite active media in place.

```text
Audio v1 active
→ generate v2
→ validate
→ atomic promotion
→ v2 active
```

If Story Content did not change, this is not Retcon.

---

# 81. Object Storage

Production:

```text
Cloudflare R2
```

Local:

```text
MinIO
```

All core media stored in **private bucket**.

Browser gets signed GET URLs.

Backend does not stream/proxy MP3.

Object keys use IDs:

```text
stories/{storyId}/chapters/{chapterId}/audio/v{version}/chapter.mp3
```

Never use raw Story titles as object identity.

Default signed audio URL TTL:

```text
2 hours
```

configurable.

Existing signed URL may remain valid until expiry after unpublish/private.

Future strict immediate takedown may add edge/CDN token revocation.

---

# 82. Story Cover

V1:

```text
Admin upload
```

or platform placeholder.

Cover stored in object storage.

AI-generated cover is future, not required for V1.

Public cover and metadata must pass content governance.

---

# 83. Listener Progress

Distinguish:

```text
Playback Position
Chapter Completion
Current Resume
Furthest Story Progress
```

They are not the same concept.

---

# 84. Guest Progress

Guest:

```text
local browser storage
```

No database identity.

Loss of browser data/device means Guest progress can be lost.

---

# 85. User Progress

Authenticated User:

```text
server-backed
```

Local state may be cache, but server is source of truth.

Progress writes occur on:

* periodic heartbeat;
* pause;
* seek;
* chapter change;
* app background/close;
* playback end.

Not every second.

---

# 86. Chapter Completion

Baseline:

```text
Playback ended
OR
position >= 95% of valid audio duration
→ COMPLETED
```

Replay does not erase historical completion.

---

# 87. Continue Listening

Uses latest valid listening intent, not:

```text
MAX(chapter_number)
```

Example:

User completed Chapter 40 but intentionally replays Chapter 5 and stops at 10m.

Then:

```text
Current Resume = Chapter 5 @ 10m
Furthest Progress = Chapter 40
```

Both remain true.

---

# 88. Caught Up

If Story ACTIVE and User completed latest Published Chapter:

```text
CAUGHT_UP
```

If Story COMPLETED and final Chapter completed:

```text
Story finished by listener
```

If a new Chapter publishes after User was caught up, Continue Listening can start at the new Chapter.

---

# 89. Multi-device Progress

Playback update includes:

* playback session ID;
* base progress version;
* position;
* event type;
* audio version.

Stale device update must not silently overwrite newer progress.

But intentional seek backwards must still be allowed.

Therefore position cannot simply be monotonic.

Use optimistic concurrency/version semantics.

---

# 90. Guest → User Merge

If no server progress:

```text
auto import local progress
```

If meaningful conflict:

* preserve server completion history;
* do not silently destroy either side;
* UI may let User choose current resume position.

Merge must be idempotent.

---

# 91. Favorite

Favorite applies to Story, not Chapter.

Unique:

```text
(user_id, story_id)
```

Idempotent.

PRIVATE/ARCHIVED Story does not delete Favorite.

---

# 92. Story/Chapter Availability Changes

Making Story PRIVATE does not delete:

* Favorite;
* Progress;
* Completion.

Unpublishing a Chapter does not delete progress.

Archived + PUBLIC Story remains playable.

Runtime audio outage does not automatically unpublish Chapter.

---

# 93. Retcon & Listener Progress

True Retcon preserves fact that User listened to historical version.

Listener impact categories:

```text
NO_RELISTEN_NEEDED
RELISTEN_RECOMMENDED
RELISTEN_REQUIRED
```

Do not silently set historical `completed=false`.

---

# 94. Search & Public Catalog

Public search only exposes eligible PUBLIC Stories.

V1 search:

* title;
* description;
* Genre.

Filters:

* Genre;
* Content Rating;
* Ongoing/Completed.

Sort:

* Recently Updated;
* New;
* Title.

Implementation:

```text
PostgreSQL Full Text Search
+
pg_trgm
```

"New" uses public launch date, not internal created_at.

"Recently Updated" uses public Chapter release, not AI/Draft activity.

Do not leak:

* internal ending;
* future exact Chapter plan;
* Arc internals;
* Canon;
* Creative Decisions.

---

# 95. Content Rating

Internal product classification:

```text
GENERAL
TEEN
MATURE
```

Not advertised as a legal age-rating system.

MATURE Story displays clear content warning.

V1 uses acknowledgment/interstitial on first access per account/device.

No identity-based age verification in V1.

---

# 96. Platform Content Governance

Three layers:

```text
Platform Content Policy
→ Story Content Profile
→ Provider Policy
```

Platform hard block cannot be weakened by Story/Admin/provider fallback.

Provider refusal is not automatically Platform violation.

---

# 97. V1 Fiction Content Scope

May support, depending on context/rating:

* violence;
* horror;
* death;
* crime;
* psychological themes;
* abuse themes;
* self-harm/suicide themes in narrative context;
* drugs/addiction depiction;
* discrimination/extremism depiction;
* romance;
* non-explicit sensual content.

V1 does **not** support normal explicit pornographic content.

Sexual content involving minors is HARD BLOCK.

Harmful serious wrongdoing instruction is blocked/restricted.

Depiction is not the same as endorsement.

---

# 98. Content Safety Reviewer

Separate from:

```text
Continuity Reviewer
Writing Quality Reviewer
```

Outcomes:

```text
PASS
PASS_WITH_WARNINGS
REVIEW_REQUIRED
BLOCKED
```

Safety review is revision-bound.

If content revision changes, old safety review becomes stale.

BLOCKED content cannot:

* be approved;
* Canon Commit;
* become READY;
* Publish.

Admin manual content follows same policy.

---

# 99. Content Warnings

Examples:

```text
VIOLENCE
GRAPHIC_VIOLENCE
HORROR
PSYCHOLOGICAL_HORROR
DEATH
SUICIDE_SELF_HARM
ABUSE
DOMESTIC_ABUSE
SEXUAL_THEMES
STRONG_LANGUAGE
DRUGS
ADDICTION
DISCRIMINATION
EXTREMISM
CHILD_ENDANGERMENT
```

Public warnings are spoiler-light.

Story public rating cannot materially understate Published Chapter content.

---

# 100. Quality Governance

Do not use one magical numeric score.

Review dimensions may include:

* continuity;
* coherence;
* pacing;
* dialogue;
* repetition;
* style consistency;
* characterization;
* scene purpose;
* opening;
* ending;
* content depth.

Quality warnings are typically ADVISORY or OVERRIDABLE.

Hard gate is reserved for actual invariant/safety/structural failures.

---

# 101. Real Persons / Copyright / Likeness

V1 prioritizes fictional characters.

Do not build the normal product workflow around harmful fictionalization of identifiable living people.

AI-generated Story should target:

```text
original fiction
```

or material with usage rights.

Do not build unauthorized franchise continuation workflows.

Prefer descriptive style attributes:

```text
dark gothic horror
slow-burn mystery
Vietnamese folklore atmosphere
```

rather than direct imitation of a living author.

Voice/likeness assets must be properly authorized.

Future imported human content requires rights attestation/provenance.

Imported content remains subject to safety policy.

---

# 102. Prompt Injection Boundary

Any imported/external/user Story text is:

```text
UNTRUSTED CONTENT
```

not system instruction.

Story text cannot trigger:

* DB mutation;
* publishing;
* permissions;
* arbitrary network access;
* tool invocation;

merely because prose contains instructions.

Tools are controlled by backend workflow.

---

# 103. Quality Gate Levels

```text
HARD
OVERRIDABLE
ADVISORY
```

## HARD

Cannot bypass.

Examples:

* Platform-prohibited content;
* missing approval;
* missing Official Canon;
* invalid sequence;
* blocking CreativeDecision;
* missing/invalid current audio;
* stale lineage;
* known rights violation.

## OVERRIDABLE

Can proceed with Admin + reason + audit.

Examples:

* actual duration below normal 20m minimum;
* selected quality warnings;
* allowed ambiguous classification cases.

## ADVISORY

Does not block.

Examples:

* below 30m target;
* minor repetition;
* minor PlotThread inactivity.

---

# 104. Publish

Transition:

```text
READY
→ Final Publish Validation
→ Admin Publish
→ PUBLISHED
```

Final validation re-checks:

* exact current revisions;
* current audio;
* Story status;
* previous Chapter Published;
* no Retcon maintenance block;
* no new HARD blocker.

Publish is idempotent.

No Auto-Publish in V1.

---

# 105. Unpublish

Conceptually:

```text
PUBLISHED → READY
```

Official Canon remains.

If middle Chapter unpublish would create gap:

```text
Cancel
OR
Unpublish this Chapter + all subsequent Published Chapters
```

with impact confirmation.

Unpublish is not Retcon.

Unpublish cannot be used as loophole to edit previously Published canonical prose.

---

# 106. Canon Data Repair

Used when Published prose is correct but structured memory is wrong.

Example:

Published text:

```text
Minh lost the key.
```

Extracted Fact incorrectly says:

```text
Minh still has the key.
```

Use:

```text
Canon Data Repair
```

not Retcon.

Flow:

```text
Published Evidence
→ detect wrong extraction
→ re-extract/correct structured state
→ impact analysis
→ new Canon revision
```

---

# 107. True Retcon

Used when actual Published story history/prose changes.

Published canonical content has no normal Save button.

Use:

```text
Request Canon Revision
```

Retcon requires explicit reason.

---

# 108. Retcon Lifecycle

```text
DRAFT
→ ANALYZING
→ WAITING_FOR_ADMIN
→ APPROVED
→ REPAIRING
→ READY_TO_APPLY
→ APPLYING
→ APPLIED
```

Alternative:

```text
REJECTED
CANCELLED
FAILED
SUPERSEDED
```

Impact scope:

```text
LOCAL
PROPAGATING
STRUCTURAL
```

---

# 109. Retcon Impact Analysis

Analyze:

* affected Chapters;
* StoryFacts;
* CharacterStates;
* PlotThreads;
* Arcs;
* Ending;
* CreativeDecisions;
* ChapterPlans;
* Drafts;
* Provisional content;
* Published downstream;
* Listener impact.

Classify affected items:

```text
DIRECTLY_AFFECTED
POTENTIALLY_AFFECTED
STALE
UNAFFECTED
REQUIRES_REVISION
REQUIRES_REVIEW
```

Do not blindly regenerate every downstream Chapter.

---

# 110. Retcon Repair Plan

AI/System proposes steps.

Admin approves/changes scope.

Examples:

* revise Chapter;
* re-extract memory;
* supersede Fact;
* recompute CharacterState;
* review downstream Chapter;
* mark future Plans stale;
* regenerate affected Drafts;
* re-run Continuity.

Do not auto-rewrite 150 Chapters without Admin understanding impact.

---

# 111. Retcon Workspace

Approved Retcon repair must not expose partially repaired public Canon.

Current public line remains active while repair candidate revisions are prepared.

Generation after affected point may be blocked once Retcon is approved.

Before `READY_TO_APPLY`:

* required revisions complete;
* memory reconciled;
* facts/states/threads reconciled;
* continuity passes;
* replacement audio ready;
* listener impact known;
* no blocking repair task.

---

# 112. Retcon Apply

L4 critical operation.

Requires:

* `RETCON_APPLY`;
* Privileged Auth;
* Recent Auth;
* explicit impact confirmation;
* READY_TO_APPLY.

Promotion updates active content/audio pointers and Canon head in a coherent DB transaction.

Readers see:

```text
old coherent state
OR
new coherent state
```

not half-retconned state.

Historical:

* old revisions;
* old audio;
* old ContextSnapshots;
* old Audit;
* old applied Decisions;

remain historical evidence.

---

# 113. Admin Control Center

Admin UI is a production control center.

Global dashboard:

```text
Needs Attention
Generation Activity
Ready to Publish
Recently Updated Stories
Production Health
```

Attention priority:

```text
BLOCKING
FAILED
WAITING_FOR_HUMAN
WARNING
INFORMATIONAL
```

Each attention item explains:

* what happened;
* why;
* impact;
* available action.

---

# 114. Story Admin Workspace

Logical sections:

```text
Overview

Planning
├── Story Bible
├── Ending Plan
├── Arcs
└── Chapter Plans

Characters

Canon & Memory
├── Facts
├── Plot Threads
├── Character States
└── Canon History

Chapters

Creative Decisions

Generation

Narration & Audio

Publication

Audit

Settings
```

---

# 115. Story Overview

Must surface:

* Story Status;
* Visibility;
* Planning Mode;
* Planning Phase;
* Current Arc;
* current Official Canon;
* latest Official Chapter;
* latest Published Chapter;
* active Generation;
* blocking issues;
* open major PlotThreads;
* current Ending version;
* next recommended action.

Next Recommended Action is advisory, not auto-execution.

---

# 116. Story Creation UX

```text
Basic Idea
→ Creation Contract
→ Planning Mode
→ Story Architect
→ Architecture Proposal
→ Admin Review/Edit
→ Foundation Consistency Review
→ Activate
```

Story Architect proposal is reviewable by:

* premise;
* tone;
* world rules;
* main Characters;
* Ending;
* Arc structure;
* initial PlotThreads;
* risks/assumptions.

Admin can regenerate individual sections when dependencies allow.

---

# 117. Admin Chapter Workspace

Sections:

```text
Plan
Content
AI Reviews
Canon Impact
Narration
Audio
Publication
History
```

CONTENT_REVIEW screen shows:

* current prose;
* Continuity report;
* Quality report;
* Safety report;
* duration;
* warnings;
* CreativeDecision blockers;
* revision history.

---

# 118. Manual Admin Edit

Manual edit is a first-class revision.

Example provenance:

```text
source = ADMIN_EDIT
based_on = Revision 7
created_by = Admin A
```

It can invalidate downstream artifacts.

Manual edits are not "outside the system".

---

# 119. AI-assisted Edit

Highlight/rewrite operation creates:

```text
GenerationRun
→ candidate revision
```

Stores:

* input revision;
* prompt version;
* provider/model;
* Admin instruction.

AI suggestion never silently replaces current revision.

---

# 120. Narration Admin UX

If `pause_before_tts=true`:

```text
Narration Generated
→ Admin Review
→ Approve
→ TTS
```

If false and system review passes:

```text
Narration Generated
→ TTS
```

Admin may generate short TTS preview before full Chapter.

Preview is not current AudioAsset.

---

# 121. Admin Long-Story Navigation

Must remain usable for 500+ Chapters.

Features:

* cursor/pagination/virtualization;
* Chapter number search;
* filter by status;
* filter by Arc;
* filter stale;
* filter failed;
* filter needs attention;
* current production window;
* Arc-centric grouping.

Do not force Admin to scroll 500 rows.

---

# 122. Notification Strategy

V1 does not require push/email notifications.

In-app Attention Center is sufficient.

Events worth surfacing:

* Generation completed/failed;
* Content Review required;
* Creative Decision required;
* Arc Review required;
* Batch blocked;
* Audio ready;
* Retcon ready.

Notification is not Audit.

Notification answers:

> What needs attention now?

Audit answers:

> What happened historically?

---

# 123. Privacy & Data Minimization

AI provider receives only Story-production context necessary.

Never send to Story AI:

* password;
* refresh token;
* MFA secret;
* User Favorite data;
* User listening data;
* unrelated personal data.

TTS provider receives narration text/direction and voice parameters only.

---

# 124. Account Deletion Lifecycle

Baseline:

```text
ACTIVE
→ deletion request
→ DEACTIVATED immediately
→ 30-day grace period
→ purge/anonymization
```

After grace period remove/anonymize:

* email;
* password;
* MFA;
* sessions;
* verification/reset tokens;
* profile PII;
* Favorite;
* ListeningProgress;
* PlaybackSessions.

Historical business Audit/provenance can remain with actor anonymized where necessary.

Admin account deletion still obeys Last Active Admin Guard.

---

# 125. Retention Defaults

Configurable operational defaults:

```text
Temporary /tmp files                  <= 24h
Cancelled/stale candidates            30 days
Detailed technical JobAttempt data    90 days
Security events                       180 days
Canonical provenance/Audit            Story lifetime / controlled purge
Published active audio                while referenced
```

Never log secrets.

---

# 126. Architecture

V1:

```text
MODULAR MONOLITH
+
SEPARATE WORKER PROCESS
```

Backend logical modules:

```text
identity
catalog
story
planning
canon
memory
generation
governance
audio
listener
audit
platform
```

Each module conceptually:

```text
domain
application
ports
infrastructure
transport/http
```

Do not over-apply architecture ceremony to every struct.

---

# 127. Backend Design Patterns

Use deliberately:

```text
Repository

Unit of Work / Transaction Manager

Provider Adapter / Strategy

Policy / Specification for Gates

Explicit State Transition Services

Provider Registry / Factory
```

Avoid:

```text
God Service
Generic BaseRepository for everything
Generic universal workflow engine
```

Cross-module state mutation must go through application services/transaction boundaries.

---

# 128. Backend Technology

Baseline:

```text
Go
PostgreSQL
pgx
sqlc
SQL migrations
chi-style net/http router
slog JSON logging
OpenAPI 3.1
FFmpeg
```

Persistence choice:

```text
sqlc + pgx
```

not GORM as core persistence.

Reason:

* Canon/version/history needs explicit SQL;
* transaction control matters;
* query/index tuning matters;
* compile-time query typing is valuable.

---

# 129. Frontend Technology

```text
Vue 3
Vite
TypeScript
Vue Router
Pinia
TanStack Vue Query
Generated OpenAPI client/types
```

Pinia:

* auth/client state;
* player;
* UI state.

Vue Query:

* API/server state.

Do not duplicate entire backend collections into Pinia.

Single application contains:

```text
public UI
/admin
```

---

# 130. API Architecture

REST resources + explicit domain actions.

Prefix:

```text
/api/v1
```

Do not force business operations into generic CRUD.

Example error:

```json
{
  "error": {
    "code": "CHAPTER_NOT_READY",
    "message": "Chapter is not ready to publish.",
    "details": {}
  }
}
```

Stable domain error codes include:

```text
NOT_AUTHENTICATED
ACCOUNT_SUSPENDED
EMAIL_VERIFICATION_REQUIRED
MFA_REQUIRED
RECENT_AUTH_REQUIRED
PERMISSION_DENIED
RESOURCE_SCOPE_DENIED
BUSINESS_STATE_BLOCKED
HARD_GATE_BLOCKED
STALE_VERSION
CONFLICT
```

---

# 131. Public API

Core:

```text
GET  /api/v1/stories

GET  /api/v1/stories/{story}

GET  /api/v1/stories/{story}/chapters

GET  /api/v1/stories/{story}/chapters/{chapterNumber}

POST /api/v1/audio/{audioAssetId}/play-url
```

`play-url` checks:

* Story PUBLIC;
* Chapter PUBLISHED;
* Audio current + READY.

Then returns signed object-storage URL.

---

# 132. Auth API

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
POST /api/v1/auth/logout-all

POST /api/v1/auth/email/verify
POST /api/v1/auth/email/resend

POST /api/v1/auth/password/forgot
POST /api/v1/auth/password/reset
POST /api/v1/auth/password/change

POST /api/v1/auth/mfa/totp/setup
POST /api/v1/auth/mfa/totp/confirm
POST /api/v1/auth/mfa/totp/disable

POST /api/v1/auth/re-auth
```

---

# 133. User API

```text
GET   /api/v1/me
PATCH /api/v1/me

GET    /api/v1/me/favorites
PUT    /api/v1/me/favorites/{storyId}
DELETE /api/v1/me/favorites/{storyId}

GET /api/v1/me/continue-listening

GET /api/v1/me/listening-progress/{chapterId}
PUT /api/v1/me/listening-progress/{chapterId}

POST /api/v1/me/playback-sessions
POST /api/v1/me/playback-sessions/{id}/end

GET    /api/v1/me/sessions
DELETE /api/v1/me/sessions/{id}
DELETE /api/v1/me/sessions

POST /api/v1/me/account/deactivate
POST /api/v1/me/account/deletion-request
```

---

# 134. Admin Story API

```text
POST  /api/v1/admin/stories
GET   /api/v1/admin/stories
GET   /api/v1/admin/stories/{id}

PATCH /api/v1/admin/stories/{id}/metadata

POST /api/v1/admin/stories/{id}/activate
POST /api/v1/admin/stories/{id}/archive
POST /api/v1/admin/stories/{id}/restore

POST /api/v1/admin/stories/{id}/make-public
POST /api/v1/admin/stories/{id}/make-private

GET /api/v1/admin/stories/{id}/workflow-settings
PUT /api/v1/admin/stories/{id}/workflow-settings
```

No normal update API for StoryGenerationPolicy.

---

# 135. Admin Chapter Actions

```text
POST /api/v1/admin/stories/{storyId}/chapters

GET  /api/v1/admin/chapters/{chapterId}

POST /api/v1/admin/chapters/{chapterId}/generate

POST /api/v1/admin/chapters/{chapterId}/rewrite

POST /api/v1/admin/chapters/{chapterId}/regenerate

POST /api/v1/admin/chapters/{chapterId}/revisions/{revisionId}/approve

POST /api/v1/admin/chapters/{chapterId}/revise-before-publish

POST /api/v1/admin/chapters/{chapterId}/narration/generate

POST /api/v1/admin/chapters/{chapterId}/audio/generate

POST /api/v1/admin/chapters/{chapterId}/publish

POST /api/v1/admin/chapters/{chapterId}/unpublish
```

---

# 136. Generation API

```text
GET /api/v1/admin/generation-runs

GET /api/v1/admin/generation-runs/{id}

POST /api/v1/admin/generation-runs/{id}/cancel

POST /api/v1/admin/generation-jobs/{id}/retry
```

Regenerate produces new Run/candidate, not mutation of old failed Run.

---

# 137. CreativeDecision API

```text
GET /api/v1/admin/creative-decisions

GET /api/v1/admin/creative-decisions/{id}

POST /api/v1/admin/creative-decisions/{id}/select

POST /api/v1/admin/creative-decisions/{id}/custom

POST /api/v1/admin/creative-decisions/{id}/reject

POST /api/v1/admin/creative-decisions/{id}/postpone
```

---

# 138. Retcon API

```text
POST /api/v1/admin/retcons

GET /api/v1/admin/retcons

GET /api/v1/admin/retcons/{id}

POST /api/v1/admin/retcons/{id}/analyze

POST /api/v1/admin/retcons/{id}/approve

POST /api/v1/admin/retcons/{id}/cancel

POST /api/v1/admin/retcons/{id}/apply
```

Apply requires Recent Auth.

---

# 139. Pagination

Use cursor/keyset pagination for large lists.

Especially:

* public catalog;
* Story list;
* Chapter list;
* Audit;
* GenerationRuns.

Long Chapter list also supports:

* Chapter number range;
* Arc;
* status;
* stale;
* failed;
* needs attention.

---

# 140. Optimistic Concurrency

Mutable revision resources use version/revision conflict detection.

Example:

```text
Admin A opens Revision 7
Admin B opens Revision 7

A saves Revision 8

B attempts old save
→ 409 CONFLICT
```

No silent last-write-wins.

---

# 141. PostgreSQL General Rules

* IDs: UUIDv7 generated application-side.
* Times: `TIMESTAMPTZ` UTC.
* Business status: `TEXT + CHECK`.
* JSONB for complex AI structured artifact.
* Normalize fields needed for relation/filter/index.
* No generic delete flag.
* Foreign keys explicit.
* Immutable historical entities do not get meaningless `updated_at`.
* Historical Canon/Audit should not cascade-delete.

---

# 142. Identity Tables

## users

```text
id UUID PK
email CITEXT UNIQUE
password_hash TEXT
display_name TEXT
status TEXT
email_verified_at TIMESTAMPTZ NULL
created_at
updated_at
deactivated_at NULL
```

## roles

```text
id
code UNIQUE
```

Seed:

```text
USER
ADMIN
```

## permissions

```text
id
code UNIQUE
```

## role_permissions

```text
role_id
permission_id
PK(role_id, permission_id)
```

## user_roles

```text
user_id
role_id
granted_by
granted_at
UNIQUE(user_id, role_id)
```

---

# 143. Auth Tables

## user_sessions

```text
id
user_id
refresh_token_hash
created_at
last_used_at
expires_at
revoked_at
mfa_verified_at
recent_auth_at
user_agent_summary
safe_ip_metadata
```

## user_mfa_methods

```text
id
user_id
type
encrypted_secret
confirmed_at
disabled_at
```

## user_mfa_recovery_codes

```text
id
user_id
code_hash
used_at
```

## email_verification_tokens

Hashed, one-time, expiry.

## password_reset_tokens

Hashed, one-time, expiry.

## account_deletion_requests

```text
id
user_id
requested_at
purge_after
cancelled_at
completed_at
```

---

# 144. Story Tables

## stories

```text
id
slug UNIQUE
title
description

status
visibility
planning_mode
planning_phase

public_rating
public_warnings[]

cover_asset_id

current_story_bible_version_id
current_ending_plan_version_id
current_content_profile_version_id
current_official_canon_version_id

public_since
last_published_at
status_before_archive

created_by
created_at
updated_at
archived_at
```

## genres

```text
id
slug UNIQUE
name UNIQUE
```

## story_genres

```text
story_id
genre_id
PK
```

## story_assets

```text
id
story_id
type
storage_key
mime_type
size_bytes
checksum
rights_status
status
created_by
created_at
```

---

# 145. Story Contract Tables

## story_generation_policies

One immutable row/Story.

```text
story_id PK
minimum_audio_duration_sec
target_audio_duration_sec
content_origin
language
narration_language
policy_version
created_by
created_at
```

## story_workflow_settings

```text
story_id PK
batch_generation_size
creative_autonomy
preferred_text_provider
preferred_text_model
preferred_tts_provider
preferred_voice_id
pause_before_tts
auto_ai_review
planning_horizon
fallback_policy JSONB
updated_by
updated_at
```

## platform_content_policy_versions

```text
id
version_no UNIQUE
policy JSONB
active_from
retired_at
created_at
```

## story_content_profile_versions

```text
id
story_id
version_no
profile JSONB
base_policy_version_id
created_by
created_at

UNIQUE(story_id, version_no)
```

---

# 146. Planning Tables

## story_bible_versions

```text
id
story_id
version_no
content JSONB
based_on_version_id
created_by
generation_run_id
created_at
```

## story_ending_plan_versions

Same version pattern.

## story_arcs

Stable identity:

```text
id
story_id
ordinal
status
current_version_id
created_at
```

## story_arc_versions

```text
id
arc_id
version_no
content JSONB
base_canon_version_id
generation_run_id
created_by
created_at
```

---

# 147. Character Tables

## characters

```text
id
story_id
canonical_name
importance
current_profile_version_id
created_at
```

## character_profile_versions

```text
id
character_id
version_no
profile JSONB
base_canon_version_id
created_by
generation_run_id
created_at
```

## character_state_versions

Immutable state snapshots:

```text
id
character_id
canon_version_id
state JSONB
source_chapter_id
source_content_revision_id
generation_run_id
created_at
```

---

# 148. Chapter Tables

## chapters

```text
id
story_id
chapter_number INT
title
status
arc_id

current_plan_revision_id
current_content_revision_id
current_narration_revision_id
current_audio_asset_id
official_canon_version_id

published_at
archived_at

created_at
updated_at

UNIQUE(story_id, chapter_number)
```

---

# 149. ChapterPlan Revisions

```text
id
chapter_id
revision_no
plan JSONB
base_canon_version_id
arc_version_id
source_type
generation_run_id
created_by
created_at
```

---

# 150. Chapter Content Revisions

```text
id
chapter_id
revision_no
content_text TEXT

source_type
based_on_revision_id
plan_revision_id
base_canon_version_id

generation_run_id
retcon_request_id

status

created_by
created_at

UNIQUE(chapter_id, revision_no)
```

Source examples:

```text
AI_GENERATED
AI_REWRITE
ADMIN_EDIT
RETCON
```

Revision status examples:

```text
CANDIDATE
APPROVED
STALE
HISTORICAL
```

Current pointer lives on `chapters`.

---

# 151. Content Approval

Append-only:

```text
content_approvals

id
chapter_id
content_revision_id
approved_by
approved_at
warnings_snapshot JSONB
override_snapshot JSONB
```

Approval history is never rewritten.

---

# 152. Chapter Reviews

```text
chapter_reviews

id
chapter_id
content_revision_id

review_type
canon_version_id
policy_version_id

outcome
report JSONB

generation_run_id
created_at
```

Review types:

```text
CONTINUITY
QUALITY
SAFETY
DURATION
```

---

# 153. Content Classifications

```text
content_classifications

id
content_revision_id
rating
warnings[]
outcome
policy_version_id
report JSONB
created_at
```

---

# 154. StoryFact Table

```text
story_facts

id
story_id

subject_type
subject_id

fact_type
value JSONB
importance

status

valid_from_canon_version_id
invalidated_at_canon_version_id

supersedes_fact_id

source_chapter_id
source_content_revision_id
generation_run_id

created_at
```

Important indexes:

```text
(story_id, status)

(story_id, subject_type, subject_id)

(story_id, fact_type)
```

---

# 155. PlotThread Tables

## plot_threads

```text
id
story_id
title
summary
importance
status

opened_chapter_id
resolved_chapter_id
last_advanced_chapter_id

created_at
updated_at
```

## plot_thread_events

Append history:

```text
id
plot_thread_id
canon_version_id
chapter_id
event_type
detail JSONB
created_at
```

---

# 156. Summary Tables

## chapter_summaries

```text
id
chapter_id
content_revision_id
summary
generation_run_id
created_at
```

## story_summaries

```text
id
story_id
scope_type
scope_id
canon_version_id
summary
created_at
```

Scope:

```text
ARC
STORY_SO_FAR
```

---

# 157. Canon Tables

## canon_branches

```text
id
story_id

type
status

base_version_id

generation_run_id
retcon_request_id

created_at
```

Types:

```text
OFFICIAL
PROVISIONAL
RETCON
```

## canon_versions

```text
id
story_id
branch_id
sequence_no
parent_version_id

source_chapter_id
source_content_revision_id
source_provisional_version_id

generation_run_id
retcon_request_id

status
committed_by
created_at
```

Unique:

```text
(branch_id, sequence_no)
```

## canon_change_items

```text
id
canon_version_id
entity_type
entity_id
change_type
metadata JSONB
```

---

# 158. Generation Tables

## generation_runs

```text
id
run_type

story_id
chapter_id

status
waiting_reason

workflow_version
priority

base_canon_version_id
context_snapshot_id

requested_by
idempotency_key

started_at
completed_at
created_at
```

## generation_jobs

```text
id
run_id
job_type

status
priority
available_at

input_fingerprint

attempt_count
max_attempts

locked_by
lock_expires_at

started_at
completed_at

last_error_class
last_error_code

output_ref JSONB

created_at
```

## generation_job_dependencies

```text
job_id
depends_on_job_id
PK
```

## generation_job_attempts

```text
id
job_id
attempt_no

provider
model

status
error_class
error_code
safe_error_detail JSONB

usage JSONB
latency_ms

started_at
completed_at
```

---

# 159. ContextSnapshot Table

```text
context_snapshots

id
run_id
story_id
chapter_id

canon_version_id
bible_version_id
ending_plan_version_id
arc_version_id
content_profile_version_id

prompt_version
workflow_version

provider
model

included_refs JSONB
historical_hits JSONB
admin_instruction

created_at
```

Immutable.

---

# 160. CreativeDecision Tables

## creative_decisions

```text
id
story_id
chapter_id
arc_id

origin
decision_type
severity
status
blocking_level

question
context_summary

recommended_option_id
selected_option_id
custom_selected_text

rejection_scope
revisit_condition JSONB

triggered_by_run_id
created_by
selected_by

created_at
selected_at
applied_at
```

## creative_decision_options

```text
id
decision_id
ordinal
title
description
impact JSONB
is_recommended
created_at
```

---

# 161. Narration / Audio Tables

## narrator_voices

```text
id
provider
provider_voice_id
display_name
rights_status
active
metadata JSONB
```

## narration_revisions

```text
id
chapter_id
revision_no
source_content_revision_id
voice_id
script
status
generation_run_id
created_by
created_at
```

## tts_segments

```text
id
narration_revision_id
segment_no
text
direction JSONB

status

provider
model
voice_id

duration_ms
temp_storage_key
generation_job_id

created_at

UNIQUE(narration_revision_id, segment_no)
```

## audio_assets

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

is_active

generation_run_id
created_at
```

Partial unique:

```text
UNIQUE(chapter_id)
WHERE is_active = true
```

---

# 162. Listener Tables

## favorites

```text
user_id
story_id
created_at

PK(user_id, story_id)
```

## playback_sessions

```text
id
user_id
chapter_id
audio_asset_id

client_instance_id

started_at
last_event_at
ended_at
```

## listening_progress

```text
user_id
chapter_id

position_ms
completed_at

last_audio_asset_id
last_playback_session_id

version BIGINT

last_listened_at
updated_at

PK(user_id, chapter_id)
```

`version` supports optimistic progress concurrency.

---

# 163. Retcon Tables

## retcon_requests

```text
id
story_id
target_chapter_id

status
impact_scope

proposed_change
reason

requested_by
approved_by
applied_by

base_official_canon_version_id
workspace_branch_id

listener_impact

created_at
approved_at
applied_at
```

## retcon_impacts

```text
id
retcon_request_id

entity_type
entity_id

impact_type
detail JSONB
```

## retcon_repair_tasks

```text
id
retcon_request_id

task_type
entity_type
entity_id

status

generation_run_id
detail JSONB

created_at
completed_at
```

---

# 164. Audit Tables

## audit_events

Append-only:

```text
id
event_type

actor_user_id
generation_run_id

story_id
chapter_id

entity_type
entity_id

request_id
metadata JSONB

created_at
```

No normal Update/Delete API.

## security_events

```text
id
user_id
event_type
success
safe_metadata JSONB
created_at
```

## idempotency_keys

```text
key
actor_user_id
operation_scope
request_hash
response_ref JSONB
created_at
expires_at

PK(key, operation_scope)
```

---

# 165. Important DB Constraints

At minimum:

```text
unique Story slug

unique Story Chapter number

unique Favorite

unique User role

one StoryGenerationPolicy per Story

Chapter number > 0

position/duration >= 0

unique Canon branch sequence

one active AudioAsset per Chapter

status CHECK constraints

historical Canon/Audit references do not cascade-delete
```

Sequential Official/Publish validation stays in application transaction because simple CHECK cannot fully enforce it.

---

# 166. Important Indexes

Catalog:

```text
stories(visibility, status, last_published_at DESC)

Full-text + trigram:
title
description

story_genres(genre_id, story_id)
```

Chapter:

```text
(story_id, chapter_number)

(story_id, status, chapter_number)
```

Generation:

```text
(status, available_at, priority)

(run_id, status)
```

Memory:

```text
story_facts(story_id, status)

story_facts(story_id, subject_type, subject_id)

plot_threads(story_id, status, importance)
```

Listener:

```text
listening_progress(user_id, last_listened_at DESC)

favorites(user_id, created_at DESC)
```

Audit:

```text
(story_id, created_at DESC)

(entity_type, entity_id, created_at DESC)
```

---

# 167. Local Environment

Docker Compose from day one.

Services:

```text
postgres
minio
api
worker
```

Vue may run directly via local dev server or Compose.

Local DB/storage never point production accidentally.

Safety flags:

```text
ALLOW_REMOTE_DATABASE_IN_DEV=false
ALLOW_REMOTE_STORAGE_IN_DEV=false
```

Development startup fails on accidental remote resources unless explicit override.

Same migrations local/prod.

Dev seed separate.

---

# 168. Production Infrastructure

Baseline:

```text
Vue/Vite       → Vercel

Go API         → Render Web Service

Go Worker      → Render Worker / separate process

PostgreSQL     → Neon

Object Storage → Cloudflare R2
```

Backend filesystem is temporary only.

Use `/tmp` for transient FFmpeg/TTS processing.

Never treat Render filesystem as persistent storage.

---

# 169. API / Worker Separation

Codebase has separate entry points:

```text
cmd/api
cmd/worker
```

Local runs both.

Production architecture prefers separate processes/services.

API public workload should not compete with long AI/TTS execution.

Microservice extraction only after actual operational need.

---

# 170. Health Endpoints

```text
GET /healthz
```

Shallow liveness:

* process alive;
* no DB/R2/AI/TTS calls.

```text
GET /readyz
```

Checks:

* DB connectivity;
* job-store connectivity.

Do not require external AI/TTS to report API ready.

AI provider outage must not take public catalog/listening API out of service.

---

# 171. Degraded Behavior

Text AI outage:

```text
Published app works.
Generation blocked/retrying.
```

TTS outage:

```text
Existing audio works.
New audio production blocked.
```

R2 outage:

```text
Metadata/text may work.
Audio/cover temporarily unavailable.
```

DB outage:

```text
Controlled 503.
```

Email outage:

```text
Public listening unaffected.
Email workflows retry.
```

---

# 172. Observability

Use structured JSON logs.

Include:

* timestamp;
* level;
* service;
* request_id;
* story/chapter/run/job IDs;
* latency;
* domain error code.

Never log:

* password;
* JWT;
* refresh token;
* auth header;
* TOTP secret;
* provider API key.

Metrics:

* API latency/error;
* queue depth;
* job success/failure;
* retries;
* provider latency;
* TTS failure;
* generation duration;
* estimated token/cost;
* storage upload failures.

---

# 173. NFR Baseline

```text
Metadata API p95 target < 500 ms under normal load.

Long-running generation is async.

Public API never waits on AI/TTS.

All timestamps use UTC TIMESTAMPTZ.

High-cost/high-impact operations are idempotent.

Generation survives restart.

Concurrent stale edits never silently overwrite.

PRIVATE content is not publicly exposed.

Backend does not proxy large MP3.

Published listener flow keeps working when AI provider is down.
```

---

# 174. Backup & Disaster Recovery

Baseline:

* DB provider-managed backup/PITR where available;
* daily logical PostgreSQL backup;
* backup stored privately;
* checksum important assets.

Retention target:

```text
14 daily backups
8 weekly backups
```

Operational target:

```text
RPO <= 24h
RTO <= 8h
```

These are MVP recovery objectives, not a commercial SLA.

Restore must periodically be tested outside production.

---

# 175. Backend Repository Structure

```text
backend/
├── cmd/
│   ├── api/
│   └── worker/
│
├── internal/
│   ├── identity/
│   ├── catalog/
│   ├── story/
│   ├── planning/
│   ├── canon/
│   ├── memory/
│   ├── generation/
│   ├── governance/
│   ├── audio/
│   ├── listener/
│   ├── audit/
│   └── platform/
│
├── db/
│   ├── migrations/
│   ├── queries/
│   └── sqlc.yaml
│
├── openapi/
│   └── api.yaml
│
├── prompts/
│   ├── story-architect/
│   ├── arc-planner/
│   ├── chapter-planner/
│   ├── story-writer/
│   ├── reviewers/
│   ├── memory-extractor/
│   └── narration/
│
└── tests/
```

Prompt definitions are versioned source artifacts.

---

# 176. Frontend Repository Structure

```text
frontend/
├── src/
│   ├── app/
│   ├── router/
│   ├── api/
│   ├── stores/
│   ├── features/
│   │   ├── auth/
│   │   ├── catalog/
│   │   ├── story/
│   │   ├── player/
│   │   ├── listener/
│   │   └── admin/
│   │       ├── dashboard/
│   │       ├── stories/
│   │       ├── chapters/
│   │       ├── generation/
│   │       ├── canon/
│   │       ├── decisions/
│   │       ├── retcons/
│   │       └── audit/
│   └── components/
```

---

# 177. Configuration

Conceptual environment configuration:

```text
APP_ENV=

DATABASE_URL=

STORAGE_PROVIDER=minio|r2
STORAGE_ENDPOINT=
STORAGE_BUCKET=
STORAGE_ACCESS_KEY=
STORAGE_SECRET_KEY=
SIGNED_AUDIO_URL_TTL=

TEXT_AI_PROVIDER=
TEXT_AI_MODEL=
TEXT_AI_API_KEY=

TTS_PROVIDER=
TTS_MODEL=
TTS_API_KEY=
DEFAULT_VOICE_ID=

EMAIL_PROVIDER=
EMAIL_FROM=

ACCESS_TOKEN_TTL=
REFRESH_SESSION_TTL=
RECENT_AUTH_WINDOW=

AI_MODE=real|mock
TTS_MODE=real|mock

ALLOW_REMOTE_DATABASE_IN_DEV=false
ALLOW_REMOTE_STORAGE_IN_DEV=false
```

Production rejects:

```text
AI_MODE=mock
TTS_MODE=mock
unsafe debug
local-only storage config
development safety override
```

---

# 178. OpenAPI

`openapi/api.yaml` is the HTTP contract source.

Workflow:

```text
Update OpenAPI
→ Generate/shared types
→ Implement backend
→ Generate frontend API types/client
→ Contract tests
```

Frontend must not infer arbitrary response shapes from backend code.

---

# 179. Testing

## Unit

Prioritize business rules:

* Story activation;
* Story completion;
* Chapter transitions;
* approval invalidation;
* Canon sequence;
* Publish sequence;
* CreativeDecision;
* Retcon;
* permission;
* duration gate;
* listener progress conflict.

## Integration

Use real PostgreSQL test instance/Testcontainers for:

* sqlc queries;
* transactions;
* job claim/recovery;
* Canon Commit;
* Retcon Apply;
* session revoke;
* audio active constraint.

## AI contract tests

Mock provider + structured fixtures.

Test output schema and invariants.

Do not test exact creative prose.

## E2E

Playwright:

```text
Register/Login

Admin Create Story

Story Activate

Generate Chapter with Mock AI

Review/Approve

Memory/Canon Commit

Mock TTS

READY

Publish

Public read/listen

Favorite/Progress

Failed Job Retry

Stale Conflict

Unpublish/Private
```

---

# 180. Implementation Phase 0 — Foundation

Build first:

1. repository structure;
2. Docker Compose;
3. PostgreSQL;
4. MinIO;
5. Go `cmd/api`;
6. Go `cmd/worker`;
7. config loader;
8. development safety guards;
9. migrations;
10. sqlc;
11. OpenAPI skeleton;
12. `/healthz`;
13. `/readyz`;
14. structured logging;
15. Vue/Vite app;
16. `/admin` shell;
17. CI.

After Phase 0, infrastructure foundation is ready.

No further product specification is required before starting this phase.

---

# 181. Implementation Phase 1 — Identity + Story Foundation

Implement:

* User;
* sessions;
* register/login;
* email verification;
* password reset;
* TOTP Admin MFA;
* Role/Permission;
* Genres;
* Story;
* GenerationPolicy;
* WorkflowSettings;
* ContentProfile;
* Cover upload;
* PRIVATE/PUBLIC shell;
* public Story catalog/search;
* Admin Story list/dashboard shell.

At the end, Admin can securely create Story records.

---

# 182. Implementation Phase 2 — Planning + Canon Foundation

Implement:

* Story Bible versions;
* Ending versions;
* Arcs;
* Characters;
* Character States;
* Chapter Plans;
* StoryFacts;
* PlotThreads;
* Canon branches;
* Canon versions;
* ContextSnapshots;
* Story Architect/Planner using mock then real provider.

At the end, Story can become ACTIVE with coherent foundation.

---

# 183. Implementation Phase 3 — Chapter Generation

Implement:

* GenerationRun;
* GenerationJob;
* JobAttempt;
* Worker queue;
* TextAI adapter;
* Context Builder;
* Writer;
* Duration Analyzer;
* Continuity Review;
* Quality Review;
* Safety Review;
* Rewriter;
* Admin Content Review;
* Content Approval;
* Memory Extractor;
* Canon validation/commit;
* retry;
* cancel;
* stale handling;
* idempotency.

At the end, canonical Chapter text production works.

---

# 184. Implementation Phase 4 — Audio + Publishing + Listener

Implement:

* NarrationRevision;
* TTS Segments;
* TTS adapter;
* FFmpeg;
* AudioAsset;
* MinIO/R2 signed URLs;
* READY gate;
* Publish;
* Unpublish;
* Story PUBLIC;
* public text reader;
* audio player;
* Favorite;
* ListeningProgress;
* Continue Listening;
* Guest local progress;
* basic multi-device conflict handling.

At the end, the product is usable end-to-end.

---

# 185. Implementation Phase 5 — Long Story Operations

Implement:

* Batch Generation;
* Provisional Canon;
* Arc Completion Review;
* CreativeDecision complete flow;
* attention queues;
* advanced Story Control Center;
* cost/usage visibility;
* Context Debug;
* quality drift/thread inactivity analysis.

---

# 186. Implementation Phase 6 — Retcon & Hardening

Implement:

* Canon Data Repair;
* Retcon analysis;
* Retcon workspace;
* Repair Tasks;
* Retcon atomic apply;
* listener revision impact;
* account purge lifecycle;
* backup/restore drill;
* degraded provider UX;
* security/operations hardening.

---

# 187. Initial Release Boundary

The first usable release does **not** need fully automated Retcon.

Core release needs:

```text
Auth/Admin

Story Foundation

Single-Chapter Generation

Reviews

Content Approval

Canon/Memory

Narration/TTS

Audio

Publish

Catalog

Read

Listen

Favorite

Progress

Audit

Private R2 signed media
```

Batch/advanced Retcon can follow.

But initial schema/module boundaries must not block them.

---

# 188. Non-goals for V1

Do not implement merely for future-proofing:

```text
Microservices

Redis just because

RabbitMQ/NATS without actual need

pgvector dependency

Recommendation engine

Comments/social features

Creator marketplace

Story creator ownership

Subscription/paywall

Multi-speaker TTS

AI cover generation

Legal-grade age verification

Auto-publish

Full human-written import

Generic workflow engine

Two-person Retcon approval

Realtime collaborative Admin editing
```

---

# 189. Definition of Done — Core Product

Core implementation is complete when:

1. Admin can create a Story with immutable GenerationPolicy.
2. Story Architect creates foundation.
3. Activation Gate works.
4. ChapterPlan/Context Builder produces controlled context.
5. GenerationRun survives restart.
6. Text generation/reviews/rewrite work through provider abstraction.
7. Admin approves an exact Content Revision.
8. Memory Extraction uses that exact revision.
9. Canon Commit is traceable and atomic.
10. Chapter enters PRODUCTION.
11. Narration/TTS segmented pipeline works.
12. Failed TTS segment can retry without rerunning all TTS.
13. FFmpeg creates final MP3.
14. Audio is stored private in MinIO/R2.
15. Chapter reaches READY only after gates.
16. Admin explicitly publishes.
17. Public APIs expose only valid content.
18. Browser streams signed audio directly from object storage.
19. User can Favorite/Resume.
20. Guest can retain local progress.
21. Stale edit/progress cannot silently overwrite current state.
22. Private/Unpublish preserves historical User data.
23. Audit traces request → generation → context → approval → Canon → audio → publish.
24. Platform content block cannot be bypassed with another provider.
25. Admin suspension/revocation works.
26. Last Admin Guard works.
27. Full E2E pipeline works locally with Mock AI/TTS.

---

# 190. Frozen Implementation Decisions

The following are now considered frozen baseline decisions:

```text
Vue 3 + Vite + TypeScript

Go Modular Monolith

Separate API and Worker entry points

PostgreSQL

pgx + sqlc

PostgreSQL-backed Job Queue

Neon production DB

MinIO local storage

Cloudflare R2 production storage

Private media bucket

Signed direct media URLs

Vercel frontend

Render API + Worker

Provider abstraction for AI/TTS

Gemini-direction text provider

Gemini TTS primary direction

Configurable exact model IDs

Configurable narrator voice

FFmpeg

Access JWT + rotating opaque Refresh Session

Argon2id

Admin TOTP MFA

GUEST / USER / ADMIN

Platform-owned Stories

Story status/visibility/planning axes separated

Chapter:
DRAFT → CONTENT_REVIEW → PRODUCTION → READY → PUBLISHED

Canon:
DRAFT → PROVISIONAL → OFFICIAL

Revision-bound approval

No Auto-Publish

StoryFact + PlotThread core

ContextSnapshot core

Relational Story Memory V1

pgvector later

Sequential Batch Generation

Controlled Retcon

GENERAL / TEEN / MATURE

No explicit pornographic product mode V1

Voice/Likeness rights enforcement

Append-only Audit

Docker Compose from day one

Development remote-resource guards

Manual Cover upload V1

OpenAPI 3.1

Polling Generation progress V1

No Microservices until justified
```

---

# 191. Configuration, Not Architecture

The following do **not** require reopening the FINAL specification:

* exact Gemini model name;
* exact TTS model ID;
* selected narrator voice;
* exact token/session TTL;
* exact retry numbers;
* exact timeout;
* exact concurrency;
* exact Genre seed;
* warning labels;
* product brand;
* visual UI design;
* patch/minor dependency versions;
* cloud instance size.

These are configuration/operational choices.

---

# 192. Change Control After FINAL

From this point forward, requirements are classified as:

## Configuration Change

No architecture redesign.

## Implementation Detail

Record in ADR/code documentation.

## Domain Change

Must explicitly document:

* new requirement;
* existing invariant affected;
* migration impact;
* compatibility impact.

Do not silently rewrite historical spec decisions.

---

# 193. First Engineering Task

Start implementation with:

```text
Phase 0 — Foundation
```

Exact first sequence:

```text
1. Repository skeleton
2. Docker Compose
3. PostgreSQL
4. MinIO
5. Go API entry point
6. Go Worker entry point
7. Environment/config loader
8. Dev safety guards
9. Migration setup
10. sqlc setup
11. OpenAPI skeleton
12. /healthz
13. /readyz
14. Logging + request IDs
15. Vue/Vite shell
16. /admin shell
17. CI
18. First Identity/Role/Story migrations
```

At this point, implementation should start.

No additional product/domain specification round is required before coding.

---

# 194. FINAL STATUS

```text
PRODUCT SPEC       FROZEN
BUSINESS RULES     FROZEN
DOMAIN MODEL       FROZEN
STATE MACHINES     FROZEN
AI WORKFLOW        FROZEN
CANON/MEMORY       FROZEN
AUTH/RBAC          FROZEN
LISTENER LOGIC     FROZEN
ADMIN WORKFLOW     FROZEN
CONTENT GOVERNANCE FROZEN

ARCHITECTURE        SELECTED
PHYSICAL DB MODEL   SELECTED
API DIRECTION       SELECTED
JOB/WORKER MODEL    SELECTED
SECURITY BASELINE   SELECTED
DEPLOYMENT BASELINE SELECTED
IMPLEMENTATION PLAN SELECTED
```

**The project is ready to move from specification to implementation.**
