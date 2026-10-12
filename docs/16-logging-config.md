# 16. Logging & Configuration trong Go

> Dành cho developer C# đang học Go. Bài này giải thích **tại sao** Go làm theo cách này, không chỉ **làm thế nào**.

---

## Tại sao Structured Logging quan trọng?

Trong C#, bạn đã quen với Serilog hoặc NLog — các thư viện hỗ trợ "structured logging" bằng cách gắn thêm metadata vào từng dòng log:

```csharp
// C# với Serilog
logger.Information("User {UserId} placed order {OrderId} for {Amount}", userId, orderId, amount);
```

Kết quả xuất ra JSON: `{"level":"info","UserId":42,"OrderId":"abc","Amount":99.5,"message":"User placed order"}`. Điều này cho phép bạn query log theo field, không phải parse text bằng regex.

Go ban đầu chỉ có `log` package cơ bản — chỉ in text, không có level, không có fields. Cộng đồng phải dùng thư viện bên thứ ba như `zap` hay `zerolog`. Từ **Go 1.21**, standard library cuối cùng đã có `log/slog` — structured logging chính thức với JSON/text handler, levels, và context fields. Đây là sự thay đổi lớn vì trước đây mỗi team dùng một thư viện khác nhau, gây ra sự phân mảnh.

Lý do Go chú trọng structured logging: trong môi trường microservices và cloud (Kubernetes, GCP, AWS), log phải được đọc bởi máy (Datadog, Splunk, CloudWatch) trước khi con người đọc. Text thuần không parse được hiệu quả; JSON với fields chuẩn hóa thì có thể query trực tiếp.

---

## 1. log/slog — Standard Library Structured Logger (Go 1.21+)

`slog` thiết kế xung quanh ba khái niệm: **Logger** (đối tượng bạn dùng để log), **Handler** (quyết định format và đích đến), và **Attr/Value** (các cặp key-value metadata).

So sánh tư duy với C#:
- `slog.Logger` ≈ `ILogger<T>` trong ASP.NET Core
- `slog.Handler` ≈ Serilog Sink hoặc NLog Target
- `slog.With(...)` ≈ `logger.ForContext(...)` trong Serilog

### Sử dụng cơ bản

```go
// BAD: dùng global logger mặc định, không có context, output text không parse được
package main

import "log"

func main() {
    // log package cũ — chỉ in text, không có level tách biệt, không có fields
    log.Println("User logged in: 42") // Output: 2024/01/15 10:30:00 User logged in: 42
    // Làm sao query "tất cả login của user 42" trong log aggregator? Không thể!
}
```

```go
// GOOD: dùng slog với structured fields, inject logger không dùng global
package main

import (
    "log/slog"
    "os"
)

func main() {
    // JSONHandler cho production — log aggregator đọc được
    // TextHandler cho development — con người đọc được
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo, // bỏ qua Debug trong production
    }))

    // Mỗi field là một cặp key-value — có thể query trong Datadog/Splunk
    logger.Info("user logged in",
        slog.Int("user_id", 42),
        slog.String("ip", "192.168.1.1"),
        slog.String("method", "oauth"),
    )
    // Output: {"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"user logged in","user_id":42,"ip":"192.168.1.1","method":"oauth"}

    logger.Error("payment failed",
        slog.Int("user_id", 42),
        slog.String("error", "insufficient funds"),
        slog.Float64("amount", 99.5),
    )
}
```

### slog.With — Thêm Context Fields vào Logger

Khi bạn xử lý một HTTP request, mọi dòng log trong request đó nên có `request_id` và `user_id`. Trong Serilog, bạn dùng `ForContext`. Trong slog, bạn dùng `With`:

```go
// BAD: lặp lại request_id ở mỗi dòng log — dễ quên, dễ sai
func handleRequest(logger *slog.Logger, requestID string, userID int) {
    logger.Info("processing request", slog.String("request_id", requestID))
    // ... 10 dòng sau ...
    logger.Info("fetching user data", slog.String("request_id", requestID), slog.Int("user_id", userID))
    // ... quên thêm request_id ở dòng này ...
    logger.Info("calling payment service") // không có context!
}
```

```go
// GOOD: tạo child logger với context fields — tự động gắn vào mọi dòng log
func handleRequest(logger *slog.Logger, requestID string, userID int) {
    // With() trả về logger mới (immutable), không sửa logger gốc
    reqLogger := logger.With(
        slog.String("request_id", requestID),
        slog.Int("user_id", userID),
    )

    reqLogger.Info("processing request")
    // Output: {..., "request_id":"abc-123", "user_id":42, "msg":"processing request"}

    reqLogger.Info("fetching user data")
    // Output: {..., "request_id":"abc-123", "user_id":42, "msg":"fetching user data"}

    reqLogger.Info("calling payment service")
    // Output: {..., "request_id":"abc-123", "user_id":42, "msg":"calling payment service"}
    // request_id luôn có mặt, không cần lặp lại!
}
```

### Development vs Production Handler

```go
package main

import (
    "log/slog"
    "os"
)

func newLogger(env string) *slog.Logger {
    var handler slog.Handler

    if env == "development" {
        // TextHandler dễ đọc hơn khi dev local
        // Output: time=2024-01-15T10:30:00Z level=INFO msg="server started" port=8080
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level:     slog.LevelDebug, // show Debug logs trong dev
            AddSource: true,            // thêm file:line — giúp debug nhanh hơn
        })
    } else {
        // JSONHandler cho staging/production — log aggregator parse được
        // Output: {"time":"...","level":"INFO","source":{"file":"main.go","line":42},"msg":"..."}
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level:     slog.LevelInfo, // bỏ Debug logs trong production — giảm noise và chi phí
            AddSource: true,
        })
    }

    return slog.New(handler)
}

func main() {
    logger := newLogger(os.Getenv("APP_ENV"))
    logger.Info("server started", slog.Int("port", 8080))
    logger.Debug("debug info — chỉ thấy trong development", slog.String("detail", "xyz"))
}
```

