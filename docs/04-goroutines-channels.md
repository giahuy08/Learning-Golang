# 04 — Goroutines & Channels

> **Đối tượng:** Lập trình viên C# đang học Go.
> **Yêu cầu trước:** Đã đọc `03-functions-methods-interfaces.md`.

---

## 1. Goroutine là gì — và tại sao nó KHÔNG phải là Thread

Trong C#, khi bạn muốn chạy song song, bạn dùng `Task`, `Thread`, hoặc `async/await`. Bên dưới, tất cả đều dựa trên OS thread hoặc thread pool của .NET runtime. Mỗi OS thread tốn khoảng 1–8 MB stack memory và việc context-switch giữa các thread do kernel quản lý — tốn kém.

Go đi theo hướng khác. **Goroutine** là một đơn vị thực thi *siêu nhẹ* được Go runtime quản lý, không phải OS. Go dùng mô hình **M:N scheduling**: M goroutine chạy trên N OS thread (N thường = số CPU core). Một goroutine mới chỉ tốn khoảng **2–8 KB stack** (stack này tự mở rộng khi cần), và context-switch giữa các goroutine xảy ra trong user-space, không cần kernel.

Hệ quả thực tế: bạn có thể chạy hàng trăm nghìn goroutine đồng thời mà không gây tràn bộ nhớ. Điều này không thể làm được với Thread trong C#.

```
C# mental model:          Go mental model:
Thread (1:1 OS thread)    Goroutine (M:N, managed by runtime)
Task (thread pool)        Goroutine (same lightweight primitive)
async/await               goroutine + channel
```

Quan trọng cần nhớ: goroutine vẫn có thể chạy thật sự song song trên nhiều CPU (không phải cooperative multitasking). Go runtime là một scheduler thực thụ với work-stealing. Khi một goroutine block I/O, runtime tự động chuyển goroutine khác lên chạy trên OS thread đó.

---

## 2. Từ khóa `go` — khởi động goroutine

Cú pháp đơn giản đến mức gây bất ngờ: thêm từ khóa `go` trước một lời gọi hàm.

```go
// BAD: main kết thúc trước khi goroutine chạy xong
// Kết quả: không in gì cả (hoặc in một phần)
package main

import "fmt"

func chao(ten string) {
	fmt.Println("Xin chào,", ten)
}

func main() {
	go chao("An")   // goroutine bắt đầu, nhưng main không đợi
	go chao("Bình") // goroutine thứ hai
	// main() kết thúc ở đây → toàn bộ chương trình thoát
	// hai goroutine trên có thể chưa kịp chạy
}
```

Vấn đề: **main goroutine** là goroutine đặc biệt — khi nó kết thúc, toàn bộ chương trình thoát, bao gồm tất cả goroutine đang chạy. Bạn phải có cơ chế chờ. Cách đơn giản nhất (nhưng không dùng cho production) là `time.Sleep`, cách đúng là `sync.WaitGroup` hoặc channel (sẽ trình bày ở phần sau).

```go
// GOOD: dùng channel để báo hiệu hoàn thành
package main

import "fmt"

func chao(ten string, done chan<- struct{}) {
	fmt.Println("Xin chào,", ten)
	done <- struct{}{} // báo hiệu: tôi xong rồi
}

func main() {
	done := make(chan struct{}, 2) // buffered để không block
	go chao("An", done)
	go chao("Bình", done)
	<-done // đợi goroutine 1
	<-done // đợi goroutine 2
}
```

---

## 3. Race Condition — cạnh tranh dữ liệu

Đây là vấn đề kinh điển của concurrency. Hãy xem ví dụ broken trước để hiểu tại sao Go cần channels.

```go
// BAD: nhiều goroutine ghi vào cùng biến — race condition
package main

import (
	"fmt"
	"sync"
)

func main() {
	dem := 0
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dem++ // ĐỌC + GHI không atomic → race condition!
			// Tương đương C#: int dem = 0; Parallel.For(0,1000, _ => dem++);
			// Kết quả: có thể là 998, 847, hoặc bất kỳ số nào < 1000
		}()
	}
	wg.Wait()
	fmt.Println("Kết quả:", dem) // KHÔNG đáng tin cậy
}
```

Chạy với `go run -race main.go` để Go race detector phát hiện ngay lỗi này.

```go
// GOOD: dùng sync/atomic hoặc Mutex (xem phần 12)
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func main() {
	var dem int64 // atomic cần kiểu cụ thể
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&dem, 1) // atomic operation — an toàn
		}()
	}
	wg.Wait()
	fmt.Println("Kết quả:", dem) // luôn luôn 1000
}
```

