# 15. Advanced Concurrency Patterns

Go được thiết kế với concurrency là tính năng cốt lõi, không phải tính năng bổ sung. Trong C#, bạn đã quen với `async/await`, `Task.WhenAll`, `SemaphoreSlim`, và TPL (Task Parallel Library). Go tiếp cận vấn đề này theo hướng khác: thay vì ẩn đi các chi tiết concurrency sau lớp abstraction, Go để lộ chúng một cách rõ ràng thông qua goroutines và channels. Triết lý này xuất phát từ câu nói nổi tiếng của Go: *"Do not communicate by sharing memory; instead, share memory by communicating."*

Bài này tập trung vào các pattern thực tế bạn sẽ gặp trong production code, không phải lý thuyết đơn thuần.

---

## 1. errgroup — Chạy concurrent tasks và thu thập lỗi

Trong C#, khi bạn muốn chạy nhiều task song song và xử lý lỗi, bạn dùng `Task.WhenAll`:

```csharp
// C# - Task.WhenAll
try
{
    await Task.WhenAll(
        FetchUrlAsync("https://example.com/a"),
        FetchUrlAsync("https://example.com/b"),
        FetchUrlAsync("https://example.com/c")
    );
}
catch (Exception ex)
{
    // Chỉ nhận được một exception đầu tiên,
    // các task khác vẫn tiếp tục chạy!
}
```

Vấn đề với `Task.WhenAll` là khi một task fail, các task còn lại vẫn tiếp tục chạy đến khi hoàn thành. Go giải quyết điều này với `errgroup` từ package `golang.org/x/sync/errgroup`. Khi một goroutine trả về lỗi, `errgroup.WithContext` sẽ hủy một `context.Context` để các goroutine còn lại có thể dừng lại sớm.

```go
// BAD: tự quản lý goroutines và lỗi — rất dễ bị race condition và goroutine leak
package main

import (
    "fmt"
    "sync"
)

func fetchAll_BAD(urls []string) error {
    var wg sync.WaitGroup
    var firstErr error // NGUY HIỂM: race condition khi nhiều goroutine ghi cùng lúc

    for _, url := range urls {
        wg.Add(1)
        go func(u string) {
            defer wg.Done()
            if err := fetch(u); err != nil {
                firstErr = err // race condition!
            }
        }(url)
    }

    wg.Wait()
    return firstErr
}

func fetch(url string) error { return nil }
func main()                  {}
```

```go
// GOOD: dùng errgroup để quản lý goroutines và lỗi một cách an toàn
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "golang.org/x/sync/errgroup"
)

// fetchURL thực hiện HTTP GET và trả về lỗi nếu status không phải 2xx
func fetchURL(ctx context.Context, url string) error {
    // Tạo request với context để có thể cancel
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("tạo request cho %s: %w", url, err)
    }

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("fetch %s: %w", url, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("%s trả về status %d", url, resp.StatusCode)
    }

    fmt.Printf("OK: %s (%d)\n", url, resp.StatusCode)
    return nil
}

func main() {
    urls := []string{
        "https://httpbin.org/get",
        "https://httpbin.org/status/404", // URL này sẽ fail
        "https://httpbin.org/delay/10",   // URL này sẽ bị cancel khi URL trên fail
    }

    // WithContext trả về group và một context đã được "wrapping"
    // Khi một goroutine trả về lỗi, context này sẽ bị cancel
    g, ctx := errgroup.WithContext(context.Background())

    for _, url := range urls {
        url := url // Tạo bản copy — quan trọng! (xem phần Common Mistakes)
        g.Go(func() error {
            return fetchURL(ctx, url)
        })
    }

    // Wait block cho đến khi tất cả goroutines hoàn thành
    // Trả về lỗi đầu tiên (nếu có)
    if err := g.Wait(); err != nil {
        fmt.Println("Lỗi:", err)
    }
}
```

`errgroup.WithContext` là pattern chuẩn trong Go để thay thế `Task.WhenAll`. Điểm khác biệt quan trọng: bạn phải tự thiết kế code để *phản hồi* với việc context bị cancel — Go không tự động dừng goroutine của bạn. Đây là sự đánh đổi có chủ ý: bạn có toàn quyền kiểm soát, nhưng phải gánh chịu trách nhiệm.

---

## 2. Worker Pool Pattern — Giới hạn concurrency

Hãy tưởng tượng bạn có 10.000 file cần xử lý. Nếu bạn spawn 10.000 goroutines cùng lúc, bạn có thể làm quá tải database, network, hoặc CPU. Worker Pool giải quyết vấn đề này bằng cách giới hạn số goroutines hoạt động đồng thời.

Trong C#, bạn có thể dùng `Parallel.ForEach` với `MaxDegreeOfParallelism` hoặc TPL Dataflow. Trong Go, pattern phổ biến nhất là dùng channel làm job queue.

```go
// BAD: spawn goroutine cho mỗi job — không kiểm soát được số lượng concurrent goroutines
package main

import (
    "fmt"
    "sync"
)

func processAll_BAD(jobs []int) {
    var wg sync.WaitGroup
    for _, job := range jobs {
        wg.Add(1)
        job := job
        go func() { // Có thể tạo ra hàng nghìn goroutines cùng lúc!
            defer wg.Done()
            fmt.Println("processing", job)
        }()
    }
    wg.Wait()
}

func main() {}
```

