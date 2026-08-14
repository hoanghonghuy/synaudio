# 13 — Story Planning and Generation Policy

## 1. Immutable Story Creation Contract

Default baseline:

```text
minimum chapter duration = 20 minutes
target chapter duration = 30 minutes
```

Code default có thể được ENV override cho Story mới.

Admin có Advanced Settings trước khi bấm Create.

Sau khi Story được tạo:

```text
resolved policy snapshot
→ immutable
```

---

## 2. Mutable Workflow Settings

Được phép đổi:

```text
batch generation size
preferred AI model
preferred TTS provider
auto review
pause points
planning horizon
```

---

## 3. Duration policy

```text
Chapter Plan Budget
      ↓
Text Estimate
      ↓
Narration Estimate
      ↓
Actual TTS Duration
```

Actual audio duration là source of truth cuối.

---

## 4. Under-duration handling

```text
AI Content Gap Analyzer
      ↓
System Validation
      ↓
Admin Decision
```

Options:

```text
expand recommended scenes
regenerate chapter
edit manually
regenerate plan
approve override
```

Override phải intentional và audit.

---

## 5. No-filler rule

Không kéo dài bằng repetition, meaningless description, fake pause, slow audio bất thường hoặc duplicated dialogue.

---

## 6. FINITE mode

AI đề xuất scope dựa trên idea/complexity/duration:

```text
A: range chapters / arcs
B: range chapters / arcs
C: range chapters / arcs
Custom
```

Admin chọn hoặc tự nhập.

Chapter count là planning range, không phải hard lock.

---

## 7. OPEN_ENDED mode

Không khóa tổng chapter nhưng vẫn bắt buộc có current planned ending / long-term destination.

OPEN_ENDED không có nghĩa AI tự viết vô hạn.

---

## 8. Planning phase

```text
ONGOING
CLOSING
FINAL_ARC
COMPLETED
```

---

## 9. Arc completion review

Trước Arc mới:

```text
Arc Completion Review
      ↓
AI Next Arc Proposal
      ↓
A/B/C + Custom
      ↓
Admin/Semi-auto decision
```

Proposal luôn được lưu.

---

## 10. FINAL_ARC rules

Major new plot thread phải warning hoặc đi qua Creative Decision Gate.

---

## 11. Planning Horizon

```text
Next chapters → detailed
Current arc   → detailed
Next arc      → medium
Future arcs   → high-level
Ending        → strategic
```

Khi Story tiến lên:

```text
rough → refine → detailed → execute
```

---

## 12. Completion

Khi completed, normal Generate Next Chapter bị disable.

Nếu muốn tiếp tục, ưu tiên Create Sequel hơn reopen story đã hoàn chỉnh.
