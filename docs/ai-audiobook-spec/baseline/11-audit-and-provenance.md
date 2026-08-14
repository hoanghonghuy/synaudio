# 11 — Audit and Provenance

## 1. Audit

Audit trả lời:

```text
WHO?
WHAT?
WHEN?
```

Target:

```text
audit_events
------------
id
story_id

actor_user_id
action

entity_type
entity_id

generation_run_id
request_id

metadata JSONB

created_at
```

Append-only. Không có `updated_at` hoặc `deleted_at`.

---

## 2. Important audit actions

```text
STORY_CREATED
STORY_ARCHIVED
STORY_BIBLE_CHANGED
CHARACTER_CHANGED
CHARACTER_STATE_MANUALLY_CHANGED
CHAPTER_GENERATED
CHAPTER_APPROVED
CHAPTER_PUBLISHED
CREATIVE_DECISION_PROPOSED
CREATIVE_DECISION_APPROVED
CANON_COMMITTED
RETCON_REQUESTED
RETCON_ANALYZED
RETCON_APPLIED
AUDIO_GENERATED
AUDIO_REPLACED
```

---

## 3. AI is not a fake user

Không tạo fake user `SYSTEM`/`AI`.

AI event:

```text
actor_user_id = NULL
generation_run_id = ...
```

Human approval:

```text
actor_user_id = admin_uuid
```

---

## 4. Provenance

Provenance trả lời:

```text
HOW was this produced?
WHERE did it come from?
```

Ví dụ StoryFact:

```text
source_type
source_chapter_id
source_segment
generation_run_id
canon_version_id
```

---

## 5. Human responsibility + AI provenance

Một fact có thể có:

```text
created_by = admin_user_id
generation_run_id = memory_extractor_run
source_chapter_id = ...
```

Actor và source là hai thứ khác nhau.

---

## 6. Audit field convention

Mutable entity có `created_*`, `updated_*`, và `deleted_*` khi nghiệp vụ cần.

Immutable record chỉ giữ field có ý nghĩa.

Business lifecycle không được thay bằng generic delete.

---

## 7. Traceability chain

```text
HTTP request
→ generation run
→ jobs
→ context snapshot
→ provider calls
→ output
→ admin action
→ canon commit
```

Retention policy sẽ chốt ở security/operations phase.
