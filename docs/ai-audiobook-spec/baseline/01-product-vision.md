# 01 — Product Vision

## 1. Product vision

Tạo một ứng dụng truyện audio tiếng Việt nơi người dùng có thể khám phá và nghe những series truyện được sản xuất bằng pipeline AI nhưng vẫn có trải nghiệm gần với audiobook được biên tập chuyên nghiệp.

---

## 2. Giá trị cốt lõi

### Đối với người nghe

- có truyện mới thường xuyên;
- nghe được ngay trên web;
- tiếp tục từ vị trí đang nghe;
- trải nghiệm giọng đọc tự nhiên;
- có nhiều thể loại;
- không phải đọc văn bản dài.

### Đối với hệ thống

AI giúp giảm công sức ở:

- viết outline;
- tạo chapter;
- kiểm tra consistency;
- chuyển văn bản thành narration;
- tạo voice;
- hậu kỳ;
- publish.

---

## 3. Người dùng mục tiêu ban đầu

MVP tập trung vào:

- người thích nghe truyện;
- người nghe trong lúc di chuyển;
- người nghe trước khi ngủ;
- người thích truyện dài tập;
- người không muốn đọc text liên tục.

Không cần xác định quá sâu persona ở phase kỹ thuật đầu tiên.

---

## 4. Core user journey

```text
Home
 ↓
Browse stories
 ↓
Open story
 ↓
Select chapter
 ↓
Play
 ↓
Pause / seek
 ↓
Close app
 ↓
Return
 ↓
Continue listening
```

---

## 5. Core content journey

```text
Create story
 ↓
Create outline
 ↓
Generate chapter
 ↓
Review
 ↓
Create narration script
 ↓
Generate audio
 ↓
Post-process
 ↓
Upload
 ↓
Publish chapter
```

---

## 6. Chất lượng audio mong muốn

Giọng đọc phải ưu tiên:

- tự nhiên;
- không robotic;
- tốc độ vừa phải;
- có khoảng nghỉ;
- biết chuyển sắc thái giữa narration và dialogue;
- không giống quảng cáo;
- không giống MC bản tin;
- không giống voice-over review phim.

Style mục tiêu:

> Một narrator trưởng thành đang kể một câu chuyện cho người nghe.

---

## 7. Voice strategy V1

V1 chỉ dùng:

- một narrator chính;
- thay đổi tone theo scene;
- narration và dialogue có thể khác delivery nhưng không bắt buộc đổi voice.

Không nên bắt đầu bằng:

```text
Narrator
Male Lead
Female Lead
Villain
Supporting Character 1
Supporting Character 2
...
```

Lý do:

- tăng độ phức tạp prompt;
- tăng số lần gọi TTS;
- khó consistency;
- khó ghép audio;
- khó kiểm soát volume;
- tăng chi phí.

---

## 8. Voice strategy tương lai

Có thể phát triển:

```text
Scene
 ├── Narrator
 ├── Character A
 └── Character B
```

Sau đó generate từng segment và ghép lại.

---

## 9. Content safety và quyền giọng nói

Không clone hoặc mô phỏng chính xác giọng của narrator/người thật cụ thể nếu không có quyền sử dụng phù hợp.

Sản phẩm nên xây dựng:

- voice identity riêng;
- style kể truyện riêng;
- prompt preset riêng.

Điều này giúp:

- giảm rủi ro quyền cá nhân;
- dễ thương mại hóa;
- tạo nhận diện sản phẩm;
- không phụ thuộc một cá nhân ngoài hệ thống.

---

## 10. Tiêu chí thành công MVP

MVP được coi là đạt nếu:

- người dùng có thể nghe trọn một chapter;
- audio phát ổn định;
- voice đủ tự nhiên;
- có ít nhất một series demo;
- pipeline có thể sinh chapter mới mà không cần thao tác thủ công quá nhiều;
- backend không bị bandwidth audio làm quá tải;
- hạ tầng duy trì ở mức chi phí rất thấp.
