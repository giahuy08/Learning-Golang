# 11. Common Go Patterns

> Dành cho developer C# đang học Go. Những pattern này xuất hiện trong hầu hết mọi codebase Go thực tế. Hiểu chúng sẽ giúp bạn đọc và viết Go idiomatic thay vì "C# bằng Go syntax".

---

## Tại sao Go có những pattern riêng?

Go không có class, không có constructor overloading, không có optional parameters, không có generics trước version 1.18, và không có inheritance. Những ràng buộc này không phải bug — chúng là design decision có chủ đích. Kết quả là cộng đồng Go đã phát triển một tập hợp các pattern đặc trưng để giải quyết những vấn đề mà C# giải quyết bằng ngôn ngữ feature trực tiếp. Hiểu *tại sao* những pattern này tồn tại quan trọng hơn nhiều so với chỉ biết *cách* viết chúng.

---

## 1. Functional Options Pattern

### Vấn đề

Trong C#, khi bạn có một constructor với nhiều tham số tùy chọn, bạn dùng named parameters hoặc Builder pattern:

```csharp
// C#: Named parameters hoặc Builder
var client = new HttpClient(
    baseUrl: "https://api.example.com",
    timeout: TimeSpan.FromSeconds(30),
    retryCount: 3,
    enableCompression: true
);

// Hoặc Builder
var client = new HttpClientBuilder()
    .WithBaseUrl("https://api.example.com")
    .WithTimeout(TimeSpan.FromSeconds(30))
    .WithRetryCount(3)
    .Build();
```

Go không có named parameters hay constructor overloading. Nếu bạn làm theo bản năng C#:

```go
// BAD: Cách naive của developer C# mới học Go
// Vấn đề 1: Nếu thêm field mới vào Config, tất cả caller code bị break
// Vấn đề 2: Caller phải biết giá trị mặc định là gì để không override
// Vấn đề 3: Với thư viện public, đây là API nightmare
func NewHTTPClient(baseURL string, timeout time.Duration, retryCount int, enableCompression bool) *HTTPClient {
    return &HTTPClient{
        baseURL:          baseURL,
        timeout:          timeout,
        retryCount:       retryCount,
        enableCompression: enableCompression,
    }
}

// Caller phải truyền đủ mọi thứ, kể cả những thứ họ không quan tâm
client := NewHTTPClient("https://api.example.com", 30*time.Second, 0, false)
```

### Giải pháp: Functional Options

Functional Options pattern (do Dave Cheney popularize) giải quyết vấn đề này một cách thanh lịch. Ý tưởng là thay vì nhận một list cố định các tham số, constructor nhận một danh sách các *hàm* — mỗi hàm biết cách modify một phần của config.

```go
// GOOD: Functional Options Pattern — pattern quan trọng nhất trong Go
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HTTPClient là struct thực sự (unexported fields — caller không thể tự set)
type HTTPClient struct {
	baseURL          string
	timeout          time.Duration
	retryCount       int
	enableCompression bool
	headers          map[string]string
	tlsConfig        *tls.Config
	logger           *log.Logger
}

// Option là một function type — đây là trái tim của pattern này
// Mỗi option là một hàm nhận pointer tới config và modify nó
type Option func(*HTTPClient)

// Các constructor function cho từng option — naming convention: WithXxx
// Lý do dùng closure: WithTimeout(30*time.Second) trả về một Option function
// chứa giá trị 30 seconds, function đó sau đó được apply lên client

func WithBaseURL(url string) Option {
	return func(c *HTTPClient) {
		c.baseURL = url
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *HTTPClient) {
		c.timeout = d
	}
}

func WithRetryCount(n int) Option {
	return func(c *HTTPClient) {
		if n < 0 {
			n = 0 // validation có thể xảy ra bên trong option
		}
		c.retryCount = n
	}
}

func WithCompression() Option {
	return func(c *HTTPClient) {
		c.enableCompression = true
	}
}

func WithHeader(key, value string) Option {
	return func(c *HTTPClient) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	}
}

func WithTLS(config *tls.Config) Option {
	return func(c *HTTPClient) {
		c.tlsConfig = config
	}
}

func WithLogger(logger *log.Logger) Option {
	return func(c *HTTPClient) {
		c.logger = logger
	}
}

// NewHTTPClient: constructor nhận variadic options
// Giá trị mặc định được set trước, sau đó options được apply theo thứ tự
func NewHTTPClient(opts ...Option) *HTTPClient {
	// Giá trị mặc định hợp lý
	client := &HTTPClient{
		baseURL:    "https://api.example.com",
		timeout:    10 * time.Second,
		retryCount: 3,
	}

	// Apply từng option lên client
	// Đây là lý do gọi là "functional" options — chúng ta apply hàm
	for _, opt := range opts {
		opt(client)
	}

	return client
}

func (c *HTTPClient) Do(method, path string) (*http.Response, error) {
	if c.logger != nil {
		c.logger.Printf("Request: %s %s%s", method, c.baseURL, path)
	}
	fmt.Printf("Calling %s %s%s (timeout: %v, retries: %d)\n",
		method, c.baseURL, path, c.timeout, c.retryCount)
	return nil, nil // simplified
}

func main() {
	// Caller 1: Chỉ cần override những gì họ quan tâm
	// Không cần biết giá trị mặc định của các field khác
	client1 := NewHTTPClient(
		WithBaseURL("https://payments.example.com"),
		WithTimeout(5*time.Second),
	)

	// Caller 2: Dùng tất cả mặc định
	client2 := NewHTTPClient()

	// Caller 3: Custom nhiều thứ
	logger := log.New(log.Writer(), "[HTTP] ", log.LstdFlags)
	client3 := NewHTTPClient(
		WithBaseURL("https://internal.example.com"),
		WithTimeout(30*time.Second),
		WithRetryCount(5),
		WithCompression(),
		WithHeader("Authorization", "Bearer token123"),
		WithLogger(logger),
	)

	client1.Do("GET", "/charge")
	client2.Do("GET", "/health")
	client3.Do("POST", "/users")
}
```

### Tại sao Functional Options tốt hơn Config struct cho thư viện?

