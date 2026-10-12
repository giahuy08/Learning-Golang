# 📚 Defer, Panic & Recover — Quản lý lỗi kiểu Go (dành cho C# developer)

---

Go không có exception. Không có `try/catch`. Không có `throw`. Đây là điều đầu tiên bạn cần chấp nhận — và hiểu tại sao.

Trong C#, exception là cơ chế kiểm soát luồng chính cho lỗi runtime. Go xem đó là anti-pattern vì nó làm ẩn đi flow thực sự của chương trình. Thay vào đó, Go dùng ba cơ chế riêng biệt, mỗi cái phục vụ một mục đích khác nhau: `defer` để cleanup, `panic` cho lỗi không thể phục hồi, và `recover` để bắt panic ở tầng cao nhất. Hiểu đúng ba cái này sẽ làm bạn viết Go như Go — không phải như "C# dịch sang Go".

---

## 1. defer — Lên lịch cleanup, không phải "chạy ngay"

`defer` là từ khóa đặt một lời gọi hàm vào một hàng đợi. Hàm được lên lịch đó **sẽ chạy khi function hiện tại kết thúc** — dù kết thúc bình thường, `return` giữa chừng, hay bị `panic`. Nó không chạy ngay khi bạn viết `defer`.

Ý tưởng cốt lõi: bạn mở một resource và ngay lập tức viết code đóng nó — để không bao giờ quên. Sau đó tiếp tục viết logic chính mà không lo cleanup.

```go
// BAD — C# developer mới học Go thường viết thế này
// Quên đóng file nếu có return sớm
package main

import (
    "fmt"
    "os"
)

func readFileBad(name string) error {
    f, err := os.Open(name)
    if err != nil {
        return err
    }

    // ... xử lý file ...
    data := make([]byte, 100)
    _, err = f.Read(data)
    if err != nil {
        return err // f không được đóng ở đây!
    }

    fmt.Println(string(data))
    f.Close() // chỉ đóng nếu đi đến đây — dễ bị bỏ sót
    return nil
}
```

```go
// GOOD — defer đảm bảo f.Close() luôn được gọi, dù return ở đâu
package main

import (
    "fmt"
    "os"
)

func readFileGood(name string) error {
    f, err := os.Open(name)
    if err != nil {
        return err
    }
    defer f.Close() // đặt ngay sau khi mở — không bao giờ quên

    data := make([]byte, 100)
    _, err = f.Read(data)
    if err != nil {
        return err // f.Close() vẫn được gọi sau return này
    }

    fmt.Println(string(data))
    return nil // f.Close() được gọi ở đây
}
```

### So sánh với C# using và try/finally

C# có ba cách làm cleanup tương đương. Hiểu khi nào Go dùng `defer` thay vì khi nào C# dùng cái gì giúp bạn đọc được intent của code Go.

**`using` trong C#** tương đương gần nhất với `defer` cho một resource duy nhất:

```csharp
// C# — using statement
using (var f = File.OpenRead(name))
{
    // f.Dispose() được gọi tự động khi ra khỏi block
    var buffer = new byte[100];
    f.Read(buffer, 0, 100);
    Console.WriteLine(Encoding.UTF8.GetString(buffer));
}
```

```go
// Go tương đương
f, err := os.Open(name)
if err != nil { return err }
defer f.Close()
// Tiếp tục dùng f...
```

Sự khác biệt quan trọng: `using` trong C# là scoped theo block `{}`. `defer` trong Go là scoped theo **function**. Trong Go, bạn không tạo ra nested scope mới chỉ để cleanup — function chính là scope.

**`try/finally` trong C#** tương đương khi bạn cần cleanup kể cả khi có exception:

```csharp
// C# — try/finally
var conn = db.GetConnection();
try
{
    conn.Open();
    // làm gì đó với conn
}
finally
{
    conn.Close(); // luôn chạy
}
```

```go
// Go tương đương — sạch hơn, ít nesting hơn
conn, err := db.GetConnection()
if err != nil { return err }
defer conn.Close()
// làm gì đó với conn — không cần try block
```

