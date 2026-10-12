# 📚 Error Handling — Xử lý lỗi trong Go (dành cho C# developer)

---

## 1. Tại sao Go không có exceptions?

Đây là câu hỏi đầu tiên mà hầu hết C# developer đặt ra khi học Go, và câu trả lời phản ánh triết lý cốt lõi của ngôn ngữ.

Trong C#, lỗi được truyền tải qua cơ chế exception: một lỗi xảy ra ở tầng dưới có thể "bay" lên nhiều tầng call stack mà không cần khai báo gì, và bạn bắt nó bằng `try/catch` ở bất kỳ tầng nào. Điều này tiện lợi, nhưng có một cái giá ẩn: **bạn không biết một hàm có thể throw exception nào trừ khi đọc tài liệu hoặc source code**. Signature `public User GetUser(int id)` không nói gì về khả năng throw `SqlException`, `NotFoundException`, hay `NullReferenceException`.

Go theo triết lý ngược lại: **lỗi là giá trị trả về bình thường, không phải luồng điều khiển đặc biệt**. Khi một hàm có thể thất bại, nó khai báo điều đó ngay trong signature bằng cách trả về thêm một giá trị `error`. Người gọi buộc phải đối mặt với khả năng lỗi ngay tại điểm gọi — không thể bỏ qua vô tình.

Điều này khiến code Go dài hơn về mặt số dòng, nhưng đổi lại bạn có một hệ thống mà **mọi con đường lỗi đều tường minh và được compiler enforce một phần**. Các kỹ sư của Google, khi thiết kế Go, nhận thấy rằng trong hệ thống lớn, exception thường bị bắt sai chỗ, bị bỏ qua bằng catch rỗng, hoặc tạo ra control flow mà không ai trong team hiểu rõ. Go chọn sự rõ ràng thay vì sự tiện lợi.

---

## 2. Interface `error` — nền tảng của toàn bộ hệ thống

Trong Go, `error` không phải là một keyword hay một class đặc biệt. Nó chỉ là một interface cực kỳ đơn giản được định nghĩa trong thư viện chuẩn:

```go
type error interface {
    Error() string
}
```

Bất kỳ type nào có method `Error() string` đều tự động thỏa mãn interface `error`. Đây là implicit interface satisfaction — không cần khai báo `implements` như C#.

Điều này có nghĩa là bạn có thể tạo ra custom error type với bất kỳ thông tin nào bạn muốn (HTTP status code, error code, stack info, field name...) miễn là type đó có method `Error() string`. Hệ thống xử lý lỗi của Go hoàn toàn có thể mở rộng mà không cần framework hay thư viện ngoài.

Giá trị zero của interface là `nil`. Khi một hàm thành công, nó trả về `nil` cho giá trị `error`. Đây là quy ước nhất quán trên toàn bộ ecosystem Go.

---

## 3. Tạo lỗi cơ bản: `errors.New()` và `fmt.Errorf()`

```go
package main

import (
    "errors"
    "fmt"
)

func main() {
    // errors.New tạo một error đơn giản với message cố định
    err1 := errors.New("không tìm thấy người dùng")

    // fmt.Errorf tạo error với message có format (giống fmt.Sprintf)
    userID := 42
    err2 := fmt.Errorf("không tìm thấy người dùng với ID %d", userID)

    fmt.Println(err1) // không tìm thấy người dùng
    fmt.Println(err2) // không tìm thấy người dùng với ID 42
}
```

Trong C#, bạn thường viết `throw new Exception("message")` hoặc `throw new ArgumentException("message", "paramName")`. Trong Go, `errors.New()` là tương đương gần nhất — nhưng thay vì throw, bạn return nó.

`fmt.Errorf()` là công cụ bạn sẽ dùng nhiều nhất vì nó kết hợp tạo message động và wrapping (sẽ nói ở phần 5).

---

## 4. Return và kiểm tra lỗi — pattern `if err != nil`

Đây là pattern bạn sẽ viết hàng chục lần mỗi ngày khi code Go, và nó trông có vẻ verbose. Hãy hiểu tại sao nó là lựa chọn đúng.

```go
package main

import (
    "errors"
    "fmt"
)

// BAD (C# style thinking — đừng làm thế này trong Go)
// Hàm này panic thay vì trả về lỗi, khiến caller không kiểm soát được
func getUserBad(id int) string {
    if id <= 0 {
        panic("id không hợp lệ") // caller không thể handle gracefully
    }
    return "Nguyen Van A"
}

// GOOD — trả về (value, error), caller quyết định xử lý thế nào
func getUser(id int) (string, error) {
    if id <= 0 {
        // Trả về zero value cho string và error có thông tin
        return "", fmt.Errorf("id người dùng không hợp lệ: %d", id)
    }
    // Giả lập tìm thấy user
    return "Nguyen Van A", nil // nil nghĩa là không có lỗi
}

func main() {
    // Mỗi lần gọi hàm có thể thất bại, bạn phải kiểm tra lỗi ngay
    user, err := getUser(-1)
    if err != nil {
        // Xử lý lỗi tại đây — không để lỗi "bay" ngầm
        fmt.Println("Lỗi:", err)
        return
    }
    // Đến đây, chắc chắn user có giá trị hợp lệ
    fmt.Println("User:", user)
}
```