Sự khác biệt then chốt là *extensibility*. Khi bạn publish một thư viện với Config struct, bạn đã expose toàn bộ cấu trúc nội bộ. Thêm một field mới vào Config struct là breaking change nếu caller dùng struct literal initialization. Với functional options, bạn chỉ thêm một `WithXxx` function mới — caller cũ không bị ảnh hưởng. Đây là lý do tại sao hầu hết các thư viện Go nổi tiếng như `grpc-go`, `go-redis`, `zap` đều dùng pattern này.

---

## 2. Builder Pattern trong Go

Builder pattern trong Go thường ít phổ biến hơn C# vì Functional Options đã giải quyết được hầu hết use cases. Tuy nhiên, Builder hợp lý hơn khi quá trình build có *nhiều bước phụ thuộc lẫn nhau* và bạn muốn đảm bảo thứ tự gọi đúng, hoặc khi bạn muốn build các object phức tạp từng bước với validation ở mỗi bước.

```go
// GOOD: Builder Pattern — dùng khi build process có bước phụ thuộc
package main

import (
	"errors"
	"fmt"
)

// QueryBuilder xây dựng SQL query phức tạp theo từng bước
// Functional options kém hơn ở đây vì thứ tự WHERE/JOIN/ORDER quan trọng
type QueryBuilder struct {
	table      string
	conditions []string
	joins      []string
	orderBy    string
	limit      int
	err        error // lưu error để chain — "error accumulation pattern"
}

func NewQueryBuilder(table string) *QueryBuilder {
	if table == "" {
		return &QueryBuilder{err: errors.New("table name is required")}
	}
	return &QueryBuilder{table: table, limit: -1}
}

// Mỗi method trả về *QueryBuilder để hỗ trợ method chaining
// Nếu đã có lỗi, skip và trả về ngay — "error accumulation"
func (qb *QueryBuilder) Where(condition string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if condition == "" {
		qb.err = errors.New("condition cannot be empty")
		return qb
	}
	qb.conditions = append(qb.conditions, condition)
	return qb
}

func (qb *QueryBuilder) Join(join string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.joins = append(qb.joins, join)
	return qb
}

func (qb *QueryBuilder) OrderBy(col string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.orderBy = col
	return qb
}

func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if n <= 0 {
		qb.err = fmt.Errorf("limit must be positive, got %d", n)
		return qb
	}
	qb.limit = n
	return qb
}

// Build là bước cuối — validate toàn bộ và trả về kết quả
// Trong C#: Build() thường throw InvalidOperationException nếu state không hợp lệ
// Trong Go: trả về (result, error) — caller phải xử lý error
func (qb *QueryBuilder) Build() (string, error) {
	if qb.err != nil {
		return "", fmt.Errorf("query builder error: %w", qb.err)
	}

	query := "SELECT * FROM " + qb.table
	for _, j := range qb.joins {
		query += " " + j
	}
	if len(qb.conditions) > 0 {
		query += " WHERE "
		for i, c := range qb.conditions {
			if i > 0 {
				query += " AND "
			}
			query += c
		}
	}
	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}
	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}
	return query, nil
}

func main() {
	// Giống LINQ fluent syntax trong C# — quen thuộc với developer C#
	query, err := NewQueryBuilder("users").
		Join("LEFT JOIN orders ON users.id = orders.user_id").
		Where("users.active = true").
		Where("orders.status = 'pending'").
		OrderBy("users.created_at DESC").
		Limit(50).
		Build()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(query)

	// BAD example — lỗi sẽ được collect và trả về ở Build()
	_, err = NewQueryBuilder("").
		Where("").
		Build()
	fmt.Println("Expected error:", err)
}
```

**Khi nào dùng Builder vs Functional Options?** Dùng Functional Options khi bạn viết thư viện public hoặc khi config là độc lập (không có bước phụ thuộc). Dùng Builder khi thứ tự bước quan trọng, khi có validation phức tạp giữa các bước, hoặc khi bạn build query/document/tree có cấu trúc phân cấp.

---

## 3. Repository Pattern

Repository pattern trong Go dựa hoàn toàn vào interfaces — đây là cách Go làm dependency injection thay vì DI frameworks như Autofac hay Microsoft.Extensions.DependencyInjection trong C#. Ý tưởng cốt lõi là: code business logic không bao giờ import package database trực tiếp; nó chỉ biết về interface. Điều này giúp test business logic mà không cần database thật.