Go dùng `defer` thay cho cả `using` và `try/finally`. Không phải vì Go lười thiết kế — mà vì Go muốn: "mở resource và cleanup ở cùng một chỗ trong code, ngay cạnh nhau, dễ review". Trong C# với `try/finally`, đôi khi cleanup code cách xa open code hàng chục dòng.

---

## 2. Thứ tự thực thi defer — LIFO (Last In, First Out)

Khi một function có nhiều `defer`, chúng chạy theo thứ tự **ngược lại** với thứ tự khai báo. Cái khai báo cuối cùng chạy trước. Đây là hành vi có chủ ý — nếu bạn mở A rồi mở B (B phụ thuộc vào A), thì bạn phải đóng B trước rồi mới đóng A.

```go
package main

import "fmt"

func main() {
    fmt.Println("=== Demo LIFO defer ===")

    defer fmt.Println("defer 1 — khai báo đầu tiên, chạy cuối cùng")
    defer fmt.Println("defer 2 — khai báo thứ hai, chạy áp chót")
    defer fmt.Println("defer 3 — khai báo cuối cùng, chạy đầu tiên")

    fmt.Println("Logic chính đang chạy...")
}

// Output:
// === Demo LIFO defer ===
// Logic chính đang chạy...
// defer 3 — khai báo cuối cùng, chạy đầu tiên
// defer 2 — khai báo thứ hai, chạy áp chót
// defer 1 — khai báo đầu tiên, chạy cuối cùng
```

Ví dụ thực tế hơn — lock và unlock mutex theo đúng thứ tự:

```go
package main

import (
    "fmt"
    "sync"
)

func processWithLocks(mu1, mu2 *sync.Mutex) {
    // Lock theo thứ tự 1, 2
    mu1.Lock()
    defer mu1.Unlock() // unlock 1 chạy SAU unlock 2 (LIFO)

    mu2.Lock()
    defer mu2.Unlock() // unlock 2 chạy TRƯỚC unlock 1

    fmt.Println("Cả hai mutex đã được lock, đang xử lý...")
    // Khi function kết thúc: mu2.Unlock() chạy trước, mu1.Unlock() chạy sau
    // Thứ tự đúng để tránh deadlock
}
```

---

## 3. Các use case phổ biến của defer

Trong project thực tế, bạn sẽ thấy `defer` xuất hiện ở những pattern này đều đặn:

**Đóng file sau khi mở:**
```go
package main

import (
    "fmt"
    "os"
)

func writeLog(filename, message string) error {
    f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("mở file thất bại: %w", err)
    }
    defer f.Close() // ngay sau os.OpenFile — không bao giờ quên

    _, err = fmt.Fprintln(f, message)
    return err
}
```

**Unlock mutex sau khi lock:**
```go
package main

import (
    "fmt"
    "sync"
)

type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock() // ngay sau Lock — luôn unlock dù có panic hay không

    c.count++ // code bên dưới viết tự nhiên, không lo quên unlock
    fmt.Printf("Count: %d\n", c.count)
}
```

**Đóng database connection hoặc rows:**
```go
// Trong thực tế với database/sql
// rows, err := db.Query("SELECT ...")
// if err != nil { return err }
// defer rows.Close() // ngay sau Query — không bao giờ leak rows

// Minh họa pattern với mock
package main

import "fmt"

type MockRows struct{ closed bool }

func (r *MockRows) Close() { r.closed = true; fmt.Println("Rows đã đóng") }
func (r *MockRows) Next() bool { return false }

func queryUsers() error {
    rows := &MockRows{}
    defer rows.Close() // đóng dù có lỗi hay không trong quá trình đọc

    for rows.Next() {
        // đọc từng row
    }
    return nil
}

func main() { queryUsers() }
```

---

## 4. defer trong vòng lặp — Pattern SAI phổ biến nhất

Đây là bẫy mà hầu hết C# developer mắc phải khi mới học Go. Trong C#, `using` bên trong vòng lặp `foreach` tự động dispose sau mỗi iteration. Trong Go, `defer` không hoạt động như vậy — tất cả defers tích lũy lại và chỉ chạy khi **function** kết thúc, không phải khi vòng lặp kết thúc.

