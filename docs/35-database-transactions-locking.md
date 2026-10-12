# 35. Database nâng cao — Transaction, Locking, N+1, Connection Pool

> Bổ sung cho [14-database.md](14-database.md). Đây là nhóm câu hỏi phỏng vấn backend hay gặp nhất: "làm sao không bán quá số hàng tồn kho khi 100 request đến cùng lúc?"

---

## 1. Helper `withTx` — viết một lần, dùng mọi nơi

```go
func withTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback() // sau Commit lỗi, Rollback trả ErrTxDone — vô hại
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
```

Quy tắc: transaction càng ngắn càng tốt. **Không** gọi HTTP/email/queue bên trong transaction (giữ lock + giữ connection trong lúc chờ mạng).

---

## 2. Bài toán kinh điển: trừ kho khi có đồng thời

### BAD — read-modify-write (race condition, bán quá số lượng)

```go
var stock int
tx.QueryRowContext(ctx, `SELECT stock FROM products WHERE id=$1`, id).Scan(&stock)
if stock < qty { return ErrOutOfStock }
tx.ExecContext(ctx, `UPDATE products SET stock=$1 WHERE id=$2`, stock-qty, id) // ghi đè theo giá trị cũ!
```
Hai request cùng đọc `stock=1`, cả hai đều qua kiểm tra, cả hai đều ghi → bán 2 cái, kho còn 0 nhưng có 2 đơn.

### GOOD 1 — Update nguyên tử có điều kiện (đơn giản nhất, nên thử trước)

```go
res, err := tx.ExecContext(ctx,
	`UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1`, qty, id)
if err != nil { return err }
if n, _ := res.RowsAffected(); n == 0 {
	return ErrOutOfStock // hoặc không tồn tại — phân biệt bằng query phụ nếu cần
}
```
DB tự khoá dòng trong lúc UPDATE → không cần lock thủ công.

### GOOD 2 — Pessimistic lock (khi logic giữa đọc và ghi phức tạp)

```go
tx.QueryRowContext(ctx, `SELECT stock FROM products WHERE id=$1 FOR UPDATE`, id).Scan(&stock)
// các request khác chờ ở dòng này tới khi tx này commit/rollback
```
Dùng `FOR UPDATE SKIP LOCKED` cho job queue trong DB (nhiều worker lấy việc, không chặn nhau).

### GOOD 3 — Optimistic lock (xung đột hiếm, đọc nhiều)

```sql
ALTER TABLE products ADD COLUMN version int NOT NULL DEFAULT 0;
```
```go
res, _ := tx.ExecContext(ctx,
	`UPDATE products SET stock=$1, version=version+1 WHERE id=$2 AND version=$3`,
	newStock, id, oldVersion)
if n, _ := res.RowsAffected(); n == 0 {
	return ErrConflict // caller đọc lại và retry
}
```

| | Pessimistic | Optimistic |
|---|---|---|
| Xung đột nhiều | Tốt (xếp hàng) | Retry storm |
| Xung đột hiếm | Khoá thừa | Tốt, không giữ lock |
| Rủi ro | Deadlock, giữ connection | Phải viết retry |

---

## 3. Isolation level — nhớ 4 dòng này

| Level | Chặn | Ghi chú Postgres |
|---|---|---|
| Read Uncommitted | — | Postgres coi như Read Committed |
| **Read Committed** (mặc định) | dirty read | Mỗi câu lệnh thấy snapshot mới → non-repeatable read vẫn xảy ra |
| Repeatable Read | + non-repeatable, phantom | Có thể lỗi `40001` serialization failure → **phải retry** |
| Serializable | + write skew | Lỗi `40001` nhiều hơn → **bắt buộc có retry loop** |

## 4. Deadlock và retry

Deadlock xảy ra khi tx A khoá (1 rồi 2), tx B khoá (2 rồi 1). Cách phòng:
1. **Luôn khoá theo thứ tự cố định** (vd. sort `product_id` tăng dần trước khi trừ kho nhiều sản phẩm).
2. Giữ transaction ngắn.
3. Có retry cho lỗi `40001` (serialization) và `40P01` (deadlock):

```go
func retryTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	for attempt := 0; attempt < 3; attempt++ {
		err := withTx(ctx, db, fn)
		var pgErr *pgconn.PgError // github.com/jackc/pgx/v5/pgconn
		if errors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "40P01") {
			time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
			continue
		}
		return err
	}
	return errors.New("tx: quá số lần retry")
}
```

---

## 5. N+1 query

### BAD
```go
orders, _ := listOrders(ctx)
for i := range orders {
	orders[i].Items, _ = listItems(ctx, orders[i].ID) // 1 query/đơn → 1 + N query
}
```
### GOOD — gom 1 query
```go
ids := make([]int64, len(orders))
for i, o := range orders { ids[i] = o.ID }
rows, _ := db.QueryContext(ctx, `SELECT order_id, sku, qty FROM order_items WHERE order_id = ANY($1)`, pq.Array(ids))
// rồi group theo order_id bằng map[int64][]Item
```
(Với pgx dùng `[]int64` trực tiếp; với `lib/pq` dùng `pq.Array`.) Phát hiện: bật log query + đếm số query mỗi request; test đặt ngưỡng.

## 6. Connection pool — 4 núm vặn

```go
db.SetMaxOpenConns(25)                 // trần, tránh làm sập Postgres (mặc định: không giới hạn!)
db.SetMaxIdleConns(25)                 // = MaxOpen để tránh đóng/mở liên tục
db.SetConnMaxLifetime(30 * time.Minute) // xoay vòng qua LB/failover
db.SetConnMaxIdleTime(5 * time.Minute)
```
- Tổng `MaxOpenConns` của mọi instance ≤ `max_connections` của DB (trừ phần admin).
- Theo dõi `db.Stats()`: `WaitCount`/`WaitDuration` tăng = pool thiếu hoặc query quá chậm/giữ connection quá lâu.
- **Rò connection** thường do quên `rows.Close()` hoặc `tx` không Commit/Rollback. Luôn `defer rows.Close()` và kiểm tra `rows.Err()`.
- Luôn dùng `...Context` + timeout, để request bị huỷ thì query cũng bị huỷ.

---

## Bài tập
1. Viết test: 50 goroutine cùng mua sản phẩm `stock=10`. Kỳ vọng đúng 10 thành công. Chạy với BAD (thấy oversell) rồi GOOD 1, 2, 3.
2. Cố tình tạo deadlock bằng 2 transaction khoá ngược thứ tự; sửa bằng sort ID.
3. Đếm số query của endpoint `GET /orders` trước/sau khi sửa N+1.
4. Giảm `MaxOpenConns=2`, chạy load, quan sát `WaitCount`.

**Liên quan:** [14-database.md](14-database.md), [36-idempotency-and-reliability.md](36-idempotency-and-reliability.md), [38-integration-testing-testcontainers.md](38-integration-testing-testcontainers.md)
