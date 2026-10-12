# 40. Chuẩn bị phỏng vấn Go — câu hỏi, thuật toán, system design

> Cách dùng: **che đáp án, tự trả lời thành tiếng 1–2 phút**, rồi so. Câu nào ấp úng → mở bài liên quan, viết code thử, thêm vào Anki. Đáp án ở đây là ý chính, không phải học thuộc.

---

## Phần A — 50 câu hỏi Go

### Ngôn ngữ & kiểu dữ liệu
1. **Slice gồm những gì? `append` hoạt động ra sao?** Header 3 trường: pointer, len, cap. `append` ghi vào mảng nền nếu còn cap (có thể làm slice khác cùng mảng nền thấy thay đổi), hết cap thì cấp mảng mới và copy. Sau `append` phải dùng giá trị trả về.
2. **Bẫy slice aliasing?** `b := a[:2]; b = append(b, x)` ghi đè `a[2]`. Tránh bằng full slice expression `a[:2:2]` hoặc `slices.Clone`.
3. **Map hoạt động thế nào? Thứ tự duyệt? Có an toàn đồng thời không?** Hash table (Swiss table từ 1.24). Thứ tự duyệt cố ý ngẫu nhiên. Không an toàn khi ghi đồng thời → `fatal error`. Đọc từ map `nil` OK, ghi vào map `nil` panic.
4. **`nil` slice vs slice rỗng?** Cả hai len 0 và `append` được; khác ở `== nil` và JSON (`null` vs `[]`).
5. **Array vs slice?** Array là giá trị, độ dài thuộc kiểu, gán là copy. Slice là view.
6. **`string` nội bộ? `rune` vs `byte`?** Header (pointer, len), bất biến, UTF-8. `range` trên string cho rune; `s[i]` cho byte. Nối chuỗi trong vòng lặp dùng `strings.Builder`.
7. **Giá trị zero của các kiểu?** `0`, `""`, `false`, `nil` (pointer, slice, map, chan, func, interface). Thiết kế type sao cho zero value dùng được (`sync.Mutex`, `bytes.Buffer`).
8. **Con trỏ vs giá trị receiver?** Con trỏ khi cần sửa, struct lớn, hoặc chứa mutex; nhất quán trong cùng một type. Method set của `T` không gồm method receiver `*T` → ảnh hưởng thoả interface.
9. **Interface nội bộ?** Hai từ: (type, data) — `eface` cho `any`, `iface` kèm itab cho interface có method. Interface chứa con trỏ nil **khác** `nil` ([29](29-typed-nil-interface-trap.md)).
10. **Typed-nil trap?** `var p *MyErr = nil; var err error = p; err != nil` là true. Luôn `return nil` tường minh.
11. **Embedding vs kế thừa?** Composition + method promotion; không có đa hình theo kiểu cha-con. Struct nhúng không "biết" struct ngoài.
12. **`defer` chạy khi nào? Thứ tự? Đối số?** Khi hàm return, LIFO. Đối số tính ngay lúc `defer`. `defer` trong vòng lặp tích lũy tới hết hàm → đóng file trong hàm con. Có thể sửa named return.
13. **`panic`/`recover`?** `recover` chỉ có tác dụng trong hàm được `defer` gọi trực tiếp, cùng goroutine. Dùng cho lỗi lập trình/biên giới (HTTP middleware), không thay cho `error`.
14. **Closure capture biến vòng lặp?** Trước Go 1.22: dùng chung một biến. Từ 1.22 (theo `go` trong go.mod): mỗi vòng một biến mới.
15. **Generics: khi nào dùng, khi nào không?** Khi logic giống nhau trên nhiều kiểu (container, `Map/Filter`). Không dùng khi interface đã đủ. `comparable`, `~int` (underlying type), `cmp.Ordered`.
16. **`==` trên struct/interface?** Struct so sánh được nếu mọi trường so sánh được. So interface chứa kiểu không so sánh được (slice, map, func) → panic runtime.
17. **`iota`?** Bộ đếm trong khối `const`, tăng mỗi dòng; dùng cho enum, bitflag (`1 << iota`).
18. **`new` vs `make`?** `new(T)` cấp zero `T` trả `*T`; `make` cho slice/map/chan, trả giá trị đã khởi tạo.

