# 10 — Story Canon and Memory Engine

## 1. Mục tiêu

AI phải giữ continuity cho series hàng trăm chapter mà không cần nhét toàn bộ raw story vào mỗi prompt.

Story Memory là capability backend, không phải memory vĩnh viễn của model.

---

## 2. Source of truth

```text
CANON
├── Story Bible
├── Ending Plan
├── Story Arcs
├── Character Profiles
├── Character States
├── Active Story Facts
├── Plot Threads
└── Approved Creative Decisions
```

Full chapter text luôn được lưu làm archive/provenance.

---

## 3. Memory hierarchy

### Level 0 — Canon Constitution

Story Bible, World Rules, Writing Rules, Major Constraints, Planned Ending.

### Level 1 — Current State

Current Arc, Character State, World State, Relationships, Inventory, Knowledge.

### Level 2 — Active Narrative Memory

Active Facts, Open Plot Threads, Unresolved Mysteries, Current Goals, Setups awaiting payoff.

### Level 3 — Hierarchical Summaries

```text
Chapter Summary
    ↓
Arc Summary
    ↓
Story-so-far Summary
```

### Level 4 — Recent Context

Previous chapter/full excerpt và recent detailed summaries.

### Level 5 — Historical Retrieval

Lookup fact/entity/scene cũ khi chapter hiện tại cần.

---

## 4. Context Builder

```text
Generation Request
      ↓
Context Builder
      ├── Policy
      ├── Story Bible
      ├── Ending Plan
      ├── Current Arc
      ├── Chapter Plan
      ├── Character States
      ├── Active Facts
      ├── Plot Threads
      ├── Recent Context
      └── Historical Retrieval
      ↓
Context Pack
      ↓
LLM
```

---

## 5. Context Snapshot

Mỗi Generation Run phải trace được Canon version, Story Bible version, Ending Plan version, Arc version, Character state versions, Fact IDs, Plot Thread IDs, historical references, prompt version và model/provider.

---

## 6. Fact provenance

Fact phải biết source chapter, source segment/scene, generation run và canon version.

Semantic retrieval chỉ tìm candidate memory; không phải source of truth.

---

## 7. Retrieval strategy

V1 ưu tiên relational retrieval theo story/entity/arc/importance/status.

Sau này có thể thêm PostgreSQL + pgvector cho semantic retrieval trước khi cân nhắc standalone vector DB.

---

## 8. Canon lifecycle

```text
DRAFT
   ↓
AI + SYSTEM GATES
   ↓
PROVISIONAL
   ↓
ADMIN APPROVAL
   ↓
OFFICIAL
```

---

## 9. Canon commit

```text
Approved Chapter
      ↓
Memory Extractor
      ↓
Validated Change Set
      ↓
Transaction
      ↓
New Official Canon Version
```

---

## 10. Batch generation

```text
Official v100
 ↓
Ch101 → Provisional v101
 ↓
Ch102 → Provisional v102
```

Nếu upstream thay đổi, downstream được đánh `STALE`.

---

## 11. Planning memory

```text
near future → detailed
current arc → detailed
next arc → medium
far future → high-level
ending → strategic
```

---

## 12. Retcon / Canon revision

Published/Official history không có normal Edit.

Flow:

```text
Request Canon Revision
      ↓
Retcon Impact Analyzer
      ↓
Dependency Analysis
      ↓
Impact Report
      ↓
Repair Plan
      ↓
Admin Confirmation
      ↓
Mark affected items stale
      ↓
Controlled repair
```

Hệ thống nên cảnh báo mạnh và khuyến nghị tránh retcon khi đã có nhiều downstream chapters.

---

## 13. Story Memory microservice

Logical subsystem riêng, nhưng V1 nằm trong modular monolith.

Chỉ tách khi có trigger như independent scaling, complex hybrid retrieval, many worker consumers, ML-heavy stack hoặc failure isolation.
