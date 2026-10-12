# 34. Microservices Patterns trong Go

> `golang-roadmap.md` Phase 4 đã nhắc gRPC/ConnectRPC như công nghệ giao tiếp, và `22-mini-ecommerce-project.md` (Giai đoạn 8) hẹn "sau này đối chiếu với EcommerceAPI (repo microservices thật)". Doc này lấp khoảng trống ở giữa: **các pattern kiến trúc** cần biết trước khi tách 1 monolith thành nhiều service — không phải chỉ công nghệ giao tiếp (gRPC) mà là *khi nào tách*, *tách theo ranh giới nào*, và *cách service độc lập xử lý những vấn đề mà monolith không gặp phải* (network fail, data không còn 1 transaction, ai gọi ai).

---

## 1. Đừng tách service trước khi có lý do rõ ràng

Sai lầm phổ biến nhất khi mới học microservices: tách sớm vì "nghe nói công ty lớn làm vậy". Chi phí thật của việc tách:

- Một request giờ có thể fail vì network, không chỉ vì logic sai — phải xử lý timeout, retry, partial failure (mục 4).
- Không còn 1 database transaction bao trọn cả nghiệp vụ — cần pattern riêng cho tính nhất quán dữ liệu (mục 5).
- Test, deploy, debug một luồng nghiệp vụ giờ chạm nhiều repo/process thay vì 1 codebase.

**Chỉ tách khi có ít nhất 1 lý do cụ thể:** team khác nhau cần deploy độc lập không chờ nhau, một phần hệ thống cần scale riêng (ví dụ service tính giá cần nhiều CPU hơn hẳn service catalog), hoặc ranh giới nghiệp vụ đã rất rõ và ổn định (ít khi đổi cùng lúc với phần còn lại). Nếu không có lý do nào trong 3 cái này — **giữ monolith**, monolith có layered structure tốt (`10-project-structure.md`) vẫn scale được rất xa trước khi cần tách.

---

## 2. Tách theo ranh giới nghiệp vụ, không tách theo layer kỹ thuật

```
SAI — tách theo tầng kỹ thuật:
  "auth service" chứa toàn bộ logic auth cho MỌI domain
  "notification service" gửi mọi loại thông báo cho MỌI domain
  → domain nào cũng phải gọi qua 2 service này, coupling cao, đổi 1 chỗ ảnh hưởng tất cả

ĐÚNG — tách theo domain (bounded context):
  "order service" — sở hữu toàn bộ dữ liệu và logic liên quan đến order (kể cả gửi email xác nhận order)
  "product service" — sở hữu catalog, giá, tồn kho
  → mỗi service tự chứa đủ logic cho 1 domain, gọi nhau ít, mỗi service có DB riêng
```

Đây chính là điểm khác biệt kiến trúc quan trọng nhất giữa bản `cmd/shop` layered-monolith bạn đang xây và EcommerceAPI: layered structure tách theo **tầng** (`handler → service → repository`) trong **1 domain**; microservices tách theo **domain** (`order`, `product`, `payment`), và **mỗi service đó lại có layered structure riêng bên trong**. Hai cách tách không loại trừ nhau — chúng ở hai chiều khác nhau.

**Nguyên tắc:** mỗi service sở hữu dữ liệu của riêng nó, không service nào đọc thẳng vào database của service khác. Muốn lấy dữ liệu từ domain khác → gọi qua API (mục 3), không query chéo DB.

---

## 3. Giao tiếp giữa các service

### 3.1. Đồng bộ — request/response

| Lựa chọn | Khi dùng |
|---|---|
| REST/JSON qua `net/http` | Đơn giản, debug dễ (`curl` được ngay), phù hợp khi caller là browser/mobile hoặc service ngoài công ty |
| gRPC | Nội bộ giữa các Go service, cần hiệu năng cao + typed contract (`.proto`) + streaming |
| ConnectRPC | Muốn contract kiểu gRPC nhưng vẫn serve REST/JSON cho browser mà không cần gRPC-Gateway riêng — xem Phase 4 trong `golang-roadmap.md` |

```go
// Service B gọi Service A qua HTTP — LUÔN truyền context để timeout/cancel lan truyền đúng
// (xem 05-context.md và 23-context-advanced.md)
func (c *ProductClient) GetProduct(ctx context.Context, id string) (*Product, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/products/%s", c.baseURL, id), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call product service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product service returned %d", resp.StatusCode)
	}

	var p Product
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &p, nil
}
```

### 3.2. Bất đồng bộ — message queue / event

Khi caller **không cần đợi kết quả ngay** (order tạo xong → gửi event "order.created", service khác tự xử lý phần của mình), dùng message broker thay vì gọi HTTP trực tiếp — giảm coupling: service gửi event không cần biết ai đang lắng nghe.

| Broker | Khi dùng |
|---|---|
| **NATS** (`nats.go`) | Nhẹ, dễ setup, phù hợp học và dự án vừa — pub/sub hoặc JetStream (có persistence) |
| **Kafka** (`segmentio/kafka-go` hoặc `confluent-kafka-go`) | Throughput cao, cần replay log, hệ sinh thái lớn (EcommerceAPI dùng Kafka) |
| **RabbitMQ** (`amqp091-go`) | Routing phức tạp (exchange/queue linh hoạt), phổ biến ở hệ thống doanh nghiệp lâu đời |

