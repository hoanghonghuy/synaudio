# 03 — Domain and Data Model

## 1. Core domain

```text
Story
Genre
StoryGenre
StoryGenerationPolicy
StoryWorkflowSettings
StoryBible
StoryArc
StoryEndingPlan

Character
CharacterState

ChapterPlan
Chapter
ChapterSummary

StoryFact
PlotThread
CreativeDecision

CanonVersion
GenerationRun
GenerationJob
ContextSnapshot

AudioAsset

User
Favorite
ListeningProgress

AuditEvent
```

Các entity trên mô tả **target domain**. Implementation được chia phase trong roadmap.

---

## 2. Story

```text
stories
-------
id
title
slug
description
cover_url

planning_mode
planning_phase
status

created_at
created_by
updated_at
updated_by
deleted_at
deleted_by
```

Planning mode:

```text
FINITE
OPEN_ENDED
```

Planning phase:

```text
ONGOING
CLOSING
FINAL_ARC
COMPLETED
```

Story status:

```text
DRAFT
ACTIVE
COMPLETED
ARCHIVED
```

`ARCHIVED` không đồng nghĩa deleted.

---

## 3. Genre

Một Story có nhiều Genre.

```text
genres
------
id
name
slug
created_at
created_by
updated_at
updated_by
```

```text
story_genres
------------
story_id
genre_id
created_at
created_by
```

Quan hệ:

```text
Story N:M Genre
```

---

## 4. Story Generation Policy — immutable

Đây là **Story Creation Contract**.

```text
story_generation_policies
-------------------------
story_id

minimum_chapter_duration_seconds
target_chapter_duration_seconds

content_origin
language
default_narration_language

policy_version

created_at
created_by
```

Không có `updated_at/updated_by` vì application không hỗ trợ sửa policy sau khi Story được tạo.

Content origin target:

```text
AI_GENERATED
HUMAN_WRITTEN
AI_ASSISTED
```

V1 tập trung `AI_GENERATED`.

---

## 5. Story Workflow Settings — mutable

```text
story_workflow_settings
-----------------------
story_id

batch_generation_size
auto_run_ai_review
pause_after_content_generation
pause_before_tts

preferred_story_model
preferred_tts_provider

detailed_chapter_horizon
rough_chapter_horizon
future_arc_horizon

created_at
created_by
updated_at
updated_by
```

`batch_generation_size` có thể đổi theo nhu cầu; generation vẫn tuần tự theo Canon.

---

## 6. Story Bible

```text
story_bibles
------------
id
story_id
version
content JSONB
is_current

created_at
created_by
```

Story Bible chứa:

```text
premise
tone
writing style
world
world rules
main plot
constraints
target audience
```

---

## 7. Story Arc

```text
story_arcs
----------
id
story_id
arc_number
title
summary
objective
status
metadata JSONB

created_at
created_by
updated_at
updated_by
```

---

## 8. Story Ending Plan

```text
story_ending_plans
------------------
id
story_id
version
content JSONB
is_current

created_at
created_by
```

Ending change phải qua `CreativeDecision` + impact analysis.

---

## 9. Character

Static:

```text
characters
----------
id
story_id
name
aliases
profile JSONB
status

created_at
created_by
updated_at
updated_by
deleted_at
deleted_by
```

Dynamic state:

```text
character_states
----------------
id
character_id
canon_version_id
state JSONB

created_at
created_by
```

State có thể chứa:

```text
location
condition
emotional_state
inventory
knowledge
relationships
goals
abilities
status
```

---

## 10. Chapter Plan

```text
chapter_plans
-------------
id
story_id
arc_id
chapter_number

objective
target_duration_seconds
plan JSONB
status

created_at
created_by
updated_at
updated_by
```

`plan` có thể chứa scene purpose, characters, duration budget, required facts, planned facts và cliffhanger.

---

## 11. Chapter

```text
chapters
--------
id
story_id
arc_id
chapter_number

title
content
narration_script
summary

status
published_at

created_at
created_by
updated_at
updated_by
```

Unique:

