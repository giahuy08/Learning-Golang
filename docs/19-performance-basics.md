# 19 — Performance Basics trong Go

> Dành cho lập trình viên C# đang học Go. Bài này giải thích **tại sao** Go làm vậy, không chỉ **làm thế nào**.

---

## 1. Triết lý hiệu năng của Go — Nhanh từ đầu, tối ưu sau khi đo

Nếu bạn đến từ C#, bạn có thể quen với việc dùng `async/await` để tránh blocking, hay cài NuGet như `BenchmarkDotNet` để đo hiệu năng. Go có cách tiếp cận khác: **ngôn ngữ tự nó đã nhanh theo mặc định**, và toolchain tích hợp sẵn công cụ đo lường mà không cần cài thêm gì.

Go biên dịch thẳng ra native binary (không có JIT như .NET CLR, không có managed runtime như Java). Runtime của Go rất nhỏ gọn — chỉ bao gồm goroutine scheduler, garbage collector, và một số runtime support. Điều này có nghĩa là một chương trình Go thường khởi động trong vài millisecond và dùng RAM ít hơn .NET đáng kể.

Tuy nhiên, triết lý quan trọng nhất cần hiểu là: **đừng tối ưu khi chưa đo**. Donald Knuth đã nói "premature optimization is the root of all evil" — điều này đúng hơn bao giờ hết với Go. Go khuyến khích bạn viết code rõ ràng trước, sau đó dùng `pprof` và `go test -bench` để tìm **đúng chỗ** cần tối ưu. Tối ưu sai chỗ không chỉ vô nghĩa mà còn làm code khó đọc hơn mà không gain được gì.

**Quy trình tối ưu đúng đắn:**

1. Viết code rõ ràng, đúng đắn trước
2. Profiling để tìm bottleneck thực sự
3. Benchmark để đo trước và sau khi tối ưu
4. Chỉ tối ưu nếu số liệu chứng minh cần thiết

---

## 2. Stack vs Heap Allocation — Go quyết định, không phải bạn

Trong C#, bạn hiểu rõ quy tắc: `struct` nằm trên stack (nếu là local variable), `class` nằm trên heap. Bạn kiểm soát điều này qua việc chọn `struct` hay `class`.

Trong Go, quy tắc khác hơn: **escape analysis** của compiler quyết định một biến nằm ở đâu, không phải bạn. Cả `int`, `struct`, hay `*MyStruct` đều có thể nằm trên stack hoặc heap tùy vào cách bạn dùng nó. Stack allocation nhanh hơn và không tạo áp lực lên GC. Heap allocation chậm hơn và khi đủ nhiều sẽ trigger GC pause.

Quy tắc cơ bản: nếu compiler thấy một biến "thoát" ra khỏi scope của function — ví dụ bạn trả về pointer của nó, hay truyền nó vào interface — biến đó sẽ bị "escape to heap". Nếu biến chỉ sống trong function và không thoát ra ngoài, nó nằm trên stack.

**Xem escape analysis bằng lệnh:**

```bash
go build -gcflags="-m" ./...
# Hoặc chi tiết hơn:
go build -gcflags="-m -m" ./...
```

Output sẽ cho bạn thấy dòng nào bị escape:

```
./main.go:12:6: moved to heap: user
./main.go:20:14: &Config{} escapes to heap
```

```go
// BAD — returning pointer causes heap allocation
// Trả về pointer khiến biến bị escape to heap
package main

import "fmt"

type User struct {
    Name string
    Age  int
}

// Hàm này tạo User trên HEAP vì trả về pointer
// C# developer quen với điều này vì object C# luôn trên heap
func newUserBad() *User {
    u := User{Name: "Alice", Age: 30} // u escapes to heap
    return &u                          // trả về địa chỉ => escape
}

func main() {
    u := newUserBad()
    fmt.Println(u.Name)
}
```

```go
// GOOD — nếu caller không cần pointer, trả về value
// Trả về value giữ biến trên stack, nhanh hơn
package main

import "fmt"

type User struct {
    Name string
    Age  int
}

// User được tạo trên STACK, không tạo áp lực GC
func newUserGood() User {
    return User{Name: "Alice", Age: 30} // nằm trên stack
}

func main() {
    u := newUserGood() // copy value, vẫn trên stack
    fmt.Println(u.Name)
}
```

Không phải lúc nào cũng tránh được heap allocation — đôi khi bạn thực sự cần pointer (chia sẻ object giữa goroutine, tránh copy struct lớn, lưu vào slice). Điểm quan trọng là **hiểu khi nào điều đó xảy ra** và không tạo heap allocation không cần thiết trong hot path.

---

## 3. Benchmarks — Đo trước khi kết luận

Go tích hợp sẵn benchmark framework trong package `testing`. Không cần BenchmarkDotNet như C#. Chỉ cần viết function có chữ ký `func BenchmarkXxx(b *testing.B)` trong file `_test.go`.

