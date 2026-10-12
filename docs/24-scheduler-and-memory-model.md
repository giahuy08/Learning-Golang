# 24. Go Scheduler & Memory Model — Hiểu "tại sao" đằng sau goroutine

> Đây là ranh giới thường phân biệt "biết dùng goroutine" và "hiểu Go từ bên trong" — chủ đề phỏng vấn senior/staff kinh điển. Không bắt buộc để viết code production, nhưng bắt buộc để debug đúng khi có sự cố lạ (goroutine leak, performance không như kỳ vọng, race condition tinh vi).

---

## 1. GMP Model — Cách goroutine thực sự chạy

Trong C#, `Task`/`async-await` được quản lý bởi .NET ThreadPool — mỗi `Task` thường tương ứng gần với một OS thread khi cần. Go dùng mô hình khác hẳn: **GMP** (Goroutine – Machine – Processor).

- **G (Goroutine)** — đơn vị công việc bạn tạo bằng `go func()`. Cực nhẹ: stack khởi tạo chỉ **2KB** (so với OS thread thường ~1-8MB), có thể tạo hàng triệu goroutine.
- **M (Machine)** — OS thread thực sự, do hệ điều hành quản lý.
- **P (Processor)** — "giấy phép" để chạy Go code, số lượng P = `GOMAXPROCS` (mặc định = số CPU core). Mỗi P giữ một hàng đợi goroutine cục bộ (local run queue).

```
        P0                    P1
    ┌────────┐            ┌────────┐
    │ local  │            │ local  │
    │ queue  │            │ queue  │
    │ [G][G] │            │ [G][G] │
    └───┬────┘            └───┬────┘
        │                     │
        M0 (OS thread)        M1 (OS thread)
        │                     │
    ┌───▼────┐            ┌───▼────┐
    │  CPU   │            │  CPU   │
    └────────┘            └────────┘

    Global run queue (dùng khi local queue rỗng, hoặc khi goroutine mới không thuộc P nào)
```

**Điểm mấu chốt:** một M chỉ có thể chạy Go code khi nó "cầm" một P. Số P giới hạn (`GOMAXPROCS`) nghĩa là **chỉ tối đa N goroutine chạy song song thực sự tại một thời điểm**, dù bạn tạo hàng nghìn goroutine — phần còn lại đang chờ trong queue, không phải "chạy cùng lúc" theo nghĩa vật lý.

```go
import "runtime"

fmt.Println(runtime.GOMAXPROCS(0)) // đọc giá trị hiện tại, thường = số CPU core
runtime.GOMAXPROCS(4)              // giới hạn còn 4 — hiếm khi cần chỉnh tay
```

---

## 2. Work Stealing — Vì sao Go scheduler hiệu quả

Khi một P đã chạy hết goroutine trong local queue, nó **"ăn cắp"** (steal) một nửa goroutine từ local queue của P khác thay vì đứng im chờ global queue. Đây là lý do Go scheduler cân bằng tải tốt mà không cần lập trình viên can thiệp — khác với .NET nơi bạn đôi khi phải tự cấu hình `ThreadPool.SetMinThreads`.

---

## 3. Cooperative vs Preemptive — Goroutine có thể bị "giành" CPU không?

Trước Go 1.14, scheduler là **cooperative**: một goroutine chạy vòng lặp `for {}` không có function call nào bên trong có thể "đóng băng" cả chương trình vì scheduler không có cơ hội chen vào.

Từ Go 1.14, scheduler có **async preemption**: runtime gửi signal ngắt goroutine chạy quá lâu (dựa trên timer, ~10ms) để nhường chỗ cho goroutine khác — kể cả vòng lặp không có function call. Đây là lý do code Go hiện đại "tự chữa lành" tốt hơn so với các phiên bản Go cũ, nhưng vẫn **không nên** viết vòng lặp CPU-bound vô hạn không có điểm dừng hợp lý — nó vẫn chiếm P trong suốt thời gian chạy.