```go
// BAD — defer trong vòng lặp tích lũy, không release sau mỗi iteration
// Nếu files là slice 1000 phần tử, có 1000 file handles mở cùng lúc
package main

import (
    "fmt"
    "os"
)

func processFilesBAD(filenames []string) error {
    for _, name := range filenames {
        f, err := os.Open(name)
        if err != nil {
            return err
        }
        defer f.Close() // SAI: f.Close() chỉ chạy khi processFilesBAD() kết thúc
                        // Không phải khi mỗi vòng lặp kết thúc
        
        // xử lý f...
        fmt.Println("Đang xử lý:", name)
    }
    return nil
    // f.Close() của TẤT CẢ files mới chạy ở đây — quá muộn!
}
```

```go
// GOOD — dùng anonymous function (closure) để tạo scope mới cho mỗi iteration
package main

import (
    "fmt"
    "os"
)

func processFilesGOOD(filenames []string) error {
    for _, name := range filenames {
        // Anonymous function tạo ra một function scope mới
        // defer bên trong sẽ chạy khi anonymous function này kết thúc
        err := func(filename string) error {
            f, err := os.Open(filename)
            if err != nil {
                return err
            }
            defer f.Close() // chạy khi anonymous function kết thúc — đúng rồi!

            fmt.Println("Đang xử lý:", filename)
            // xử lý f...
            return nil
        }(name) // gọi ngay lập tức với name hiện tại

        if err != nil {
            return err
        }
    }
    return nil
}

func main() {
    // Tạo file test
    files := []string{}
    for i := 0; i < 3; i++ {
        name := fmt.Sprintf("test%d.txt", i)
        os.WriteFile(name, []byte("hello"), 0644)
        files = append(files, name)
    }
    processFilesGOOD(files)
    // Cleanup
    for _, f := range files { os.Remove(f) }
}
```

Cách fix thứ hai — tách ra thành helper function riêng (thường sạch hơn):

```go
package main

import (
    "fmt"
    "os"
)

// Tách logic xử lý một file thành function riêng
// defer trong đây chạy sau mỗi lần gọi processOneFile
func processOneFile(name string) error {
    f, err := os.Open(name)
    if err != nil {
        return err
    }
    defer f.Close() // đúng: chạy khi processOneFile kết thúc

    fmt.Println("Đang xử lý:", name)
    return nil
}

func processFilesClean(filenames []string) error {
    for _, name := range filenames {
        if err := processOneFile(name); err != nil {
            return err
        }
    }
    return nil
}
```

---

## 5. defer với Named Return Values — Pattern nâng cao

Go cho phép đặt tên cho return values. Khi kết hợp với `defer`, bạn có thể **sửa đổi giá trị trả về** ngay trước khi function thực sự return. Đây là pattern độc đáo của Go, không có tương đương trực tiếp trong C#.

```go
package main

import (
    "errors"
    "fmt"
)

// Named return value: err được đặt tên
// defer có thể đọc và ghi vào err này
func doOperation() (result int, err error) {
    // defer này chạy SAU khi return statement thực thi
    // nhưng TRƯỚC khi function thực sự trả về cho caller
    defer func() {
        if err != nil {
            // Thêm context vào error — không cần biết err là gì ở đây
            err = fmt.Errorf("doOperation thất bại: %w", err)
        }
    }()

    // Giả sử logic có thể lỗi
    result, err = compute()
    if err != nil {
        return 0, err // err đã được set, defer sẽ wrap nó
    }

    return result, nil
}

func compute() (int, error) {
    return 0, errors.New("lỗi tính toán")
}

func main() {
    result, err := doOperation()
    if err != nil {
        fmt.Println("Lỗi:", err)
        // Output: Lỗi: doOperation thất bại: lỗi tính toán
    } else {
        fmt.Println("Kết quả:", result)
    }
}
```

Pattern này thường dùng để wrap error với context mà không cần viết lại logic ở nhiều return point.

---

## 6. Arguments của defer được evaluate ngay lập tức — Gotcha quan trọng

