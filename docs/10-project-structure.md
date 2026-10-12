# 10. Project Structure trong Go

> Dành cho C# developer: Sau khi học các kiểu dữ liệu, goroutine, và interface, câu hỏi thực tế tiếp theo là: "Tôi tổ chức code như thế nào trong một dự án thực?" Đây là chủ đề quan trọng nhất trước khi bạn viết bất kỳ production code nào bằng Go.

---

## 1. Tại sao project structure quan trọng trong Go

Trong C#, Visual Studio và .NET SDK *áp đặt* cấu trúc dự án lên bạn. Bạn có file `.sln` để nhóm các project, file `.csproj` khai báo dependencies, namespace tự động ánh xạ với folder, và NuGet quản lý packages. Tooling dẫn đường cho bạn.

Go không làm vậy. Go compiler không quan tâm bạn đặt file ở đâu, miễn là `package` declaration đúng. Không có file `.sln`, không có file project XML, không có namespace được suy ra từ folder. Điều này nghe có vẻ tự do, nhưng với developer mới, nó thường dẫn đến cấu trúc hỗn độn khó maintain.

Lý do Go chọn hướng này là triết lý của ngôn ngữ: *convention over configuration, nhưng không bắt buộc convention*. Thay vào đó, cộng đồng Go đã hội tụ về một số layout phổ biến qua nhiều năm. Hiểu *tại sao* các convention này tồn tại sẽ giúp bạn áp dụng đúng, thay vì copy mù quáng.

