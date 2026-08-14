# 08 — Target Capability and Roadmap

## 1. Nguyên tắc

Spec mô tả cả target architecture và implementation phases.

---

## 2. Core target capabilities

```text
Story Bible
Characters / Character State
Story Arcs
Ending Plan
Chapter Plan
Story Facts
Plot Threads
Creative Decisions
Canon Versions
Story Memory Engine
Context Builder
Context Snapshot
AI Reviews
Memory Extraction
Retcon Analysis
Narration/TTS
Audit/Provenance
Docker Compose
Environment Isolation
```

---

## 3. MVP / Phase 1

### Foundation

```text
Repo structure
Docker Compose
PostgreSQL local
MinIO local
Go modular monolith
Vue app
env/config validation
migration baseline
```

### Story

```text
Story
Genre
Story Generation Policy
Story Workflow Settings
Story Bible
Story Arc
Character
Chapter Plan
Chapter
Story Fact
Plot Thread
Generation Run
Generation Job
Audit Event
```

### AI

```text
Story Architect
Arc Planner
Chapter Planner
Story Writer
Continuity Review
Quality Review
Rewrite
Memory Extractor
Context Builder v1
```

### Audio

```text
Narration
TTS abstraction
Gemini provider
FFmpeg
R2/MinIO abstraction
AudioAsset
Player
```

### Admin

```text
Create story from idea
Edit Story Bible
Edit Characters
Edit Arcs
Review chapter
Resolve Creative Decision
Generate audio
Publish
```

---

## 4. Phase 2

```text
Semantic memory with pgvector
Historical retrieval improvements
Context ranking
Retcon impact analyzer
Repair plan workflow
Provisional Canon batch generation
Advanced generation dashboard
Provider fallback
Worker separation
Task queue if justified
```

---

## 5. Phase 3 / Scale

```text
Story Memory microservice if needed
Dedicated generation workers
Dedicated audio workers
Hybrid search / reranking
Advanced observability
Distributed tracing
Autoscaling
Notification subsystem
```

---

## 6. Future product expansion

```text
Human-written import
AI-assisted author mode
Multiple character voices
Subscriptions
Payments
Creator tools
Recommendations
Native mobile apps
Multilingual narration
```

---

## 7. Initial Definition of Done

1. chạy local bằng Docker Compose mà không cần production DB/storage;
2. tạo Story từ idea;
3. lưu immutable Story Generation Policy;
4. tạo Story Bible/Arc/Characters;
5. plan và generate chapter;
6. chạy AI review;
7. approve content;
8. extract memory và commit Canon;
9. generate narration + TTS;
10. xử lý FFmpeg;
11. lưu audio MinIO local hoặc R2 production;
12. publish và phát audio;
13. lưu listening progress;
14. ghi Audit Log cho action quan trọng.
