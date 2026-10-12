# 17. Go vs C# — Những Cạm Bẫy Phổ Biến (Gotchas)

Đây là tài liệu tổng hợp những lỗi sai phổ biến nhất mà lập trình viên C# mắc phải khi học Go. Mỗi mục giải thích **tại sao** đây là cạm bẫy, không chỉ đơn thuần là "làm đúng thế này". Hiểu nguyên nhân giúp bạn suy luận đúng trong các tình huống mới, thay vì chỉ thuộc lòng quy tắc.

---

## 1. Nil Interface Gotcha — Typed Nil vs Untyped Nil

Trong C#, `null == null` luôn luôn là `true`. Không có ngoại lệ. Go không hoạt động theo cách đó, và đây là cạm bẫy số một gây ra bug khó debug nhất.

Một interface trong Go được biểu diễn nội bộ như một cặp `(type, value)`. Một interface chỉ bằng `nil` khi **cả hai** trường đó đều là nil. Khi bạn gán một con trỏ typed nil vào một interface, phần `type` của cặp đó được điền, khiến interface không còn là nil nữa, dù giá trị bên trong là nil.

Hãy tưởng tượng interface như một chiếc hộp có nhãn tên loại và chứa giá trị bên trong. Một chiếc hộp rỗng hoàn toàn (không nhãn, không nội dung) là nil. Nhưng một chiếc hộp có nhãn `*MyError` và bên trong trống thì **không phải** nil — nó vẫn là một chiếc hộp hợp lệ.

```go
// BAD — lỗi kinh điển của developer C#
package main

import "fmt"

type MyError struct {
    Message string
}

func (e *MyError) Error() string {
    return e.Message
}

// Hàm này có vẻ đúng nhưng thực ra có bug
func getError(fail bool) error {
    var err *MyError // typed nil — kiểu là *MyError, giá trị là nil

    if fail {
        err = &MyError{Message: "something went wrong"}
    }

    // BUG: khi fail=false, err là *MyError(nil), không phải error(nil)
    // Interface (type=*MyError, value=nil) != nil
    return err
}

func main() {
    err := getError(false)

    // Developer C# expect: err == nil → true
    // Thực tế: err == nil → FALSE! Vì interface có type=*MyError
    if err != nil {
        fmt.Println("CÓ lỗi — unexpected!") // Dòng này sẽ chạy!
    }
}
```

```go
// GOOD — trả về untyped nil trực tiếp
package main

import "fmt"

type MyError struct {
    Message string
}

func (e *MyError) Error() string {
    return e.Message
}

// Cách đúng: trả về nil trực tiếp khi không có lỗi
func getError(fail bool) error {
    if fail {
        return &MyError{Message: "something went wrong"}
    }
    // Trả về untyped nil — interface (type=nil, value=nil) == nil
    return nil
}

func main() {
    err := getError(false)

    if err != nil {
        fmt.Println("CÓ lỗi")
    } else {
        fmt.Println("Không có lỗi — đúng rồi!") // Dòng này chạy
    }
}
```

Quy tắc: **không bao giờ** trả về một con trỏ typed nil thông qua kiểu interface. Luôn trả về `nil` trực tiếp, hoặc kiểm tra với `reflect.ValueOf(err).IsNil()` nếu buộc phải nhận interface từ code ngoài.

---

## 2. Slice Dùng Chung Underlying Array

Trong C#, khi bạn gán `List<int> b = a`, bạn sao chép tham chiếu đến list, và khi bạn gọi `b.Add(...)`, nó tác động lên cùng list. Go hoạt động tương tự, nhưng có một điểm phức tạp hơn: slice là một struct ba trường `(pointer, length, capacity)`, và hai slice khác nhau có thể trỏ vào **cùng một mảng nền**.

Khi `append` không vượt quá capacity, Go ghi dữ liệu vào mảng nền hiện có và trả về slice mới với length tăng lên — điều đó nghĩa là slice cũ và mới **đang chia sẻ bộ nhớ**. Khi capacity bị vượt quá, Go cấp phát mảng mới và copy toàn bộ dữ liệu sang.

```go
// BAD — không hiểu slice sharing gây ra bug khó phát hiện
package main

import "fmt"

func addAdmin(users []string) []string {
    // Thêm admin vào slice được truyền vào
    // Nếu capacity còn đủ, thao tác này ghi vào mảng nền của caller!
    return append(users, "admin")
}

func main() {
    // Tạo slice với capacity=5 nhưng length=3
    original := make([]string, 3, 5)
    original[0] = "alice"
    original[1] = "bob"
    original[2] = "charlie"

    withAdmin := addAdmin(original)

    // Developer C# expect: original không thay đổi
    // Thực tế: original[3] đã bị ghi (dù length vẫn là 3)
    // Và nếu ai đó reslice: original[:4] sẽ thấy "admin"!
    fmt.Println("original:", original)       // [alice bob charlie]
    fmt.Println("withAdmin:", withAdmin)     // [alice bob charlie admin]
    fmt.Println("Cùng array?", &original[0] == &withAdmin[0]) // true!
}
```

```go
// GOOD — copy rõ ràng khi cần độc lập
package main

import "fmt"

func addAdmin(users []string) []string {
    // Tạo slice mới, đảm bảo không ảnh hưởng đến caller
    result := make([]string, len(users), len(users)+1)
    copy(result, users)
    return append(result, "admin")
}

// Hoặc dùng cú pháp copy ngắn gọn hơn
func addAdminV2(users []string) []string {
    result := append([]string(nil), users...) // copy toàn bộ vào slice mới
    return append(result, "admin")
}

func main() {
    original := []string{"alice", "bob", "charlie"}

    withAdmin := addAdmin(original)
    withAdmin2 := addAdminV2(original)

    fmt.Println("original:", original)        // [alice bob charlie] — không đổi
    fmt.Println("withAdmin:", withAdmin)      // [alice bob charlie admin]
    fmt.Println("withAdmin2:", withAdmin2)    // [alice bob charlie admin]
    fmt.Println("Cùng array?", &original[0] == &withAdmin[0]) // false
}
```

Quy tắc: nếu bạn nhận slice và cần chắc chắn thay đổi của mình không ảnh hưởng đến caller, hãy `copy` trước. Khi viết hàm nhận slice, document rõ hàm đó có mutate slice hay không.

---

## 3. Map Không Thread-Safe và Go Panic Hard

C# `Dictionary<K,V>` không thread-safe và developer C# biết điều này. Tuy nhiên, trong C# nếu bạn đọc/ghi concurrent mà không lock, kết quả là **undefined behavior** — thường là race condition thầm lặng hoặc exception. Trong Go, runtime **phát hiện concurrent map access** và gọi `panic` ngay lập tức với message rõ ràng. Đây thực ra là thiết kế tốt hơn, nhưng developer C# thường bị bất ngờ.

Go có built-in race detector (`go run -race`) giúp phát hiện data race sớm trong quá trình development.

```go
// BAD — concurrent map access gây panic trong Go
package main

import (
    "fmt"
    "sync"
)

func main() {
    m := make(map[string]int)
    var wg sync.WaitGroup

    // Chạy 100 goroutine cùng ghi vào map
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            key := fmt.Sprintf("key%d", n%10)
            m[key] = n // PANIC: concurrent map writes!
        }(i)
    }

    wg.Wait()
    fmt.Println("Done:", len(m))
}
// Output: fatal error: concurrent map writes
```

