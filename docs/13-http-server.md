# 13. HTTP Server & Middleware trong Go

> **Dành cho:** Lập trình viên C# đã quen với ASP.NET Core, controllers, attributes, middleware pipeline.

---

## 1. Triết lý của Go: Interface thay vì Framework

Trong ASP.NET Core, bạn đăng ký services, dùng `[ApiController]`, `[HttpGet]`, dependency injection container, và framework "ma thuật" tìm controllers cho bạn. Go đi theo hướng ngược lại hoàn toàn: thư viện chuẩn `net/http` chỉ cần bạn implement **một interface duy nhất**.

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

Đó là tất cả. Nếu struct của bạn có method `ServeHTTP(w http.ResponseWriter, r *http.Request)`, nó là một HTTP handler. Không cần kế thừa, không cần attribute, không cần registration magic. Đây là Go idiom: **compose bằng interface nhỏ**, không phải kế thừa class lớn.

---

## 2. `net/http` Basics

### 2.1 `http.HandlerFunc` — Adapter từ function thành Handler

Viết một struct implement `Handler` cho mọi endpoint sẽ rất verbose. Go cung cấp `http.HandlerFunc` — một type adapter cho phép dùng function thường làm handler:

```go
// BAD: Verbose không cần thiết cho handler đơn giản
// ❌ Không sai về kỹ thuật, nhưng thừa code
type HelloHandler struct{}

func (h HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Xin chào")
}

func main() {
    http.Handle("/hello", HelloHandler{})
    http.ListenAndServe(":8080", nil)
}
```

```go
// GOOD: Dùng http.HandlerFunc — idiomatic Go
// ✅ Function thường, dễ đọc, dễ test
package main

import (
    "fmt"
    "net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Xin chào")
}

func main() {
    // http.HandleFunc tự bọc function vào http.HandlerFunc
    http.HandleFunc("/hello", helloHandler)
    http.ListenAndServe(":8080", nil)
}
```

`http.HandlerFunc` được định nghĩa như sau trong source Go:

```go
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r)
}
```

Đây là pattern "adapter": một type mới wraps một function và tự implement interface. C# không có khái niệm tương đương trực tiếp — gần nhất là `Func<HttpContext, Task>` trong minimal API của ASP.NET Core.

### 2.2 `http.ServeMux` — Router mặc định

`http.ServeMux` là router mặc định. Trước Go 1.22, nó chỉ hỗ trợ prefix matching đơn giản. Từ Go 1.22, nó được nâng cấp đáng kể.

```go
// BAD: Dùng DefaultServeMux toàn cục — khó test, dễ conflict
// ❌ DefaultServeMux là biến global, không nên dùng trong production
http.HandleFunc("/users", usersHandler) // đăng ký vào DefaultServeMux toàn cục
http.ListenAndServe(":8080", nil)       // nil = dùng DefaultServeMux
```

```go
// GOOD: Tạo ServeMux riêng — testable, không side effect toàn cục
// ✅ Mỗi server có mux riêng, dễ test, dễ compose
package main

import (
    "fmt"
    "net/http"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/users", usersHandler)
    mux.HandleFunc("/products", productsHandler)

    server := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }
    server.ListenAndServe()
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "users")
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "products")
}
```

---

## 3. Go 1.22 Enhanced Routing

Trước Go 1.22, nếu bạn cần routing theo HTTP method hoặc path parameters (`/users/{id}`), bạn **phải** dùng thư viện bên ngoài. Từ Go 1.22, `net/http` hỗ trợ built-in:

```go
// Go 1.22+: Method + Path pattern matching
// So sánh với ASP.NET: [HttpGet("users/{id}")] trên controller method
mux.HandleFunc("GET /users/{id}", getUserHandler)
mux.HandleFunc("POST /users", createUserHandler)
mux.HandleFunc("DELETE /users/{id}", deleteUserHandler)
```

Lấy path parameter trong Go 1.22:

```go
func getUserHandler(w http.ResponseWriter, r *http.Request) {
    // r.PathValue("id") thay thế cho route values trong ASP.NET Core
    id := r.PathValue("id")
    fmt.Fprintf(w, "User ID: %s", id)
}
```

Dù vậy, với project thực tế cần middleware, route groups, và nhiều tính năng khác, `chi` vẫn là lựa chọn phổ biến hơn.

---

## 4. Chi Router — Khi `net/http` Chưa Đủ

