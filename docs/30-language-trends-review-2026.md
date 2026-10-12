# 30. Review toàn diện — Version Drift & Trending Topics (2026)

> **Cập nhật:** toàn bộ mục hành động ưu tiên Cao/Cao nhất/Trung bình ở mục 4, và 2 xu hướng ngành (ConnectRPC, OpenTelemetry, AI/LLM engineering) ở mục 3, đã được viết bổ sung vào docs — xem bảng trạng thái ở cuối file.

> Review có mục đích khác với [21-roadmap-gap-analysis.md](21-roadmap-gap-analysis.md): bài 21 review **độ sâu nội dung** (internals, concurrency, performance). Bài này review **độ mới** — liệu roadmap có theo kịp Go version thực tế đang dùng (`go.mod` khai báo `go 1.26.1`) và các xu hướng ngành 2025–2026 hay không. Kết luận ngắn: **có một khoảng trống version thật sự** — phần lớn docs neo ở kiến thức Go 1.21/1.22 (~2023-2024), trong khi project đã ở Go 1.26.

---

## 1. Version drift — vấn đề nghiêm trọng nhất

`go.mod` gốc:
```
module learn-go
go 1.26.1
```

Nhưng khi grep toàn bộ `docs/`, hầu hết ví dụ và giải thích dừng ở **Go 1.21–1.22**:
- `slices`/`maps`/`cmp` packages (1.21) — có, dạy tốt (bài 09).
- `log/slog` (1.21) — có, dạy sâu (bài 16).
- Loop variable per-iteration + `http.ServeMux` pattern matching (1.22) — có (bài 13, 17).

**Hoàn toàn vắng mặt**: mọi thứ từ Go 1.23 trở đi. Đây không phải lỗi nhỏ — 3 phiên bản (1.23, 1.24, 1.25, và 1.26 đang dùng) mỗi phiên bản đều có ít nhất 1 thay đổi ảnh hưởng cách viết code idiomatic. Với người học "để làm best Go dev", đây là khoảng trống ưu tiên cao nhất, cao hơn cả các gap nội dung đã liệt kê ở bài 21.

---

## 2. Tính năng theo version — thiếu hoàn toàn trong docs

### Go 1.23 (giữa 2024)

| Tính năng | Vì sao quan trọng |
|---|---|
| **Range-over-func iterators** (`iter.Seq[V]`, `iter.Seq2[K,V]`) — `for x := range myIterFunc { ... }` | Đây là thay đổi ngôn ngữ lớn nhất kể từ generics (1.18). Cho phép viết custom iterator (lazy, pull-based) mà không cần channel/goroutine — đổi hẳn cách bạn nghĩ về "duyệt qua một collection tuỳ chỉnh". Nên dạy ngay sau bài 09-generics vì nó dùng chung tư duy generic. |
| `slices.Collect`, `slices.Sorted`, `maps.Collect`, `maps.Keys`/`Values` trả về `iter.Seq` thay vì slice | Thay đổi API `slices`/`maps` đã dạy ở bài 09 — cần update, không phải thêm mới hoàn toàn |
| Package `unique` (string/value interning) | Ít dùng nhưng là câu trả lời chuẩn Go cho vấn đề dedupe string tốn memory — liên quan trực tiếp đến bài 27-performance-engineering |
| `go env -changed`, cải thiện `go.mod` telemetry | Nhỏ, không cấp thiết |

### Go 1.24 (đầu 2025)

| Tính năng | Vì sao quan trọng |
|---|---|
| **`tool` directive trong `go.mod`** (`go get -tool`, khai báo tool dependency trực tiếp trong go.mod thay vì hack `tools.go` với `_ "import"`) | Đây là cách chính thức mới để quản lý dev tool (linter, codegen) theo version — **bài 26-modules-and-workspaces nên dạy cái này thay vì (hoặc cùng với) pattern `tools.go` cũ nếu đang dạy pattern cũ** |
| **Generic type aliases** (`type Set[T comparable] = map[T]struct{}`) hoàn thiện | Bài 09-generics cần thêm mục này — trước 1.24 chỉ có generic type definition, không có generic alias |
| **`os.Root`** — giới hạn filesystem access trong một thư mục gốc, chống path traversal | Liên quan trực tiếp bài 28-security-and-supply-chain — đây chính là công cụ chuẩn để fix lỗi path traversal mà bài đó đang dạy phòng tránh thủ công |
| **`testing/synctest`** (experimental, cần `GOEXPERIMENT=synctest` ở 1.24) — test code concurrent với fake clock, deterministic | Giải quyết vấn đề kinh điển "test code dùng `time.Sleep`/timer không đáng tin cậy, chạy chậm hoặc flaky" — nên thêm vào bài 06-testing |
| Map implementation dùng Swiss Tables (tăng performance, transparent — không đổi API) | Đáng nhắc trong 27-performance-engineering như một ví dụ "runtime tự cải thiện, không cần đổi code" |
| `weak` package (weak pointers) | Niche, dùng cho cache/memoization nâng cao — có thể để sau |