Yếu tố quan trọng nhất trong Go project structure là **package boundary** (ranh giới package). Go kiểm soát khả năng truy cập ở cấp độ package, không phải class (như `internal` trong C#). Nếu bạn không tổ chức package cẩn thận, bạn sẽ gặp circular import — lỗi compiler phổ biến nhất khi code Go lớn dần.

---

## 2. So sánh với C# solution structure

Hãy bắt đầu với thứ bạn đã biết. Một C# REST API điển hình trông như sau:

```
MyApi.sln
├── MyApi.Web/           ← .csproj — ASP.NET controllers, Program.cs
├── MyApi.Application/   ← .csproj — Business logic, CQRS commands
├── MyApi.Domain/        ← .csproj — Entities, value objects
├── MyApi.Infrastructure/← .csproj — EF Core, external services
└── MyApi.Tests/         ← .csproj — Unit/integration tests
```

Mỗi `.csproj` là một *assembly* riêng biệt. Bạn quản lý visibility bằng `public`/`internal`/`protected`. NuGet xử lý external dependencies. `Program.cs` dùng `IServiceCollection` để wire DI container.

Go tương đương trông như sau:

```
myapi/
├── go.mod               ← Tương đương .sln + NuGet config, khai báo module name và dependencies
├── go.sum               ← Lock file, tương đương packages.lock.json
├── cmd/
│   └── api/
│       └── main.go      ← Entry point, tương đương Program.cs
├── internal/            ← Code KHÔNG thể import từ bên ngoài module này
│   ├── handler/         ← Tương đương Controllers
│   ├── service/         ← Tương đương Application layer
│   └── repository/      ← Tương đương Infrastructure/Data
└── pkg/                 ← Code CÓ THỂ import từ module khác (shared utilities)
```

Sự khác biệt cốt lõi: trong C#, isolation được thực thi bởi assembly boundary và access modifiers. Trong Go, isolation được thực thi bởi **directory/package structure** và quy tắc `internal/`. Đây là lý do tổ chức folder trong Go quan trọng hơn nhiều so với C#.

---

## 3. Giải phẫu standard layout

### `go.mod` — Module identity

File `go.mod` là nền tảng của mọi Go project. Nó khai báo *module path* — tên duy nhất để identify module của bạn và là prefix cho mọi import path trong project.

```go
// go.mod
module github.com/mycompany/myapi

go 1.22

require (
    github.com/go-chi/chi/v5 v5.0.12
    github.com/jackc/pgx/v5 v5.5.5
)
```

Module path thường là URL của repository (dù không cần phải là URL thực sự). Khi bạn import một package nội bộ, bạn dùng module path làm prefix:

```go
import "github.com/mycompany/myapi/internal/service"
```

Không có gì tương đương hoàn toàn trong C#. Gần nhất là namespace prefix, nhưng namespace C# không kiểm soát được khả năng import từ bên ngoài.

### `cmd/` — Entry points

Thư mục `cmd/` chứa các entry point cho từng binary executable. Mỗi subdirectory là một application riêng.

```
cmd/
├── api/
│   └── main.go      ← Binary chính: REST API server
├── worker/
│   └── main.go      ← Background job processor
└── migrate/
    └── main.go      ← Database migration tool
```

Tại sao lại cần thư mục `cmd/`? Vì trong một mono-repo Go, bạn có thể build nhiều binary từ cùng một module. Lệnh `go build ./cmd/api` build binary cho API, `go build ./cmd/worker` build binary cho worker. Trong C#, bạn tạo nhiều `.csproj` riêng cho từng executable.

Convention quan trọng: `main.go` trong `cmd/<appname>/` phải **mỏng** (thin). Nó chỉ nên làm 3 việc: load configuration, wire dependencies, start application. Toàn bộ business logic phải nằm ở `internal/`.

### `internal/` — Private packages

Đây là tính năng Go compiler thực sự *enforce*. Bất kỳ package nào nằm trong thư mục `internal/` **không thể được import** bởi code bên ngoài module hiện tại. Nếu cố import, compiler sẽ báo lỗi ngay lập tức.

Điều này tương đương với `internal` access modifier trong C#, nhưng được áp dụng ở cấp độ package path thay vì assembly.

```go
// ĐÂY SẼ COMPILE ERROR nếu code này nằm ngoài module github.com/mycompany/myapi
import "github.com/mycompany/myapi/internal/service"
// Error: use of internal package github.com/mycompany/myapi/internal/service not allowed
```

Mọi business logic, database access, và application-specific code nên nằm trong `internal/`. Điều này bảo vệ implementation details của bạn khỏi bị depend on bởi consumers bên ngoài.

### `pkg/` — Public shared packages

Thư mục `pkg/` là nơi chứa code bạn *muốn* chia sẻ với các module khác. Ví dụ: custom error types, utility functions, shared domain types.

Tuy nhiên, `pkg/` gây tranh cãi trong cộng đồng Go. Nhiều project lớn (kể cả một số của Google) đặt public packages thẳng ở root level thay vì trong `pkg/`. Quan điểm của tôi: dùng `pkg/` khi bạn cần phân biệt rõ ràng giữa "code nội bộ" và "code chia sẻ". Với project nhỏ-vừa, có thể bỏ qua `pkg/` và dùng `internal/` cho tất cả.

---

## 4. Real-world REST API project structure

Đây là cấu trúc của một REST API thực tế mà bạn sẽ thấy trong nhiều Go project ở các công ty:

```
myapi/
│
├── go.mod
├── go.sum
│
├── cmd/
│   └── api/
│       └── main.go              ← Entry point: wire everything, start HTTP server
│
├── internal/
│   ├── config/
│   │   └── config.go            ← Config struct, loading từ env/file
│   │
│   ├── domain/                  ← Core business types (tương đương Domain layer C#)
│   │   ├── user.go              ← User entity, value objects
│   │   └── errors.go            ← Domain-specific errors
│   │
│   ├── handler/                 ← HTTP handlers (tương đương Controllers)
│   │   ├── user_handler.go
│   │   ├── health_handler.go
│   │   └── middleware.go        ← Auth, logging middleware
│   │
│   ├── service/                 ← Business logic (tương đương Application/Service layer)
│   │   └── user_service.go
│   │
│   ├── repository/              ← Data access (tương đương Infrastructure/Repository)
│   │   └── user_repository.go
│   │
│   └── platform/                ← Cross-cutting concerns: DB, cache, external services
│       ├── database/
│       │   └── postgres.go      ← DB connection setup
│       └── logger/
│           └── logger.go        ← Logger initialization
│
├── pkg/
│   └── httputil/                ← Shared HTTP utilities (có thể dùng ở module khác)
│       └── response.go          ← Standard JSON response helpers
│
├── api/
│   └── openapi.yaml             ← API specification (không phải Go code)
│
├── configs/
│   └── config.yaml              ← Default config values
│
└── scripts/
    ├── migrate.sh
    └── seed.sh
```

Cấu trúc này phản ánh **layered architecture**: request đi vào qua `handler`, được xử lý bởi `service`, dữ liệu được lấy bởi `repository`. Mỗi layer chỉ biết về layer bên dưới thông qua interface, không phải concrete type. Đây chính là cách Go implement Dependency Inversion Principle — không cần DI container.

---

## 5. `main.go` — Thin entry point

Một trong những sai lầm phổ biến nhất của developer mới là nhét logic vào `main.go`. Trong Go, `main.go` phải là file mỏng nhất trong project.

```go
// BAD: main.go với quá nhiều logic
package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    // ...
)

func main() {
    // KHÔNG NÊN: Kết nối DB trực tiếp trong main
    db, err := sql.Open("postgres", "postgres://...")
    if err != nil {
        log.Fatal(err)
    }

    // KHÔNG NÊN: Define handler trực tiếp trong main
    http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT * FROM users")
        // ... SQL logic thẳng trong handler
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        var users []map[string]interface{}
        // ... decode rows
        json.NewEncoder(w).Encode(users)
    })

    // KHÔNG NÊN: Hard-code config
    fmt.Println("Starting server on :8080")
    http.ListenAndServe(":8080", nil)
}
```

```go
// GOOD: main.go mỏng, chỉ wire dependencies và start
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/mycompany/myapi/internal/config"
    "github.com/mycompany/myapi/internal/platform/database"
    "github.com/mycompany/myapi/internal/handler"
    "github.com/mycompany/myapi/internal/repository"
    "github.com/mycompany/myapi/internal/service"
)

func main() {
    // 1. Load config từ environment variables
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    // 2. Khởi tạo logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: cfg.LogLevel,
    }))

    // 3. Kết nối infrastructure (DB, cache...)
    db, err := database.Connect(cfg.DatabaseURL)
    if err != nil {
        logger.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer db.Close()

    // 4. Wire các layer (manual DI — thay thế cho Autofac/MS.DI)
    userRepo := repository.NewUserRepository(db)
    userSvc  := service.NewUserService(userRepo, logger)
    userHdlr := handler.NewUserHandler(userSvc, logger)

    // 5. Setup router và start server
    router := handler.NewRouter(userHdlr, logger)
    server := &http.Server{Addr: cfg.Addr, Handler: router}

    // 6. Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        logger.Info("server started", "addr", cfg.Addr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    <-ctx.Done()
    logger.Info("shutting down...")
    server.Shutdown(context.Background())
}
```

`main.go` tốt có thể đọc như một **bản tóm tắt kiến trúc** của toàn bộ application: load config, connect infrastructure, wire layers, start server, handle shutdown. Không có business logic nào ở đây.

---

## 6. Dependency Injection trong Go — Không có DI container

Đây là điểm khác biệt lớn nhất giữa Go và C# với .NET.

Trong C#, bạn đăng ký services vào DI container:

```csharp
// C# — DI container đảm nhiệm việc khởi tạo
builder.Services.AddScoped<IUserRepository, UserRepository>();
builder.Services.AddScoped<IUserService, UserService>();
builder.Services.AddScoped<UserController>();
// Container tự resolve dependencies khi cần
```

Trong Go, bạn tự khởi tạo và truyền dependencies bằng tay — đây gọi là **manual wiring** hay **constructor injection**:

```go
// Go — bạn tự wire, không có container
userRepo := repository.NewUserRepository(db)
userSvc  := service.NewUserService(userRepo, logger)
userHdlr := handler.NewUserHandler(userSvc, logger)
```

Tại sao Go không có DI container? Vì Go ưu tiên *explicit over implicit*. Khi bạn đọc `main.go`, bạn thấy ngay mọi dependency được tạo ở đâu và truyền vào đâu. Không có magic, không có reflection, không cần debug "tại sao dependency này không được resolve". Trade-off là với project rất lớn (hàng trăm services), việc wire thủ công trở nên verbose.

### Constructor function pattern

Mỗi layer export một `New*` function nhận dependencies và trả về instance:

```go
// internal/repository/user_repository.go
package repository

import (
    "context"
    "database/sql"
    "github.com/mycompany/myapi/internal/domain"
)

// UserRepository là struct chứa dependencies
type UserRepository struct {
    db *sql.DB
}

// NewUserRepository là "constructor" — tương đương constructor injection trong C#
// Dependencies được inject tường minh, không có DI container
func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
    var u domain.User
    err := r.db.QueryRowContext(ctx,
        "SELECT id, email, name FROM users WHERE id = $1", id,
    ).Scan(&u.ID, &u.Email, &u.Name)
    if err == sql.ErrNoRows {
        return nil, domain.ErrUserNotFound
    }
    return &u, err
}
```

```go
// internal/service/user_service.go
package service

import (
    "context"
    "log/slog"
    "github.com/mycompany/myapi/internal/domain"
)

// UserRepository là interface — service chỉ biết về interface, không biết về implementation
// Đây chính là Dependency Inversion Principle trong Go, không cần DI container
type UserRepository interface {
    FindByID(ctx context.Context, id int64) (*domain.User, error)
}

type UserService struct {
    repo   UserRepository  // dependency được inject vào
    logger *slog.Logger
}

// NewUserService nhận interface, không phải concrete type
// Cho phép dễ dàng mock trong tests
func NewUserService(repo UserRepository, logger *slog.Logger) *UserService {
    return &UserService{repo: repo, logger: logger}
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*domain.User, error) {
    s.logger.Info("fetching user", "id", id)
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user %d: %w", id, err)
    }
    return user, nil
}
```

### Khi nào dùng `google/wire`

Khi project có hàng chục services với dependency phức tạp, bạn có thể dùng [google/wire](https://github.com/google/wire) — một code generation tool (không phải runtime DI container). Wire đọc các `New*` functions, phân tích dependency graph, và *generate* Go code để wire chúng. Code được generate vẫn là plain Go, bạn vẫn có thể đọc và debug được.

Với project vừa và nhỏ (dưới 20-30 services), manual wiring trong `main.go` là đủ và rõ ràng hơn.

---

## 7. Layered Architecture: Handler → Service → Repository

Đây là pattern được dùng rộng rãi nhất trong Go REST APIs, tương đương với N-tier architecture hoặc Clean Architecture trong C#.

```
HTTP Request
     │
     ▼
┌─────────────┐
│   Handler   │  Parse request, validate input, call service, format response
│  (HTTP)     │  KHÔNG chứa business logic
└──────┬──────┘
       │ gọi qua interface
       ▼
┌─────────────┐
│   Service   │  Business logic, orchestration, validation rules
│  (Business) │  KHÔNG biết về HTTP, KHÔNG biết DB implementation
└──────┬──────┘
       │ gọi qua interface
       ▼
┌─────────────┐
│ Repository  │  Data access, SQL queries, cache
│   (Data)    │  KHÔNG biết về business rules
└─────────────┘
```

Mỗi layer giao tiếp với layer dưới thông qua **interface được định nghĩa ở layer trên**. Đây là điểm Go khác với nhiều ngôn ngữ khác: interface được định nghĩa ở phía *consumer*, không phải phía *implementor*. Service định nghĩa `UserRepository` interface mà nó cần, repository implement interface đó — không phải ngược lại.

```go
// internal/handler/user_handler.go
package handler

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"

    "log/slog"
    "github.com/mycompany/myapi/internal/domain"
)

// Handler định nghĩa interface nó cần từ service layer
// Interface nằm ở phía consumer (handler), không phải phía service
type UserServicer interface {
    GetUser(ctx context.Context, id int64) (*domain.User, error)
}

type UserHandler struct {
    svc    UserServicer
    logger *slog.Logger
}

func NewUserHandler(svc UserServicer, logger *slog.Logger) *UserHandler {
    return &UserHandler{svc: svc, logger: logger}
}

// GetUser xử lý GET /users/{id}
// Handler chỉ quan tâm đến HTTP: parse, validate, respond
// Business logic nằm hoàn toàn ở service layer
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    idStr := r.PathValue("id") // Go 1.22+
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "invalid user id", http.StatusBadRequest)
        return
    }

    user, err := h.svc.GetUser(r.Context(), id)
    if err != nil {
        // Map domain errors sang HTTP status codes
        if errors.Is(err, domain.ErrUserNotFound) {
            http.Error(w, "user not found", http.StatusNotFound)
            return
        }
        h.logger.Error("get user failed", "error", err, "id", id)
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

---

## 8. Tránh Circular Dependencies

Circular import là lỗi compiler thường gặp nhất khi Go project lớn dần. Go compiler *không cho phép* circular import — đây là lý do tại sao cấu trúc package phải được thiết kế cẩn thận.

```go
// BAD: Circular dependency — service import handler, handler import service
// Đây sẽ gây compile error: import cycle not allowed

// package service
import "github.com/mycompany/myapi/internal/handler" // SAI: service không được biết về handler

// package handler  
import "github.com/mycompany/myapi/internal/service" // SAI nếu service đã import handler
```

```go
// GOOD: Dependency chỉ chảy một chiều — handler → service → repository → domain
// Không có package nào import package ở "trên" mình trong hierarchy

// package domain — không import gì từ internal/
type User struct {
    ID    int64
    Email string
    Name  string
}

// package repository — chỉ import domain
import "github.com/mycompany/myapi/internal/domain"

// package service — chỉ import domain, KHÔNG import repository package
// service định nghĩa interface của repository mà nó cần
type UserRepository interface { // interface nằm trong service package
    FindByID(ctx context.Context, id int64) (*domain.User, error)
}

// package handler — chỉ import domain, KHÔNG import service package
// handler định nghĩa interface của service mà nó cần
type UserServicer interface { // interface nằm trong handler package
    GetUser(ctx context.Context, id int64) (*domain.User, error)
}
```

Quy tắc đơn giản để tránh circular import: **dependency graph phải là DAG (Directed Acyclic Graph)**. Vẽ sơ đồ dependency, nếu thấy vòng tròn — bạn cần extract shared code vào package thứ ba (thường là `domain/`) hoặc đảo ngược dependency bằng interface.

Cách phổ biến nhất để break cycle: đặt shared types vào `domain/` package. `domain/` không import bất kỳ package nội bộ nào khác — nó là "trung tâm vũ trụ" mà mọi layer đều có thể import.

---

## 9. Configuration Struct Pattern

Trong C# với `Microsoft.Extensions.Configuration`, bạn bind config vào typed classes và inject chúng qua DI. Trong Go, pattern phổ biến là một `Config` struct duy nhất được load một lần lúc startup và truyền vào mọi component cần nó.

```go
// BAD: Đọc config rải rác khắp nơi trong code
// Khó test, khó biết component nào cần config gì

// trong repository
func NewUserRepository() *UserRepository {
    // KHÔNG NÊN: Đọc env var trực tiếp trong constructor
    dbURL := os.Getenv("DATABASE_URL")
    db, _ := sql.Open("postgres", dbURL)
    return &UserRepository{db: db}
}

// trong handler
func NewUserHandler() *UserHandler {
    // KHÔNG NÊN: Đọc config ở đây, không thể test được
    timeout, _ := strconv.Atoi(os.Getenv("REQUEST_TIMEOUT"))
    return &UserHandler{timeout: time.Duration(timeout) * time.Second}
}
```

```go
// GOOD: Config struct tập trung, load một lần ở main.go
// internal/config/config.go
package config

import (
    "fmt"
    "log/slog"
    "os"
    "time"
)

// Config chứa toàn bộ configuration của application
// Rõ ràng: bạn biết chính xác app cần những config gì
type Config struct {
    // Server
    Addr         string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration

    // Database
    DatabaseURL     string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration

    // Logging
    LogLevel slog.Level

    // Feature flags
    EnableMetrics bool
}

// Load đọc config từ environment variables
// Trả về error nếu required config bị thiếu
func Load() (*Config, error) {
    cfg := &Config{
        // Giá trị mặc định
        Addr:            getEnv("ADDR", ":8080"),
        ReadTimeout:     getDuration("READ_TIMEOUT", 5*time.Second),
        WriteTimeout:    getDuration("WRITE_TIMEOUT", 10*time.Second),
        MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 25),
        MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 5),
        ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
        EnableMetrics:   getBool("ENABLE_METRICS", false),
    }

    // Required config — trả về error nếu thiếu
    cfg.DatabaseURL = os.Getenv("DATABASE_URL")
    if cfg.DatabaseURL == "" {
        return nil, fmt.Errorf("DATABASE_URL is required")
    }

    return cfg, nil
}

