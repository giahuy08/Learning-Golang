# 05 — Context Package

## Tại sao Context tồn tại?

Trong C#, khi bạn muốn huỷ một tác vụ bất đồng bộ, bạn dùng `CancellationToken`. Bạn tạo một `CancellationTokenSource`, truyền token vào method, và khi cần huỷ, bạn gọi `source.Cancel()`. Cơ chế này hoạt động tốt trong một ứng dụng đơn lẻ, nhưng nó không được thiết kế để truyền xuyên qua nhiều tầng của một hệ thống phân tán.

Go tiếp cận vấn đề này khác hơn. Package `context` được sinh ra từ kinh nghiệm vận hành hệ thống phân tán quy mô lớn của Google, nơi một request HTTP có thể kích hoạt hàng chục RPC calls đến các service khác nhau, mỗi call lại có thể spawn thêm goroutines. Khi người dùng đóng trình duyệt — hoặc khi một timeout xảy ra — tất cả công việc đang thực hiện cho request đó phải dừng lại ngay lập tức, không phải từng cái một mà là đồng loạt. `context.Context` là cơ chế để thực hiện điều đó: nó là một signal được truyền xuống theo cây cuộc gọi, và khi node gốc bị huỷ, toàn bộ cây đều biết.

Sự khác biệt cốt lõi so với `CancellationToken`: context trong Go không chỉ mang signal huỷ mà còn mang deadline (thời hạn tuyệt đối), timeout (khoảng thời gian chờ), và các giá trị request-scoped như request ID hay user ID. Nó là một "carrier" cho metadata của toàn bộ vòng đời một request.

---

## Context Interface

Interface `context.Context` có bốn method:

```go
type Context interface {
    // Deadline trả về thời điểm context sẽ bị huỷ tự động.
    // ok == false nếu không có deadline được đặt.
    Deadline() (deadline time.Time, ok bool)

    // Done trả về một channel bị đóng khi context bị huỷ.
    // Giống như CancellationToken.WaitHandle trong C#.
    Done() <-chan struct{}

    // Err trả về lý do tại sao context bị huỷ.
    // context.Canceled nếu bị huỷ thủ công.
    // context.DeadlineExceeded nếu hết thời gian.
    // nil nếu context chưa bị huỷ.
    Err() error

    // Value trả về giá trị được gắn với key trong context.
    // Giống như HttpContext.Items trong ASP.NET Core.
    Value(key any) any
}
```

Bạn sẽ không bao giờ tự implement interface này. Thay vào đó, bạn dùng các hàm trong package `context` để tạo ra các implementation cụ thể.

---

## context.Background() và context.TODO()

`context.Background()` là context gốc — nó không bao giờ bị huỷ, không có deadline, không có giá trị nào. Tất cả các context khác đều được tạo từ nó. Trong C#, điều này tương đương với `CancellationToken.None` nhưng đóng vai trò là "nguồn gốc" thay vì "không có gì".

Dùng `context.Background()` ở những nơi bạn biết chắc không cần huỷ: trong `main()`, trong `init()`, hoặc trong test setup. Đây là điểm khởi đầu của "cây context".

`context.TODO()` là placeholder cho những chỗ bạn biết mình cần truyền context nhưng chưa biết context đó đến từ đâu. Nó có cùng hành vi với `Background()` nhưng mang ý nghĩa: "cần quay lại xử lý sau". Các tool phân tích tĩnh có thể tìm kiếm `context.TODO()` để nhắc nhở bạn.

```go
package main

import (
    "context"
    "fmt"
)

func main() {
    // Background là root context — dùng ở entry point
    ctx := context.Background()

    // TODO dùng khi bạn chưa chắc context đến từ đâu
    // Giống comment "// TODO: refactor this" trong C#
    var _ context.Context = context.TODO()

    fmt.Println(ctx)
}
```

---

## context.WithCancel() — Huỷ thủ công

`WithCancel` tạo một context mới kèm hàm `cancel`. Khi bạn gọi `cancel()`, channel `Done()` của context đó (và tất cả context con của nó) sẽ bị đóng.

So sánh với C#:
```csharp
// C# — CancellationTokenSource
var cts = new CancellationTokenSource();
var token = cts.Token;
// Huỷ:
cts.Cancel();
```

```go
// Go — context.WithCancel
ctx, cancel := context.WithCancel(context.Background())
// Huỷ:
cancel()
```

Điều quan trọng nhất: **luôn luôn gọi `cancel()`**, dù context kết thúc theo cách nào. Nếu không, bạn sẽ leak goroutines và memory. Go runtime không tự dọn dẹp các context chưa được cancel. Đây là lý do tại sao pattern `defer cancel()` tồn tại.