```go
// Producer — publish event, KHÔNG đợi consumer xử lý xong
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	order, err := s.repo.Save(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}

	event, _ := json.Marshal(OrderCreatedEvent{OrderID: order.ID, Total: order.Total})
	if err := s.publisher.Publish(ctx, "order.created", event); err != nil {
		// QUYẾT ĐỊNH KIẾN TRÚC: publish fail có nên làm CreateOrder fail không?
		// Thường KHÔNG — order đã lưu thành công là nghiệp vụ chính, publish fail
		// nên log + có cơ chế retry riêng (outbox pattern, xem mục 5), không rollback order.
		s.logger.Error("failed to publish order.created", "order_id", order.ID, "err", err)
	}
	return order, nil
}
```

---

## 4. Resilience — service khác sập không được kéo sập service của bạn

### 4.1. Timeout luôn phải có, không bao giờ gọi service khác vô thời hạn

```go
// BAD: không timeout — nếu Product service treo, request này treo mãi, chiếm connection pool
resp, err := http.Get(productServiceURL)

// GOOD: context với timeout rõ ràng, truyền xuống mọi lời gọi bên dưới
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, productServiceURL, nil)
resp, err := http.DefaultClient.Do(req)
```

### 4.2. Circuit breaker — ngừng gọi service đang sập thay vì tiếp tục thử và timeout hàng loạt

```go
import "github.com/sony/gobreaker"

var cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
	Name:        "product-service",
	MaxRequests: 3,                // số request thử khi ở trạng thái half-open
	Timeout:     10 * time.Second, // sau bao lâu chuyển từ open -> half-open để thử lại
	ReadyToTrip: func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures > 5 // mở breaker sau 5 lần fail liên tiếp
	},
})

func GetProduct(ctx context.Context, id string) (*Product, error) {
	result, err := cb.Execute(func() (interface{}, error) {
		return productClient.GetProduct(ctx, id)
	})
	if err != nil {
		return nil, err // breaker đang open -> fail NGAY, không tốn thời gian chờ timeout
	}
	return result.(*Product), nil
}
```

**Cách hiểu 3 trạng thái:** `closed` (bình thường, request đi qua) → quá nhiều lỗi liên tiếp → `open` (từ chối request ngay lập tức, không gọi thật, để service kia có thời gian hồi phục) → sau `Timeout` → `half-open` (thử vài request xem đã ổn chưa) → thành công thì về `closed`, fail thì quay lại `open`.

### 4.3. Retry với exponential backoff — chỉ retry lỗi tạm thời

```go
// BAD: retry ngay lập tức liên tục -> làm service đang quá tải càng quá tải hơn (retry storm)
for i := 0; i < 3; i++ {
	if resp, err := client.Do(req); err == nil {
		return resp, nil
	}
}

// GOOD: backoff tăng dần + jitter, và CHỈ retry lỗi có thể là tạm thời (timeout, 503)
// không retry lỗi 400 (request sai) vì retry cũng sẽ fail y hệt
func doWithRetry(ctx context.Context, fn func() (*http.Response, error)) (*http.Response, error) {
	backoff := 100 * time.Millisecond
	for attempt := 0; attempt < 4; attempt++ {
		resp, err := fn()
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff + time.Duration(rand.Int63n(int64(backoff)))): // jitter
		}
		backoff *= 2
	}
	return nil, errors.New("max retries exceeded")
}
```

---

## 5. Dữ liệu phân tán — không còn transaction bao trọn nhiều service

Trong monolith, "tạo order + trừ tồn kho" là 1 DB transaction — hoặc cả 2 thành công, hoặc cả 2 rollback. Khi `order` và `inventory` là 2 service với 2 database riêng, **không thể** có 1 transaction bao cả hai.

### 5.1. Saga pattern — chuỗi bước cục bộ + compensating action khi lỗi giữa chừng

```
1. Order service: tạo order (status = "pending")
2. Order service publish "order.created"
3. Inventory service nhận event, trừ tồn kho
     -> thành công: publish "inventory.reserved"  -> Order service set status = "confirmed"
     -> THẤT BẠI (hết hàng): publish "inventory.failed"
        -> Order service phải chạy COMPENSATING ACTION: set status = "cancelled",
           hoàn tiền nếu đã charge — chứ không "rollback" theo nghĩa DB transaction,
           vì bước 1 đã commit thật vào DB của order service rồi.
```

Điểm khác biệt tư duy quan trọng nhất: **không có rollback tự động** — bạn phải tự thiết kế bước "undo" (compensating action) cho từng bước, và hệ thống có một khoảng thời gian ngắn ở trạng thái **không nhất quán tạm thời** (order "pending" trong lúc chờ inventory) — đây gọi là **eventual consistency**, đánh đổi chấp nhận được để không cần 2-phase-commit phân tán (vốn phức tạp và hiếm dùng thực tế).

