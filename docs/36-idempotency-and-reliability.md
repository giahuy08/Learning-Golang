# 36. Idempotency, Retry, Rate Limit, Singleflight, Cache

> Nhóm kỹ thuật "làm hệ thống không vỡ khi mạng chập chờn và tải tăng". Liên quan trực tiếp tới đặt hàng/thanh toán.

---

## 1. Idempotency — gửi 2 lần = xử lý 1 lần

Vấn đề: client timeout, retry `POST /orders` → tạo 2 đơn, trừ tiền 2 lần.

Giải pháp: client gửi header `Idempotency-Key: <uuid>`; server lưu kết quả theo key.

```sql
CREATE TABLE idempotency_keys (
  key          text PRIMARY KEY,
  request_hash text NOT NULL,           -- chống dùng lại key cho request khác
  status       text NOT NULL,           -- 'processing' | 'done'
  response     jsonb,
  created_at   timestamptz NOT NULL DEFAULT now()
);
```

```go
func (s *Service) CreateOrder(ctx context.Context, key, hash string, req OrderReq) (*Order, error) {
	// 1. Chiếm key (nguyên tử nhờ PRIMARY KEY)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO idempotency_keys(key, request_hash, status) VALUES ($1,$2,'processing')
		 ON CONFLICT (key) DO NOTHING`, key, hash)
	if err != nil { return nil, err }

	if n, _ := res.RowsAffected(); n == 0 { // key đã tồn tại
		var h, status string
		var resp []byte
		err := s.db.QueryRowContext(ctx,
			`SELECT request_hash, status, response FROM idempotency_keys WHERE key=$1`, key).
			Scan(&h, &status, &resp)
		if err != nil { return nil, err }
		if h != hash { return nil, ErrKeyReused }          // 422
		if status == "processing" { return nil, ErrInFlight } // 409, client thử lại sau
		var o Order
		return &o, json.Unmarshal(resp, &o)                  // trả lại kết quả cũ
	}

	// 2. Xử lý thật, rồi lưu kết quả
	o, err := s.create(ctx, req)
	if err != nil {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE key=$1`, key) // cho phép retry
		return nil, err
	}
	b, _ := json.Marshal(o)
	_, err = s.db.ExecContext(ctx,
		`UPDATE idempotency_keys SET status='done', response=$2 WHERE key=$1`, key, b)
	return o, err
}
```
Tốt hơn: ghi đơn hàng và cập nhật key **trong cùng một transaction** ([35](35-database-transactions-locking.md)) để không có khoảng hở. Dọn key cũ bằng job định kỳ (TTL 24–48h).

Cùng ý tưởng cho consumer message queue: dedupe theo `message_id` vì queue thường là *at-least-once*.

---

## 2. Retry đúng cách

Quy tắc: chỉ retry lỗi **tạm thời** (timeout, 503, 429, deadlock) và thao tác **idempotent**; có **trần số lần**, **exponential backoff + jitter**, và tôn trọng `ctx`.

```go
func Retry(ctx context.Context, attempts int, base time.Duration, fn func() error, retryable func(error) bool) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil || !retryable(err) {
			return err
		}
		backoff := base << i                                  // 100ms, 200ms, 400ms...
		backoff = backoff/2 + rand.N(backoff/2+1)             // jitter, tránh thundering herd (math/rand/v2)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("sau %d lần: %w", attempts, err)
}
```
Sai lầm hay gặp: retry ở nhiều tầng cùng lúc (client × gateway × service → nhân số lần), retry không giới hạn, retry lỗi 4xx.

---

## 3. Rate limiting & backpressure

### Per-IP với `golang.org/x/time/rate`
```go
type limiter struct {
	mu   sync.Mutex
	m    map[string]*rate.Limiter
	r    rate.Limit
	b    int
}

func (l *limiter) get(ip string) *rate.Limiter {
	l.mu.Lock(); defer l.mu.Unlock()
	if v, ok := l.m[ip]; ok { return v }
	v := rate.NewLimiter(l.r, l.b)
	l.m[ip] = v
	return v
}

func (l *limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if !l.get(ip).Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```
Lưu ý: map này **tăng vô hạn** — production cần xoá entry lâu không dùng (hoặc dùng LRU/Redis cho nhiều instance). Phía sau reverse proxy thì dùng `X-Forwarded-For` đã được tin cậy, không dùng `RemoteAddr`.

### Backpressure — giới hạn công việc đang chạy, từ chối khi đầy
```go
sem := make(chan struct{}, 100)
func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "overloaded", http.StatusServiceUnavailable) // fail fast thay vì xếp hàng vô hạn
		}
	})
}
```
Hàng đợi không giới hạn = che giấu quá tải đến khi hết RAM.

---

## 4. Cache và cache stampede với `singleflight`

Stampede: key nóng hết hạn, 1000 request cùng lúc đập xuống DB.

```go
import "golang.org/x/sync/singleflight"

var g singleflight.Group

func (s *Store) GetProduct(ctx context.Context, id string) (*Product, error) {
	if p, ok := s.cache.Get(id); ok {
		return p, nil
	}
	v, err, _ := g.Do(id, func() (any, error) { // chỉ 1 goroutine chạy, số còn lại chờ và dùng chung kết quả
		p, err := s.repo.Get(ctx, id)
		if err == nil {
			s.cache.Set(id, p, ttlWithJitter(5*time.Minute))
		}
		return p, err
	})
	if err != nil { return nil, err }
	return v.(*Product), nil
}

func ttlWithJitter(d time.Duration) time.Duration { return d + rand.N(d/10) } // tránh hết hạn đồng loạt
```
Cẩn thận: `ctx` của goroutine đầu tiên được dùng chung — nếu nó bị huỷ, mọi người nhận lỗi. Với dữ liệu quan trọng, dùng `context.WithoutCancel` + timeout riêng bên trong `Do`.

Các câu hỏi cache hay gặp: **cache-aside** (đọc: cache → DB → ghi cache; ghi: ghi DB rồi **xoá** cache), **stampede** (singleflight, jitter), **penetration** (key không tồn tại: cache cả "not found" ngắn hạn), **invalidation** (TTL là mạng lưới an toàn cuối cùng).

---

## Bài tập
1. Viết test gọi `CreateOrder` 20 lần song song cùng key → đúng 1 đơn trong DB.
2. Viết test `Retry` với `fn` lỗi 2 lần rồi thành công; kiểm tra ctx huỷ giữa chừng dừng ngay.
3. Load test endpoint có rate limit, quan sát tỉ lệ 429.
4. Đếm số lần gọi DB khi 1000 goroutine cùng `GetProduct` — trước/sau `singleflight`.

**Liên quan:** [15-concurrency-patterns.md](15-concurrency-patterns.md), [34-microservices-patterns.md](34-microservices-patterns.md), [35-database-transactions-locking.md](35-database-transactions-locking.md)