```go
// Vẫn là bad practice dù có async preemption:
// goroutine này chiếm 1 P liên tục, giảm throughput tổng thể
func busyLoop() {
    for {
        _ = calculatePrime() // CPU-bound liên tục, không I/O, không channel op
    }
}
```

---

## 4. Khi nào goroutine bị "block" khỏi M — Syscall và Network I/O

Khi goroutine gọi **blocking syscall** (đọc file, gọi hệ thống), M đang chạy nó sẽ bị block theo. Runtime phát hiện điều này và **tách P ra khỏi M bị block**, gán P đó cho một M khác (tạo mới hoặc lấy từ pool) để tiếp tục chạy goroutine khác — đây là lý do 1 goroutine block syscall không làm "đứng" toàn bộ chương trình.

Với **network I/O** (HTTP request, đọc socket), Go dùng `netpoller` (dựa trên epoll/kqueue/IOCP tùy OS) — goroutine gọi `net.Conn.Read()` sẽ được đưa ra khỏi M hoàn toàn (không chiếm cả M lẫn P) và chỉ được "đánh thức" khi data sẵn sàng. Đây là lý do bạn có thể mở hàng chục nghìn HTTP connection đồng thời trong Go server mà không cần hàng chục nghìn OS thread — khác biệt lớn nhất so với mô hình thread-per-request truyền thống.

---

## 5. Go Memory Model — "Happens-Before" là gì

Câu hỏi cốt lõi: khi goroutine A ghi một biến, goroutine B đọc biến đó — **B có chắc chắn thấy giá trị mới nhất không?**

Trên phần cứng hiện đại, CPU và compiler được phép **reorder** instruction miễn là không đổi kết quả trong 1 thread duy nhất. Nhưng giữa 2 goroutine, nếu không có **synchronization**, hoàn toàn có thể B đọc giá trị cũ dù A đã ghi "trước" theo thời gian thực.

```go
// BAD: Không có synchronization — CÓ THỂ (dù hiếm) B không bao giờ thấy ready == true
var ready bool
var data int

func writer() {
    data = 42      // (1)
    ready = true   // (2)
}

func reader() {
    for !ready {   // (3) — compiler/CPU có thể cache giá trị ready, không bao giờ thấy update
    }
    fmt.Println(data) // (4) — dù thấy ready == true, KHÔNG đảm bảo thấy data == 42 nếu không sync đúng
}
```

Go Memory Model định nghĩa rõ **các sự kiện nào tạo ra "happens-before" relationship** — tức là đảm bảo mọi ghi nhớ trước sự kiện đó sẽ được thấy bởi bên nhận sau sự kiện đó:

| Cơ chế | Đảm bảo happens-before |
|---|---|
| `go func()` | Câu lệnh `go` xảy ra trước khi goroutine mới bắt đầu chạy |
| Channel send → receive | Send trên channel happens-before receive tương ứng hoàn tất |
| Channel close → receive giá trị zero | Close happens-before receive nhận được zero value |
| `sync.Mutex.Unlock()` → `Lock()` (lần tiếp theo) | Unlock happens-before Lock kế tiếp thành công |
| `sync.WaitGroup.Wait()` return | Mọi `Done()` trước đó happens-before `Wait()` return |
| `sync/atomic` operations | Có memory ordering guarantee riêng, tương đương C++ `memory_order_seq_cst` |

```go
// GOOD: Dùng channel để tạo happens-before — đảm bảo reader thấy đúng data
var data int
done := make(chan bool)

func writer() {
    data = 42
    done <- true // send: happens-before receive
}

func reader() {
    <-done // receive: đảm bảo mọi ghi nhớ TRƯỚC send đã "commit" và visible ở đây
    fmt.Println(data) // luôn thấy 42
}
```

**Kết luận thực tế:** đây chính là lý do `go test -race` (race detector) và nguyên tắc "luôn dùng mutex/channel/atomic để chia sẻ state giữa goroutine" không phải là "best practice tùy chọn" — nó là yêu cầu bắt buộc về mặt correctness theo spec ngôn ngữ, không phải chỉ để "code sạch".