```go
// GOOD — Option 1: dùng sync.RWMutex
package main

import (
    "fmt"
    "sync"
)

type SafeMap struct {
    mu sync.RWMutex
    m  map[string]int
}

func NewSafeMap() *SafeMap {
    return &SafeMap{m: make(map[string]int)}
}

func (sm *SafeMap) Set(key string, value int) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.m[key] = value
}

func (sm *SafeMap) Get(key string) (int, bool) {
    sm.mu.RLock() // Read lock — cho phép nhiều reader cùng lúc
    defer sm.mu.RUnlock()
    v, ok := sm.m[key]
    return v, ok
}

func main() {
    sm := NewSafeMap()
    var wg sync.WaitGroup

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            key := fmt.Sprintf("key%d", n%10)
            sm.Set(key, n)
        }(i)
    }

    wg.Wait()
    fmt.Println("Done, map size:", len(sm.m))
}
```

```go
// GOOD — Option 2: dùng sync.Map (tối ưu cho write-once, read-many)
package main

import (
    "fmt"
    "sync"
)

func main() {
    var sm sync.Map

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            key := fmt.Sprintf("key%d", n%10)
            sm.Store(key, n) // thread-safe
        }(i)
    }

    wg.Wait()

    sm.Range(func(k, v any) bool {
        fmt.Printf("%s = %v\n", k, v)
        return true // true = tiếp tục iterate
    })
}
```

Chạy với `-race` flag trong development: `go run -race main.go`. Race detector sẽ báo ngay khi có vấn đề.

---

## 4. For Range Loop Variable Capture

Đây là bug cổ điển nhất trong Go, từng xuất hiện trong hàng nghìn codebase trước Go 1.22. Khi bạn dùng goroutine bên trong `for range` loop, goroutine capture **biến loop** theo reference, không phải theo value. Vì goroutine chạy bất đồng bộ, đến khi nó thực sự chạy thì biến loop đã thay đổi sang giá trị khác.

Trong C#, `foreach` với lambda có behavior tương tự — đây là lý do C# phải thêm `int captured = i; Task.Run(() => captured)`.

```go
// BAD — tất cả goroutine capture cùng một biến v
package main

import (
    "fmt"
    "sync"
)

func main() {
    names := []string{"Alice", "Bob", "Charlie"}
    var wg sync.WaitGroup

    for _, v := range names {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // BUG: v là biến của loop, khi goroutine chạy
            // loop đã kết thúc → v = "Charlie" cho tất cả goroutine
            fmt.Println(v)
        }()
    }

    wg.Wait()
    // Output thường là: Charlie Charlie Charlie
    // (thứ tự có thể khác nhưng tất cả đều là "Charlie")
}
```

```go
// GOOD — Go 1.22+: mỗi iteration tạo biến v mới (tự động fixed!)
// Nếu dùng Go 1.22+, behavior đã được thay đổi: mỗi iteration
// có biến v riêng. Đây là breaking change nhỏ nhưng Fix lỗi phổ biến.

// GOOD — tương thích tất cả version: truyền qua parameter
package main

import (
    "fmt"
    "sync"
)

func main() {
    names := []string{"Alice", "Bob", "Charlie"}
    var wg sync.WaitGroup

    for _, v := range names {
        wg.Add(1)
        v := v // Tạo biến mới trong scope của iteration này (shadow)
        go func() {
            defer wg.Done()
            fmt.Println(v) // capture biến v mới của iteration này
        }()
    }

    wg.Wait()
    // Output: Alice, Bob, Charlie (thứ tự khác nhau nhưng đủ cả 3)
}
```

```go
// GOOD — cách khác: truyền làm parameter của goroutine
package main

import (
    "fmt"
    "sync"
)

func main() {
    names := []string{"Alice", "Bob", "Charlie"}
    var wg sync.WaitGroup

    for _, v := range names {
        wg.Add(1)
        go func(name string) { // name là copy của v tại thời điểm gọi
            defer wg.Done()
            fmt.Println(name)
        }(v) // truyền v ngay lúc khởi tạo goroutine
    }

    wg.Wait()
}
```

Nếu dùng Go 1.22+, hãy kiểm tra `go.mod` để đảm bảo `go 1.22` hoặc cao hơn — khi đó behavior mặc định đã an toàn. Với project cũ hơn, luôn dùng `v := v` hoặc truyền parameter.

---

## 5. String vs []byte — Immutability và Allocation

Trong C#, `string` là immutable và có nhiều built-in methods tiện lợi. `byte[]` ít dùng hơn. Trong Go, `string` cũng immutable, nhưng Go developer thường xuyên phải chuyển đổi qua lại giữa `string` và `[]byte` vì nhiều API dùng một trong hai.

Điều quan trọng cần hiểu: **mỗi lần convert giữa `string` và `[]byte` là một allocation mới** (trừ một số trường hợp compiler optimize được). Go runtime phải copy dữ liệu vì string là immutable nhưng `[]byte` thì mutable. Trong hot path, điều này ảnh hưởng performance đáng kể.

```go
// BAD — convert không cần thiết trong hot path
package main

import (
    "fmt"
    "strings"
)

func processLog(line string) string {
    // Convert sang []byte, xử lý, convert ngược lại
    // Mỗi bước = allocation mới
    b := []byte(line)                    // allocation 1
    b = append(b, []byte(" [processed]")...) // allocation 2
    upper := strings.ToUpper(string(b)) // allocation 3
    return upper
}

func main() {
    result := processLog("user logged in")
    fmt.Println(result)
}
```

```go
// GOOD — dùng strings.Builder để tránh allocation thừa
package main

import (
    "fmt"
    "strings"
)

func processLog(line string) string {
    var sb strings.Builder
    sb.Grow(len(line) + 12) // pre-allocate để tránh resize
    sb.WriteString(strings.ToUpper(line))
    sb.WriteString(" [PROCESSED]")
    return sb.String() // chỉ một allocation ở đây
}

// Khi nào dùng []byte vs string:
// - string: text đọc nhiều, key trong map, không cần modify
// - []byte: I/O (file, network, HTTP body), cần modify, dùng với bufio
// - strings.Builder: build string từng phần
// - bytes.Buffer: như strings.Builder nhưng cho []byte API

func main() {
    result := processLog("user logged in")
    fmt.Println(result)
}
```

```go
// GOOD — ví dụ thực tế: đọc HTTP body
package main

import (
    "fmt"
    "io"
    "strings"
)

func readBody(r io.Reader) (string, error) {
    // io.ReadAll trả về []byte — convert sang string để parse JSON
    // Đây là allocation cần thiết, không tránh được
    data, err := io.ReadAll(r)
    if err != nil {
        return "", err
    }
    return string(data), nil // một allocation
}

func main() {
    body := strings.NewReader(`{"name": "Alice"}`)
    content, _ := readBody(body)
    fmt.Println(content)
}
```

---

## 6. Integer Overflow Im Lặng

C# với `checked` context (hoặc `checked {}` block) sẽ throw `OverflowException` khi integer overflow. Unchecked context (default) wrap around như Go. Go **không có** exception khi overflow — nó luôn wrap around theo two's complement arithmetic, không bao giờ panic.

Đây là thiết kế cố ý: Go ưu tiên performance và explicit control. Nhưng nếu bạn từ C# quen với việc runtime "bảo vệ" bạn, đây là cạm bẫy nghiêm trọng, đặc biệt trong financial calculation.

