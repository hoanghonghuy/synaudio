# 06 — Infrastructure and Deployment

## 1. Production baseline

```text
Frontend      Vercel
Backend       Render
Database      Neon PostgreSQL
Storage       Cloudflare R2
AI/TTS        External providers
```

---

## 2. Initial deployment architecture

```text
Vue / Vercel
     │
     ▼
Go Modular Monolith / Render
     │
     ├── Neon PostgreSQL
     ├── Cloudflare R2
     ├── AI APIs
     └── TTS APIs
```

---

## 3. Target evolution architecture

```text
Web App
   │
API Layer
   │
   ├── Story Core
   ├── User Domain
   └── Job System
           │
        Task Queue
           │
    ┌──────┴─────────┐
    ▼                ▼
AI Workers       Audio Workers
    │                │
Story Memory        FFmpeg
    │                │
PostgreSQL       Object Storage
```

Future services có thể gồm API Service, Story Generation Service, Story Memory Service, Audio/TTS Service, Worker Pool, Task Queue và Notification Service.

---

## 4. Worker separation trước microservice memory

Expected evolution:

```text
Phase 1:
Go App
├── API
└── Worker

Phase 2:
Go API
   │
DB / Queue
   │
Generation Worker

Phase 3:
API
Generation Workers
Story Memory Service
Task Queue
```

---

## 5. Vercel / Render / Neon / R2

- Vue 3 + Vite → Vercel.
- Go API → Render.
- PostgreSQL production → Neon.
- Media production → Cloudflare R2.
- Health endpoint `/healthz` cực nhẹ, không query dependency.
- Backend không proxy audio lớn.

---

## 6. Filesystem

Hosting filesystem chỉ dùng temporary artifacts:

```text
Generate
 ↓
/tmp
 ↓
FFmpeg
 ↓
Upload R2
 ↓
Delete temp
```

---

## 7. Migration

Local và production dùng cùng migration history.

---

## 8. Secrets

Không commit `.env`, API key, DB URL thật, R2 secret hoặc private key.

---

## 9. Logging / backup

Structured log nên có `request_id`, `story_id`, `chapter_id`, `generation_run_id`, `job_id`, provider và status.

Backup target:

```text
Code      → Git
DB        → managed backup/export
Audio     → object storage
Prompts   → version control
Story     → export/recovery capability
Audit     → retained by policy
```

---

## 10. Local development

Chi tiết ở `12-environments-and-local-development.md`.