```go
// GOOD: Worker Pool với channel làm job queue
package main

import (
    "fmt"
    "sync"
    "time"
)

// Result chứa kết quả của mỗi job
type Result struct {
    JobID  int
    Output string
    Err    error
}

// processJob mô phỏng công việc tốn thời gian (vd: resize ảnh, gọi API...)
func processJob(jobID int) Result {
    time.Sleep(100 * time.Millisecond) // mô phỏng công việc
    return Result{
        JobID:  jobID,
        Output: fmt.Sprintf("kết quả của job %d", jobID),
    }
}

func workerPool(numWorkers int, jobs []int) []Result {
    // jobCh là "hàng đợi" — workers đọc từ đây
    jobCh := make(chan int, len(jobs))
    // resultCh để thu thập kết quả từ tất cả workers
    resultCh := make(chan Result, len(jobs))

    var wg sync.WaitGroup

    // Khởi động N workers — đây là điểm mấu chốt của pattern
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Worker block ở đây, chờ job từ channel
            // Vòng lặp kết thúc khi jobCh bị đóng
            for jobID := range jobCh {
                result := processJob(jobID)
                resultCh <- result
            }
        }()
    }

    // Đưa tất cả jobs vào queue
    for _, job := range jobs {
        jobCh <- job
    }
    close(jobCh) // Báo cho workers biết không còn job nào nữa

    // Goroutine này chờ tất cả workers xong rồi đóng resultCh
    go func() {
        wg.Wait()
        close(resultCh)
    }()

    // Thu thập kết quả
    var results []Result
    for result := range resultCh {
        results = append(results, result)
    }
    return results
}

func main() {
    jobs := make([]int, 20)
    for i := range jobs {
        jobs[i] = i + 1
    }

    start := time.Now()
    results := workerPool(5, jobs) // 5 workers xử lý 20 jobs
    elapsed := time.Since(start)

    fmt.Printf("Hoàn thành %d jobs trong %v (5 workers)\n", len(results), elapsed)
    // ~400ms thay vì 2000ms nếu chạy tuần tự
}
```

Worker Pool là một trong những pattern quan trọng nhất trong production Go code. Số workers lý tưởng phụ thuộc vào bản chất của công việc: CPU-bound thì dùng `runtime.NumCPU()`, I/O-bound thì có thể cao hơn nhiều.

---

## 3. Semaphore — Giới hạn concurrent operations

`semaphore` từ `golang.org/x/sync/semaphore` là tương đương với `SemaphoreSlim` trong C#. Điểm khác biệt: trong Go, semaphore có *weight* (trọng số), cho phép một operation "chiếm" nhiều slot cùng lúc — hữu ích khi các operations có chi phí khác nhau (vd: file nhỏ chiếm 1 slot, file lớn chiếm 4 slots).

```go
// GOOD: dùng semaphore để giới hạn số concurrent database connections
package main

import (
    "context"
    "fmt"
    "time"

    "golang.org/x/sync/semaphore"
)

const maxConcurrent = 3 // tối đa 3 operations đồng thời

func processWithSemaphore(ctx context.Context, items []string) error {
    // Tạo semaphore với weight = maxConcurrent
    sem := semaphore.NewWeighted(maxConcurrent)

    for _, item := range items {
        item := item

        // Acquire "chiếm" 1 slot — block nếu đã đầy
        // Nếu context bị cancel, Acquire trả về lỗi ngay lập tức
        if err := sem.Acquire(ctx, 1); err != nil {
            return fmt.Errorf("context bị cancel: %w", err)
        }

        go func() {
            defer sem.Release(1) // Luôn giải phóng slot khi xong

            // Mô phỏng công việc
            fmt.Printf("đang xử lý: %s\n", item)
            time.Sleep(500 * time.Millisecond)
            fmt.Printf("xong: %s\n", item)
        }()
    }

    // Chờ tất cả goroutines hoàn thành bằng cách acquire tất cả slots
    if err := sem.Acquire(ctx, maxConcurrent); err != nil {
        return err
    }
    sem.Release(maxConcurrent)
    return nil
}

func main() {
    items := []string{"a", "b", "c", "d", "e", "f"}
    ctx := context.Background()
    if err := processWithSemaphore(ctx, items); err != nil {
        fmt.Println("lỗi:", err)
    }
}
```

Lưu ý: trong ví dụ trên, pattern "acquire all slots để chờ" không phải cách tốt nhất — trong thực tế bạn nên kết hợp với `errgroup` hoặc `WaitGroup`. Semaphore thường được dùng để giới hạn kết nối database, HTTP requests ra ngoài, hoặc file I/O.

---

## 4. Pipeline Pattern — Chuỗi các stages

Pipeline là một trong những pattern đặc trưng nhất của Go. Ý tưởng: dữ liệu chạy qua một chuỗi stages, mỗi stage là một goroutine độc lập. Các stages kết nối với nhau bằng channels. Pattern này tương tự LINQ chaining trong C# nhưng chạy *song song thực sự* — trong khi stage 2 xử lý item 1, stage 1 đã có thể xử lý item 2.

