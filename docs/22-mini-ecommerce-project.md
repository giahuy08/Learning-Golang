# 22. Mini E-commerce Project — Lộ trình tự xây dựng, đối chiếu với EcommerceAPI

> Dành cho C# developer đang học Go theo lộ trình trong `docs/`. Mục tiêu file này: thay vì setup thẳng repo lớn [rasadov/EcommerceAPI](https://github.com/rasadov/EcommerceAPI) (microservices, gRPC, Kafka, Docker) ngay từ đầu, bạn tự xây một bản e-commerce API tối giản của riêng mình theo từng giai đoạn, bám sát các bài đã học. Chỉ khi Go cơ bản đã chắc, bạn mới quay lại đọc/so sánh với repo tham khảo đó.

---

## 1. Vì sao không setup EcommerceAPI ngay

EcommerceAPI là một hệ thống microservices thực thụ: nhiều service (auth, product, order, payment...), giao tiếp qua gRPC, message queue (Kafka), containerize bằng Docker, có thể còn Kubernetes/CI. Nếu setup nó khi mới học đến struct/interface (bài 02), phần lớn thời gian sẽ dành cho việc dựng infrastructure và debug config — không phải luyện tư duy Go.

Cách hiệu quả hơn: tự viết một e-commerce API nhỏ, tăng độ phức tạp dần theo đúng thứ tự bạn học, rồi mới đọc repo lớn kia như tài liệu tham khảo kiến trúc.

---

## 2. Lộ trình theo giai đoạn

### Giai đoạn 1 — Domain models thuần Go (sau bài [02-structs-interfaces](02-structs-interfaces.md))

Mục tiêu: luyện struct, method, interface, không cần HTTP hay DB.

- Định nghĩa struct `Product`, `Cart`, `CartItem` (in-memory).
- Viết method `(c *Cart) AddItem(p Product, qty int)`, `(c *Cart) Total() float64`.
- Định nghĩa interface `Discounter` với 1-2 implementation (`PercentDiscount`, `FixedDiscount`) — luyện implicit interface satisfaction.
- Chạy bằng `go run` với `main()` in kết quả ra console, chưa cần server.

### Giai đoạn 2 — Error handling đúng chuẩn (sau bài [03-error-handling](03-error-handling.md))

- Thêm validation: số lượng phải > 0, sản phẩm phải tồn tại trong catalog.
- Custom error type `ErrOutOfStock`, `ErrProductNotFound`; dùng `errors.Is`/`errors.As`.
- Áp dụng đúng lưu ý ở [02-structs-interfaces.md#413](02-structs-interfaces.md) — luôn `return nil` tường minh cho `error`, tránh bẫy typed-nil-interface.

### Giai đoạn 3 — JSON + CRUD in-memory (sau bài [12-json-struct-tags](12-json-struct-tags.md))

- Thêm `json` struct tags cho `Product`, `Order`.
- Viết hàm marshal/unmarshal, thử input JSON sai định dạng để luyện xử lý lỗi decode.

### Giai đoạn 4 — HTTP server thật (sau bài [13-http-server](13-http-server.md))

- Dựng REST API với `net/http` (ServeMux Go 1.22+) theo đúng layered structure ở [10-project-structure](10-project-structure.md): `handler → service → repository`.
- Endpoint tối thiểu: `GET /products`, `POST /cart/items`, `GET /cart`, `POST /orders`.
- Repository vẫn in-memory (map + mutex), chưa cần DB thật.

### Giai đoạn 5 — Database thật (sau bài [14-database](14-database.md))

- Thay repository in-memory bằng Postgres (`database/sql` + driver, hoặc `sqlc`).
- Thêm bảng `products`, `orders`, `order_items`.
- Áp dụng interface-ở-phía-consumer đúng như ví dụ `UserRepo` trong bài 10.

### Giai đoạn 6 — Concurrency (sau bài [04-goroutines-channels](04-goroutines-channels.md) + [15-concurrency-patterns](15-concurrency-patterns.md))

- Thêm một tác vụ bất đồng bộ: khi order được tạo, gửi "confirmation" qua channel tới một worker goroutine (giả lập gửi email) thay vì xử lý đồng bộ trong request.
- Giới hạn số worker xử lý song song bằng worker pool — tái dùng pattern đã luyện ở milestone project Phase 2 trong `golang-roadmap.md`.

### Giai đoạn 7 — Refactor theo chuẩn structure (sau bài [26-modules-and-workspaces](26-modules-and-workspaces.md))

- Tách lại project theo `cmd/`, `internal/domain`, `internal/handler`, `internal/service`, `internal/repository`, `internal/platform` — đúng layout ở bài 10, mục 4.
- Thêm `config.Load()` tập trung thay vì `os.Getenv` rải rác.
- Thêm graceful shutdown.

### Giai đoạn 8 — Đọc và đối chiếu với EcommerceAPI thật

Sau khi có bản của riêng bạn chạy được với các bước trên, mới quay lại:

1. Clone repo, đọc `README.md` để biết chạy bằng Docker Compose thế nào.
2. So sánh cách họ tách service (auth/product/order/payment) với layered structure bạn vừa làm — họ tách theo **service boundary** (microservices), bạn tách theo **layer** (monolith). Đây là khác biệt kiến trúc quan trọng nhất cần hiểu, không phải copy code.
3. Đọc cách họ dùng gRPC giữa các service — liên hệ với phần gRPC ở Phase 4 trong `golang-roadmap.md`.
4. Nếu muốn, thử refactor bản của bạn từ monolith sang tách 1 service riêng (vd. tách `order` thành service độc lập gọi qua HTTP hoặc gRPC) — đây là bài tập nâng cao, không bắt buộc. Xem [34-microservices-patterns.md](34-microservices-patterns.md) cho các pattern cần biết trước khi tách (ranh giới service, resilience, saga/outbox, docker-compose) và bài tập cụ thể ở mục 8 của doc đó.

---

## 3. Nguyên tắc xuyên suốt

- Không nhảy cóc: mỗi giai đoạn chỉ thêm đúng 1 khái niệm mới đang học, phần còn lại giữ nguyên.
- Ưu tiên chạy được (`go run`) sau mỗi giai đoạn, đừng để dồn nhiều thay đổi rồi mới test.
- Khi bí, tra `golang-roadmap.md` để biết concept đó thuộc phase nào và có milestone project tương ứng không.

---

**Liên quan:** [10-project-structure.md](10-project-structure.md), [golang-roadmap.md](golang-roadmap.md), [21-roadmap-gap-analysis.md](21-roadmap-gap-analysis.md)
</content>
</invoke>