```go
// BAD — overflow im lặng gây kết quả sai
package main

import "fmt"

func main() {
    var x int8 = 127
    x++ // int8 max là 127, overflow → -128 (không panic!)
    fmt.Println("int8 overflow:", x) // -128

    var price uint32 = 4_000_000_000 // gần max uint32
    quantity := uint32(2)
    total := price * quantity // overflow! kết quả sai hoàn toàn
    fmt.Println("total (sai):", total) // số rác

    // int16 max = 32767
    var score int16 = 32767
    score++
    fmt.Println("int16 overflow:", score) // -32768
}
```

```go
// GOOD — kiểm tra overflow trước khi thực hiện phép tính
package main

import (
    "fmt"
    "math"
)

func safeMultiplyUint32(a, b uint32) (uint32, error) {
    if a > 0 && b > math.MaxUint32/a {
        return 0, fmt.Errorf("overflow: %d * %d vượt quá uint32 max", a, b)
    }
    return a * b, nil
}

func safeAddInt(a, b int) (int, error) {
    if (b > 0 && a > math.MaxInt-b) || (b < 0 && a < math.MinInt-b) {
        return 0, fmt.Errorf("overflow khi cộng %d + %d", a, b)
    }
    return a + b, nil
}

func main() {
    // Với financial data, dùng int64 hoặc thư viện decimal
    var price int64 = 4_000_000_000
    var quantity int64 = 2
    total := price * quantity // an toàn với int64 cho giá trị này
    fmt.Println("total (đúng):", total) // 8000000000

    // Kiểm tra overflow
    result, err := safeMultiplyUint32(4_000_000_000, 2)
    if err != nil {
        fmt.Println("Lỗi:", err) // sẽ in lỗi
    } else {
        fmt.Println("result:", result)
    }
}
```

Với financial calculation, dùng `int64` (lưu cents thay vì dollars) hoặc thư viện như `github.com/shopspring/decimal`. Không bao giờ dùng `float64` cho tiền tệ — lý do tương tự như C#.

---

## 7. Shadowing với := — Biến Mới Che Biến Cũ

Operator `:=` trong Go vừa khai báo vừa gán. Khi bạn dùng `:=` trong một inner scope, Go tạo ra một biến **mới** với cùng tên, che khuất (shadow) biến ở outer scope. Biến outer scope không bị thay đổi. Compiler **không cảnh báo** về điều này (trừ khi bạn dùng `go vet` hoặc linter).

Đây khác với C# — C# không cho phép khai báo biến cùng tên trong scope lồng nhau nếu outer variable đang trong scope.

```go
// BAD — shadowing gây bug: err ở ngoài không bao giờ được gán
package main

import (
    "errors"
    "fmt"
)

func riskyOperation() error {
    return errors.New("operation failed")
}

func process() error {
    var err error // biến err ở outer scope

    if true {
        // := tạo biến err MỚI trong scope này, không ảnh hưởng outer err
        result, err := riskyOperation()
        _ = result
        fmt.Println("inner err:", err) // in lỗi đúng
    }

    // BUG: outer err vẫn là nil! Không phải lỗi từ riskyOperation
    return err // luôn trả về nil
}

func main() {
    err := process()
    if err != nil {
        fmt.Println("Có lỗi:", err)
    } else {
        fmt.Println("Không có lỗi — nhưng thực ra có lỗi bị bỏ qua!") // dòng này chạy
    }
}
```

```go
// GOOD — khai báo biến riêng, gán err ở outer scope
package main

import (
    "errors"
    "fmt"
)

func riskyOperation() (string, error) {
    return "", errors.New("operation failed")
}

func process() error {
    var err error
    var result string

    if true {
        // Gán vào biến đã khai báo — dùng = không phải :=
        result, err = riskyOperation()
        _ = result
    }

    return err // đúng: err được gán từ riskyOperation
}

// Hoặc khai báo ngay tại chỗ dùng
func processV2() error {
    if true {
        result, err := riskyOperation()
        _ = result
        if err != nil {
            return fmt.Errorf("processV2: %w", err)
        }
    }
    return nil
}

func main() {
    err := process()
    fmt.Println("process err:", err) // operation failed

    err2 := processV2()
    fmt.Println("processV2 err:", err2) // processV2: operation failed
}
```

Dùng `go vet ./...` và linter như `staticcheck` hoặc `golangci-lint` để phát hiện shadowing. Cấu hình IDE (VSCode với gopls) cũng highlight shadowed variables.

---

## 8. Goroutine Không Phải là Task

Developer C# quen với `Task<T>` — nó có built-in result, cancellation thông qua `CancellationToken`, continuation với `.ContinueWith()` hoặc `await`, và exception propagation. Goroutine là primitive cấp thấp hơn nhiều: chỉ là "chạy hàm này concurrently". Không có kết quả, không có cancellation tự động, không có exception propagation.

Nếu một goroutine panic và không được recover, **toàn bộ program crash**, không phải chỉ goroutine đó. Đây khác hoàn toàn với C# Task — unhandled exception trong Task không crash app (chỉ gây UnobservedTaskException).

```go
// BAD — goroutine không có cơ chế trả kết quả hay xử lý lỗi
package main

import (
    "fmt"
    "time"
)

func fetchData(url string) string {
    // Simulate work
    time.Sleep(100 * time.Millisecond)
    return "data from " + url
}

func main() {
    // Không thể nhận kết quả từ goroutine như này
    go func() {
        result := fetchData("https://api.example.com")
        // result bị "mất" — không ai nhận
        _ = result
    }()

    // Và không biết khi nào goroutine xong
    time.Sleep(200 * time.Millisecond) // hack tệ hại
    fmt.Println("Done?")
}
```

```go
// GOOD — dùng channel để truyền kết quả và lỗi (pattern chuẩn)
package main

import (
    "fmt"
    "time"
)

type Result struct {
    Data string
    Err  error
}

func fetchData(url string) (string, error) {
    time.Sleep(50 * time.Millisecond)
    if url == "" {
        return "", fmt.Errorf("url trống")
    }
    return "data from " + url, nil
}

func main() {
    urls := []string{
        "https://api.example.com/users",
        "https://api.example.com/orders",
        "",
    }

    // Mỗi goroutine gửi kết quả qua channel
    ch := make(chan Result, len(urls)) // buffered — goroutine không bị block

    for _, url := range urls {
        url := url // capture
        go func() {
            data, err := fetchData(url)
            ch <- Result{Data: data, Err: err}
        }()
    }

    // Thu thập kết quả — tương đương Task.WhenAll
    for range urls {
        r := <-ch
        if r.Err != nil {
            fmt.Println("Lỗi:", r.Err)
        } else {
            fmt.Println("OK:", r.Data)
        }
    }
}
```

```go
// GOOD — cancellation với context (tương đương CancellationToken)
package main

import (
    "context"
    "fmt"
    "time"
)

func fetchWithTimeout(ctx context.Context, url string) (string, error) {
    // Simulate I/O với context awareness
    select {
    case <-time.After(200 * time.Millisecond):
        return "data from " + url, nil
    case <-ctx.Done():
        return "", ctx.Err() // context.DeadlineExceeded hoặc Canceled
    }
}

func main() {
    // Timeout 100ms — fetch cần 200ms → sẽ bị cancel
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    result, err := fetchWithTimeout(ctx, "https://api.example.com")
    if err != nil {
        fmt.Println("Lỗi:", err) // context deadline exceeded
    } else {
        fmt.Println("OK:", result)
    }
}
```

---

## 9. Không Có While Loop — Dùng for

Đây là gotcha nhỏ nhưng gây friction cho developer C#. Go chỉ có một loại loop: `for`. Không có `while`, không có `do...while`. Tất cả đều là `for` với syntax khác nhau.