### Lỗi
19. **`errors.Is` vs `errors.As`?** `Is` so sánh giá trị sentinel qua chuỗi wrap; `As` tìm và gán một kiểu cụ thể.
20. **Wrap lỗi thế nào, khi nào không wrap?** `fmt.Errorf("ctx: %w", err)`. Không wrap nếu không muốn lộ chi tiết nội bộ thành một phần API ổn định.
21. **Sentinel vs typed error vs opaque?** Sentinel để so sánh; typed để mang dữ liệu; opaque (chỉ kiểm tra hành vi qua interface) giảm coupling.
22. **`errors.Join` / multi-wrap?** Gộp nhiều lỗi; `Is/As` duyệt tất cả nhánh.

### Concurrency
23. **Goroutine vs OS thread?** Stack khởi đầu nhỏ (~2KB, tự lớn), do runtime lập lịch; chi phí tạo rất thấp.
24. **Mô hình GMP?** G goroutine, M OS thread, P processor (context chạy, mỗi P có local run queue); số P = `GOMAXPROCS`. Work stealing giữa các P; syscall chặn thì P bàn giao sang M khác ([24](24-scheduler-and-memory-model.md)).
25. **Preemption?** Từ 1.14 có async preemption qua tín hiệu → vòng lặp chặt không còn chiếm P mãi.
26. **Unbuffered vs buffered channel?** Unbuffered đồng bộ hoá (rendezvous); buffered tách producer/consumer tới giới hạn cap.
27. **Hành vi channel nil/closed?** Gửi/nhận trên nil: chặn vĩnh viễn. Gửi vào closed: panic. Nhận từ closed: ra zero value, `ok=false`. Đóng channel đã đóng: panic. **Bên gửi** là bên đóng.
28. **`select` nhiều case sẵn sàng?** Chọn ngẫu nhiên đều. `default` làm nó không chặn.
29. **Mutex vs RWMutex vs atomic vs channel?** Channel để chuyển quyền sở hữu/điều phối; mutex để bảo vệ trạng thái chia sẻ; `RWMutex` khi đọc áp đảo (nhưng có overhead); atomic cho bộ đếm/cờ đơn giản.
30. **Data race là gì? Phát hiện?** Hai truy cập đồng thời, ít nhất một là ghi, không có happens-before. `go test -race`.
31. **Happens-before?** Gửi channel xảy ra trước nhận tương ứng hoàn tất; `Unlock` trước `Lock` kế; `go` statement trước goroutine bắt đầu; `sync.Once`… ([24](24-scheduler-and-memory-model.md)).
32. **Goroutine leak: nguyên nhân/phát hiện/phòng?** Chặn mãi trên channel/lock/IO không timeout. pprof goroutine, `goleak`. Mỗi `go` phải có đường dừng qua `ctx` ([37](37-concurrency-bug-lab.md)).
33. **`context` dùng để làm gì, không dùng để làm gì?** Huỷ, deadline, giá trị phạm vi request nhỏ (trace id). Không nhét tham số nghiệp vụ. Truyền làm tham số đầu, không lưu trong struct; luôn `defer cancel()` ([23](23-context-advanced.md)).
34. **`sync.WaitGroup` vs `errgroup`?** `errgroup` thêm lỗi đầu tiên + huỷ ctx + `SetLimit`.
35. **Worker pool: thiết kế?** N worker đọc từ channel job, kết quả ra channel, ctx để dừng, đóng channel job để kết thúc, `WaitGroup` rồi đóng channel kết quả.
36. **`sync.Pool` dùng khi nào?** Tái dùng object tạm cấp phát nhiều (buffer); có thể bị GC xoá bất cứ lúc nào, không dùng làm cache hay pool kết nối.
37. **`sync.Once`/`OnceValue`?** Khởi tạo lười an toàn đồng thời.
38. **Pipeline/fan-in/fan-out?** Chuỗi stage nối bằng channel; mỗi stage đóng channel đầu ra khi xong; huỷ bằng ctx.