Pattern `if err != nil` xuất hiện lặp đi lặp lại vì **mỗi điểm có thể thất bại là một điểm quyết định**. Bạn buộc phải hỏi: "Nếu bước này thất bại, tôi làm gì?" — return lỗi lên trên, log và tiếp tục, hay retry? Go không cho phép bạn bỏ qua câu hỏi này. Đây chính là điểm mà Go khác với C#: trong C# bạn có thể quên bắt exception; trong Go bạn khó mà quên kiểm tra return value vì compiler sẽ cảnh báo về biến chưa dùng.

---

## 5. Wrapping errors với `%w` — tại sao quan trọng

Khi lỗi xảy ra ở tầng thấp (database, HTTP client...), bạn muốn thêm context khi propagate lên trên. `fmt.Errorf` với verb `%w` cho phép bạn "bọc" error gốc bên trong một error mới.

```go
package main

import (
    "errors"
    "fmt"
)

// Lỗi tầng thấp (giả lập database error)
var ErrConnection = errors.New("không thể kết nối database")

func queryDB(userID int) (string, error) {
    // Giả lập DB fail
    return "", ErrConnection
}

// BAD — mất thông tin context, caller không biết lỗi xảy ra ở đâu
func getUserBad(id int) (string, error) {
    _, err := queryDB(id)
    if err != nil {
        return "", err // trả nguyên xi lỗi gốc, mất context
    }
    return "user", nil
}

// GOOD — wrap error với context, giữ nguyên error gốc bên trong
func getUser(id int) (string, error) {
    name, err := queryDB(id)
    if err != nil {
        // %w "bọc" err vào trong error mới
        // Error message: "lấy user 42: không thể kết nối database"
        return "", fmt.Errorf("lấy user %d: %w", id, err)
    }
    return name, nil
}

func main() {
    _, err := getUser(42)
    if err != nil {
        fmt.Println(err)
        // Output: lấy user 42: không thể kết nối database

        // Vẫn có thể kiểm tra error gốc bên trong chain
        if errors.Is(err, ErrConnection) {
            fmt.Println("Xác nhận: đây là lỗi kết nối DB")
        }
    }
}
```

Verb `%w` (wrap) khác với `%v` (chỉ format thành string). `%w` tạo ra một error "chain": error mới chứa error cũ bên trong, và `errors.Is()` / `errors.As()` có thể traverse chain này. Nếu bạn dùng `%v`, error gốc biến thành text và không còn có thể kiểm tra type hay identity nữa — đây là một lỗi phổ biến.

Quy ước đặt tên cho wrapped error message: dùng dấu hai chấm, không viết hoa, không chấm câu cuối. Ví dụ: `"lấy user: query database: kết nối timeout"`. Khi stack nhiều tầng lại, bạn có một chuỗi context có thể đọc được.

---

## 6. `errors.Is()` — kiểm tra error chain

Trong C#, bạn dùng `catch (NotFoundException ex)` hoặc `ex is NotFoundException` để kiểm tra loại exception. Trong Go, `errors.Is()` kiểm tra xem một error có phải là (hoặc chứa trong chain) một error cụ thể không.

```go
package main

import (
    "errors"
    "fmt"
)

// Sentinel errors — khai báo ở package level
var (
    ErrNotFound   = errors.New("không tìm thấy")
    ErrPermission = errors.New("không có quyền")
)

func findUser(id int) error {
    if id == 0 {
        return ErrNotFound
    }
    if id < 0 {
        // Wrap ErrNotFound vào trong lỗi có context hơn
        return fmt.Errorf("tìm user %d: %w", id, ErrNotFound)
    }
    return nil
}

func main() {
    // BAD — so sánh bằng == chỉ đúng với error trực tiếp, không xuyên qua chain
    err := findUser(-5)
    if err == ErrNotFound { // FALSE! vì err là wrapped error
        fmt.Println("Không tìm thấy (so sánh ==)") // sẽ không in ra
    }

    // GOOD — errors.Is() traverse toàn bộ error chain
    if errors.Is(err, ErrNotFound) { // TRUE — tìm thấy trong chain
        fmt.Println("Không tìm thấy (errors.Is)") // sẽ in ra
    }

    // errors.Is hoạt động với cả error trực tiếp lẫn wrapped
    err2 := findUser(0) // trả về ErrNotFound trực tiếp (không wrap)
    fmt.Println(errors.Is(err2, ErrNotFound)) // true
}
```

`errors.Is()` unwrap error chain lặp đi lặp lại (gọi method `Unwrap()` nếu có) cho đến khi tìm thấy match hoặc hết chain. Đây là lý do tại sao `%w` quan trọng hơn `%v` — `%w` giữ lại chain, `%v` phá vỡ nó.

---

## 7. `errors.As()` — trích xuất error type từ chain

Trong C# bạn dùng `catch (HttpException ex) { Console.WriteLine(ex.StatusCode); }` để vừa kiểm tra type vừa truy cập fields của exception. Trong Go, `errors.As()` làm điều tương tự.