func getEnv(key, defaultVal string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultVal
}

func getDuration(key string, defaultVal time.Duration) time.Duration {
    v := os.Getenv(key)
    if v == "" {
        return defaultVal
    }
    d, err := time.ParseDuration(v)
    if err != nil {
        return defaultVal
    }
    return d
}

func getInt(key string, defaultVal int) int {
    v := os.Getenv(key)
    if v == "" {
        return defaultVal
    }
    n, err := strconv.Atoi(v)
    if err != nil {
        return defaultVal
    }
    return n
}

func getBool(key string, defaultVal bool) bool {
    v := os.Getenv(key)
    if v == "" {
        return defaultVal
    }
    b, err := strconv.ParseBool(v)
    if err != nil {
        return defaultVal
    }
    return b
}
```

`Config` struct được tạo một lần trong `main.go`, sau đó các phần cần thiết được truyền vào từng component. Bạn không cần truyền toàn bộ `Config` vào mọi nơi — chỉ truyền những field thực sự cần:

```go
// main.go: truyền chỉ những gì cần thiết
db, _ := database.Connect(cfg.DatabaseURL, cfg.MaxOpenConns, cfg.MaxIdleConns)
server := &http.Server{
    Addr:         cfg.Addr,
    ReadTimeout:  cfg.ReadTimeout,
    WriteTimeout: cfg.WriteTimeout,
}
```

---

## 10. Complete Example: Go REST API với proper structure

Dưới đây là một ví dụ hoàn chỉnh — đủ nhỏ để chạy được, đủ lớn để minh họa đúng cấu trúc. Đây là "User API" với một endpoint GET /users/{id}.

### File tree

```
userapi/
├── go.mod
├── cmd/
│   └── api/
│       └── main.go
└── internal/
    ├── config/
    │   └── config.go
    ├── domain/
    │   └── user.go
    ├── repository/
    │   └── user_repository.go
    ├── service/
    │   └── user_service.go
    └── handler/
        ├── router.go
        └── user_handler.go