```go
// GOOD: Repository Pattern — interface-based, enables testing without DB
package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Domain entity — pure struct, không có database annotation
// Khác C# Entity Framework: không có [Table], [Column] attributes
type User struct {
	ID        int64
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepository interface — đây là "contract"
// Nguyên tắc: interface được define ở nơi SỬ DỤNG, không phải nơi IMPLEMENT
// Đây là điểm khác biệt lớn với C# — trong C#, interface thường nằm cùng với implementation
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]*User, error)
}

// --- PostgreSQL Implementation ---

// postgresUserRepository implement UserRepository với real database
// Lowercase: không export trực tiếp — caller nhận interface, không phải struct
type postgresUserRepository struct {
	// db *sql.DB  // trong thực tế sẽ là real DB connection
	db string // simplified cho example
}

// NewPostgresUserRepository trả về interface, không phải concrete type
// "Accept interfaces, return structs" — nhưng đây là factory, trả về interface là hợp lý
// vì chúng ta muốn caller không biết về concrete type
func NewPostgresUserRepository(connStr string) UserRepository {
	return &postgresUserRepository{db: connStr}
}

func (r *postgresUserRepository) Create(ctx context.Context, user *User) error {
	// Thực tế: INSERT INTO users ... RETURNING id
	user.ID = 42 // simulate auto-generated ID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	fmt.Printf("[Postgres] INSERT user: %s\n", user.Email)
	return nil
}

func (r *postgresUserRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	fmt.Printf("[Postgres] SELECT * FROM users WHERE id = %d\n", id)
	if id == 0 {
		return nil, errors.New("user not found")
	}
	return &User{ID: id, Email: "found@example.com", Name: "Found User"}, nil
}

func (r *postgresUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	fmt.Printf("[Postgres] SELECT * FROM users WHERE email = '%s'\n", email)
	return nil, errors.New("user not found")
}

func (r *postgresUserRepository) Update(ctx context.Context, user *User) error {
	fmt.Printf("[Postgres] UPDATE users SET ... WHERE id = %d\n", user.ID)
	user.UpdatedAt = time.Now()
	return nil
}

func (r *postgresUserRepository) Delete(ctx context.Context, id int64) error {
	fmt.Printf("[Postgres] DELETE FROM users WHERE id = %d\n", id)
	return nil
}

func (r *postgresUserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	fmt.Printf("[Postgres] SELECT * FROM users LIMIT %d OFFSET %d\n", limit, offset)
	return []*User{{ID: 1, Email: "a@example.com"}, {ID: 2, Email: "b@example.com"}}, nil
}

// --- Mock Implementation (for testing) ---

// mockUserRepository là in-memory implementation cho unit tests
// Không cần Moq hay NSubstitute như trong C# — chỉ cần implement interface
type mockUserRepository struct {
	users  map[int64]*User
	nextID int64
	// Có thể thêm error injection để test error paths
	createErr  error
	findErr    error
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:  make(map[int64]*User),
		nextID: 1,
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *User) error {
	if m.createErr != nil {
		return m.createErr
	}
	user.ID = m.nextID
	m.nextID++
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	u, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return u, nil
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) Update(ctx context.Context, user *User) error {
	if _, ok := m.users[user.ID]; !ok {
		return fmt.Errorf("user %d not found", user.ID)
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id int64) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	var users []*User
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, nil
}

// --- Business Logic (Service layer) ---

// UserService chứa business logic, phụ thuộc vào interface (không phải concrete type)
// Đây là Dependency Injection trong Go — không cần DI container
type UserService struct {
	repo UserRepository // interface, không phải concrete type
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) RegisterUser(ctx context.Context, email, name string) (*User, error) {
	// Business rule: check duplicate email
	existing, _ := s.repo.FindByEmail(ctx, email)
	if existing != nil {
		return nil, fmt.Errorf("email %s already registered", email)
	}

	user := &User{Email: email, Name: name}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func main() {
	ctx := context.Background()

	// Production: dùng real database
	fmt.Println("=== Production (Postgres) ===")
	prodRepo := NewPostgresUserRepository("postgres://localhost/mydb")
	prodService := NewUserService(prodRepo)
	user, err := prodService.RegisterUser(ctx, "john@example.com", "John Doe")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Created user ID: %d\n", user.ID)
	}

	// Testing: dùng mock — không cần database thật
	fmt.Println("\n=== Testing (Mock) ===")
	mockRepo := newMockUserRepository()
	testService := NewUserService(mockRepo)
	user, err = testService.RegisterUser(ctx, "jane@example.com", "Jane Doe")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Created user ID: %d\n", user.ID)
	}

	// Test duplicate email
	_, err = testService.RegisterUser(ctx, "jane@example.com", "Jane Clone")
	fmt.Println("Duplicate error (expected):", err)
}
```

---

## 4. Options/Config Struct Pattern

Config struct pattern khác với Functional Options ở chỗ nó dùng cho *application-level configuration* — khi bạn đọc từ file JSON/YAML/environment variables, không phải khi bạn thiết kế API của thư viện. Trong C#, đây tương đương với `IOptions<T>` hoặc `appsettings.json`.

```go
// GOOD: Config struct cho app-level config, thường load từ env/file
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// AppConfig đại diện cho toàn bộ application configuration
// Tất cả fields đều exported vì cần deserialize từ JSON/YAML
type AppConfig struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
}

type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type CacheConfig struct {
	RedisAddr string
	TTL       time.Duration
}

// DefaultConfig trả về config với giá trị an toàn, hợp lý
// Đây là pattern Go: luôn cung cấp defaults, override bằng env vars
func DefaultConfig() AppConfig {
	return AppConfig{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			DSN:             "postgres://localhost/myapp",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Cache: CacheConfig{
			RedisAddr: "localhost:6379",
			TTL:       1 * time.Hour,
		},
	}
}

// LoadConfig: start với defaults, override bằng environment variables
// Giống IConfiguration trong ASP.NET Core nhưng explicit hơn
func LoadConfig() AppConfig {
	cfg := DefaultConfig()

	// Override từ environment variables (pattern phổ biến cho Docker/Kubernetes)
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		cfg.Database.DSN = dsn
	}
	if redis := os.Getenv("REDIS_ADDR"); redis != "" {
		cfg.Cache.RedisAddr = redis
	}

	return cfg
}

func main() {
	cfg := LoadConfig()
	fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Database: %s\n", cfg.Database.DSN)
	fmt.Printf("Cache TTL: %v\n", cfg.Cache.TTL)
}
```

---

## 5. Middleware / Decorator Pattern

Middleware pattern trong Go là cách wrap behavior quanh một function hoặc handler. Trong C#, bạn dùng ASP.NET Core middleware pipeline hoặc decorator pattern với interfaces. Trong Go, middleware chỉ là higher-order function — function nhận function và trả về function.

```go
// GOOD: Middleware pattern — function wrapping function
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// --- HTTP Middleware ---

// HandlerFunc là function type — giống delegate trong C#
// http.Handler interface chỉ có một method: ServeHTTP
// Middleware nhận http.Handler và trả về http.Handler

func LoggingMiddleware(next http.Handler) http.Handler {
	// http.HandlerFunc là adapter type — convert func sang http.Handler interface
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Gọi handler tiếp theo trong chain (giống next() trong Express.js)
		next.ServeHTTP(w, r)
		log.Printf("%s %s took %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return // không gọi next — request bị reject
		}
		// Thêm user info vào context để handler sau sử dụng
		ctx := context.WithValue(r.Context(), "user", "authenticated-user")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("Panic recovered: %v", rec)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Chain giúp compose nhiều middleware theo thứ tự dễ đọc
// Kết quả: RecoveryMiddleware(LoggingMiddleware(AuthMiddleware(handler)))
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	// Apply theo thứ tự ngược để outer middleware chạy trước
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// --- Generic Function Decorator (Go 1.18+) ---

// Decorator pattern cho business functions — không chỉ HTTP
// Trong C#: dùng AOP hoặc dynamic proxy (Castle Windsor, etc.)
// Trong Go: higher-order functions đơn giản hơn nhiều

type ServiceFunc[T any] func(ctx context.Context) (T, error)

// WithRetry wraps bất kỳ ServiceFunc nào với retry logic
// Generics (Go 1.18+) giúp function này work với mọi return type
func WithRetry[T any](fn ServiceFunc[T], maxRetries int) ServiceFunc[T] {
	return func(ctx context.Context) (T, error) {
		var (
			result T
			err    error
		)
		for attempt := 0; attempt <= maxRetries; attempt++ {
			result, err = fn(ctx)
			if err == nil {
				return result, nil
			}
			if attempt < maxRetries {
				log.Printf("Attempt %d failed: %v. Retrying...", attempt+1, err)
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			}
		}
		var zero T
		return zero, fmt.Errorf("after %d retries: %w", maxRetries, err)
	}
}

func main() {
	// HTTP Middleware composition
	mux := http.NewServeMux()
	myHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, authenticated user!")
	})

	// Đọc như: Request đi qua Recovery -> Logging -> Auth -> Handler
	mux.Handle("/api/users", Chain(myHandler,
		RecoveryMiddleware,
		LoggingMiddleware,
		AuthMiddleware,
	))

	// Function decorator example
	callCount := 0
	unreliableService := ServiceFunc[string](func(ctx context.Context) (string, error) {
		callCount++
		if callCount < 3 {
			return "", fmt.Errorf("service unavailable (attempt %d)", callCount)
		}
		return "success", nil
	})

	resilientService := WithRetry(unreliableService, 3)
	result, err := resilientService(context.Background())
	fmt.Printf("Result: %s, Error: %v (took %d attempts)\n", result, err, callCount)

	fmt.Println("HTTP server example set up (not starting server in this example)")
}
```

