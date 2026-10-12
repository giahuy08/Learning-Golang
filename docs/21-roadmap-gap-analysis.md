# 21. Roadmap Review — What's Missing to Reach "Expert"

Review của `golang-roadmap.md` (8 phases, basics → open source). Kết luận: **roadmap đã rất đầy đủ** cho việc trở thành Go developer sản xuất được — hiếm khi thấy self-study roadmap chi tiết đến vậy. Dưới đây là các khoảng trống thực sự đáng bổ sung để đạt mức "expert", không phải "production-ready".

---

## Gaps quan trọng (nên thêm vào Phase tương ứng)

| Chủ đề thiếu | Nên thêm vào Phase | Vì sao quan trọng |
|---|---|---|
| **`go:embed`** — nhúng file tĩnh vào binary | Phase 3 (Standard Library) | Dùng rất nhiều trong CLI/web tools thực tế (templates, static assets) để ship single binary thật sự không cần file rời |
| **Go Memory Model** (happens-before, `sync/atomic` semantics) | Phase 2 (Concurrency) | Hiểu tại sao race detector bắt được gì, và tại sao "works on my machine" concurrent code vẫn crash production |
| **Go scheduler internals** (GMP model, `GOMAXPROCS`, cách goroutine được multiplex lên OS thread) | Phase 2/8 | Ranh giới giữa "biết dùng goroutine" và "hiểu tại sao nó nhanh/khi nào nó chậm" — câu hỏi phỏng vấn senior kinh điển |
| **Go workspaces** (`go.work`) | Phase 4 hoặc 7 | Cần khi làm việc với multi-module repo (monorepo pattern phổ biến ở công ty) |
| **Module versioning nâng cao** (semantic import versioning, `v2+` module path, `go mod why`, `go mod graph`) | Phase 7 | Roadmap có nhắc go.mod cơ bản nhưng chưa có phần viết/publish thư viện versioned đúng chuẩn |
| **Benchmark statistics** (`benchstat`, so sánh benchmark trước/sau để tránh noise) | Phase 8 (Performance) | `go test -bench` một lần không đủ tin cậy — cần biết đo đúng cách trước khi optimize |
| **`context` nâng cao** — custom context keys, context leak, khi nào KHÔNG dùng context (chỉ mention trong Phase 3, chưa sâu) | Phase 3/4 | Lỗi phổ biến nhất của mid-level Go dev: lạm dụng context để truyền data thay vì cancellation/deadline |
| **Reflection (`reflect` package)** | Không có trong roadmap | Cần để hiểu cách `encoding/json`, ORM, validator libraries hoạt động bên dưới — dù ít khi tự viết code dùng reflection trực tiếp |
| **Struct field alignment / memory layout thực hành** | Có nhắc ở Phase 8 nhưng chưa có exercise | Nên có bài tập dùng `unsafe.Sizeof`/`fieldalignment` linter để thấy tác động thật |
| **WebAssembly (Go → WASM)** | Không có | Không bắt buộc, nhưng là chủ đề "biết rộng" của Go expert — Go compile thẳng ra `.wasm` |
| **Supply chain security** — `govulncheck` đã có trong CI (Phase 4 gRPC section) nhưng chưa được dạy như một kỹ năng riêng | Phase 6/7 | Nên là thói quen: chạy trước mỗi release, không chỉ trong CI |

---

## Gaps về "cách học" (không phải nội dung kỹ thuật)

1. **Không có bước git/version control cho chính learning repo.** Roadmap nói về git workflow ở Phase 7 (khi contribute open source) nhưng không nói bắt đầu commit code luyện tập từ Phase 1. → Đã fix: `C:\Learn\Go` giờ đã có git init (xem `20-setup-checklist.md`).
2. **Milestone projects không có bước "code review AI hoặc dùng golangci-lint trước khi coi là xong".** Nên thêm: mỗi milestone project phải pass `make ci` (fmt + vet + lint + test-race) trước khi chuyển phase tiếp theo — biến nó thành thói quen chứ không phải checklist cuối cùng.
3. **Không có mốc "đọc code người khác"** sớm hơn Phase 7. Nên chèn: từ Phase 3 trở đi, đọc 1 file trong Go standard library mỗi tuần (`strings`, `net/http`, `context`) — đây là cách nhanh nhất học idiomatic Go.

---

## Đính chính (sau khi review chi tiết 17 file docs còn lại)