**BAD — Quên gọi cancel(), gây memory leak:**

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func fetchData(ctx context.Context) error {
    // Giả lập công việc
    select {
    case <-time.After(100 * time.Millisecond):
        fmt.Println("Xong công việc")
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func processRequest() {
    // BAD: tạo context nhưng không bao giờ gọi cancel()
    // Nếu fetchData kết thúc sớm, context vẫn còn tồn tại trong bộ nhớ
    // cùng với tất cả tài nguyên liên quan đến nó
    ctx, _ := context.WithCancel(context.Background()) // linter sẽ cảnh báo dòng này
    fetchData(ctx)
    // cancel không bao giờ được gọi — MEMORY LEAK
}

func main() {
    processRequest()
}
```

**GOOD — Dùng defer cancel() ngay sau khi tạo context:**

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func fetchData(ctx context.Context) error {
    select {
    case <-time.After(100 * time.Millisecond):
        fmt.Println("Xong công việc")
        return nil
    case <-ctx.Done():
        fmt.Println("Bị huỷ:", ctx.Err())
        return ctx.Err()
    }
}

func processRequest() {
    // GOOD: defer cancel() ngay lập tức sau khi WithCancel
    // Đảm bảo cancel luôn được gọi khi hàm này kết thúc
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel() // Gọi kể cả khi có panic

    fetchData(ctx)
}

func main() {
    processRequest()
}
```

---

## context.WithTimeout() — Huỷ sau một khoảng thời gian

Đây là pattern phổ biến nhất trong production code. Mỗi lần gọi database, HTTP request ra ngoài, hay bất kỳ I/O nào đều nên có timeout. Trong C#, bạn quen với `HttpClient.Timeout` hay `SqlCommand.CommandTimeout` — nhưng các timeout đó được set riêng lẻ trên từng object. Trong Go, bạn set timeout một lần trên context và truyền nó qua tất cả các tầng.

```go
package main

import (
    "context"
    "fmt"
    "time"
)

// simulateDBQuery giả lập một database query mất 200ms
func simulateDBQuery(ctx context.Context, userID int) (string, error) {
    // Mô phỏng query chậm
    select {
    case <-time.After(200 * time.Millisecond):
        return fmt.Sprintf("User %d", userID), nil
    case <-ctx.Done():
        // ctx.Err() trả về context.DeadlineExceeded khi timeout
        return "", fmt.Errorf("query bị huỷ: %w", ctx.Err())
    }
}

func main() {
    // Tạo context với timeout 100ms — quá ngắn để query xong
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    result, err := simulateDBQuery(ctx, 42)
    if err != nil {
        fmt.Println("Lỗi:", err) // "Lỗi: query bị huỷ: context deadline exceeded"
        return
    }
    fmt.Println("Kết quả:", result)
}
```

`WithTimeout(ctx, d)` thực chất là shorthand cho `WithDeadline(ctx, time.Now().Add(d))`. Sau khoảng thời gian `d`, context tự động bị huỷ và `Done()` channel được đóng.

---

## context.WithDeadline() — Huỷ tại thời điểm tuyệt đối

Khi bạn cần huỷ tại một thời điểm cụ thể thay vì sau một khoảng thời gian, dùng `WithDeadline`. Ví dụ: "xử lý request này nhưng phải xong trước 23:59:59 ngày hôm nay để đảm bảo báo cáo cuối ngày".

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func runBatchJob(ctx context.Context) error {
    for i := 0; i < 10; i++ {
        select {
        case <-ctx.Done():
            return fmt.Errorf("batch job dừng ở item %d: %w", i, ctx.Err())
        default:
            // Xử lý từng item
            fmt.Printf("Xử lý item %d\n", i)
            time.Sleep(50 * time.Millisecond)
        }
    }
    return nil
}

func main() {
    // Deadline tuyệt đối: 300ms từ bây giờ
    deadline := time.Now().Add(300 * time.Millisecond)
    ctx, cancel := context.WithDeadline(context.Background(), deadline)
    defer cancel()

    if err := runBatchJob(ctx); err != nil {
        fmt.Println("Lỗi:", err)
        return
    }
    fmt.Println("Batch job hoàn thành")
}
```

---

## context.WithValue() — Lưu trữ giá trị request-scoped

`WithValue` cho phép bạn gắn một cặp key-value vào context. Trong ASP.NET Core, bạn quen với `HttpContext.Items` hay `IHttpContextAccessor` để truyền request ID, user ID xuyên suốt pipeline. `context.WithValue` phục vụ mục đích tương tự.

Quy tắc quan trọng nhất: **không dùng kiểu primitive như `string` hay `int` làm key**. Nếu bạn dùng `"userID"` làm key, package khác cũng có thể dùng `"userID"` và gây ra xung đột. Thay vào đó, định nghĩa một kiểu private để làm key — điều này đảm bảo chỉ package của bạn mới có thể đọc giá trị đó.

**BAD — Dùng string làm key, dễ xung đột:**

```go
package main

import (
    "context"
    "fmt"
)

func getUser(ctx context.Context) string {
    // BAD: dùng string "userID" làm key
    // Bất kỳ package nào cũng có thể đọc/ghi giá trị này
    // Không có type safety — trả về interface{}, phải type assert
    val := ctx.Value("userID")
    if val == nil {
        return ""
    }
    return val.(string) // Có thể panic nếu val không phải string
}

func main() {
    // BAD: dùng string literal làm key
    ctx := context.WithValue(context.Background(), "userID", "user-123")
    fmt.Println(getUser(ctx))
}
```

**GOOD — Dùng custom type làm key, type-safe và không xung đột:**

```go
package main

import (
    "context"
    "fmt"
)

// Định nghĩa kiểu key riêng cho package này
// Không export (chữ thường) để package khác không thể dùng trực tiếp
type contextKey string

const (
    contextKeyUserID    contextKey = "userID"
    contextKeyRequestID contextKey = "requestID"
)

// Helper functions để set/get — giống extension methods trong C#
func WithUserID(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, contextKeyUserID, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
    userID, ok := ctx.Value(contextKeyUserID).(string)
    return userID, ok
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, contextKeyRequestID, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
    requestID, _ := ctx.Value(contextKeyRequestID).(string)
    return requestID
}

func main() {
    ctx := context.Background()

    // Thêm giá trị vào context — context là immutable, WithValue tạo context mới
    ctx = WithUserID(ctx, "user-456")
    ctx = WithRequestID(ctx, "req-abc-123")

    // Đọc giá trị
    if userID, ok := UserIDFromContext(ctx); ok {
        fmt.Println("User ID:", userID)
    }
    fmt.Println("Request ID:", RequestIDFromContext(ctx))
}
```

Lưu ý: context là immutable. `WithValue` không modify context hiện tại mà tạo ra một context mới bọc bên ngoài context cũ. Đây là lý do bạn thấy `ctx = context.WithValue(ctx, ...)` chứ không phải `ctx.SetValue(...)`.

---

## Truyền Context đúng cách

Go có hai quy ước không thành văn nhưng được toàn cộng đồng tuân thủ:

**Thứ nhất**, context luôn là tham số đầu tiên của một hàm, không phải embedded trong struct. Trong C#, bạn thường inject `IHttpContextAccessor` vào constructor. Go từ chối cách đó vì nó ẩn dependency và làm cho hàm khó test hơn.

**Thứ hai**, tham số context luôn được đặt tên là `ctx`.

**BAD — Lưu context trong struct:**

```go
package main

import (
    "context"
    "fmt"
)

// BAD: lưu context trong struct
// Context có vòng đời gắn với một request cụ thể
// Struct thường tồn tại lâu hơn một request
// Điều này dẫn đến goroutine leak và hành vi khó đoán
type UserService struct {
    ctx context.Context // ĐÂY LÀ ANTI-PATTERN
}

func (s *UserService) GetUser(userID int) (string, error) {
    // Dùng s.ctx — nhưng ctx này thuộc về request nào?
    select {
    case <-s.ctx.Done():
        return "", s.ctx.Err()
    default:
        return fmt.Sprintf("User %d", userID), nil
    }
}

func main() {
    ctx := context.Background()
    svc := &UserService{ctx: ctx} // BAD: gắn ctx vào struct
    user, _ := svc.GetUser(1)
    fmt.Println(user)
}
```

**GOOD — Context là tham số đầu tiên:**

```go
package main

import (
    "context"
    "fmt"
)

// GOOD: UserService không chứa context
// Mỗi method nhận ctx riêng cho từng operation
type UserService struct {
    // chỉ chứa dependencies có vòng đời dài: db pool, logger, config...
}

// ctx luôn là tham số đầu tiên, luôn được đặt tên là "ctx"
func (s *UserService) GetUser(ctx context.Context, userID int) (string, error) {
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    default:
        return fmt.Sprintf("User %d", userID), nil
    }
}

func main() {
    svc := &UserService{}
    ctx := context.Background()
    user, err := svc.GetUser(ctx, 1) // ctx truyền vào mỗi lần gọi
    if err != nil {
        fmt.Println("Lỗi:", err)
        return
    }
    fmt.Println(user)
}
```

---

## Kiểm tra cancellation trong vòng lặp dài

Khi bạn có một vòng lặp xử lý nhiều item, bạn cần kiểm tra context sau mỗi iteration. Nếu không, vòng lặp sẽ tiếp tục chạy dù context đã bị huỷ — giống như một goroutine bị "runaway" mà runtime không thể dừng.

Dùng `select` với `ctx.Done()` và `default` để kiểm tra non-blocking:

```go
package main

import (
    "context"
    "fmt"
    "time"
)

// processItems xử lý một danh sách item, có thể bị huỷ giữa chừng
func processItems(ctx context.Context, items []string) error {
    for i, item := range items {
        // Kiểm tra cancellation trước mỗi iteration
        // select với default là non-blocking — không chặn nếu Done() chưa đóng
        select {
        case <-ctx.Done():
            // Context bị huỷ — dọn dẹp và return ngay
            return fmt.Errorf("dừng ở item %d (%s): %w", i, item, ctx.Err())
        default:
            // Tiếp tục xử lý bình thường
        }

        // Giả lập công việc thực tế
        fmt.Printf("Đang xử lý: %s\n", item)
        time.Sleep(100 * time.Millisecond)
    }
    return nil
}

func main() {
    items := []string{"alpha", "beta", "gamma", "delta", "epsilon"}

    // Timeout 250ms — đủ để xử lý 2 item, item thứ 3 bị cắt
    ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
    defer cancel()

    if err := processItems(ctx, items); err != nil {
        fmt.Println("Lỗi:", err)
        return
    }
    fmt.Println("Hoàn thành tất cả item")
}
```

Khi vòng lặp chứa công việc nặng hơn và bạn muốn cả goroutine phụ cũng bị huỷ, truyền ctx xuống hàm con:

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func processItem(ctx context.Context, item string) error {
    select {
    case <-time.After(80 * time.Millisecond):
        fmt.Printf("Xong: %s\n", item)
        return nil
    case <-ctx.Done():
        return fmt.Errorf("item %s bị huỷ: %w", item, ctx.Err())
    }
}

func processAll(ctx context.Context, items []string) error {
    for _, item := range items {
        // Truyền ctx xuống — nếu ctx bị huỷ, processItem cũng dừng
        if err := processItem(ctx, item); err != nil {
            return err
        }
    }
    return nil
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()

    items := []string{"A", "B", "C", "D"}
    if err := processAll(ctx, items); err != nil {
        fmt.Println("Dừng sớm:", err)
    }
}
```

---

## Context trong HTTP Handlers

Khi bạn viết HTTP handler trong Go, Go's `net/http` package tự động đính kèm một context vào mỗi request. Context này bị huỷ khi client ngắt kết nối — đây là tính năng mạnh mà trong ASP.NET Core bạn phải manually check `HttpContext.RequestAborted`.

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

// UserRepository mô phỏng database access
type UserRepository struct{}

func (r *UserRepository) FindByID(ctx context.Context, userID string) (string, error) {
    // Mô phỏng database query 500ms
    select {
    case <-time.After(500 * time.Millisecond):
        return fmt.Sprintf("User{id: %s}", userID), nil
    case <-ctx.Done():
        // Client đã ngắt kết nối hoặc timeout — không cần tiếp tục query
        return "", fmt.Errorf("database query bị huỷ: %w", ctx.Err())
    }
}

// UserService tầng business logic
type UserService struct {
    repo *UserRepository
}

func (s *UserService) GetUser(ctx context.Context, userID string) (string, error) {
    // Context được truyền thẳng xuống repository
    return s.repo.FindByID(ctx, userID)
}

// Handler tầng HTTP
func (s *UserService) HandleGetUser(w http.ResponseWriter, r *http.Request) {
    // r.Context() trả về context của request này
    // Context này sẽ bị huỷ khi:
    // 1. Client ngắt kết nối
    // 2. Handler kết thúc
    // 3. Server bị shutdown
    ctx := r.Context()

    // Thêm timeout cho toàn bộ request processing
    // Tổng thời gian xử lý không được quá 2 giây
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()

    userID := r.URL.Query().Get("id")
    if userID == "" {
        http.Error(w, "thiếu tham số id", http.StatusBadRequest)
        return
    }

    user, err := s.GetUser(ctx, userID)
    if err != nil {
        // Phân biệt lỗi do client cancel vs server error
        if ctx.Err() != nil {
            // Client ngắt kết nối hoặc timeout
            http.Error(w, "request bị huỷ", http.StatusRequestTimeout)
            return
        }
        http.Error(w, "lỗi server: "+err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Fprintln(w, user)
}

func main() {
    repo := &UserRepository{}
    svc := &UserService{repo: repo}

    http.HandleFunc("/user", svc.HandleGetUser)
    fmt.Println("Server đang chạy tại :8080")
    http.ListenAndServe(":8080", nil)
}
```

---

## Context trong Database Calls

Tất cả các database driver hiện đại của Go (`database/sql`, `pgx`, `mongo-driver`, `redis/go-redis`...) đều có phiên bản context-aware của mọi method. **Luôn dùng phiên bản có context** — nếu không, database query sẽ tiếp tục chạy trên server dù client đã ngắt kết nối, lãng phí tài nguyên database.

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"
)

type User struct {
    ID   int
    Name string
    Email string
}

type UserRepo struct {
    db *sql.DB
}

// BAD — không dùng context:
func (r *UserRepo) FindByIDBad(userID int) (*User, error) {
    // QueryRow không có context — query sẽ chạy đến khi xong dù handler đã cancel
    row := r.db.QueryRow("SELECT id, name, email FROM users WHERE id = $1", userID)
    var u User
    if err := row.Scan(&u.ID, &u.Name, &u.Email); err != nil {
        return nil, err
    }
    return &u, nil
}

// GOOD — dùng context:
func (r *UserRepo) FindByID(ctx context.Context, userID int) (*User, error) {
    // QueryRowContext nhận ctx — query sẽ bị cancel nếu ctx bị cancel
    row := r.db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE id = $1", userID)
    var u User
    if err := row.Scan(&u.ID, &u.Name, &u.Email); err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("không tìm thấy user %d", userID)
        }
        return nil, fmt.Errorf("lỗi scan: %w", err)
    }
    return &u, nil
}

// ExecWithContext cho INSERT/UPDATE/DELETE
func (r *UserRepo) UpdateEmail(ctx context.Context, userID int, email string) error {
    _, err := r.db.ExecContext(ctx,
        "UPDATE users SET email = $1 WHERE id = $2",
        email, userID,
    )
    return err
}

// Transaction cũng cần context
func (r *UserRepo) TransferWithContext(ctx context.Context, fromID, toID int, amount float64) error {
    // BeginTx nhận context — transaction sẽ bị rollback nếu ctx bị cancel
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("không thể bắt đầu transaction: %w", err)
    }
    defer tx.Rollback() // Rollback nếu Commit không được gọi

    _, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, fromID)
    if err != nil {
        return fmt.Errorf("debit thất bại: %w", err)
    }

    _, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", amount, toID)
    if err != nil {
        return fmt.Errorf("credit thất bại: %w", err)
    }

    return tx.Commit()
}

func main() {
    // Ví dụ minh hoạ — trong thực tế cần kết nối DB thật
    fmt.Println("UserRepo sử dụng context-aware database methods")
    _ = time.Second // suppress unused import
}
```

---

## Anti-patterns cần tránh

### Anti-pattern 1: Dùng context.Background() trong handler thay vì r.Context()

**BAD:**

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

func queryDB(ctx context.Context) string {
    select {
    case <-time.After(1 * time.Second):
        return "kết quả"
    case <-ctx.Done():
        return ""
    }
}

func handlerBad(w http.ResponseWriter, r *http.Request) {
    // BAD: dùng Background() thay vì r.Context()
    // Query sẽ không bao giờ bị cancel kể cả khi client ngắt kết nối
    // Tốn tài nguyên server vô ích
    ctx := context.Background() // ĐÂY LÀ LỖI
    result := queryDB(ctx)
    fmt.Fprintln(w, result)
}

func handlerGood(w http.ResponseWriter, r *http.Request) {
    // GOOD: dùng r.Context() — context này được cancel khi client ngắt kết nối
    ctx := r.Context()
    result := queryDB(ctx)
    if result == "" {
        return // Client đã ngắt kết nối, không cần write response
    }
    fmt.Fprintln(w, result)
}

func main() {
    http.HandleFunc("/bad", handlerBad)
    http.HandleFunc("/good", handlerGood)
    http.ListenAndServe(":8080", nil)
}
```

### Anti-pattern 2: Bỏ qua cancellation signal

**BAD:**

```go
package main

import (
    "context"
    "fmt"
    "time"
)

// BAD: vòng lặp không kiểm tra ctx.Done()
// Dù context bị cancel, vòng lặp vẫn chạy hết
func processItemsBad(ctx context.Context, items []int) {
    for _, item := range items {
        // Không check ctx.Done() — goroutine này không thể bị cancel
        time.Sleep(100 * time.Millisecond)
        fmt.Println("Processed:", item)
    }
}

// GOOD: kiểm tra ctx.Done() trong mỗi iteration
func processItemsGood(ctx context.Context, items []int) error {
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err() // Dừng ngay và trả về lý do
        default:
        }
        time.Sleep(100 * time.Millisecond)
        fmt.Println("Processed:", item)
    }
    return nil
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
    defer cancel()

    items := []int{1, 2, 3, 4, 5}

    if err := processItemsGood(ctx, items); err != nil {
        fmt.Println("Dừng sớm:", err)
    }
}
```

### Anti-pattern 3: Truyền nil thay vì context

Nếu bạn không có context nhưng hàm yêu cầu một, **đừng bao giờ truyền `nil`**. `nil` context sẽ gây panic khi code cố gắng gọi bất kỳ method nào trên nó. Thay vào đó, dùng `context.Background()` hoặc `context.TODO()`.

```go
package main

import (
    "context"
    "fmt"
)

func doWork(ctx context.Context) {
    // Nếu ctx là nil, dòng này sẽ panic: nil pointer dereference
    select {
    case <-ctx.Done():
        fmt.Println("Done")
    default:
        fmt.Println("Đang làm việc")
    }
}

func main() {
    // BAD: doWork(nil) — sẽ panic
    // GOOD:
    doWork(context.Background())
    doWork(context.TODO())
}
```

---

## Bảng các vấn đề thường gặp

| Vấn đề | Nguyên nhân | Cách khắc phục |
|--------|-------------|----------------|
| Goroutine leak | Không gọi `cancel()` sau `WithCancel/WithTimeout/WithDeadline` | Luôn `defer cancel()` ngay sau khi tạo context |
| Query DB không bị cancel khi client ngắt | Dùng `QueryRow` thay vì `QueryRowContext` | Dùng phiên bản `*Context` của mọi DB method |
| Context value không tìm thấy | Dùng wrong key type (string thay vì custom type) | Định nghĩa `type myKey string` và dùng làm key |
| Panic: nil pointer | Truyền `nil` làm context | Dùng `context.Background()` hoặc `context.TODO()` |
| Timeout quá nhanh | Timeout của context con ngắn hơn context cha | Kiểm tra `ctx.Err()` để phân biệt nguyên nhân |
| Context value bị overwrite | Hai package dùng cùng key string | Dùng unexported custom type làm key, unique per package |
| Long loop không responsive | Không check `ctx.Done()` trong vòng lặp | Thêm `select { case <-ctx.Done(): return ctx.Err() default: }` |
| Context lưu trong struct | Gắn `context.Context` vào field của struct | Truyền ctx làm tham số của từng method |

---

## Best practices

Context trong Go không chỉ là một convention — nó là một cam kết với người dùng của code bạn rằng operations có thể bị cancel. Khi bạn nhận một `ctx context.Context` làm tham số, bạn đang hứa sẽ tôn trọng signal cancel đó. Giống như khi bạn nhận `CancellationToken` trong C# và không bao giờ gọi `token.ThrowIfCancellationRequested()` — nó là broken promise.

Mỗi I/O operation — database, HTTP client, file, queue — đều nên nhận context. Nếu library bạn dùng không hỗ trợ context, đó là dấu hiệu library đó cũ hoặc không production-ready.

Deadline inheritance là một tính năng mạnh: khi bạn tạo context con từ context cha, context con không bao giờ có deadline dài hơn cha. Điều này có nghĩa là bạn có thể set timeout ở tầng HTTP handler và nó sẽ tự động áp dụng cho mọi database call, RPC call bên dưới — không cần propagate thủ công.

`context.WithValue` chỉ nên dùng cho metadata của request scope: request ID, user ID, trace ID, tenant ID. Không nên dùng để truyền dependencies (logger, database) — những thứ đó nên được inject qua constructor hoặc tham số explicit. Rule of thumb: nếu thứ bạn muốn truyền là "thứ gì đó để làm việc với", dùng tham số. Nếu là "metadata về request này", dùng context value.

Cuối cùng: khi bạn thấy mình viết `context.Background()` bên trong một function nhận `ctx` làm tham số, đó là dấu hiệu bạn đang cắt đứt chuỗi cancellation. Trừ khi bạn có lý do cực kỳ chính đáng (như muốn một goroutine sống sót qua request lifecycle để ghi log), hãy truyền `ctx` xuống thay vì tạo mới.

---

## Ví dụ thực tế: HTTP Handler → Service → Repository

Đây là ví dụ đầy đủ mô phỏng một ứng dụng thực tế với ba tầng. Context chảy từ HTTP handler xuống service xuống repository mà không bị gián đoạn.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "net/http"
    "time"
)

// ==================== Models ====================

type Product struct {
    ID    int     `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

// ==================== Context Keys ====================

// Kiểu key riêng để tránh xung đột với các package khác
type ctxKey string

const (
    ctxKeyRequestID ctxKey = "requestID"
    ctxKeyUserID    ctxKey = "userID"
)

func withRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, ctxKeyRequestID, id)
}