### Logger như Dependency (Inject, không dùng Global)

Trong C# với ASP.NET Core, `ILogger<T>` được inject qua DI container — bạn không bao giờ gọi `LogManager.GetLogger()` static trong production code. Go nên làm tương tự.

```go
// BAD: dùng slog global — khó test, khó swap handler
package main

import "log/slog"

type UserService struct{}

func (s *UserService) GetUser(id int) {
    slog.Info("getting user", slog.Int("id", id)) // global! không thể inject mock logger khi test
}
```

```go
// GOOD: inject logger vào struct — testable, configurable per-component
package main

import (
    "fmt"
    "log/slog"
    "os"
)

type UserService struct {
    logger *slog.Logger // field riêng, không phải global
    // db     *sql.DB  -- các dependency khác inject tương tự
}

func NewUserService(logger *slog.Logger) *UserService {
    // tạo child logger với component name — biết log này đến từ đâu
    return &UserService{
        logger: logger.With(slog.String("component", "UserService")),
    }
}

func (s *UserService) GetUser(id int) (string, error) {
    s.logger.Debug("fetching user from db", slog.Int("user_id", id))

    // giả lập DB call
    if id <= 0 {
        s.logger.Error("invalid user id", slog.Int("user_id", id))
        return "", fmt.Errorf("invalid user id: %d", id)
    }

    s.logger.Info("user fetched successfully", slog.Int("user_id", id))
    return fmt.Sprintf("User_%d", id), nil
}

type OrderService struct {
    logger      *slog.Logger
    userService *UserService
}

func NewOrderService(logger *slog.Logger, userService *UserService) *OrderService {
    return &OrderService{
        logger:      logger.With(slog.String("component", "OrderService")),
        userService: userService,
    }
}

func (s *OrderService) PlaceOrder(userID int, amount float64) error {
    s.logger.Info("placing order",
        slog.Int("user_id", userID),
        slog.Float64("amount", amount),
    )
    _, err := s.userService.GetUser(userID)
    if err != nil {
        return fmt.Errorf("place order: %w", err)
    }
    return nil
}

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    userSvc := NewUserService(logger)
    orderSvc := NewOrderService(logger, userSvc)

    if err := orderSvc.PlaceOrder(42, 99.5); err != nil {
        logger.Error("failed to place order", slog.String("error", err.Error()))
    }
}
```

---

## 2. zerolog — Zero Allocation Structured Logger

`zerolog` là thư viện bên thứ ba phổ biến, được thiết kế để **zero allocation** — không tạo object trên heap trong hot path. Trong C#, điều này tương đương với `ValueStringBuilder` hay `Span<T>` patterns để tránh GC pressure.

API của zerolog dùng **method chaining** — mỗi field là một method call trả về builder, rồi kết thúc bằng `.Msg()` hoặc `.Msgf()`:

```go
// zerolog — cần: go get github.com/rs/zerolog
package main

import (
    "os"
    "time"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

func main() {
    // Development: pretty print có màu sắc
    // zerolog.SetGlobalLevel(zerolog.DebugLevel)
    // log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

    // Production: JSON output
    zerolog.TimeFieldFormat = time.RFC3339
    logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

    // Chaining API — không tạo intermediate strings
    logger.Info().
        Int("user_id", 42).
        Str("action", "login").
        Str("ip", "192.168.1.1").
        Msg("user logged in")
    // Output: {"level":"info","time":"2024-01-15T10:30:00Z","user_id":42,"action":"login","ip":"192.168.1.1","message":"user logged in"}

    // Error với err field tự động
    err := fmt.Errorf("connection refused")
    logger.Error().
        Err(err).                    // tự động thêm "error":"connection refused"
        Str("service", "payment").
        Dur("retry_after", 5*time.Second).
        Msg("service unavailable")

    // Child logger với context cố định — tương tự slog.With()
    requestLogger := logger.With().
        Str("request_id", "abc-123").
        Int("user_id", 42).
        Logger()

    requestLogger.Info().Msg("processing request")
    requestLogger.Info().Str("step", "validation").Msg("input validated")
}
```

Khi nào chọn `slog` vs `zerolog`? Nếu project mới và không có yêu cầu performance đặc biệt, hãy dùng `slog` — nó là standard library, không cần dependency ngoài. Nếu bạn viết high-throughput service (game server, trading system, log aggregator) và profiling cho thấy logging là bottleneck, hãy xem xét `zerolog`. Trong hầu hết business applications, `slog` là đủ.

---

## 3. Logging Best Practices

### Log tại Boundaries, không phải trong Business Logic

Trong kiến trúc layered (Handler → Service → Repository), chỉ nên log tại các "ranh giới" — nơi request đến và đi. Business logic thuần túy (tính toán, validation, transformation) không nên log — nó làm business logic phụ thuộc vào infrastructure concern.

```go
// BAD: log rải rác khắp business logic — khó test, business logic biết về logger
type OrderCalculator struct {
    logger *slog.Logger
}

func (c *OrderCalculator) CalculateTotal(items []Item) float64 {
    c.logger.Debug("starting calculation") // không cần thiết trong pure business logic
    total := 0.0
    for _, item := range items {
        c.logger.Debug("processing item", slog.String("name", item.Name)) // quá nhiều noise
        total += item.Price * float64(item.Quantity)
    }
    c.logger.Debug("calculation done", slog.Float64("total", total))
    return total
}
```