---

## 6. Accept Interfaces, Return Structs

Đây là một trong những nguyên tắc quan trọng nhất trong Go, được trích từ "Go Proverbs" của Rob Pike. Nguyên tắc này khác với C# nơi bạn thường thấy `IService` được trả về từ factory methods và DI container. Lý do Go khuyến khích trả về struct là để tránh "premature abstraction" — đừng tạo interface cho đến khi bạn thực sự cần polymorphism.

```go
// BAD: Trả về interface khi không cần thiết — "premature abstraction"
// Developer C# thường làm thế này vì thói quen từ IService/IRepository
package main

import "fmt"

// BAD example type (chỉ để minh họa)
type Writer interface {
	Write(data []byte) (int, error)
}

// BAD: Trả về interface — caller mất đi access vào các method của concrete type
// Nếu FileWriter có method Close(), caller không thể gọi nó qua Writer interface
type badFileWriter struct{ path string }
func (f *badFileWriter) Write(data []byte) (int, error) { return len(data), nil }
func (f *badFileWriter) Close() error { return nil } // caller không thấy method này!
func newBadFileWriter(path string) Writer { // BAD: trả về interface
	return &badFileWriter{path: path}
}

// GOOD: Trả về concrete struct — caller có thể dùng tất cả methods
// Khi cần interface, caller tự define hoặc dùng interface có sẵn
type FileWriter struct {
	path string
	// ... other fields
}

func (f *FileWriter) Write(data []byte) (int, error) {
	fmt.Printf("Writing %d bytes to %s\n", len(data), f.path)
	return len(data), nil
}

func (f *FileWriter) Close() error {
	fmt.Printf("Closing %s\n", f.path)
	return nil
}

func (f *FileWriter) Flush() error {
	fmt.Printf("Flushing %s\n", f.path)
	return nil
}

// GOOD: Trả về *FileWriter — caller có access đến Write, Close, Flush
func NewFileWriter(path string) *FileWriter {
	return &FileWriter{path: path}
}

// GOOD: Function NHẬN interface — rộng rãi, flexible
// io.Writer là interface có sẵn của Go standard library
// FileWriter tự động thỏa mãn io.Writer vì có method Write([]byte)(int,error)
import "io"

func ProcessData(w io.Writer, data string) error {
	_, err := fmt.Fprintf(w, "Processed: %s", data)
	return err
}

func main() {
	fw := NewFileWriter("/tmp/output.txt")
	
	// Caller có thể dùng TẤT CẢ methods của FileWriter
	fw.Write([]byte("hello"))
	fw.Flush() // available vì fw là *FileWriter, không phải interface
	fw.Close()

	// Khi cần truyền vào function accept interface — Go tự động convert
	ProcessData(fw, "some data") // *FileWriter thỏa io.Writer interface
}
```

Thực tế, "accept interfaces, return structs" có nghĩa là: viết functions với parameter là interface (để linh hoạt và testable), nhưng các constructor/factory của bạn trả về struct cụ thể. Caller quyết định dùng struct đó như interface nào.

---

## 7. Singleton với sync.Once

Trong C#, Singleton thường được implement qua DI container (`AddSingleton<T>()`) hoặc `Lazy<T>`. Trong Go, nếu bạn cần lazy initialization thread-safe, `sync.Once` là công cụ chuẩn. Tuy nhiên, Go khuyến khích *tránh global state* — dùng dependency injection (qua constructor) thay vì singleton khi có thể.

```go
// GOOD: Singleton với sync.Once — thread-safe, lazy initialization
package main

import (
	"fmt"
	"sync"
)

// AppLogger là example về resource tốn kém để khởi tạo
type AppLogger struct {
	prefix string
	// Thực tế: file handle, database connection pool, etc.
}

func (l *AppLogger) Log(msg string) {
	fmt.Printf("[%s] %s\n", l.prefix, msg)
}

// package-level variables — unexported, chỉ access qua GetLogger()
var (
	loggerInstance *AppLogger
	loggerOnce    sync.Once
)

// GetLogger trả về singleton instance
// sync.Once đảm bảo initialization chỉ chạy đúng 1 lần dù bao nhiêu goroutine gọi đồng thời
// Khác Lazy<T> trong C#: Once.Do() không trả về value — bạn phải dùng package var
func GetLogger() *AppLogger {
	loggerOnce.Do(func() {
		// Code này chạy đúng 1 lần — kể cả khi 1000 goroutine gọi GetLogger() cùng lúc
		fmt.Println("Initializing logger (chỉ thấy dòng này 1 lần)...")
		loggerInstance = &AppLogger{prefix: "APP"}
	})
	return loggerInstance
}

// Cách làm tốt hơn: dependency injection thay vì singleton
// Nhưng khi cần singleton (config, logger, metrics), sync.Once là cách Go idiomatic

func main() {
	// Gọi từ nhiều goroutines — vẫn chỉ init 1 lần
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger := GetLogger()
			logger.Log(fmt.Sprintf("Message from goroutine %d", id))
		}(i)
	}
	wg.Wait()

	// Verify singleton
	l1 := GetLogger()
	l2 := GetLogger()
	fmt.Printf("Same instance: %v\n", l1 == l2) // true
}
```