---

## 6. Ứng dụng thực tế — Khi nào cần biết điều này

1. **Debug race condition "chỉ xảy ra trên production, không tái hiện ở local"** — thường do compiler/CPU tối ưu hóa khác nhau giữa máy dev và server, hoặc do số CPU core khác nhau ảnh hưởng tới GOMAXPROCS.
2. **Quyết định `GOMAXPROCS` trong container** — trước Go 1.5 mặc định là 1; sau đó mặc định = số CPU logic của **máy host**, không phải CPU limit của container (Kubernetes `resources.limits.cpu`). Nếu container giới hạn 2 core nhưng host có 32 core, Go có thể tạo quá nhiều OS thread gây context-switching overhead.

   **Cập nhật quan trọng (Go 1.25+):** runtime giờ **tự động nhận diện CPU limit từ cgroup** (Linux) và set `GOMAXPROCS` đúng theo container limit — không cần code gì thêm, không cần thư viện ngoài. Đây là thay đổi mặc định, không phải opt-in.

   Thư viện bên thứ ba `go.uber.org/automaxprocs` (side-effect import, tự set `GOMAXPROCS` trong `init()`) từng là **bắt buộc phải thêm thủ công** ở mọi service Go chạy trong container trước Go 1.25 — vẫn hoạt động và vô hại nếu bạn giữ nó lại (chỉ set lại giá trị runtime đã tự tính đúng), nhưng với module `go 1.25` trở lên, **không còn cần thêm nó nữa** trừ khi bạn cần logic custom (ví dụ: giới hạn thấp hơn cgroup limit vì lý do riêng, hoặc chạy trên Go < 1.25).

   ```go
   // Trước Go 1.25: bắt buộc phải làm thế này trong container
   import _ "go.uber.org/automaxprocs"

   func main() {
       // automaxprocs tự detect cgroup limit và set GOMAXPROCS trong init()
   }
   ```

   ```go
   // Go 1.25+: không cần làm gì — runtime tự set GOMAXPROCS đúng theo cgroup limit
   func main() {
       // GOMAXPROCS đã đúng ngay từ đầu, kiểm tra bằng runtime.GOMAXPROCS(0) nếu muốn xác nhận
   }
   ```
3. **Hiểu tại sao `sync.Pool` giúp giảm GC pressure** — liên quan tới cách allocation tương tác với scheduler (mỗi P có local cache riêng cho `sync.Pool`, giảm lock contention).

---

## Bảng so sánh nhanh với .NET

| Go | .NET | Khác biệt cốt lõi |
|---|---|---|
| Goroutine (2KB stack, grow dynamically) | `Task` chạy trên ThreadPool | Goroutine rẻ hơn nhiều bậc, không cần async/await syntax riêng |
| GMP scheduler, work-stealing | ThreadPool work-stealing queue | Khá tương đồng về ý tưởng, nhưng Go tích hợp vào runtime ngôn ngữ, không phải library |
| `sync/atomic`, Go Memory Model | `System.Threading.Volatile`, .NET Memory Model | Cùng khái niệm happens-before, cú pháp khác |
| Network I/O qua netpoller, không cần thread riêng | I/O Completion Ports (IOCP) qua `async`/`await` | Ý tưởng giống nhau (non-blocking I/O), Go che giấu hoàn toàn sau `go func()` |

---

## Bài tập

1. Chạy chương trình sau với `go run -race` và giải thích tại sao race detector báo lỗi dù chương trình "chạy đúng" phần lớn thời gian:
```go
var counter int
for i := 0; i < 100; i++ {
    go func() { counter++ }()
}
time.Sleep(time.Second)
fmt.Println(counter)
```
2. Set `GOMAXPROCS(1)` và so sánh thời gian chạy của một chương trình CPU-bound có 4 goroutine tính toán song song, so với `GOMAXPROCS(4)`. Giải thích kết quả dựa trên mô hình GMP.