```go
// GOOD: business logic thuần, log tại handler/service boundary
type OrderCalculator struct{} // không cần logger

func (c *OrderCalculator) CalculateTotal(items []Item) float64 {
    total := 0.0
    for _, item := range items {
        total += item.Price * float64(item.Quantity)
    }
    return total // pure function, dễ test, không side effect
}

// Log tại service layer — nơi có đủ context
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) error {
    start := time.Now()
    s.logger.Info("creating order", slog.Int("user_id", req.UserID), slog.Int("item_count", len(req.Items)))

    total := s.calculator.CalculateTotal(req.Items) // gọi pure function, không log trong đó

    if err := s.repo.Save(ctx, req, total); err != nil {
        s.logger.Error("failed to save order",
            slog.Int("user_id", req.UserID),
            slog.String("error", err.Error()),
            slog.Duration("duration", time.Since(start)),
        )
        return fmt.Errorf("create order: %w", err)
    }

    s.logger.Info("order created successfully",
        slog.Int("user_id", req.UserID),
        slog.Float64("total", total),
        slog.Duration("duration", time.Since(start)),
    )
    return nil
}
```

### Log OR Return — Không làm cả hai

Đây là nguyên tắc quan trọng nhất. Nếu một function log lỗi rồi return lỗi đó, caller có thể log lại — kết quả là cùng một lỗi xuất hiện nhiều lần trong log với stack trace khác nhau, rất khó trace.

```go
// BAD: log VÀ return — lỗi sẽ bị log nhiều lần ở nhiều layer
func (r *UserRepository) GetByID(ctx context.Context, id int) (*User, error) {
    user, err := r.db.QueryRow(ctx, "SELECT ...", id)
    if err != nil {
        r.logger.Error("db query failed", slog.String("error", err.Error())) // log ở đây
        return nil, fmt.Errorf("get user: %w", err) // VÀ return error
    }
    return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    user, err := s.repo.GetByID(ctx, id)
    if err != nil {
        s.logger.Error("failed to get user", slog.String("error", err.Error())) // log LẠI!
        return nil, err
    }
    return user, nil
}
// Kết quả: 2 dòng log cho cùng 1 lỗi — confusion khi debug
```

```go
// GOOD: chỉ log ở layer cao nhất (handler), các layer dưới chỉ wrap và return
func (r *UserRepository) GetByID(ctx context.Context, id int) (*User, error) {
    // KHÔNG log ở đây — chỉ wrap error với context
    user, err := r.db.QueryRow(ctx, "SELECT ...", id)
    if err != nil {
        return nil, fmt.Errorf("user repository get by id %d: %w", id, err) // wrap, không log
    }
    return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    user, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("user service get user: %w", err) // wrap thêm, không log
    }
    return user, nil
}

// Handler — đây là boundary, log một lần duy nhất
func (h *UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
    id := parseID(r)
    user, err := h.service.GetUser(r.Context(), id)
    if err != nil {
        // Log một lần với full error chain
        h.logger.Error("get user request failed",
            slog.Int("user_id", id),
            slog.String("error", err.Error()), // chứa cả chain: "handler > service > repository"
        )
        http.Error(w, "internal error", 500)
        return
    }
    json.NewEncoder(w).Encode(user)
}
```

### Không Log Sensitive Data

```go
// BAD: log thông tin nhạy cảm — vi phạm GDPR, PCI DSS
func (s *AuthService) Login(username, password string) {
    s.logger.Info("login attempt",
        slog.String("username", username),
        slog.String("password", password),    // KHÔNG BAO GIỜ! password trong log
        slog.String("credit_card", "4111..."), // KHÔNG BAO GIỜ!
    )
}
```

```go
// GOOD: chỉ log identifier, không log secret values
func (s *AuthService) Login(username, password string) {
    s.logger.Info("login attempt",
        slog.String("username", username), // OK: identifier
        // password: không log — nếu cần debug, log "password_provided: true"
        slog.Bool("password_provided", password != ""),
    )
}
```

---

## 4. Configuration Management — 12-Factor App

Trong C#, configuration thường đến từ `appsettings.json` + `appsettings.{Environment}.json` + environment variables, orchestrated bởi `IConfiguration`. Đây là cách tiếp cận tốt, nhưng đôi khi config file bị commit nhầm vào git với secret values.

Go và cộng đồng cloud-native theo nguyên tắc **12-Factor App** (12factor.net): **config phải đến từ environment variables, không phải file**. Lý do:

Khi deploy lên Kubernetes hoặc cloud, environment variables là cách tự nhiên để inject config khác nhau cho mỗi environment mà không cần thay đổi Docker image. Config file thường bị hardcode path, khó override trong container, và nguy hiểm nếu chứa secret (có thể bị git commit).

---

## 5. godotenv — Load .env File trong Development

Trong production (Kubernetes, ECS, Lambda), environment variables được set bởi platform. Nhưng khi dev local, bạn cần một cách tiện lợi để set chúng. `godotenv` load file `.env` vào environment variables — chỉ dùng trong development.

```go
// .env file (KHÔNG commit vào git — thêm vào .gitignore)
// DATABASE_URL=postgres://user:pass@localhost:5432/mydb
// APP_PORT=8080
// JWT_SECRET=dev-secret-không-dùng-trong-prod
// LOG_LEVEL=debug

// main.go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/joho/godotenv" // go get github.com/joho/godotenv
)

func main() {
    // Chỉ load .env trong development — production không có file này (và không cần)
    // godotenv.Load() không lỗi nếu file không tồn tại khi dùng overload này
    if err := godotenv.Load(); err != nil {
        // Trong production: .env không có là bình thường — env vars được set bởi platform
        // Trong development: nếu không có .env, dev có thể quên set up
        log.Println("no .env file found, using system environment variables")
    }

    dbURL := os.Getenv("DATABASE_URL")
    port := os.Getenv("APP_PORT")
    fmt.Printf("Connecting to %s on port %s\n", dbURL, port)
}
```

