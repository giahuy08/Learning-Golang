# 37. Concurrency Bug Lab — tự gây bug, tự bắt bug

> Cách duy nhất để nhớ lâu: viết đoạn code lỗi, chạy, **thấy nó hỏng**, dùng công cụ tìm ra, rồi sửa. Mỗi bài dưới đây làm theo thứ tự đó. Ghi từng bug vào "nhật ký lỗi" (mục 7) — đó là chất liệu phỏng vấn.

Công cụ: `go test -race`, `go test -count=100`, `pprof` goroutine profile, `go.uber.org/goleak`.

---

## Bug 1 — Data race trên map

```go
m := map[int]int{}
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
	wg.Add(1)
	go func() { defer wg.Done(); m[i] = i }() // ghi đồng thời
}
wg.Wait()
```
Triệu chứng: `fatal error: concurrent map writes` (không recover được) hoặc im lặng sai dữ liệu. `-race` báo `DATA RACE`.
Sửa: `sync.Mutex`/`RWMutex` bao quanh map; hoặc `sync.Map` (chỉ hợp khi key ghi một lần đọc nhiều, hoặc key tách biệt giữa các goroutine); hoặc để một goroutine sở hữu map và nhận lệnh qua channel.

## Bug 2 — `wg.Add` đặt sai chỗ

```go
for i := 0; i < 3; i++ {
	go func() { wg.Add(1); defer wg.Done(); work() }() // Add chạy muộn hơn Wait
}
wg.Wait() // có thể return trước khi goroutine kịp Add
```
Sửa: `wg.Add(1)` **trước** `go`. (Go 1.25+ có `wg.Go(func(){...})` gọn hơn — kiểm tra phiên bản của bạn.)

## Bug 3 — Goroutine leak: gửi vào channel không ai nhận

```go
func firstResult(urls []string) string {
	ch := make(chan string) // unbuffered
	for _, u := range urls {
		go func() { ch <- fetch(u) }()
	}
	return <-ch // chỉ nhận 1; len(urls)-1 goroutine kẹt ở ch <- vĩnh viễn
}
```
Phát hiện: số goroutine tăng dần (`runtime.NumGoroutine()`, `/debug/pprof/goroutine?debug=1`, stack đều dừng ở `chan send`).
Sửa: `make(chan string, len(urls))` (buffered đủ chỗ), hoặc dùng `ctx` để huỷ các goroutine còn lại:

```go
ctx, cancel := context.WithCancel(ctx); defer cancel()
select { case ch <- fetch(ctx, u): case <-ctx.Done(): }
```

## Bug 4 — Goroutine leak: vòng lặp không có đường thoát

```go
go func() {
	for { select { case v := <-in: handle(v) } } // không có <-ctx.Done() và không check đóng channel
}()
```
Sửa: thêm `case <-ctx.Done(): return` và `v, ok := <-in; if !ok { return }`. Quy tắc: **mỗi `go` phải trả lời được "khi nào goroutine này dừng và ai làm nó dừng?"**

## Bug 5 — Deadlock do thứ tự lock

```go
// goroutine A: a.Lock(); b.Lock()
// goroutine B: b.Lock(); a.Lock()
```
Triệu chứng: treo vĩnh viễn (runtime chỉ báo `all goroutines are asleep` khi **toàn bộ** đều kẹt; trong server còn goroutine khác thì không báo gì!). Phát hiện: dump goroutine (`kill -QUIT` hoặc pprof) thấy 2 goroutine chờ `sync.Mutex.Lock`.
Sửa: thứ tự khoá nhất quán; hoặc gộp thành một mutex; hoặc thiết kế lại để không giữ 2 lock.

## Bug 6 — Lock bị copy

```go
type Counter struct{ mu sync.Mutex; n int }
func (c Counter) Inc() { c.mu.Lock(); c.n++; c.mu.Unlock() } // receiver theo giá trị: copy cả mutex
```
`go vet` bắt (`copylocks`). Sửa: receiver con trỏ `*Counter`.

## Bug 7 — Rò timer / ticker

```go
for { select { case <-time.After(time.Second): ...; case v := <-ch: ... } } // mỗi vòng lặp tạo timer mới
t := time.NewTicker(time.Second) // quên t.Stop() → rò
```
Go 1.23+ GC timer chưa dùng tốt hơn, nhưng vẫn nên dùng `time.NewTimer` + `Reset`, và luôn `defer t.Stop()` cho Ticker.

## Bug 8 — Đọc biến chia sẻ không đồng bộ

```go
var done bool
go func() { time.Sleep(time.Second); done = true }()
for !done {} // có thể không bao giờ thoát: không có happens-before
```
Sửa: `atomic.Bool`, channel hoặc mutex. Xem [24-scheduler-and-memory-model.md](24-scheduler-and-memory-model.md).

## Bug 9 — Panic trong goroutine kéo sập cả process

`recover` ở `main` **không** bắt được panic của goroutine khác. Mỗi goroutine do bạn tạo trong server cần recover riêng (hoặc dùng wrapper `safeGo`). `net/http` đã recover cho từng request handler, nhưng goroutine bạn tự `go` trong handler thì không.

---

## Bắt bug trong test: goleak

```go
func TestMain(m *testing.M) { goleak.VerifyTestMain(m) } // go.uber.org/goleak: fail nếu test để lại goroutine
```
```bash
go test -race -count=50 ./...     # chạy lặp để lộ bug thời gian
GORACE="halt_on_error=1" go test -race ./...
```
Race detector chỉ thấy race **đã xảy ra trong lần chạy đó** và làm chậm ~5–10×; nó không có false positive, nhưng có false negative → chạy lặp và có test đồng thời thật.

## Bài tập
1. Với mỗi Bug 1–8: viết file `_test.go` tái hiện, xác nhận test **fail/treo/`-race` báo**, rồi sửa cho pass.
2. Viết server nhỏ rò goroutine (Bug 3), bật `net/http/pprof`, chạy 1000 request, đọc `/debug/pprof/goroutine?debug=2` để chỉ ra dòng gây rò.
3. Viết `safeGo(fn)` log stack khi panic.

## 7. Nhật ký lỗi (mẫu)

| Ngày | Triệu chứng | Nguyên nhân gốc | Cách phát hiện | Cách sửa |
|---|---|---|---|---|
| | goroutine tăng 1/request | gửi channel không có receiver | pprof goroutine | buffered + ctx |

**Liên quan:** [04-goroutines-channels.md](04-goroutines-channels.md), [15-concurrency-patterns.md](15-concurrency-patterns.md), [24-scheduler-and-memory-model.md](24-scheduler-and-memory-model.md), [18-tooling-debugging.md](18-tooling-debugging.md)