```go
// GOOD: Pipeline pattern — Generator → Transform → Sink
package main

import (
    "fmt"
    "strings"
)

// generator tạo ra stream dữ liệu
// Đây là "nguồn" của pipeline — tương tự IEnumerable trong C#
func generator(done <-chan struct{}, items ...string) <-chan string {
    out := make(chan string)
    go func() {
        defer close(out)
        for _, item := range items {
            select {
            case out <- item:
            case <-done: // Dừng lại nếu nhận được tín hiệu cancel
                return
            }
        }
    }()
    return out
}

// toUpperCase là một transform stage
// Nhận channel đầu vào, trả về channel đầu ra
func toUpperCase(done <-chan struct{}, in <-chan string) <-chan string {
    out := make(chan string)
    go func() {
        defer close(out)
        for item := range in {
            select {
            case out <- strings.ToUpper(item):
            case <-done:
                return
            }
        }
    }()
    return out
}

// addExclamation là transform stage thứ hai
func addExclamation(done <-chan struct{}, in <-chan string) <-chan string {
    out := make(chan string)
    go func() {
        defer close(out)
        for item := range in {
            select {
            case out <- item + "!":
            case <-done:
                return
            }
        }
    }()
    return out
}

func main() {
    // done channel dùng để cancel toàn bộ pipeline
    done := make(chan struct{})
    defer close(done)

    // Kết nối các stages: generator → toUpperCase → addExclamation
    // Giống như: items.Select(x => x.ToUpper()).Select(x => x + "!")
    // Nhưng chạy song song thực sự!
    words := generator(done, "hello", "world", "golang", "pipeline")
    upper := toUpperCase(done, words)
    excited := addExclamation(done, upper)

    // Sink: tiêu thụ kết quả cuối cùng
    for result := range excited {
        fmt.Println(result)
    }
    // Output:
    // HELLO!
    // WORLD!
    // GOLANG!
    // PIPELINE!
}
```

Pipeline pattern thể hiện rõ nhất sức mạnh của channels trong Go. So với LINQ, pipeline chạy thực sự song song: stage 2 không cần đợi stage 1 xử lý hết tất cả items.

---

## 5. Fan-out / Fan-in — Phân phối và thu thập

Fan-out: một input channel được đọc bởi nhiều workers. Fan-in: nhiều output channels được gộp lại thành một. Kết hợp hai patterns này cho phép xử lý song song với throughput tối đa.

```go
// GOOD: Fan-out / Fan-in hoàn chỉnh
package main

import (
    "fmt"
    "math/rand"
    "sync"
    "time"
)

// fanOut phân phối jobs từ một channel đến nhiều workers
// Trả về slice các output channels (một channel per worker)
func fanOut(in <-chan int, numWorkers int) []<-chan int {
    channels := make([]<-chan int, numWorkers)
    for i := 0; i < numWorkers; i++ {
        channels[i] = worker(in, i)
    }
    return channels
}

func worker(in <-chan int, id int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for job := range in {
            // Mô phỏng thời gian xử lý khác nhau cho mỗi worker
            time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
            result := job * job // bình phương số
            fmt.Printf("worker %d: %d → %d\n", id, job, result)
            out <- result
        }
    }()
    return out
}

// fanIn gộp nhiều channels thành một channel duy nhất
func fanIn(channels ...<-chan int) <-chan int {
    var wg sync.WaitGroup
    merged := make(chan int)

    // Mỗi input channel có một goroutine forward data sang merged channel
    output := func(ch <-chan int) {
        defer wg.Done()
        for val := range ch {
            merged <- val
        }
    }

    wg.Add(len(channels))
    for _, ch := range channels {
        go output(ch)
    }

    // Khi tất cả input channels đóng, đóng merged channel
    go func() {
        wg.Wait()
        close(merged)
    }()

    return merged
}

func main() {
    // Tạo input stream
    jobs := make(chan int, 10)
    for i := 1; i <= 10; i++ {
        jobs <- i
    }
    close(jobs)

    // Fan-out: 3 workers đọc từ cùng một jobs channel
    workerChans := fanOut(jobs, 3)

    // Fan-in: gộp kết quả từ 3 workers về một channel
    results := fanIn(workerChans...)

    var total int
    for result := range results {
        total += result
    }
    fmt.Printf("Tổng bình phương 1..10 = %d\n", total) // 385
}
```

---

## 6. Rate Limiting với time.Ticker

Rate limiting là bài toán thường gặp khi gọi API có giới hạn request/giây. Trong C#, bạn có thể dùng `RateLimiter` (.NET 7+) hoặc các thư viện như Polly. Trong Go, `time.Ticker` là building block cơ bản nhất.