func requestIDFrom(ctx context.Context) string {
    id, _ := ctx.Value(ctxKeyRequestID).(string)
    return id
}

// ==================== Repository Layer ====================

type ProductRepository struct {
    // Trong thực tế: *sql.DB hay *pgx.Pool
}

func NewProductRepository() *ProductRepository {
    return &ProductRepository{}
}

// FindByID mô phỏng database query
func (r *ProductRepository) FindByID(ctx context.Context, productID int) (*Product, error) {
    // Mô phỏng latency của database
    select {
    case <-time.After(150 * time.Millisecond):
        // Trong thực tế: db.QueryRowContext(ctx, "SELECT ...", productID)
        if productID == 0 {
            return nil, errors.New("product không tồn tại")
        }
        return &Product{
            ID:    productID,
            Name:  fmt.Sprintf("Product #%d", productID),
            Price: float64(productID) * 10.5,
        }, nil
    case <-ctx.Done():
        // Context bị cancel (timeout hoặc client ngắt kết nối)
        reqID := requestIDFrom(ctx)
        log.Printf("[%s] DB query bị cancel: %v", reqID, ctx.Err())
        return nil, fmt.Errorf("database query bị huỷ: %w", ctx.Err())
    }
}

// ==================== Service Layer ====================

type ProductService struct {
    repo *ProductRepository
}