### Runtime, bộ nhớ, hiệu năng
39. **Stack vs heap, escape analysis?** Compiler quyết định; giá trị thoát ra ngoài phạm vi hàm (trả con trỏ, lưu vào interface/closure, quá lớn) lên heap. Xem bằng `go build -gcflags=-m`.
40. **GC của Go?** Concurrent tri-color mark-sweep, không nén, tối ưu độ trễ thấp. Núm vặn: `GOGC`, `GOMEMLIMIT`. Giảm áp lực GC = giảm cấp phát.
41. **Cách tối ưu một hàm chậm?** Đo trước (`pprof` CPU/heap, benchmark + `benchstat`), tìm hot path, giảm cấp phát (prealloc `make([]T,0,n)`, `strings.Builder`, `sync.Pool`), tránh interface/reflect trong vòng nóng, sau đó đo lại ([27](27-performance-engineering.md)).
42. **Struct alignment?** Thứ tự trường ảnh hưởng padding; xếp trường lớn trước. Công cụ: `fieldalignment`.
43. **`GOMAXPROCS` trong container?** Từ Go 1.25 runtime tôn trọng giới hạn CPU cgroup; bản cũ cần `automaxprocs`.

### Thực hành backend
44. **Graceful shutdown?** Bắt `SIGTERM` → `srv.Shutdown(ctx)` (ngừng nhận mới, chờ request đang chạy) → đóng DB/queue → thoát; timeout < grace period của orchestrator.
45. **Middleware là gì?** `func(http.Handler) http.Handler`; logging, recover, auth, request-id, rate limit.
46. **Đảm bảo không bán quá kho?** Update nguyên tử có điều kiện / `FOR UPDATE` / optimistic version ([35](35-database-transactions-locking.md)).
47. **Thanh toán retry không trừ tiền 2 lần?** Idempotency key ([36](36-idempotency-and-reliability.md)).
48. **Thiết kế cấu hình/secret?** Env + validate lúc khởi động (fail fast), không log secret, không commit.
49. **Khi nào tách microservice?** Khi ranh giới nghiệp vụ rõ, đội/scale/deploy độc lập thực sự cần; kèm chi phí: mạng, nhất quán dữ liệu (saga/outbox), quan sát ([34](34-microservices-patterns.md)).
50. **Kể một bug khó bạn từng gặp.** Dùng nhật ký lỗi ở [37](37-concurrency-bug-lab.md): triệu chứng → giả thuyết → công cụ → nguyên nhân gốc → bài học.

---

## Phần B — Thuật toán & cấu trúc dữ liệu bằng Go (mục tiêu ~30 bài)

Idiom Go cần nhớ khi giải bài: `sort.Slice`/`slices.SortFunc`, `container/heap`, `container/list`, `map[T]struct{}` làm set, `[]byte`/`[]rune` để sửa chuỗi, `math.MaxInt`, đệ quy không tối ưu tail-call (cẩn thận độ sâu).

Danh sách theo chủ đề (mỗi mục 3–5 bài): hai con trỏ/cửa sổ trượt, hash map đếm, stack/queue (ngoặc hợp lệ, monotonic stack), tìm kiếm nhị phân, linked list, cây (DFS/BFS), heap (top-K), đồ thị (BFS/Topo sort, union-find), quy hoạch động cơ bản, backtracking.

### Bài mẫu 1 — LRU cache (câu hỏi cực kỳ phổ biến)

```go
type LRU[K comparable, V any] struct {
	cap int
	ll  *list.List           // phần tử mới dùng ở đầu
	m   map[K]*list.Element
}
type entry[K comparable, V any] struct { k K; v V }

func NewLRU[K comparable, V any](cap int) *LRU[K, V] {
	return &LRU[K, V]{cap: cap, ll: list.New(), m: make(map[K]*list.Element)}
}

func (c *LRU[K, V]) Get(k K) (v V, ok bool) {
	if e, hit := c.m[k]; hit {
		c.ll.MoveToFront(e)
		return e.Value.(*entry[K, V]).v, true
	}
	return
}

func (c *LRU[K, V]) Put(k K, v V) {
	if e, hit := c.m[k]; hit {
		e.Value.(*entry[K, V]).v = v
		c.ll.MoveToFront(e)
		return
	}
	c.m[k] = c.ll.PushFront(&entry[K, V]{k, v})
	if c.ll.Len() > c.cap {
		last := c.ll.Back()
		c.ll.Remove(last)
		delete(c.m, last.Value.(*entry[K, V]).k)
	}
}
```
O(1) mỗi thao tác. Câu hỏi theo sau: làm nó an toàn đồng thời (thêm `sync.Mutex`; `Get` cũng ghi nên không dùng `RWMutex` đọc được), thêm TTL, shard để giảm tranh chấp.