---

## 8. Error Sentinel Pattern

Trong C#, bạn ném exceptions cụ thể (`NotFoundException`, `ValidationException`) và caller dùng `catch (NotFoundException ex)`. Trong Go, errors là values, và bạn cần cách để caller kiểm tra *loại* error mà không phải so sánh string. Sentinel errors là exported error variables để caller dùng `errors.Is()`.

```go
// GOOD: Sentinel errors — exported error variables cho caller check
package main

import (
	"errors"
	"fmt"
)

// Sentinel errors — exported, package-level variables
// Convention: ErrXxx — khác C# ErrXxx không phổ biến, nhưng trong Go là chuẩn
// Ví dụ trong stdlib: io.EOF, sql.ErrNoRows, http.ErrNoCookie
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict: resource already exists")
	ErrInvalidInput = errors.New("invalid input")
)

// Wrapped error — giữ context nhưng vẫn có thể check bằng errors.Is()
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field %q: %s", e.Field, e.Message)
}

// Unwrap cho phép errors.As() tìm thấy ValidationError trong chuỗi wrapped errors
// Đây giống InnerException trong C#
func (e *ValidationError) Unwrap() error {
	return ErrInvalidInput
}

// UserStore simulate database operations
type UserStore struct {
	data map[int64]string
}

func NewUserStore() *UserStore {
	return &UserStore{data: map[int64]string{1: "Alice", 2: "Bob"}}
}

func (s *UserStore) GetUser(id int64) (string, error) {
	if id <= 0 {
		// Wrap context + sentinel: caller có thể check cả hai
		return "", fmt.Errorf("getUser id=%d: %w", id, &ValidationError{
			Field:   "id",
			Message: "must be positive",
		})
	}
	name, ok := s.data[id]
	if !ok {
		// Wrap context around sentinel error
		return "", fmt.Errorf("getUser id=%d: %w", id, ErrNotFound)
	}
	return name, nil
}

func (s *UserStore) CreateUser(id int64, name string) error {
	if _, exists := s.data[id]; exists {
		return fmt.Errorf("createUser id=%d: %w", id, ErrConflict)
	}
	s.data[id] = name
	return nil
}

func main() {
	store := NewUserStore()

	// errors.Is() kiểm tra toàn bộ error chain — như is operator trong C# catch
	_, err := store.GetUser(999)
	if errors.Is(err, ErrNotFound) {
		fmt.Println("Handle 404:", err)
	}

	// errors.As() extract concrete error type — như catch (SpecificException ex) trong C#
	_, err = store.GetUser(-1)
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		fmt.Printf("Validation failed: field=%s, msg=%s\n", valErr.Field, valErr.Message)
	}
	// errors.Is() vẫn work vì ValidationError.Unwrap() trả về ErrInvalidInput
	fmt.Println("Is ErrInvalidInput:", errors.Is(err, ErrInvalidInput))

	// Conflict error
	err = store.CreateUser(1, "Alice Clone")
	if errors.Is(err, ErrConflict) {
		fmt.Println("Handle conflict:", err)
	}
}
```

---

## 9. Table-Driven Design

Khi bạn thấy một switch statement lớn với nhiều case xử lý tương tự nhau, Go developers thường thay thế bằng `map[string]func()`. Đây không phải pattern riêng của Go — C# cũng có thể làm — nhưng trong Go nó phổ biến hơn vì Go không có `switch` expression như C# 8+.

```go
// BAD: Switch-heavy code — khó extend, vi phạm Open/Closed Principle
package main

import (
	"fmt"
	"strings"
)

// BAD: Thêm command mới phải sửa switch — và hàm này ngày càng dài
func handleCommandBad(cmd string, args []string) error {
	switch cmd {
	case "create":
		fmt.Println("Creating...", args)
	case "delete":
		fmt.Println("Deleting...", args)
	case "list":
		fmt.Println("Listing...", args)
	case "update":
		fmt.Println("Updating...", args)
	// Thêm case mới ở đây — hàm ngày càng dài
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
	return nil
}

// GOOD: Table-driven design — map[string]func() thay cho switch
// Thêm command mới: chỉ thêm entry vào map, không sửa logic core

// CommandFunc định nghĩa signature của mỗi command handler
type CommandFunc func(args []string) error

// CommandRegistry chứa tất cả registered commands
type CommandRegistry struct {
	commands map[string]CommandFunc
	aliases  map[string]string // alias -> canonical name
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]CommandFunc),
		aliases:  make(map[string]string),
	}
}

func (r *CommandRegistry) Register(name string, fn CommandFunc, aliases ...string) {
	r.commands[name] = fn
	for _, alias := range aliases {
		r.aliases[alias] = name
	}
}

func (r *CommandRegistry) Execute(cmd string, args []string) error {
	// Resolve alias
	if canonical, ok := r.aliases[cmd]; ok {
		cmd = canonical
	}
	fn, ok := r.commands[strings.ToLower(cmd)]
	if !ok {
		// Suggest similar commands — có thể add fuzzy matching
		return fmt.Errorf("unknown command %q. Available: %s",
			cmd, r.listCommands())
	}
	return fn(args)
}

func (r *CommandRegistry) listCommands() string {
	var cmds []string
	for name := range r.commands {
		cmds = append(cmds, name)
	}
	return strings.Join(cmds, ", ")
}

// --- Concrete Commands ---

func createCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("create requires a name argument")
	}
	fmt.Println("Creating:", args[0])
	return nil
}

func deleteCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("delete requires an ID argument")
	}
	fmt.Println("Deleting:", args[0])
	return nil
}

func listCommand(args []string) error {
	fmt.Println("Listing all resources")
	return nil
}

func main() {
	registry := NewCommandRegistry()
	
	// Register commands — thêm command mới chỉ cần thêm ở đây
	registry.Register("create", createCommand, "new", "add")
	registry.Register("delete", deleteCommand, "remove", "rm")
	registry.Register("list", listCommand, "ls", "show")
	registry.Register("update", func(args []string) error {
		fmt.Println("Updating:", args)
		return nil
	}, "edit")

	// Execute commands
	registry.Execute("create", []string{"my-resource"})
	registry.Execute("ls", []string{})          // alias
	registry.Execute("rm", []string{"resource-id"}) // alias

	err := registry.Execute("unknown", []string{})
	fmt.Println("Error:", err)

	// Table-driven cũng hay dùng cho HTTP routing
	// hoặc message type dispatch trong event-driven systems
	handlers := map[string]func(payload any){
		"user.created":  func(p any) { fmt.Println("Handle user created:", p) },
		"user.deleted":  func(p any) { fmt.Println("Handle user deleted:", p) },
		"order.placed":  func(p any) { fmt.Println("Handle order placed:", p) },
	}

	eventType := "user.created"
	if handler, ok := handlers[eventType]; ok {
		handler(map[string]string{"id": "123"})
	}
}
```