`chi` (github.com/go-chi/chi/v5) là router phổ biến nhất trong Go. Lý do chọn `chi` thay vì `net/http` thuần: nó **100% compatible với `http.Handler`** (không lock-in), hỗ trợ middleware chaining fluent, route groups, và sub-routers — trong khi vẫn nhẹ và không có "magic".

So sánh với C#: `chi` gần với cách ASP.NET Core Minimal API group routes, nhưng không có controller overhead.

```bash
go get github.com/go-chi/chi/v5
```

```go
// GOOD: Chi router với route groups và middleware
// ✅ Cấu trúc rõ ràng, middleware per-group, compatible với http.Handler
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

func main() {
    r := chi.NewRouter()

    // Middleware toàn cục (áp dụng cho mọi route)
    // Tương đương app.Use() trong ASP.NET Core
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Route đơn giản
    r.Get("/", homeHandler)

    // Route group — tương đương MapGroup("/api/v1") trong ASP.NET Core
    r.Route("/api/v1", func(r chi.Router) {
        // Middleware chỉ áp dụng cho group này
        r.Use(authMiddleware)

        r.Route("/users", func(r chi.Router) {
            r.Get("/", listUsersHandler)        // GET /api/v1/users
            r.Post("/", createUserHandler)      // POST /api/v1/users
            r.Get("/{id}", getUserHandler)      // GET /api/v1/users/{id}
            r.Put("/{id}", updateUserHandler)   // PUT /api/v1/users/{id}
            r.Delete("/{id}", deleteUserHandler) // DELETE /api/v1/users/{id}
        })
    })

    http.ListenAndServe(":8080", r)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Home"))
}

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Kiểm tra auth trước khi gọi handler tiếp theo
        next.ServeHTTP(w, r)
    })
}

func listUsersHandler(w http.ResponseWriter, r *http.Request)   {}
func createUserHandler(w http.ResponseWriter, r *http.Request)  {}
func getUserHandler(w http.ResponseWriter, r *http.Request)     {}
func updateUserHandler(w http.ResponseWriter, r *http.Request)  {}
func deleteUserHandler(w http.ResponseWriter, r *http.Request)  {}
```

---

## 5. Đọc Request Data

### 5.1 JSON Body

Trong ASP.NET Core, framework tự deserialize body thành parameter nhờ `[FromBody]`. Go không có magic đó — bạn decode thủ công, nhưng đổi lại bạn hiểu chính xác điều gì đang xảy ra.

```go
// BAD: Đọc JSON không đóng body, không kiểm tra lỗi đúng cách
// ❌ Body leak, lỗi decode không được xử lý rõ ràng
func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    json.NewDecoder(r.Body).Decode(&req) // bỏ qua lỗi!
    // Xử lý req...
}
```

```go
// GOOD: Đọc JSON body đúng cách
// ✅ Defer close body, kiểm tra lỗi, giới hạn kích thước request
func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // defer đảm bảo body luôn được đóng — tương đương using statement trong C#
    defer r.Body.Close()

    // Giới hạn kích thước body để tránh DoS attack
    // Trong ASP.NET Core có MaxRequestBodySize tương đương
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB

    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        // Trả về lỗi rõ ràng thay vì panic hoặc bỏ qua
        http.Error(w, "Invalid JSON body", http.StatusBadRequest)
        return
    }

    // Validate sau khi decode
    if req.Name == "" {
        http.Error(w, "Name is required", http.StatusBadRequest)
        return
    }

    // Xử lý tiếp...
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

### 5.2 Query Parameters, Path Parameters, và Headers

```go
// ✅ Tất cả cách đọc request data trong một handler
func searchHandler(w http.ResponseWriter, r *http.Request) {
    // Query params: /search?q=golang&page=2
    // Tương đương [FromQuery] trong ASP.NET Core
    query := r.URL.Query().Get("q")
    page := r.URL.Query().Get("page")

    // Path params với chi: /users/{id}/posts/{postId}
    // Tương đương route values trong ASP.NET Core
    userID := chi.URLParam(r, "id")
    postID := chi.URLParam(r, "postId")

    // Path params với Go 1.22 net/http thuần:
    // userID := r.PathValue("id")

    // Headers
    // Tương đương HttpContext.Request.Headers["Authorization"]
    authHeader := r.Header.Get("Authorization")
    contentType := r.Header.Get("Content-Type")

    // Context values (được set bởi middleware — xem phần middleware)
    // Tương đương HttpContext.Items trong ASP.NET Core
    requestID, _ := r.Context().Value(requestIDKey).(string)

    _ = query; _ = page; _ = userID; _ = postID
    _ = authHeader; _ = contentType; _ = requestID
}

