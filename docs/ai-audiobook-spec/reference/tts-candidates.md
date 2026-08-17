# TTS Candidates — Tham khảo (chưa xác minh)

Ghi chú các giải pháp TTS tiếng Việt / mã nguồn mở để cân nhắc khi triển khai
audio pipeline thật (thay cho `MockTTS` hiện tại). Chưa kiểm chứng hoạt động —
chỉ lưu lại để tham khảo, cần đánh giá kỹ trước khi dùng.

## 1. KhanhTTS-OmniVoice

- URL: https://huggingface.co/kjanh/KhanhTTS-OmniVoice
- Loại: model TTS tiếng Việt trên Hugging Face.
- Ghi chú: cần kiểm tra license, chất lượng giọng, khả năng self-host, và
  cách tích hợp (inference API / local).

## 2. F5-TTS-Vietnamese

- URL: https://github.com/nguyenthienhy/F5-TTS-Vietnamese
- Loại: fork/port của F5-TTS cho tiếng Việt.
- Ghi chú: F5-TTS là dòng TTS zero-shot (clone giọng từ mẫu ngắn). Cần lưu ý
  vấn đề bản quyền giọng nói (spec mục 78: không clone giọng người thật khi
  chưa có quyền).

## 3. vietnormalizer

- URL: https://github.com/nghimestudio/vietnormalizer
- Loại: công cụ chuẩn hóa văn bản tiếng Việt (số, ngày, chữ viết tắt, đơn vị...)
  trước khi đưa vào TTS.
- Ghi chú: hữu ích cho bước "Narration Script" — chuẩn hóa text trước khi
  segment + synthesize để phát âm đúng.

## 4. Piper

- URL: https://github.com/rhasspy/piper
- Loại: TTS neural mã nguồn mở, chạy local nhanh, nhẹ (CPU), có nhiều voice.
- Ghi chú: cần kiểm tra có voice tiếng Việt chất lượng tốt hay không. Ưu điểm
  là self-host dễ, latency thấp, không phụ thuộc quota bên ngoài.

---

## Ghi chú tích hợp

- Tất cả đều nằm sau interface `TTSProvider` trong `backend/internal/audio/tts.go`
  — không hard-code provider vào domain (theo spec mục 77, 78).
- Cần đánh giá theo tiêu chí trong `baseline/05-audio-tts-pipeline.md`:
  giọng kể chuyện tự nhiên, ấm, tốc độ vừa/chậm, ngắt nghỉ rõ, không giống
  quảng cáo / bản tin / recap phim.
- `vietnormalizer` nên được dùng ở bước chuẩn hóa script, độc lập với việc
  chọn TTS engine nào.