func NewProductService(repo *ProductRepository) *ProductService {
    return &ProductService{repo: repo}
}

func (s *ProductService) GetProduct(ctx context.Context, productID int) (*Product, error) {
    reqID := requestIDFrom(ctx)
    log.Printf("[%s] Service: lấy product %d", reqID, productID)

    // Validate business rules
    if productID <= 0 {
        return nil, fmt.Errorf("productID phải > 0, nhận được: %d", productID)
    }

    // Truyền ctx xuống repository — context chảy xuyên suốt
    product, err := s.repo.FindByID(ctx, productID)
    if err != nil {
        return nil, fmt.Errorf("service.GetProduct: %w", err)
    }

    log.Printf("[%s] Service: tìm thấy product: %s", reqID, product.Name)
    return product, nil
}

// ==================== HTTP Handler ====================

type ProductHandler struct {
    service *ProductService
}

func NewProductHandler(service *ProductService) *ProductHandler {
    return &ProductHandler{service: service}
}

// Middleware gắn request ID vào context
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Trong thực tế: lấy từ header X-Request-ID hoặc generate UUID
        requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

        // Gắn vào context — context mới được tạo, r gốc không bị thay đổi
        ctx := withRequestID(r.Context(), requestID)
        r = r.WithContext(ctx)

        // Thêm vào response header để client có thể trace
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r)
    })
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
    // Lấy context từ request — đây là context của HTTP request lifecycle
    // Sẽ bị cancel khi client ngắt kết nối
    ctx := r.Context()

    reqID := requestIDFrom(ctx)
    log.Printf("[%s] Handler: nhận request GET /product", reqID)

    // Thêm timeout cho toàn bộ request processing
    // Nếu client context có deadline ngắn hơn, deadline đó sẽ được dùng
    ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
    defer cancel() // Luôn defer cancel

    // Parse product ID từ query string
    var productID int
    fmt.Sscanf(r.URL.Query().Get("id"), "%d", &productID)

    // Gọi service — truyền ctx xuống
    product, err := h.service.GetProduct(ctx, productID)
    if err != nil {
        log.Printf("[%s] Handler: lỗi: %v", reqID, err)

        // Kiểm tra xem lỗi có phải do context cancel không
        if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
            http.Error(w, "request timeout hoặc bị huỷ", http.StatusRequestTimeout)
            return
        }
        http.Error(w, "lỗi server: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(product)
    log.Printf("[%s] Handler: response thành công", reqID)
}