```go
// GOOD: Rate limiter xử lý tối đa N items mỗi giây
package main

import (
    "context"
    "fmt"
    "time"
)

// rateLimitedProcessor xử lý items với giới hạn rate
// ratePerSecond: số items được xử lý tối đa mỗi giây
func rateLimitedProcessor(ctx context.Context, items []string, ratePerSecond int) {
    // Ticker "đánh nhịp" mỗi khoảng thời gian đều đặn
    interval := time.Second / time.Duration(ratePerSecond)
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for _, item := range items {
        select {
        case <-ctx.Done():
            fmt.Println("bị cancel, dừng xử lý")
            return
        case <-ticker.C: // Chỉ tiến hành khi ticker "cho phép"
            fmt.Printf("[%s] xử lý: %s\n",
                time.Now().Format("15:04:05.000"), item)
        }
    }
}

func main() {
    items := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    fmt.Println("Bắt đầu xử lý (2 items/giây)...")
    rateLimitedProcessor(ctx, items, 2) // 2 items mỗi giây
}
```

Trong production, bạn thường dùng thư viện như `golang.org/x/time/rate` với `rate.Limiter` — nó hỗ trợ burst (cho phép xử lý nhanh hơn trong thời gian ngắn) và thread-safe hơn.

---

## 7. Timeout Pattern với select

Trong C#, bạn dùng `Task.WhenAny` hoặc `CancellationTokenSource` với timeout. Trong Go, `select` với `time.After` là idiom chuẩn.

```go
// GOOD: Timeout pattern — hủy bỏ nếu quá lâu
package main

import (
    "context"
    "errors"
    "fmt"
    "time"
)

var ErrTimeout = errors.New("thao tác quá thời gian chờ")

// slowOperation mô phỏng một operation chậm (vd: database query)
func slowOperation(ctx context.Context) (string, error) {
    // Mô phỏng: operation mất 2 giây
    select {
    case <-time.After(2 * time.Second):
        return "kết quả", nil
    case <-ctx.Done():
        return "", ctx.Err()
    }
}

func main() {
    // Pattern 1: Dùng context.WithTimeout (cách được khuyến nghị)
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel() // Luôn gọi cancel để giải phóng resources

    result, err := slowOperation(ctx)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("Timeout! Operation quá chậm")
        } else {
            fmt.Println("Lỗi khác:", err)
        }
        return
    }
    fmt.Println("Kết quả:", result)

    // Pattern 2: select với time.After cho ad-hoc timeout
    // Dùng khi bạn không muốn tạo context mới
    done := make(chan string, 1)
    go func() {
        time.Sleep(500 * time.Millisecond)
        done <- "hoàn thành"
    }()

    select {
    case result := <-done:
        fmt.Println("Nhận được:", result)
    case <-time.After(1 * time.Second):
        fmt.Println("Timeout!")
    }
}
```

Quy tắc vàng: luôn dùng `context.WithTimeout` hoặc `context.WithDeadline` thay vì `time.After` khi có thể. Context propagates — khi bạn pass context xuống các function con, chúng đều có thể phản hồi với timeout một cách tự nhiên.

---

## 8. Done Channel Pattern — Cancellation không có context

Đây là pattern cũ hơn, xuất hiện nhiều trong code viết trước Go 1.7 (khi `context` package được giới thiệu). Bạn vẫn sẽ thấy nó trong nhiều open source projects, nên cần hiểu để đọc code.

```go
// GOOD: Done channel pattern — cách cũ để cancel goroutines
package main

import (
    "fmt"
    "time"
)

// longRunningTask chạy liên tục cho đến khi done channel bị đóng
func longRunningTask(done <-chan struct{}) <-chan int {
    results := make(chan int)
    go func() {
        defer close(results)
        i := 0
        for {
            select {
            case <-done:
                fmt.Println("longRunningTask: nhận tín hiệu dừng")
                return
            case results <- i:
                i++
                time.Sleep(200 * time.Millisecond)
            }
        }
    }()
    return results
}

func main() {
    done := make(chan struct{})

    results := longRunningTask(done)

    // Đọc 5 kết quả rồi dừng
    for i := 0; i < 5; i++ {
        fmt.Println("Nhận:", <-results)
    }

    // Đóng done channel để báo cho tất cả goroutines dừng lại
    // Đây là broadcast: TẤT CẢ goroutines đang chờ <-done đều nhận được tín hiệu
    close(done)

    time.Sleep(300 * time.Millisecond) // chờ goroutine dọn dẹp
    fmt.Println("Chương trình kết thúc")
}
```

Trong code mới, hãy ưu tiên dùng `context.Context` thay vì done channel. `context` cung cấp thêm deadline, timeout, và có thể mang giá trị (values). Done channel vẫn hữu ích trong các trường hợp đơn giản hoặc khi bạn cần kiểm soát chính xác *khi nào* signal được gửi.

---

## 9. Mutex vs Channel — Khi nào dùng cái nào?