type contextKey string
const requestIDKey contextKey = "requestID"
```

---

## 6. Viết Response

### 6.1 JSON Response

Trong ASP.NET Core, bạn `return Ok(data)` và framework tự serialize. Trong Go, bạn set header trước rồi encode — thứ tự quan trọng.

```go
// BAD: Sai thứ tự header và status code
// ❌ WriteHeader phải được gọi TRƯỚC khi Write/Encode
// Một khi Write() được gọi, status code 200 đã được gửi
func badHandler(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"message": "created"})
    w.WriteHeader(http.StatusCreated) // ❌ Quá muộn! Header đã được flush
}
```

```go
// GOOD: Đúng thứ tự: Header → WriteHeader → Body
// ✅ Set Content-Type, rồi status code, rồi mới encode body
func goodHandler(w http.ResponseWriter, r *http.Request) {
    resp := map[string]string{"message": "created", "id": "123"}

    // 1. Set headers TRƯỚC
    w.Header().Set("Content-Type", "application/json")

    // 2. WriteHeader — status code (mặc định 200 nếu bỏ qua)
    w.WriteHeader(http.StatusCreated) // 201

    // 3. Encode body
    json.NewEncoder(w).Encode(resp)
}
```

### 6.2 Consistent Error Response

Định nghĩa một format lỗi nhất quán là best practice — giống như `ProblemDetails` trong ASP.NET Core:

```go
// ✅ Error response nhất quán cho toàn bộ API
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Detail  string `json:"detail,omitempty"`
}

func writeError(w http.ResponseWriter, status int, message string, detail string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ErrorResponse{
        Code:    status,
        Message: message,
        Detail:  detail,
    })
}

// Dùng như thế này trong handlers:
func someHandler(w http.ResponseWriter, r *http.Request) {
    user, err := findUser("123")
    if err != nil {
        writeError(w, http.StatusNotFound, "User not found", err.Error())
        return // ⚠️ QUAN TRỌNG: luôn return sau khi write error
    }
    _ = user
}

type User struct{ ID string }
func findUser(id string) (User, error) { return User{}, nil }
```

---

## 7. Middleware Pattern

### Middleware là gì trong Go?

Trong ASP.NET Core, middleware là class với `InvokeAsync(HttpContext context, RequestDelegate next)`. Trong Go, middleware chỉ đơn giản là một **function nhận Handler và trả về Handler**:

```go
func myMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Code chạy TRƯỚC handler (pre-processing)
        // Tương đương: code trước await next(context) trong ASP.NET middleware

        next.ServeHTTP(w, r) // gọi handler tiếp theo trong chain

        // Code chạy SAU handler (post-processing)
        // Tương đương: code sau await next(context) trong ASP.NET middleware
    })
}
```

Pipeline trong Go là **tay trái gọi tay phải** — giống ASP.NET Core, nhưng được compose thủ công thay vì framework tự wire.

### 7.1 Logging Middleware

```go
// ✅ Logging middleware — ghi lại method, path, status, thời gian
// So sánh: UseHttpLogging() trong ASP.NET Core
package main

import (
    "log"
    "net/http"
    "time"
)

// responseWriter wraps http.ResponseWriter để capture status code
// Vì http.ResponseWriter không expose status code sau khi WriteHeader được gọi
type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(status int) {
    rw.status = status
    rw.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap ResponseWriter để capture status
        wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

        next.ServeHTTP(wrapped, r)

        // Log sau khi handler xong
        log.Printf(
            "method=%s path=%s status=%d duration=%s ip=%s",
            r.Method,
            r.URL.Path,
            wrapped.status,
            time.Since(start),
            r.RemoteAddr,
        )
    })
}
```

### 7.2 Auth Middleware (JWT)

```go
// ✅ Auth middleware — validate JWT và đưa claims vào context
// So sánh: [Authorize] attribute hoặc UseAuthentication() trong ASP.NET Core
package main

import (
    "context"
    "net/http"
    "strings"
)

type contextKey string