---

## 10. Graceful Shutdown Pattern

Trong C#, graceful shutdown thường được handle bởi ASP.NET Core host (`IHostedService.StopAsync`, `IApplicationLifetime`). Trong Go, bạn phải implement nó explicitly — và điều này là tốt vì bạn kiểm soát hoàn toàn. Pattern chuẩn là: lắng nghe OS signals (SIGTERM, SIGINT), sau đó cho phép requests đang chạy hoàn thành trong một khoảng thời gian nhất định trước khi shutdown.

```go
// GOOD: Graceful shutdown — pattern quan trọng cho production services
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// setupServer tạo và configure HTTP server
func setupServer() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow request — sẽ được hoàn thành trước khi shutdown
		select {
		case <-time.After(3 * time.Second):
			fmt.Fprintln(w, "Slow response completed")
		case <-r.Context().Done():
			// Request bị cancel (client disconnect hoặc server shutdown)
			http.Error(w, "Request cancelled", http.StatusServiceUnavailable)
		}
	})

	return &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
}

func main() {
	server := setupServer()

	// Channel để nhận OS signals
	// Dùng buffered channel (size 1) để signal không bị drop nếu chưa kịp receive
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// syscall.SIGINT  = Ctrl+C
	// syscall.SIGTERM = Docker/Kubernetes stop signal

	// Chạy server trong goroutine riêng
	// Main goroutine sẽ block ở signal receive
	serverErr := make(chan error, 1)
	go func() {
		fmt.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// http.ErrServerClosed là expected error khi Shutdown() được gọi
			serverErr <- fmt.Errorf("server error: %w", err)
		}
		close(serverErr)
	}()

	// Block ở đây cho đến khi nhận signal hoặc server error
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %v. Starting graceful shutdown...\n", sig)
	case err := <-serverErr:
		if err != nil {
			fmt.Println("Server failed:", err)
			os.Exit(1)
		}
		return
	}

	// Graceful shutdown với timeout
	// Nếu requests đang chạy không complete trong ShutdownTimeout,
	// chúng sẽ bị cancel — tốt hơn là bị kill abruptly
	shutdownTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	fmt.Printf("Waiting up to %v for in-flight requests to complete...\n", shutdownTimeout)
	
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Shutdown error (forced after timeout): %v\n", err)
		os.Exit(1)
	}

	// Đợi server goroutine hoàn thành
	if err := <-serverErr; err != nil {
		fmt.Println("Server error during shutdown:", err)
	}

	fmt.Println("Server shutdown complete. Goodbye!")
}
```

### Graceful Shutdown với signal.NotifyContext (Go 1.16+)

Go 1.16 thêm `signal.NotifyContext` giúp code ngắn gọn hơn cho trường hợp đơn giản:

```go
// GOOD: signal.NotifyContext — cách ngắn gọn hơn cho Go 1.16+
package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func mainWithNotifyContext() {
	// ctx sẽ bị cancel khi nhận SIGINT hoặc SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop() // cleanup signal handler

	server := &http.Server{Addr: ":8080"}

	go func() {
		fmt.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil {
			fmt.Println("Server stopped:", err)
		}
	}()

	// Đợi signal
	<-ctx.Done()
	stop() // Reset signal handling để Ctrl+C lần 2 force exit

	fmt.Println("Shutting down gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Println("Forced shutdown:", err)
	}
}
```

---

## Common Issues Table

| Vấn đề | Nguyên nhân | Giải pháp |
|--------|-------------|-----------|
| Functional option không có effect | Quên apply option `opt(config)` trong loop | Kiểm tra vòng lặp `for _, opt := range opts { opt(c) }` |
| Interface nil panic | Trả về concrete nil thay vì typed nil | Trả về `(*ConcreteType)(nil)` rõ ràng hoặc kiểm tra interface nil đúng cách |
| Repository mock không implement interface | Thêm method mới vào interface nhưng quên update mock | Dùng `var _ UserRepository = (*mockUserRepository)(nil)` để compile-time check |
| sync.Once không reset được | Once đã Do() không thể undo | Dùng cho initialization thực sự, không dùng cho state có thể reset |
| Middleware thứ tự ngược | Apply middleware từ trái sang phải nhưng outer wrap inner | Vẽ diagram request flow; dùng `Chain()` helper theo thứ tự tự nhiên |
| Sentinel error bị mất khi wrap | Dùng `fmt.Errorf("%s")` thay vì `%w` | Luôn dùng `fmt.Errorf("context: %w", err)` để preserve error chain |
| Goroutine leak khi shutdown | Goroutines không nhận cancel signal | Dùng context cancellation; check `ctx.Done()` trong goroutines |
| Builder state không reset giữa builds | Reuse builder instance sau `Build()` | Build() nên là terminal — document rõ, hoặc return new builder sau build |

---

## Best Practices

**Về Functional Options:** Luôn cung cấp giá trị mặc định hợp lý trong constructor. Người dùng không nên phải biết giá trị "an toàn" là bao nhiêu — đó là trách nhiệm của thư viện. Validate input trong mỗi `WithXxx` function thay vì validate tất cả trong `New()`. Đặt tên option theo dạng `WithXxx` để nhất quán với standard library và hầu hết thư viện Go nổi tiếng.

**Về Repository Pattern:** Define interface tại nơi sử dụng (service layer), không phải tại nơi implement (infrastructure layer). Điều này là "dependency inversion" thực sự — dependency đi từ concrete (postgres) về phía abstract (interface), không phải ngược lại. Mỗi interface trong Go nên nhỏ — nếu interface của bạn có 20 methods, hãy cân nhắc chia nhỏ.