```

### `go.mod`

```
module github.com/example/userapi

go 1.22
```

### `internal/domain/user.go`

```go
// package domain chứa core business types.
// Đây là package "trung tâm" — không import bất kỳ internal package nào khác.
// Tương đương Domain layer trong Clean Architecture C#.
package domain

import "errors"

// User là core entity của application
type User struct {
    ID    int64  `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
}

// Domain errors — được định nghĩa ở đây để mọi layer có thể dùng
// Handler sẽ map các error này sang HTTP status codes
var (
    ErrUserNotFound = errors.New("user not found")
    ErrInvalidInput = errors.New("invalid input")
)
```

### `internal/config/config.go`

```go
package config

import (
    "fmt"
    "os"
)

type Config struct {
    Addr string
}

// Load đọc config từ environment. Trả về defaults nếu không set.
func Load() (*Config, error) {
    cfg := &Config{
        Addr: getEnv("ADDR", ":8080"),
    }
    _ = fmt.Sprintf // đảm bảo fmt được import nếu cần trong future
    return cfg, nil
}

func getEnv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}
```

### `internal/repository/user_repository.go`

```go
// package repository chứa data access logic.
// Trong ví dụ này dùng in-memory store thay vì DB thật để có thể chạy ngay.
package repository

import (
    "context"
    "sync"

    "github.com/example/userapi/internal/domain"
)

// UserRepository lưu trữ users trong memory (thay thế DB cho ví dụ này)
type UserRepository struct {
    mu    sync.RWMutex
    users map[int64]*domain.User
}

// NewUserRepository khởi tạo repository với một số dữ liệu mẫu
func NewUserRepository() *UserRepository {
    return &UserRepository{
        users: map[int64]*domain.User{
            1: {ID: 1, Email: "alice@example.com", Name: "Alice"},
            2: {ID: 2, Email: "bob@example.com", Name: "Bob"},
        },
    }
}

// FindByID tìm user theo ID. Trả về domain.ErrUserNotFound nếu không có.
func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    user, ok := r.users[id]
    if !ok {
        return nil, domain.ErrUserNotFound
    }
    // Trả về copy để tránh caller modify data trong map
    copy := *user
    return &copy, nil
}
```

### `internal/service/user_service.go`

```go
// package service chứa business logic.
// Service không biết gì về HTTP — nó chỉ biết về domain và repository interface.
package service

import (
    "context"
    "fmt"
    "log/slog"

    "github.com/example/userapi/internal/domain"
)

// UserRepo là interface mà service cần từ repository layer.
// Interface được định nghĩa ở đây (consumer), không phải ở repository package.
// Đây là cách Go implement Dependency Inversion — không cần DI container.
type UserRepo interface {
    FindByID(ctx context.Context, id int64) (*domain.User, error)
}

// UserService xử lý business logic liên quan đến User.
type UserService struct {
    repo   UserRepo
    logger *slog.Logger
}

// NewUserService là constructor — tương đương constructor injection trong C#.
func NewUserService(repo UserRepo, logger *slog.Logger) *UserService {
    return &UserService{repo: repo, logger: logger}
}

// GetUser lấy thông tin user theo ID.
// Business rule: ID phải > 0.
func (s *UserService) GetUser(ctx context.Context, id int64) (*domain.User, error) {
    // Business validation — không phải HTTP validation
    if id <= 0 {
        return nil, fmt.Errorf("%w: id must be positive", domain.ErrInvalidInput)
    }

    s.logger.InfoContext(ctx, "fetching user", "id", id)

    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        // Wrap error với context — errors.Is vẫn hoạt động được
        return nil, fmt.Errorf("get user %d: %w", id, err)
    }
    return user, nil
}
```

### `internal/handler/user_handler.go`

```go
// package handler chứa HTTP handlers.
// Handler chỉ quan tâm đến HTTP protocol: parse request, gọi service, format response.
// Business logic không được đặt ở đây.
package handler

import (
    "context"
    "encoding/json"
    "errors"
    "log/slog"
    "net/http"
    "strconv"

    "github.com/example/userapi/internal/domain"
)

// UserSvc là interface mà handler cần từ service layer.
// Lại một lần nữa: interface nằm ở phía consumer (handler).
type UserSvc interface {
    GetUser(ctx context.Context, id int64) (*domain.User, error)
}

// UserHandler xử lý HTTP requests liên quan đến User.
type UserHandler struct {
    svc    UserSvc
    logger *slog.Logger
}

func NewUserHandler(svc UserSvc, logger *slog.Logger) *UserHandler {
    return &UserHandler{svc: svc, logger: logger}
}

// GetUser xử lý GET /users/{id}
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    // Parse path parameter
    idStr := r.PathValue("id") // Yêu cầu Go 1.22+
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        h.writeError(w, "invalid user id", http.StatusBadRequest)
        return
    }

    user, err := h.svc.GetUser(r.Context(), id)
    if err != nil {
        // Map domain errors sang HTTP status codes — đây là trách nhiệm của handler
        switch {
        case errors.Is(err, domain.ErrUserNotFound):
            h.writeError(w, "user not found", http.StatusNotFound)
        case errors.Is(err, domain.ErrInvalidInput):
            h.writeError(w, err.Error(), http.StatusBadRequest)
        default:
            h.logger.ErrorContext(r.Context(), "unexpected error", "error", err)
            h.writeError(w, "internal server error", http.StatusInternalServerError)
        }
        return
    }

    // Trả về JSON response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

// writeError là helper để viết JSON error response nhất quán
func (h *UserHandler) writeError(w http.ResponseWriter, msg string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

### `internal/handler/router.go`

```go
package handler

import (
    "log/slog"
    "net/http"
)

// NewRouter tạo HTTP router với tất cả routes được đăng ký.
// Tách router ra file riêng giúp main.go gọn hơn và dễ test.
func NewRouter(userHdlr *UserHandler, logger *slog.Logger) http.Handler {
    mux := http.NewServeMux()

    // Route patterns với path parameters — Go 1.22+
    mux.HandleFunc("GET /users/{id}", userHdlr.GetUser)

    // Health check endpoint — quan trọng cho Kubernetes/load balancer
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })

    return mux
}
```

### `cmd/api/main.go`

```go
// main.go là entry point duy nhất của application.
// Nó chỉ làm một việc: wire tất cả dependencies và start server.
// Không có business logic ở đây — chỉ có "plumbing" (ống nước kết nối các component).
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/example/userapi/internal/config"
    "github.com/example/userapi/internal/handler"
    "github.com/example/userapi/internal/repository"
    "github.com/example/userapi/internal/service"
)