```text
UNIQUE(story_id, chapter_number)
```

Status:

```text
DRAFT
REVIEW
READY
PUBLISHED
ARCHIVED
```

---

## 12. Story Fact

```text
story_facts
-----------
id
story_id
chapter_id
canon_version_id

subject_type
subject_id
fact_type
content

importance
status

source_segment
generation_run_id

created_at
created_by
```

Status:

```text
ACTIVE
SUPERSEDED
INVALIDATED
```

Không dùng generic soft-delete để thay semantic lifecycle.

---

## 13. Plot Thread

```text
plot_threads
------------
id
story_id
arc_id

title
description
importance
status

introduced_chapter_id
resolved_chapter_id

metadata JSONB

created_at
created_by
updated_at
updated_by
```

Status:

```text
OPEN
ADVANCING
RESOLVED
ABANDONED
```

---

## 14. Creative Decision

```text
creative_decisions
------------------
id
story_id
arc_id
chapter_id

decision_type
importance

context
options JSONB
selected_option
custom_instruction

impact_analysis JSONB
status

created_at
created_by
resolved_at
resolved_by
```

Status:

```text
PROPOSED
WAITING_FOR_ADMIN
SELECTED
REJECTED
APPLIED
```

Decision type ví dụ:

```text
CHARACTER_DEATH
BETRAYAL
ROMANCE_CHANGE
IDENTITY_REVEAL
MAJOR_DISCOVERY
POWER_CHANGE
VILLAIN_REVEAL
ARC_CHANGE
ENDING_CHANGE
CUSTOM
```

---

## 15. Canon Version

```text
canon_versions
--------------
id
story_id
version_number
status
source_chapter_id
generation_run_id

created_at
created_by
```

Canon states:

```text
DRAFT
PROVISIONAL
OFFICIAL
```

---

## 16. Generation Run

```text
generation_runs
---------------
id
story_id
chapter_id
run_type
status
base_canon_version_id

created_at
created_by
completed_at
```

---

## 17. Generation Job

```text
generation_jobs
---------------
id
generation_run_id
story_id
chapter_id

job_type
provider
model
status

attempts
error_message

started_at
completed_at
created_at
created_by
```

Status:

```text
PENDING
RUNNING
SUCCEEDED
FAILED
CANCELLED
```

---

## 18. Context Snapshot

```text
context_snapshots
-----------------
id
generation_run_id
story_id
chapter_id
canon_version_id

context_manifest JSONB

created_at
created_by
```

Manifest ghi lại version/reference đã cấp cho model.

---

## 19. Audio Asset

```text
audio_assets
------------
id
chapter_id
version

provider
model
voice

object_key
format
duration_seconds
bitrate_kbps

status
is_active

created_at
created_by
updated_at
updated_by
deleted_at
deleted_by
```

---

## 20. User / Favorite / Listening Progress

```text
users
-----
id
email
display_name
avatar_url
created_at
updated_at
```

```text
favorites
---------
user_id
story_id
created_at
```

```text
listening_progress
------------------
user_id
chapter_id
position_seconds
completed
updated_at
```

---

## 21. Audit field convention

### Mutable business entity

Khi có ý nghĩa:

```text
created_at
created_by
updated_at
updated_by
deleted_at
deleted_by
```

### Immutable record

Ví dụ `StoryGenerationPolicy`, `AuditEvent`, `CanonVersion` chỉ có field có ý nghĩa; không thêm `updated_at` nếu không được update.

### Soft delete

Không cần cả `delete_flag` và `deleted_at`. Business status như `ARCHIVED`, `SUPERSEDED`, `INVALIDATED` không được thay bằng generic delete.

---

## 22. Timestamp convention

```text
TIMESTAMPTZ
UTC
```

---

## 23. Physical schema

Các quyết định sau sẽ khóa ở phase database implementation:

- enum hay VARCHAR/check constraint;
- exact FK/cascade strategy;
- indexes/partial indexes;
- JSONB validation;
- migration tool;
- pgvector phase;
- retention/archive policy.