`b.N` là số lần Go sẽ chạy đoạn code của bạn — Go tự động điều chỉnh con số này để đạt được thời gian đo đủ chính xác (thường ít nhất 1 giây). Bạn **phải** loop `b.N` lần, không được hardcode số khác.

`b.ResetTimer()` dùng để bỏ qua thời gian setup — ví dụ nếu bạn cần tạo dữ liệu test trước khi chạy benchmark thực sự.

`b.ReportAllocs()` yêu cầu Go báo cáo số lần allocation và bytes dùng mỗi operation — thông tin này quan trọng hơn thời gian trong nhiều trường hợp.

```go
// File: benchmark_test.go
package main

import (
    "strings"
    "testing"
)

// Benchmark nối chuỗi bằng += (BAD)
func BenchmarkStringConcat(b *testing.B) {
    b.ReportAllocs() // bật báo cáo allocation
    for i := 0; i < b.N; i++ {
        s := ""
        for j := 0; j < 100; j++ {
            s += "hello" // mỗi lần tạo string mới trên heap
        }
        _ = s // dùng biến để tránh compiler optimize away
    }
}

// Benchmark nối chuỗi bằng strings.Builder (GOOD)
func BenchmarkStringBuilder(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        var sb strings.Builder
        sb.Grow(500) // pre-allocate
        for j := 0; j < 100; j++ {
            sb.WriteString("hello")
        }
        _ = sb.String()
    }
}
```

**Chạy benchmark:**

```bash
go test -bench=. -benchmem
# -bench=.    : chạy tất cả benchmark (regex match all)
# -benchmem   : hiện số allocation và bytes
# -count=5    : chạy mỗi benchmark 5 lần để ổn định kết quả
# -benchtime=3s : chạy ít nhất 3 giây mỗi benchmark
```

**Kết quả mẫu:**

```
BenchmarkStringConcat-8     45234    26543 ns/op    53424 B/op    99 allocs/op
BenchmarkStringBuilder-8   412876     2891 ns/op      512 B/op     1 allocs/op
```

Đọc kết quả: `45234` là số lần chạy, `26543 ns/op` là nanosecond mỗi operation, `53424 B/op` là bytes allocated mỗi op, `99 allocs/op` là số lần gọi allocator. `strings.Builder` nhanh hơn ~9x và ít allocation hơn 99x.

**Lưu ý quan trọng khi viết benchmark:**

```go
// BAD — compiler có thể optimize away toàn bộ computation
// vì kết quả không được dùng
func BenchmarkBad(b *testing.B) {
    for i := 0; i < b.N; i++ {
        compute() // compiler thấy kết quả bỏ đi, có thể xóa luôn
    }
}

// GOOD — gán kết quả ra package-level variable
// để compiler không thể optimize away
var result int

func BenchmarkGood(b *testing.B) {
    var r int
    for i := 0; i < b.N; i++ {
        r = compute() // compiler không dám xóa vì r được dùng
    }
    result = r // gán ra ngoài để đảm bảo không bị eliminate
}
```

---

## 4. Pre-allocate Slices — Tránh reallocation

Trong C#, `List<T>` tự động resize khi đầy — mỗi lần resize nó nhân đôi capacity và copy toàn bộ data sang vùng nhớ mới. Go's slice hoạt động y hệt. Vấn đề là khi bạn biết trước số phần tử, những lần resize này hoàn toàn có thể tránh được.

`make([]T, 0, n)` tạo slice với length 0 nhưng capacity n — Go cấp phát đủ bộ nhớ ngay từ đầu, không cần resize sau. Tương đương `new List<T>(capacity)` trong C#.

```go
// BAD — không pre-allocate, slice phải resize nhiều lần
// Giống new List<int>() không truyền capacity trong C#
func processItemsBad(ids []int) []string {
    result := []string{} // capacity = 0
    for _, id := range ids {
        result = append(result, fmt.Sprintf("item-%d", id))
        // Mỗi khi đầy: Go cấp phát mảng mới, copy toàn bộ, bỏ mảng cũ
        // Với 1000 phần tử: ~10 lần reallocation
    }
    return result
}

// GOOD — pre-allocate khi biết size
// Giống new List<int>(ids.Count) trong C#
func processItemsGood(ids []int) []string {
    result := make([]string, 0, len(ids)) // capacity = len(ids)
    for _, id := range ids {
        result = append(result, fmt.Sprintf("item-%d", id))
        // Không bao giờ cần resize vì capacity đã đủ
    }
    return result
}
```

**Benchmark so sánh:**

