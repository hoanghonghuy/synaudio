# Hệ Thống Thiết Kế Thủy Mặc (Thuỷ Mặc Design System)

> **Mục tiêu**: Định hình và cố định (lock-in) toàn bộ quy chuẩn thiết kế UI/UX, bảng màu, thư pháp, thước đo khoảng cách (spacing rhythm), bo góc (radii) và chuẩn chạm (touch targets) cho nền tảng Synaudio, đảm bảo mã nguồn giao diện luôn đồng nhất và không bị rời rạc trong tương lai.

---

## 1. Triết Lý Thiết Kế (Design Philosophy)

Synaudio lấy cảm hứng từ nghệ thuật **Thủy Mặc Á Đông (Ink Wash Painting)** kết hợp cùng tiêu chuẩn kỹ thuật số hiện đại (**Mobile-First Digital Craftsmanship**):
- **Giấy Xuyến Chỉ (Rice Paper Foundation)**: Nền giấy ấm áp, mộc mạc (`#f4f0e8`), giảm mỏi mắt khi đọc lâu và tạo cảm giác trang nhã như một cuốn cổ thư.
- **Ngũ Sắc Chi Mặc (Calligraphy Ink Spectrum)**: Các cấp độ mực thư pháp từ đậm đặc than củi (`#1a1817`) đến mực nhạt khói sương (`#6e6862`), tạo chiều sâu phân cấp thị giác tự nhiên.
- **Ấn Triện Chu Sa (Vermilion Seal Stamp)**: Sắc đỏ chu sa (`#b23a2b`) đóng vai trò điểm nhấn linh hồn (accent), tượng trưng cho con dấu triện truyền thống, hướng sự chú ý vào các hành động cốt lõi (Play, Đọc truyện, Tạo nội dung).
- **Thanh Sơn & Trúc Mặc (Pine Wash)**: Tông xanh rêu trầm (`#2e3e37`) điểm xuyết trạng thái cân bằng, chiêm nghiệm.

---

## 2. Bảng Màu & Biến CSS (Color Tokens)

Tất cả thành phần giao diện **phải** sử dụng biến CSS token định nghĩa trong `:root` tại `frontend/src/styles.css`. Tuyệt đối không tự ý viết mã hex thô (`#...`) vào các component.

```css
:root {
  /* Giấy xuyến chỉ (Canvas & Surface) */
  --bg: #f4f0e8;               /* Nền chính toàn trang */
  --surface: #fbf9f5;          /* Thẻ card, panel, ô nhập liệu */
  --surface-soft: #ede6da;     /* Nền phụ, hover background, code block */
  --surface-tint: #e4ded2;     /* Nền nhấn nhẹ */

  /* Mực thư pháp (Ink Typography) */
  --ink: #1a1817;              /* Chữ chính, tiêu đề cấp cao */
  --ink-dark: #282523;         /* Văn bản đậm */
  --ink-medium: #45413d;       /* Nội dung đọc, đoạn văn */
  --muted: #6e6862;            /* Chú thích, phụ đề, thời gian */
  --ink-faint: #9c958d;        /* Đường kẻ mờ, placeholder */

  /* Ấn triện Chu Sa (Accent & Action) */
  --accent: #b23a2b;           /* Nút chính, liên kết nổi bật, con dấu */
  --accent-strong: #8c271b;    /* Hover / active của nút chu sa */
  --accent-soft: rgba(178, 58, 43, 0.08);   /* Nền highlight chu sa nhạt */
  --accent-border: rgba(178, 58, 43, 0.32); /* Viền highlight chu sa */
  --seal-stamp: #b23a2b;

  /* Trúc mặc (Pine & Auxiliary) */
  --ink-pine: #2e3e37;
  --ink-pine-soft: rgba(46, 62, 55, 0.08);

  /* Trạng thái (Status Colors) */
  --amber: #b86b1b;            /* Cảnh báo, trạng thái chờ, nháp */
  --danger: #b23a2b;           /* Lỗi, nút xoá, thao tác huỷ */
  --error: #b23a2b;
  --error-container: rgba(178, 58, 43, 0.08);

  /* Đường nét & Bóng đổ lan tỏa (Lines & Diffused Shadows) */
  --line: rgba(45, 38, 32, 0.12);        /* Viền card, phân cách chuẩn */
  --line-soft: rgba(45, 38, 32, 0.06);   /* Phân cách cực nhẹ */
  --line-strong: rgba(45, 38, 32, 0.22); /* Viền ô nhập liệu khi active */
  --shadow: 0 4px 20px rgba(35, 28, 22, 0.05), 0 1px 3px rgba(35, 28, 22, 0.03);
  --shadow-lg: 0 12px 36px rgba(35, 28, 22, 0.08), 0 2px 8px rgba(35, 28, 22, 0.04);
  --shadow-seal: 0 3px 12px rgba(178, 58, 43, 0.22);
}
```