---

## 6. Viper — Full-Featured Configuration

`viper` là thư viện config phổ biến nhất trong Go ecosystem — tương đương với Microsoft.Extensions.Configuration trong C# nhưng với nhiều tính năng hơn. Viper hỗ trợ: environment variables, config files (JSON/YAML/TOML), remote config (etcd/Consul), command-line flags, và defaults.

```go
// go get github.com/spf13/viper

package main

import (
    "fmt"
    "log"
    "strings"

    "github.com/spf13/viper"
)

func setupViper() {
    // 1. Đặt default values — tương tự appsettings.json defaults
    viper.SetDefault("app.port", 8080)
    viper.SetDefault("app.env", "development")
    viper.SetDefault("log.level", "info")
    viper.SetDefault("db.max_connections", 25)
    viper.SetDefault("db.timeout_seconds", 30)

    // 2. Config file — đọc nếu có (optional trong 12-factor)
    viper.SetConfigName("config")      // tên file không có extension
    viper.SetConfigType("yaml")         // hoặc "json", "toml"
    viper.AddConfigPath("./config")     // thư mục tìm config
    viper.AddConfigPath(".")            // hoặc thư mục hiện tại

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            log.Fatalf("error reading config file: %v", err)
        }
        // Config file không có — OK, dùng env vars và defaults
    }

    // 3. Environment variables — ưu tiên cao hơn config file
    // Prefix APP_ để tránh conflict với system env vars
    viper.SetEnvPrefix("APP")           // APP_PORT=9090 sẽ override app.port
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // app.port → APP_PORT
    viper.AutomaticEnv()                // tự động map env vars

    // Ví dụ: APP_DB_MAX_CONNECTIONS=50 sẽ override db.max_connections
}

func main() {
    setupViper()

    fmt.Printf("Port: %d\n", viper.GetInt("app.port"))
    fmt.Printf("Env: %s\n", viper.GetString("app.env"))
    fmt.Printf("DB connections: %d\n", viper.GetInt("db.max_connections"))
}
```

---

## 7. Config Struct Pattern — Một Struct cho Tất cả Settings

Dùng raw `viper.GetString(...)` rải rác trong code tạo ra vấn đề tương tự dùng magic strings — khó refactor, không type-safe, không rõ config nào cần thiết. Pattern tốt hơn là một `Config` struct với tất cả settings, load một lần ở `main()`, rồi inject như dependency.

Trong C#, đây tương đương với Options Pattern (`IOptions<MySettings>`) — một strongly-typed class chứa toàn bộ settings của một section.

```go
// BAD: đọc config rải rác khắp nơi — không biết config nào được dùng, khó test
package main

import "github.com/spf13/viper"

type UserHandler struct{}

func (h *UserHandler) Handle() {
    port := viper.GetInt("app.port")           // magic key, dễ typo
    dbURL := viper.GetString("database.url")   // không ai biết field này tồn tại ở đâu
    _ = port
    _ = dbURL
}
```

```go
// GOOD: một Config struct tập trung — rõ ràng, type-safe, dễ validate và test
package main

import (
    "fmt"
    "log/slog"
    "os"
    "strings"
    "time"

    "github.com/spf13/viper"
)

// Config là single source of truth cho toàn bộ configuration
// Mọi setting đều ở đây — không có viper.GetXxx() rải rác trong code
type Config struct {
    App      AppConfig
    Database DatabaseConfig
    Auth     AuthConfig
    Log      LogConfig
}

type AppConfig struct {
    Port int
    Env  string // "development", "staging", "production"
    Name string
}

type DatabaseConfig struct {
    URL            string
    MaxConnections int
    Timeout        time.Duration
}

type AuthConfig struct {
    JWTSecret      string
    TokenExpiryHours int
}

type LogConfig struct {
    Level  string // "debug", "info", "warn", "error"
    Format string // "json", "text"
}

// LoadConfig đọc từ env vars và config file, trả về Config struct đã validated
func LoadConfig() (*Config, error) {
    v := viper.New()

    // Defaults
    v.SetDefault("app.port", 8080)
    v.SetDefault("app.env", "development")
    v.SetDefault("app.name", "my-service")
    v.SetDefault("database.max_connections", 25)
    v.SetDefault("database.timeout", "30s")
    v.SetDefault("auth.token_expiry_hours", 24)
    v.SetDefault("log.level", "info")
    v.SetDefault("log.format", "json")

    // Config file (optional)
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath("./config")
    v.AddConfigPath(".")
    _ = v.ReadInConfig() // ignore error nếu không có file

    // Environment variables — override mọi thứ
    v.SetEnvPrefix("APP")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
    v.AutomaticEnv()

    cfg := &Config{}

    // Unmarshal vào struct — type-safe, không magic strings nữa
    if err := v.Unmarshal(cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }

    // Validate — fail fast nếu thiếu required config
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return cfg, nil
}

// Validate kiểm tra required fields và logical constraints
// Fail fast tại startup tốt hơn là fail ở giữa chừng khi đang serve traffic
func (c *Config) Validate() error {
    if c.Database.URL == "" {
        return fmt.Errorf("DATABASE_URL is required")
    }
    if c.Auth.JWTSecret == "" {
        return fmt.Errorf("APP_AUTH_JWTSECRET is required")
    }
    if len(c.Auth.JWTSecret) < 32 {
        return fmt.Errorf("APP_AUTH_JWTSECRET must be at least 32 characters")
    }
    if c.App.Port <= 0 || c.App.Port > 65535 {
        return fmt.Errorf("APP_PORT must be between 1 and 65535")
    }
    validEnvs := map[string]bool{"development": true, "staging": true, "production": true}
    if !validEnvs[c.App.Env] {
        return fmt.Errorf("APP_ENV must be one of: development, staging, production")
    }
    return nil
}

// newLogger tạo logger dựa trên config — không dùng global
func newLogger(cfg LogConfig) *slog.Logger {
    var level slog.Level
    switch cfg.Level {
    case "debug":
        level = slog.LevelDebug
    case "warn":
        level = slog.LevelWarn
    case "error":
        level = slog.LevelError
    default:
        level = slog.LevelInfo
    }

    opts := &slog.HandlerOptions{Level: level, AddSource: true}

    var handler slog.Handler
    if cfg.Format == "text" {
        handler = slog.NewTextHandler(os.Stdout, opts)
    } else {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    }

    return slog.New(handler)
}

func main() {
    // Load config — fail fast nếu có lỗi
    cfg, err := LoadConfig()
    if err != nil {
        // Dùng log cơ bản vì logger chưa được khởi tạo
        fmt.Fprintf(os.Stderr, "FATAL: failed to load config: %v\n", err)
        os.Exit(1)
    }

    // Khởi tạo logger với config
    logger := newLogger(cfg.Log)
    logger.Info("config loaded successfully",
        slog.String("env", cfg.App.Env),
        slog.Int("port", cfg.App.Port),
    )

    // Inject cfg và logger vào mọi component — không dùng global
    // userRepo := NewUserRepository(cfg.Database, logger)
    // userSvc := NewUserService(userRepo, logger)
    // handler := NewUserHandler(userSvc, logger)

    logger.Info("server starting", slog.Int("port", cfg.App.Port))
}
```