const (
    userIDKey    contextKey = "userID"
    userEmailKey contextKey = "userEmail"
)

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, `{"code":401,"message":"Authorization header required"}`, http.StatusUnauthorized)
            return // ⚠️ return để dừng chain
        }

        // Expect "Bearer <token>"
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            http.Error(w, `{"code":401,"message":"Invalid authorization format"}`, http.StatusUnauthorized)
            return
        }

        token := parts[1]

        // Validate token (trong thực tế dùng thư viện như golang-jwt/jwt)
        userID, email, err := validateJWT(token)
        if err != nil {
            http.Error(w, `{"code":401,"message":"Invalid token"}`, http.StatusUnauthorized)
            return
        }

        // Đưa user info vào context — tương đương HttpContext.User.Claims
        // Context trong Go là immutable, phải tạo context mới với giá trị mới
        ctx := context.WithValue(r.Context(), userIDKey, userID)
        ctx = context.WithValue(ctx, userEmailKey, email)

        // Truyền request với context mới xuống handler tiếp theo
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Handler lấy user info từ context
func protectedHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value(userIDKey).(string)
    w.Write([]byte("Hello, user " + userID))
}

// Stub — trong thực tế parse và verify JWT signature
func validateJWT(token string) (userID, email string, err error) {
    return "user-123", "user@example.com", nil
}
```

### 7.3 Panic Recovery Middleware

```go
// ✅ Recovery middleware — bắt panic và trả về 500 thay vì crash server
// So sánh: UseExceptionHandler() / UseDeveloperExceptionPage() trong ASP.NET Core
package main

import (
    "log"
    "net/http"
    "runtime/debug"
)

func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                // Log stack trace để debug
                log.Printf("PANIC recovered: %v\n%s", err, debug.Stack())

                // Trả về 500 cho client thay vì crash toàn bộ server
                // Trong Go, panic trong một goroutine sẽ kill toàn bộ program
                // nếu không được recover
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()

        next.ServeHTTP(w, r)
    })
}
```

### 7.4 Request ID Middleware

```go
// ✅ Request ID middleware — gán unique ID cho mỗi request để trace logs
// So sánh: ActivityTracing / W3C TraceContext trong ASP.NET Core
package main

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "net/http"
)

const requestIDKey contextKey = "requestID"

func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Lấy từ header nếu có (khi đứng sau load balancer)
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = generateID()
        }

        // Đưa vào context và response header
        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func generateID() string {
    b := make([]byte, 8)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

### 7.5 CORS Middleware

```go
// ✅ CORS middleware — cho phép cross-origin requests
// So sánh: UseCors() với builder trong ASP.NET Core
// Trong thực tế dùng github.com/rs/cors cho đầy đủ tính năng
package main

import "net/http"

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            // Kiểm tra origin có được phép không
            allowed := false
            for _, o := range allowedOrigins {
                if o == "*" || o == origin {
                    allowed = true
                    break
                }
            }

            if allowed {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-ID")
                w.Header().Set("Access-Control-Max-Age", "86400")
            }

            // Preflight request — OPTIONS method
            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 8. Graceful Shutdown

Trong ASP.NET Core, `IHostedService` và `IHostApplicationLifetime` handle shutdown. Go cung cấp `http.Server.Shutdown()` — đây là cách làm đúng để không "cắt đứt" requests đang xử lý khi server nhận SIGTERM (ví dụ từ Kubernetes).

```go
// ✅ Graceful shutdown hoàn chỉnh
// So sánh: host.RunAsync() với cancellation token trong ASP.NET Core
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"status":"ok"}`))
    })

    server := &http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  15 * time.Second, // thời gian đọc toàn bộ request
        WriteTimeout: 15 * time.Second, // thời gian ghi toàn bộ response
        IdleTimeout:  60 * time.Second, // keep-alive timeout
    }

    // Start server trong goroutine riêng để không block main goroutine
    go func() {
        log.Println("Server listening on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Chờ OS signal — tương đương CancellationToken từ Ctrl+C
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit // Block ở đây cho đến khi nhận signal

    log.Println("Shutting down server...")

    // Cho server 30 giây để hoàn thành các request đang xử lý
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Shutdown không chấp nhận request mới, chờ request hiện tại xong
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Forced shutdown: %v", err)
    }

    log.Println("Server exited cleanly")
}
```

---

## 9. HTTP Client

Đây là điều nhiều developer mới với Go hay mắc phải: **không bao giờ dùng `http.DefaultClient` trong production**. `http.DefaultClient` không có timeout, nghĩa là một request có thể block mãi mãi và làm leak goroutine.

```go
// BAD: Dùng DefaultClient — không timeout, không giới hạn connection
// ❌ Một request treo sẽ block goroutine mãi mãi
func fetchUserBAD(id string) ([]byte, error) {
    resp, err := http.Get("https://api.example.com/users/" + id) // dùng DefaultClient!
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
```