```go
// File: slice_bench_test.go
package main

import (
    "fmt"
    "testing"
)

func BenchmarkSliceNoPrealloc(b *testing.B) {
    b.ReportAllocs()
    ids := make([]int, 1000)
    for i := range ids {
        ids[i] = i
    }
    b.ResetTimer() // bỏ qua thời gian setup data
    for i := 0; i < b.N; i++ {
        result := []string{}
        for _, id := range ids {
            result = append(result, fmt.Sprintf("item-%d", id))
        }
        _ = result
    }
}

func BenchmarkSlicePrealloc(b *testing.B) {
    b.ReportAllocs()
    ids := make([]int, 1000)
    for i := range ids {
        ids[i] = i
    }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        result := make([]string, 0, len(ids))
        for _, id := range ids {
            result = append(result, fmt.Sprintf("item-%d", id))
        }
        _ = result
    }
}
```

Kết quả điển hình: version pre-allocate nhanh hơn ~30-40% và có ít hơn ~10x số allocation.

---

## 5. strings.Builder — Nối chuỗi hiệu quả

Trong C#, bạn biết không được dùng `+` trong vòng lặp — phải dùng `StringBuilder`. Go có rule tương tự nhưng lý do sâu hơn: string trong Go **immutable** (giống C#). Mỗi lần `+` tạo ra một string mới hoàn toàn trên heap, string cũ trở thành garbage. Trong loop 1000 lần, bạn tạo ra 1000 string objects thừa, tổng cộng có thể là hàng MB garbage cho GC dọn.

`strings.Builder` hoạt động bằng cách giữ một `[]byte` nội bộ và chỉ convert sang string một lần khi bạn gọi `.String()`. Tương đương `System.Text.StringBuilder` trong C#.

```go
// BAD — dùng += trong loop
// Tương đương: string s = ""; for(...) s += "text"; trong C#
package main

import "fmt"

func buildQueryBad(fields []string) string {
    result := "SELECT "
    for i, f := range fields {
        result += f // mỗi lần tạo string mới, allocate lại toàn bộ
        if i < len(fields)-1 {
            result += ", "
        }
    }
    result += " FROM users"
    return result
}

func main() {
    fields := []string{"id", "name", "email", "created_at"}
    fmt.Println(buildQueryBad(fields))
}
```

```go
// GOOD — dùng strings.Builder
// Tương đương StringBuilder trong C#
package main

import (
    "fmt"
    "strings"
)

func buildQueryGood(fields []string) string {
    var sb strings.Builder
    // Ước lượng size để tránh internal resize
    sb.Grow(len("SELECT  FROM users") + len(fields)*15)

    sb.WriteString("SELECT ")
    for i, f := range fields {
        sb.WriteString(f)
        if i < len(fields)-1 {
            sb.WriteString(", ")
        }
    }
    sb.WriteString(" FROM users")
    return sb.String() // chỉ allocate string một lần duy nhất
}

func main() {
    fields := []string{"id", "name", "email", "created_at"}
    fmt.Println(buildQueryGood(fields))
}
```

`strings.Builder` không thể Reset để tái sử dụng nội bộ buffer (gọi `Reset()` thì xóa content nhưng có thể giữ lại capacity). Nếu bạn cần tái sử dụng buffer giữa các request (giảm allocation hoàn toàn), hãy dùng `bytes.Buffer` với `sync.Pool`.

---

## 6. bytes.Buffer vs strings.Builder — Khi nào dùng cái nào

Cả hai đều dùng để build chuỗi/bytes hiệu quả, nhưng có mục đích khác nhau.

`strings.Builder` chỉ có một mục đích: xây dựng `string`. Nó không implement `io.Reader`, không dùng được với API cần `io.Writer` (trừ `WriteString`). API đơn giản, hiệu năng tốt cho use case đó.

`bytes.Buffer` linh hoạt hơn: nó implement cả `io.Reader` và `io.Writer`. Bạn có thể dùng nó để đọc, ghi, truyền vào `json.Encoder`, `http.ResponseWriter`, hay bất kỳ function nào nhận `io.Writer`. Nó cũng có thể `Reset()` để tái sử dụng với `sync.Pool`.

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "strings"
)

// Dùng strings.Builder khi: chỉ cần build string cuối cùng
func example_stringsBuilder() string {
    var sb strings.Builder
    sb.WriteString("Hello, ")
    sb.WriteString("World!")
    return sb.String()
}

// Dùng bytes.Buffer khi: cần io.Writer hoặc tái sử dụng buffer
func example_bytesBuffer() {
    var buf bytes.Buffer

    // bytes.Buffer implement io.Writer — dùng được với json.Encoder
    encoder := json.NewEncoder(&buf)
    encoder.Encode(map[string]string{"key": "value"})

    fmt.Println(buf.String()) // {"key":"value"}

    // Reset để tái sử dụng buffer (không cần allocate lại)
    buf.Reset()
    buf.WriteString("second use")
    fmt.Println(buf.String())
}