```go
package main

import (
    "errors"
    "fmt"
)

// Custom error type với thông tin bổ sung
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation lỗi field '%s': %s", e.Field, e.Message)
}

func validateAge(age int) error {
    if age < 0 || age > 150 {
        return &ValidationError{
            Field:   "age",
            Message: fmt.Sprintf("giá trị %d không hợp lệ, phải từ 0 đến 150", age),
        }
    }
    return nil
}

func processUser(age int) error {
    if err := validateAge(age); err != nil {
        // Wrap ValidationError vào trong lỗi có context
        return fmt.Errorf("xử lý user: %w", err)
    }
    return nil
}

func main() {
    err := processUser(-5)
    if err != nil {
        // errors.As tìm *ValidationError trong chain và gán vào biến target
        var valErr *ValidationError
        if errors.As(err, &valErr) {
            // Bây giờ có thể truy cập fields của ValidationError
            fmt.Printf("Field bị lỗi: %s\n", valErr.Field)
            fmt.Printf("Chi tiết: %s\n", valErr.Message)
        } else {
            fmt.Println("Lỗi khác:", err)
        }
    }
}
```

Lưu ý quan trọng: `errors.As()` nhận pointer-to-pointer (`&valErr` khi `valErr` là `*ValidationError`). Nếu sai kiểu, nó không panic — nó chỉ trả về `false`. Hãy nhớ: dùng `errors.Is()` khi muốn kiểm tra identity (đây có phải error này không?), dùng `errors.As()` khi muốn trích xuất data từ error.

---

## 8. Custom error types — khi nào dùng và khi nào không

Không phải lúc nào cũng cần custom error type. Hãy hiểu khi nào thì đáng đầu tư.

```go
package main

import (
    "errors"
    "fmt"
    "net/http"
)

// --- Ví dụ custom error type hữu ích ---

// HTTPError mang thêm status code — caller cần thông tin này để response
type HTTPError struct {
    StatusCode int
    Message    string
}

func (e *HTTPError) Error() string {
    return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// DBError mang thêm query để debug
type DBError struct {
    Query string
    Cause error
}

func (e *DBError) Error() string {
    return fmt.Sprintf("lỗi DB khi chạy query '%s': %v", e.Query, e.Cause)
}

// Unwrap cho phép errors.Is/As traverse vào Cause
func (e *DBError) Unwrap() error {
    return e.Cause
}

// BAD — tạo custom type chỉ để có message, errors.New là đủ
type SimpleNotFoundError struct {
    Message string
}
func (e *SimpleNotFoundError) Error() string { return e.Message }
// Dùng thế này thì thừa — errors.New("không tìm thấy") là đủ

// GOOD — dùng custom type khi caller cần truy cập data trong error
func fetchFromAPI(url string) error {
    // Giả lập 404
    return &HTTPError{
        StatusCode: http.StatusNotFound,
        Message:    "resource không tồn tại",
    }
}

func handler() {
    err := fetchFromAPI("/api/users/999")
    if err != nil {
        var httpErr *HTTPError
        if errors.As(err, &httpErr) {
            // Có thể dùng StatusCode để quyết định response
            fmt.Printf("Trả về status %d cho client\n", httpErr.StatusCode)
        }
    }
}

func main() {
    handler()
}
```

Quy tắc thực tế: dùng `errors.New()` hoặc `fmt.Errorf()` cho 80% trường hợp. Dùng custom error type khi caller cần truy cập structured data từ error (status code, field name, retry-after duration...). Đừng tạo custom type chỉ để có message đẹp hơn.

---

## 9. Sentinel errors — `var ErrNotFound = errors.New(...)`

Sentinel error là các error được khai báo ở package level như biến, dùng làm "mã lỗi" mà caller có thể so sánh với `errors.Is()`.

```go
package main

import (
    "errors"
    "fmt"
)

// Khai báo ở package level — đây là sentinel errors
// Convention: bắt đầu bằng "Err", export nếu caller bên ngoài package cần dùng
var (
    ErrNotFound     = errors.New("không tìm thấy")
    ErrUnauthorized = errors.New("chưa đăng nhập")
    ErrForbidden    = errors.New("không có quyền")
)

// BAD — trả về error với message khác nhau mỗi lần,
// caller không thể kiểm tra bằng errors.Is
func getUserBad(id int) (string, error) {
    if id == 0 {
        return "", errors.New("không tìm thấy user") // mỗi lần là instance mới
    }
    return "user", nil
}

// GOOD — trả về sentinel error hoặc wrap nó với context
func getUser(id int) (string, error) {
    if id == 0 {
        return "", ErrNotFound // instance duy nhất, errors.Is hoạt động đúng
    }
    if id < 0 {
        return "", fmt.Errorf("getUser %d: %w", id, ErrNotFound) // wrap nhưng giữ identity
    }
    return "Nguyen Van A", nil
}

func main() {
    _, err := getUser(0)

    // Caller kiểm tra loại lỗi để quyết định xử lý
    switch {
    case errors.Is(err, ErrNotFound):
        fmt.Println("→ Trả về 404 cho client")
    case errors.Is(err, ErrUnauthorized):
        fmt.Println("→ Redirect đến trang đăng nhập")
    case err != nil:
        fmt.Println("→ Lỗi nội bộ:", err)
    }
}
```

Sentinel errors tương tự như việc dùng các exception class cụ thể trong C# (`NotFoundException`, `UnauthorizedException`...) thay vì throw `new Exception("not found")`. Chúng cho phép caller phân biệt các loại lỗi khác nhau mà không cần parse string message — đây là điều tối kỵ.

Khi nào không dùng sentinel: khi error chỉ dùng nội bộ và caller không cần phân biệt nó với error khác. Khi đó `fmt.Errorf()` là đủ.

---

## 10. `panic()` vs `os.Exit()` vs `log.Fatal()` — phân biệt quan trọng