Đây là câu hỏi khiến nhiều người mới học Go bối rối. Go có cả hai: `sync.Mutex` (giống C#) và channels. Quy tắc ngón tay cái: **mutex cho state, channel cho communication (coordination).**

Nếu bạn đang bảo vệ một cấu trúc dữ liệu chia sẻ (như một counter hoặc cache), dùng mutex. Nếu bạn đang truyền dữ liệu hoặc điều phối giữa các goroutines, dùng channel.

```go
// BAD: dùng channel để bảo vệ state — overcomplicated và chậm hơn
package main

import "fmt"

type Counter_BAD struct {
    ch chan int
    n  int
}

func NewCounter_BAD() *Counter_BAD {
    c := &Counter_BAD{ch: make(chan int)}
    // Dùng goroutine làm "serializer" — không cần thiết!
    go func() {
        for delta := range c.ch {
            c.n += delta
        }
    }()
    return c
}

func (c *Counter_BAD) Inc() { c.ch <- 1 }

func main() {}
```

```go
// GOOD: dùng mutex cho state management, channel cho coordination
package main

import (
    "fmt"
    "sync"
)

// SafeCounter bảo vệ state với mutex — đơn giản và hiệu quả
type SafeCounter struct {
    mu sync.Mutex
    n  int
}

func (c *SafeCounter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.n++
}

func (c *SafeCounter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.n
}

// WorkQueue dùng channel để điều phối — phù hợp cho communication
type WorkQueue struct {
    jobs chan func()
    done chan struct{}
}

func NewWorkQueue() *WorkQueue {
    wq := &WorkQueue{
        jobs: make(chan func(), 100),
        done: make(chan struct{}),
    }
    go wq.run()
    return wq
}

func (wq *WorkQueue) run() {
    for {
        select {
        case job := <-wq.jobs:
            job() // Thực thi job tuần tự — không cần mutex!
        case <-wq.done:
            return
        }
    }
}

func (wq *WorkQueue) Submit(job func()) { wq.jobs <- job }
func (wq *WorkQueue) Stop()            { close(wq.done) }

func main() {
    // Mutex cho state
    counter := &SafeCounter{}
    var wg sync.WaitGroup

    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter.Inc()
        }()
    }
    wg.Wait()
    fmt.Println("Counter:", counter.Value()) // 1000

    // Channel cho coordination
    wq := NewWorkQueue()
    defer wq.Stop()

    results := make(chan string, 3)
    wq.Submit(func() { results <- "job 1 xong" })
    wq.Submit(func() { results <- "job 2 xong" })
    wq.Submit(func() { results <- "job 3 xong" })

    for i := 0; i < 3; i++ {
        fmt.Println(<-results)
    }
}
```

---

## 10. sync.Map — Thread-safe map

`sync.Map` là map được tối ưu cho hai trường hợp cụ thể: (1) entry chỉ được ghi một lần nhưng đọc nhiều lần (ví dụ: cache), và (2) nhiều goroutines đọc/ghi các keys *khác nhau*. Nếu workload của bạn không khớp với hai trường hợp này, dùng `map + sync.RWMutex` thông thường sẽ hiệu quả hơn.

```go
// GOOD: sync.Map cho concurrent cache
package main

import (
    "fmt"
    "sync"
)

// SimpleCache dùng sync.Map để cache kết quả tính toán tốn kém
type SimpleCache struct {
    m sync.Map
}

func (c *SimpleCache) LoadOrCompute(key string, compute func() string) string {
    // Thử load trước — không cần lock!
    if val, ok := c.m.Load(key); ok {
        return val.(string) // Type assertion vì sync.Map lưu interface{}
    }

    // Nếu chưa có, tính toán và lưu
    // LoadOrStore: atomic — chỉ store nếu key chưa tồn tại
    result := compute()
    actual, _ := c.m.LoadOrStore(key, result)
    return actual.(string)
}

func (c *SimpleCache) Delete(key string) {
    c.m.Delete(key)
}

func (c *SimpleCache) PrintAll() {
    c.m.Range(func(key, value interface{}) bool {
        fmt.Printf("  %v = %v\n", key, value)
        return true // trả về false để dừng iteration
    })
}

func main() {
    cache := &SimpleCache{}
    var wg sync.WaitGroup

    // 100 goroutines cùng cố gắng compute cùng key
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            result := cache.LoadOrCompute("expensive-key", func() string {
                // Trong thực tế: query database, gọi API...
                return fmt.Sprintf("computed by goroutine %d", id)
            })
            _ = result
        }(i)
    }

    wg.Wait()
    fmt.Println("Cache contents:")
    cache.PrintAll()
    // Chỉ có 1 entry vì LoadOrStore là atomic
}
```

Nhược điểm của `sync.Map`: vì nó lưu `interface{}`, bạn phải type assert mỗi khi đọc giá trị, và bạn mất type safety tại compile time. Từ Go 1.18, nếu bạn muốn type-safe concurrent map, hãy dùng `map[K]V + sync.RWMutex` với generics.

---

## 11. atomic package — Counters không cần mutex

`sync/atomic` cung cấp các atomic operations cho các kiểu dữ liệu primitive. Đây là tương đương với `Interlocked` class trong C# (`Interlocked.Increment`, `Interlocked.CompareExchange`...). Atomic operations nhanh hơn mutex cho các giá trị đơn giản vì chúng được thực hiện bằng CPU instructions đặc biệt, không cần kernel-level locking.

```go
// GOOD: atomic.Int64 và atomic.Bool cho counters và flags đơn giản
package main

import (
    "fmt"
    "sync"
    "sync/atomic"
    "time"
)

type Metrics struct {
    requestCount  atomic.Int64 // Thay vì int64 + mutex
    errorCount    atomic.Int64
    isShuttingDown atomic.Bool // Thay vì bool + mutex
}

func (m *Metrics) RecordRequest(isError bool) {
    m.requestCount.Add(1) // atomic, thread-safe, không cần lock!
    if isError {
        m.errorCount.Add(1)
    }
}

func (m *Metrics) Shutdown() {
    m.isShuttingDown.Store(true)
}

func (m *Metrics) IsShuttingDown() bool {
    return m.isShuttingDown.Load()
}

func (m *Metrics) Report() {
    total := m.requestCount.Load()
    errors := m.errorCount.Load()
    fmt.Printf("Tổng requests: %d, Lỗi: %d (%.1f%%)\n",
        total, errors, float64(errors)/float64(total)*100)
}

func main() {
    metrics := &Metrics{}
    var wg sync.WaitGroup

    // Mô phỏng 1000 requests đồng thời
    for i := 0; i < 1000; i++ {
        wg.Add(1)
        i := i
        go func() {
            defer wg.Done()
            if metrics.IsShuttingDown() {
                return
            }
            isError := i%10 == 0 // 10% requests bị lỗi
            metrics.RecordRequest(isError)
        }()
    }

    // Sau 100ms, bắt đầu shutdown
    go func() {
        time.Sleep(100 * time.Millisecond)
        metrics.Shutdown()
        fmt.Println("Đang shutdown...")
    }()

    wg.Wait()
    metrics.Report()
}
```

Dùng `atomic` khi bạn chỉ cần thao tác với **một giá trị đơn**: increment counter, set flag, CAS (Compare-And-Swap). Khi bạn cần thao tác với **nhiều giá trị cùng lúc** (vd: update 2 fields cùng lúc một cách atomic), hãy dùng mutex.

---

## 12. Detecting Goroutine Leaks với goleak

Goroutine leak là một trong những vấn đề nguy hiểm nhất trong Go — một goroutine bị kẹt sẽ tồn tại mãi mãi, leak memory và resources. Thư viện `go.uber.org/goleak` giúp phát hiện leak trong tests.

```go
// GOOD: Test phát hiện goroutine leak
// File: leak_test.go
package main

import (
    "testing"
    "time"

    "go.uber.org/goleak"
)

// leakyFunction tạo goroutine sẽ bị leak nếu không có cách cancel
func leakyFunction() {
    // BAD: goroutine này chạy mãi mãi, không có cách dừng!
    go func() {
        for {
            time.Sleep(time.Hour)
        }
    }()
}

// safeFunction tạo goroutine có thể dừng được
func safeFunction(done <-chan struct{}) {
    go func() {
        select {
        case <-done:
            return // Goroutine kết thúc đúng cách
        case <-time.After(time.Hour):
            return
        }
    }()
}

func TestNoLeak(t *testing.T) {
    defer goleak.VerifyNone(t) // Kiểm tra goroutine leak sau khi test kết thúc

    done := make(chan struct{})
    safeFunction(done)
    close(done) // Dừng goroutine trước khi test kết thúc
}

// TestWithLeak sẽ FAIL vì có goroutine leak
// func TestWithLeak(t *testing.T) {
//     defer goleak.VerifyNone(t)
//     leakyFunction() // Test này sẽ fail!
// }
```

`goleak` kiểm tra số lượng goroutines trước và sau test. Nếu số lượng tăng, test sẽ fail và báo cáo stack trace của goroutine bị leak. Tích hợp goleak vào CI/CD của bạn là một thực hành tốt trong production services.

---

## 13. Common Mistakes — Lỗi thường gặp

### Lỗi 1: Loop variable capture

Đây là bug kinh điển nhất trong Go concurrency, tương tự closure-over-loop-variable trong C# và JavaScript.

```go
// BAD: tất cả goroutines đều dùng cùng một biến i
package main

import (
    "fmt"
    "sync"
)

func buggyLoop() {
    var wg sync.WaitGroup
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            fmt.Println(i) // BUG: i đã thay đổi trước khi goroutine chạy!
            // Thường in ra: 5 5 5 5 5
        }()
    }
    wg.Wait()
}

func main() { buggyLoop() }
```

```go
// GOOD: tạo bản copy của biến loop trước khi dùng trong goroutine
package main

import (
    "fmt"
    "sync"
)

func fixedLoop() {
    var wg sync.WaitGroup
    for i := 0; i < 5; i++ {
        i := i // Tạo biến mới trong scope của iteration này
        wg.Add(1)
        go func() {
            defer wg.Done()
            fmt.Println(i) // Đúng: mỗi goroutine có bản copy riêng của i
        }()
    }
    wg.Wait()
}

// Hoặc: pass như argument (rõ ràng hơn)
func fixedLoop2() {
    var wg sync.WaitGroup
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(n int) { // n là parameter riêng, không phải closure
            defer wg.Done()
            fmt.Println(n) // Đúng
        }(i) // Pass i như argument ngay lúc này
    }
    wg.Wait()
}

func main() {
    fixedLoop()
    fixedLoop2()
}
```

**Lưu ý:** Từ Go 1.22, compiler đã thay đổi ngữ nghĩa của loop variables — mỗi iteration tạo ra một biến mới. Tuy nhiên, nếu code của bạn cần tương thích với Go 1.21 trở xuống, bạn vẫn phải viết `i := i`.

### Lỗi 2: Gửi vào closed channel

```go
// BAD: panic khi gửi vào channel đã đóng
package main

func main() {
    ch := make(chan int)
    close(ch)
    ch <- 1 // PANIC: send on closed channel
}
```

```go
// GOOD: chỉ người gửi mới đóng channel, và chỉ khi chắc chắn không gửi nữa
package main

import "fmt"

func producer() <-chan int {
    ch := make(chan int)
    go func() {
        defer close(ch) // Đóng khi goroutine kết thúc — safe
        for i := 0; i < 5; i++ {
            ch <- i
        }
    }()
    return ch // Consumer nhận channel read-only, không thể đóng
}

func main() {
    for v := range producer() {
        fmt.Println(v)
    }
}
```

---

## Bảng: Các vấn đề phổ biến và cách xử lý

| Vấn đề | Triệu chứng | Nguyên nhân | Giải pháp |
|--------|-------------|-------------|-----------|
| Goroutine leak | Memory tăng dần theo thời gian | Goroutine bị block mãi mãi không có cách thoát | Luôn có done channel hoặc context; dùng goleak trong tests |
| Race condition | Output không nhất quán, data bị corrupt | Nhiều goroutines đọc/ghi shared state không có sync | `sync.Mutex`, `sync/atomic`, hoặc channel-based design |
| Deadlock | Chương trình treo hoàn toàn | Circular dependency giữa channels hoặc mutexes | Thiết kế channel direction rõ ràng; tránh lock trong lock |
| Loop variable bug | Goroutines in ra cùng một giá trị cuối cùng | Closure capture biến loop thay vì giá trị | `i := i` hoặc pass như argument |
| Panic: send on closed channel | Runtime panic | Sender không kiểm soát được lifecycle | Chỉ người tạo channel mới đóng; dùng `sync.Once` nếu cần |
| Panic: nil channel operation | Runtime panic | Đọc/ghi nil channel (block mãi), close nil channel (panic) | Khởi tạo channel với `make` trước khi dùng |
| Data race | Crash hoặc incorrect behavior | Không phát hiện được không có `-race` flag | Chạy tests với `go test -race ./...` |
| Thundering herd | Spike CPU/memory khi restart | Nhiều goroutines cùng gửi request một lúc | Worker pool, rate limiter, `singleflight` pattern |

---

## Best Practices

**Luôn chạy tests với `-race` flag.** Go có built-in race detector: `go test -race ./...`. Race condition là loại bug rất khó reproduce và debug, nhưng race detector có thể phát hiện chúng tại thời điểm chạy. Tích hợp vào CI/CD pipeline của bạn.

**Nghĩ về ownership của channel.** Ai tạo channel thì người đó đóng channel. Consumer không bao giờ đóng channel. Nguyên tắc này giúp tránh panic và làm code dễ đọc hơn. Khi define function signature, dùng `chan<-` (write-only) và `<-chan` (read-only) để communicate ownership.

**Ưu tiên context.Context over done channel.** `context.Context` là cách tiêu chuẩn hiện đại để propagate cancellation và deadline. Mọi function có thể block nên nhận `ctx context.Context` làm tham số đầu tiên. Đây là convention của Go, không phải quy tắc cứng nhắc, nhưng vi phạm nó sẽ làm code của bạn khó tích hợp với ecosystem.

**Đừng tối ưu sớm với goroutines.** Spawning goroutine có chi phí thấp (~2KB stack ban đầu), nhưng nếu bạn spawn quá nhiều, scheduler sẽ bị quá tải. Hãy benchmark trước khi quyết định kiến trúc. Worker pool với số workers cố định thường tốt hơn "một goroutine per task" cho workloads lớn.

**Sử dụng `errgroup` thay vì tự manage WaitGroup + error.** Đây là lỗi mà ngay cả developer có kinh nghiệm cũng hay mắc phải. `errgroup` đã xử lý đúng tất cả các edge cases, trong khi tự manage thường dẫn đến race condition tinh vi.

---

## Bài tập thực hành

### Bài tập 1: URL Health Checker với Worker Pool và errgroup

Viết chương trình kiểm tra "health" của một danh sách URLs. Yêu cầu: tối đa 5 requests đồng thời, timeout 3 giây mỗi URL, báo cáo kết quả đầy đủ kể cả khi có lỗi.

**Giải pháp:**

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "sync"
    "time"
)

