# 02 — System Architecture

## 1. Kiến trúc tổng thể

```text
                       ┌────────────────────┐
                       │     Text Models    │
                       └──────────┬─────────┘
                                  │
                                  │ Story generation
                                  ▼
┌────────────────┐       ┌────────────────────┐
│ Vue 3 + Vite   │──────▶│     Go Backend     │
│    Vercel      │       │      Render        │
└───────┬────────┘       └──────┬──────┬──────┘
        │                       │      │
        │                       │      │
        │                       │      ▼
        │                       │   ┌──────────────┐
        │                       │   │ TTS Provider │
        │                       │   └──────┬───────┘
        │                       │          │
        │                       ▼          │
        │                 ┌───────────┐    │
        │                 │   Neon    │    │
        │                 │PostgreSQL │    │
        │                 └───────────┘    │
        │                                  │ audio
        │                                  ▼
        │                           ┌───────────────┐
        └──────────────────────────▶│Cloudflare R2 │
               audio stream        └───────────────┘
```

---

## 2. Thành phần

### 2.1 Vue frontend

Trách nhiệm:

- browse truyện;
- story detail;
- chapter list;
- audio player;
- login sau này;
- favorite;
- listening progress;
- admin UI sau này.

Frontend không được chứa API key của:

- Gemini;
- FPT;
- Azure;
- R2 secret;
- database.

---

## 3. Go backend

Backend là orchestration layer.

Trách nhiệm:

- REST API;
- authentication;
- domain logic;
- CRUD story/chapter;
- generation jobs;
- gọi AI provider;
- gọi TTS provider;
- upload object;
- generate signed/public URL;
- lưu metadata;
- progress/favorite/history.

Backend không nên:

- proxy toàn bộ audio stream;
- lưu MP3 trên filesystem lâu dài;
- sinh TTS trực tiếp trong request Play.

---

## 4. PostgreSQL

PostgreSQL chỉ lưu structured data.

Ví dụ:

```text
Story metadata
Chapter metadata
Story content
Narration script
Audio metadata
Generation job status
User progress
Favorites
```

Không lưu:

```text
MP3 binary
cover image binary
large media blob
```

---

## 5. Cloudflare R2

R2 lưu:

```text
audio/
  stories/
    {story_id}/
      chapters/
        {chapter_id}/
          narration-v1.mp3

covers/
  stories/
    {story_id}/cover.webp
```

Object storage chịu trách nhiệm media delivery.

---

## 6. AI provider layer

Không viết business logic phụ thuộc trực tiếp Gemini.

Nên có abstraction:

```go
type StoryGenerator interface {
    GenerateOutline(...)
    GenerateChapter(...)
    GenerateNarrationScript(...)
}

type TTSProvider interface {
    GenerateSpeech(...)
}
```

Implementation:

```text
GeminiStoryGenerator
GeminiTTSProvider
GoogleCloudTTSProvider
AzureTTSProvider
FPTTTSProvider
```

---

## 7. Audio delivery

### Không dùng

```text
Browser
 ↓
Go Backend
 ↓
R2
 ↓
Go Backend
 ↓
Browser
```

### Dùng

```text
Browser
 ↓ metadata request
Go Backend

Browser
 ↓ audio request
R2
```

Backend chỉ trả:

```json
{
  "chapterId": "...",
  "audioUrl": "...",
  "duration": 1432
}
```

---

## 8. Generation architecture

Generation không nên gắn trực tiếp vào request kéo dài.

Concept:

```text
POST /admin/chapters/{id}/generate-audio
            ↓
        create job
            ↓
      generation_jobs
            ↓
     worker/process job
            ↓
     TTS + FFmpeg + R2
            ↓
       update status
```

MVP có thể chạy worker đơn giản cùng service trước.

Sau này có thể tách:

```text
API Service
Worker Service
Scheduler
Queue
```

---

## 9. Health check

Endpoint:

```text
GET /healthz
```

Response:

```json
{
  "status": "ok"
}
```

Health check cơ bản không nên:

- query database;
- gọi external provider;
- gọi object storage.

Có thể bổ sung endpoint khác:

```text
GET /readyz
```

để kiểm tra dependency khi cần.

---

## 10. Kiến trúc hướng tới scale

V1:

```text
Vue
Go API + Worker
Neon
R2
```

Khi lớn:

```text
Vue / CDN
      │
API Gateway
      │
Go API
 ├── PostgreSQL
 ├── Redis
 └── Queue
       │
    Workers
    ├── Story AI
    ├── TTS
    └── FFmpeg
       │
       R2/CDN
```

## 11. Initial vs target architecture

Initial backend là **Go Modular Monolith** với boundary rõ cho Story, Canon, Memory, Generation, Audio và Audit. Worker, task queue và Story Memory microservice là evolution path khi có tải/độ phức tạp thực tế.

Local development dùng Docker Compose với PostgreSQL + MinIO; production dùng Neon + R2.