func main() {
    fmt.Println(example_stringsBuilder())
    example_bytesBuffer()
}
```

**Quy tắc ngắn gọn:** dùng `strings.Builder` khi chỉ cần build string; dùng `bytes.Buffer` khi cần tương tác với io interface hoặc kết hợp với `sync.Pool`.

---

## 7. sync.Pool — Tái sử dụng object để giảm GC

Trong .NET, GC tự quản lý memory và bạn hiếm khi cần nghĩ đến việc tái sử dụng object. Trong Go, GC cũng hoạt động tự động, nhưng khi bạn tạo hàng nghìn object tạm thời mỗi giây (ví dụ trong HTTP handler), GC phải làm việc nhiều hơn, dẫn đến latency spike.

`sync.Pool` giải quyết vấn đề này bằng cách lưu trữ các object tạm thời để tái sử dụng thay vì allocate mới mỗi lần. Quan trọng: Pool **không** phải là persistent cache — GC có thể xóa các object trong Pool bất cứ lúc nào (đặc biệt sau mỗi GC cycle). Pool phù hợp cho **temporary buffers**, không phải long-lived objects.

```go
// BAD — mỗi request tạo buffer mới, GC phải dọn
// Trong production với 1000 req/s, đây là 1000 allocation/s
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func handlerBad(w http.ResponseWriter, r *http.Request) {
    // Mỗi request tạo một buffer mới trên heap
    var buf bytes.Buffer // allocate mỗi lần
    json.NewEncoder(&buf).Encode(map[string]string{"status": "ok"})
    w.Write(buf.Bytes())
}
```

```go
// GOOD — dùng sync.Pool để tái sử dụng buffer
// Pool giữ buffer đã dùng xong, request tiếp theo lấy lại dùng
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "sync"
)