Đây là phần mà C# developer hay mắc sai lầm nhất. Ba function này trông giống nhau nhưng có hành vi rất khác nhau, và dùng sai có thể gây hậu quả nghiêm trọng.

```go
package main

import (
    "fmt"
    "log"
    "os"
)

func demonstratePanic() {
    // panic() dừng goroutine hiện tại, unwind stack, chạy deferred functions,
    // sau đó crash chương trình (trừ khi có recover)
    // Tương tự throw Exception trong C# — nhưng KHÔNG dùng để xử lý lỗi thông thường

    // BAD — dùng panic để xử lý lỗi nghiệp vụ
    // Đây là cách sai hoàn toàn
    userID := -1
    if userID < 0 {
        // panic("userID không hợp lệ") // ĐỪNG làm thế này
        _ = userID
    }

    // GOOD — panic chỉ dùng cho "điều không bao giờ nên xảy ra"
    // Ví dụ: khởi tạo thất bại, invariant bị vi phạm
    slice := make([]int, 5)
    _ = slice
    // slice[10] = 1 // Go tự panic với index out of range — đây là bug, không phải lỗi nghiệp vụ
}

func main() {
    // os.Exit(code) — thoát ngay lập tức, KHÔNG chạy deferred functions
    // Dùng khi: chương trình muốn thoát với exit code cụ thể (ví dụ: CLI tool)
    // NGUY HIỂM: deferred cleanup sẽ không chạy

    // log.Fatal(msg) — tương đương log.Print(msg) + os.Exit(1)
    // Dùng khi: khởi tạo thất bại ở main(), không thể tiếp tục
    // CŨNG NGUY HIỂM: deferred functions không chạy

    // --- Ví dụ đúng cho từng loại ---

    // Khởi tạo ứng dụng — dùng log.Fatal nếu thiếu config quan trọng
    configFile := "config.yaml"
    if configFile == "" {
        log.Fatal("Thiếu config file — không thể khởi động") // thoát code 1
    }

    // Lỗi nghiệp vụ thông thường — return error
    _, err := fmt.Println("Xử lý bình thường")
    if err != nil {
        fmt.Fprintln(os.Stderr, "Lỗi:", err)
        os.Exit(1) // chỉ dùng ở main hoặc CLI entry point
    }

    // panic chỉ dùng khi phát hiện bug, không phải lỗi runtime thông thường
    m := map[string]int{"a": 1}
    val, ok := m["b"]
    if !ok {
        // KHÔNG panic ở đây — đây là trường hợp bình thường
        _ = val
        fmt.Println("Key không tồn tại — xử lý bình thường")
    }
}
```

Tóm lại nguyên tắc:

`panic` — chỉ dùng khi chương trình gặp trạng thái không thể phục hồi do BUG (không phải do input sai hay network timeout). Ví dụ: `template.Must(template.ParseFiles(...))` trong khởi tạo. Nếu bạn từng viết `panic(err)` trong business logic, đó là dấu hiệu cần refactor.

`log.Fatal` — chỉ dùng ở `main()` hoặc `init()` khi không thể khởi động ứng dụng. Không dùng trong middleware, handlers, hay service functions.

`os.Exit` — dùng ở CLI entry point khi cần exit code cụ thể. Tránh hoàn toàn bên trong libraries.

---

## 11. `recover()` — chỉ dùng ở top-level handlers

`recover()` là cơ chế để bắt `panic` và ngăn chương trình crash. Nó tương tự `catch` trong C#, nhưng với một quy tắc cứng: **chỉ dùng ở top-level handler, không bao giờ dùng để thay thế error handling thông thường**.

```go
package main

import "fmt"

// BAD — dùng recover như try/catch, đây là sai hoàn toàn trong Go
func doWorkBad() (result string) {
    defer func() {
        if r := recover(); r != nil {
            result = "lỗi bị nuốt" // che giấu panic, che giấu bug
        }
    }()
    // ... code có thể panic
    return "ok"
}

// GOOD — recover chỉ hợp lý ở server handler để ngăn một request crash toàn server
func safeHandler(handler func()) {
    defer func() {
        if r := recover(); r != nil {
            // Log panic để debug, nhưng không để crash server
            fmt.Printf("Panic bị bắt trong handler: %v\n", r)
            // Trong web server thực: trả về 500 cho client
        }
    }()
    handler()
}

// Ví dụ: middleware trong HTTP server (pseudocode)
func panicRecoveryMiddleware(next func()) func() {
    return func() {
        defer func() {
            if r := recover(); r != nil {
                fmt.Printf("[PANIC] %v — trả về 500\n", r)
                // w.WriteHeader(http.StatusInternalServerError)
            }
        }()
        next()
    }
}

func main() {
    // safeHandler sẽ bắt panic từ handler bên trong
    safeHandler(func() {
        fmt.Println("Handler chạy bình thường")
    })

    safeHandler(func() {
        panic("something went very wrong") // bị bắt bởi safeHandler
    })

    fmt.Println("Chương trình vẫn tiếp tục sau panic được recover")
}
```

Trong hầu hết các web framework Go (Gin, Echo, Chi...), đã có recovery middleware sẵn. Bạn hiếm khi cần tự viết `recover()`. Nếu bạn thấy mình muốn dùng `recover()` bên trong business logic để bắt lỗi, đó là dấu hiệu cần dùng `return error` thay thế.

---

## 12. Common patterns — wrap ở mỗi tầng, không log VÀ return

Có hai pattern phổ biến cần nắm vững khi xây dựng ứng dụng thực tế.