---

## 8. Environment-Specific Configuration

Trong C#, bạn có `appsettings.Development.json` và `appsettings.Production.json`. Trong Go theo 12-factor, các giá trị khác nhau giữa environments đều đến từ environment variables, không phải nhiều file config. Config file chỉ chứa defaults và structure, không chứa environment-specific values.

```yaml
# config/config.yaml — chỉ chứa non-sensitive defaults và structure
# Không có URL, secret, hay bất kỳ giá trị production nào ở đây
app:
  name: "my-service"
  port: 8080

database:
  max_connections: 25
  timeout: "30s"

log:
  level: "info"
  format: "json"
```

```bash
# Development: set trong .env (không commit)
APP_APP_ENV=development
APP_DATABASE_URL=postgres://dev_user:dev_pass@localhost:5432/mydb_dev
APP_AUTH_JWTSECRET=dev-secret-ít-nhất-32-ký-tự-nhé-đây
APP_LOG_LEVEL=debug
APP_LOG_FORMAT=text

# Staging: set trong CI/CD hoặc Kubernetes Secret
APP_APP_ENV=staging
APP_DATABASE_URL=postgres://staging_user:${SECRET}@staging-db:5432/mydb
APP_AUTH_JWTSECRET=${K8S_SECRET_JWT}
APP_LOG_LEVEL=info

# Production: set trong Kubernetes Secret hoặc cloud secrets manager
APP_APP_ENV=production
APP_DATABASE_URL=${VAULT_SECRET_DB_URL}
APP_AUTH_JWTSECRET=${VAULT_SECRET_JWT}
APP_LOG_LEVEL=warn
```

---

## 9. Secrets Management

Secret (password, API key, JWT secret) **không bao giờ** được hardcode trong code hoặc commit vào git. Ngay cả trong development, dùng `.env` file với `.gitignore`, không phải hardcode trong source.

Trong production, các option thường gặp:
- **HashiCorp Vault**: enterprise-grade secret management, rotation tự động
- **AWS Secrets Manager / Parameter Store**: native nếu dùng AWS
- **GCP Secret Manager / Azure Key Vault**: tương tự cho GCP/Azure
- **Kubernetes Secrets**: đơn giản nhưng cần thêm encryption at rest

Go application đọc secrets qua environment variables (được set bởi platform từ secret manager) — không đọc trực tiếp từ Vault trong code (trừ khi có yêu cầu dynamic rotation).

```go
// BAD: hardcode secret — commit vào git = security incident
func connectDB() *sql.DB {
    db, _ := sql.Open("postgres", "postgres://admin:SuperSecret123!@prod-db:5432/mydb")
    return db
}

// BAD: secret trong config file — dễ bị commit nhầm
// config.yaml:
// database:
//   password: "SuperSecret123!"  # ĐỪNG LÀM VẬY!
```

```go
// GOOD: secret từ environment variable — platform inject, không bao giờ trong code
func (c *DatabaseConfig) ConnectionString() string {
    // URL đầy đủ từ env var — platform (K8s/ECS/Lambda) inject từ secret manager
    return c.URL // vd: "postgres://user:${secret}@host:5432/db" đã được resolve bởi platform
}

// GOOD: validate secret tồn tại và đủ strong tại startup
func (c *Config) Validate() error {
    if c.Auth.JWTSecret == "" {
        return fmt.Errorf("APP_AUTH_JWTSECRET must be set (use Vault or Kubernetes Secret)")
    }
    if c.Auth.JWTSecret == "changeme" || c.Auth.JWTSecret == "secret" {
        return fmt.Errorf("APP_AUTH_JWTSECRET appears to be a placeholder — use a real secret")
    }
    // ...
    return nil
}
```

---

## Common Issues