func main() {
    // Bước 1: Load config — thất bại sớm nếu config sai
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    // Bước 2: Khởi tạo logger — tất cả component dùng chung logger này
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    // Bước 3: Wire các layer theo thứ tự từ dưới lên
    // repository ← service ← handler
    // Mỗi layer nhận dependencies từ layer bên ngoài
    userRepo := repository.NewUserRepository()
    userSvc  := service.NewUserService(userRepo, logger)
    userHdlr := handler.NewUserHandler(userSvc, logger)

    // Bước 4: Tạo router với tất cả handlers
    router := handler.NewRouter(userHdlr, logger)

    // Bước 5: Configure HTTP server
    server := &http.Server{
        Addr:         cfg.Addr,
        Handler:      router,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    // Bước 6: Graceful shutdown — lắng nghe SIGINT/SIGTERM
    // Quan trọng trong production: không ngắt kết nối đang xử lý
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        logger.Info("server started", "addr", cfg.Addr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Error("server failed", "error", err)
            os.Exit(1)
        }
    }()

    // Chờ shutdown signal
    <-ctx.Done()
    logger.Info("shutting down server...")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(shutdownCtx); err != nil {
        logger.Error("graceful shutdown failed", "error", err)
    }

    logger.Info("server stopped")
}
```

Để chạy ví dụ này:

```bash
# Khởi tạo module (nếu chưa có go.mod)
go mod init github.com/example/userapi