### Bài mẫu 2 — Top-K bằng heap

```go
type minHeap []int
func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *minHeap) Pop() any          { old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x }

func topK(nums []int, k int) []int {
	h := &minHeap{}
	for _, n := range nums {
		heap.Push(h, n)
		if h.Len() > k { heap.Pop(h) } // giữ k phần tử lớn nhất
	}
	return *h
}
```
O(n log k). Nhớ: `heap.Push/Pop` của package, không gọi trực tiếp method `Push/Pop` của kiểu.

### Bài mẫu 3 — Rate limiter token bucket (kết hợp concurrency)
Viết bằng `time.Now()` + mutex (tính token bù theo thời gian trôi, không cần goroutine nền); so sánh với `x/time/rate` ([36](36-idempotency-and-reliability.md)).

Gợi ý lộ trình: 5 bài/tuần × 6 tuần; luôn viết kèm test bảng. Mục tiêu là *nói được hướng giải và độ phức tạp*, không phải học thuộc đáp án.

---

## Phần C — System design (khung 45 phút)

**Khung trả lời:** (1) làm rõ yêu cầu chức năng + phi chức năng (QPS, độ trễ, nhất quán, dung lượng) → (2) ước lượng thô → (3) API + mô hình dữ liệu → (4) sơ đồ cấp cao → (5) đào sâu 1–2 điểm khó → (6) lỗi, quan sát, scale → (7) đánh đổi.

### Đề 1 — URL shortener
Điểm chính: sinh ID (counter + base62, hoặc hash + xử lý va chạm), đọc nhiều hơn ghi nhiều → cache (Redis) + CDN, 301 vs 302 (đo lượt click), DB key-value hoặc Postgres + index, hết hạn link, chống abuse (rate limit). Sau scale: shard theo key, tách bộ sinh ID.

### Đề 2 — Job queue / xử lý nền (gửi email đơn hàng)
Điểm chính: outbox trong cùng transaction với đơn hàng → worker đọc (`FOR UPDATE SKIP LOCKED` hoặc Kafka/NATS), at-least-once + idempotent consumer, retry backoff + DLQ, giới hạn đồng thời bằng worker pool, graceful shutdown không mất job, quan sát độ trễ hàng đợi.

### Đề 3 — API rate limiter phân tán
Điểm chính: thuật toán (token bucket, sliding window), lưu bộ đếm ở Redis (Lua script cho nguyên tử), key theo user/IP/API-key, hành vi khi Redis chết (fail open hay closed), header `Retry-After`, đồng bộ đồng hồ.

### Đề 4 — Hệ thống đặt hàng (chính dự án của bạn)
Điểm chính: tính idempotent, giữ chỗ kho (reserve → confirm/expire), thanh toán bất đồng bộ + webhook, saga bù trừ, nhất quán cuối cùng, outbox. Đây là đề bạn nên kể trôi chảy nhất vì bạn đã tự xây ([22](22-mini-ecommerce-project.md), [35](35-database-transactions-locking.md), [36](36-idempotency-and-reliability.md)).

---

## Phần D — Hành vi và cách kể chuyện dự án

Chuẩn bị 3 câu chuyện theo khung **Bối cảnh → Vấn đề → Hành động của bạn → Kết quả đo được → Bài học**:
1. Một bug concurrency/DB khó (từ nhật ký lỗi).
2. Một lần tối ưu hiệu năng có số liệu trước/sau (pprof + benchstat).
3. Một quyết định thiết kế bạn đánh đổi (vd. monolith trước, tách service sau; optimistic vs pessimistic).

Câu hỏi ngược lại nên hỏi người phỏng vấn: cách họ deploy/rollback, quy trình review, cách xử lý on-call, mức độ test.

## Danh sách kiểm tra cuối
- [ ] Trả lời trôi chảy ≥ 40/50 câu mục A không nhìn đáp án.
- [ ] Giải được LRU, top-K, và 25+ bài thuật toán khác bằng Go.
- [ ] Trình bày được 2 đề system design không cần gợi ý.
- [ ] Có 3 câu chuyện mục D, mỗi cái có số liệu.

**Liên quan:** [21-roadmap-gap-analysis.md](21-roadmap-gap-analysis.md), [37-concurrency-bug-lab.md](37-concurrency-bug-lab.md), [24-scheduler-and-memory-model.md](24-scheduler-and-memory-model.md)