---

## 4. Channel — triết lý CSP của Go

Go được thiết kế theo triết lý **CSP (Communicating Sequential Processes)**. Thay vì nhiều goroutine cùng tranh nhau đọc/ghi vào một vùng nhớ chung (shared memory + lock — cách C# hay dùng), Go khuyến khích: **dùng channel để truyền dữ liệu giữa các goroutine**.

Câu thần chú của Go:

> *"Do not communicate by sharing memory; instead, share memory by communicating."*

Channel trong Go giống như một **đường ống có kiểu** (typed pipe). Goroutine gửi giá trị vào một đầu, goroutine khác nhận từ đầu kia. Không cần lock, không cần concurrent collection.

So sánh tương đương C#:
- `chan T` ≈ `Channel<T>` từ `System.Threading.Channels` (nhưng nguyên thủy hơn, tích hợp sẵn)
- Unbuffered channel ≈ `Channel.CreateUnbounded` với backpressure synchronous
- Buffered channel ≈ `Channel.CreateBounded(capacity)`

```go
// Tạo channel
ch := make(chan int)        // unbuffered channel truyền int
ch := make(chan int, 10)    // buffered channel, sức chứa 10
ch := make(chan string)     // unbuffered channel truyền string

// Gửi và nhận
ch <- 42        // gửi giá trị 42 vào channel
val := <-ch     // nhận giá trị từ channel, lưu vào val
<-ch            // nhận và bỏ giá trị (chỉ để drain hoặc synchronize)
```

---

## 5. Unbuffered Channel — đồng bộ hoàn toàn

Unbuffered channel hoạt động như **cuộc hẹn**: người gửi phải chờ người nhận sẵn sàng, và ngược lại. Không ai có thể rời đi trước khi cả hai đều có mặt. Đây là cơ chế synchronization mạnh mẽ.

```go
// GOOD: unbuffered channel làm điểm hẹn giữa hai goroutine
package main

import "fmt"

func tinh(a, b int, ketqua chan<- int) {
	ketqua <- a + b // gửi xong mới tiếp tục
}

func main() {
	ketqua := make(chan int) // unbuffered
	go tinh(3, 4, ketqua)
	go tinh(10, 20, ketqua)

	x := <-ketqua // nhận kết quả thứ nhất — main block ở đây cho đến khi có giá trị
	y := <-ketqua // nhận kết quả thứ hai
	fmt.Println(x, y) // thứ tự có thể khác nhau mỗi lần chạy
}
```

Đặc tính quan trọng: nếu goroutine gửi vào unbuffered channel mà không có ai nhận, goroutine đó **block mãi mãi** — đây là một trong những nguyên nhân gây goroutine leak (xem phần 14).

---

## 6. Buffered Channel — bất đồng bộ có giới hạn

Buffered channel có một hàng đợi nội bộ. Người gửi chỉ block khi hàng đợi đầy; người nhận chỉ block khi hàng đợi rỗng. Đây giống `BlockingCollection<T>` hoặc `Channel.CreateBounded(n)` trong C#.

```go
// GOOD: buffered channel cho phép gửi mà không cần nhận ngay
package main

import "fmt"

func main() {
	// Tạo buffered channel sức chứa 3
	ch := make(chan string, 3)

	// Có thể gửi 3 giá trị mà không bị block (không cần goroutine receiver)
	ch <- "công việc 1"
	ch <- "công việc 2"
	ch <- "công việc 3"
	// ch <- "công việc 4" // BAD: sẽ deadlock vì buffer đầy!

	// Nhận khi cần
	fmt.Println(<-ch) // "công việc 1"
	fmt.Println(<-ch) // "công việc 2"
	fmt.Println(<-ch) // "công việc 3"
}
```

**Khi nào dùng buffered vs unbuffered?**

Dùng unbuffered khi bạn muốn đảm bảo handoff — đảm bảo goroutine kia đã nhận xong trước khi tiếp tục. Dùng buffered khi bạn muốn decoupling — producer và consumer có thể chạy ở tốc độ khác nhau, buffer hấp thụ sự chênh lệch. Nếu không chắc, hãy bắt đầu với unbuffered; chỉ thêm buffer khi profiling cho thấy cần thiết.

---

## 7. Channel Direction — hướng của channel

Khi truyền channel vào hàm, Go cho phép bạn giới hạn hàm đó chỉ được gửi hoặc chỉ được nhận. Đây là cách Go enforce "ownership" ở compile time — giống `IProducerConsumerCollection` vs chỉ `IEnumerable` trong C#, nhưng ngay trong type system.

```go
package main

import "fmt"

// chan<- int : chỉ được GỬI vào channel (send-only)
// Tương đương producer
func sanXuat(ch chan<- int, so int) {
	ch <- so * so
	// val := <-ch // BAD: compile error — không được nhận từ send-only channel
}

// <-chan int : chỉ được NHẬN từ channel (receive-only)
// Tương đương consumer
func tieuThu(ch <-chan int) {
	val := <-ch
	fmt.Println("Nhận được:", val)
	// ch <- 99 // BAD: compile error — không được gửi vào receive-only channel
}

func main() {
	ch := make(chan int) // bidirectional — chuyển đổi tự động khi truyền vào hàm
	go sanXuat(ch, 7)
	tieuThu(ch) // Nhận được: 49
}
```

Quy tắc vàng: **hàm nào tạo channel thì nên đóng nó**, không phải bên nhận. Channel direction giúp compiler enforce điều này — chỉ bên có `chan T` (bidirectional) mới gọi được `close()`.

---

## 8. Đóng Channel — khi nào và tại sao

Đóng channel là tín hiệu nói "không có thêm dữ liệu nào nữa". Người nhận vẫn có thể drain hết dữ liệu còn lại trong buffer, sau đó sẽ nhận được zero value kèm `false` khi kiểm tra.

```go
// BAD: đóng channel từ phía receiver — gây panic
package main

func main() {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2

	// Tưởng tượng một goroutine khác đang gửi vào ch
	// Nếu receiver đóng ch trước khi sender xong → sender bị panic khi ghi
	close(ch)

	// Sau đó sender goroutine vẫn làm: ch <- 3 → PANIC: send on closed channel
}
```

```go
// GOOD: chỉ sender đóng channel, dùng comma-ok idiom để kiểm tra
package main

import "fmt"

func main() {
	ch := make(chan int, 5)

	// Producer goroutine — chính nó đóng channel khi xong
	go func() {
		for i := 1; i <= 5; i++ {
			ch <- i
		}
		close(ch) // sender đóng — đúng quy tắc
	}()

	// Consumer dùng comma-ok để phát hiện channel đã đóng
	for {
		val, ok := <-ch
		if !ok {
			fmt.Println("Channel đã đóng, không còn dữ liệu")
			break
		}
		fmt.Println("Nhận:", val)
	}
}
```

Nhận từ channel đã đóng: trả về zero value của kiểu dữ liệu và `false`. Gửi vào channel đã đóng: **panic ngay lập tức**. Đóng channel đã đóng: **panic ngay lập tức**. Đây là lý do ownership rõ ràng rất quan trọng.

---

## 9. `range` trên Channel — lặp đến khi đóng

Thay vì vòng lặp `for` với comma-ok, Go có cú pháp `range` gọn hơn — nó tự động dừng khi channel bị đóng.

```go
// GOOD: range tự dừng khi channel đóng
package main

import "fmt"

func taoSo(n int) <-chan int {
	ch := make(chan int)
	go func() {
		for i := 0; i < n; i++ {
			ch <- i * i // gửi bình phương của i
		}
		close(ch) // QUAN TRỌNG: phải close, không thì range block mãi
	}()
	return ch // trả về receive-only channel
}

func main() {
	for val := range taoSo(5) {
		fmt.Println(val) // 0 1 4 9 16
	}
	// range tự thoát khi taoSo đóng channel
}
```

```go
// BAD: quên close channel → range chạy mãi, goroutine bị leak
func taoSoBroken(n int) <-chan int {
	ch := make(chan int)
	go func() {
		for i := 0; i < n; i++ {
			ch <- i * i
		}
		// Quên close(ch) → goroutine nhận sẽ block mãi tại range
	}()
	return ch
}
```

Pattern `func taoSo() <-chan int` — trả về receive-only channel — là một pattern rất phổ biến trong Go để xây dựng pipeline.

---

## 10. `select` — chờ nhiều channel cùng lúc

`select` trong Go như `switch` nhưng dành cho channel operations. Nó block cho đến khi **một trong các case** sẵn sàng, rồi chạy case đó. Nếu nhiều case sẵn sàng cùng lúc, Go chọn ngẫu nhiên — tránh starvation.

Tương đương C#: `Task.WhenAny()`, nhưng tích hợp trực tiếp vào ngôn ngữ và không có overhead async state machine.

```go
// GOOD: select cơ bản — nhận từ channel nào sẵn trước
package main

import (
	"fmt"
	"time"
)

func main() {
	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() {
		time.Sleep(100 * time.Millisecond)
		ch1 <- "từ ch1"
	}()
	go func() {
		time.Sleep(200 * time.Millisecond)
		ch2 <- "từ ch2"
	}()

	// Đợi cái nào đến trước
	select {
	case msg1 := <-ch1:
		fmt.Println("Nhận:", msg1)
	case msg2 := <-ch2:
		fmt.Println("Nhận:", msg2)
	}
}
```

**Default case** — non-blocking select. Nếu không có channel nào sẵn, chạy `default` ngay lập tức. Hữu ích để kiểm tra channel có dữ liệu không mà không block.

```go
// GOOD: default case để non-blocking check
select {
case val := <-ch:
	fmt.Println("Có dữ liệu:", val)
default:
	fmt.Println("Không có dữ liệu ngay lúc này") // chạy ngay nếu ch rỗng
}
```

**Timeout pattern** — rất phổ biến trong production. Thay vì chờ mãi, đặt giới hạn thời gian bằng `time.After` (trả về một channel sẽ nhận giá trị sau khoảng thời gian chỉ định).

```go
// GOOD: timeout pattern — tương đương C# CancellationToken với timeout
package main

import (
	"fmt"
	"time"
)

func layDuLieuTuAPI() <-chan string {
	ch := make(chan string)
	go func() {
		time.Sleep(2 * time.Second) // giả lập API chậm
		ch <- "kết quả từ API"
	}()
	return ch
}

func main() {
	ketqua := layDuLieuTuAPI()

	select {
	case data := <-ketqua:
		fmt.Println("Thành công:", data)
	case <-time.After(1 * time.Second): // timeout sau 1 giây
		fmt.Println("Timeout! API quá chậm.")
	}
}
```

**Cancellation pattern** với `context` (thực tế hơn trong production):

```go
// GOOD: context cancellation — tương đương CancellationToken trong C#
package main

import (
	"context"
	"fmt"
	"time"
)

func congViec(ctx context.Context) {
	select {
	case <-time.After(5 * time.Second):
		fmt.Println("Công việc hoàn thành")
	case <-ctx.Done(): // nhận tín hiệu cancel
		fmt.Println("Bị hủy:", ctx.Err())
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // luôn gọi cancel để tránh resource leak
	congViec(ctx)
	// In ra: "Bị hủy: context deadline exceeded"
}
```

---

## 11. `sync.WaitGroup` — đợi nhiều goroutine

`sync.WaitGroup` là công cụ để main goroutine (hoặc bất kỳ goroutine nào) đợi một nhóm goroutine khác hoàn thành. Tương đương `Task.WhenAll()` trong C# nhưng không dùng async/await.

```go
// BAD: quên wg.Add trước go, hoặc Add sai vị trí
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		go func(id int) {
			wg.Add(1) // BAD: Add bên trong goroutine — có thể wg.Wait() chạy trước Add(1)!
			defer wg.Done()
			fmt.Println("Worker", id, "đang làm việc")
		}(i)
	}
	wg.Wait() // có thể thoát ngay nếu các goroutine chưa kịp Add(1)
}
```

```go
// GOOD: Add trước khi start goroutine
package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int, wg *sync.WaitGroup) {
	defer wg.Done() // đảm bảo Done được gọi dù có panic
	fmt.Printf("Worker %d bắt đầu\n", id)
	time.Sleep(100 * time.Millisecond) // giả lập công việc
	fmt.Printf("Worker %d xong\n", id)
}

func main() {
	var wg sync.WaitGroup

	for i := 1; i <= 5; i++ {
		wg.Add(1)    // Add TRƯỚC khi go — đây là quy tắc bắt buộc
		go worker(i, &wg) // truyền pointer, không copy WaitGroup
	}

	wg.Wait()
	fmt.Println("Tất cả workers đã hoàn thành")
}
```

Ba quy tắc của WaitGroup: (1) luôn `Add(1)` trước `go`, (2) luôn `Done()` bằng `defer`, (3) luôn truyền WaitGroup bằng pointer (không bao giờ copy).

---

## 12. `sync.Mutex` và `sync.RWMutex` — bảo vệ shared state

Đôi khi channel không phải giải pháp tốt nhất — ví dụ khi bạn cần một cache được truy cập từ nhiều goroutine. `sync.Mutex` giống `lock` trong C#, còn `sync.RWMutex` giống `ReaderWriterLockSlim`.

```go
// BAD: truy cập map từ nhiều goroutine không có khóa → race condition + panic
package main

import (
	"fmt"
	"sync"
)

func main() {
	cache := make(map[string]int)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cache["key"] = n // PANIC: concurrent map write!
		}(i)
	}
	wg.Wait()
	fmt.Println(cache)
}
```

```go
// GOOD: dùng Mutex để bảo vệ map
package main

import (
	"fmt"
	"sync"
)

// Cache an toàn với goroutine
type CacheAnToan struct {
	mu sync.RWMutex     // RWMutex: nhiều reader đồng thời, chỉ một writer
	du map[string]int
}

func (c *CacheAnToan) Dat(key string, val int) {
	c.mu.Lock()         // khóa ghi — không ai đọc hay ghi được
	defer c.mu.Unlock()
	c.du[key] = val
}

func (c *CacheAnToan) Lay(key string) (int, bool) {
	c.mu.RLock()         // khóa đọc — nhiều goroutine có thể đọc cùng lúc
	defer c.mu.RUnlock() // nhưng không ai ghi được
	val, ok := c.du[key]
	return val, ok
}

func main() {
	cache := &CacheAnToan{du: make(map[string]int)}
	var wg sync.WaitGroup

	// 50 writer goroutines
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cache.Dat(fmt.Sprintf("key%d", n), n*n)
		}(i)
	}

	// 50 reader goroutines
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cache.Lay(fmt.Sprintf("key%d", n)) // an toàn vì RLock
		}(i)
	}

	wg.Wait()
	fmt.Println("Xong, không có race condition")
}
```

**Mutex vs Channel:** Dùng Mutex khi bạn bảo vệ trạng thái (state) — cache, counter, map chung. Dùng Channel khi bạn truyền dữ liệu (data flow) hoặc cần coordination giữa các goroutine.

---

## 13. `sync.Once` — khởi tạo một lần duy nhất

`sync.Once` đảm bảo một hàm chỉ được gọi đúng một lần, dù có bao nhiêu goroutine gọi nó đồng thời. Đây là Go equivalent của `Lazy<T>` hay `double-checked locking` trong C#, nhưng đơn giản và an toàn hơn nhiều.

```go
// BAD: khởi tạo không an toàn khi có nhiều goroutine
package main

import (
	"fmt"
	"sync"
)

var ketNoiDB *string // giả lập database connection

func layKetNoi() *string {
	if ketNoiDB == nil { // kiểm tra này không atomic!
		// Nhiều goroutine có thể vào đây cùng lúc
		ket := "connected"
		ketNoiDB = &ket // ghi nhiều lần → race condition
	}
	return ketNoiDB
}

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			layKetNoi() // BAD: race condition khi khởi tạo
		}()
	}
	wg.Wait()
}
```

```go
// GOOD: sync.Once đảm bảo khởi tạo đúng một lần
package main

import (
	"fmt"
	"sync"
)

type KetNoiDB struct {
	url string
}

var (
	instance *KetNoiDB
	once     sync.Once
)

func LayInstance() *KetNoiDB {
	once.Do(func() {
		// Hàm này chỉ chạy MỘT LẦN DUY NHẤT
		// Dù 1000 goroutine gọi LayInstance() đồng thời
		fmt.Println("Khởi tạo kết nối database...")
		instance = &KetNoiDB{url: "postgres://localhost/mydb"}
	})
	return instance
}

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db := LayInstance()
			_ = db // dùng db
		}()
	}
	wg.Wait()
	fmt.Println("Instance:", LayInstance().url)
	// "Khởi tạo kết nối database..." chỉ in 1 lần
}
```

So với `Lazy<T>` trong C#: `sync.Once` không giữ reference đến giá trị — bạn tự quản lý biến riêng. Nhưng nó rõ ràng hơn và không có overhead reflection. Nếu hàm trong `once.Do` panic, Once vẫn được đánh dấu là đã chạy — hàm sẽ không được gọi lại.

---

## 14. Goroutine Leak — rò rỉ goroutine

Goroutine leak là khi goroutine bị block mãi mãi và không bao giờ kết thúc. Khác với memory leak ở chỗ goroutine còn chiếm cả CPU scheduler slot. Trong một server chạy lâu dài, goroutine leak tích lũy dần và gây chậm hệ thống hoặc OOM.

**Nguyên nhân phổ biến nhất:** goroutine block trên channel nhưng không có ai gửi/nhận vào channel đó (hoặc channel không bao giờ được đóng).

```go
// BAD: goroutine leak — goroutine chờ channel không bao giờ nhận
package main

import (
	"fmt"
	"time"
)

func layDuLieu(url string) <-chan string {
	ch := make(chan string)
	go func() {
		// Giả sử xảy ra lỗi sớm, hàm này return mà không gửi vào ch
		if url == "" {
			return // BAD: goroutine kết thúc nhưng ch không được gửi/đóng
		}
		ch <- "dữ liệu từ " + url
	}()
	return ch
}

func main() {
	ch := layDuLieu("") // url rỗng
	select {
	case data := <-ch:
		fmt.Println(data)
	case <-time.After(1 * time.Second):
		fmt.Println("Timeout")
		// main thoát, nhưng goroutine bên trong layDuLieu đã chết
		// OK trong trường hợp này, nhưng nếu gọi layDuLieu trong loop → leak!
	}
}
```

```go
// GOOD: dùng context để cancel goroutine khi không cần nữa
package main

import (
	"context"
	"fmt"
	"time"
)

func layDuLieu(ctx context.Context, url string) <-chan string {
	ch := make(chan string, 1) // buffered 1 để sender không bị block
	go func() {
		// Giả lập fetch
		select {
		case <-time.After(500 * time.Millisecond):
			ch <- "dữ liệu từ " + url
		case <-ctx.Done():
			// Context bị cancel → goroutine thoát sạch sẽ
			return
		}
	}()
	return ch
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	ch := layDuLieu(ctx, "https://api.example.com")
	select {
	case data := <-ch:
		fmt.Println("Nhận được:", data)
	case <-ctx.Done():
		fmt.Println("Hủy:", ctx.Err())
	}
}
```

**Cách phát hiện goroutine leak trong production:** dùng `runtime.NumGoroutine()` để monitor số lượng goroutine, hoặc endpoint `/debug/pprof/goroutine` từ package `net/http/pprof`. Nếu số goroutine tăng đều theo thời gian mà không giảm, đó là dấu hiệu leak.

**Quy tắc phòng tránh:** mọi goroutine phải có ít nhất một con đường để kết thúc. Khi goroutine chờ channel, phải có ai đó đóng channel đó hoặc truyền tín hiệu cancel (context).

---

## 15. Các lỗi thường gặp

| Lỗi | Hậu quả | Cách sửa |
|-----|---------|----------|
| `wg.Add(1)` bên trong goroutine | `wg.Wait()` có thể thoát sớm | Luôn `Add` trước `go` |
| `close(ch)` từ phía receiver | Panic nếu sender còn ghi | Chỉ sender mới đóng channel |
| `close(ch)` nhiều lần | Panic ngay lập tức | Dùng `sync.Once` để bọc `close` |
| `range` trên channel không bao giờ đóng | Goroutine block mãi, leak | Sender phải `close` khi xong |
| Copy `sync.Mutex` / `sync.WaitGroup` | Race condition tinh vi | Luôn dùng pointer `*sync.Mutex` |
| Gửi vào channel đã đóng | Panic | Dùng `recover` hoặc đảm bảo ownership rõ ràng |
| Goroutine không có exit path | Goroutine leak | Dùng `context.Context` để cancel |
| Dùng unbuffered channel trong select không có default | Deadlock | Dùng buffered hoặc thêm default |
| Không dùng `-race` khi test | Race condition không bị phát hiện | Luôn chạy `go test -race ./...` |
| Nhầm goroutine capture loop variable | Tất cả goroutine dùng giá trị cuối cùng | Truyền biến qua tham số hàm |

---

## Best Practices

**Thiết kế ownership rõ ràng.** Mỗi channel nên có một goroutine "chủ sở hữu" duy nhất chịu trách nhiệm tạo, gửi, và đóng nó. Khi ownership rõ ràng, các lỗi về đóng channel sai chỗ tự nhiên biến mất.

**Ưu tiên channel cho data flow, Mutex cho state protection.** Đây không phải quy tắc tuyệt đối nhưng là kim chỉ nam tốt. Nếu bạn đang truyền dữ liệu từ goroutine A sang goroutine B, dùng channel. Nếu bạn đang bảo vệ một cấu trúc dữ liệu được truy cập từ nhiều nơi, dùng Mutex.

**Luôn pass context xuống call chain.** Mọi hàm khởi động goroutine dài hạn (HTTP handler, background job, v.v.) nên nhận `context.Context` làm tham số đầu tiên. Đây là convention chuẩn của Go ecosystem và là cách duy nhất để cancel goroutine một cách graceful.

**Luôn chạy tests với `-race`.** Go race detector có chi phí runtime khoảng 5–10x nhưng cực kỳ chính xác. Tích hợp vào CI: `go test -race ./...`. Đừng chờ đến khi lên production mới phát hiện race condition.

**Đừng cố optimize sớm với buffer size.** Buffer không phải liều thuốc thần kỳ. Channel có buffer vẫn có thể block. Bắt đầu với unbuffered hoặc buffer nhỏ, đo lường, rồi điều chỉnh. Buffer size quá lớn che giấu vấn đề thiết kế.

**`defer cancel()` ngay sau `context.WithTimeout/WithCancel`.** Đây là pattern bắt buộc — quên `cancel()` gây resource leak dù context có hết hạn đi nữa.

---

## Bài tập thực hành

### Bài tập 1: Fan-out / Fan-in Pipeline

Implement một pipeline xử lý danh sách URL: nhiều worker goroutine fetch song song (giả lập), kết quả gom lại theo thứ tự tùy ý.

```go
// Exercise 1: Fan-out Fan-in
// Chạy: go run exercise1.go
package main

import (
	"fmt"
	"sync"
	"time"
)

type KetQua struct {
	URL  string
	Data string
	Err  error
}

// fetchURL giả lập fetch một URL (trong thực tế dùng http.Get)
func fetchURL(url string) KetQua {
	// Giả lập độ trễ mạng khác nhau
	switch url {
	case "https://api1.example.com":
		time.Sleep(100 * time.Millisecond)
	case "https://api2.example.com":
		time.Sleep(200 * time.Millisecond)
	case "https://api3.example.com":
		time.Sleep(50 * time.Millisecond)
	default:
		time.Sleep(150 * time.Millisecond)
	}
	return KetQua{URL: url, Data: fmt.Sprintf("response từ %s", url)}
}

// fanOut phân phối công việc cho nhiều worker
func fanOut(urls []string, soWorker int) <-chan KetQua {
	viecLam := make(chan string, len(urls))    // input channel có buffer
	ketQua := make(chan KetQua, len(urls))     // output channel có buffer

	// Đưa tất cả URLs vào channel
	for _, url := range urls {
		viecLam <- url
	}
	close(viecLam) // đóng sau khi đã đưa hết việc

	// Khởi động các workers
	var wg sync.WaitGroup
	for i := 0; i < soWorker; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for url := range viecLam { // nhận việc cho đến khi channel đóng
				fmt.Printf("[Worker %d] đang fetch %s\n", workerID, url)
				ketQua <- fetchURL(url)
			}
		}(i + 1)
	}

	// Goroutine phụ đợi tất cả workers xong rồi đóng output channel
	go func() {
		wg.Wait()
		close(ketQua) // báo hiệu: không còn kết quả nào nữa
	}()

	return ketQua
}

func main() {
	urls := []string{
		"https://api1.example.com",
		"https://api2.example.com",
		"https://api3.example.com",
		"https://api4.example.com",
		"https://api5.example.com",
	}

	batDau := time.Now()
	ketQuaChan := fanOut(urls, 3) // 3 workers xử lý 5 URLs song song

	for kq := range ketQuaChan {
		if kq.Err != nil {
			fmt.Printf("Lỗi fetch %s: %v\n", kq.URL, kq.Err)
		} else {
			fmt.Printf("OK: %s\n", kq.Data)
		}
	}

	fmt.Printf("\nTổng thời gian: %v (nếu tuần tự sẽ mất ~600ms)\n", time.Since(batDau))
}
```

### Bài tập 2: Worker Pool đầy đủ

Implement một worker pool tổng quát hơn — cho phép submit job bất kỳ lúc nào, có timeout, graceful shutdown.

```go
// Exercise 2: Worker Pool với graceful shutdown
// Chạy: go run exercise2.go
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Job là một đơn vị công việc cần xử lý
type Job struct {
	ID    int
	Input int
}

// Result là kết quả từ một Job
type Result struct {
	JobID  int
	Output int
	Err    error
}

// WorkerPool quản lý một nhóm goroutine worker
type WorkerPool struct {
	soWorker int
	viecChan chan Job
	ketQua   chan Result
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewWorkerPool tạo pool mới với số worker và buffer size chỉ định
func NewWorkerPool(soWorker, bufferSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		soWorker: soWorker,
		viecChan: make(chan Job, bufferSize),
		ketQua:   make(chan Result, bufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}
	pool.start()
	return pool
}

// start khởi động tất cả worker goroutines
func (p *WorkerPool) start() {
	for i := 1; i <= p.soWorker; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			p.worker(id)
		}(i)
	}
}

// worker xử lý từng job cho đến khi pool shutdown
func (p *WorkerPool) worker(id int) {
	for {
		select {
		case job, ok := <-p.viecChan:
			if !ok {
				// Channel đóng → thoát vòng lặp
				fmt.Printf("[Worker %d] thoát\n", id)
				return
			}
			// Xử lý job
			time.Sleep(50 * time.Millisecond) // giả lập công việc
			p.ketQua <- Result{
				JobID:  job.ID,
				Output: job.Input * job.Input,
			}
		case <-p.ctx.Done():
			// Pool bị cancel → thoát ngay
			fmt.Printf("[Worker %d] bị cancel\n", id)
			return
		}
	}
}

// Submit gửi job vào pool (non-blocking nếu buffer còn chỗ)
func (p *WorkerPool) Submit(job Job) error {
	select {
	case p.viecChan <- job:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("pool đã shutdown")
	}
}

// Results trả về channel để nhận kết quả
func (p *WorkerPool) Results() <-chan Result {
	return p.ketQua
}

// Shutdown dừng pool gracefully: đợi jobs đang chạy xong
func (p *WorkerPool) Shutdown() {
	close(p.viecChan) // không nhận job mới, workers drain jobs còn lại
	p.wg.Wait()       // đợi tất cả workers xong
	close(p.ketQua)   // không còn kết quả nào nữa
	p.cancel()        // cleanup context
}

func main() {
	// Tạo pool 3 workers, buffer 10 jobs
	pool := NewWorkerPool(3, 10)

	// Submit 10 jobs
	go func() {
		for i := 1; i <= 10; i++ {
			err := pool.Submit(Job{ID: i, Input: i})
			if err != nil {
				fmt.Println("Lỗi submit:", err)
			}
		}
		// Sau khi submit xong tất cả, shutdown pool
		pool.Shutdown()
	}()

	// Nhận và in kết quả
	tong := 0
	for result := range pool.Results() {
		fmt.Printf("Job %d: %d^2 = %d\n", result.JobID, result.JobID, result.Output)
		tong++
	}
	fmt.Printf("\nĐã xử lý %d jobs\n", tong)

	// Demo: shutdown với context timeout
	fmt.Println("\n--- Demo timeout shutdown ---")
	pool2 := NewWorkerPool(2, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Submit nhiều job nhưng timeout trước khi xong
	go func() {
		for i := 1; i <= 20; i++ {
			select {
			case <-ctx.Done():
				pool2.Shutdown()
				return
			default:
				pool2.Submit(Job{ID: i, Input: i})
			}
		}
		pool2.Shutdown()
	}()

	dem := 0
	for range pool2.Results() {
		dem++
	}
	fmt.Printf("Xử lý được %d/20 jobs trước khi timeout\n", dem)
}
```

---

## Bảng tổng kết — Go vs C# Concurrency

| Khái niệm Go | Tương đương C# | Khác biệt chính |
|--------------|----------------|-----------------|
| `goroutine` | `Task` / `Thread` | M:N scheduling, 2-8KB stack ban đầu |
| `chan T` | `Channel<T>` / `BlockingCollection<T>` | Nguyên thủy của ngôn ngữ, không cần import |
| Unbuffered channel | `SemaphoreSlim(0,1)` như rendezvous | Sender và receiver gặp nhau tại channel |
| Buffered channel | `Channel.CreateBounded(n)` | Buffer trong ngôn ngữ, không cần wrapper |
| `select` | `Task.WhenAny()` | Tích hợp trong ngôn ngữ, không có async overhead |
| `sync.WaitGroup` | `Task.WhenAll()` / `CountdownEvent` | Imperative counting, không có async state machine |
| `sync.Mutex` | `lock` / `Monitor` | Tương tự, nhưng không tích hợp vào runtime |
| `sync.RWMutex` | `ReaderWriterLockSlim` | Tương tự |
| `sync.Once` | `Lazy<T>` | Đơn giản hơn, không giữ reference |
| `context.Context` | `CancellationToken` | Convention toàn hệ sinh thái, truyền qua tham số |
| `-race` flag | Analyzer / Thread Sanitizer | Dynamic detection, rất chính xác |

---

## Next

[05 — Context](05-context.md)

Tiếp theo: Go không có exceptions. Tìm hiểu tại sao `error` là một value, cách handle lỗi một cách tường minh, và khi nào dùng `panic`/`recover` (hint: hiếm hơn bạn nghĩ).
