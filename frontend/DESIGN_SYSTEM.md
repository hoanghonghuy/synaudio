# Synaudio Night Studio Design System

Synaudio dùng một ngôn ngữ giao diện dark cinematic, ưu tiên trải nghiệm nghe lâu, thao tác bằng một tay và khả năng đọc rõ trên màn hình nhỏ. Mọi view dùng chung token trong `src/styles.css`; component không tự tạo màu hoặc kích thước bo góc mới.

## Nguyên tắc

- **Audio-first**: play, tiến độ, chapter và trạng thái lưu tiến độ phải nổi bật hơn metadata.
- **Dark by default**: obsidian làm nền, surface nhiều lớp tạo chiều sâu; amber chỉ dành cho hành động chính.
- **Một hệ thống**: catalog, reader, library, auth và Creator Studio dùng cùng nhịp spacing, trạng thái và focus state.
- **Mobile trước**: vùng chạm tối thiểu 44px; nav đáy trên mobile; bảng và danh sách chuyển một cột ở `<= 760px`.
- **Rõ trạng thái**: loading, empty, success, warning và error luôn có vùng hiển thị riêng, không phụ thuộc vào màu đơn lẻ.

## Token chính

| Nhóm | Token | Vai trò |
| --- | --- | --- |
| Nền | `--bg`, `--surface`, `--surface-soft`, `--surface-raised` | Canvas và các lớp panel |
| Chữ | `--ink`, `--ink-medium`, `--muted`, `--ink-faint` | Phân cấp đọc và metadata |
| Accent | `--accent`, `--accent-strong`, `--accent-soft` | Play, CTA, active state |
| Chiều sâu | `--violet`, `--violet-soft`, `--cyan`, `--cyan-soft` | Focus phụ, ambience, thông tin |
| Trạng thái | `--success`, `--warning`, `--danger`, `--error-container` | Kết quả và cảnh báo |
| Hình học | `--radius-sm`, `--radius-md`, `--radius-lg`, `--radius-full` | Input, panel, pill |

## Quy ước component

- Panel dùng `border: 1px solid var(--line)`, nền surface và `border-radius: var(--radius-lg)`.
- CTA dùng pill `var(--radius-full)`; action nguy hiểm luôn có màu `--danger` và cần confirmation ở layer logic.
- Input/select/textarea cao ít nhất `48px`; button/link tương tác cao ít nhất `44px`.
- Nội dung truyện dùng `var(--font-reading)`; điều khiển và metadata dùng `var(--font-ui)`.
- Không dùng inline style cho màu hoặc layout; không đặt hex/radius tuỳ ý trong component.

## Responsive checkpoints

- `360px–760px`: bottom navigation, chapter drawer, single-column forms, horizontal chips.
- `768px–1039px`: hai/ba cột linh hoạt, panel không bị dính quá sát viewport.
- `1040px+`: grid catalog bốn cột, reader có chapter rail, admin có workspace hai cột.