**Pattern 1: Wrap error ở mỗi tầng để có context đầy đủ**

```go
package main

import (
    "errors"
    "fmt"
)

var ErrNotFound = errors.New("không tìm thấy")

// Tầng repository — lỗi cụ thể nhất
func dbQuery(id int) (string, error) {
    if id == 99 {
        return "", fmt.Errorf("dbQuery id=%d: %w", id, ErrNotFound)
    }
    return "raw_data", nil
}

// Tầng repository — wrap với context của tầng này
func userRepo_GetByID(id int) (string, error) {
    data, err := dbQuery(id)
    if err != nil {
        return "", fmt.Errorf("userRepo.GetByID id=%d: %w", id, err)
    }
    return data, nil
}

// Tầng service — wrap với context của tầng này
func userService_GetUser(id int) (string, error) {
    user, err := userRepo_GetByID(id)
    if err != nil {
        return "", fmt.Errorf("userService.GetUser: %w", err)
    }
    return user, nil
}

func main() {
    _, err := userService_GetUser(99)
    if err != nil {
        // Full context chain: biết chính xác lỗi xảy ra ở đâu
        fmt.Println(err)
        // Output: userService.GetUser: userRepo.GetByID id=99: dbQuery id=99: không tìm thấy

        // Vẫn kiểm tra được root cause
        if errors.Is(err, ErrNotFound) {
            fmt.Println("Root cause: không tìm thấy record")
        }
    }
}
```

**Pattern 2: Không log VÀ return cùng lúc — chọn một**

```go
package main

import (
    "errors"
    "fmt"
    "log"
)

var ErrDB = errors.New("lỗi database")

func queryDatabase() error {
    return ErrDB
}

// BAD — log error rồi lại return, caller cũng log → log bị duplicate
func getDataBad() error {
    err := queryDatabase()
    if err != nil {
        log.Printf("Lỗi query: %v", err) // log ở đây...
        return fmt.Errorf("getData: %w", err) // ...rồi lại return, caller sẽ log nữa
    }
    return nil
}

// GOOD — chỉ return, để tầng trên cùng (handler/main) quyết định log
func getData() error {
    err := queryDatabase()
    if err != nil {
        return fmt.Errorf("getData: %w", err) // chỉ wrap và return
    }
    return nil
}

// Tầng xử lý cuối cùng — nơi DUY NHẤT log error
func handleRequest() {
    err := getData()
    if err != nil {
        // Chỉ log ở đây, một lần duy nhất
        log.Printf("handleRequest thất bại: %v", err)
        // Trả về response lỗi cho client
        fmt.Println("→ Gửi HTTP 500 cho client")
    }
}

func main() {
    handleRequest()
}
```

---

## 13. Real example: service gọi repo, wrap errors ở mỗi tầng

Đây là ví dụ hoàn chỉnh mô phỏng một ứng dụng thực tế với ba tầng rõ ràng.