**Về Error Handling:** Wrap errors với context ở mỗi layer (`fmt.Errorf("service.GetUser: %w", err)`) để tạo ra error messages tự documenting. Sentinel errors là public API — một khi export, không xóa hay thay đổi giá trị của chúng.

**Về Graceful Shutdown:** Mọi production service cần implement graceful shutdown. Docker và Kubernetes gửi SIGTERM trước khi kill process — nếu không handle, requests đang chạy sẽ bị cắt đột ngột. Shutdown timeout nên nhỏ hơn Kubernetes terminationGracePeriodSeconds (mặc định 30 giây).

**Về Middleware:** Middleware nên stateless hoặc thread-safe — chúng phục vụ nhiều requests đồng thời. Dùng context để pass data giữa middleware và handler thay vì global state. Document thứ tự middleware của bạn vì thứ tự quan trọng (authentication trước authorization, logging bao ngoài cùng).

---

## Bài tập thực hành

### Bài tập 1: Cache Service với Functional Options

Xây dựng một in-memory cache service sử dụng Functional Options pattern để configure TTL, max size, và eviction policy. Cache phải thread-safe.

```go
// Bài tập 1: In-memory cache với functional options và thread safety
package main

import (
	"fmt"
	"sync"
	"time"
)

// cacheEntry lưu value cùng expiration time
type cacheEntry struct {
	value     any
	expiresAt time.Time
}

func (e cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// Cache là thread-safe in-memory cache
type Cache struct {
	mu        sync.RWMutex
	data      map[string]cacheEntry
	ttl       time.Duration
	maxSize   int
	onEvict   func(key string, value any)
	// cleanup goroutine interval
	cleanupInterval time.Duration
}

// CacheOption là functional option type cho Cache
type CacheOption func(*Cache)

func WithTTL(ttl time.Duration) CacheOption {
	return func(c *Cache) {
		if ttl > 0 {
			c.ttl = ttl
		}
	}
}

func WithMaxSize(size int) CacheOption {
	return func(c *Cache) {
		if size > 0 {
			c.maxSize = size
		}
	}
}

func WithEvictionCallback(fn func(key string, value any)) CacheOption {
	return func(c *Cache) {
		c.onEvict = fn
	}
}

func WithCleanupInterval(interval time.Duration) CacheOption {
	return func(c *Cache) {
		if interval > 0 {
			c.cleanupInterval = interval
		}
	}
}

// NewCache tạo cache mới với default values và apply options
func NewCache(opts ...CacheOption) *Cache {
	c := &Cache{
		data:            make(map[string]cacheEntry),
		ttl:             5 * time.Minute,  // default TTL
		maxSize:         1000,             // default max 1000 entries
		cleanupInterval: 1 * time.Minute,  // cleanup mỗi phút
	}
	for _, opt := range opts {
		opt(c)
	}
	// Chạy background cleanup goroutine
	go c.cleanupLoop()
	return c
}

// Set lưu key-value với TTL
// Nếu đã đầy, evict key đầu tiên tìm được (LRU thực sự cần doubly-linked list)
func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: nếu đầy và key mới, xóa một key hết hạn hoặc key đầu tiên
	if _, exists := c.data[key]; !exists && len(c.data) >= c.maxSize {
		c.evictOne()
	}

	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *Cache) evictOne() {
	// Ưu tiên evict expired entries
	for k, v := range c.data {
		if v.isExpired() {
			if c.onEvict != nil {
				c.onEvict(k, v.value)
			}
			delete(c.data, k)
			return
		}
	}
	// Không có expired entry — evict bất kỳ (trong thực tế dùng LRU list)
	for k, v := range c.data {
		if c.onEvict != nil {
			c.onEvict(k, v.value)
		}
		delete(c.data, k)
		return
	}
}

// Get lấy value theo key
// ok=false nếu key không tồn tại hoặc đã hết hạn
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok || entry.isExpired() {
		return nil, false
	}
	return entry.value, true
}

// Delete xóa key
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.data[key]; ok {
		if c.onEvict != nil {
			c.onEvict(key, entry.value)
		}
		delete(c.data, key)
	}
}

// Size trả về số entries hiện tại (bao gồm cả expired chưa cleanup)
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// cleanupLoop định kỳ xóa expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.data {
		if v.isExpired() {
			if c.onEvict != nil {
				c.onEvict(k, v.value)
			}
			delete(c.data, k)
		}
	}
}

func main() {
	evicted := []string{}

	cache := NewCache(
		WithTTL(2*time.Second),
		WithMaxSize(3),
		WithCleanupInterval(500*time.Millisecond),
		WithEvictionCallback(func(key string, value any) {
			evicted = append(evicted, key)
			fmt.Printf("Evicted: %s = %v\n", key, value)
		}),
	)

	// Set entries
	cache.Set("user:1", map[string]string{"name": "Alice"})
	cache.Set("user:2", map[string]string{"name": "Bob"})
	cache.Set("user:3", map[string]string{"name": "Charlie"})
	fmt.Println("Size after 3 sets:", cache.Size())

	// Get
	if val, ok := cache.Get("user:1"); ok {
		fmt.Println("Got user:1:", val)
	}

	// Set thứ 4 — sẽ evict một entry vì maxSize=3
	cache.Set("user:4", map[string]string{"name": "Dave"})
	fmt.Println("Size after 4th set:", cache.Size())

	// Đợi TTL expire
	fmt.Println("Waiting for TTL to expire...")
	time.Sleep(3 * time.Second)

	if _, ok := cache.Get("user:1"); !ok {
		fmt.Println("user:1 expired correctly")
	}

	fmt.Printf("Total evicted: %v\n", evicted)
}
```

### Bài tập 2: Event Bus với Repository Pattern và Middleware

Xây dựng một event bus đơn giản kết hợp Repository pattern (để lưu events), Middleware pattern (để add logging và validation), và Graceful Shutdown.