// ==================== Main ====================

func main() {
    // Khởi tạo các tầng
    repo := NewProductRepository()
    service := NewProductService(repo)
    handler := NewProductHandler(service)

    // Setup routes với middleware
    mux := http.NewServeMux()
    mux.HandleFunc("/product", handler.GetProduct)

    // Bọc tất cả routes với middleware
    wrappedMux := requestIDMiddleware(mux)

    fmt.Println("Server chạy tại http://localhost:8080")
    fmt.Println("Test: curl 'http://localhost:8080/product?id=1'")
    fmt.Println("Test (lỗi): curl 'http://localhost:8080/product?id=0'")

    if err := http.ListenAndServe(":8080", wrappedMux); err != nil {
        log.Fatal(err)
    }
}
```

---

## Bài tập 1: Rate limiter với context

Viết một hàm `FetchWithRetry` nhận context, URL, số lần retry tối đa, và khoảng delay giữa các lần retry. Hàm phải dừng retry nếu context bị cancel, ngay cả khi đang trong khoảng delay.

**Giải pháp:**

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "time"
)

// FetchWithRetry thực hiện HTTP GET với retry, có thể bị cancel qua context.
// So sánh C#: Polly's RetryPolicy nhưng context-aware.
func FetchWithRetry(ctx context.Context, url string, maxRetries int, delay time.Duration) (*http.Response, error) {
    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        // Kiểm tra context trước khi thử — không thử nếu đã cancel
        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("bị cancel trước lần thử %d: %w", attempt, ctx.Err())
        default:
        }

        if attempt > 0 {
            fmt.Printf("Lần thử %d/%d, chờ %v...\n", attempt, maxRetries, delay)

            // Wait với khả năng cancel — dùng select thay vì time.Sleep
            // time.Sleep không thể bị interrupt, select có thể
            select {
            case <-time.After(delay):
                // Hết thời gian chờ, tiếp tục thử
            case <-ctx.Done():
                // Context bị cancel trong lúc đang chờ
                return nil, fmt.Errorf("bị cancel trong lúc chờ retry: %w", ctx.Err())
            }
        }

        fmt.Printf("Đang fetch: %s (lần thử %d)\n", url, attempt+1)

        // Tạo request với context — request sẽ bị cancel nếu ctx cancel
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
        if err != nil {
            return nil, fmt.Errorf("tạo request thất bại: %w", err)
        }

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            // Phân biệt context error vs network error
            if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
                return nil, fmt.Errorf("request bị cancel: %w", ctx.Err())
            }
            lastErr = err
            fmt.Printf("Lỗi lần thử %d: %v\n", attempt+1, err)
            continue
        }

        // Retry nếu server error (5xx)
        if resp.StatusCode >= 500 {
            resp.Body.Close()
            lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
            fmt.Printf("Server error %d ở lần thử %d\n", resp.StatusCode, attempt+1)
            continue
        }

        return resp, nil
    }

    return nil, fmt.Errorf("thất bại sau %d lần thử: %w", maxRetries+1, lastErr)
}

func main() {
    // Test 1: Fetch thành công
    fmt.Println("=== Test 1: Fetch thành công ===")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := FetchWithRetry(ctx, "https://httpbin.org/get", 3, 500*time.Millisecond)
    if err != nil {
        fmt.Println("Lỗi:", err)
    } else {
        fmt.Println("Thành công, status:", resp.StatusCode)
        resp.Body.Close()
    }

    // Test 2: Timeout quá ngắn
    fmt.Println("\n=== Test 2: Context timeout ===")
    ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel2()

    // httpbin.org/delay/2 chờ 2 giây — sẽ timeout
    _, err = FetchWithRetry(ctx2, "https://httpbin.org/delay/2", 3, 50*time.Millisecond)
    if err != nil {
        fmt.Println("Bị cancel (đúng như mong đợi):", err)
    }

    // Test 3: Manual cancel
    fmt.Println("\n=== Test 3: Manual cancel ===")
    ctx3, cancel3 := context.WithCancel(context.Background())

    // Cancel sau 200ms
    go func() {
        time.Sleep(200 * time.Millisecond)
        fmt.Println("Đang cancel context...")
        cancel3()
    }()

    _, err = FetchWithRetry(ctx3, "https://httpbin.org/delay/5", 5, 300*time.Millisecond)
    if err != nil {
        fmt.Println("Bị manual cancel:", err)
    }
}
```

