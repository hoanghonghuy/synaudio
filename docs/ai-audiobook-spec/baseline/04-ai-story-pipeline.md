# 04 — AI Story Pipeline

## 1. Nguyên tắc

Series dài không generate bằng một prompt độc lập. AI có creative freedom có kiểm soát, nhưng không được âm thầm phá Canon.

---

## 2. Pipeline tổng thể

```text
Story Idea
    ↓
Story Architect
    ↓
Story Bible + Characters + Ending Plan + High-level Arcs
    ↓
Arc Planner
    ↓
Chapter Planner
    ↓
Story Writer
    ↓
Content Gap / Duration Analysis
    ↓
Continuity Reviewer
    ↓
Writing Quality Reviewer
    ↓
Rewriter
    ↓
Admin Content Review
    ↓
Memory Extractor
    ↓
Canon Commit
    ↓
Narration Director
    ↓
Admin Narration Review
    ↓
TTS
    ↓
Audio Quality Gate
    ↓
Publish
```

---

## 3. Story từ một câu idea

Admin có thể nhập một idea ngắn. Story Architect tạo proposal cho premise, Story Bible, Characters, Planned Ending, Major Arcs và initial Plot Threads.

Không generate chi tiết hàng trăm chapter ngay.

---

## 4. Creative Contract

AI được tự xử lý minor detail như dialogue, environment, clue nhỏ, conflict phụ hoặc minor plot thread.

Major change cần `CreativeDecision`, ví dụ:

```text
character death
betrayal
romance change
identity reveal
major power change
villain reveal
arc ending change
planned ending change
```

---

## 5. Creative Decision flow

```text
Need Major Decision
      ↓
AI proposes A/B/C
      +
Admin Custom
      ↓
Impact Analyzer
      ↓
Canon Conflict Check
      ↓
Future Plot Analysis
      ↓
Admin Confirm
      ↓
Apply
```

---

## 6. Chapter Plan bắt buộc

```text
objective
opening
scenes
characters
required facts
planned facts
cliffhanger
duration budget
```

Planning và prose generation là hai bước độc lập.

---

## 7. Context Builder

Backend build context từ:

```text
Story Policy
Story Bible
Ending Plan
Current Arc
Chapter Plan
Relevant Characters
Character States
Active Facts
Active Plot Threads
Recent Context
Historical Retrieval
Approved Creative Decisions
```

Chi tiết ở `10-story-canon-and-memory.md`.

---

## 8. AI Review

### Continuity Review

Kiểm tra timeline, facts, characters, world rules, knowledge, relationships, arc objectives và plot threads.

### Writing Quality Review

Kiểm tra repetition, pacing, dialogue, AI-like wording, over-explanation, boring section, opening, ending và filler.

### Rewriter

Reviewer trả issues; Rewriter mới sửa prose.

---

## 9. Duration Quality Gate

Under-duration không được giải quyết bằng filler.

```text
Under target
    ↓
Content Gap Analyzer
    ↓
Expansion proposal
    ↓
Admin/system decision
```

Options có thể gồm:

```text
Expand recommended scenes
Regenerate chapter
Edit manually
Regenerate chapter plan
Approve override
```

---

## 10. Human review

Default thiên về Semi-auto:

```text
Plan
→ Generate
→ AI Reviews
→ Rewrite
→ STOP at Content Review
```

Sau Admin approve mới Memory Extraction và Canon Commit.

---

## 11. Memory Extraction

Extractor chạy trên **final approved content**, trả structured change set:

```text
summary
factsCreated
factsSuperseded
characterUpdates
plotThreadsOpened
plotThreadsAdvanced
plotThreadsResolved
worldUpdates
```

Backend validate rồi mới commit Canon.

---

## 12. Batch generation

Batch là mutable workflow setting, nhưng generation luôn sequential.

```text
Official v100
 ↓
Ch101 → Provisional v101
 ↓
Ch102 → Provisional v102
```

Nếu upstream chapter thay đổi, downstream draft/provisional được đánh dấu `STALE`.

---

## 13. Prompt versioning

Giai đoạn đầu prompt version control trong repository:

```text
prompts/
├── story-architect.md
├── arc-planner.md
├── chapter-plan.md
├── chapter-write.md
├── continuity-review.md
├── quality-review.md
├── rewrite.md
├── memory-extract.md
├── narration.md
└── tts-director.md
```