type HealthResult struct {
    URL        string
    StatusCode int
    Latency    time.Duration
    Err        error
}

func checkURL(ctx context.Context, url string) HealthResult {
    start := time.Now()
    result := HealthResult{URL: url}

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        result.Err = err
        return result
    }

    client := &http.Client{} // timeout đến từ context
    resp, err := client.Do(req)
    result.Latency = time.Since(start)

    if err != nil {
        result.Err = err
        return result
    }
    defer resp.Body.Close()
    result.StatusCode = resp.StatusCode
    return result
}

func healthCheck(urls []string, maxConcurrent int, timeout time.Duration) []HealthResult {
    jobs := make(chan string, len(urls))
    for _, url := range urls {
        jobs <- url
    }
    close(jobs)

    results := make([]HealthResult, 0, len(urls))
    var mu sync.Mutex
    var wg sync.WaitGroup

    for i := 0; i < maxConcurrent; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for url := range jobs {
                // Mỗi request có timeout riêng
                ctx, cancel := context.WithTimeout(context.Background(), timeout)
                result := checkURL(ctx, url)
                cancel()

                mu.Lock()
                results = append(results, result)
                mu.Unlock()
            }
        }()
    }

    wg.Wait()
    return results
}

func main() {
    urls := []string{
        "https://httpbin.org/status/200",
        "https://httpbin.org/status/404",
        "https://httpbin.org/status/500",
        "https://httpbin.org/delay/5", // Sẽ timeout
        "https://httpbin.org/get",
    }

    fmt.Printf("Kiểm tra %d URLs (tối đa 5 concurrent, timeout 3s)...\n\n", len(urls))
    start := time.Now()
    results := healthCheck(urls, 5, 3*time.Second)
    elapsed := time.Since(start)

    for _, r := range results {
        if r.Err != nil {
            fmt.Printf("❌ %s — LỖI: %v\n", r.URL, r.Err)
        } else {
            status := "✅"
            if r.StatusCode >= 400 {
                status = "⚠️"
            }
            fmt.Printf("%s %s — %d (%v)\n", status, r.URL, r.StatusCode, r.Latency.Round(time.Millisecond))
        }
    }
    fmt.Printf("\nTổng thời gian: %v\n", elapsed.Round(time.Millisecond))
}
```

### Bài tập 2: Pipeline xử lý log file

Viết pipeline đọc log entries, lọc chỉ giữ lại errors, đếm theo loại lỗi, và in báo cáo. Mỗi stage chạy trong goroutine riêng.

**Giải pháp:**

```go
package main