| Vấn đề | Nguyên nhân | Giải pháp |
|--------|------------|-----------|
| Duplicate log entries cùng lỗi | Log ở cả callee và caller | Chỉ log ở handler, các layer dưới chỉ wrap error |
| Log không có context (request_id, user_id) | Dùng logger gốc thay vì child logger | Dùng `logger.With(...)` tạo child logger per-request |
| `viper.GetString("key")` trả về empty | Key không match với env var name | Kiểm tra `SetEnvPrefix` và `SetEnvKeyReplacer` |
| Config validation xảy ra quá muộn | Validate trong handler thay vì startup | Gọi `cfg.Validate()` ngay trong `main()`, `os.Exit(1)` nếu fail |
| Secret bị log vô tình | Log toàn bộ config struct | Implement `String()` method trên Config để mask secrets |
| `slog.Debug` không xuất hiện | Level mặc định là Info | Set `slog.HandlerOptions{Level: slog.LevelDebug}` |
| `.env` bị commit vào git | Quên thêm vào `.gitignore` | Thêm `.env` vào `.gitignore` TRƯỚC KHI tạo file |
| zerolog chain không kết thúc bằng `.Msg()` | Quên gọi `.Msg("")` | zerolog yêu cầu `.Msg()` hoặc `.Send()` để emit log entry |

---

## Best Practices

Khởi tạo logger và config **một lần duy nhất** trong `main()` rồi truyền xuống qua dependency injection — tương tự cách ASP.NET Core DI container hoạt động. Không bao giờ gọi `slog.SetDefault()` trong library code vì nó ảnh hưởng đến global state của toàn bộ application.

Luôn thêm `duration` vào log của operations quan trọng (DB queries, external API calls, request handling). Đây là data quan trọng nhất để phát hiện performance regression — bạn có thể set alert "p99 DB query > 100ms" nếu log luôn có duration field.

Trong production, log level nên là `Info` hoặc `Warn`. `Debug` quá nhiều noise và chi phí lưu trữ log. `Info` đủ để trace flow. Chỉ tạm thời set `Debug` khi investigating production issue.

Implement `fmt.Stringer` interface trên Config struct để mask sensitive values khi print config ở startup:

```go
func (c AuthConfig) String() string {
    masked := "***"
    if c.JWTSecret != "" {
        masked = c.JWTSecret[:4] + "..." // chỉ show 4 ký tự đầu để xác nhận đúng secret
    }
    return fmt.Sprintf("AuthConfig{JWTSecret: %s, TokenExpiryHours: %d}", masked, c.TokenExpiryHours)
}
```

Validate config tại startup với **fail fast** strategy. Nếu `DATABASE_URL` không được set, tốt hơn là crash ngay với error message rõ ràng (`FATAL: DATABASE_URL is required`) thay vì start lên rồi crash khi có request đầu tiên với error khó trace hơn.

---

## Bài Tập 1: HTTP Server với Request Logging Middleware

Viết một HTTP server đơn giản với:
- Config struct load từ env vars
- slog JSONHandler logger
- Middleware tự động log mọi request với method, path, status code, duration, request_id

```go
package main

import (
    "encoding/json"
    "fmt"
    "log/slog"
    "math/rand"
    "net/http"
    "os"
    "strconv"
    "time"
)

// --- Config ---

type Config struct {
    Port     int
    LogLevel string
    AppName  string
}

func LoadConfig() Config {
    port := 8080
    if p := os.Getenv("PORT"); p != "" {
        if parsed, err := strconv.Atoi(p); err == nil {
            port = parsed
        }
    }

    logLevel := os.Getenv("LOG_LEVEL")
    if logLevel == "" {
        logLevel = "info"
    }

    appName := os.Getenv("APP_NAME")
    if appName == "" {
        appName = "demo-server"
    }

    return Config{Port: port, LogLevel: logLevel, AppName: appName}
}

// --- Logger ---

func newLogger(level string) *slog.Logger {
    var lvl slog.Level
    switch level {
    case "debug":
        lvl = slog.LevelDebug
    case "warn":
        lvl = slog.LevelWarn
    default:
        lvl = slog.LevelInfo
    }

    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
    return slog.New(handler)
}

// --- Middleware ---

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Tạo unique request ID
        requestID := fmt.Sprintf("%016x", rand.Int63())

        // Child logger với request context — tất cả log trong request này có request_id
        reqLogger := logger.With(
            slog.String("request_id", requestID),
            slog.String("method", r.Method),
            slog.String("path", r.URL.Path),
            slog.String("remote_addr", r.RemoteAddr),
        )

        // Inject request_id vào response header — client có thể dùng để trace
        w.Header().Set("X-Request-ID", requestID)

        // Wrap ResponseWriter để capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

        reqLogger.Info("request started")

        next.ServeHTTP(wrapped, r)

        reqLogger.Info("request completed",
            slog.Int("status", wrapped.statusCode),
            slog.Duration("duration", time.Since(start)),
        )
    })
}

// --- Handlers ---

type Server struct {
    logger *slog.Logger
    cfg    Config
}

func NewServer(cfg Config, logger *slog.Logger) *Server {
    return &Server{
        logger: logger.With(slog.String("component", "Server")),
        cfg:    cfg,
    }
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "ok",
        "app":    s.cfg.AppName,
    })
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
    // Simulate business logic — không log ở đây, middleware đã lo
    users := []map[string]any{
        {"id": 1, "name": "Alice"},
        {"id": 2, "name": "Bob"},
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(users)
}

func (s *Server) handleSlowEndpoint(w http.ResponseWriter, r *http.Request) {
    // Simulate slow operation — duration trong log sẽ reveal performance issue
    time.Sleep(150 * time.Millisecond)
    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "done")
}

// --- Main ---

func main() {
    cfg := LoadConfig()
    logger := newLogger(cfg.LogLevel)

    logger.Info("starting server",
        slog.String("app", cfg.AppName),
        slog.Int("port", cfg.Port),
        slog.String("log_level", cfg.LogLevel),
    )

    srv := NewServer(cfg, logger)

    mux := http.NewServeMux()
    mux.HandleFunc("/health", srv.handleHealth)
    mux.HandleFunc("/users", srv.handleUsers)
    mux.HandleFunc("/slow", srv.handleSlowEndpoint)

    // Wrap toàn bộ mux với logging middleware
    handler := loggingMiddleware(logger, mux)

    addr := fmt.Sprintf(":%d", cfg.Port)
    logger.Info("server listening", slog.String("addr", addr))

    if err := http.ListenAndServe(addr, handler); err != nil {
        logger.Error("server failed", slog.String("error", err.Error()))
        os.Exit(1)
    }
}

// Chạy:
//   go run main.go
//   PORT=9090 LOG_LEVEL=debug APP_NAME=my-api go run main.go
//
// Test:
//   curl http://localhost:8080/health
//   curl http://localhost:8080/users
//   curl http://localhost:8080/slow
//
// Sample output:
// {"time":"...","level":"INFO","msg":"request started","request_id":"a1b2c3d4...","method":"GET","path":"/users",...}
// {"time":"...","level":"INFO","msg":"request completed","request_id":"a1b2c3d4...","status":200,"duration":"0.123ms"}
```