```go
// Bài tập 2: Event Bus với nhiều patterns kết hợp
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// === Domain ===

type Event struct {
	ID        string
	Type      string
	Payload   any
	CreatedAt time.Time
}

var ErrEventNotFound = errors.New("event not found")
var ErrInvalidEvent = errors.New("invalid event")

// === Repository ===

type EventRepository interface {
	Save(ctx context.Context, event *Event) error
	FindByID(ctx context.Context, id string) (*Event, error)
	FindByType(ctx context.Context, eventType string) ([]*Event, error)
}

// InMemoryEventRepository — implementation cho testing và development
type InMemoryEventRepository struct {
	mu     sync.RWMutex
	events map[string]*Event
}

func NewInMemoryEventRepository() EventRepository {
	return &InMemoryEventRepository{
		events: make(map[string]*Event),
	}
}

func (r *InMemoryEventRepository) Save(ctx context.Context, event *Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[event.ID] = event
	return nil
}

func (r *InMemoryEventRepository) FindByID(ctx context.Context, id string) (*Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.events[id]
	if !ok {
		return nil, fmt.Errorf("findByID %s: %w", id, ErrEventNotFound)
	}
	return e, nil
}

func (r *InMemoryEventRepository) FindByType(ctx context.Context, eventType string) ([]*Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Event
	for _, e := range r.events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result, nil
}

// === Event Bus với Middleware ===

type Handler func(ctx context.Context, event *Event) error
type Middleware func(Handler) Handler

type EventBus struct {
	repo        EventRepository
	handlers    map[string][]Handler
	middlewares []Middleware
	mu          sync.RWMutex
	logger      *log.Logger
}

// NewEventBus: accept interface (repo), return struct (*EventBus)
func NewEventBus(repo EventRepository, logger *log.Logger) *EventBus {
	return &EventBus{
		repo:     repo,
		handlers: make(map[string][]Handler),
		logger:   logger,
	}
}

// Use thêm middleware — apply cho tất cả handlers
func (b *EventBus) Use(m Middleware) {
	b.middlewares = append(b.middlewares, m)
}

// Subscribe đăng ký handler cho một event type
func (b *EventBus) Subscribe(eventType string, handler Handler) {
	// Apply middlewares (theo thứ tự ngược)
	h := handler
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		h = b.middlewares[i](h)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], h)
}

// Publish lưu event và dispatch tới handlers
func (b *EventBus) Publish(ctx context.Context, event *Event) error {
	if event.ID == "" || event.Type == "" {
		return fmt.Errorf("publish: %w", ErrInvalidEvent)
	}
	event.CreatedAt = time.Now()

	// Lưu event (Repository pattern)
	if err := b.repo.Save(ctx, event); err != nil {
		return fmt.Errorf("publish save: %w", err)
	}

	// Dispatch tới handlers
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			b.logger.Printf("Handler error for %s: %v", event.Type, err)
		}
	}
	return nil
}

// === Built-in Middlewares ===

func LoggingMiddleware(logger *log.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, event *Event) error {
			start := time.Now()
			err := next(ctx, event)
			logger.Printf("Event %s [%s] handled in %v (err=%v)",
				event.ID, event.Type, time.Since(start), err)
			return err
		}
	}
}

func RetryMiddleware(maxRetries int, delay time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, event *Event) error {
			var err error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				err = next(ctx, event)
				if err == nil {
					return nil
				}
				if attempt < maxRetries {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(delay):
					}
				}
			}
			return fmt.Errorf("after %d retries: %w", maxRetries, err)
		}
	}
}

func main() {
	logger := log.New(log.Writer(), "[EventBus] ", log.LstdFlags|log.Lmicroseconds)
	repo := NewInMemoryEventRepository()
	bus := NewEventBus(repo, logger)

	// Add global middlewares
	bus.Use(LoggingMiddleware(logger))
	bus.Use(RetryMiddleware(2, 100*time.Millisecond))

	// Subscribe handlers
	bus.Subscribe("user.created", func(ctx context.Context, event *Event) error {
		payload := event.Payload.(map[string]string)
		fmt.Printf("Sending welcome email to %s\n", payload["email"])
		return nil
	})

	bus.Subscribe("user.created", func(ctx context.Context, event *Event) error {
		fmt.Printf("Creating default preferences for user %s\n", event.ID)
		return nil
	})

	bus.Subscribe("order.placed", func(ctx context.Context, event *Event) error {
		fmt.Printf("Processing order: %v\n", event.Payload)
		return nil
	})

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Publish some events
	events := []*Event{
		{ID: "evt-001", Type: "user.created", Payload: map[string]string{"email": "alice@example.com"}},
		{ID: "evt-002", Type: "user.created", Payload: map[string]string{"email": "bob@example.com"}},
		{ID: "evt-003", Type: "order.placed", Payload: map[string]int{"order_id": 42, "amount": 100}},
		{ID: "evt-004", Type: "unknown.event", Payload: nil}, // no handlers registered
	}

	for _, evt := range events {
		select {
		case <-ctx.Done():
			fmt.Println("Shutdown requested, stopping event publishing")
			return
		default:
			if err := bus.Publish(ctx, evt); err != nil {
				logger.Printf("Failed to publish %s: %v", evt.ID, err)
			}
		}
	}

	// Query stored events
	userEvents, _ := repo.FindByType(ctx, "user.created")
	fmt.Printf("\nStored %d user.created events\n", len(userEvents))

	e, err := repo.FindByID(ctx, "evt-001")
	if err == nil {
		fmt.Printf("Found event: %s created at %v\n", e.ID, e.CreatedAt.Format(time.RFC3339))
	}

	// Simulate waiting for more work (in real app: HTTP server, message queue, etc.)
	fmt.Println("\nService running. Press Ctrl+C to stop...")
	<-ctx.Done()
	fmt.Println("Graceful shutdown complete.")
}
```

---

## Tổng kết

Những pattern trong bài này không phải là "rules" cứng nhắc — chúng là solutions đã được kiểm chứng cho các vấn đề thường gặp. Điều quan trọng hơn là hiểu vấn đề mà mỗi pattern giải quyết:

Functional Options giải quyết vấn đề API flexibility. Repository Pattern giải quyết vấn đề testability và decoupling. Middleware Pattern giải quyết vấn đề cross-cutting concerns. "Accept interfaces, return structs" giải quyết vấn đề premature abstraction. Graceful Shutdown giải quyết vấn đề reliability trong production.

Khi bạn gặp một vấn đề trong code Go, hãy tự hỏi: "Pattern nào phù hợp nhất với vấn đề này?" thay vì "Tôi sẽ làm điều này như thế nào trong C#?"

---

## Next

[12. JSON & Struct Tags →](12-json-struct-tags.md) — Marshal/unmarshal JSON, struct tags, xử lý optional fields và custom (un)marshaling.