### Go 1.25 (giữa 2025)

| Tính năng | Vì sao quan trọng |
|---|---|
| **`GOMAXPROCS` tự động nhận diện cgroup/container CPU limit** | **Đây là một correction cần sửa trong `24-scheduler-and-memory-model.md`** — file đó đang khuyên dùng `go.uber.org/automaxprocs` để fix vấn đề GOMAXPROCS không nhận container limit. Từ Go 1.25, runtime tự làm việc này — thư viện automaxprocs giờ chỉ cần thiết nếu bạn *chưa* lên 1.25, hoặc cần custom logic. Cần update lại nội dung, không chỉ thêm mới. |
| `testing/synctest` graduate khỏi experimental (không cần `GOEXPERIMENT` nữa) | Càng có lý do dạy nó ngay ở bài 06 |
| `encoding/json/v2` (experimental, `GOEXPERIMENT=jsonv2`) | JSON stdlib viết lại — nhanh hơn, sửa nhiều pain point cũ (`omitzero`, custom marshaling dễ hơn). Chưa nên dạy sâu vì còn experimental, nhưng nên **mention** trong 12-json-struct-tags để không bị bất ngờ khi nó stable |
| Trace flight recorder (`runtime/trace`) — thu execution trace liên tục, chỉ dump khi có sự kiện bất thường (giống flight recorder máy bay) | Công cụ debug production mới, liên quan bài 18-tooling-debugging |

### Go 1.26 (bản đang dùng trong `go.mod` của bạn)

Tôi không có changelog chính thức đầy đủ của 1.26 trong dữ liệu huấn luyện (cutoff kiến thức khoảng đầu 2026) để liệt kê chi tiết đáng tin cậy — **khuyến nghị: đọc trực tiếp https://go.dev/doc/go1.26 trước khi viết doc mới**, đừng để tôi đoán nội dung release notes. Việc quan trọng hơn là: bạn đang chạy 1.26.1 nhưng học tài liệu neo ở 1.21/1.22 — khoảng cách 5 minor version là vấn đề thật, bất kể chi tiết 1.26 là gì.

---

## 3. Xu hướng ngành 2025–2026 chưa được cover (ngoài phạm vi "tính năng ngôn ngữ")

| Xu hướng | Tình trạng trong docs | Đề xuất |
|---|---|---|
| **ConnectRPC** (buf.build/connect-go) — lớp thay thế/bổ sung cho gRPC thuần, hỗ trợ HTTP/1.1 + JSON + browser-compatible, đang thành lựa chọn mặc định cho service mới ở nhiều công ty thay vì raw gRPC | `golang-roadmap.md` chỉ nhắc gRPC truyền thống (`google.golang.org/grpc` + `protoc`) | Thêm 1 đoạn so sánh trong Phase 4 của roadmap: khi nào dùng gRPC thuần vs ConnectRPC |
| **OpenTelemetry Go SDK** — chỉ là 1 dòng bullet trong Phase 6 | Quá sơ sài so với mức độ phổ biến thực tế (gần như bắt buộc ở production 2025+) | Đáng có 1 file riêng (tracing + metrics + log correlation), tương tự cách `16-logging-config.md` đã dạy sâu `slog` |
| **AI/LLM engineering bằng Go** — SDK chính thức (`anthropic-sdk-go`, `openai-go`), MCP (Model Context Protocol) server viết bằng Go (`mcp-go`), Go đang là ngôn ngữ phổ biến để viết AI agent CLI tools (chính công cụ bạn đang dùng để hỏi câu này thuộc nhóm xu hướng này) | Không có trong roadmap | Không bắt buộc cho track "backend API", nhưng đáng thêm như 1 optional milestone ở Phase 5 (CLI Tools) hoặc Phase 8 — vì đây là nơi Go đang giành thị phần rõ rệt nhất 2025-2026 |
| **golangci-lint v2** — đổi format config (`version: "2"`, tách `formatters:` khỏi `linters:`) | Đã kiểm tra thực tế: `golangci-lint --version` trên máy này là **v1.64.8** — `.golangci.yml` hiện tại (format v1) là đúng, **không cần migrate**. Ghi lại ở đây chỉ để nhớ: khi nâng cấp lên golangci-lint v2 trong tương lai, config sẽ cần viết lại theo schema mới | Không hành động gì bây giờ; theo dõi khi quyết định nâng version linter |
| **WASM/WASI** (`GOOS=wasip1`) cho edge computing (Cloudflare Workers, Fastly Compute) | Đã được note ở bài 21 là "không cấp bách" | Giữ nguyên đánh giá — vẫn đúng, đây là niche |
| **Supply chain / SLSA / sigstore-cosign** để ký binary release | `28-security-and-supply-chain.md` có `govulncheck`/`gosec` nhưng chưa có phần ký/verify binary | Đáng thêm 1 mục ngắn nếu track của bạn có Phase 6 (Cloud/DevOps) — liên quan trực tiếp release pipeline |

