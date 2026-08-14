# 12 — Environments and Local Development

## 1. Nguyên tắc

Development dùng Docker Compose từ ngày đầu.

Local data mặc định không được truy cập Neon/R2 production hoặc production Canon/user data.

---

## 2. Environment model

### Local Development

```text
Docker Compose
├── PostgreSQL
├── MinIO
├── Go API
├── Worker / in-process worker
└── Vue optional
```

AI/TTS có thể mock hoặc gọi real external provider.

### Production-like Local

Docker Compose với config gần production để test integration.

### Production

```text
Vue        → Vercel
Go API     → Render / future platform
PostgreSQL → Neon
Storage    → Cloudflare R2
AI/TTS     → external providers
```

---

## 3. Local PostgreSQL

Dữ liệu Story/Chapter/Canon/Facts/Characters/Jobs/Users/Audit nằm trong PostgreSQL local + Docker volume.

---

## 4. Local object storage

Production dùng R2. Local dùng MinIO để không làm bẩn production bucket và vẫn giữ S3-compatible API.

---

## 5. FFmpeg

```text
TTS
 ↓
temp file
 ↓
FFmpeg
 ↓
MinIO
```

---

## 6. AI/TTS development modes

### Real provider

```text
AI_PROVIDER=...
TTS_PROVIDER=...
```

Data vẫn local; chỉ inference đi Internet.

### Mock mode

```text
AI_PROVIDER=mock
TTS_PROVIDER=mock
```

Dùng để test UI/API/job/storage/player mà không tốn quota.

### Future local provider

Architecture cho phép `AI_PROVIDER=local`, nhưng không phải MVP requirement.

---

## 7. Docker Compose evolution

Initial:

```text
postgres
minio
api
frontend optional
```

Later:

```text
postgres
minio
api
worker
frontend
queue if justified
```

---

## 8. Config precedence

```text
Code Default
    ↓
ENV Override
    ↓
Create Story UI Advanced Settings
    ↓
Create Story
    ↓
Immutable Story Generation Policy Snapshot
```

ENV đổi sau này không ảnh hưởng Story cũ.

---

## 9. Environment files

Commit template:

```text
.env.example
.env.production.example
```

Không commit secret.

---

## 10. Adapter bootstrap

Không rải `if production` trong business code.

```text
STORAGE_PROVIDER=minio → MinIO adapter
STORAGE_PROVIDER=r2    → R2 adapter
```

Tương tự AI/TTS/Queue.

---

## 11. Development safety guard

Default:

```text
ALLOW_REMOTE_DATABASE_IN_DEV=false
ALLOW_REMOTE_STORAGE_IN_DEV=false
```

Nếu `APP_ENV=development` mà config trỏ production remote dependency, startup phải fail trừ explicit override.

---

## 12. Production safety guard

Production có thể reject:

```text
AI_PROVIDER=mock
TTS_PROVIDER=mock
unsafe debug mode
local-only storage
```

---

## 13. Migration / seed

Local và production dùng cùng migration history.

Dev seed riêng:

```text
demo admin
demo story
mock audio
```

không chạy production.

---

## 14. Future worker/queue

Initial jobs trong PostgreSQL.

Future có dedicated worker, task queue, retry/dead-letter/priority/scheduling nếu thực sự cần.