### 5.2. Outbox pattern — đảm bảo "lưu DB" và "publish event" không bao giờ lệch nhau

Vấn đề ở ví dụ mục 3.2: nếu server crash **giữa** lúc lưu order xong và trước khi publish event, event bị mất vĩnh viễn dù order đã tồn tại trong DB — dữ liệu lệch nhau vĩnh viễn.

```go
// GOOD: lưu order VÀ event vào CÙNG 1 DB transaction (cùng database, nên transaction bình thường
// vẫn dùng được ở bước này) — một worker riêng đọc bảng outbox và publish, đánh dấu đã gửi.
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	return s.repo.WithTx(ctx, func(tx *sql.Tx) (*Order, error) {
		order, err := s.repo.SaveTx(tx, req)
		if err != nil {
			return nil, err
		}
		event, _ := json.Marshal(OrderCreatedEvent{OrderID: order.ID})
		if err := s.repo.SaveOutboxEventTx(tx, "order.created", event); err != nil {
			return nil, err
		}
		return order, nil // commit: order + outbox event ghi nhận ATOMIC cùng lúc
	})
}

// Worker riêng, chạy định kỳ hoặc lắng nghe DB change: đọc outbox chưa publish -> publish -> đánh dấu
```

Đây là lý do các hệ thống production nghiêm túc không publish event trực tiếp trong request handler — outbox đảm bảo "đã lưu DB" luôn đi kèm "sẽ được publish", kể cả khi crash giữa chừng.

---

## 6. Quan sát hệ thống nhiều service — bắt buộc, không phải tùy chọn

Với 1 service, log là đủ để debug. Với nhiều service, một request đi qua 4-5 service — nếu không có cách nối các log/span lại, debug gần như bất khả thi.

- **Distributed tracing:** propagate trace ID qua mọi lời gọi (HTTP header, message metadata) — xem `32-observability-opentelemetry.md` để trace context lan truyền qua `context.Context` đúng cách.
- **Correlation ID trong log:** mọi log line ở mọi service cho cùng 1 request phải có chung `request_id`/`trace_id` — nếu không, tìm log liên quan đến 1 lỗi cụ thể giữa nhiều service gần như không thể.
- **Health check chuẩn hóa:** mỗi service expose `/healthz` (process còn sống) và `/readyz` (sẵn sàng nhận traffic, ví dụ đã kết nối được DB) — đã nhắc ở Phase 6 trong `golang-roadmap.md`, bắt buộc khi có nhiều service để orchestrator (K8s) biết khi nào restart/khi nào ngừng route traffic tới.

---

## 7. Local development — chạy nhiều service cùng lúc

```yaml
# docker-compose.yml — cách chuẩn để chạy nhiều Go service + Postgres + broker cục bộ
services:
  order-service:
    build: ./order-service
    ports: ["8081:8080"]
    environment:
      DATABASE_URL: postgres://order-db:5432/orders
      NATS_URL: nats://broker:4222
  product-service:
    build: ./product-service
    ports: ["8082:8080"]
  order-db:
    image: postgres:16
  broker:
    image: nats:2
```

`docker-compose up` khởi động toàn bộ hệ thống local trong 1 lệnh — đây là lý do EcommerceAPI (và hầu hết dự án microservices thật) luôn có `docker-compose.yml` ở root, đọc `README.md` của nó trước tiên khi bạn quay lại đối chiếu ở Giai đoạn 8 của `22-mini-ecommerce-project.md`.

---

## 8. Bài tập nâng cao — áp dụng vào `cmd/shop`

Chỉ làm bài này **sau khi** đã hoàn thành hết 7 giai đoạn trong `22-mini-ecommerce-project.md` (layered monolith chạy tốt với DB thật + concurrency). Đây là bài tập tùy chọn, không nằm trong lộ trình bắt buộc:

1. Tách `cmd/shop` thành 2 service riêng: `order-service` và `product-service`, mỗi service có database/schema riêng, không query chéo.
2. `order-service` gọi `product-service` qua HTTP (dùng pattern mục 3.1) để lấy giá sản phẩm khi tạo order — thêm timeout + 1 lần retry.
3. Thêm `docker-compose.yml` chạy cả 2 service + Postgres cùng lúc.
4. (Nâng cao) Thêm NATS, publish event `order.created` khi order tạo thành công, viết 1 consumer đơn giản log lại event nhận được — không cần outbox pattern đầy đủ ở bước này, chỉ cần thấy luồng event hoạt động.
5. Sau khi xong, đọc `README.md` của EcommerceAPI và so sánh: họ tách service theo ranh giới nào, dùng broker gì, có dùng outbox/saga không.

---

**Liên quan:** [golang-roadmap.md](golang-roadmap.md) (Phase 4, 6), [22-mini-ecommerce-project.md](22-mini-ecommerce-project.md), [23-context-advanced.md](23-context-advanced.md), [32-observability-opentelemetry.md](32-observability-opentelemetry.md), [28-security-and-supply-chain.md](28-security-and-supply-chain.md)