```go
// BAD — cố dùng pattern không tồn tại trong Go
// while (condition) { ... }   ← KHÔNG CÓ
// do { ... } while (condition) ← KHÔNG CÓ

// GOOD — các dạng for loop trong Go
package main

import "fmt"

func main() {
    // 1. for như while(condition)
    n := 0
    for n < 5 {
        fmt.Print(n, " ")
        n++
    }
    fmt.Println()

    // 2. for như while(true) / infinite loop
    count := 0
    for {
        if count >= 3 {
            break
        }
        fmt.Print("tick ", count, " | ")
        count++
    }
    fmt.Println()

    // 3. for như do-while (check ở cuối)
    i := 0
    for {
        fmt.Print("do: ", i, " ")
        i++
        if i >= 3 {
            break // check condition ở cuối → giống do-while
        }
    }
    fmt.Println()

    // 4. for range — như foreach trong C#
    names := []string{"Alice", "Bob", "Charlie"}
    for idx, name := range names {
        fmt.Printf("[%d] %s\n", idx, name)
    }

    // 5. for range chỉ lấy index
    for i := range names {
        fmt.Print(i, " ")
    }
    fmt.Println()

    // 6. for range map
    m := map[string]int{"a": 1, "b": 2}
    for k, v := range m {
        fmt.Printf("%s=%d ", k, v)
    }
    fmt.Println()
}
```

---

## 10. Error Bị Bỏ Qua Bằng _ Là Im Lặng

Trong C#, nếu một method throw exception và bạn không catch, exception đó bubble up và có thể crash app — rất khó bỏ qua hoàn toàn. Trong Go, error là giá trị trả về bình thường. Dùng `_` để discard error là hoàn toàn hợp lệ về mặt compiler, nhưng nghĩa là bạn đã tắt toàn bộ error handling.

Go tin tưởng developer phải tự xử lý lỗi. Compiler không bắt buộc bạn handle error — đây là tradeoff có chủ ý: ít boilerplate hơn Java/C# checked exceptions, nhưng đòi hỏi discipline cao hơn.

```go
// BAD — bỏ qua error với _
package main

import (
    "fmt"
    "os"
    "strconv"
)

func main() {
    // Tất cả error đều bị bỏ qua!
    data, _ := os.ReadFile("config.json")   // nếu file không tồn tại, data là nil
    value, _ := strconv.Atoi(string(data))  // parse "" → value = 0, lỗi bị bỏ
    fmt.Println("Config value:", value)     // in 0, không biết có lỗi không

    // Ghi file — nếu lỗi permission, không biết
    _ = os.WriteFile("output.txt", []byte("hello"), 0644)
    fmt.Println("Ghi file xong?") // Không chắc!
}
```

```go
// GOOD — xử lý error explicit
package main

import (
    "fmt"
    "os"
    "strconv"
)

func loadConfig(path string) (int, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return 0, fmt.Errorf("loadConfig: đọc file %s: %w", path, err)
    }

    value, err := strconv.Atoi(string(data))
    if err != nil {
        return 0, fmt.Errorf("loadConfig: parse giá trị '%s': %w", string(data), err)
    }

    return value, nil
}

func main() {
    value, err := loadConfig("config.txt")
    if err != nil {
        // Log và xử lý — không im lặng bỏ qua
        fmt.Fprintf(os.Stderr, "Lỗi load config: %v\n", err)
        // Dùng giá trị mặc định hoặc exit
        value = 42 // fallback default
    }

    fmt.Println("Config value:", value)

    // Ghi file với error handling
    if err := os.WriteFile("output.txt", []byte("hello"), 0644); err != nil {
        fmt.Fprintf(os.Stderr, "Không thể ghi file: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("Ghi file thành công")
}
```

Khi nào `_` là chấp nhận được: khi bạn đang prototype, khi API document rõ rằng lỗi không thể xảy ra trong điều kiện đó (ví dụ `fmt.Fprintf(os.Stdout, ...)` — stdout không bao giờ lỗi trong practice), hoặc khi bạn đang defer cleanup và không còn gì để làm với lỗi.

---

## 11. Struct Comparison — Chỉ Comparable Khi Tất Cả Fields Comparable

Trong C#, `==` trên struct value types so sánh từng field tự động (value semantics). Trong Go, struct có thể được so sánh bằng `==` nhưng **chỉ khi tất cả fields của nó đều comparable**. Map, slice, và function không comparable — struct chứa bất kỳ field nào trong số đó sẽ không thể dùng `==` hoặc làm map key.

```go
// BAD — cố compare struct có slice field
package main

import "fmt"

type UserBad struct {
    Name  string
    Roles []string // slice không phải comparable!
}

func main() {
    u1 := UserBad{Name: "Alice", Roles: []string{"admin"}}
    u2 := UserBad{Name: "Alice", Roles: []string{"admin"}}

    // Compiler error: invalid operation: u1 == u2
    // (struct containing []string cannot be compared)
    fmt.Println(u1 == u2) // COMPILE ERROR
}
```

```go
// GOOD — dùng reflect.DeepEqual hoặc thiết kế struct khác
package main

import (
    "fmt"
    "reflect"
    "sort"
)

type User struct {
    Name  string
    Roles []string
}

// Comparable version — không có slice
type UserKey struct {
    Name string
    Role string // denormalized, hoặc join thành string
}

func usersEqual(a, b User) bool {
    if a.Name != b.Name {
        return false
    }
    // DeepEqual so sánh slice theo value
    return reflect.DeepEqual(a.Roles, b.Roles)
}

// Nếu cần dùng làm map key, normalize trước
func userMapKey(u User) string {
    roles := append([]string(nil), u.Roles...)
    sort.Strings(roles)
    return u.Name + ":" + fmt.Sprint(roles)
}

func main() {
    u1 := User{Name: "Alice", Roles: []string{"admin", "user"}}
    u2 := User{Name: "Alice", Roles: []string{"admin", "user"}}

    fmt.Println("Equal:", usersEqual(u1, u2)) // true
    fmt.Println("DeepEqual:", reflect.DeepEqual(u1, u2)) // true

    // Dùng string key thay vì struct key
    cache := make(map[string]bool)
    cache[userMapKey(u1)] = true
    fmt.Println("In cache:", cache[userMapKey(u2)]) // true
}
```

Các type comparable trong Go: bool, int/float/complex variants, string, pointer, channel, array (khi element type comparable), struct (khi tất cả field comparable). Không comparable: slice, map, function.

---

## 12. JSON: nil Slice vs Empty Slice