---

## 4. Ưu tiên hành động (theo mức độ quan trọng, không phải theo thứ tự file)

1. **Cao nhất — sửa ngay khi tới bài liên quan**: correction GOMAXPROCS/automaxprocs trong `24-scheduler-and-memory-model.md` (thông tin sai lệch với Go 1.25+, không chỉ là "thiếu").
2. **Cao — thêm trước khi học xong Phase tương ứng**: range-over-func iterators (ảnh hưởng cách viết code idiomatic hàng ngày từ giờ trở đi, không phải kiến thức "biết thêm cho vui").
3. **Trung bình**: `tool` directive (bài 26), `os.Root` (bài 28), `testing/synctest` (bài 06) — mỗi cái là 1 công cụ cụ thể thay thế pattern cũ đang dạy.
4. **Thấp/optional**: `unique` package, `weak` pointers, `encoding/json/v2` (còn experimental), WASM.
5. **Đã kiểm tra, không cần hành động**: `golangci-lint --version` = v1.64.8 → `.golangci.yml` format v1 hiện tại là đúng, không cần migrate sang schema v2.
6. **Ngành/career, để dành Phase 6-8**: OpenTelemetry đào sâu, ConnectRPC, AI/LLM engineering bằng Go — 3 mục này quyết định bạn có bắt kịp "trending" hay không nhiều hơn là chi tiết cú pháp version nào.

---

## 5. Trạng thái sau khi update toàn bộ

| Hành động | Trạng thái | File |
|---|---|---|
| Correction GOMAXPROCS/automaxprocs (Go 1.25+) | ✅ Đã sửa | `24-scheduler-and-memory-model.md` §6 |
| Range-over-func iterators (`iter.Seq`, `slices`/`maps` API mới) | ✅ Đã viết mới + link từ `09-generics.md` | `31-iterators-and-range-over-func.md` |
| Generic type alias (Go 1.24) | ✅ Đã thêm | `09-generics.md` §5 |
| `tool` directive trong go.mod (Go 1.24, thay `tools.go` hack) | ✅ Đã thêm | `26-modules-and-workspaces.md` §5 |
| `os.Root` (Go 1.24, path traversal) | ✅ Đã thêm | `28-security-and-supply-chain.md` §3.1 |
| `testing/synctest` (Go 1.24 experimental → 1.25 stable) | ✅ Đã thêm | `06-testing.md` §14.1 |
| Swiss Tables map + package `unique` | ✅ Đã thêm | `27-performance-engineering.md` §5.1 |
| ConnectRPC vs gRPC thuần | ✅ Đã thêm | `golang-roadmap.md` Phase 4 |
| OpenTelemetry đào sâu (traces/metrics/log correlation) | ✅ Đã viết mới, link từ roadmap Phase 6 | `32-observability-opentelemetry.md` |
| AI/LLM engineering bằng Go (MCP server, streaming, agent loop) | ✅ Đã viết mới, optional milestone Phase 5 | `33-ai-llm-engineering-go.md` |
| `golangci-lint` v1 vs v2 config | ✅ Đã kiểm tra thực tế — không cần hành động | — |
| WASM/WASI, `weak` pointers, `encoding/json/v2` | Giữ nguyên: optional, chưa viết — vẫn còn experimental hoặc quá niche | — |
| Go 1.26 chi tiết đầy đủ | Chưa viết — khuyến nghị đọc trực tiếp https://go.dev/doc/go1.26 trước khi bổ sung, tránh đoán nhầm changelog | — |
| Supply chain: ký binary (sigstore/cosign, SLSA) | Chưa viết — để dành khi học tới Phase 6 (Cloud/DevOps) thực tế | — |

---

**Liên quan:** [21-roadmap-gap-analysis.md](21-roadmap-gap-analysis.md), [09-generics.md](09-generics.md), [24-scheduler-and-memory-model.md](24-scheduler-and-memory-model.md), [26-modules-and-workspaces.md](26-modules-and-workspaces.md), [28-security-and-supply-chain.md](28-security-and-supply-chain.md)
</content>
</invoke>
