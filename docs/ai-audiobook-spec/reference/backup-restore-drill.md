# Backup & Restore Drill

Quy trình sao lưu và phục hồi cơ sở dữ liệu PostgreSQL cho Synaudio.

## Mục đích

Đảm bảo dữ liệu (stories, canon, audio metadata, listener progress, audit) có thể
được khôi phục an toàn khi xảy ra sự cố. Drill này phải được thực hiện định kỳ để
xác nhận quy trình hoạt động, không chỉ tồn tại trên giấy.

## Công cụ

| Script | Chức năng |
|---|---|
| `scripts/backup.sh` | Tạo dump (custom `.dump` + plain `.sql`) có timestamp |
| `scripts/restore.sh` | Khôi phục từ file dump (`.dump` hoặc `.sql`) |

Yêu cầu `pg_dump`, `pg_restore`, `psql` (đi kèm PostgreSQL client tools).

## Sao lưu

```bash
# Dùng DATABASE_URL
DATABASE_URL="postgres://synaudio:synaudio@localhost:5432/synaudio?sslmode=disable" \
  ./scripts/backup.sh

# Hoặc dùng biến riêng lẻ (mặc định khớp docker-compose)
./scripts/backup.sh ./backups
```

Kết quả được ghi vào `./backups/` với tên `synaudio-<UTC timestamp>.dump` và `.sql`.

## Phục hồi

```bash
./scripts/restore.sh ./backups/synaudio-20260826T120000Z.dump
```

> **Cảnh báo:** `restore.sh` DROP và tạo lại database đích. Chỉ chạy trong drill
> có kế hoạch hoặc disaster recovery, không bao giờ chạy trên production khi chưa
> được phê duyệt rõ ràng.

## Quy trình drill (khuyến nghị)

1. **Chuẩn bị** — chạy `backup.sh` trên môi trường staging.
2. **Xác minh dump** — kiểm tra file `.dump` tồn tại và có kích thước hợp lý.
3. **Phục hồi thử** — chạy `restore.sh` vào một database staging riêng.
4. **Kiểm tra toàn vẹn** — xác nhận số bản ghi khớp (vd: `SELECT COUNT(*) FROM stories`).
5. **Ghi nhận** — lưu lại kết quả drill (thời gian, kích thước, lỗi nếu có).

## Lưu ý vận hành

- Dump chứa dữ liệu nhạy cảm (email, hash). Lưu trữ ở nơi có kiểm soát truy cập.
- Không commit file dump vào git (thêm `backups/` vào `.gitignore`).
- Đối với production, dùng `pg_dump` với kết nối read-replica nếu có để tránh tải.
- Object storage (MinIO/R2) được sao lưu riêng, không nằm trong phạm vi script này.