```go
package main

import (
    "errors"
    "fmt"
    "log"
)

// ============================================================
// Định nghĩa errors
// ============================================================

var (
    ErrNotFound     = errors.New("không tìm thấy")
    ErrUnauthorized = errors.New("chưa đăng nhập")
)

// Custom error type khi cần trả thêm thông tin cho caller
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: field '%s' — %s", e.Field, e.Message)
}

// ============================================================
// Model
// ============================================================

type User struct {
    ID    int
    Name  string
    Email string
    Role  string
}

type Order struct {
    ID     int
    UserID int
    Amount float64
}

// ============================================================
// Tầng Repository (giả lập DB)
// ============================================================

type UserRepository struct{}

func (r *UserRepository) FindByID(id int) (*User, error) {
    // Giả lập DB query
    fakeDB := map[int]*User{
        1: {ID: 1, Name: "Nguyen Van A", Email: "a@example.com", Role: "admin"},
        2: {ID: 2, Name: "Tran Thi B", Email: "b@example.com", Role: "user"},
    }

    user, ok := fakeDB[id]
    if !ok {
        // Trả về sentinel error có context
        return nil, fmt.Errorf("UserRepository.FindByID id=%d: %w", id, ErrNotFound)
    }
    return user, nil
}

type OrderRepository struct{}

func (r *OrderRepository) FindByUserID(userID int) ([]Order, error) {
    // Giả lập DB connection timeout
    if userID == 999 {
        return nil, fmt.Errorf("OrderRepository.FindByUserID: lỗi kết nối DB")
    }

    fakeOrders := map[int][]Order{
        1: {{ID: 101, UserID: 1, Amount: 150000}, {ID: 102, UserID: 1, Amount: 75000}},
        2: {},
    }

    orders, _ := fakeOrders[userID]
    return orders, nil
}

// ============================================================
// Tầng Service
// ============================================================

type OrderService struct {
    userRepo  *UserRepository
    orderRepo *OrderRepository
}

func NewOrderService() *OrderService {
    return &OrderService{
        userRepo:  &UserRepository{},
        orderRepo: &OrderRepository{},
    }
}

// GetUserOrders lấy danh sách order của user, với kiểm tra quyền
func (s *OrderService) GetUserOrders(requesterID, targetUserID int) ([]Order, error) {
    // Bước 1: Validate input
    if targetUserID <= 0 {
        return nil, &ValidationError{
            Field:   "targetUserID",
            Message: "phải là số nguyên dương",
        }
    }

    // Bước 2: Lấy thông tin requester để kiểm tra quyền
    requester, err := s.userRepo.FindByID(requesterID)
    if err != nil {
        // Wrap với context của service
        return nil, fmt.Errorf("OrderService.GetUserOrders — lấy thông tin requester: %w", err)
    }

    // Bước 3: Kiểm tra quyền — chỉ admin hoặc chính user đó mới được xem
    if requester.Role != "admin" && requesterID != targetUserID {
        return nil, fmt.Errorf("OrderService.GetUserOrders — user %d xem order của user %d: %w",
            requesterID, targetUserID, ErrUnauthorized)
    }

    // Bước 4: Lấy orders
    orders, err := s.orderRepo.FindByUserID(targetUserID)
    if err != nil {
        return nil, fmt.Errorf("OrderService.GetUserOrders — lấy orders của user %d: %w",
            targetUserID, err)
    }

    return orders, nil
}

// ============================================================
// Tầng Handler (entry point — nơi duy nhất log và quyết định response)
// ============================================================

func handleGetOrders(service *OrderService, requesterID, targetUserID int) {
    orders, err := service.GetUserOrders(requesterID, targetUserID)
    if err != nil {
        // Kiểm tra loại lỗi để trả về HTTP status phù hợp
        switch {
        case errors.Is(err, ErrNotFound):
            log.Printf("[404] %v", err)
            fmt.Printf("→ HTTP 404: %v\n\n", err)

        case errors.Is(err, ErrUnauthorized):
            log.Printf("[401] %v", err)
            fmt.Printf("→ HTTP 401: Bạn không có quyền xem dữ liệu này\n\n")

        default:
            // Kiểm tra ValidationError
            var valErr *ValidationError
            if errors.As(err, &valErr) {
                log.Printf("[400] %v", err)
                fmt.Printf("→ HTTP 400: Dữ liệu không hợp lệ — field '%s': %s\n\n",
                    valErr.Field, valErr.Message)
            } else {
                // Lỗi nội bộ — log đầy đủ, trả về message chung cho client
                log.Printf("[500] %v", err)
                fmt.Printf("→ HTTP 500: Lỗi hệ thống, vui lòng thử lại\n\n")
            }
        }
        return
    }

    fmt.Printf("→ HTTP 200: Tìm thấy %d đơn hàng\n\n", len(orders))
}

func main() {
    service := NewOrderService()

    fmt.Println("=== Test 1: Requester không tồn tại ===")
    handleGetOrders(service, 999, 1) // requesterID không có trong DB

    fmt.Println("=== Test 2: Không đủ quyền ===")
    handleGetOrders(service, 2, 1) // user thường xem order của người khác

    fmt.Println("=== Test 3: Input không hợp lệ ===")
    handleGetOrders(service, 1, -1) // targetUserID âm

    fmt.Println("=== Test 4: Thành công — admin xem tất cả ===")
    handleGetOrders(service, 1, 2) // admin (id=1) xem order của user 2

    fmt.Println("=== Test 5: Thành công — user xem của chính mình ===")
    handleGetOrders(service, 2, 2) // user xem order của chính mình
}
```

---

## Bảng: Common Issues và cách tránh

| Lỗi thường gặp | Ví dụ sai | Cách đúng |
|---|---|---|
| Bỏ qua error | `result, _ := getUser(1)` | Luôn kiểm tra `err != nil`, chỉ dùng `_` khi chắc chắn không thể lỗi |
| Dùng `%v` thay `%w` khi wrap | `fmt.Errorf("ctx: %v", err)` | `fmt.Errorf("ctx: %w", err)` để giữ error chain |
| So sánh error bằng `==` | `if err == ErrNotFound` | `if errors.Is(err, ErrNotFound)` |
| Log VÀ return error | `log.Print(err); return err` | Chỉ return, để tầng trên log |
| Dùng panic thay return error | `panic("user not found")` | `return nil, ErrNotFound` |
| log.Fatal trong business logic | `log.Fatal("query failed")` | Return error, chỉ dùng `log.Fatal` ở `main()` |
| Tạo custom type khi không cần | Struct chỉ có message | Dùng `errors.New()` hoặc `fmt.Errorf()` |
| Wrap quá nhiều tầng gây noise | Mỗi dòng code wrap error | Chỉ wrap khi thêm context có ý nghĩa |
| Không implement `Unwrap()` | Custom error không có `Unwrap()` | Thêm `Unwrap() error` nếu wrap error khác |
| Parse error message bằng string | `strings.Contains(err.Error(), "not found")` | Dùng `errors.Is()` hoặc `errors.As()` |

---

## Best Practices

**Luôn wrap với context có ý nghĩa.** Mỗi tầng trong ứng dụng nên thêm context mô tả hành động đang thực hiện, không phải chỉ pass error nguyên xi. Context tốt giúp bạn đọc log và biết ngay lỗi xảy ra ở đâu mà không cần debugger.

**Tầng trên cùng là nơi duy nhất log.** Trong web service, đó là middleware hoặc handler. Trong CLI, đó là `main()`. Nếu mọi tầng đều log, log file sẽ đầy những entry trùng lặp khó đọc.

**Đặt tên sentinel error theo convention `ErrXxx`.** Export sentinel error nếu package khác cần dùng `errors.Is()`. Không export nếu chỉ dùng nội bộ.

