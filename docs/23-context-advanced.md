# 23. Context nâng cao — Dùng đúng, tránh lỗi phổ biến nhất

> Doc `05-context.md` đã cover context cơ bản (cancellation, deadline, `context.Background()`). Doc này đi sâu vào những lỗi mid-level developer hay mắc phải khi dùng context trong code thực tế.

---

## 1. Context KHÔNG phải là nơi để truyền data tuỳ tiện

Sai lầm phổ biến nhất: dùng `context.WithValue` như một "global bag" để nhét mọi thứ — user ID, DB connection, config, logger...

```go
// BAD: Lạm dụng context để truyền dependency
func GetUser(ctx context.Context, id int64) (*User, error) {
    db := ctx.Value("db").(*sql.DB) // type assertion không an toàn, dễ panic
    logger := ctx.Value("logger").(*slog.Logger)
    // Không ai biết context này cần chứa gì nếu không đọc hết code
    return db.QueryUser(ctx, id)
}
```

```go
// GOOD: Dependency truyền qua constructor (như doc 10-project-structure.md),
// context CHỈ mang request-scoped metadata: request ID, trace ID, deadline
type UserService struct {
    db     *sql.DB
    logger *slog.Logger
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
    // ctx chỉ dùng cho cancellation/deadline, không chứa dependency
    return s.db.QueryUser(ctx, id)
}
```

**Quy tắc:** `context.WithValue` chỉ nên chứa **request-scoped data cần đi qua nhiều layer mà không tiện truyền tham số** — ví dụ: request ID để log, trace ID cho distributed tracing, authenticated user claims. KHÔNG chứa: database connection, logger, config, hay bất kỳ thứ gì là dependency thực sự của function.

---

## 2. Custom context key — Tránh collision

Nếu dùng string làm key, hai package khác nhau vô tình dùng cùng string sẽ ghi đè lên nhau.

```go
// BAD: string key có thể collision với package khác
ctx = context.WithValue(ctx, "userID", 123)
```

```go
// GOOD: dùng type riêng (unexported) làm key — không thể bị package khác đụng
package requestctx

type contextKey int

const (
    userIDKey contextKey = iota
    requestIDKey
)

func WithUserID(ctx context.Context, id int64) context.Context {
    return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) (int64, bool) {
    id, ok := ctx.Value(userIDKey).(int64)
    return id, ok
}
```

Vì `contextKey` là type riêng của package `requestctx`, không package nào khác có thể tạo key trùng — kể cả nếu họ cũng dùng giá trị `0`. Đây là lý do Go docs khuyến cáo: "context keys nên là unexported type".

---

## 3. Context leak — Quên gọi `cancel()`

Mọi hàm trả về `cancel func()` từ `context.WithCancel`, `WithTimeout`, `WithDeadline` **phải** gọi `cancel()`, kể cả khi context đã hết hạn tự nhiên. Nếu không, goroutine bên trong context (timer nội bộ) sẽ leak cho tới khi bị garbage collect hoặc hết deadline.

```go
// BAD: Không gọi cancel — leak resource (timer goroutine) nếu return sớm
func fetchData(ctx context.Context) (*Data, error) {
    ctx, _ = context.WithTimeout(ctx, 5*time.Second) // cancel bị bỏ qua bằng "_"
    return callAPI(ctx)
}
```

```go
// GOOD: luôn defer cancel() ngay sau khi tạo
func fetchData(ctx context.Context) (*Data, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel() // giải phóng resource ngay khi function return, dù thành công hay lỗi
    return callAPI(ctx)
}
```

`go vet` **không** bắt được lỗi này (khác với unused variable). `golangci-lint` với linter `lostcancel` sẽ bắt được — đảm bảo bật linter này trong `.golangci.yml` (đã có trong config của bạn qua `govet`).

---

## 4. Context propagation qua goroutine — Đừng dùng context đã hết hạn

```go
// BAD: request context bị cancel khi HTTP handler return,
// nhưng goroutine nền vẫn cố dùng nó → fail ngay lập tức
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    go func() {
        // r.Context() sẽ bị cancel ngay khi handler return response
        sendConfirmationEmail(r.Context(), orderID) // LUÔN fail vì ctx đã cancel
    }()
    w.WriteHeader(http.StatusCreated)
}
```

```go
// GOOD: Tạo context độc lập cho việc chạy nền, không phụ thuộc vòng đời request
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    go func() {
        // context.WithoutCancel (Go 1.21+) giữ lại values nhưng KHÔNG kế thừa cancellation
        bgCtx := context.WithoutCancel(r.Context())
        bgCtx, cancel := context.WithTimeout(bgCtx, 30*time.Second)
        defer cancel()

        if err := sendConfirmationEmail(bgCtx, orderID); err != nil {
            h.logger.Error("failed to send confirmation", "error", err)
        }
    }()
    w.WriteHeader(http.StatusCreated)
}
```

Trước Go 1.21 (không có `WithoutCancel`), pattern phổ biến là tạo `context.Background()` mới và tự copy các value cần thiết (request ID, trace ID) sang.

---

## 5. Khi nào KHÔNG cần context

- Function thuần logic, không I/O, không thể bị cancel giữa chừng (ví dụ: `func Add(a, b int) int`) — không cần context.
- Constructor (`NewUserService(db, logger)`) — context là request-scoped, không nên lưu vào struct field. **Không bao giờ** lưu context vào struct để dùng lại sau:

```go
// BAD: lưu context vào struct — vi phạm nguyên tắc "context không sống lâu hơn 1 request"
type Worker struct {
    ctx context.Context // ANTI-PATTERN
}
```

```go
// GOOD: context luôn là tham số đầu tiên của method, không lưu trong struct
type Worker struct {
    db *sql.DB
}

func (w *Worker) Process(ctx context.Context, job Job) error {
    return w.db.SaveContext(ctx, job)
}
```

Đây là convention chính thức từ Go team: *"Do not store Contexts inside a struct type; instead, pass a Context explicitly to each function that needs it. The Context should be the first parameter, typically named ctx."*

---

## Bảng tổng kết lỗi thường gặp

| Lỗi | Hậu quả | Cách fix |
|---|---|---|
| Dùng `context.WithValue` cho dependency | Type assertion panic, khó test, khó biết context cần gì | Truyền dependency qua constructor |
| String làm context key | Collision giữa các package | Dùng `type contextKey int` riêng, unexported |
| Quên `defer cancel()` | Leak timer goroutine | Luôn `defer cancel()` ngay dòng sau khi tạo |
| Dùng `r.Context()` trong goroutine nền | Goroutine fail ngay vì ctx bị cancel khi handler return | Dùng `context.WithoutCancel` + timeout riêng |
| Lưu context vào struct field | Context "sống" quá lâu, sai vòng đời | Luôn truyền ctx làm tham số đầu tiên của method |

---

## Bài tập

Viết một hàm `ProcessBatch(ctx context.Context, items []Item) error` xử lý từng item tuần tự, nhưng dừng ngay lập tức nếu `ctx` bị cancel (deadline hoặc caller huỷ) — không xử lý tiếp các item còn lại. Dùng `select` với `ctx.Done()` giữa mỗi iteration để kiểm tra.