```go
// GOOD: HTTP client với timeout và custom transport
// ✅ Timeout, connection pooling, retry-friendly
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// Tạo client một lần, dùng lại — tương đương IHttpClientFactory trong ASP.NET Core
// http.Client được thiết kế để reuse vì nó giữ connection pool bên trong
var httpClient = &http.Client{
    Timeout: 10 * time.Second, // timeout toàn bộ request (connect + write + read)
    Transport: &http.Transport{
        MaxIdleConns:        100,           // số connection idle tối đa
        MaxIdleConnsPerHost: 10,            // per host
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 5 * time.Second,
    },
}

type GitHubUser struct {
    Login string `json:"login"`
    Name  string `json:"name"`
}

func fetchGitHubUser(ctx context.Context, username string) (*GitHubUser, error) {
    url := fmt.Sprintf("https://api.github.com/users/%s", username)

    // Dùng NewRequest để có thể đính kèm context
    // Context cho phép caller cancel request (ví dụ khi client disconnect)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }

    req.Header.Set("Accept", "application/vnd.github.v3+json")
    req.Header.Set("User-Agent", "my-go-app/1.0")

    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()

    // Kiểm tra status code trước khi decode
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
    }

    var user GitHubUser
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return nil, fmt.Errorf("decoding response: %w", err)
    }

    return &user, nil
}
```

---

## 10. Bảng So Sánh ASP.NET Core vs Go

| Khái niệm ASP.NET Core | Tương đương Go | Ghi chú |
|---|---|---|
| `app.Use(middleware)` | `r.Use(middleware)` với chi | Middleware được áp dụng theo thứ tự |
| `[HttpGet("/users")]` | `r.Get("/users", handler)` | Không cần attribute |
| `[FromBody] UserDto` | `json.NewDecoder(r.Body).Decode(&req)` | Thủ công nhưng explicit |
| `[FromQuery] string name` | `r.URL.Query().Get("name")` | |
| `[FromRoute] int id` | `chi.URLParam(r, "id")` | String, cần convert thủ công |
| `HttpContext.Items` | `r.Context().Value(key)` | Immutable context chain |
| `IHttpClientFactory` | `&http.Client{...}` tái sử dụng | Singleton per dependency |
| `return Ok(data)` | `json.NewEncoder(w).Encode(data)` | Phải set Content-Type thủ công |
| `return NotFound()` | `w.WriteHeader(http.StatusNotFound)` + body | |
| `UseExceptionHandler` | `recoveryMiddleware` | Pattern tương tự |
| `IHostedService` | goroutine + channel | Lightweight hơn |

---

## 11. Common Issues

| Vấn đề | Nguyên nhân | Giải pháp |
|---|---|---|
| Status code 200 dù đã gọi `WriteHeader(201)` | Gọi `Encode/Write` trước `WriteHeader` | Luôn: Header → WriteHeader → Body |
| Middleware không chạy theo thứ tự mong đợi | `r.Use()` áp dụng LIFO (last in, first out cho post-processing) | Log và test thứ tự middleware |
| Context value bị `nil` khi type assert | Dùng sai key type hoặc key value | Dùng unexported custom type cho context key |
| HTTP client treo mãi | Dùng `http.DefaultClient` không có timeout | Tạo `http.Client` với `Timeout` |
| Body đọc được lần 2 là empty | `r.Body` là stream, chỉ đọc một lần | Đọc vào `[]byte` nếu cần dùng lại, hoặc dùng `io.TeeReader` |
| Server không shutdown sạch | Không handle SIGTERM | Dùng `signal.Notify` + `server.Shutdown(ctx)` |
| CORS OPTIONS request trả 404 | Router không đăng ký OPTIONS route | CORS middleware xử lý OPTIONS trước khi vào router |
| Goroutine leak trong test | Client không đóng response body | Luôn `defer resp.Body.Close()` |
| `json.Encode` thêm newline thừa | `json.Encoder.Encode` tự thêm `\n` | Dùng `json.Marshal` + `w.Write` nếu không muốn newline |
| Path parameter là empty string | Dùng `r.PathValue` với chi (không tương thích) | Dùng `chi.URLParam` với chi, `r.PathValue` với net/http 1.22 |

---

## 12. Best Practices

Về cấu trúc server, hãy tạo `http.Server` struct với đầy đủ timeout thay vì gọi `http.ListenAndServe` trực tiếp. `ReadTimeout`, `WriteTimeout`, và `IdleTimeout` bảo vệ server khỏi slow client attacks — đây là điều mà ASP.NET Core Kestrel đã cấu hình sẵn nhưng Go không làm default.