---

## Bài tập 2: Worker pool với context cancellation

Viết một worker pool xử lý jobs song song. Khi context bị cancel, tất cả worker phải dừng lại, và function phải trả về các kết quả đã xử lý được cho đến thời điểm đó.

**Giải pháp:**

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

type Job struct {
    ID   int
    Data string
}

type Result struct {
    JobID  int
    Output string
    Err    error
}

// processJob mô phỏng xử lý một job
func processJob(ctx context.Context, job Job) Result {
    // Mỗi job mất 100ms
    select {
    case <-time.After(100 * time.Millisecond):
        return Result{
            JobID:  job.ID,
            Output: fmt.Sprintf("Kết quả của job %d: %s -> DONE", job.ID, job.Data),
        }
    case <-ctx.Done():
        return Result{
            JobID: job.ID,
            Err:   fmt.Errorf("job %d bị huỷ: %w", job.ID, ctx.Err()),
        }
    }
}

// WorkerPool chạy numWorkers goroutines xử lý jobs từ channel.
// Khi ctx bị cancel, tất cả worker dừng lại.
// Returns: tất cả results (kể cả partial results trước khi cancel)
func WorkerPool(ctx context.Context, jobs []Job, numWorkers int) []Result {
    // Channel jobs — buffered để không block khi gửi
    jobChan := make(chan Job, len(jobs))

    // Gửi tất cả jobs vào channel
    for _, job := range jobs {
        jobChan <- job
    }
    close(jobChan) // Đóng channel để worker biết không còn job mới

    // Collect results
    resultChan := make(chan Result, len(jobs))

    var wg sync.WaitGroup

    // Spawn workers
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        workerID := i + 1

        go func() {
            defer wg.Done()
            fmt.Printf("Worker %d bắt đầu\n", workerID)

            for {
                select {
                case <-ctx.Done():
                    // Context bị cancel — dừng worker
                    fmt.Printf("Worker %d dừng: %v\n", workerID, ctx.Err())
                    return
                case job, ok := <-jobChan:
                    if !ok {
                        // Channel đã đóng, không còn job
                        fmt.Printf("Worker %d: hết việc\n", workerID)
                        return
                    }

                    fmt.Printf("Worker %d đang xử lý job %d\n", workerID, job.ID)
                    result := processJob(ctx, job)
                    resultChan <- result
                }
            }
        }()
    }

    // Goroutine chờ tất cả worker xong rồi đóng resultChan
    go func() {
        wg.Wait()
        close(resultChan)
    }()

    // Thu thập results
    var results []Result
    for result := range resultChan {
        results = append(results, result)
    }

    return results
}

