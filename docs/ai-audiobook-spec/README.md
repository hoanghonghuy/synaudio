# AI Audiobook Platform — Specification Index

> Bộ specification nền tảng cho ứng dụng truyện audio có nội dung và giọng đọc được tạo bằng AI.

## Specification Status

The project specification is finalized and ready for implementation.

### Effective source of truth

Implementation must read the following documents in order:

1. `spec_final.md`
2. `SPEC-AMENDMENT-001-POST-VERIFICATION.md`

The amendment supersedes only the sections and decisions that it explicitly
clarifies or corrects. All other sections of `spec_final.md` remain
authoritative.

The `checkpoints/` and `baseline/` documents are historical reference
materials. They must not override decisions recorded in the effective
specification above.

The effective specification freezes the Product, Business Rules, Domain,
State Machines, AI Workflow, Canon/Memory, Auth/RBAC, Listener/Admin
Workflow, and Content Governance decisions. Architecture, physical database
model, API, job/worker model, security, deployment, and implementation plan
are selected for implementation.

### Change control

Do not silently rewrite finalized decisions. Future changes must be
classified as one of:

- **Configuration Change** — operational/configuration adjustment without architecture redesign.
- **Implementation Detail** — implementation choice recorded in ADR or code documentation.
- **Domain Change** — explicitly documents the new requirement, affected invariant, migration impact, and compatibility impact.

### Implementation status

**Status:** Ready for implementation  
**Current phase:** Phase 0 — Foundation  
**Next phase:** Phase 1 — Identity + Story Foundation  

No additional Product/Business discovery round is required before starting
Phase 0 or Phase 1.

## Mục tiêu

Spec mô tả đồng thời:

1. **Target Architecture** — hệ thống cuối cùng muốn trở thành gì.
2. **Initial/MVP Architecture** — phần nào được triển khai trước.
3. **Evolution Path** — khi nào tách worker, queue, microservice, semantic memory và các capability nâng cao.

## Tài liệu

### Final implementation specification

| File | Vai trò |
|---|---|
| `spec_final.md` | Final System Specification, source of truth chính |
| `SPEC-AMENDMENT-001-POST-VERIFICATION.md` | Correction/clarification bắt buộc áp dụng sau `spec_final.md` |

### Baseline (reference only — do not edit)

| File | Nội dung |
|---|---|
| `baseline/00-project-overview.md` | Tổng quan dự án, bài toán và mục tiêu |
| `baseline/01-product-vision.md` | Product vision, role, user journey và trải nghiệm |
| `baseline/02-system-architecture.md` | Kiến trúc hệ thống tổng thể |
| `baseline/03-domain-and-data-model.md` | Domain model và schema logical |
| `baseline/04-ai-story-pipeline.md` | Pipeline Story Architect → Writer → Review |
| `baseline/05-audio-tts-pipeline.md` | Narration/TTS/FFmpeg/audio delivery |
| `baseline/06-infrastructure-and-deployment.md` | Production infrastructure và deployment |
| `baseline/07-backend-boundaries.md` | Module, boundary và evolution path backend |
| `baseline/08-mvp-scope-and-roadmap.md` | Target capability + implementation phases |
| `baseline/09-non-functional-requirements.md` | Security, reliability, performance, maintainability |
| `baseline/10-story-canon-and-memory.md` | Canon, Story Memory Engine, Context Builder, retcon |
| `baseline/11-audit-and-provenance.md` | Audit log, provenance, actor model và data history |
| `baseline/12-environments-and-local-development.md` | Docker Compose, local/dev/prod isolation, MinIO, mock providers |
| `baseline/13-story-planning-and-generation-policy.md` | FINITE/OPEN_ENDED, immutable Story Contract, duration policy |

### Historical checkpoints (reference only)

| File | Vai trò |
|---|---|
| `checkpoints/01-consolidated-specification.md` | Consolidated checkpoint lịch sử |
| `checkpoints/02-consolidated-specification.md` | Consolidated checkpoint lịch sử |

## Stack MVP đã chốt

- Frontend: Vue 3 + Vite
- Frontend hosting: Vercel
- Backend: Golang
- Initial backend architecture: Modular Monolith
- Backend hosting: Render
- Database production: Neon PostgreSQL
- Database local: PostgreSQL trong Docker Compose
- Object storage production: Cloudflare R2
- Object storage local: MinIO
- Story AI: provider abstraction
- TTS chính ban đầu: Gemini TTS
- TTS fallback: Google Cloud / Azure / FPT.AI
- Audio processing: FFmpeg
- Local orchestration: Docker Compose
- Initial job system: PostgreSQL-backed jobs
- Future: worker separation, task queue, microservices, semantic retrieval

## Core principles đã chốt

1. Audio được generate trước khi publish; không generate lại mỗi lần Play.
2. Backend không proxy audio lớn; client stream trực tiếp từ object storage.
3. Story text và narration script là hai artifact khác nhau.
4. Story generation có Canon và Story Memory do backend quản lý.
5. AI không được coi lịch sử chat/model memory là source of truth.
6. Draft không được mutate Official Canon.
7. Canon update xảy ra sau khi content được approve và Memory Extractor hoàn tất.
8. Story hỗ trợ cả FINITE và OPEN_ENDED.
9. Chapter có hard minimum duration mặc định 20 phút và target mặc định 30 phút.
10. Story Generation Policy được resolve khi tạo Story và immutable về sau.
11. Workflow settings như batch generation được phép thay đổi.
12. Major creative decisions phải qua proposal + impact analysis + admin decision.
13. Retcon được hỗ trợ nhưng là dangerous operation và phải có impact/repair plan.
14. Audit log và provenance là capability core.
15. Local development mặc định không được truy cập production DB/storage.
16. Spec mô tả đầy đủ target architecture ngay cả khi implementation được chia phase.

## Trạng thái

**Phase:** Phase 0 — Foundation  
**Status:** Final specification frozen; implementation ready  
**Next major milestone:** Hoàn tất repository/infrastructure foundation trước khi bắt đầu Phase 1.