---

## Bài Tập 2: Production-Ready App Setup với slog + Config Validation

Bài này tích hợp tất cả concepts: Config struct, Validate(), slog với JSONHandler, và proper dependency injection.

```go
package main

import (
    "errors"
    "fmt"
    "log/slog"
    "os"
    "strconv"
    "strings"
    "time"
)

// ===== CONFIG =====

type AppConfig struct {
    Name string
    Env  string
    Port int
}

type DBConfig struct {
    URL            string
    MaxConnections int
    ConnectTimeout time.Duration
}

type CacheConfig struct {
    Host     string
    Port     int
    Password string // sẽ được mask khi log
    TTL      time.Duration
}

type Config struct {
    App   AppConfig
    DB    DBConfig
    Cache CacheConfig
}

// String mask sensitive fields — an toàn khi log config ở startup
func (c Config) String() string {
    dbURL := c.DB.URL
    if len(dbURL) > 20 {
        dbURL = dbURL[:20] + "..." // truncate URL — có thể chứa password
    }

    cachePass := "(not set)"
    if c.Cache.Password != "" {
        cachePass = "***" // mask password
    }

    return fmt.Sprintf(
        "Config{App: %s/%s port=%d, DB: url=%s max_conn=%d, Cache: %s:%d pass=%s ttl=%s}",
        c.App.Name, c.App.Env, c.App.Port,
        dbURL, c.DB.MaxConnections,
        c.Cache.Host, c.Cache.Port, cachePass, c.Cache.TTL,
    )
}

func (c *Config) Validate() error {
    var errs []string

    // Required fields
    if c.DB.URL == "" {
        errs = append(errs, "DB_URL is required")
    }
    if c.Cache.Host == "" {
        errs = append(errs, "CACHE_HOST is required")
    }

    // Logical constraints
    validEnvs := map[string]bool{"development": true, "staging": true, "production": true}
    if !validEnvs[c.App.Env] {
        errs = append(errs, fmt.Sprintf("APP_ENV must be one of development|staging|production, got: %q", c.App.Env))
    }

    if c.App.Port < 1 || c.App.Port > 65535 {
        errs = append(errs, fmt.Sprintf("APP_PORT must be 1-65535, got: %d", c.App.Port))
    }

    if c.DB.MaxConnections < 1 || c.DB.MaxConnections > 1000 {
        errs = append(errs, fmt.Sprintf("DB_MAX_CONNECTIONS must be 1-1000, got: %d", c.DB.MaxConnections))
    }

    // Production-specific constraints
    if c.App.Env == "production" {
        if strings.Contains(c.DB.URL, "localhost") {
            errs = append(errs, "DB_URL should not point to localhost in production")
        }
    }

    if len(errs) > 0 {
        return errors.New("config validation failed:\n  - " + strings.Join(errs, "\n  - "))
    }
    return nil
}

// LoadConfig đọc từ environment variables với defaults
// Không dùng viper để bài này tự chứa (standalone), trong thực tế dùng viper
func LoadConfig() (*Config, error) {
    cfg := &Config{
        App: AppConfig{
            Name: getEnvOrDefault("APP_NAME", "my-service"),
            Env:  getEnvOrDefault("APP_ENV", "development"),
            Port: getEnvInt("APP_PORT", 8080),
        },
        DB: DBConfig{
            URL:            os.Getenv("DB_URL"),
            MaxConnections: getEnvInt("DB_MAX_CONNECTIONS", 25),
            ConnectTimeout: getEnvDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
        },
        Cache: CacheConfig{
            Host:     getEnvOrDefault("CACHE_HOST", "localhost"),
            Port:     getEnvInt("CACHE_PORT", 6379),
            Password: os.Getenv("CACHE_PASSWORD"),
            TTL:      getEnvDuration("CACHE_TTL", 5*time.Minute),
        },
    }

    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return cfg, nil
}

// ===== LOGGER =====

func newLogger(env string) *slog.Logger {
    opts := &slog.HandlerOptions{
        AddSource: true,
        Level:     slog.LevelInfo,
    }

    if env == "development" {
        opts.Level = slog.LevelDebug
        // TextHandler dễ đọc hơn cho local development
        return slog.New(slog.NewTextHandler(os.Stdout, opts))
    }

    // JSON cho staging và production
    return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// ===== HELPER FUNCTIONS =====

func getEnvOrDefault(key, defaultVal string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
    if v := os.Getenv(key); v != "" {
        if parsed, err := strconv.Atoi(v); err == nil {
            return parsed
        }
    }
    return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
    if v := os.Getenv(key); v != "" {
        if parsed, err := time.ParseDuration(v); err == nil {
            return parsed
        }
    }
    return defaultVal
}

// ===== APPLICATION =====

type Application struct {
    cfg    *Config
    logger *slog.Logger
}

func NewApplication(cfg *Config, logger *slog.Logger) *Application {
    return &Application{
        cfg:    cfg,
        logger: logger.With(slog.String("app", cfg.App.Name), slog.String("env", cfg.App.Env)),
    }
}

func (a *Application) Run() error {
    a.logger.Info("application starting",
        slog.String("config", a.cfg.String()), // String() tự động mask secrets
        slog.Int("port", a.cfg.App.Port),
    )

    // Simulate startup checks
    if err := a.connectDB(); err != nil {
        return fmt.Errorf("startup: %w", err)
    }
    if err := a.connectCache(); err != nil {
        return fmt.Errorf("startup: %w", err)
    }

    a.logger.Info("application ready",
        slog.Int("port", a.cfg.App.Port),
    )

    // Simulate running...
    time.Sleep(100 * time.Millisecond)

    a.logger.Info("application shutting down gracefully")
    return nil
}

func (a *Application) connectDB() error {
    start := time.Now()
    a.logger.Info("connecting to database",
        slog.String("url", maskURL(a.cfg.DB.URL)),
        slog.Int("max_connections", a.cfg.DB.MaxConnections),
    )

    // Simulate connection
    time.Sleep(10 * time.Millisecond)

    a.logger.Info("database connected",
        slog.Duration("duration", time.Since(start)),
    )
    return nil
}

func (a *Application) connectCache() error {
    start := time.Now()
    a.logger.Info("connecting to cache",
        slog.String("host", a.cfg.Cache.Host),
        slog.Int("port", a.cfg.Cache.Port),
    )

    time.Sleep(5 * time.Millisecond)

    a.logger.Info("cache connected",
        slog.Duration("duration", time.Since(start)),
        slog.Duration("ttl", a.cfg.Cache.TTL),
    )
    return nil
}

// maskURL che password trong connection string
// "postgres://user:secretpass@host:5432/db" → "postgres://user:***@host:5432/db"
func maskURL(url string) string {
    if idx := strings.Index(url, "@"); idx > 0 {
        prefix := url[:idx]
        suffix := url[idx:]
        if colonIdx := strings.LastIndex(prefix, ":"); colonIdx > 0 {
            return prefix[:colonIdx] + ":***" + suffix
        }
    }
    return url
}

// ===== MAIN =====

func main() {
    // Bước 1: Load và validate config — TRƯỚC KHI làm bất cứ điều gì khác
    cfg, err := LoadConfig()
    if err != nil {
        // Dùng fmt.Fprintf vì logger chưa sẵn sàng
        fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
        fmt.Fprintf(os.Stderr, "\nRequired environment variables:\n")
        fmt.Fprintf(os.Stderr, "  DB_URL          - PostgreSQL connection string\n")
        fmt.Fprintf(os.Stderr, "  CACHE_HOST      - Redis hostname\n")
        fmt.Fprintf(os.Stderr, "  APP_ENV         - Environment: development|staging|production\n")
        fmt.Fprintf(os.Stderr, "\nOptional:\n")
        fmt.Fprintf(os.Stderr, "  APP_NAME        - Application name (default: my-service)\n")
        fmt.Fprintf(os.Stderr, "  APP_PORT        - HTTP port (default: 8080)\n")
        fmt.Fprintf(os.Stderr, "  DB_MAX_CONNECTIONS - (default: 25)\n")
        os.Exit(1)
    }

    // Bước 2: Khởi tạo logger với config đã validated
    logger := newLogger(cfg.App.Env)

    // Bước 3: Tạo application với dependency injection
    app := NewApplication(cfg, logger)

    // Bước 4: Run
    if err := app.Run(); err != nil {
        logger.Error("application failed", slog.String("error", err.Error()))
        os.Exit(1)
    }
}

// Chạy thành công:
//   DB_URL=postgres://user:pass@localhost:5432/mydb CACHE_HOST=localhost go run main.go
//
// Chạy với lỗi validate (thiếu DB_URL):
//   CACHE_HOST=localhost go run main.go
//   → FATAL: config validation failed:
//       - DB_URL is required
//
// Xem output JSON (production mode):
//   DB_URL=postgres://user:pass@localhost:5432/mydb CACHE_HOST=localhost APP_ENV=production go run main.go
```