Bảng gap phía trên ban đầu có ghi **fuzzing** và **`errgroup`** là thiếu/sơ sài — **sai**, đã kiểm tra lại và cả hai đã được cover đầy đủ:

- `06-testing.md` §14 có ví dụ native fuzzing hoàn chỉnh (`go test -fuzz`).
- `15-concurrency-patterns.md` §1 cover `errgroup.WithContext` kỹ, có so sánh BAD (raw WaitGroup+channel) vs GOOD (errgroup).

Đã xóa 2 dòng đó khỏi bảng gap. Gap thật sự tìm thấy khi review 17 file: `06-testing.md` thiếu `t.Parallel()` — đã bổ sung trực tiếp vào file đó (mục 5.1).

Ngoài ra: 2 link "Next" bị sai đã fix (`04-goroutines-channels.md` trỏ đúng sang `05-context.md`; `06-testing.md` trỏ đúng sang `07-defer-panic-recover.md`). File `19-performance-basics.md` trỏ tới `20-deployment-docker.md` (chưa viết) — nội dung deployment hiện nằm ở `39-docker-deployment-ci.md`.

---

## Đã viết bổ sung (docs 23–28)

Các gap sau trong bảng phía trên đã được viết thành doc riêng, kèm ví dụ code BAD/GOOD và bài tập:

| Doc | Cover gap nào |
|---|---|
| `23-context-advanced.md` | Context nâng cao (custom key, leak, propagation qua goroutine) |
| `24-scheduler-and-memory-model.md` | GMP scheduler internals, GOMAXPROCS, Go Memory Model / happens-before |
| `25-reflection.md` | `reflect` package — cách JSON/ORM hoạt động bên dưới |
| `26-modules-and-workspaces.md` | `go:embed`, `go.work` workspaces, semantic import versioning (`v2+`), `go mod why/graph` |
| `27-performance-engineering.md` | `benchstat`, quy trình benchmark đáng tin cậy, struct alignment thực hành với `fieldalignment` |
| `28-security-and-supply-chain.md` | `govulncheck`/`gosec` như thói quen, path traversal, secrets management, supply chain (`go.sum` verify) |

### Đợt bổ sung 35–40 (gap về thực chiến backend và phỏng vấn)

| Doc | Nội dung |
|---|---|
| `35-database-transactions-locking.md` | Transaction helper, chống oversell (atomic/FOR UPDATE/optimistic), isolation, deadlock + retry, N+1, connection pool |
| `36-idempotency-and-reliability.md` | Idempotency key, retry backoff + jitter, rate limit, backpressure, singleflight, cache stampede |
| `37-concurrency-bug-lab.md` | 9 bug concurrency tự gây và tự bắt (race, leak, deadlock, copylocks…), goleak, nhật ký lỗi |
| `38-integration-testing-testcontainers.md` | Postgres thật trong test, test oversell/idempotency, golden file, fake thay mock |
| `39-docker-deployment-ci.md` | Dockerfile multi-stage, compose, graceful shutdown, GitHub Actions, release (thay cho bài deployment bị thiếu) |
| `40-interview-prep.md` | 50 câu hỏi Go, thuật toán bằng Go (LRU, heap), khung system design, câu chuyện hành vi |

Lưu ý: đây vẫn là **tài liệu**. Giá trị chỉ có khi bạn áp dụng vào mini-ecommerce (bài 22) và có code chứng minh. Chưa viết: WASM, ký binary (sigstore/cosign), Go 1.26 release notes (đọc trực tiếp go.dev/doc/go1.26).

Còn lại chưa viết (không cấp bách, để làm khi tới đúng phase): module versioning publish thực tế lên registry công khai, WebAssembly (Go → WASM) — chủ đề "biết rộng", không bắt buộc cho track backend/CLI.

---

## Kết luận

Roadmap không thiếu về mặt "đủ để làm việc được" — nó đã cover production concerns (graceful shutdown, structured logging, testing, CI, K8s operators) tốt hơn nhiều roadmap phổ biến trên mạng. Khoảng trống chủ yếu nằm ở **internals** (scheduler, memory model, reflection) và vài **stdlib feature ít dùng nhưng đặc trưng của Go** (`go:embed`, fuzzing). Thêm các mục trên vào đúng phase sẽ đưa bạn từ "rất giỏi Go ứng dụng" lên "hiểu Go từ trong ra ngoài" — ranh giới thường phân biệt senior và staff/expert-level Gopher.