var bufPool = sync.Pool{
    // New được gọi khi Pool không có object nào sẵn
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func handlerGood(w http.ResponseWriter, r *http.Request) {
    // Lấy buffer từ Pool (có thể là buffer cũ đã được reset)
    buf := bufPool.Get().(*bytes.Buffer)
    buf.Reset() // xóa content cũ, giữ lại capacity

    // Sử dụng buffer
    json.NewEncoder(buf).Encode(map[string]string{"status": "ok"})
    w.Write(buf.Bytes())

    // Trả buffer về Pool để request khác tái sử dụng
    // QUAN TRỌNG: đừng giữ reference đến buf sau khi Put
    bufPool.Put(buf)
}

func main() {
    http.HandleFunc("/", handlerGood)
    http.ListenAndServe(":8080", nil)
}
```

`sync.Pool` phù hợp nhất khi: object expensive để tạo (như buffer lớn, connection object), object dùng xong thì không cần nữa, và workload tạo ra nhiều object tạm thời. Không phải mọi struct đều cần Pool — chỉ dùng khi profiling cho thấy allocation là bottleneck.

---

## 8. Struct Field Alignment — Thứ tự field ảnh hưởng đến memory

Đây là khái niệm ít người biết nhưng quan trọng trong low-level optimization. CPU đọc memory hiệu quả nhất khi data được "align" — tức là địa chỉ bộ nhớ phải là bội số của size của kiểu dữ liệu đó. Để đảm bảo alignment, compiler chèn "padding" bytes giữa các field.

Nếu bạn đặt field theo thứ tự ngẫu nhiên, padding có thể chiếm nhiều space không cần thiết. Quy tắc đơn giản: **sắp xếp field từ lớn nhất đến nhỏ nhất** (64-bit, 32-bit, 16-bit, 8-bit, rồi đến bool và byte).

```go
// BAD — thứ tự field tạo ra nhiều padding
// Khai báo theo thứ tự ngẫu nhiên, không quan tâm alignment
package main

import (
    "fmt"
    "unsafe"
)

type UserBad struct {
    Active    bool    // 1 byte + 7 bytes padding (để align Age)
    Age       int64   // 8 bytes
    Verified  bool    // 1 byte + 3 bytes padding (để align Score)
    Score     int32   // 4 bytes
    // Tổng: 1+7+8+1+3+4 = 24 bytes
}

// GOOD — field lớn trước, nhỏ sau, tối thiểu padding
// Sắp xếp: int64 > int32 > bool
type UserGood struct {
    Age      int64 // 8 bytes
    Score    int32 // 4 bytes
    Active   bool  // 1 byte
    Verified bool  // 1 byte + 2 bytes padding (align toàn struct về 8)
    // Tổng: 8+4+1+1+2 = 16 bytes
}

func main() {
    fmt.Printf("UserBad size:  %d bytes\n", unsafe.Sizeof(UserBad{}))
    fmt.Printf("UserGood size: %d bytes\n", unsafe.Sizeof(UserGood{}))
    // Output:
    // UserBad size:  24 bytes
    // UserGood size: 16 bytes
    // Tiết kiệm 8 bytes mỗi struct — quan trọng khi có hàng triệu instance
}
```

Tiết kiệm 8 bytes nghe nhỏ, nhưng nếu bạn có slice chứa 1 triệu UserBad trong một cache service, đó là 8MB lãng phí. Hơn nữa, struct nhỏ hơn nghĩa là nhiều struct hơn fit vào CPU cache line, tăng tốc traversal.

Công cụ `fieldalignment` trong `golang.org/x/tools` có thể tự động báo cáo và fix struct alignment cho cả codebase:

```bash
go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
fieldalignment -fix ./...
```

---

## 9. Tránh Interface trong Hot Path

Trong C#, interface là nền tảng của OOP và DI. Trong Go, interface cũng được dùng nhiều, nhưng có một chi phí ẩn quan trọng: **interface dispatch**.

Khi bạn gọi method qua interface, Go phải thực hiện hai bước: tra cứu trong itab (interface table) để tìm đúng implementation, rồi mới gọi function. Đây là indirect call, không phải direct call. Thêm vào đó, interface boxing khiến value phải escape to heap.

Trong code bình thường, overhead này không đáng kể. Nhưng trong **hot path** — tức là đoạn code chạy hàng triệu lần trong vòng lặp — nó có thể chiếm 10-30% runtime.

```go
// BAD — gọi interface method trong hot loop
// Mỗi lần gọi tốn indirect dispatch overhead
package main

import "fmt"

type Processor interface {
    Process(data []byte) int
}

type FastProcessor struct{}

func (p *FastProcessor) Process(data []byte) int {
    // compute something
    return len(data)
}

func processAllBad(p Processor, items [][]byte) int {
    total := 0
    for _, item := range items {
        total += p.Process(item) // interface dispatch mỗi iteration
    }
    return total
}

// GOOD — dùng concrete type trong hot loop
// Compiler biết chính xác function nào cần gọi, có thể inline
func processAllGood(p *FastProcessor, items [][]byte) int {
    total := 0
    for _, item := range items {
        total += p.Process(item) // direct call, có thể được inline
    }
    return total
}

func main() {
    p := &FastProcessor{}
    items := make([][]byte, 1000)
    for i := range items {
        items[i] = []byte("hello world")
    }
    fmt.Println(processAllGood(p, items))
}
```

Pattern thực tế: nhận interface ở layer trên (để dễ test và mock), nhưng trong inner loop, type assert ra concrete type nếu hiệu năng quan trọng. Tuy nhiên, chỉ làm điều này khi profiling đã xác nhận interface là bottleneck.

---

## 10. Map Pre-sizing — Tránh rehashing

Giống slice, Go's map tự động resize khi load factor vượt ngưỡng (~6.5 entry/bucket). Mỗi lần resize phải rehash toàn bộ entry — expensive operation. Nếu bạn biết trước số entry, truyền hint vào `make`:

```go
// BAD — map không có size hint, phải rehash nhiều lần
func buildIndexBad(users []User) map[int]User {
    index := make(map[int]User) // capacity = 0
    for _, u := range users {
        index[u.ID] = u // có thể trigger rehash
    }
    return index
}

// GOOD — pre-size map với số entry mong đợi
// Tương đương new Dictionary<int, User>(users.Count) trong C#
func buildIndexGood(users []User) map[int]User {
    index := make(map[int]User, len(users)) // hint: sẽ có len(users) entry
    for _, u := range users {
        index[u.ID] = u // ít khả năng trigger rehash
    }
    return index
}
```

Lưu ý: `make(map[K]V, n)` chỉ là **hint**, không phải cap cứng. Go có thể ignore hay điều chỉnh hint. Nhưng trong thực tế, với hint đúng, số lần rehash giảm đáng kể.

---

## 11. Đọc pprof Output — Tìm bottleneck bằng profiling

pprof là profiler tích hợp sẵn của Go, không cần cài thêm gì. Tương đương Visual Studio Profiler hay dotTrace của JetBrains, nhưng miễn phí và built-in.

**Bật profiling trong HTTP server:**

```go
package main

import (
    "net/http"
    _ "net/http/pprof" // import side effect: đăng ký /debug/pprof routes
    "log"
)

func main() {
    http.HandleFunc("/", myHandler)
    // Endpoint profiling có sẵn tại /debug/pprof/
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func myHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("hello"))
}
```

**Thu thập và xem profile:**

```bash
# CPU profile: record 30 giây
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Memory profile: snapshot hiện tại
go tool pprof http://localhost:8080/debug/pprof/heap

# Trong pprof interactive shell:
# top10        — xem 10 function tốn nhiều nhất
# list myFunc  — xem từng dòng của function
# web          — mở flame graph trong browser (cần graphviz)
```

**Profiling trong test (thường dùng hơn):**

```bash
# Chạy benchmark và lưu CPU profile
go test -bench=BenchmarkMyFunc -cpuprofile=cpu.prof

