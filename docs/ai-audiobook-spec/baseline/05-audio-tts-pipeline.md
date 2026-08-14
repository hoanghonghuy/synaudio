# 05 — Audio and TTS Pipeline

## 1. TTS strategy

Provider ưu tiên ban đầu:

```text
Primary:
Gemini 3.1 Flash TTS

Fallback:
Google Cloud TTS
Azure Speech
FPT.AI
```

Provider cụ thể có thể thay đổi nếu:

- quota thay đổi;
- chất lượng voice thay đổi;
- pricing thay đổi;
- API preview thay đổi.

Do đó không hard-code provider vào domain.

---

## 2. Mục tiêu voice

Narrator V1:

```text
Vietnamese
adult
warm
natural
storytelling
controlled emotion
medium/slow pace
clear pauses
```

Tránh style:

```text
advertisement
news anchor
movie recap
hyperactive short-video voice
```

---

## 3. Narration direction

Prompt TTS có thể chứa:

```text
Audio profile
Scene context
Narrator persona
Pace
Tone
Emotion
Dialogue direction
Pause direction
```

Ví dụ concept:

```text
Read in Vietnamese as a professional audiobook narrator.

Voice:
- mature
- warm
- low energy
- natural

Style:
- storytelling
- calm
- cinematic but restrained

Do not sound like:
- an advertisement
- a news report
- a movie recap

Scene:
The protagonist enters an abandoned house at midnight.
```

---

## 4. Không generate một chapter cực dài trong một request

Nên chia narration thành chunk.

Ví dụ:

```text
Chapter
  ↓
Scene 1
  ↓
Segment 1
Segment 2
Segment 3
```

Chunking giúp:

- tránh API limits;
- retry dễ;
- giảm tổn thất nếu request fail;
- kiểm soát scene;
- ghép audio dễ hơn;
- support multi-speaker về sau.

---

## 5. Pipeline hoàn chỉnh

```text
Chapter Content
       ↓
Narration Script
       ↓
Segmenter
       ↓
TTS Segments
       ↓
Temporary audio
       ↓
FFmpeg
       ↓
Normalize
       ↓
Join
       ↓
Encode MP3
       ↓
Upload R2
       ↓
AudioAsset READY
```

---

## 6. FFmpeg

FFmpeg dùng cho:

- concatenate;
- normalize loudness;
- convert format;
- set bitrate;
- inspect duration;
- optional silence trimming.

MVP target:

```text
MP3
64–96 kbps
mono hoặc stereo tùy provider
```

Voice audiobook không cần bitrate quá cao.

---

## 7. Audio naming

Không dùng filename dựa vào title chưa sanitize.

Nên dùng ID:

```text
audio/stories/{story_id}/chapters/{chapter_id}/v1.mp3
```

Nếu có voice version:

```text
audio/stories/{story_id}/chapters/{chapter_id}/{voice_id}/v1.mp3
```

---

## 8. Retry

Nếu một segment fail:

```text
retry segment
```

không cần generate lại toàn chapter.

Thông tin nên lưu trong job:

```text
attempts
provider
model
error
```

---

## 9. Idempotency

Nếu audio đã READY:

```text
GenerateAudio(chapter_id)
```

không nên tự tạo thêm object trùng khi endpoint bị gọi lại.

Có thể:

- reject;
- require `force=true`;
- tạo version mới.

---

## 10. Streaming

Frontend phát URL từ R2.

Audio player cần support:

```text
play
pause
seek
duration
current time
playback speed
resume
```

Tương lai:

```text
0.75x
1.0x
1.25x
1.5x
2.0x
```

---

## 11. Vì sao không generate real-time

Không dùng:

```text
Play
 ↓
Call TTS
 ↓
Wait
 ↓
Audio
```

Vì:

- latency;
- cost;
- provider quota;
- inconsistent result;
- khó seek;
- dễ fail;
- tốn CPU/network.

Đúng:

```text
Publish pipeline
 ↓
Generate once
 ↓
Store
 ↓
Play many times
```

---

## 12. Multi-speaker tương lai

Architecture:

```text
Narration Script
      ↓
Speaker segmentation
      ↓
Narrator segment
Character A segment
Character B segment
      ↓
TTS
      ↓
Mix/Join
```

Không đưa vào MVP.