func main() {
    // Tạo 10 jobs
    jobs := make([]Job, 10)
    for i := range jobs {
        jobs[i] = Job{ID: i + 1, Data: fmt.Sprintf("data-%d", i+1)}
    }

    // Test 1: Tất cả jobs hoàn thành
    fmt.Println("=== Test 1: Hoàn thành tất cả ===")
    ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel1()

    results1 := WorkerPool(ctx1, jobs, 3)
    fmt.Printf("Hoàn thành: %d/%d jobs\n", len(results1), len(jobs))
    for _, r := range results1 {
        if r.Err == nil {
            fmt.Println(" ", r.Output)
        }
    }

    // Test 2: Bị cancel giữa chừng
    fmt.Println("\n=== Test 2: Cancel giữa chừng ===")
    ctx2, cancel2 := context.WithTimeout(context.Background(), 350*time.Millisecond)
    defer cancel2()

    results2 := WorkerPool(ctx2, jobs, 3)
    successful := 0
    cancelled := 0
    for _, r := range results2 {
        if r.Err == nil {
            successful++
        } else {
            cancelled++
        }
    }
    fmt.Printf("Thành công: %d, Bị cancel: %d, Tổng: %d/%d\n",
        successful, cancelled, len(results2), len(jobs))
}
```

---

## So sánh nhanh Context Go vs C#

| Tình huống | C# | Go |
|------------|----|----|
| Tạo root context | `CancellationToken.None` | `context.Background()` |
| Huỷ thủ công | `new CancellationTokenSource()` + `.Cancel()` | `context.WithCancel()` + `cancel()` |
| Timeout | `new CancellationTokenSource(timeout)` | `context.WithTimeout(ctx, duration)` |
| Deadline tuyệt đối | `CancelAfter(timeSpan)` | `context.WithDeadline(ctx, time)` |
| Request-scoped data | `HttpContext.Items` | `context.WithValue()` |
| Kiểm tra đã cancel | `token.IsCancellationRequested` | `ctx.Err() != nil` hoặc `<-ctx.Done()` |
| Chờ cancel | `WaitHandle.WaitOne()` | `<-ctx.Done()` |
| Lý do cancel | `OperationCanceledException` | `context.Canceled` / `context.DeadlineExceeded` |
| Propagation | Truyền `CancellationToken` qua tham số | Truyền `ctx` qua tham số (tham số đầu tiên) |

---

## Tiếp theo

[06 — Goroutines và Channels: Concurrency model của Go →](./06-goroutines-channels.md)