Về middleware order, đặt Recovery trước tất cả — nó phải bắt panic từ bất kỳ middleware nào phía sau. Đặt RequestID sau Recovery nhưng trước Logger, để logger có thể ghi request ID. Auth middleware chỉ cần áp dụng cho routes cần protect, đừng áp dụng toàn cục nếu có public endpoints.

Về error handling, luôn định nghĩa một `ErrorResponse` struct nhất quán và một `writeError` helper function. Pattern này tương tự `ProblemDetails` trong ASP.NET Core — client của bạn sẽ cảm ơn vì format lỗi dự đoán được.

Về context, chỉ lưu request-scoped data vào context (request ID, user info từ auth). Đừng lưu database connection hay logger vào context — truyền chúng qua dependency injection (struct fields hoặc closure) là cách Go idiom.

Về HTTP client, hãy nghĩ về `http.Client` như `HttpClient` trong .NET — tạo một lần, dùng lại. Mỗi client giữ một connection pool; tạo nhiều client sẽ làm mất đi lợi ích này và có thể exhausted file descriptors.

---

## 13. Full Example: REST API với Chi, Middleware Stack, JSON In/Out

Đây là ví dụ hoàn chỉnh có thể chạy ngay, thể hiện tất cả các pattern ở trên:

```go
// File: main.go
// Chạy: go mod init myapi && go get github.com/go-chi/chi/v5 && go run main.go
package main

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "runtime/debug"
    "strings"
    "sync"
    "syscall"
    "time"
    "strconv"

    "github.com/go-chi/chi/v5"
)

// ============================================================
// Domain Types
// ============================================================

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type ErrorResponse struct {
    Code      int    `json:"code"`
    Message   string `json:"message"`
    RequestID string `json:"request_id,omitempty"`
}

// ============================================================
// In-memory "database" (thay thế bằng real DB trong thực tế)
// ============================================================

type UserStore struct {
    mu     sync.RWMutex
    users  map[int]User
    nextID int
}

func NewUserStore() *UserStore {
    s := &UserStore{users: make(map[int]User), nextID: 1}
    // Seed data
    s.users[1] = User{ID: 1, Name: "Alice", Email: "alice@example.com"}
    s.users[2] = User{ID: 2, Name: "Bob", Email: "bob@example.com"}
    s.nextID = 3
    return s
}

func (s *UserStore) List() []User {
    s.mu.RLock()
    defer s.mu.RUnlock()
    users := make([]User, 0, len(s.users))
    for _, u := range s.users {
        users = append(users, u)
    }
    return users
}

func (s *UserStore) Get(id int) (User, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    u, ok := s.users[id]
    return u, ok
}

func (s *UserStore) Create(req CreateUserRequest) User {
    s.mu.Lock()
    defer s.mu.Unlock()
    u := User{ID: s.nextID, Name: req.Name, Email: req.Email}
    s.users[s.nextID] = u
    s.nextID++
    return u
}

func (s *UserStore) Delete(id int) bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.users[id]; !ok {
        return false
    }
    delete(s.users, id)
    return true
}

// ============================================================
// Context Keys (unexported type để tránh collision)
// ============================================================

type ctxKey string

const (
    ctxRequestID ctxKey = "requestID"
    ctxUserEmail ctxKey = "userEmail"
)

// ============================================================
// Response helpers
// ============================================================

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(v); err != nil {
        log.Printf("writeJSON encode error: %v", err)
    }
}

func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
    requestID, _ := r.Context().Value(ctxRequestID).(string)
    writeJSON(w, status, ErrorResponse{
        Code:      status,
        Message:   message,
        RequestID: requestID,
    })
}

// ============================================================
// Middleware
// ============================================================

// Recovery — phải là middleware đầu tiên trong chain
func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("[PANIC] %v\n%s", err, debug.Stack())
                http.Error(w, `{"code":500,"message":"Internal Server Error"}`, http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// RequestID — gán ID cho mỗi request
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get("X-Request-ID")
        if id == "" {
            b := make([]byte, 8)
            rand.Read(b)
            id = hex.EncodeToString(b)
        }
        ctx := context.WithValue(r.Context(), ctxRequestID, id)
        w.Header().Set("X-Request-ID", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Logger — ghi log mỗi request
type statusRecorder struct {
    http.ResponseWriter
    status int
}

func (sr *statusRecorder) WriteHeader(status int) {
    sr.status = status
    sr.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
        next.ServeHTTP(rec, r)

        requestID, _ := r.Context().Value(ctxRequestID).(string)
        log.Printf("[%s] %s %s → %d (%s)",
            requestID, r.Method, r.URL.Path,
            rec.status, time.Since(start))
    })
}

// Auth — validate "Bearer demo-token" (stub)
// Trong thực tế: parse và verify JWT với golang-jwt/jwt
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            writeJSON(w, http.StatusUnauthorized, ErrorResponse{
                Code:    http.StatusUnauthorized,
                Message: "Authorization header required",
            })
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != "demo-token" {
            writeJSON(w, http.StatusUnauthorized, ErrorResponse{
                Code:    http.StatusUnauthorized,
                Message: "Invalid token",
            })
            return
        }

        // Trong thực tế: đưa claims vào context
        ctx := context.WithValue(r.Context(), ctxUserEmail, "demo@example.com")
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// ============================================================
// Handlers
// ============================================================

type UserHandler struct {
    store *UserStore
}

func NewUserHandler(store *UserStore) *UserHandler {
    return &UserHandler{store: store}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
    users := h.store.List()
    writeJSON(w, http.StatusOK, map[string]any{
        "data":  users,
        "count": len(users),
    })
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(chi.URLParam(r, "id"))
    if err != nil {
        writeError(w, r, http.StatusBadRequest, "Invalid user ID")
        return
    }

    user, ok := h.store.Get(id)
    if !ok {
        writeError(w, r, http.StatusNotFound, "User not found")
        return
    }

    writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    defer r.Body.Close()
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, r, http.StatusBadRequest, "Invalid JSON body")
        return
    }

    if strings.TrimSpace(req.Name) == "" {
        writeError(w, r, http.StatusBadRequest, "Name is required")
        return
    }
    if strings.TrimSpace(req.Email) == "" {
        writeError(w, r, http.StatusBadRequest, "Email is required")
        return
    }

    user := h.store.Create(req)
    writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(chi.URLParam(r, "id"))
    if err != nil {
        writeError(w, r, http.StatusBadRequest, "Invalid user ID")
        return
    }

    if !h.store.Delete(id) {
        writeError(w, r, http.StatusNotFound, "User not found")
        return
    }

    w.WriteHeader(http.StatusNoContent) // 204 — không có body
}

// ============================================================
// Routes — wire everything together
// ============================================================

func buildRouter(userHandler *UserHandler) http.Handler {
    r := chi.NewRouter()

    // Global middleware — áp dụng cho tất cả routes
    // Thứ tự: Recovery → RequestID → Logger
    r.Use(recoveryMiddleware)
    r.Use(requestIDMiddleware)
    r.Use(loggingMiddleware)

    // Health check — không cần auth
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, http.StatusOK, map[string]string{
            "status": "ok",
            "time":   time.Now().UTC().Format(time.RFC3339),
        })
    })

    // API routes — cần auth
    r.Route("/api/v1", func(r chi.Router) {
        r.Use(authMiddleware)

        r.Route("/users", func(r chi.Router) {
            r.Get("/", userHandler.List)
            r.Post("/", userHandler.Create)
            r.Get("/{id}", userHandler.Get)
            r.Delete("/{id}", userHandler.Delete)
        })
    })

    return r
}

// ============================================================
// Main
// ============================================================

func main() {
    store := NewUserStore()
    userHandler := NewUserHandler(store)
    router := buildRouter(userHandler)

    server := &http.Server{
        Addr:         ":8080",
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server trong goroutine riêng
    go func() {
        log.Println("Server listening on http://localhost:8080")
        log.Println("Test: curl http://localhost:8080/health")
        log.Println("Test: curl -H 'Authorization: Bearer demo-token' http://localhost:8080/api/v1/users")
        if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Shutdown error: %v", err)
    }
    log.Println("Server stopped.")
}
```