**Không bao giờ trả về error interface với concrete type nil.** Đây là một gotcha nổi tiếng trong Go: nếu bạn return một `*MyError` nil được cast thành `error`, caller sẽ thấy `err != nil` là `true` dù không có lỗi. Luôn return `nil` trực tiếp khi không có lỗi.

**Viết error message ở dạng lowercase, không dấu chấm cuối câu.** Vì error message thường được wrap và nối với nhau, việc viết hoa hay có dấu chấm sẽ làm chuỗi context trông lạ kỳ.

**Dùng `errors.Is()` và `errors.As()` thay vì type assertion trực tiếp.** Type assertion (`err.(*MyError)`) không traverse error chain, trong khi `errors.As()` thì có. Luôn ưu tiên `errors.As()` khi làm việc với wrapped errors.

---

## Bài tập thực hành

### Bài tập 1: Xây dựng inventory service với error handling đúng chuẩn

Viết một chương trình quản lý kho hàng với các yêu cầu:
- Sản phẩm có ID, tên, số lượng trong kho
- Hàm `ReserveItem(productID, quantity int)` trừ kho
- Xử lý các trường hợp: sản phẩm không tồn tại, số lượng không đủ, số lượng request âm
- Caller phải phân biệt được từng loại lỗi

```go
package main

import (
    "errors"
    "fmt"
)

// ============================================================
// Sentinel errors
// ============================================================

var (
    ErrProductNotFound  = errors.New("sản phẩm không tồn tại")
    ErrInsufficientStock = errors.New("số lượng trong kho không đủ")
)

// ============================================================
// Custom error type cho validation
// ============================================================

type InvalidQuantityError struct {
    Requested int
    Reason    string
}

func (e *InvalidQuantityError) Error() string {
    return fmt.Sprintf("số lượng không hợp lệ (%d): %s", e.Requested, e.Reason)
}

// ============================================================
// Model & Repository
// ============================================================

type Product struct {
    ID       int
    Name     string
    Stock    int
}

type InventoryRepository struct {
    products map[int]*Product
}

func NewInventoryRepository() *InventoryRepository {
    return &InventoryRepository{
        products: map[int]*Product{
            1: {ID: 1, Name: "Laptop Dell XPS", Stock: 5},
            2: {ID: 2, Name: "Chuột Logitech MX", Stock: 0},
            3: {ID: 3, Name: "Bàn phím Keychron", Stock: 12},
        },
    }
}

func (r *InventoryRepository) FindByID(id int) (*Product, error) {
    p, ok := r.products[id]
    if !ok {
        return nil, fmt.Errorf("InventoryRepository.FindByID id=%d: %w", id, ErrProductNotFound)
    }
    return p, nil
}

func (r *InventoryRepository) UpdateStock(id, newStock int) error {
    p, ok := r.products[id]
    if !ok {
        return fmt.Errorf("InventoryRepository.UpdateStock id=%d: %w", id, ErrProductNotFound)
    }
    p.Stock = newStock
    return nil
}

// ============================================================
// Service
// ============================================================

type InventoryService struct {
    repo *InventoryRepository
}

func NewInventoryService(repo *InventoryRepository) *InventoryService {
    return &InventoryService{repo: repo}
}

func (s *InventoryService) ReserveItem(productID, quantity int) error {
    // Bước 1: Validate quantity
    if quantity <= 0 {
        return &InvalidQuantityError{
            Requested: quantity,
            Reason:    "phải lớn hơn 0",
        }
    }
    if quantity > 1000 {
        return &InvalidQuantityError{
            Requested: quantity,
            Reason:    "không thể đặt hàng quá 1000 đơn vị một lần",
        }
    }

    // Bước 2: Lấy sản phẩm
    product, err := s.repo.FindByID(productID)
    if err != nil {
        return fmt.Errorf("InventoryService.ReserveItem productID=%d: %w", productID, err)
    }

    // Bước 3: Kiểm tra tồn kho
    if product.Stock < quantity {
        return fmt.Errorf("InventoryService.ReserveItem — '%s' còn %d, yêu cầu %d: %w",
            product.Name, product.Stock, quantity, ErrInsufficientStock)
    }

    // Bước 4: Cập nhật kho
    err = s.repo.UpdateStock(productID, product.Stock-quantity)
    if err != nil {
        return fmt.Errorf("InventoryService.ReserveItem — cập nhật kho: %w", err)
    }

    return nil
}

// ============================================================
// Handler — nơi duy nhất quyết định response và log
// ============================================================

func processReservation(service *InventoryService, productID, quantity int) {
    fmt.Printf("Đặt hàng: productID=%d, quantity=%d\n", productID, quantity)

    err := service.ReserveItem(productID, quantity)
    if err != nil {
        switch {
        case errors.Is(err, ErrProductNotFound):
            fmt.Printf("  → [404] Sản phẩm không tồn tại\n")

        case errors.Is(err, ErrInsufficientStock):
            fmt.Printf("  → [409] Hàng không đủ: %v\n", err)

        default:
            var invErr *InvalidQuantityError
            if errors.As(err, &invErr) {
                fmt.Printf("  → [400] %v\n", invErr)
            } else {
                fmt.Printf("  → [500] Lỗi hệ thống: %v\n", err)
            }
        }
    } else {
        fmt.Printf("  → [200] Đặt hàng thành công\n")
    }
    fmt.Println()
}

func main() {
    repo := NewInventoryRepository()
    service := NewInventoryService(repo)

    fmt.Println("=== Inventory Service Error Handling Demo ===\n")

    processReservation(service, 1, 3)    // Thành công: Laptop còn 5, đặt 3
    processReservation(service, 2, 1)    // Lỗi: Chuột hết hàng (stock=0)
    processReservation(service, 99, 1)   // Lỗi: Sản phẩm không tồn tại
    processReservation(service, 3, -5)   // Lỗi: Quantity âm
    processReservation(service, 1, 9999) // Lỗi: Quantity quá lớn
    processReservation(service, 1, 10)   // Lỗi: Laptop còn 2 (sau lần đặt trước), đặt 10
}
```

