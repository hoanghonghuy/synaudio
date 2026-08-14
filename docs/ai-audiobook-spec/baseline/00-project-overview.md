# 00 — Project Overview

## 1. Tên tạm thời

**AI Audiobook Platform**

Tên thương mại sẽ được quyết định sau. Trong tài liệu kỹ thuật, hệ thống được gọi là `AI Audiobook Platform`.

---

## 2. Ý tưởng

Xây dựng một nền tảng nghe truyện audio trong đó:

- nội dung truyện được AI hỗ trợ tạo;
- truyện được chuyển thành kịch bản đọc phù hợp với audiobook;
- audio được sinh bằng AI Text-to-Speech;
- người dùng có thể nghe theo truyện, chương và tiếp tục vị trí đang nghe;
- hệ thống tự động hóa phần lớn quy trình từ viết truyện đến publish audio.

Mục tiêu không phải tạo giọng kiểu đọc tin, quảng cáo hay review phim.

Trải nghiệm mong muốn là:

> Người nghe có cảm giác đang nghe một narrator thực sự kể truyện, với nhịp nghỉ, cảm xúc và cách diễn phù hợp ngữ cảnh.

---

## 3. Bài toán chính

Một hệ thống kiểu này có bốn bài toán độc lập:

### 3.1 Story generation

AI cần tạo được truyện:

- có cốt truyện nhất quán;
- nhân vật ổn định;
- chia chương hợp lý;
- hạn chế mâu thuẫn giữa các chương;
- có khả năng tiếp tục series dài.

### 3.2 Narration adaptation

Văn bản dùng để đọc không nên giống hoàn toàn văn bản hiển thị.

Hệ thống cần chuyển:

```text
Story content
      ↓
Narration-ready script
```

Narration script có thể chứa:

- điểm ngắt nghỉ;
- phân đoạn;
- chỉ dẫn tone;
- lời dẫn;
- lời thoại;
- nhấn mạnh;
- audio direction.

### 3.3 Text-to-Speech

TTS cần:

- hỗ trợ tiếng Việt tốt;
- giọng tự nhiên;
- đọc được truyện dài;
- có khả năng kiểm soát tone/pace/style;
- chi phí đủ thấp;
- có API để tự động hóa.

### 3.4 Distribution

Không được sinh TTS lại mỗi lần người dùng nghe.

Quy trình đúng:

```text
Generate một lần
      ↓
Encode audio
      ↓
Store object
      ↓
Stream nhiều lần
```

---

## 4. Mục tiêu MVP

MVP cần chứng minh được ba giả thuyết:

1. AI có thể tạo nội dung đủ hấp dẫn để người dùng nghe tiếp.
2. TTS tiếng Việt có thể tạo trải nghiệm gần audiobook thật.
3. Kiến trúc có thể phục vụ audio với chi phí thấp.

MVP chưa cần trở thành một nền tảng nội dung hoàn chỉnh.

---

## 5. Kiến trúc công nghệ đã chọn

```text
Frontend
Vue 3 + Vite
      │
      ▼
Vercel

Backend
Golang
      │
      ▼
Render

Database
PostgreSQL
      │
      ▼
Neon

Object Storage
Cloudflare R2

AI Story
Text models

TTS
Gemini TTS
+ fallback providers

Post-processing
FFmpeg
```

---

## 6. Nguyên tắc sản phẩm

### Nội dung

- Một `Story` có nhiều `Chapter`.
- Một chapter có bản text để đọc và narration script để sinh audio.
- Chỉ chapter đã hoàn thiện mới được publish.

### Audio

- Audio được pre-generate.
- Một chapter có thể có nhiều audio version trong tương lai.
- Client không gọi trực tiếp TTS provider.

### AI

- AI generation là backend responsibility.
- Provider có thể thay đổi.
- Prompt và model phải được version hóa về sau.

---

## 7. Không nằm trong MVP

Các tính năng sau chưa phải ưu tiên:

- marketplace cho tác giả;
- thanh toán;
- subscription;
- clone giọng người thật;
- 10+ voice trong một chapter;
- social network;
- live streaming;
- audiobook generation real-time khi user bấm Play;
- recommendation engine phức tạp;
- mobile native app.

---

## 8. Định hướng dài hạn

Nếu MVP thành công, hệ thống có thể phát triển thành một pipeline xuất bản audiobook tự động:

```text
Idea
 ↓
Story Outline
 ↓
Chapter Generation
 ↓
Editorial Review
 ↓
Narration Script
 ↓
TTS Generation
 ↓
Audio Post-processing
 ↓
Publishing
 ↓
Listening Analytics
```

## 9. Core AI Story subsystems đã chốt

```text
Story Planning
Canon
Story Memory Engine
Creative Decision System
Quality Gates
Audit & Provenance
Retcon Impact Analysis
```

Chi tiết nằm trong `10-story-canon-and-memory.md`, `11-audit-and-provenance.md`, `13-story-planning-and-generation-policy.md`.