# Chạy benchmark và lưu memory profile
go test -bench=BenchmarkMyFunc -memprofile=mem.prof

# Phân tích profile
go tool pprof cpu.prof
go tool pprof mem.prof
```

**Những gì cần tìm trong CPU profile:**

Nhìn vào output của `top10` trong pprof, bạn sẽ thấy dạng:

```
      flat  flat%   sum%        cum   cum%
    350ms 35.00% 35.00%      400ms 40.00%  main.processData
    200ms 20.00% 55.00%      200ms 20.00%  runtime.mallocgc
    150ms 15.00% 70.00%      150ms 15.00%  runtime.mapaccess2
```

`flat` là thời gian function đó thực sự chạy. `cum` là bao gồm cả function nó gọi. Nếu `runtime.mallocgc` nằm top — bạn đang allocate quá nhiều. Nếu `runtime.mapaccess2` cao — map lookup là bottleneck.

**Memory profile** cho thấy đâu trong code tạo ra nhiều allocation nhất — đây thường là nơi đầu tiên cần tối ưu.

---

## 12. Low-hanging Fruit — Những vấn đề phổ biến dễ fix

**Unnecessary allocations trong hot loops:**

```go
// BAD — tạo slice mới mỗi iteration
for _, user := range users {
    tags := []string{} // allocate mỗi lần!
    if user.Active {
        tags = append(tags, "active")
    }
    process(tags)
}

// GOOD — tái sử dụng slice, reset length chứ không phải reallocate
tags := make([]string, 0, 5) // allocate một lần ngoài loop
for _, user := range users {
    tags = tags[:0] // reset length về 0, giữ lại capacity
    if user.Active {
        tags = append(tags, "active")
    }
    process(tags)
}
```

**String formatting trong hot loop:**

```go
// BAD — fmt.Sprintf allocate string mới mỗi lần
for _, id := range ids {
    key := fmt.Sprintf("user:%d", id) // allocation!
    cache.Get(key)
}

// GOOD — nếu pattern đơn giản, dùng strconv
import "strconv"

for _, id := range ids {
    key := "user:" + strconv.Itoa(id) // vẫn allocate nhưng ít hơn
    cache.Get(key)
}

// BEST — nếu thực sự cần tối ưu: dùng fixed buffer
var buf [32]byte // stack allocated!
for _, id := range ids {
    b := strconv.AppendInt(buf[:0], int64(id), 10)
    key := "user:" + string(b)
    cache.Get(key)
}
```

**Repeated map lookups:**

```go
// BAD — lookup map 2 lần
if _, ok := cache[key]; ok {
    value := cache[key] // lookup lần 2!
    process(value)
}

// GOOD — lookup một lần, dùng comma-ok idiom
if value, ok := cache[key]; ok {
    process(value) // dùng value từ lần lookup đầu
}
```

---

## 13. Khi Nào KHÔNG Nên Tối Ưu

Đây là phần quan trọng nhất trong bài, nhưng thường bị bỏ qua. Go được thiết kế để code rõ ràng và dễ maintain — đó là ưu tiên số một, không phải hiệu năng.

Tối ưu **trước khi profiling** thường dẫn đến: code khó đọc hơn, bug nhiều hơn, và thường không giải quyết đúng vấn đề. Kinh nghiệm thực tế: 80% bottleneck nằm ở 20% code — và bạn hiếm khi đoán đúng 20% đó mà không có data.

**Những lúc không nên tối ưu:**

Code chạy ít lần (ví dụ: startup initialization, one-off migration script) — optimization ở đây không có ý nghĩa. Code trong administrative path (không phải hot path) — readability quan trọng hơn. Khi chưa có benchmark trước/sau để chứng minh improvement thực sự.

**Dấu hiệu tối ưu quá sớm:**

```go
// Đây là tối ưu quá sớm — khó đọc, không rõ intent
// Chỉ justify nếu profiling chứng minh cần thiết
var keyBuf [64]byte
n := copy(keyBuf[:], "user:")
n += strconv.AppendInt(keyBuf[n:n], int64(id), 10) - keyBuf[n:n]