# Chạy server
go run ./cmd/api

# Test trong terminal khác
curl http://localhost:8080/health
curl http://localhost:8080/users/1
curl http://localhost:8080/users/999   # Sẽ trả về 404
curl http://localhost:8080/users/abc   # Sẽ trả về 400
```

---

## Common Issues

| Vấn đề | Nguyên nhân | Giải pháp |
|---|---|---|
| `import cycle not allowed` | Package A import B, B import A | Extract shared types vào `domain/` package; dùng interface để đảo ngược dependency |
| `use of internal package not allowed` | Code ngoài module cố import `internal/` | Đây là feature — refactor để expose qua `pkg/` nếu cần share |
| Logic trong `main.go` ngày càng phình to | Không tách layer rõ ràng từ đầu | Tạo packages mới trong `internal/`, move code ra |
| Khó mock dependency trong tests | Dùng concrete type thay vì interface | Định nghĩa interface ở phía consumer, inject qua constructor |
| `undefined: strconv` hoặc tương tự | Quên import | Go không auto-import; dùng `goimports` hoặc IDE plugin |
| Config bị scatter khắp nơi (`os.Getenv` random) | Không có config struct tập trung | Tạo `internal/config/config.go`, load một lần trong `main.go` |
| Package name không khớp với directory name | Convention bị vi phạm | Package name phải khớp với directory name (trừ `main`) |
| Nhiều files trong một package có `package main` | Nhầm lẫn về package | Chỉ `cmd/*/main.go` mới dùng `package main`; internal packages dùng tên riêng |

---

## Best Practices

Go project structure tốt bắt đầu từ việc hiểu rằng *package là đơn vị của encapsulation*, không phải class. Khi bạn đặt code vào một package, bạn đang tuyên bố: "đây là một nhóm chức năng liên quan". Vì vậy, package name phải là danh từ mô tả rõ ràng nhóm đó — `repository`, `service`, `handler` — không phải tên chung chung như `utils`, `common`, `helpers`. Package tên `utils` là dấu hiệu của package đang làm quá nhiều việc.

Khi bạn thiết kế interfaces, hãy đặt chúng ở phía consumer. Đây là khác biệt lớn nhất so với C#. Trong C#, bạn thường định nghĩa `IUserRepository` trong Infrastructure project rồi implement nó. Trong Go, `service` package định nghĩa interface `UserRepo` mà nó cần — và `repository` package implement interface đó mà không cần biết interface đó tồn tại. Cách này giúp packages ít coupled hơn và dễ test hơn.

Hãy resist temptation tạo `utils/` package. Nếu bạn có utility functions, hỏi xem chúng thực sự thuộc về đâu. HTTP response helpers thuộc về `httputil`. String processing cho domain cụ thể thuộc về domain package đó. Chỉ khi function thực sự generic và không thuộc về bất kỳ domain nào, mới đặt vào `pkg/` với tên package mô tả cụ thể.

Về việc đặt tên file: Go không có quy tắc cứng về tên file, nhưng convention phổ biến là `{entity}_{layer}.go` — `user_handler.go`, `user_service.go`, `user_repository.go`. Khi test, file test phải cùng package hoặc dùng package `_test` suffix: `user_service_test.go`.

Cuối cùng, hãy kiểm tra dependency graph thường xuyên. Công cụ `go mod graph` và `go list -f '{{.ImportPath}}: {{.Imports}}'` giúp bạn visualize dependencies. Nếu thấy một package import quá nhiều thứ, đó là dấu hiệu cần tách nhỏ hơn.

---

## Exercises

### Exercise 1: Thêm endpoint tạo User

Mở rộng project ví dụ ở trên để hỗ trợ `POST /users` tạo user mới. Yêu cầu:

1. Request body là JSON: `{"email": "...", "name": "..."}`
2. Validate: email và name không được rỗng (ở service layer)
3. Repository cần method `Save(ctx, user) error`
4. Response trả về user vừa tạo với ID được generate (dùng `sync/atomic` để generate ID)
5. Đúng layered architecture: validation ở service, HTTP parsing ở handler

**Giải pháp:**

```go
// internal/domain/user.go — thêm CreateUserInput
package domain

import "errors"

type User struct {
    ID    int64  `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
}

// CreateUserInput là input DTO cho việc tạo user
// Tách khỏi User entity để không expose ID (được generate bởi server)
type CreateUserInput struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

var (
    ErrUserNotFound = errors.New("user not found")
    ErrInvalidInput = errors.New("invalid input")
)
```

```go
// internal/repository/user_repository.go — thêm Save method
package repository

import (
    "context"
    "sync"
    "sync/atomic"

    "github.com/example/userapi/internal/domain"
)

type UserRepository struct {
    mu     sync.RWMutex
    users  map[int64]*domain.User
    nextID int64 // dùng atomic để thread-safe
}

func NewUserRepository() *UserRepository {
    return &UserRepository{
        users: map[int64]*domain.User{
            1: {ID: 1, Email: "alice@example.com", Name: "Alice"},
            2: {ID: 2, Email: "bob@example.com", Name: "Bob"},
        },
        nextID: 2, // bắt đầu từ 2 vì đã có 1, 2
    }
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    user, ok := r.users[id]
    if !ok {
        return nil, domain.ErrUserNotFound
    }
    copy := *user
    return &copy, nil
}

// Save lưu user mới với ID được generate
func (r *UserRepository) Save(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
    // Generate ID thread-safe với atomic
    id := atomic.AddInt64(&r.nextID, 1)

    user := &domain.User{
        ID:    id,
        Email: input.Email,
        Name:  input.Name,
    }

    r.mu.Lock()
    defer r.mu.Unlock()
    r.users[id] = user

    copy := *user
    return &copy, nil
}
```

```go
// internal/service/user_service.go — thêm CreateUser method
package service

import (
    "context"
    "fmt"
    "log/slog"
    "strings"

    "github.com/example/userapi/internal/domain"
)

// UserRepo interface — thêm Save
type UserRepo interface {
    FindByID(ctx context.Context, id int64) (*domain.User, error)
    Save(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
}

type UserService struct {
    repo   UserRepo
    logger *slog.Logger
}

func NewUserService(repo UserRepo, logger *slog.Logger) *UserService {
    return &UserService{repo: repo, logger: logger}
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*domain.User, error) {
    if id <= 0 {
        return nil, fmt.Errorf("%w: id must be positive", domain.ErrInvalidInput)
    }
    s.logger.InfoContext(ctx, "fetching user", "id", id)
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user %d: %w", id, err)
    }
    return user, nil
}

// CreateUser validate và tạo user mới.
// Validation business rules ở đây — không phải trong handler.
func (s *UserService) CreateUser(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
    // Validate — đây là business logic, không phải HTTP concerns
    input.Email = strings.TrimSpace(input.Email)
    input.Name  = strings.TrimSpace(input.Name)

    if input.Email == "" {
        return nil, fmt.Errorf("%w: email is required", domain.ErrInvalidInput)
    }
    if input.Name == "" {
        return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
    }
    if !strings.Contains(input.Email, "@") {
        return nil, fmt.Errorf("%w: email is not valid", domain.ErrInvalidInput)
    }

    s.logger.InfoContext(ctx, "creating user", "email", input.Email)

    user, err := s.repo.Save(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("create user: %w", err)
    }
    return user, nil
}
```

```go
// internal/handler/user_handler.go — thêm CreateUser handler
package handler

import (
    "context"
    "encoding/json"
    "errors"
    "log/slog"
    "net/http"
    "strconv"

    "github.com/example/userapi/internal/domain"
)

type UserSvc interface {
    GetUser(ctx context.Context, id int64) (*domain.User, error)
    CreateUser(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
}

type UserHandler struct {
    svc    UserSvc
    logger *slog.Logger
}

func NewUserHandler(svc UserSvc, logger *slog.Logger) *UserHandler {
    return &UserHandler{svc: svc, logger: logger}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    idStr := r.PathValue("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        h.writeError(w, "invalid user id", http.StatusBadRequest)
        return
    }

    user, err := h.svc.GetUser(r.Context(), id)
    if err != nil {
        h.handleError(w, r, err)
        return
    }

    h.writeJSON(w, http.StatusOK, user)
}

// CreateUser xử lý POST /users
// Handler chỉ lo parse JSON và delegate sang service
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var input domain.CreateUserInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        h.writeError(w, "invalid JSON body", http.StatusBadRequest)
        return
    }

    user, err := h.svc.CreateUser(r.Context(), input)
    if err != nil {
        h.handleError(w, r, err)
        return
    }

    // 201 Created với user mới trong body
    h.writeJSON(w, http.StatusCreated, user)
}

// handleError map domain errors sang HTTP status — logic tập trung ở một nơi
func (h *UserHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
    switch {
    case errors.Is(err, domain.ErrUserNotFound):
        h.writeError(w, "user not found", http.StatusNotFound)
    case errors.Is(err, domain.ErrInvalidInput):
        h.writeError(w, err.Error(), http.StatusBadRequest)
    default:
        h.logger.ErrorContext(r.Context(), "unexpected error", "error", err)
        h.writeError(w, "internal server error", http.StatusInternalServerError)
    }
}

func (h *UserHandler) writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}

func (h *UserHandler) writeError(w http.ResponseWriter, msg string, status int) {
    h.writeJSON(w, status, map[string]string{"error": msg})
}
```

```go
// internal/handler/router.go — đăng ký route mới
package handler

import (
    "log/slog"
    "net/http"
)

func NewRouter(userHdlr *UserHandler, logger *slog.Logger) http.Handler {
    mux := http.NewServeMux()

    mux.HandleFunc("GET /users/{id}", userHdlr.GetUser)
    mux.HandleFunc("POST /users", userHdlr.CreateUser) // Route mới

    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })

    return mux
}
```

Test:

```bash
go run ./cmd/api

# Tạo user mới
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"email":"charlie@example.com","name":"Charlie"}'
# Kết quả: {"id":3,"email":"charlie@example.com","name":"Charlie"}

# Validate: email rỗng
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"email":"","name":"Dave"}'
# Kết quả: {"error":"invalid input: email is required"} với status 400
```

---

### Exercise 2: Thêm middleware logging

Thêm middleware log mỗi HTTP request với method, path, status code, và duration. Yêu cầu:

1. Middleware phải là một function trả về `http.Handler`
2. Log dùng `slog` với structured fields
3. Gắn middleware vào router, không phải từng handler riêng lẻ

**Giải pháp:**

```go
// internal/handler/middleware.go
package handler

import (
    "log/slog"
    "net/http"
    "time"
)

// responseWriter là wrapper để capture status code
// http.ResponseWriter không expose status code sau khi WriteHeader được gọi
type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(status int) {
    rw.status = status
    rw.ResponseWriter.WriteHeader(status)
}

// LoggingMiddleware wrap handler với request logging.
// Pattern: nhận http.Handler, trả về http.Handler — composable và reusable.
// Tương đương IMiddleware trong ASP.NET Core, nhưng không cần DI container.
func LoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap ResponseWriter để capture status code
        rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

        // Gọi handler tiếp theo
        next.ServeHTTP(rw, r)

        // Log sau khi handler xử lý xong
        logger.InfoContext(r.Context(), "request",
            "method",   r.Method,
            "path",     r.URL.Path,
            "status",   rw.status,
            "duration", time.Since(start).String(),
            "remote",   r.RemoteAddr,
        )
    })
}
```

```go
// internal/handler/router.go — áp dụng middleware
package handler

import (
    "log/slog"
    "net/http"
)

func NewRouter(userHdlr *UserHandler, logger *slog.Logger) http.Handler {
    mux := http.NewServeMux()

    mux.HandleFunc("GET /users/{id}", userHdlr.GetUser)
    mux.HandleFunc("POST /users", userHdlr.CreateUser)
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })

    // Wrap toàn bộ mux với middleware
    // Thứ tự: request đi vào LoggingMiddleware trước, sau đó mới vào mux
    return LoggingMiddleware(logger, mux)
}
```

Test:

```bash
go run ./cmd/api

curl http://localhost:8080/users/1
# Server log: time=... level=INFO msg=request method=GET path=/users/1 status=200 duration=124µs remote=127.0.0.1:...

curl http://localhost:8080/users/999
# Server log: time=... level=INFO msg=request method=GET path=/users/999 status=404 duration=98µs remote=127.0.0.1:...
```

---

## Tóm tắt

Project structure trong Go không phải là convention ngẫu nhiên — mỗi quyết định đều xuất phát từ đặc tính của ngôn ngữ. `internal/` tồn tại vì Go kiểm soát visibility bằng package path. `cmd/` tồn tại vì một module có thể build nhiều binaries. Layered architecture với interface tồn tại vì Go không có DI container và muốn mọi thứ explicit. `domain/` package tồn tại để phá vỡ circular dependencies.

Khi bạn hiểu *tại sao*, bạn có thể adapt cấu trúc cho project cụ thể của mình thay vì copy blindly. Một API nhỏ có thể không cần `pkg/`. Một microservice đơn giản có thể gộp `service` và `handler` vào một package. Quy tắc bất biến duy nhất: dependency chỉ chảy một chiều, và `main.go` phải mỏng.

---

**Next:** [11. Common Go Patterns](11-common-patterns.md) — Các pattern idiomatic thường gặp: functional options, builder, dependency injection kiểu Go.