**Chạy thử:**
```
go run 03-exercise.go
```

**Kết quả mong đợi:**
```
=== Inventory Service Error Handling Demo ===

Đặt hàng: productID=1, quantity=3
  → [200] Đặt hàng thành công

Đặt hàng: productID=2, quantity=1
  → [409] Hàng không đủ: ...

Đặt hàng: productID=99, quantity=1
  → [404] Sản phẩm không tồn tại
...
```

---

### Bài tập 2 (nâng cao): Thêm retry logic với error classification

Mở rộng bài tập 1: thêm khái niệm "lỗi có thể retry" (transient) và "lỗi không thể retry" (permanent). Implement một `RetryableError` wrapper và một hàm `WithRetry` tự động retry khi gặp transient error.

```go
package main

import (
    "errors"
    "fmt"
    "time"
)

// ============================================================
// Transient error wrapper — lỗi tạm thời, có thể retry
// ============================================================

type TransientError struct {
    Cause      error
    RetryAfter time.Duration
}

func (e *TransientError) Error() string {
    return fmt.Sprintf("lỗi tạm thời (retry sau %v): %v", e.RetryAfter, e.Cause)
}

func (e *TransientError) Unwrap() error {
    return e.Cause
}

// Wrap một error thành transient
func AsTransient(err error, retryAfter time.Duration) error {
    return &TransientError{Cause: err, RetryAfter: retryAfter}
}

// ============================================================
// Retry logic
// ============================================================

func WithRetry(maxAttempts int, operation func() error) error {
    var lastErr error

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        err := operation()
        if err == nil {
            return nil // Thành công
        }

        lastErr = err

        // Kiểm tra xem có phải transient error không
        var transient *TransientError
        if errors.As(err, &transient) {
            fmt.Printf("  [Attempt %d/%d] Lỗi tạm thời, chờ %v rồi retry...\n",
                attempt, maxAttempts, transient.RetryAfter)
            // Trong production: time.Sleep(transient.RetryAfter)
            // Ở đây bỏ qua sleep để chạy nhanh
            continue
        }

        // Permanent error — không retry
        fmt.Printf("  [Attempt %d/%d] Lỗi vĩnh viễn, dừng retry\n", attempt, maxAttempts)
        return fmt.Errorf("WithRetry: %w", err)
    }

    return fmt.Errorf("WithRetry: đã thử %d lần nhưng thất bại: %w", maxAttempts, lastErr)
}

// ============================================================
// Giả lập external service
// ============================================================

var callCount int

func callExternalAPI(succeed bool) error {
    callCount++
    if succeed && callCount >= 3 {
        return nil // Thành công lần thứ 3
    }

    // Giả lập network timeout — transient
    networkErr := errors.New("network timeout")
    return AsTransient(networkErr, 100*time.Millisecond)
}

func callExternalAPIPermanentFail() error {
    return fmt.Errorf("API key không hợp lệ: %w", errors.New("unauthorized"))
}

// ============================================================
// Main
// ============================================================

func main() {
    fmt.Println("=== Test 1: Retry thành công sau vài lần ===")
    callCount = 0
    err := WithRetry(5, func() error {
        return callExternalAPI(true)
    })
    if err != nil {
        fmt.Println("Thất bại:", err)
    } else {
        fmt.Printf("Thành công sau %d lần gọi\n", callCount)
    }

    fmt.Println()
    fmt.Println("=== Test 2: Permanent error — không retry ===")
    err = WithRetry(5, func() error {
        return callExternalAPIPermanentFail()
    })
    if err != nil {
        fmt.Println("Thất bại:", err)
    }

    fmt.Println()
    fmt.Println("=== Test 3: Hết số lần retry ===")
    callCount = 0
    err = WithRetry(3, func() error {
        return callExternalAPI(false) // không bao giờ succeed
    })
    if err != nil {
        fmt.Println("Thất bại:", err)
    }
}
```

---

## Tóm tắt

Hệ thống error handling của Go không phải là limitation — đó là một lựa chọn thiết kế có chủ đích để làm cho code dễ đọc, dễ debug, và dễ maintain hơn trong dài hạn. Khi bạn cảm thấy bực bội vì phải viết `if err != nil` nhiều lần, hãy nhớ rằng mỗi lần như vậy là một điểm trong code mà behavior được tường minh hóa — không có con đường lỗi nào ẩn sau exception handler nhiều tầng.

Ba thói quen quan trọng nhất cần hình thành:
1. Luôn wrap error với context khi propagate lên tầng trên
2. Dùng `errors.Is()` và `errors.As()` thay vì so sánh hay type assertion trực tiếp
3. Chỉ log một lần, ở tầng entry point

---

**Next:** [04-goroutines-channels.md](04-goroutines-channels.md) — Concurrency trong Go: goroutines, channels, và cách tư duy khác với `async/await` của C#