Đây là điều gây nhầm lẫn nhất về `defer`. Khi bạn viết `defer f(x)`, **giá trị của `x` được tính toán ngay lúc đó** (lúc khai báo defer), không phải lúc defer thực sự chạy. Chỉ có việc *gọi hàm* là bị trì hoãn.

```go
// BAD — nghĩ rằng defer sẽ in ra giá trị i cuối cùng
package main

import "fmt"

func deferArgBAD() {
    i := 0
    defer fmt.Println("Giá trị i khi defer chạy:", i) // i được capture = 0 NGAY LÚC NÀY
    
    i = 100
    fmt.Println("i hiện tại:", i)
    // Output:
    // i hiện tại: 100
    // Giá trị i khi defer chạy: 0   <-- không phải 100!
}
```

```go
// GOOD — dùng closure (anonymous function không nhận argument) để capture biến by reference
package main

import "fmt"

func deferArgGOOD() {
    i := 0
    defer func() {
        // Closure này đọc i lúc NÓ CHẠY, không phải lúc defer khai báo
        fmt.Println("Giá trị i khi defer chạy:", i)
    }()

    i = 100
    fmt.Println("i hiện tại:", i)
    // Output:
    // i hiện tại: 100
    // Giá trị i khi defer chạy: 100  <-- đúng rồi!
}

func main() {
    fmt.Println("--- BAD ---")
    deferArgBAD()
    fmt.Println("--- GOOD ---")
    deferArgGOOD()
}
```

Quy tắc ghi nhớ: nếu bạn muốn defer nhìn thấy giá trị *tại thời điểm chạy*, dùng closure. Nếu bạn muốn defer dùng giá trị *tại thời điểm khai báo*, truyền argument trực tiếp.

---

## 7. panic — Không phải exception

`panic` trong Go không phải là exception của C#. Trong C#, bạn có thể `throw` một exception cho bất kỳ tình huống lỗi nào — từ "file not found" đến "user không đủ quyền". Go coi đó là design mistake. Trong Go, `panic` chỉ dành cho một tình huống cụ thể: **lỗi lập trình viên** hoặc **trạng thái hệ thống không thể phục hồi**.

Go tự động panic khi:

```go
package main

import "fmt"

func main() {
    // 1. Nil pointer dereference — tương đương NullReferenceException trong C#
    var p *int
    fmt.Println(*p) // panic: runtime error: invalid memory address or nil pointer dereference

    // 2. Index out of range
    s := []int{1, 2, 3}
    fmt.Println(s[10]) // panic: runtime error: index out of range [10] with length 3

    // 3. Type assertion thất bại
    var i interface{} = "hello"
    n := i.(int) // panic: interface conversion: interface {} is string, not int
    fmt.Println(n)
}
```

Khi `panic` xảy ra, chương trình bắt đầu "unwind" — tất cả deferred functions của goroutine hiện tại chạy theo thứ tự LIFO, sau đó chương trình crash và in stack trace.

### Khi nào bạn nên tự gọi panic?

Chỉ khi lỗi là **lỗi của lập trình viên** (không thể xảy ra nếu code đúng), không phải lỗi runtime bình thường.

```go
// BAD — dùng panic như exception để xử lý lỗi runtime bình thường
package main

import "os"

func readConfigBAD(path string) []byte {
    data, err := os.ReadFile(path)
    if err != nil {
        panic(err) // SAI! File không tìm thấy là lỗi runtime hợp lệ, không phải bug
    }
    return data
}
```

```go
// GOOD — panic chỉ cho invariant violations và lỗi lập trình viên
package main

import "fmt"

// Hàm này yêu cầu divisor != 0 — đây là contract của hàm
// Nếu caller truyền 0, đó là lỗi của caller, không phải runtime error
func divideGOOD(a, b int) int {
    if b == 0 {
        // Đây là lỗi lập trình viên — vi phạm precondition của hàm
        // Không thể "xử lý" được, phải crash để lộ bug sớm
        panic(fmt.Sprintf("divideGOOD: divisor không được bằng 0 (a=%d)", a))
    }
    return a / b
}

// Pattern phổ biến khác: Must* wrapper
// Dùng trong initialization — nếu fail thì chương trình không thể tiếp tục
func mustParseConfig(path string) []byte {
    // os.ReadFile trong thực tế
    data := []byte(`{"key": "value"}`)
    if len(data) == 0 {
        panic("config trống — chương trình không thể khởi động")
    }
    return data
}

func main() {
    result := divideGOOD(10, 2)
    fmt.Println("Kết quả:", result)
}
```