import (
    "fmt"
    "math/rand"
    "strings"
    "sync"
    "time"
)

type LogEntry struct {
    Level   string
    Message string
    Time    time.Time
}

// Stage 1: Generator — tạo ra log entries (mô phỏng đọc file)
func generateLogs(done <-chan struct{}) <-chan LogEntry {
    out := make(chan LogEntry)
    levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "ERROR", "INFO"}
    messages := []string{
        "database connection failed",
        "user logged in",
        "cache miss",
        "null pointer exception",
        "timeout exceeded",
        "request processed",
    }

    go func() {
        defer close(out)
        for i := 0; i < 50; i++ {
            entry := LogEntry{
                Level:   levels[rand.Intn(len(levels))],
                Message: messages[rand.Intn(len(messages))],
                Time:    time.Now(),
            }
            select {
            case out <- entry:
            case <-done:
                return
            }
        }
    }()
    return out
}

// Stage 2: Filter — chỉ giữ lại ERROR entries
func filterErrors(done <-chan struct{}, in <-chan LogEntry) <-chan LogEntry {
    out := make(chan LogEntry)
    go func() {
        defer close(out)
        for entry := range in {
            if entry.Level == "ERROR" {
                select {
                case out <- entry:
                case <-done:
                    return
                }
            }
        }
    }()
    return out
}