Chạy và test:

```bash
# Khởi động server
go run main.go

# Health check (không cần auth)
curl http://localhost:8080/health

# Lấy danh sách users (cần auth)
curl -H "Authorization: Bearer demo-token" http://localhost:8080/api/v1/users

# Lấy một user
curl -H "Authorization: Bearer demo-token" http://localhost:8080/api/v1/users/1

# Tạo user mới
curl -X POST \
  -H "Authorization: Bearer demo-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"Charlie","email":"charlie@example.com"}' \
  http://localhost:8080/api/v1/users

# Xóa user
curl -X DELETE \
  -H "Authorization: Bearer demo-token" \
  http://localhost:8080/api/v1/users/1

# Test thiếu auth
curl http://localhost:8080/api/v1/users
```

---

## 14. Bài Tập Thực Hành

### Bài tập 1: Thêm Rate Limiting Middleware

Viết một middleware giới hạn mỗi IP chỉ được gửi tối đa 10 requests mỗi phút. Dùng `sync.Map` để lưu counter per IP. Middleware phải trả về `429 Too Many Requests` khi vượt giới hạn.

**Gợi ý:** Dùng một goroutine riêng để reset counter mỗi phút với `time.Tick`.

**Solution:**

```go
package main

import (
    "net/http"
    "strings"
    "sync"
    "time"
)

type RateLimiter struct {
    mu       sync.Mutex
    requests map[string]int // IP → số request trong cửa sổ hiện tại
    limit    int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    rl := &RateLimiter{
        requests: make(map[string]int),
        limit:    limit,
    }

    // Goroutine reset counter theo chu kỳ
    go func() {
        ticker := time.NewTicker(window)
        defer ticker.Stop()
        for range ticker.C {
            rl.mu.Lock()
            rl.requests = make(map[string]int) // reset tất cả counters
            rl.mu.Unlock()
        }
    }()

    return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Lấy IP từ header (X-Forwarded-For khi đứng sau proxy)
        ip := r.Header.Get("X-Forwarded-For")
        if ip == "" {
            ip = r.RemoteAddr
            // RemoteAddr có dạng "ip:port", cần tách ra
            if idx := strings.LastIndex(ip, ":"); idx != -1 {
                ip = ip[:idx]
            }
        }

        rl.mu.Lock()
        rl.requests[ip]++
        count := rl.requests[ip]
        rl.mu.Unlock()

        if count > rl.limit {
            w.Header().Set("Retry-After", "60")
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusTooManyRequests)
            w.Write([]byte(`{"code":429,"message":"Too many requests, please try again later"}`))
            return
        }

        // Thêm header cho client biết còn bao nhiêu request
        // Tương đương rate limit headers trong ASP.NET Core Rate Limiting middleware
        w.Header().Set("X-RateLimit-Limit", "10")
        next.ServeHTTP(w, r)
    })
}

// Tích hợp vào server:
func exampleRateLimitUsage() {
    limiter := NewRateLimiter(10, time.Minute)

    mux := http.NewServeMux()
    mux.HandleFunc("GET /api/data", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"data":"ok"}`))
    })

    // Bọc toàn bộ mux bằng rate limiter
    handler := limiter.Middleware(mux)
    http.ListenAndServe(":8080", handler)
}
```

---

### Bài tập 2: Middleware đo thời gian và log chậm

Viết middleware ghi cảnh báo khi một request mất hơn 200ms. Log phải bao gồm method, path, và thời gian thực tế.

**Solution:**

```go
package main