---

## 8. os.Exit() vs log.Fatal() vs panic() — Khi nào dùng cái nào?

Ba cách kết thúc chương trình bất thường, ba tình huống khác nhau:

| Cách kết thúc | Deferred functions chạy? | Stack trace? | Khi nào dùng |
|---|---|---|---|
| `os.Exit(code)` | Không | Không | Kết thúc có chủ ý với exit code, ví dụ: CLI tool sau khi in help |
| `log.Fatal(msg)` | Không | Không | Lỗi startup không thể phục hồi, in log trước khi exit |
| `panic(value)` | Có (LIFO) | Có | Bug của lập trình viên, invariant violation |

```go
package main

import (
    "fmt"
    "log"
    "os"
)

func demonstrateExit() {
    defer fmt.Println("Tôi sẽ KHÔNG được in — os.Exit bỏ qua defer")
    
    args := os.Args
    if len(args) < 2 {
        fmt.Fprintln(os.Stderr, "Usage: program <input-file>")
        os.Exit(1) // CLI tool — exit code 1 cho lỗi usage
    }
}

func demonstrateFatal() {
    defer fmt.Println("Tôi sẽ KHÔNG được in — log.Fatal gọi os.Exit bên trong")
    
    // Thường dùng trong main() khi không thể connect DB, bind port, v.v.
    // log.Fatal("Không thể kết nối database:", err)
    // Tương đương: log.Print(msg); os.Exit(1)
    log.Fatal("Lỗi khởi động — chương trình dừng")
}

func demonstratePanic() {
    defer fmt.Println("Tôi SẼ được in — panic chạy defers")
    
    panic("invariant bị vi phạm") // defers chạy, sau đó crash với stack trace
}

func main() {
    // Bỏ comment để test từng cái một:
    // demonstrateExit()
    // demonstrateFatal()
    demonstratePanic()
}
```

Trong C#, bạn quen dùng `Environment.Exit()` hoặc throw exception. Trong Go: dùng `os.Exit` cho CLI, `log.Fatal` cho startup failures, `panic` cho bugs — và return error cho mọi thứ còn lại.

---

## 9. recover() — Bắt panic, nhưng chỉ từ trong defer

`recover()` là hàm built-in cho phép bạn bắt một panic đang diễn ra và ngăn chương trình crash. Nhưng nó chỉ hoạt động khi được gọi **trực tiếp bên trong một deferred function**. Gọi `recover()` ở bất kỳ chỗ nào khác sẽ trả về `nil` và không làm gì cả.

```go
package main

import "fmt"

func safeOperation() {
    // defer này bắt bất kỳ panic nào trong safeOperation
    defer func() {
        if r := recover(); r != nil {
            // r là giá trị được truyền vào panic()
            fmt.Println("Đã bắt được panic:", r)
            // Sau khi recover, execution tiếp tục ở caller của safeOperation
            // KHÔNG tiếp tục tại điểm xảy ra panic
        }
    }()

    fmt.Println("Trước panic")
    panic("có gì đó sai rồi!")
    fmt.Println("Sau panic — dòng này không bao giờ chạy")
}

func main() {
    safeOperation()
    fmt.Println("main tiếp tục sau khi safeOperation kết thúc bình thường")
    // Output:
    // Trước panic
    // Đã bắt được panic: có gì đó sai rồi!
    // main tiếp tục sau khi safeOperation kết thúc bình thường
}
```

`recover` chỉ ngăn crash — nó không "resume" từ điểm panic. Execution tiếp tục sau lời gọi `safeOperation()` trong caller.

---

## 10. Middleware Recovery Pattern — Cách dùng recover đúng trong production