// Stage 3: Enrich — thêm thông tin (mô phỏng lookup)
func enrichEntries(done <-chan struct{}, in <-chan LogEntry) <-chan LogEntry {
    out := make(chan LogEntry)
    go func() {
        defer close(out)
        for entry := range in {
            // Mô phỏng lookup thêm context
            entry.Message = strings.ToUpper(entry.Message[:1]) + entry.Message[1:]
            select {
            case out <- entry:
            case <-done:
                return
            }
        }
    }()
    return out
}

// Stage 4: Sink — thu thập kết quả và tạo báo cáo
func aggregateResults(done <-chan struct{}, in <-chan LogEntry) map[string]int {
    counts := make(map[string]int)
    var mu sync.Mutex

    for entry := range in {
        select {
        case <-done:
            return counts
        default:
            mu.Lock()
            counts[entry.Message]++
            mu.Unlock()
        }
    }
    return counts
}

func main() {
    rand.Seed(time.Now().UnixNano())

    done := make(chan struct{})
    defer close(done)

    // Kết nối pipeline
    logs := generateLogs(done)
    errors := filterErrors(done, logs)
    enriched := enrichEntries(done, errors)
    report := aggregateResults(done, enriched)

    fmt.Println("=== Báo cáo lỗi ===")
    total := 0
    for msg, count := range report {
        fmt.Printf("  [%d] %s\n", count, msg)
        total += count
    }
    fmt.Printf("\nTổng số lỗi: %d\n", total)
}
```

---

## Tổng kết

Go tiếp cận concurrency không phải bằng cách ẩn nó đi (như C# async/await) mà bằng cách làm nó *tường minh và có thể kiểm soát*. Sau khi nắm vững các patterns trong bài này, bạn sẽ có thể:

- Thay thế `Task.WhenAll` bằng `errgroup` với cancellation đúng cách
- Kiểm soát resource usage với worker pool và semaphore
- Xây dựng data processing pipelines hiệu quả
- Chọn đúng primitive (mutex vs channel vs atomic) cho từng tình huống
- Phát hiện và fix goroutine leaks trước khi lên production

---

**Next:** [16. Logging & Config](16-logging-config.md) — Structured logging với `log/slog`, quản lý config qua env var/file, so với `ILogger`/`IConfiguration` trong .NET.