import (
    "log"
    "net/http"
    "time"
)

func slowRequestMiddleware(threshold time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            next.ServeHTTP(w, r)
            elapsed := time.Since(start)

            if elapsed > threshold {
                // Dùng log.Printf với prefix WARN để dễ grep trong log aggregator
                log.Printf("[SLOW REQUEST] %s %s took %s (threshold: %s)",
                    r.Method, r.URL.Path, elapsed, threshold)
            }
        })
    }
}

// Dùng:
// r.Use(slowRequestMiddleware(200 * time.Millisecond))
```

---

## Middleware Pipeline — So sánh với ASP.NET Core

Trong ASP.NET Core:

```csharp
app.UseExceptionHandler();      // 1. Recovery
app.UseHttpLogging();           // 2. Logger  
app.UseAuthentication();        // 3. Auth (populate User)
app.UseAuthorization();         // 4. Auth (check permissions)
app.MapControllers();           // 5. Routing
```

Trong Go với chi:

```go
r.Use(recoveryMiddleware)       // 1. Recovery — giống UseExceptionHandler
r.Use(requestIDMiddleware)      // 2. Request ID (không có built-in equivalent)
r.Use(loggingMiddleware)        // 3. Logger — giống UseHttpLogging
// Auth áp dụng cho sub-router:
r.Route("/api", func(r chi.Router) {
    r.Use(authMiddleware)       // 4. Auth — giống UseAuthentication + UseAuthorization
})
```

Điểm khác biệt quan trọng: ASP.NET Core phân tách Authentication (ai đây?) và Authorization (được phép không?) thành hai middleware riêng với nhiều abstraction. Go không ép bạn làm vậy — bạn quyết định mức độ phân tách phù hợp với project.

---

## Next

[14. Go Database (SQL) →](14-database.md) — Làm việc với `database/sql`, driver, `sqlc`, transaction, và connection pooling.