// Trong 99% trường hợp, đây là đủ tốt và rõ hơn nhiều
key := fmt.Sprintf("user:%d", id)
```

Viết code đúng và rõ ràng trước. Profile sau. Tối ưu chỉ khi có data. Đây không phải lười biếng — đây là engineering discipline.

---

## Bảng Common Issues

| Vấn đề | Triệu chứng | Nguyên nhân | Cách fix |
|--------|-------------|-------------|----------|
| Nhiều allocation | `allocs/op` cao, GC pause nhiều | `+` trong string loop, slice không pre-alloc | `strings.Builder`, `make([]T, 0, n)` |
| Heap escape không cần thiết | Memory cao bất thường | Trả về pointer không cần thiết | Trả về value, dùng `go build -gcflags="-m"` để kiểm tra |
| Map rehash chậm | Insert vào map đột ngột chậm | Map không có size hint | `make(map[K]V, expectedSize)` |
| Interface overhead | CPU profile cho thấy indirect call cao | Interface dispatch trong hot loop | Dùng concrete type trong inner loop |
| Struct padding lãng phí | Memory usage cao hơn dự kiến | Field nhỏ trước field lớn | Sắp xếp field lớn trước, dùng `fieldalignment` |
| Tạo buffer mỗi request | High GC pressure trong HTTP server | Allocate `bytes.Buffer` trong handler | `sync.Pool` để tái sử dụng buffer |
| Repeated map lookup | CPU profile cho thấy `mapaccess2` cao | Lookup key 2 lần trong if | Comma-ok idiom: `v, ok := m[k]` |
| Format string trong loop | Nhiều `fmt.Sprintf` allocation | String format tạo allocation mỗi lần | `strconv`, `strings.Builder`, hoặc cache kết quả |

---

## Bài Tập Thực Hành

### Bài tập 1: Tối ưu CSV Builder

Cho đoạn code sau tạo CSV từ slice of records. Viết benchmark, xác định vấn đề, và tối ưu.

**Code ban đầu (BAD):**

```go
package main

import (
    "fmt"
    "strings"
)

type Record struct {
    ID    int
    Name  string
    Email string
    Score float64
}

func buildCSVBad(records []Record) string {
    result := "id,name,email,score\n"
    for _, r := range records {
        result += fmt.Sprintf("%d,%s,%s,%.2f\n", r.ID, r.Name, r.Email, r.Score)
    }
    return result
}

func main() {
    records := []Record{
        {1, "Alice", "alice@example.com", 95.5},
        {2, "Bob", "bob@example.com", 87.3},
    }
    fmt.Println(buildCSVBad(records))
    _ = strings.TrimSpace // import để tránh lỗi unused
}
```

**Solution đầy đủ với benchmark:**

```go
// File: csv_bench_test.go
package main

import (
    "fmt"
    "strconv"
    "strings"
    "testing"
)

type Record struct {
    ID    int
    Name  string
    Email string
    Score float64
}

// BAD: dùng += và fmt.Sprintf trong loop
func buildCSVBad(records []Record) string {
    result := "id,name,email,score\n"
    for _, r := range records {
        // Mỗi iteration: 1 allocation từ Sprintf + 1 allocation từ +=
        result += fmt.Sprintf("%d,%s,%s,%.2f\n", r.ID, r.Name, r.Email, r.Score)
    }
    return result
}

// GOOD: dùng strings.Builder với pre-allocated capacity
func buildCSVGood(records []Record) string {
    // Ước tính: header ~20 bytes + mỗi record ~50 bytes
    var sb strings.Builder
    sb.Grow(20 + len(records)*50)

    sb.WriteString("id,name,email,score\n")
    for _, r := range records {
        // strconv tránh allocation của fmt.Sprintf
        sb.WriteString(strconv.Itoa(r.ID))
        sb.WriteByte(',')
        sb.WriteString(r.Name)
        sb.WriteByte(',')
        sb.WriteString(r.Email)
        sb.WriteByte(',')
        sb.WriteString(strconv.FormatFloat(r.Score, 'f', 2, 64))
        sb.WriteByte('\n')
    }
    return sb.String()
}

// Tạo test data để benchmark
func makeRecords(n int) []Record {
    records := make([]Record, n)
    for i := range records {
        records[i] = Record{
            ID:    i + 1,
            Name:  "User" + strconv.Itoa(i),
            Email: "user" + strconv.Itoa(i) + "@example.com",
            Score: float64(i%100) + 0.5,
        }
    }
    return records
}

func BenchmarkCSVBad(b *testing.B) {
    records := makeRecords(1000)
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = buildCSVBad(records)
    }
}

func BenchmarkCSVGood(b *testing.B) {
    records := makeRecords(1000)
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = buildCSVGood(records)
    }
}

// Đảm bảo output giống nhau (correctness check)
func TestCSVCorrectness(t *testing.T) {
    records := makeRecords(5)
    bad := buildCSVBad(records)
    good := buildCSVGood(records)
    if bad != good {
        t.Errorf("Output mismatch:\nBAD:\n%s\nGOOD:\n%s", bad, good)
    }
}

// Chạy: go test -bench=. -benchmem
// Kết quả mong đợi:
// BenchmarkCSVBad-8     1000    1523000 ns/op   2048321 B/op   2001 allocs/op
// BenchmarkCSVGood-8   12000     95000 ns/op     75000 B/op      1 allocs/op