---

## 3. Thư Pháp & Kiểu Chữ (Typography)

Synaudio sử dụng hệ thống phông chữ kép phân định rõ ràng giữa **chất văn học** và **tương tác điều khiển**:

| Token | Giá Trị Font Family | Mục Đích Sử Dụng |
| :--- | :--- | :--- |
| `--font-heading` | `"Noto Serif", Georgia, "Songti SC", "SimSun", serif` | Tiêu đề trang (`h1`), tiêu đề truyện, tiêu đề chương, nút hành động chính chữ thư pháp (`.primary-link`, `.seal-btn`). |
| `--font-reading` | `"Noto Serif", Georgia, "Times New Roman", serif` | Nội dung chương truyện đọc (`StoryReader`), đoạn trích dẫn văn học, trải nghiệm thưởng thức câu chữ. |
| `--font-ui` | `"Plus Jakarta Sans", "Segoe UI", -apple-system, BlinkMacSystemFont, "Noto Sans", sans-serif` | Toàn bộ giao diện thao tác người dùng: thanh điều hướng, form nhập liệu, bảng danh sách quản trị, số liệu audio player. |

---

## 4. Thước Đo Không Gian & Padding (Rhythmic Spacing Scale)

Synaudio áp dụng thang đo khoảng cách bội số chuẩn (8pt rhythm) để loại bỏ hoàn toàn các giá trị padding/margin ngẫu hứng:

| Token | Kích Thước | Ứng Dụng |
| :--- | :--- | :--- |
| `--space-3xs` | `2px` | Khoảng hở vi mô, viền nét phụ |
| `--space-2xs` | `4px` | Khoảng cách giữa icon và label nhỏ |
| `--space-xs` | `8px` | Khoảng cách phần tử inline, badge, chip gap |
| `--space-sm` | `12px` | Padding trong của dropdown item, sub-panel, pre/code |
| `--space-md` | `16px` | Khoảng cách chuẩn giữa các cột/hàng nhỏ, padding card nhỏ |
| `--space-lg` | `24px` | Padding trong của `.panel`, card lớn, spacing giữa các block |
| `--space-xl` | `32px` | Khoảng cách giữa các section trong cùng một trang |
| `--space-2xl` | `48px` | Margin lớn phân cách các khối nội dung độc lập |
| `--space-3xl` | `64px` | Khoảng cách trên/dưới của trang chủ và banner |

### Khung Chứa Trang (Page Container)
```css
.page {
  width: min(calc(100% - 2 * var(--gutter)), var(--content-max));
  margin: 0 auto;
  padding: 2.5rem 0 5rem; /* Desktop: 40px trên, 80px dưới */
}

@media (max-width: 720px) {
  .page {
    padding: 1.5rem 0 4.5rem; /* Mobile: tối ưu vùng nhìn thấy phía trên */
  }
}
```

---

## 5. Ngôn Ngữ Hình Học & Bo Góc (Geometry & Radii)

Synaudio sử dụng hình học bo tròn mềm mại như nét cọ bút lông:

1. **Dạng Con Nhộng (Pills - `--radius-full: 999px`)**:
   - Dành cho **mọi nút hành động tương tác chính**: nút tìm kiếm, nút nghe/đọc truyện, nút chuyển tab bộ lọc (`.chip-pill`, `.sort-chip`), nút lưu/huỷ, con dấu triện.
   - Nút `.primary-link` và `.seal-btn` mang viền tròn hoàn hảo tạo cảm giác như một viên ngọc hoặc con dấu mực đỏ.
2. **Dạng Thẻ Bìa Giấy (Cards / Panels - `--radius-lg: 16px`)**:
   - Dành cho toàn bộ thẻ nội dung: bìa truyện, danh sách chương, audio player bar, hộp điều khiển quản trị studio, modal hội thoại.
3. **Dạng Ô Nhập Liệu (Input Fields - `--radius-md: 10px`)**:
   - Dành cho các ô input text, select box, textarea, khối trích dẫn code `pre`.
4. **Viền Chi Tiết Nhỏ (`--radius-sm: 4px`)**:
   - Sử dụng cho thanh trượt tiến trình (progress bar), thanh âm lượng.

---

## 6. Tiêu Chuẩn Vùng Chạm Mobile-First (Touch Targets & Accessibility)

Nhằm đảm bảo trải nghiệm sử dụng trên điện thoại di động mượt mà, không bấm hụt:
- **Chiều cao tối thiểu chuẩn**: Tất cả các ô nhập liệu (`input`, `select`), nút chính (`button`, `.primary-link`), tab điều hướng phải đạt tối thiểu **`44px - 48px`**.
- **Chips & Lọc nhanh**: Tối thiểu **`36px - 40px`** với padding cân đối (`padding: 0 14px` đến `0 16px`).
- **Focus States**: Mọi liên kết và nút bấm khi focus bằng bàn phím phải hiển thị viền tương phản sắc nét:
  ```css
  :focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
  ```
- **Tap Highlight**: Loại bỏ viền nháy xám mặc định của trình duyệt di động: `-webkit-tap-highlight-color: transparent`.

---

## 7. Quy Tắc Bắt Buộc Khi Lập Trình (Mandatory Implementation Rules)

Bất kỳ lập trình viên hoặc AI agent nào khi thêm mới hoặc chỉnh sửa giao diện **bắt buộc tuân thủ 6 điều răn**:

1. **Không Viết Mã Màu Hex Tuỳ Hứng**:
   * *Sai*: `background: #ffffff; color: #000; border: 1px solid #ddd;`
   * *Đúng*: `background: var(--surface); color: var(--ink); border: 1px solid var(--line);`
2. **Không Viết Giá Trị Bo Góc (Border-Radius) Bừa Bãi**:
   * *Sai*: `border-radius: 6px;` hoặc `border-radius: 14px;`
   * *Đúng*: Chỉ dùng các token chuẩn: `var(--radius-sm)`, `var(--radius-md)`, `var(--radius-lg)`, `var(--radius-full)`.
3. **Không Sử Dụng Inline Styles Cho Bố Cục Hoặc Màu Sắc**:
   * *Sai*: `<div style="padding: 20px; color: red;">`
   * *Đúng*: Viết class ngữ nghĩa hoặc scoped style sử dụng biến CSS hệ thống.
4. **Đồng Nhất Nút Hành Động Theo Dạng Pill**:
   * Tất cả nút Call-to-Action (CTA), filter chips, tab filter phải bo cong toàn phần với `border-radius: var(--radius-full)`.
5. **Thẻ Panel Luôn Có Viền Mảnh Hòa Sắc**:
   * Các khối nội dung phải dùng class `.panel` hoặc khai báo `background: var(--surface); border: 1px solid var(--line); box-shadow: var(--shadow); border-radius: var(--radius-lg);`.
6. **Mobile-First Phải Co Giãn Mượt Mà (Graceful Degradation)**:
   * Mọi màn hình phải kiểm tra trên độ phân giải hẹp `<= 720px`. Bảng biểu và danh sách kiểm tra (key-value metadata) phải chuyển từ lưới nhiều cột sang cột đơn (single column) để tránh tràn chữ và vỡ layout.