Đây là gotcha ảnh hưởng trực tiếp đến API compatibility. Trong Go, `nil` slice và empty slice (`[]string{}`) là hai thứ khác nhau khi marshal JSON: nil slice cho ra `null`, empty slice cho ra `[]`. Phía client API (JavaScript, C#) thường expect `[]` thay vì `null` cho danh sách rỗng.

```go
// BAD — nil slice trong struct field → null trong JSON
package main

import (
    "encoding/json"
    "fmt"
)

type Response struct {
    Items []string `json:"items"`
    Tags  []string `json:"tags"`
}

func main() {
    // Không initialize slice → nil
    resp := Response{
        Items: []string{"apple", "banana"},
        // Tags không được set → nil slice
    }

    data, _ := json.Marshal(resp)
    fmt.Println(string(data))
    // Output: {"items":["apple","banana"],"tags":null}
    // Client nhận null thay vì [] — có thể gây lỗi client-side!
}
```

```go
// GOOD — initialize slice rỗng thay vì nil
package main

import (
    "encoding/json"
    "fmt"
)

type Response struct {
    Items []string `json:"items"`
    Tags  []string `json:"tags"`
}

func NewResponse() Response {
    return Response{
        Items: []string{}, // empty, không phải nil
        Tags:  []string{}, // empty, không phải nil
    }
}

func main() {
    resp := NewResponse()
    resp.Items = append(resp.Items, "apple", "banana")

    data, _ := json.Marshal(resp)
    fmt.Println(string(data))
    // Output: {"items":["apple","banana"],"tags":[]}
    // Client nhận [] cho Tags — đúng hành vi mong đợi

    // Kiểm tra sự khác biệt
    type Demo struct {
        NilSlice   []int `json:"nil_slice"`
        EmptySlice []int `json:"empty_slice"`
    }

    d := Demo{
        NilSlice:   nil,
        EmptySlice: []int{},
    }
    out, _ := json.Marshal(d)
    fmt.Println(string(out))
    // {"nil_slice":null,"empty_slice":[]}
}
```

Khi design API response struct, luôn initialize slice fields với `make([]T, 0)` hoặc `[]T{}` trong constructor. Hoặc dùng `json:",omitempty"` nếu muốn bỏ hoàn toàn field khi rỗng.

---

## 13. time.Time Zero Value Không Phải "Không Có Thời Gian"

Khi bạn khai báo `var t time.Time` mà không assign, zero value là `0001-01-01 00:00:00 UTC` — ngày 1 tháng 1 năm 1. Developer C# quen với `DateTime.MinValue` (tương tự) nhưng thường dùng `DateTime?` (nullable) để biểu diễn "chưa có giá trị". Trong Go, bạn phải kiểm tra bằng `t.IsZero()`.

```go
// BAD — check sai zero value của time.Time
package main

import (
    "fmt"
    "time"
)

type Event struct {
    Name      string
    StartTime time.Time
    EndTime   time.Time // optional — chưa kết thúc nếu zero
}

func main() {
    e := Event{
        Name:      "Workshop",
        StartTime: time.Now(),
        // EndTime không set → zero value
    }

    // BUG: check sai — time.Time zero không phải nil
    // C# developer nghĩ e.EndTime == nil hoặc e.EndTime == default(DateTime)
    // Không có syntax như vậy trong Go

    // Cách sai: in ra mà không check
    fmt.Printf("Event '%s' kết thúc lúc: %s\n", e.Name, e.EndTime)
    // Output: Event 'Workshop' kết thúc lúc: 0001-01-01 00:00:00 +0000 UTC
    // Nhìn rất lạ và sai!

    // Cũng sai: compare với time.Time{} (dễ bỏ sót)
    if e.EndTime == (time.Time{}) {
        fmt.Println("Chưa kết thúc (cách này hoạt động nhưng không idiomatic)")
    }
}
```

```go
// GOOD — dùng IsZero() và *time.Time cho optional time
package main

import (
    "fmt"
    "time"
)

type Event struct {
    Name      string
    StartTime time.Time
    EndTime   *time.Time // pointer = nullable, nil = "chưa có giá trị"
}

func (e Event) IsOngoing() bool {
    return e.EndTime == nil || e.EndTime.IsZero()
}

func (e Event) Duration() (time.Duration, bool) {
    if e.IsOngoing() {
        return 0, false
    }
    return e.EndTime.Sub(e.StartTime), true
}

func main() {
    now := time.Now()
    end := now.Add(2 * time.Hour)

    ongoing := Event{Name: "Workshop", StartTime: now}
    finished := Event{Name: "Meeting", StartTime: now, EndTime: &end}

    fmt.Println("Workshop ongoing:", ongoing.IsOngoing()) // true
    fmt.Println("Meeting ongoing:", finished.IsOngoing()) // false

    if d, ok := finished.Duration(); ok {
        fmt.Printf("Meeting kéo dài: %v\n", d) // 2h0m0s
    }

    // Kiểm tra zero value đúng cách
    var t time.Time
    fmt.Println("Is zero:", t.IsZero()) // true
    fmt.Println("Zero value:", t.Format(time.RFC3339)) // 0001-01-01T00:00:00Z
}
```

---

## 14. Timezone Handling — Luôn Dùng UTC Để Lưu Trữ

`time.Now()` trong Go trả về thời gian theo **local timezone của máy đang chạy**. Trong production, server thường chạy UTC, nhưng developer machine thường chạy local timezone. Code chạy tốt trên máy bạn nhưng có thể lỗi khi deploy. Đây là nguồn gốc của hàng loạt bug timezone kinh điển.

```go
// BAD — dùng local time, timezone tùy thuộc vào máy chạy
package main

import (
    "fmt"
    "time"
)

type Order struct {
    ID        int
    CreatedAt time.Time
}

func createOrder(id int) Order {
    return Order{
        ID:        id,
        CreatedAt: time.Now(), // local timezone! Khác nhau trên các máy
    }
}

func main() {
    order := createOrder(123)
    fmt.Println("Order created:", order.CreatedAt)
    // Trên máy VN: 2024-01-15 08:30:00 +0700 ICT
    // Trên server UTC: 2024-01-15 01:30:00 +0000 UTC
    // Khi parse và so sánh: có thể sai nếu không consistent
}
```

```go
// GOOD — luôn dùng UTC, chỉ convert sang local khi hiển thị cho user
package main

import (
    "fmt"
    "time"
)

type Order struct {
    ID        int
    CreatedAt time.Time // luôn lưu UTC
}

func createOrder(id int) Order {
    return Order{
        ID:        id,
        CreatedAt: time.Now().UTC(), // explicit UTC
    }
}

// Chỉ convert sang local timezone khi display
func displayTime(t time.Time, userTZ string) string {
    loc, err := time.LoadLocation(userTZ)
    if err != nil {
        return t.UTC().Format(time.RFC3339)
    }
    return t.In(loc).Format("02/01/2006 15:04:05 MST")
}

func main() {
    order := createOrder(123)
    fmt.Println("UTC stored:", order.CreatedAt.Format(time.RFC3339))

    // Display theo timezone của user
    fmt.Println("VN display:", displayTime(order.CreatedAt, "Asia/Ho_Chi_Minh"))
    fmt.Println("US display:", displayTime(order.CreatedAt, "America/New_York"))

    // Parse time string luôn specify timezone
    t, err := time.Parse(time.RFC3339, "2024-01-15T08:30:00+07:00")
    if err != nil {
        fmt.Println("Parse error:", err)
        return
    }
    fmt.Println("Parsed UTC:", t.UTC().Format(time.RFC3339))
}
```

Quy tắc vàng: lưu UTC trong database và internal state, convert sang local timezone chỉ khi hiển thị cho user. Dùng `time.RFC3339` hoặc Unix timestamp khi serialize.

---

## 15. init() Order — Không Thể Đoán Được Giữa Các Package

Go cho phép mỗi file có một hoặc nhiều hàm `init()`. Các `init()` trong cùng file chạy theo thứ tự khai báo. Các `init()` giữa các file trong cùng package chạy theo thứ tự **file được compiler xử lý** (thường là alphabetical nhưng không được đảm bảo). Giữa các package, thứ tự phụ thuộc vào dependency graph.

```go
// BAD — init() phụ thuộc vào thứ tự giữa packages
// File: config/config.go
package config

var GlobalConfig map[string]string

func init() {
    // Assume database package chưa được init
    GlobalConfig = map[string]string{
        "db_host": "localhost",
    }
}

// File: database/database.go
package database

import "config" // import config

var DB *Connection

func init() {
    // BUG tiềm ẩn: nếu config.init() chưa chạy, GlobalConfig là nil
    host := config.GlobalConfig["db_host"]
    DB = connect(host)
}

// Thứ tự init(): config.init() → database.init() (đúng theo dependency)
// Nhưng nếu refactor làm thay đổi dependency graph, thứ tự có thể bị ảnh hưởng
```

```go
// GOOD — init() chỉ để đăng ký, không phụ thuộc vào state từ package khác
package main

import (
    "fmt"
    "sync"
)

// Dùng init() cho việc đơn giản, không phụ thuộc package khác
var registry = make(map[string]func() error)
var mu sync.RWMutex

func Register(name string, fn func() error) {
    mu.Lock()
    defer mu.Unlock()
    registry[name] = fn
}

func init() {
    // OK: đăng ký vào registry — không phụ thuộc state bên ngoài
    Register("default", func() error { return nil })
}

// TỐTHƠN: explicit initialization function thay vì init()
type App struct {
    config map[string]string
    db     string
}

func NewApp(configPath string) (*App, error) {
    // Explicit, testable, controllable — không phải magic init()
    cfg, err := loadConfig(configPath)
    if err != nil {
        return nil, fmt.Errorf("NewApp: %w", err)
    }

    db, err := connectDB(cfg["db_host"])
    if err != nil {
        return nil, fmt.Errorf("NewApp: connect db: %w", err)
    }

    return &App{config: cfg, db: db}, nil
}

func loadConfig(path string) (map[string]string, error) {
    return map[string]string{"db_host": "localhost"}, nil
}

func connectDB(host string) (string, error) {
    return "connected to " + host, nil
}

func main() {
    app, err := NewApp("config.yaml")
    if err != nil {
        fmt.Println("Khởi động thất bại:", err)
        return
    }
    fmt.Println("App started, db:", app.db)
}
```

---

## 16. Defer Trong Loop — Chỉ Chạy Khi Hàm Return

`defer` không chạy khi kết thúc iteration — nó chạy khi **hàm chứa nó return**. Nếu bạn mở file hoặc lock mutex trong loop với `defer`, tất cả deferred calls sẽ tích lũy và chỉ chạy khi hàm kết thúc. Với loop lớn, bạn sẽ giữ nhiều file handle hoặc lock cùng lúc, gây cạn tài nguyên.

```go
// BAD — defer trong loop không close file sau mỗi iteration
package main

import (
    "fmt"
    "os"
)

func processAllFilesBad(paths []string) error {
    for _, path := range paths {
        f, err := os.Open(path)
        if err != nil {
            return err
        }
        defer f.Close() // BUG: close chỉ chạy khi processAllFiles return!
        // Nếu paths có 1000 file, 1000 file handle mở cùng lúc
        // Hệ điều hành giới hạn số file handle → lỗi "too many open files"

        // process file...
        fmt.Println("Processing:", path)
    }
    return nil
}
```

```go
// GOOD — đóng resource trong cùng iteration bằng anonymous function
package main

import (
    "fmt"
    "os"
)

func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("mở file %s: %w", path, err)
    }
    defer f.Close() // OK: hàm này return sau khi xử lý xong một file

    // process f...
    info, _ := f.Stat()
    fmt.Printf("File %s: %d bytes\n", path, info.Size())
    return nil
}

func processAllFilesGood(paths []string) error {
    for _, path := range paths {
        // Đóng gói trong hàm riêng — defer chạy đúng lúc
        if err := processFile(path); err != nil {
            return err
        }
    }
    return nil
}

// Hoặc dùng anonymous function inline
func processAllFilesInline(paths []string) error {
    for _, path := range paths {
        err := func(p string) error {
            f, err := os.Open(p)
            if err != nil {
                return err
            }
            defer f.Close() // chạy khi anonymous func này return

            info, _ := f.Stat()
            fmt.Printf("File %s: %d bytes\n", p, info.Size())
            return nil
        }(path)

        if err != nil {
            return err
        }
    }
    return nil
}

func main() {
    // Tạo file test
    os.WriteFile("test1.txt", []byte("hello"), 0644)
    os.WriteFile("test2.txt", []byte("world"), 0644)
    defer os.Remove("test1.txt")
    defer os.Remove("test2.txt")

    paths := []string{"test1.txt", "test2.txt"}
    if err := processAllFilesGood(paths); err != nil {
        fmt.Println("Lỗi:", err)
    }
}
```

---

## 17. http.DefaultClient Không Có Timeout

`http.DefaultClient` trong Go **không có timeout**. Một request có thể chờ mãi mãi nếu server không respond. Trong C#, `HttpClient` cũng cần set timeout nhưng default là 100 giây — đủ lâu để không gây vấn đề ngay, nhưng Go không có default nào cả.

Trong production, không có timeout là thảm họa: goroutine tích lũy, memory tăng, service bị treo. Đây là lỗi ngây thơ nhất nhưng có hậu quả nghiêm trọng nhất.

```go
// BAD — dùng http.DefaultClient hoặc không set timeout
package main

import (
    "fmt"
    "io"
    "net/http"
)

func fetchBad(url string) (string, error) {
    // http.Get dùng http.DefaultClient — không có timeout!
    resp, err := http.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    return string(body), nil
}

// Cũng BAD — tạo client nhưng không set timeout
func fetchAlsoBad(url string) (string, error) {
    client := &http.Client{} // timeout = 0 = không giới hạn
    resp, err := client.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    return string(body), nil
}

func main() {
    // Giả sử gọi server chậm hoặc dead
    result, err := fetchBad("http://httpbin.org/delay/10")
    if err != nil {
        fmt.Println("Lỗi:", err)
        return
    }
    fmt.Println(result)
    // Đợi 10 giây hoặc mãi mãi nếu server không respond
}
```

```go
// GOOD — luôn tạo client riêng với timeout
package main

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "time"
)

// HTTP client với timeout — tạo một lần, dùng nhiều lần (thread-safe)
var httpClient = &http.Client{
    Timeout: 30 * time.Second, // timeout tổng cộng cho toàn bộ request
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

func fetch(ctx context.Context, url string) (string, error) {
    // Tạo request với context — cho phép cancel từ bên ngoài
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return "", fmt.Errorf("tạo request: %w", err)
    }

    resp, err := httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("thực hiện request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("server trả về status %d", resp.StatusCode)
    }

    // Giới hạn kích thước response để tránh OOM
    body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // max 10MB
    if err != nil {
        return "", fmt.Errorf("đọc response body: %w", err)
    }

    return string(body), nil
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := fetch(ctx, "https://httpbin.org/json")
    if err != nil {
        fmt.Println("Lỗi:", err)
        return
    }
    fmt.Println("OK, received", len(result), "bytes")
}
```

Hai layer timeout: `http.Client.Timeout` là deadline tổng thể cho request, `context.WithTimeout` cho phép caller cancel sớm hơn. Trong production service, luôn dùng cả hai.

---

## Bảng Tổng Hợp — Common Issues

| Gotcha | C# Behavior | Go Behavior | Fix |
|--------|-------------|-------------|-----|
| `nil == nil` với interface | Luôn `true` | `false` nếu typed nil | Trả về `nil` trực tiếp, không dùng typed nil pointer |
| Gán slice | Shared reference | Shared underlying array | `copy()` hoặc `append([]T(nil), src...)` |
| Map concurrent | Race condition thầm lặng | `panic: concurrent map writes` | `sync.RWMutex` hoặc `sync.Map` |
| Loop variable goroutine | Lambda capture (C# cũng có) | Capture by reference → tất cả dùng giá trị cuối | `v := v` hoặc pass parameter |
| String/[]byte convert | String.Encoding.GetBytes() | Allocation mới mỗi lần | Dùng `strings.Builder`, minimize conversion |
| Integer overflow | `OverflowException` (checked) | Silent wrap-around | Kiểm tra thủ công, dùng `int64` |
| `:=` shadowing | Compile error | Tạo biến mới, che outer | Dùng `=` cho outer, `golangci-lint` |
| Goroutine vs Task | Built-in result, cancellation, propagation | Chỉ là function concurrent | Channel + context.Context |
| `while` loop | `while`, `do-while` | Không có | `for condition { }`, `for { break }` |
| Bỏ qua error | Exception crash app | Lỗi im lặng | Xử lý tất cả error, tránh `_` |
| Struct comparison | Value semantics tự động | Chỉ comparable nếu tất cả fields comparable | `reflect.DeepEqual`, thiết kế lại struct |
| JSON nil slice | `null` | `null` | Initialize với `[]T{}` |
| `time.Time` zero | `DateTime.MinValue` | `0001-01-01 00:00:00 UTC` | `t.IsZero()`, dùng `*time.Time` |
| Timezone | Explicit TimeZoneInfo | `time.Now()` = local timezone | `time.Now().UTC()` |
| `init()` order | Static constructor rõ ràng | Không đảm bảo giữa packages | Explicit `NewXxx()` constructor |
| `defer` trong loop | `using` chạy ngay cuối block | Tích lũy đến khi hàm return | Đặt trong hàm riêng |
| HTTP client | Default 100s timeout | Không có timeout! | Luôn set `Timeout` |

---

## Bài Tập Thực Hành

### Bài Tập 1: Fix Các Bugs Trong Code Cho Sẵn

Đoạn code dưới đây có **ít nhất 5 bugs** từ các gotcha đã học. Tìm và sửa tất cả.

```go
// BuggyCode — TÌM VÀ SỬA CÁC LỖI
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
)

type Product struct {
    ID    int
    Name  string
    Tags  []string
    Price float64
}

// Bug 1: trả về typed nil qua interface
func validateProduct(p Product) error {
    var err *fmt.Stringer // sai type, chỉ để minh họa pattern
    if p.Name == "" {
        // err = ... (không được assign)
        _ = err
    }
    return nil
}

// Bug 2: map không thread-safe
var productCache = make(map[int]Product)

func cacheProduct(wg *sync.WaitGroup, p Product) {
    defer wg.Done()
    productCache[p.ID] = p // concurrent write!
}

// Bug 3: goroutine capture loop variable
func processProducts(products []Product) {
    var wg sync.WaitGroup
    for _, p := range products {
        wg.Add(1)
        go func() {
            defer wg.Done()
            fmt.Printf("Processing: %s\n", p.Name) // capture p by reference
        }()
    }
    wg.Wait()
}

// Bug 4: nil slice trong JSON response
type APIResponse struct {
    Products []Product `json:"products"`
    Errors   []string  `json:"errors"`
}

func buildResponse(products []Product) APIResponse {
    return APIResponse{Products: products}
    // Errors không được initialize → nil → "errors":null trong JSON
}

// Bug 5: HTTP client không có timeout
func fetchExternalData(url string) ([]byte, error) {
    resp, err := http.Get(url) // dùng DefaultClient!
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return nil, nil
}

func main() {
    products := []Product{
        {ID: 1, Name: "Laptop", Tags: []string{"tech"}, Price: 999.99},
        {ID: 2, Name: "Mouse", Tags: []string{"tech", "peripheral"}, Price: 29.99},
        {ID: 3, Name: "Desk", Price: 299.99}, // Tags là nil!
    }

    // Test JSON output
    resp := buildResponse(products)
    data, _ := json.Marshal(resp)
    fmt.Println(string(data))

    // Test concurrent caching
    var wg sync.WaitGroup
    for _, p := range products {
        wg.Add(1)
        go cacheProduct(&wg, p)
    }
    wg.Wait()
    fmt.Println("Cached:", len(productCache))

    // Test goroutine
    processProducts(products)
}
```

**Lời giải:**

```go
// SOLUTION — Đã sửa tất cả 5 bugs
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
)

type Product struct {
    ID    int      `json:"id"`
    Name  string   `json:"name"`
    Tags  []string `json:"tags"`
    Price float64  `json:"price"`
}

// Fix Bug 1: Không dùng typed nil — trả về error trực tiếp
func validateProduct(p Product) error {
    if p.Name == "" {
        return fmt.Errorf("tên sản phẩm không được trống")
    }
    if p.Price < 0 {
        return fmt.Errorf("giá không được âm: %f", p.Price)
    }
    return nil // untyped nil — đúng
}

// Fix Bug 2: Bảo vệ map bằng RWMutex
var (
    productCache   = make(map[int]Product)
    productCacheMu sync.RWMutex
)

func cacheProduct(wg *sync.WaitGroup, p Product) {
    defer wg.Done()
    productCacheMu.Lock()
    defer productCacheMu.Unlock()
    productCache[p.ID] = p // thread-safe
}

// Fix Bug 3: Truyền p làm parameter để tránh capture
func processProducts(products []Product) {
    var wg sync.WaitGroup
    for _, p := range products {
        wg.Add(1)
        p := p // Fix: tạo biến mới cho mỗi iteration (hoặc pass parameter)
        go func() {
            defer wg.Done()
            fmt.Printf("Processing: %s (price: %.2f)\n", p.Name, p.Price)
        }()
    }
    wg.Wait()
}

// Fix Bug 4: Initialize Errors slice để tránh null trong JSON
type APIResponse struct {
    Products []Product `json:"products"`
    Errors   []string  `json:"errors"`
}

func buildResponse(products []Product) APIResponse {
    // Đảm bảo Products không nil
    if products == nil {
        products = []Product{}
    }

    // Đảm bảo Tags trong mỗi product không nil
    for i := range products {
        if products[i].Tags == nil {
            products[i].Tags = []string{} // [] thay vì null trong JSON
        }
    }

    return APIResponse{
        Products: products,
        Errors:   []string{}, // Fix: initialize empty slice
    }
}

// Fix Bug 5: Client với timeout
var apiClient = &http.Client{
    Timeout: 30 * time.Second,
}

func fetchExternalData(url string) ([]byte, error) {
    resp, err := apiClient.Get(url) // Fix: client riêng có timeout
    if err != nil {
        return nil, fmt.Errorf("fetch %s: %w", url, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("status không mong đợi: %d", resp.StatusCode)
    }

    data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
    if err != nil {
        return nil, fmt.Errorf("đọc response: %w", err)
    }
    return data, nil
}

func main() {
    products := []Product{
        {ID: 1, Name: "Laptop", Tags: []string{"tech"}, Price: 999.99},
        {ID: 2, Name: "Mouse", Tags: []string{"tech", "peripheral"}, Price: 29.99},
        {ID: 3, Name: "Desk", Price: 299.99}, // Tags nil → sẽ được fix thành []
    }

    // Test validation
    badProduct := Product{Name: "", Price: 100}
    if err := validateProduct(badProduct); err != nil {
        fmt.Println("Validation error:", err)
    }

    // Test JSON output — tất cả fields phải là [] không phải null
    resp := buildResponse(products)
    data, _ := json.MarshalIndent(resp, "", "  ")
    fmt.Println(string(data))
    // Verify: "tags": [] và "errors": [] (không có null)

    // Test concurrent caching
    var wg sync.WaitGroup
    for _, p := range products {
        wg.Add(1)
        go cacheProduct(&wg, p)
    }
    wg.Wait()
    fmt.Println("Cached products:", len(productCache))

    // Test goroutine processing
    fmt.Println("\n--- Processing ---")
    processProducts(products)
}
```

### Bài Tập 2: Implement Thread-Safe Cache Với Expiry

Viết một generic cache thread-safe có TTL (time-to-live), xử lý đúng tất cả các gotcha đã học.

**Yêu cầu:**
- Thread-safe (nhiều goroutine đọc/ghi đồng thời)
- TTL: item hết hạn sau thời gian nhất định
- Không leak goroutine khi cleanup
- JSON serializable (empty map → `{}` không phải `null`)

```go
// SOLUTION — Thread-Safe TTL Cache
package main

import (
    "encoding/json"
    "fmt"
    "sync"
    "time"
)

type cacheEntry[V any] struct {
    Value     V
    ExpiresAt time.Time
}

func (e cacheEntry[V]) isExpired() bool {
    return time.Now().UTC().After(e.ExpiresAt)
}

type TTLCache[K comparable, V any] struct {
    mu      sync.RWMutex
    entries map[K]cacheEntry[V]
    ttl     time.Duration
    stopCh  chan struct{}
}

func NewTTLCache[K comparable, V any](ttl time.Duration) *TTLCache[K, V] {
    c := &TTLCache[K, V]{
        entries: make(map[K]cacheEntry[V]), // initialize, không phải nil
        ttl:     ttl,
        stopCh:  make(chan struct{}),
    }

    // Cleanup goroutine — chạy background để xóa expired entries
    go c.cleanupLoop()

    return c
}

func (c *TTLCache[K, V]) cleanupLoop() {
    ticker := time.NewTicker(c.ttl / 2) // cleanup 2x per TTL period
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            c.evictExpired()
        case <-c.stopCh:
            return // goroutine kết thúc sạch sẽ — không leak
        }
    }
}

func (c *TTLCache[K, V]) evictExpired() {
    c.mu.Lock()
    defer c.mu.Unlock()

    for k, entry := range c.entries {
        if entry.isExpired() {
            delete(c.entries, k)
        }
    }
}

func (c *TTLCache[K, V]) Set(key K, value V) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.entries[key] = cacheEntry[V]{
        Value:     value,
        ExpiresAt: time.Now().UTC().Add(c.ttl), // UTC!
    }
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, ok := c.entries[key]
    if !ok || entry.isExpired() {
        var zero V
        return zero, false
    }
    return entry.Value, true
}

func (c *TTLCache[K, V]) Delete(key K) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.entries, key)
}

func (c *TTLCache[K, V]) Len() int {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return len(c.entries)
}

// Stop dừng cleanup goroutine — quan trọng để tránh goroutine leak
func (c *TTLCache[K, V]) Stop() {
    close(c.stopCh)
}

// MarshalJSON custom để serialize cache state
func (c *TTLCache[K, V]) MarshalJSON() ([]byte, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    // Chỉ serialize non-expired entries
    result := make(map[K]V) // empty map → {} không phải null
    for k, entry := range c.entries {
        if !entry.isExpired() {
            result[k] = entry.Value
        }
    }
    return json.Marshal(result)
}

func main() {
    // Test basic operations
    cache := NewTTLCache[string, int](500 * time.Millisecond)
    defer cache.Stop() // cleanup goroutine khi xong

    // Set values
    cache.Set("a", 1)
    cache.Set("b", 2)
    cache.Set("c", 3)

    fmt.Println("Initial size:", cache.Len()) // 3

    // Get existing value
    if v, ok := cache.Get("a"); ok {
        fmt.Println("Get 'a':", v) // 1
    }

    // Test concurrent access
    var wg sync.WaitGroup
    for i := 0; i < 50; i++ {
        wg.Add(2)
        i := i
        go func() {
            defer wg.Done()
            cache.Set(fmt.Sprintf("key%d", i), i)
        }()
        go func() {
            defer wg.Done()
            cache.Get(fmt.Sprintf("key%d", i))
        }()
    }
    wg.Wait()
    fmt.Println("After concurrent ops:", cache.Len())

    // Test JSON serialization
    data, err := json.Marshal(cache)
    if err != nil {
        fmt.Println("Marshal error:", err)
    } else {
        fmt.Printf("JSON (first 100 chars): %.100s...\n", string(data))
        // {} hoặc {"a":1,"b":2,...} — không bao giờ null
    }

    // Test TTL expiry
    shortCache := NewTTLCache[string, string](100 * time.Millisecond)
    defer shortCache.Stop()

    shortCache.Set("temp", "value")
    if v, ok := shortCache.Get("temp"); ok {
        fmt.Println("Before expiry:", v) // value
    }

    time.Sleep(200 * time.Millisecond) // chờ TTL hết hạn

    if _, ok := shortCache.Get("temp"); !ok {
        fmt.Println("After expiry: entry đã expired") // đúng
    }
}
```

---

## Best Practices Tổng Hợp

Sau khi học qua 17 gotcha, có một số nguyên tắc tổng quát giúp bạn viết Go đúng từ đầu.

**Về error handling:** Go không có exception vì lý do thiết kế rõ ràng — error là giá trị, buộc developer phải quyết định ở mỗi điểm gọi. Hãy embrace điều này thay vì chống lại. Dùng `fmt.Errorf("context: %w", err)` để wrap error với context, dùng `errors.Is()` và `errors.As()` để unwrap khi cần. Không bao giờ discard error trừ khi bạn có lý do rõ ràng và document trong comment.

**Về concurrency:** Model mental đúng là: goroutine rẻ (2KB stack, tạo goroutine nhanh như function call) nhưng không phải miễn phí. Channel là cơ chế đồng bộ ưa tiên — "don't communicate by sharing memory, share memory by communicating". Mutex phù hợp khi cần protect struct có nhiều fields cùng lúc. Luôn chạy `go test -race ./...` trước khi commit.

**Về memory và allocation:** Biết khi nào allocation xảy ra giúp bạn viết code nhanh. String-to-[]byte conversion là allocation. Interface boxing là allocation. Closure capture là allocation. Không cần obsess về điều này sớm — profile trước khi optimize. Nhưng trong hot path (loop chạy triệu lần), minimize allocation bằng `sync.Pool` hoặc reuse buffer.

**Về nil:** Luôn nghĩ về zero value của mỗi type trước khi viết. `var s []string` là nil slice (không phải empty). `var m map[string]int` là nil map (đọc được nhưng ghi sẽ panic). `var p *MyStruct` là nil pointer. Initialize explicit khi cần behavior rõ ràng. Dùng `make()` cho slice và map khi bạn biết sẽ thêm dữ liệu vào.

**Về testing và tooling:** `go vet ./...` phát hiện nhiều bug đã học ở trên. `go test -race ./...` bắt concurrent access. `golangci-lint` với config phù hợp bắt shadowing, ineffectual assignment, và nhiều hơn nữa. Đây không phải optional — integrate vào CI từ ngày đầu.

---

## Next

[18. Testing in Go — Unit Tests, Table-Driven Tests, Mocks →](./18-testing-in-go.md)

*Bao gồm: `testing` package, table-driven tests, `testify`, mock interfaces, test coverage, benchmark tests, và cách structure test files trong Go project thực tế.*