// main để file compile được
func main() {
    records := makeRecords(3)
    fmt.Println(buildCSVGood(records))
    _ = strings.TrimSpace
}
```

Chạy: `go test -bench=. -benchmem -v`

---

### Bài tập 2: HTTP Handler với sync.Pool

Viết một HTTP handler trả về JSON response. Benchmark version không dùng Pool và version dùng Pool. Sau đó thêm pprof để có thể profile trong thực tế.

**Solution đầy đủ:**

```go
// File: handler_pool_test.go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "sync"
    "testing"
)

// Response struct cho JSON
type APIResponse struct {
    Status  string            `json:"status"`
    Data    map[string]string `json:"data"`
    Count   int               `json:"count"`
}

// BAD: allocate buffer mới mỗi request
func handlerBad(w http.ResponseWriter, r *http.Request) {
    resp := APIResponse{
        Status: "ok",
        Data:   map[string]string{"key": "value", "env": "prod"},
        Count:  42,
    }
    var buf bytes.Buffer // allocate mỗi lần!
    json.NewEncoder(&buf).Encode(resp)
    w.Header().Set("Content-Type", "application/json")
    w.Write(buf.Bytes())
}

// GOOD: dùng sync.Pool để tái sử dụng buffer
var responsePool = sync.Pool{
    New: func() interface{} {
        // Tạo buffer với initial capacity để tránh resize
        buf := bytes.NewBuffer(make([]byte, 0, 512))
        return buf
    },
}

func handlerGood(w http.ResponseWriter, r *http.Request) {
    resp := APIResponse{
        Status: "ok",
        Data:   map[string]string{"key": "value", "env": "prod"},
        Count:  42,
    }

    // Lấy buffer từ Pool
    buf := responsePool.Get().(*bytes.Buffer)
    buf.Reset() // reset content, giữ capacity

    json.NewEncoder(buf).Encode(resp)

    w.Header().Set("Content-Type", "application/json")
    w.Write(buf.Bytes())

    // Trả buffer về Pool
    // Đừng dùng buf sau dòng này!
    responsePool.Put(buf)
}

func BenchmarkHandlerBad(b *testing.B) {
    b.ReportAllocs()
    req := httptest.NewRequest("GET", "/", nil)
    for i := 0; i < b.N; i++ {
        w := httptest.NewRecorder()
        handlerBad(w, req)
    }
}

func BenchmarkHandlerGood(b *testing.B) {
    b.ReportAllocs()
    req := httptest.NewRequest("GET", "/", nil)
    for i := 0; i < b.N; i++ {
        w := httptest.NewRecorder()
        handlerGood(w, req)
    }
}

// Chạy: go test -bench=. -benchmem
// Kết quả mong đợi:
// BenchmarkHandlerBad-8    200000    8234 ns/op    1024 B/op    7 allocs/op
// BenchmarkHandlerGood-8   350000    4123 ns/op     256 B/op    4 allocs/op

// Để profil trong production, thêm vào main:
// import _ "net/http/pprof"
// Sau đó: go tool pprof http://localhost:8080/debug/pprof/heap

func main() {
    // Version với pprof endpoint cho production profiling
    http.HandleFunc("/bad", handlerBad)
    http.HandleFunc("/good", handlerGood)
    // Uncommit dòng dưới để thêm pprof:
    // import _ "net/http/pprof" ở đầu file
    http.ListenAndServe(":8080", nil)
}
```

Sau khi chạy server, thu thập heap profile bằng:

```bash
# Load test với 1000 concurrent requests
go tool pprof http://localhost:8080/debug/pprof/heap

# Trong pprof shell
(pprof) top10
(pprof) list handlerBad
(pprof) web    # mở flame graph (cần graphviz)
```

---

## Tổng kết

Go nhanh theo mặc định, nhưng hiểu rõ cơ chế bên dưới giúp bạn viết code không chỉ đúng mà còn hiệu quả. Những gì cần nhớ:

Escape analysis quyết định stack hay heap — dùng `go build -gcflags="-m"` để xem. Pre-allocate slice và map khi biết size trước. `strings.Builder` cho string building, `bytes.Buffer` khi cần io interface. `sync.Pool` cho temporary objects trong hot path. Sắp xếp struct field từ lớn đến nhỏ. Interface có overhead trong hot loop — concrete type nhanh hơn. Luôn đo trước khi tối ưu — dùng `go test -bench` và `pprof`.

Quan trọng nhất: **đừng tối ưu khi chưa đo**. Code rõ ràng, đúng đắn, có test — đó là nền tảng. Tối ưu là bước tiếp theo, chỉ khi profiling cho thấy cần thiết.

---

**Next:** [20. Project Setup Checklist](20-setup-checklist.md) — Cài đặt toolchain đầy đủ (Docker, migrate CLI, sqlc, grpcurl...) trước khi bước vào các dự án thực tế ở Phase 4 trở đi. Sau đó xem [22. Mini E-commerce Project](22-mini-ecommerce-project.md) để bắt đầu áp dụng tổng hợp những gì đã học.
