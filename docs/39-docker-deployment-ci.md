# 39. Docker, docker-compose, CI/CD — đưa app Go ra chạy thật

> Đây là bài "deployment" mà các doc trước đã tham chiếu nhưng chưa có. Mục tiêu: `docker compose up` chạy cả app + Postgres, và mỗi push đều tự động kiểm tra.

---

## 1. Dockerfile multi-stage

```dockerfile
# ---- build ----
FROM golang:1.26 AS build          # khớp phiên bản trong go.mod
WORKDIR /src
COPY go.* ./                       # copy go.mod (+ go.sum nếu có) trước để cache layer download
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/shop

# ---- run ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
```
Giải thích các lựa chọn (hay bị hỏi):
- **Multi-stage**: image cuối chỉ có binary (~10–20MB) thay vì cả toolchain (~1GB).
- **`CGO_ENABLED=0`**: binary tĩnh, chạy được trên `scratch`/distroless. Nếu cần cgo (vd. sqlite) thì phải dùng base có libc.
- **`-trimpath -ldflags="-s -w"`**: bỏ đường dẫn build và symbol debug → nhỏ hơn, build tái lập được. Giữ symbol nếu cần debug production.
- **distroless `nonroot`**: không shell, không package manager, không chạy bằng root → bề mặt tấn công nhỏ. Debug bằng image `:debug` khi cần.
- **Copy `go.mod` trước source**: sửa code không làm mất cache bước download dependency.
- Thêm `.dockerignore` (`.git`, `docs`, `*.md`, `bin`) để context nhỏ và cache ổn định.
- Cần CA certs/timezone? distroless static đã có CA certs và tzdata.

## 2. docker-compose cho môi trường dev

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: shop
      POSTGRES_PASSWORD: shop
      POSTGRES_DB: shop
    ports: ["5432:5432"]
    volumes: [pgdata:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U shop -d shop"]
      interval: 3s
      timeout: 3s
      retries: 10

  app:
    build: .
    environment:
      DATABASE_URL: postgres://shop:shop@db:5432/shop?sslmode=disable
      PORT: "8080"
    ports: ["8080:8080"]
    depends_on:
      db:
        condition: service_healthy   # chờ DB sẵn sàng, không chỉ "container đã start"

volumes:
  pgdata:
```
Trong compose, host của DB là **tên service** (`db`), không phải `localhost`. Secret thật không commit — dùng file `.env` (đã gitignore) hoặc secret manager.

Yêu cầu từ phía app (đã học ở bài 16/13): đọc cấu hình từ env, `SIGTERM` → graceful shutdown, có `/healthz` và `/readyz`, log ra stdout dạng JSON.

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
go srv.ListenAndServe()
<-ctx.Done()
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = srv.Shutdown(shutdownCtx) // đặt timeout < terminationGracePeriodSeconds của K8s (mặc định 30s)
```

## 3. CI với GitHub Actions

`.github/workflows/ci.yml`
```yaml
name: ci
on:
  push: { branches: [main] }
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod     # một nguồn sự thật cho phiên bản Go
          cache: true
      - run: go vet ./...
      - run: go test -race -count=1 ./...
      - run: go test -race -tags=integration ./...   # runner Ubuntu có sẵn Docker cho testcontainers
      - uses: golangci/golangci-lint-action@v6
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
```
Lưu ý phiên bản action (`@v4`, `@v5`, `@v6`) thay đổi — kiểm tra bản mới trong tài liệu của từng action. Nên pin theo commit SHA cho dự án nghiêm túc (xem [28-security-and-supply-chain.md](28-security-and-supply-chain.md)).

Branch protection: bắt buộc CI xanh mới được merge vào `main`.

## 4. Release / CD tối thiểu

```bash
docker build -t ghcr.io/<user>/shop:$(git rev-parse --short HEAD) .
docker push  ghcr.io/<user>/shop:$(git rev-parse --short HEAD)
```
- Tag theo **commit SHA hoặc version semver**, không dùng `:latest` để deploy (không biết đang chạy bản nào, rollback khó).
- Migration DB chạy như bước riêng trước khi rollout, và phải **tương thích ngược** (expand → deploy → contract) để rolling update không vỡ.
- Rollback = deploy lại tag cũ; vì vậy migration phá huỷ (drop cột) phải tách làm release sau.

## Bài tập
1. Viết Dockerfile cho `cmd/shop`, so sánh kích thước với `FROM golang` chạy thẳng (`docker images`).
2. `docker compose up` rồi `docker compose stop app` → xem log graceful shutdown (có in "shutting down" không, request đang xử lý có hoàn tất không).
3. Tạo workflow CI, cố tình làm test fail để thấy PR bị chặn.
4. Thêm `govulncheck` và xem báo cáo.

**Liên quan:** [16-logging-config.md](16-logging-config.md), [28-security-and-supply-chain.md](28-security-and-supply-chain.md), [20-setup-checklist.md](20-setup-checklist.md), [34-microservices-patterns.md](34-microservices-patterns.md)