---

## Tổng kết

| Concept | C# Equivalent | Go Approach |
|---------|--------------|-------------|
| Structured logging | Serilog/NLog | `log/slog` (stdlib, Go 1.21+) |
| Logger interface | `ILogger<T>` | `*slog.Logger` (inject as dependency) |
| Log context | `ForContext(...)` | `logger.With(...)` |
| Log sink/target | Serilog Sink | `slog.Handler` |
| App settings | `IConfiguration` + appsettings.json | `viper` + env vars |
| Options pattern | `IOptions<T>` | Config struct (load once, inject) |
| Secret management | Azure Key Vault / User Secrets | Env vars + Vault/cloud secrets manager |
| Environment-specific | appsettings.{Env}.json | Env vars per deployment, không phải file |

Go ưu tiên **simplicity và explicitness**: một Config struct rõ ràng, một logger được inject, config từ env vars. Không có magic DI container, không có phép thuật reflection lúc runtime. Bạn biết chính xác config đến từ đâu và logger được tạo như thế nào vì bạn tự viết từng dòng trong `main()`.

---

## Next

[17. Go vs C# — Gotchas →](17-go-vs-csharp-gotchas.md) — Những cạm bẫy phổ biến khi chuyển từ C# sang Go: nil interface, value vs pointer receiver, closure trong loop, và hơn thế nữa.