Pattern hợp lệ nhất cho `recover` trong production code là **HTTP middleware** hoặc **goroutine wrapper**. Ý tưởng: một panic trong một request handler không nên làm crash toàn bộ server. Middleware bắt panic, trả về 500, log stack trace, và server tiếp tục phục vụ các request khác.

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "runtime/debug"
)

// recoveryMiddleware bao bọc một http.Handler
// Bắt bất kỳ panic nào từ handler bên trong
// Trả về 500 thay vì crash server
func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                // Log stack trace đầy đủ để debug
                log.Printf("PANIC recovered: %v\n%s", rec, debug.Stack())
                
                // Trả về 500 cho client
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()

        // Gọi handler tiếp theo trong chain
        next.ServeHTTP(w, r)
    })
}

// Handler bình thường có thể panic vì bug
func userHandler(w http.ResponseWriter, r *http.Request) {
    // Giả sử có bug gây nil pointer panic
    var user *struct{ Name string }
    fmt.Fprintf(w, "Hello, %s", user.Name) // panic: nil pointer dereference
}

// Handler bình thường
func healthHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "OK")
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/user", userHandler)
    mux.HandleFunc("/health", healthHandler)

    // Bọc toàn bộ mux trong recovery middleware
    handler := recoveryMiddleware(mux)

    fmt.Println("Server đang chạy tại :8080")
    fmt.Println("Thử: curl http://localhost:8080/health")
    fmt.Println("Thử: curl http://localhost:8080/user  (sẽ panic nhưng server không crash)")
    
    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

Pattern tương tự cho goroutines:

```go
package main

import (
    "fmt"
    "log"
    "runtime/debug"
    "sync"
)

// safeGo thay thế go keyword, bắt panic trong goroutine
// Goroutine panic không được recover sẽ crash toàn bộ chương trình
func safeGo(fn func()) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Goroutine panic recovered: %v\n%s", r, debug.Stack())
            }
        }()
        fn()
    }()
}

func main() {
    var wg sync.WaitGroup
    wg.Add(2)

    safeGo(func() {
        defer wg.Done()
        fmt.Println("Goroutine 1: đang chạy bình thường")
    })

    safeGo(func() {
        defer wg.Done()
        panic("goroutine 2 bị panic!") // safeGo bắt cái này
    })

    wg.Wait()
    fmt.Println("main vẫn chạy sau khi goroutine panic")
}
```

---

## 11. Anti-pattern — KHÔNG dùng panic/recover như exception handling

Đây là điều quan trọng nhất cần hiểu. C# developer đôi khi cố dùng panic/recover để tái tạo try/catch. Đây là sai lầm nghiêm trọng trong Go.

```go
// BAD — dùng panic/recover như try/catch của C#
// Đây là code Go tệ nhất có thể viết
package main

import (
    "fmt"
    "strconv"
)

// Hàm này dùng panic thay vì return error — ĐỪNG LÀM VẬY
func parseIntBad(s string) int {
    n, err := strconv.Atoi(s)
    if err != nil {
        panic(err) // SAI: đây không phải bug, đây là lỗi input hợp lệ
    }
    return n
}

// Caller phải dùng recover như try/catch — rất xấu
func callerBad() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("Parse lỗi:", r) // Giống catch(Exception e) trong C#
        }
    }()
    
    result := parseIntBad("abc") // panic được bắt bởi recover ở trên
    fmt.Println("Kết quả:", result)
}
```

```go
// GOOD — return error như Go developer thực thụ
package main

import (
    "fmt"
    "strconv"
)

// Hàm này return error — đúng cách Go
func parseIntGood(s string) (int, error) {
    n, err := strconv.Atoi(s)
    if err != nil {
        return 0, fmt.Errorf("parseIntGood: input không hợp lệ %q: %w", s, err)
    }
    return n, nil
}

func callerGood() {
    result, err := parseIntGood("abc")
    if err != nil {
        fmt.Println("Parse lỗi:", err) // Xử lý rõ ràng, không cần recover
        return
    }
    fmt.Println("Kết quả:", result)
}

func main() {
    callerBad()
    callerGood()
}
```

