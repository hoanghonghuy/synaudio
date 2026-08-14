# 07 — Backend Boundaries

## 1. Initial architecture

**Modular Monolith** với boundary rõ để sau này tách service mà không rewrite domain.

```text
Go Application
│
├── HTTP/API
├── Story
├── Planning
├── Canon
├── Memory
├── Generation
├── Audio
├── User
├── Audit
└── Infrastructure adapters
```

---

## 2. Suggested modules

```text
internal/
  story/
  planning/
  canon/
  memory/
  generation/
  audio/
  user/
  listening/
  audit/

  platform/
    postgres/
    storage/
    ai/
    tts/
```

---

## 3. Story Memory subsystem

V1 là module cùng application:

```text
memory/
├── ContextBuilder
├── MemoryRetriever
├── FactResolver
├── HistoricalRetriever
└── ContextSnapshot
```

Generation layer không biết chi tiết SQL/pgvector/embedding.

---

## 4. Future microservice extraction

Trigger:

```text
independent scaling
complex hybrid retrieval
many worker consumers
different technology stack
failure isolation
```

Không tách chỉ vì “muốn microservice”.

---

## 5. Provider boundaries

```text
StoryAI
TTSProvider
ObjectStorage
StoryMemory
```

đều qua interface/port.

---

## 6. Job boundary

HTTP không chờ tác vụ dài.

```text
Generation intent
      ↓
GenerationRun
      ↓
GenerationJob(s)
      ↓
Worker
```

Initial job store/queue: PostgreSQL.

Future: Redis/RabbitMQ/NATS/managed queue nếu có nhu cầu.

---

## 7. API groups dự kiến

Public:

```text
GET /stories
GET /stories/{slug}
GET /stories/{storyId}/chapters
GET /chapters/{chapterId}
GET /chapters/{chapterId}/audio
```

User:

```text
PUT /me/progress/{chapterId}
GET /me/progress/{chapterId}
POST /me/favorites/{storyId}
DELETE /me/favorites/{storyId}
```

Admin:

```text
POST /admin/stories
POST /admin/stories/{id}/architect
POST /admin/stories/{id}/arcs
POST /admin/chapters/{id}/plan
POST /admin/chapters/{id}/generate
POST /admin/chapters/{id}/approve
POST /admin/chapters/{id}/narration
POST /admin/chapters/{id}/audio
POST /admin/chapters/{id}/publish
POST /admin/creative-decisions/{id}/resolve
POST /admin/retcons
```

Exact REST contract sẽ chốt sau.

---

## 8. Transaction boundaries

Canon commit phải transactional. Không để partial state update.

---

## 9. Environment-neutral application

Business code không chứa `if production` rải rác.

Bootstrap chọn adapter bằng config:

```text
STORAGE_PROVIDER=minio → MinIO
STORAGE_PROVIDER=r2    → R2
```

Tương tự AI/TTS/Queue.