Lý do panic/recover không dùng cho error handling thông thường: nó ẩn flow của chương trình, làm caller không biết hàm có thể "panic", khó compose với các hàm khác, và vi phạm triết lý Go về explicit errors.

---

## Bảng lỗi phổ biến

| Lỗi | Triệu chứng | Fix |
|---|---|---|
| `defer` trong vòng lặp | File handles / connections bị leak đến khi function kết thúc | Dùng anonymous function hoặc tách thành helper function |
| Quên `defer` arguments evaluate ngay | Biến in ra giá trị cũ, không phải giá trị lúc defer chạy | Dùng closure `defer func() { dùng biến ở đây }()` |
| `recover()` ngoài defer | `recover()` trả về nil, panic không bị bắt | Luôn gọi `recover()` bên trong `defer func() { ... }()` |
| Dùng `panic` cho lỗi thông thường | Code khó test, caller không biết cách xử lý | Return `error` thay vì panic |
| Quên `defer` sau khi mở resource | Resource leak khi có early return | Luôn viết `defer f.Close()` ngay dòng tiếp theo sau `os.Open()` |
| `log.Fatal` bên trong goroutine thay vì main | Server shutdown không graceful | Chỉ dùng `log.Fatal` trong `main()`, goroutine nên return error |
| Expect code chạy sau panic | Hiểu nhầm recover resume từ điểm panic | Sau recover, execution tiếp tục ở caller của function bị panic |

---

## Best Practices

Rule số một: **viết `defer` ngay sau khi mở resource, không bao giờ viết xa hơn**. Đây là convention của toàn bộ cộng đồng Go — ai cũng mong đọc thấy `f.Close()` ngay dòng sau `os.Open()`. Khoảng cách giữa "mở" và "đóng" trong code phải gần bằng không.

Rule số hai: **đừng bao giờ `defer` trong vòng lặp** trừ khi bạn hiểu rõ hệ quả. Nếu cần xử lý nhiều resources trong vòng lặp, tách logic thành function riêng hoặc dùng anonymous function. Đây là nguồn gốc của hầu hết resource leak bugs trong code Go của người mới.

Rule số ba: **`panic` chỉ dành cho những thứ "không thể xảy ra nếu code đúng"**. Nếu bạn đang xử lý input từ user, network, file system, hoặc database — luôn return error, không bao giờ panic. Nếu bạn đang enforce invariant của data structure hoặc API contract trong thư viện — panic là hợp lý.

Rule số bốn: **`recover` chỉ ở boundary layer**. HTTP handler middleware, goroutine launcher, hoặc plugin host — những nơi bạn không muốn một bug nhỏ làm sập toàn bộ hệ thống. Không có `recover` nằm sâu bên trong business logic.

Rule số năm: **khi recover, luôn log stack trace**. `debug.Stack()` từ package `runtime/debug` cho bạn stack trace đầy đủ. Nếu bạn recover mà không log, bạn đang ẩn bug và sẽ không bao giờ tìm ra nguyên nhân.

---

## Bài tập

### Bài tập 1 — File processor với defer đúng cách

Viết một function đọc nhiều file và in nội dung của chúng. Yêu cầu: không leak file handles, handle lỗi đúng cách, dùng defer đúng vị trí.

**Giải:**

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
)

// processFile đọc và in nội dung một file
// defer đảm bảo file luôn được đóng
func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("không mở được file %s: %w", filepath.Base(path), err)
    }
    defer f.Close() // Đặt ngay sau Open — không bao giờ quên

    data := make([]byte, 1024)
    n, err := f.Read(data)
    if err != nil {
        return fmt.Errorf("không đọc được file %s: %w", filepath.Base(path), err)
        // f.Close() vẫn chạy nhờ defer
    }

    fmt.Printf("=== %s (%d bytes) ===\n%s\n", filepath.Base(path), n, data[:n])
    return nil
    // f.Close() chạy ở đây
}

// processAllFiles xử lý nhiều files mà không leak handles
func processAllFiles(paths []string) {
    for _, path := range paths {
        // processFile là separate function → defer chạy đúng sau mỗi file
        if err := processFile(path); err != nil {
            fmt.Fprintf(os.Stderr, "Lỗi: %v\n", err)
        }
    }
}

func main() {
    // Tạo file test
    testFiles := []string{"a.txt", "b.txt", "c.txt"}
    for i, name := range testFiles {
        os.WriteFile(name, []byte(fmt.Sprintf("Nội dung file %d\nDòng thứ hai của file %d", i+1, i+1)), 0644)
    }
    defer func() {
        for _, f := range testFiles { os.Remove(f) }
    }()

    processAllFiles(testFiles)
    
    // Test với file không tồn tại — phải handle gracefully
    processAllFiles([]string{"khong_ton_tai.txt"})
}
```

---

### Bài tập 2 — HTTP server với recovery middleware

Viết một HTTP server đơn giản với recovery middleware. Server phải tiếp tục chạy dù handler có panic.

**Giải:**

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "runtime/debug"
    "time"
)

// ErrorResponse là struct trả về khi có lỗi
type ErrorResponse struct {
    Error     string `json:"error"`
    Timestamp string `json:"timestamp"`
}

// recoveryMiddleware bắt panic và trả về JSON 500
func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                // Log đầy đủ để debug
                log.Printf("[PANIC] %s %s\nCause: %v\nStack:\n%s",
                    r.Method, r.URL.Path, rec, debug.Stack())

                // Trả về JSON error cho client
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusInternalServerError)
                json.NewEncoder(w).Encode(ErrorResponse{
                    Error:     "Internal server error",
                    Timestamp: time.Now().Format(time.RFC3339),
                })
            }
        }()

        next.ServeHTTP(w, r)
    })
}

// loggingMiddleware log mỗi request
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        log.Printf("[%s] %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
        log.Printf("[%s] %s — %v", r.Method, r.URL.Path, time.Since(start))
    })
}

// healthHandler — luôn OK
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// buggyHandler — có bug gây panic (nil pointer)
func buggyHandler(w http.ResponseWriter, r *http.Request) {
    var data *struct{ Name string }
    fmt.Fprintf(w, "Hello, %s", data.Name) // panic: nil pointer dereference
}

// divideHandler — panic khi divisor = 0
func divideHandler(w http.ResponseWriter, r *http.Request) {
    a, b := 10, 0
    if b == 0 {
        panic("division by zero — lỗi lập trình viên")
    }
    fmt.Fprintf(w, "Kết quả: %d", a/b)
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)
    mux.HandleFunc("/buggy", buggyHandler)
    mux.HandleFunc("/divide", divideHandler)

    // Chain middleware: request → logging → recovery → handler
    handler := loggingMiddleware(recoveryMiddleware(mux))

    addr := ":8080"
    fmt.Println("Server khởi động tại", addr)
    fmt.Println()
    fmt.Println("Test commands:")
    fmt.Println("  curl http://localhost:8080/health")
    fmt.Println("  curl http://localhost:8080/buggy   # sẽ panic, server vẫn sống")
    fmt.Println("  curl http://localhost:8080/divide  # sẽ panic, server vẫn sống")
    fmt.Println("  curl http://localhost:8080/health  # server vẫn trả lời sau các panic trên")

    log.Fatal(http.ListenAndServe(addr, handler))
}
```

---

## Tóm tắt nhanh

`defer` là cleanup mechanism — dùng mọi lúc bạn mở resource. `panic` là crash-and-burn cho bugs không thể phục hồi — không dùng cho lỗi runtime bình thường. `recover` là safety net ở boundary layer — không dùng như try/catch.

Nếu bạn thấy mình muốn viết `panic` cho một lỗi mà caller có thể xử lý được — hãy return error. Nếu bạn thấy mình muốn viết `recover` để bắt "expected errors" — hãy return error. Trong Go, explicit error handling không phải boilerplate — nó là documentation của mọi thứ có thể sai.

---

**Next:** [08. Packages & Modules](08-packages-modules.md) — Tổ chức code Go: package layout, import, go.mod, visibility (exported/unexported).
