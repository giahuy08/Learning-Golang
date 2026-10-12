# 06 — Testing trong Go

> Dành cho lập trình viên C# đang học Go. Bài này giải thích **tại sao** Go làm vậy, không chỉ **làm thế nào**.

---

## 1. Triết lý testing trong Go — Không cần test framework

Nếu bạn đến từ C#, câu hỏi đầu tiên thường là: "Tôi cần cài NuGet package nào? xUnit? NUnit? MSTest?" Câu trả lời trong Go là: **không cần gì cả**.

Go tích hợp sẵn một test runner trong toolchain (`go test`), và thư viện chuẩn cung cấp package `testing` đủ mạnh để viết test thực tế. Lý do Go chọn hướng này là vì Go coi testing là một phần không thể tách rời của ngôn ngữ, không phải addon. Điều này giống như C# có `dotnet test`, nhưng khác ở chỗ bạn không cần cài thêm bất kỳ thứ gì — không có `[TestClass]`, không có `[Fact]`, không có assertion library mặc định.

Kiểu dữ liệu trung tâm là `*testing.T` — đây là object bạn dùng để báo cáo test thất bại và ghi log. Không có attribute, không có reflection magic, không có test runner phức tạp. Chỉ là một function Go bình thường với chữ ký đặc biệt.

**So sánh tư duy:**

| C# (xUnit) | Go |
|---|---|
| `[Fact] public void Test_Add()` | `func TestAdd(t *testing.T)` |
| `Assert.Equal(expected, actual)` | `t.Errorf("got %d, want %d", got, want)` |
| `dotnet test` | `go test ./...` |
| xUnit / NUnit / MSTest (NuGet) | `testing` (stdlib, built-in) |
| Test class / fixture | Chỉ là file `_test.go` |

---

## 2. File `_test.go` — Quy ước đặt tên và phạm vi package

Go dùng quy ước tên file thay vì attribute để xác định test code. Bất kỳ file nào có đuôi `_test.go` sẽ **chỉ được biên dịch khi chạy `go test`**, không bao giờ có mặt trong binary production. Đây là cách Go tách biệt code test mà không cần project riêng như C# (thường có `MyApp.Tests.csproj`).

Có hai cách đặt tên package trong file test:

**Cùng package (white-box testing):** Dùng `package calculator` — bạn có thể truy cập các identifier unexported (viết thường). Dùng khi bạn cần test internal logic.

**Package `_test` (black-box testing):** Dùng `package calculator_test` — bạn chỉ thấy những gì exported. Dùng khi bạn muốn test public API như một user thực sự. Đây là cách Go khuyến nghị vì nó ngăn bạn test implementation details.

```
mypackage/
  calculator.go          // package calculator
  calculator_test.go     // package calculator_test  (black-box)
  calculator_internal_test.go  // package calculator  (white-box, nếu cần)
```

---

## 3. Hàm test cơ bản: `func TestXxx(t *testing.T)`

Quy tắc: tên hàm phải bắt đầu bằng `Test` theo sau là chữ hoa (hoặc dấu gạch dưới). `go test` dùng reflection để tìm những function này.

```go
// BAD: tên không đúng quy tắc — go test sẽ bỏ qua hàm này
// package calculator_test
//
// func testAdd(t *testing.T) {  // chữ 't' thường — KHÔNG CHẠY
//     // ...
// }

// GOOD: tên đúng quy tắc
package calculator_test

import (
    "testing"
)

// Giả sử có hàm Add trong package calculator
// func Add(a, b int) int { return a + b }

func TestAdd_BasicCase(t *testing.T) {
    got := 2 + 2 // giả lập Add(2, 2) cho ví dụ này tự chạy được
    want := 4
    if got != want {
        // t.Errorf ghi log lỗi nhưng test TIẾP TỤC chạy
        t.Errorf("Add(2, 2) = %d, want %d", got, want)
    }
}
```

Chạy tất cả test trong project: `go test ./...`  
Chạy test cụ thể: `go test -run TestAdd_BasicCase ./...`  
Xem output chi tiết: `go test -v ./...`

---

## 4. `t.Error` vs `t.Fatal` — Sự khác biệt quan trọng

Đây là điểm mà nhiều người C# hay nhầm. Trong xUnit, khi một assertion fail, nó throw exception và test dừng ngay. Trong Go, bạn kiểm soát điều này một cách tường minh.

`t.Error` / `t.Errorf`: Đánh dấu test là FAIL nhưng **tiếp tục chạy phần còn lại** của hàm test. Dùng khi bạn muốn thu thập nhiều lỗi cùng lúc.

`t.Fatal` / `t.Fatalf`: Đánh dấu test là FAIL và **dừng ngay hàm test đó** (gọi `runtime.Goexit()` bên trong). Dùng khi lỗi đầu tiên làm cho các bước sau vô nghĩa — ví dụ kết nối DB fail thì không cần chạy tiếp.

```go
package main

import (
    "testing"
)

// BAD: dùng Fatal khi lẽ ra nên dùng Error
// Nếu check đầu tiên fail, bạn mất đi thông tin về check thứ hai
func TestMultipleChecks_Bad(t *testing.T) {
    result := map[string]int{"a": 1, "b": 2}

    if result["a"] != 1 {
        t.Fatalf("a sai: got %d", result["a"]) // Dừng ở đây nếu fail
    }
    if result["b"] != 2 {
        t.Fatalf("b sai: got %d", result["b"]) // Sẽ không chạy nếu check trên fail
    }
}

// GOOD: dùng Error để thu thập TẤT CẢ lỗi trong một lần chạy
func TestMultipleChecks_Good(t *testing.T) {
    result := map[string]int{"a": 1, "b": 2}

    if result["a"] != 1 {
        t.Errorf("a sai: got %d, want 1", result["a"]) // Tiếp tục chạy
    }
    if result["b"] != 2 {
        t.Errorf("b sai: got %d, want 2", result["b"]) // Luôn được kiểm tra
    }
}

// GOOD: dùng Fatal khi setup fail làm mọi thứ sau vô nghĩa
func TestWithSetup_Good(t *testing.T) {
    conn, err := setupTestDB() // giả lập
    if err != nil {
        t.Fatalf("không thể kết nối DB: %v", err) // Dừng là đúng — không có gì để test
    }
    defer conn.Close()

    // Các test sau đây có ý nghĩa vì conn đã sẵn sàng
    if conn == nil {
        t.Error("conn không được nil")
    }
}

// Hàm giả lập để code trên biên dịch được
type fakeConn struct{}
func (f *fakeConn) Close() {}
func setupTestDB() (*fakeConn, error) { return &fakeConn{}, nil }
```

---

## 5. `t.Run()` — Subtest để tổ chức

`t.Run()` cho phép bạn nhóm các test liên quan dưới một tên cha. Tương đương với `[Theory]` + `[InlineData]` của xUnit nhưng linh hoạt hơn vì bạn kiểm soát hoàn toàn vòng lặp.

Mỗi subtest chạy trong goroutine riêng và có `*testing.T` riêng. Bạn có thể chạy từng subtest riêng lẻ: `go test -run "TestMath/addition"`.

```go
package main

import "testing"

func TestMath(t *testing.T) {
    // Subtest cho phép tổ chức theo nhóm logic
    t.Run("addition", func(t *testing.T) {
        got := 1 + 1
        if got != 2 {
            t.Errorf("1+1 = %d, want 2", got)
        }
    })

    t.Run("subtraction", func(t *testing.T) {
        got := 5 - 3
        if got != 2 {
            t.Errorf("5-3 = %d, want 2", got)
        }
    })

    // Subtest lồng nhau cũng được
    t.Run("multiplication", func(t *testing.T) {
        t.Run("positive numbers", func(t *testing.T) {
            if 3*4 != 12 {
                t.Error("3*4 phải bằng 12")
            }
        })
        t.Run("with zero", func(t *testing.T) {
            if 3*0 != 0 {
                t.Error("3*0 phải bằng 0")
            }
        })
    })
}
```

---

### 5.1. `t.Parallel()` — Chạy test song song

Mặc định, `go test` chạy các test tuần tự trong một package. Gọi `t.Parallel()` ở đầu một test (hoặc subtest) báo cho testing framework: "test này có thể chạy đồng thời với các test khác cũng gọi `t.Parallel()`". Test sẽ tạm dừng đến khi mọi test tuần tự (không gọi `t.Parallel()`) chạy xong, sau đó tất cả test parallel chạy cùng lúc.

```go
func TestSlow(t *testing.T) {
    t.Parallel() // đánh dấu test này chạy song song với các test parallel khác

    // ... test logic tốn thời gian (I/O, sleep giả lập...)
}
```

Với table-driven test, gọi `t.Parallel()` bên trong closure của `t.Run()`:

```go
func TestMath(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"add small", 1, 1, 2},
        {"add large", 1000, 2000, 3000},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // mỗi subtest chạy song song với subtest khác
            if got := tt.a + tt.b; got != tt.want {
                t.Errorf("got %d, want %d", got, tt.want)
            }
        })
    }
}
```

Khi nào nên dùng: test suite có nhiều test I/O-bound (gọi HTTP, đọc file, query DB test container) — chạy song song giảm đáng kể tổng thời gian CI. Khi nào KHÔNG nên: test dùng shared mutable state (biến global, file cố định, cổng cố định) mà không có synchronization — parallel sẽ gây race hoặc flaky test. Luôn kết hợp với `go test -race` để chắc chắn không có race condition ẩn khi bật parallel.

---

## 6. Table-driven tests — Pattern quan trọng nhất trong Go testing

Đây là pattern bạn sẽ thấy trong hầu hết mọi codebase Go thực tế. Thay vì viết nhiều hàm test riêng lẻ cho từng case, bạn định nghĩa một slice of struct chứa tất cả input/output, rồi lặp qua chúng.

Tại sao Go yêu thích pattern này? Vì nó tách biệt rõ ràng giữa **dữ liệu test** và **logic kiểm tra**. Khi thêm case mới, bạn chỉ thêm một dòng vào bảng, không cần thêm function mới. Trong C#, xUnit có `[Theory]` + `[InlineData]` nhưng bị giới hạn ở compile-time constants. Go cho phép bạn dùng bất kỳ giá trị nào kể cả struct, slice, function.

```go
package main

import (
    "errors"
    "testing"
)

// Hàm cần test
func Divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("chia cho 0")
    }
    return a / b, nil
}

func TestDivide(t *testing.T) {
    // Định nghĩa bảng test case — đây là trái tim của pattern
    tests := []struct {
        name    string  // tên mô tả case này
        a, b    float64 // input
        want    float64 // kết quả mong đợi
        wantErr bool    // có mong đợi lỗi không
    }{
        // Các happy path cases
        {name: "chia đều", a: 10, b: 2, want: 5, wantErr: false},
        {name: "chia lẻ", a: 7, b: 2, want: 3.5, wantErr: false},
        {name: "số âm", a: -6, b: 3, want: -2, wantErr: false},
        {name: "chia 1", a: 5, b: 1, want: 5, wantErr: false},

        // Error cases
        {name: "chia cho 0", a: 5, b: 0, want: 0, wantErr: true},
        {name: "cả hai là 0", a: 0, b: 0, want: 0, wantErr: true},
    }

    // Chạy từng case như một subtest
    for _, tt := range tests {
        tt := tt // Quan trọng: capture loop variable (Go < 1.22)
        t.Run(tt.name, func(t *testing.T) {
            got, err := Divide(tt.a, tt.b)

            // Kiểm tra lỗi trước
            if (err != nil) != tt.wantErr {
                t.Fatalf("Divide(%v, %v) error = %v, wantErr %v",
                    tt.a, tt.b, err, tt.wantErr)
            }

            // Nếu mong đợi lỗi, không cần kiểm tra giá trị trả về
            if tt.wantErr {
                return
            }

            if got != tt.want {
                t.Errorf("Divide(%v, %v) = %v, want %v",
                    tt.a, tt.b, got, tt.want)
            }
        })
    }
}
```

**Lưu ý `tt := tt`:** Trước Go 1.22, biến loop được share trong tất cả iterations. Nếu subtest chạy parallel (`t.Parallel()`), tất cả sẽ dùng giá trị cuối cùng của `tt`. Capture lại là best practice. Từ Go 1.22 trở đi, vòng lặp `for range` tạo biến mới mỗi iteration nên không cần nữa, nhưng bạn vẫn sẽ thấy pattern này trong code cũ.

---

## 7. Thư viện Testify — Khi nào dùng thay stdlib

Testify (`github.com/stretchr/testify`) là thư viện testing phổ biến nhất trong Go. Nó cung cấp assertion functions quen thuộc hơn với developer C#. Cài: `go get github.com/stretchr/testify`.

Testify có hai package chính:
- `assert`: Khi assertion fail, test **tiếp tục** (giống `t.Error`)
- `require`: Khi assertion fail, test **dừng lại** (giống `t.Fatal`)

Dùng Testify khi: message lỗi của stdlib quá verbose, bạn cần deep equal cho struct/slice, hoặc team quen với fluent assertion style. Không dùng khi: project đơn giản, muốn zero dependencies, hoặc stdlib đủ rõ ràng.

```go
package main

import (
    "errors"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type User struct {
    ID   int
    Name string
    Age  int
}

func GetUser(id int) (*User, error) {
    if id <= 0 {
        return nil, errors.New("id không hợp lệ")
    }
    return &User{ID: id, Name: "Alice", Age: 30}, nil
}

// BAD: stdlib thuần — message lỗi kém rõ ràng với struct phức tạp
func TestGetUser_Stdlib(t *testing.T) {
    user, err := GetUser(1)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // Phải tự so sánh từng field — dễ bỏ sót
    if user.ID != 1 {
        t.Errorf("ID sai: got %d, want 1", user.ID)
    }
    if user.Name != "Alice" {
        t.Errorf("Name sai: got %s, want Alice", user.Name)
    }
}

// GOOD: Testify — rõ ràng, message lỗi tốt hơn, deep equal tự động
func TestGetUser_Testify(t *testing.T) {
    // require.NoError dừng test ngay nếu có lỗi (tương đương t.Fatal)
    user, err := GetUser(1)
    require.NoError(t, err, "GetUser không nên trả về lỗi với ID hợp lệ")

    // assert.Equal so sánh deep equal — hoạt động với struct, slice, map
    expected := &User{ID: 1, Name: "Alice", Age: 30}
    assert.Equal(t, expected, user)

    // Các assertion hữu ích khác
    assert.NotNil(t, user)
    assert.Equal(t, 30, user.Age)
}

func TestGetUser_InvalidID(t *testing.T) {
    user, err := GetUser(-1)

    // require vì nếu err == nil, test bên dưới không có nghĩa
    require.Error(t, err, "phải có lỗi với ID âm")
    assert.Nil(t, user)
    assert.Contains(t, err.Error(), "không hợp lệ")
}
```

---

## 8. Testing interfaces để mock — Không cần mock framework

Đây là một trong những điểm Go tỏa sáng so với C#. Trong C#, bạn thường cần Moq hoặc NSubstitute để mock dependencies. Trong Go, vì interface được implement implicitly (không cần từ khóa `implements`), bạn chỉ cần tạo một struct mới implement interface đó trong test file.

Lý do Go không cần mock framework: interface trong Go là structural typing — bất kỳ type nào có đúng method signatures đều thỏa mãn interface. Tạo một "fake" implementation chỉ cần vài dòng code.

```go
package main

import (
    "errors"
    "testing"
)

// --- Production code ---

// Interface được định nghĩa ở production code
type EmailSender interface {
    Send(to, subject, body string) error
}

type NotificationService struct {
    emailSender EmailSender
}

func NewNotificationService(sender EmailSender) *NotificationService {
    return &NotificationService{emailSender: sender}
}

func (n *NotificationService) NotifyUser(userEmail, message string) error {
    if userEmail == "" {
        return errors.New("email không được rỗng")
    }
    return n.emailSender.Send(userEmail, "Thông báo", message)
}

// --- Test code (trong file _test.go) ---

// BAD: Test trực tiếp với real email sender — slow, side effects, cần config
// func TestNotify_Bad(t *testing.T) {
//     realSender := smtp.NewSender("smtp.gmail.com", ...)  // Cần server thật!
//     svc := NewNotificationService(realSender)
//     err := svc.NotifyUser("test@example.com", "hello")
//     // Test này gửi email thật — không ai muốn điều này
// }

// GOOD: Tạo fake implementation trong test file
type fakeEmailSender struct {
    // Lưu lại những gì đã được gọi để assert
    sentTo      string
    sentSubject string
    sentBody    string
    callCount   int

    // Cho phép test kiểm soát hành vi
    shouldFail bool
}

// fakeEmailSender tự động thỏa mãn interface EmailSender
// vì nó có đúng method signature — không cần khai báo gì thêm
func (f *fakeEmailSender) Send(to, subject, body string) error {
    if f.shouldFail {
        return errors.New("smtp connection refused")
    }
    f.sentTo = to
    f.sentSubject = subject
    f.sentBody = body
    f.callCount++
    return nil
}

func TestNotifyUser_Success(t *testing.T) {
    fake := &fakeEmailSender{}
    svc := NewNotificationService(fake)

    err := svc.NotifyUser("user@example.com", "Chào bạn!")

    if err != nil {
        t.Fatalf("không mong đợi lỗi: %v", err)
    }
    if fake.callCount != 1 {
        t.Errorf("Send phải được gọi 1 lần, thực tế: %d", fake.callCount)
    }
    if fake.sentTo != "user@example.com" {
        t.Errorf("sentTo sai: got %s", fake.sentTo)
    }
}

func TestNotifyUser_EmptyEmail(t *testing.T) {
    fake := &fakeEmailSender{}
    svc := NewNotificationService(fake)

    err := svc.NotifyUser("", "message")

    if err == nil {
        t.Fatal("phải có lỗi khi email rỗng")
    }
    if fake.callCount != 0 {
        t.Error("Send không được gọi khi email rỗng")
    }
}

func TestNotifyUser_SenderFails(t *testing.T) {
    fake := &fakeEmailSender{shouldFail: true}
    svc := NewNotificationService(fake)

    err := svc.NotifyUser("user@example.com", "message")

    if err == nil {
        t.Fatal("phải trả về lỗi khi sender fail")
    }
}
```

---

## 9. Package `httptest` — Test HTTP handler không cần server thật

Trong C#, bạn thường dùng `WebApplicationFactory` hoặc `HttpClient` với mock để test HTTP handler. Go cung cấp package `net/http/httptest` trong stdlib để làm điều tương tự mà không cần spin up server thật.

`httptest.NewRecorder()` tạo một `http.ResponseWriter` giả lập — nó ghi lại status code, headers, và body thay vì gửi qua network. `httptest.NewRequest()` tạo `*http.Request` giả lập.

```go
package main

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

// --- Production code ---

type Product struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Price float64 `json:"price"`
}

func GetProductHandler(w http.ResponseWriter, r *http.Request) {
    // Giả lập: chỉ hỗ trợ GET
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    product := Product{ID: 1, Name: "Laptop", Price: 999.99}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(product)
}

func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
    var p Product
    if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
        http.Error(w, "invalid body", http.StatusBadRequest)
        return
    }
    p.ID = 42 // Giả lập DB assign ID
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(p)
}

// --- Test code ---

func TestGetProductHandler(t *testing.T) {
    // Tạo request và response recorder giả — không có network
    req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
    rec := httptest.NewRecorder()

    // Gọi handler trực tiếp như function bình thường
    GetProductHandler(rec, req)

    // Kiểm tra response
    if rec.Code != http.StatusOK {
        t.Errorf("status code sai: got %d, want %d", rec.Code, http.StatusOK)
    }

    contentType := rec.Header().Get("Content-Type")
    if !strings.Contains(contentType, "application/json") {
        t.Errorf("Content-Type sai: %s", contentType)
    }

    var product Product
    if err := json.NewDecoder(rec.Body).Decode(&product); err != nil {
        t.Fatalf("không thể decode response: %v", err)
    }

    if product.Name != "Laptop" {
        t.Errorf("Name sai: got %s, want Laptop", product.Name)
    }
}

func TestGetProductHandler_WrongMethod(t *testing.T) {
    req := httptest.NewRequest(http.MethodPost, "/products/1", nil)
    rec := httptest.NewRecorder()

    GetProductHandler(rec, req)

    if rec.Code != http.StatusMethodNotAllowed {
        t.Errorf("phải trả về 405, got %d", rec.Code)
    }
}

func TestCreateProductHandler(t *testing.T) {
    body := `{"name":"Mouse","price":29.99}`
    req := httptest.NewRequest(http.MethodPost, "/products",
        strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    CreateProductHandler(rec, req)

    if rec.Code != http.StatusCreated {
        t.Errorf("phải trả về 201, got %d", rec.Code)
    }

    var created Product
    json.NewDecoder(rec.Body).Decode(&created)
    if created.ID != 42 {
        t.Errorf("ID phải là 42 (từ DB giả), got %d", created.ID)
    }
}

// Test với httptest.Server — dùng khi cần test client code (ít phổ biến hơn)
func TestWithRealServer(t *testing.T) {
    // Tạo server thật trên random port, tự đóng sau test
    server := httptest.NewServer(http.HandlerFunc(GetProductHandler))
    defer server.Close()

    // Bây giờ bạn có server URL thật để test HTTP client
    resp, err := http.Get(server.URL + "/products/1")
    if err != nil {
        t.Fatalf("request thất bại: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("status sai: %d", resp.StatusCode)
    }
}
```

---

## 10. `testing.B` — Benchmark

Benchmark trong Go là first-class citizen, không cần tool bên ngoài như BenchmarkDotNet của C#. Hàm benchmark có prefix `Benchmark` và nhận `*testing.B`.

`b.N` là số lần Go test runner quyết định chạy vòng lặp của bạn — nó tự động điều chỉnh để đạt kết quả ổn định. Bạn không set `b.N`, bạn chỉ dùng nó.

Chạy benchmark: `go test -bench=. ./...`  
Kèm memory stats: `go test -bench=. -benchmem ./...`  
Chạy benchmark cụ thể: `go test -bench=BenchmarkSort -benchtime=5s ./...`

```go
package main

import (
    "sort"
    "testing"
)

// Hai cách sắp xếp để so sánh
func bubbleSort(arr []int) []int {
    result := make([]int, len(arr))
    copy(result, arr)
    n := len(result)
    for i := 0; i < n-1; i++ {
        for j := 0; j < n-i-1; j++ {
            if result[j] > result[j+1] {
                result[j], result[j+1] = result[j+1], result[j]
            }
        }
    }
    return result
}

func stdSort(arr []int) []int {
    result := make([]int, len(arr))
    copy(result, arr)
    sort.Ints(result)
    return result
}

// BAD: Không reset timer khi có setup code phức tạp
func BenchmarkBubbleSort_Bad(b *testing.B) {
    // Setup tốn thời gian — nhưng không reset timer
    data := make([]int, 1000)
    for i := range data {
        data[i] = 1000 - i
    }
    // Thời gian setup sẽ được tính vào kết quả benchmark — sai!
    for i := 0; i < b.N; i++ {
        bubbleSort(data)
    }
}

// GOOD: Reset timer sau setup để chỉ đo thời gian thực tế
func BenchmarkBubbleSort(b *testing.B) {
    data := make([]int, 1000)
    for i := range data {
        data[i] = 1000 - i
    }
    b.ResetTimer() // Loại bỏ thời gian setup khỏi kết quả

    for i := 0; i < b.N; i++ {
        bubbleSort(data)
    }
}

func BenchmarkStdSort(b *testing.B) {
    data := make([]int, 1000)
    for i := range data {
        data[i] = 1000 - i
    }
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        stdSort(data)
    }
}

// Benchmark với sub-benchmarks để so sánh theo kích thước input
func BenchmarkSort_Sizes(b *testing.B) {
    sizes := []int{10, 100, 1000, 10000}
    for _, size := range sizes {
        size := size
        b.Run("bubble_"+string(rune('0'+size/1000)), func(b *testing.B) {
            data := make([]int, size)
            for i := range data {
                data[i] = size - i
            }
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                bubbleSort(data)
            }
        })
    }
}
```

Output mẫu của `go test -bench=. -benchmem`:
```
BenchmarkBubbleSort-8     1234    987654 ns/op    8192 B/op    1 allocs/op
BenchmarkStdSort-8       89234     13456 ns/op    8192 B/op    1 allocs/op
```

---

## 11. Test helpers — `t.Helper()` và setup/teardown

`t.Helper()` đánh dấu hàm hiện tại là helper — khi test fail bên trong helper, Go sẽ báo dòng lỗi ở nơi gọi helper chứ không phải bên trong helper. Đây là điểm tương đương với việc dùng custom assertion trong C#.

Go không có `[SetUp]` / `[TearDown]` như NUnit, nhưng có `TestMain` để setup/teardown cho cả package, và `defer` trong từng test để cleanup.

```go
package main

import (
    "os"
    "path/filepath"
    "testing"
)

// BAD: Không có t.Helper() — khi fail, báo dòng lỗi trong assertStringEqual
// chứ không phải ở nơi gọi nó (vô ích!)
func assertStringEqual_Bad(t *testing.T, got, want string) {
    if got != want {
        t.Errorf("got %q, want %q", got, want) // Dòng này được báo — khó debug
    }
}

// GOOD: t.Helper() làm cho error message trỏ đến caller
func assertStringEqual(t *testing.T, got, want string) {
    t.Helper() // Phải là dòng đầu tiên trong helper
    if got != want {
        t.Errorf("got %q, want %q", got, want) // Nhưng caller được báo
    }
}

// Helper để tạo temp file — tự cleanup với t.Cleanup
func createTempFile(t *testing.T, content string) string {
    t.Helper()

    dir := t.TempDir() // t.TempDir() tự xóa sau test — tốt hơn os.MkdirTemp
    path := filepath.Join(dir, "test.txt")

    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
        t.Fatalf("không tạo được temp file: %v", err)
    }

    // t.Cleanup đăng ký hàm chạy sau khi test xong — như defer nhưng scoped to test
    t.Cleanup(func() {
        // t.TempDir() đã xử lý cleanup, đây chỉ là ví dụ
        t.Logf("cleanup cho file: %s", path)
    })

    return path
}

func TestReadFile(t *testing.T) {
    path := createTempFile(t, "Hello, Go!")

    content, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("không đọc được file: %v", err)
    }

    assertStringEqual(t, string(content), "Hello, Go!")
    // Nếu fail, error message sẽ trỏ đến dòng này — KHÔNG phải vào trong assertStringEqual
}

// TestMain — Setup/Teardown cho toàn bộ package (chạy một lần)
// Tương đương [OneTimeSetUp] / [OneTimeTearDown] trong NUnit
// Uncomment khi dùng thực tế:
//
// func TestMain(m *testing.M) {
//     // Setup: khởi tạo DB, load config, etc.
//     setupIntegrationTestDB()
//
//     // m.Run() chạy tất cả test trong package
//     exitCode := m.Run()
//
//     // Teardown: dọn dẹp
//     teardownIntegrationTestDB()
//
//     os.Exit(exitCode)
// }
```

---

## 12. Test coverage — Đo độ phủ của test

Go có coverage tool tích hợp sẵn, không cần plugin hay tool riêng như dotCover hay Coverlet của C#.

Xem coverage trong terminal: `go test -cover ./...`  
Tạo coverage profile: `go test -coverprofile=coverage.out ./...`  
Xem chi tiết trong browser: `go tool cover -html=coverage.out`  
Xem theo function: `go tool cover -func=coverage.out`

```
--- Ví dụ output của go test -cover ---
ok  myproject/calculator    87.5% of statements
ok  myproject/httphandler   91.2% of statements
FAIL myproject/service      [build failed]
```

Coverage trong Go đo **statement coverage** (từng dòng code có được thực thi không), không phải branch coverage. Con số 100% không đảm bảo code đúng — đó chỉ là metric, không phải mục tiêu cuối cùng. Team Go khuyến nghị tập trung vào **test các behavior quan trọng** thay vì đuổi theo số phần trăm.

Để xem coverage theo function, chạy:
```
go tool cover -func=coverage.out
myproject/calculator/calc.go:5:  Add         100.0%
myproject/calculator/calc.go:9:  Divide      75.0%
total:                           (statements) 87.5%
```

---

## 13. Integration tests vs Unit tests — Build tags để tách biệt

Trong C# solution, bạn thường có project riêng `MyApp.IntegrationTests.csproj`. Go dùng **build tags** để đánh dấu file chỉ biên dịch khi có flag cụ thể. Điều này giữ mọi thứ trong cùng module nhưng vẫn tách biệt được.

Build tag phải là comment đầu tiên trong file, trước `package` declaration.

```go
//go:build integration

// File: user_repository_integration_test.go
// Chỉ chạy khi: go test -tags=integration ./...
// Unit test bình thường không bao gồm file này

package repository_test

import (
    "testing"
    // "database/sql"
    // _ "github.com/lib/pq"
)

// Test này cần PostgreSQL thật — không chạy trong CI bình thường
func TestUserRepository_Create_Integration(t *testing.T) {
    // db, err := sql.Open("postgres", os.Getenv("TEST_DATABASE_URL"))
    // ...
    t.Log("Integration test: kết nối DB thật, tạo user, kiểm tra kết quả")
    // Đây chỉ là placeholder — implement khi có DB thật
}
```

```go
// File: user_service_test.go  (KHÔNG có build tag)
// Chạy bình thường: go test ./...

package service_test

import "testing"

// Unit test — dùng fake/mock, không cần external dependencies
func TestUserService_Create_Unit(t *testing.T) {
    t.Log("Unit test: chạy bình thường, nhanh, không cần DB")
}
```

**Workflow thực tế:**
```bash
# CI pipeline: chỉ chạy unit tests (nhanh)
go test ./...

# CI pipeline riêng cho integration (cần DB, chậm hơn)
go test -tags=integration ./...

# Chạy tất cả
go test -tags=integration,e2e ./...
```

---

## 14. Fuzzing — Go 1.18+

Fuzzing là kỹ thuật tự động generate input ngẫu nhiên để tìm edge cases mà bạn không nghĩ đến. C# có SharpFuzz và FsCheck, nhưng Go tích hợp sẵn từ 1.18. Đây là điểm Go đi trước nhiều ngôn ngữ khác.

Fuzz function có prefix `Fuzz` và nhận `*testing.F`. Bạn cung cấp "seed corpus" (input mẫu), và fuzzer dùng chúng làm điểm khởi đầu để generate variations.

```go
package main

import (
    "testing"
    "unicode/utf8"
)

// Hàm cần fuzz test — đếm số ký tự UTF-8
func CountChars(s string) int {
    return utf8.RuneCountInString(s)
}

// Hàm có bug tiềm ẩn — fuzz sẽ tìm ra
func ReverseString(s string) string {
    runes := []rune(s)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }
    return string(runes)
}

func FuzzCountChars(f *testing.F) {
    // Seed corpus — input mẫu để fuzzer bắt đầu
    f.Add("hello")
    f.Add("")
    f.Add("xin chào") // UTF-8 multibyte
    f.Add("🎉🚀")      // Emoji

    // Fuzzer sẽ generate các variation dựa trên seed corpus
    f.Fuzz(func(t *testing.T, s string) {
        // Invariant: CountChars phải >= 0
        count := CountChars(s)
        if count < 0 {
            t.Errorf("CountChars(%q) trả về âm: %d", s, count)
        }

        // Invariant: Reverse của Reverse phải bằng original
        doubled := ReverseString(ReverseString(s))
        if doubled != s {
            t.Errorf("ReverseString(ReverseString(%q)) = %q, want %q",
                s, doubled, s)
        }
    })
}
```

Chạy fuzz test:
```bash
# Chạy như test thông thường (chỉ dùng seed corpus) — chạy nhanh
go test -run FuzzCountChars ./...

# Chạy fuzzing thật sự — generate random input (chạy mãi cho đến khi Ctrl+C)
go test -fuzz=FuzzCountChars ./...

# Chạy fuzzing trong 30 giây
go test -fuzz=FuzzCountChars -fuzztime=30s ./...
```

Khi fuzzer tìm thấy input gây crash, nó lưu vào `testdata/fuzz/FuzzCountChars/` — những file này tự động trở thành regression test cho các lần chạy sau.

---

## 14.1. `testing/synctest` (Go 1.24+, ổn định từ Go 1.25) — Test code concurrent không cần chờ thời gian thật

**Vấn đề kinh điển:** test code dùng `time.Sleep`, `context.WithTimeout`, hay `time.Timer` buộc bạn chọn 1 trong 2 điều tệ: (1) test chạy **chậm** vì phải chờ thời gian thật (`time.Sleep(5 * time.Second)` trong test = 5 giây thật mỗi lần chạy CI), hoặc (2) test **flaky** vì cố rút ngắn timeout xuống vài ms để chạy nhanh, nhưng máy CI chậm bất thường một hôm nào đó làm test fail sai.

```go
// BAD: test chậm và có thể flaky
func TestRetryWithBackoff(t *testing.T) {
    start := time.Now()
    err := retryWithBackoff(3, 100*time.Millisecond) // thực sự sleep tổng ~700ms
    if time.Since(start) < 700*time.Millisecond {
        t.Error("backoff không đủ thời gian")
    }
    _ = err
}
```

`testing/synctest` giải quyết bằng cách chạy test bên trong một **bubble** có đồng hồ giả (fake clock): mọi goroutine trong bubble dùng chung một "thời gian ảo", `time.Sleep`/`timer` bên trong bubble **trôi qua tức thì** về mặt thời gian thật, nhưng logic thứ tự (goroutine nào chạy trước/sau theo thời gian) vẫn đúng y hệt production.

```go
import "testing/synctest"

func TestRetryWithBackoff(t *testing.T) {
    synctest.Run(func() {
        start := time.Now() // thời gian ảo bên trong bubble
        err := retryWithBackoff(3, 100*time.Millisecond)

        // Test chạy XONG NGAY LẬP TỨC về mặt thời gian thật (vài microsecond),
        // nhưng time.Since(start) vẫn trả về ~700ms vì đồng hồ ảo đã "trôi" đúng vậy
        if time.Since(start) < 700*time.Millisecond {
            t.Error("backoff không đủ thời gian")
        }
        _ = err
    })
}
```

Bên trong `synctest.Run`, khi mọi goroutine đều đang "chờ" (blocked trên channel, timer, hoặc `sync` primitive), runtime tự động **tua nhanh đồng hồ ảo** đến thời điểm sự kiện tiếp theo sẽ xảy ra — không có `time.Sleep` thật nào chạy cả. Điều này biến test concurrent vốn chậm/flaky thành **nhanh và deterministic**.

**Lưu ý version:** ở Go 1.24, tính năng này còn experimental, cần build với `GOEXPERIMENT=synctest`. Từ Go 1.25, `testing/synctest` đã ổn định, dùng trực tiếp không cần cờ gì thêm.

---

## 15. Sai lầm phổ biến: Test implementation details thay vì behavior

Đây là vấn đề xuất hiện ở mọi ngôn ngữ, nhưng đặc biệt phổ biến khi developer từ C# chuyển sang Go và muốn verify "mọi thứ được gọi đúng".

```go
package main

import (
    "errors"
    "fmt"
    "testing"
)

// --- Code cần test ---

type Cache interface {
    Get(key string) (string, bool)
    Set(key string, value string)
}

type UserService struct {
    cache Cache
    db    UserDB
}

type UserDB interface {
    FindByID(id int) (string, error)
}

func (s *UserService) GetUsername(id int) (string, error) {
    // Implementation: check cache trước, rồi DB
    key := fmt.Sprintf("user:%d", id)
    if name, ok := s.cache.Get(key); ok {
        return name, nil
    }
    name, err := s.db.FindByID(id)
    if err != nil {
        return "", err
    }
    s.cache.Set(key, name)
    return name, nil
}

// --- Fake implementations ---

type fakeCache struct {
    data        map[string]string
    getCalled   int  // BAD: đếm số lần gọi — implementation detail
    setCalled   int  // BAD: như trên
}

func (f *fakeCache) Get(key string) (string, bool) {
    f.getCalled++
    v, ok := f.data[key]
    return v, ok
}

func (f *fakeCache) Set(key, value string) {
    f.setCalled++
    if f.data == nil {
        f.data = make(map[string]string)
    }
    f.data[key] = value
}

type fakeDB struct {
    users map[int]string
}

func (f *fakeDB) FindByID(id int) (string, error) {
    name, ok := f.users[id]
    if !ok {
        return "", errors.New("not found")
    }
    return name, nil
}

// BAD: Test implementation detail — "cache.Get phải được gọi đúng 1 lần"
// Test này sẽ fail nếu bạn refactor implementation (dù behavior vẫn đúng)
func TestGetUsername_BAD(t *testing.T) {
    cache := &fakeCache{data: map[string]string{"user:1": "Alice"}}
    db := &fakeDB{}
    svc := &UserService{cache: cache, db: db}

    svc.GetUsername(1)

    // Đây là test implementation — bạn test CÁI GÌ xảy ra bên trong
    // thay vì KẾT QUẢ như thế nào
    if cache.getCalled != 1 {
        t.Errorf("Get phải được gọi 1 lần, got %d", cache.getCalled)
    }
    if cache.setCalled != 0 {
        t.Errorf("Set không được gọi khi cache hit, got %d", cache.setCalled)
    }
}

// GOOD: Test behavior — "với user có trong cache, trả về đúng tên"
func TestGetUsername_CacheHit_GOOD(t *testing.T) {
    cache := &fakeCache{data: map[string]string{"user:1": "Alice"}}
    db := &fakeDB{users: map[int]string{1: "Wrong"}} // DB có giá trị sai
    svc := &UserService{cache: cache, db: db}

    name, err := svc.GetUsername(1)

    // Test KẾT QUẢ — không quan tâm cache hay DB được gọi mấy lần
    if err != nil {
        t.Fatalf("không mong đợi lỗi: %v", err)
    }
    if name != "Alice" {
        t.Errorf("phải trả về tên từ cache: got %q, want %q", name, "Alice")
    }
}

func TestGetUsername_DBFallback_GOOD(t *testing.T) {
    cache := &fakeCache{data: map[string]string{}} // Cache rỗng
    db := &fakeDB{users: map[int]string{2: "Bob"}}
    svc := &UserService{cache: cache, db: db}

    name, err := svc.GetUsername(2)

    if err != nil {
        t.Fatalf("không mong đợi lỗi: %v", err)
    }
    if name != "Bob" {
        t.Errorf("phải trả về tên từ DB: got %q, want %q", name, "Bob")
    }
    // Có thể verify cache đã được set — đây là BEHAVIOR, không phải implementation detail
    cached, ok := cache.data["user:2"]
    if !ok || cached != "Bob" {
        t.Error("sau khi fetch từ DB, user phải được cache lại")
    }
}
```

---

## Bảng các lỗi phổ biến

| Lỗi | Vấn đề | Cách sửa |
|---|---|---|
| `func testAdd(t *testing.T)` | Chữ `t` thường — test bị bỏ qua | Đổi thành `TestAdd` |
| Không có `t.Helper()` trong helper | Error message trỏ vào helper, khó debug | Thêm `t.Helper()` làm dòng đầu tiên |
| Dùng `t.Fatal` cho mọi assertion | Mất thông tin về nhiều lỗi cùng lúc | Dùng `t.Error` cho independent checks, `t.Fatal` khi setup fail |
| Quên `tt := tt` (Go < 1.22) | Parallel subtests dùng chung biến loop | Capture với `tt := tt` ngay đầu loop body |
| Test implementation thay vì behavior | Test bị break khi refactor dù code đúng | Test input/output, không test "bao nhiêu lần gọi" |
| Không dùng `b.ResetTimer()` | Thời gian setup tính vào benchmark | Gọi `b.ResetTimer()` sau khi setup xong |
| File test không có `_test.go` suffix | Code test bị compile vào binary production | Luôn đặt tên `*_test.go` |
| Import cycle trong test | `package foo` test import package cần `foo` | Dùng `package foo_test` (external test package) |
| Không dùng `t.TempDir()` | Temp files không được dọn dẹp | Dùng `t.TempDir()` thay vì `os.MkdirTemp` thủ công |
| Benchmark không đủ lần chạy | `b.N` = 1, kết quả không ổn định | Để Go tự quyết định `b.N`, không set thủ công |

---

## Best practices

Viết test trước khi code (TDD) không phải yêu cầu bắt buộc trong Go community, nhưng table-driven tests nên là default choice của bạn cho mọi function có nhiều hơn hai case. Khi bạn thấy mình copy-paste một hàm test với input khác nhau, đó là dấu hiệu bạn cần chuyển sang table-driven.

Mỗi test nên kiểm tra **một behavior**, không phải một function. Một function `Divide` có thể cần nhiều test: chia bình thường, chia cho 0, chia số âm. Đặt tên test mô tả behavior: `TestDivide_ByZero_ReturnsError` rõ ràng hơn `TestDivide2`.

Giữ test **independent** với nhau — mỗi test tự setup và tự cleanup. State dùng chung giữa các test là nguồn gốc của những bug khó tìm nhất. `t.Cleanup()` và `t.TempDir()` là công cụ Go cung cấp để làm điều này dễ dàng.

Với Testify, quy tắc đơn giản là: dùng `require` khi lỗi làm cho các assertion sau vô nghĩa (setup, creation), dùng `assert` cho tất cả còn lại. Đừng mix `require` từ một package và `assert` từ package khác trong cùng một test — luôn import từ `testify/assert` và `testify/require`.

Coverage 80% thường là sweet spot trong thực tế. Đừng tốn thời gian đuổi theo 100% — thay vào đó đảm bảo **các path quan trọng** (error handling, business logic, edge cases) được cover.

---

## Bài tập thực hành

### Bài tập 1: Viết test đầy đủ cho Calculator package

Tạo file `calculator/calculator.go` và `calculator/calculator_test.go`. Implement và test các hàm sau với table-driven tests.

**Yêu cầu:**
- `Add(a, b float64) float64`
- `Subtract(a, b float64) float64`
- `Multiply(a, b float64) float64`
- `Divide(a, b float64) (float64, error)` — error khi b = 0
- `Percent(value, total float64) (float64, error)` — error khi total = 0

**Giải:**

`calculator/calculator.go`:
```go
package calculator

import (
    "errors"
    "fmt"
)

// ErrDivisionByZero được export để caller có thể check bằng errors.Is
var ErrDivisionByZero = errors.New("chia cho 0 không được phép")

func Add(a, b float64) float64 {
    return a + b
}

func Subtract(a, b float64) float64 {
    return a - b
}

func Multiply(a, b float64) float64 {
    return a * b
}

func Divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, fmt.Errorf("Divide(%v, %v): %w", a, b, ErrDivisionByZero)
    }
    return a / b, nil
}

func Percent(value, total float64) (float64, error) {
    if total == 0 {
        return 0, fmt.Errorf("Percent: %w", ErrDivisionByZero)
    }
    return (value / total) * 100, nil
}
```

`calculator/calculator_test.go`:
```go
package calculator_test

import (
    "errors"
    "math"
    "testing"

    "myproject/calculator" // thay bằng module path thực tế của bạn
)

func TestAdd(t *testing.T) {
    tests := []struct {
        name string
        a, b float64
        want float64
    }{
        {name: "cộng hai số dương", a: 2, b: 3, want: 5},
        {name: "cộng với số âm", a: -1, b: 4, want: 3},
        {name: "cộng hai số âm", a: -2, b: -3, want: -5},
        {name: "cộng với 0", a: 5, b: 0, want: 5},
        {name: "cộng số thập phân", a: 1.5, b: 2.5, want: 4.0},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            got := calculator.Add(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("Add(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
            }
        })
    }
}

func TestSubtract(t *testing.T) {
    tests := []struct {
        name string
        a, b float64
        want float64
    }{
        {name: "trừ bình thường", a: 5, b: 3, want: 2},
        {name: "kết quả âm", a: 2, b: 5, want: -3},
        {name: "trừ số âm", a: 3, b: -2, want: 5},
        {name: "trừ cùng giá trị", a: 4, b: 4, want: 0},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            got := calculator.Subtract(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("Subtract(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
            }
        })
    }
}

func TestMultiply(t *testing.T) {
    tests := []struct {
        name string
        a, b float64
        want float64
    }{
        {name: "nhân hai số dương", a: 3, b: 4, want: 12},
        {name: "nhân với 0", a: 5, b: 0, want: 0},
        {name: "nhân hai số âm", a: -2, b: -3, want: 6},
        {name: "nhân số âm và dương", a: -2, b: 4, want: -8},
        {name: "nhân với 1", a: 7, b: 1, want: 7},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            got := calculator.Multiply(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("Multiply(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
            }
        })
    }
}

func TestDivide(t *testing.T) {
    tests := []struct {
        name    string
        a, b    float64
        want    float64
        wantErr bool
    }{
        {name: "chia bình thường", a: 10, b: 2, want: 5, wantErr: false},
        {name: "chia lẻ", a: 7, b: 2, want: 3.5, wantErr: false},
        {name: "chia số âm", a: -6, b: 3, want: -2, wantErr: false},
        {name: "chia cho 1", a: 8, b: 1, want: 8, wantErr: false},
        {name: "chia cho 0", a: 5, b: 0, want: 0, wantErr: true},
        {name: "0 chia cho 0", a: 0, b: 0, want: 0, wantErr: true},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            got, err := calculator.Divide(tt.a, tt.b)

            if tt.wantErr {
                if err == nil {
                    t.Fatal("mong đợi lỗi nhưng không có")
                }
                // Kiểm tra đúng loại lỗi bằng errors.Is
                if !errors.Is(err, calculator.ErrDivisionByZero) {
                    t.Errorf("sai loại lỗi: got %v, want ErrDivisionByZero", err)
                }
                return
            }

            if err != nil {
                t.Fatalf("không mong đợi lỗi: %v", err)
            }
            if got != tt.want {
                t.Errorf("Divide(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
            }
        })
    }
}

func TestPercent(t *testing.T) {
    tests := []struct {
        name        string
        value, total float64
        want        float64
        wantErr     bool
    }{
        {name: "50% của 200", value: 100, total: 200, want: 50, wantErr: false},
        {name: "100%", value: 50, total: 50, want: 100, wantErr: false},
        {name: "0%", value: 0, total: 100, want: 0, wantErr: false},
        {name: "33.33%", value: 1, total: 3, want: 33.333333333333336, wantErr: false},
        {name: "total bằng 0", value: 5, total: 0, want: 0, wantErr: true},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            got, err := calculator.Percent(tt.value, tt.total)

            if tt.wantErr {
                if err == nil {
                    t.Fatal("mong đợi lỗi nhưng không có")
                }
                return
            }

            if err != nil {
                t.Fatalf("không mong đợi lỗi: %v", err)
            }

            // Dùng tolerance cho số thập phân — floating point không exact
            const tolerance = 1e-9
            if math.Abs(got-tt.want) > tolerance {
                t.Errorf("Percent(%v, %v) = %v, want %v", tt.value, tt.total, got, tt.want)
            }
        })
    }
}

// Benchmark để đo hiệu năng
func BenchmarkDivide(b *testing.B) {
    for i := 0; i < b.N; i++ {
        calculator.Divide(1234567.89, 9876.54)
    }
}
```

Chạy:
```bash
# Tạo module trước
go mod init myproject

# Chạy tất cả test
go test ./calculator/...

# Chạy với coverage
go test -cover ./calculator/...

# Chạy verbose để thấy từng subtest
go test -v ./calculator/...

# Chạy benchmark
go test -bench=. -benchmem ./calculator/...
```

---

### Bài tập 2: Test HTTP handler với httptest

Tạo một HTTP handler đơn giản và viết test đầy đủ cho nó, bao gồm cả table-driven tests cho các HTTP methods và status codes.

**Giải:**

`handlers/health.go`:
```go
package handlers

import (
    "encoding/json"
    "net/http"
    "time"
)

type HealthResponse struct {
    Status    string `json:"status"`
    Timestamp string `json:"timestamp"`
    Version   string `json:"version"`
}

type HealthHandler struct {
    version   string
    timeNow   func() time.Time // Dependency injection cho time — dễ test
}

func NewHealthHandler(version string) *HealthHandler {
    return &HealthHandler{
        version: version,
        timeNow: time.Now, // Default: dùng real time
    }
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    response := HealthResponse{
        Status:    "ok",
        Timestamp: h.timeNow().UTC().Format(time.RFC3339),
        Version:   h.version,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}
```

`handlers/health_test.go`:
```go
package handlers_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "myproject/handlers"
)

func TestHealthHandler(t *testing.T) {
    // Thời gian cố định để test deterministic
    fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

    handler := handlers.NewHealthHandler("1.2.3")
    // Override time function để test không phụ thuộc vào real time
    handler.TimeNow = func() time.Time { return fixedTime }

    tests := []struct {
        name           string
        method         string
        wantStatusCode int
        wantBody       bool // có kiểm tra body không
    }{
        {
            name:           "GET trả về 200",
            method:         http.MethodGet,
            wantStatusCode: http.StatusOK,
            wantBody:       true,
        },
        {
            name:           "POST trả về 405",
            method:         http.MethodPost,
            wantStatusCode: http.StatusMethodNotAllowed,
            wantBody:       false,
        },
        {
            name:           "PUT trả về 405",
            method:         http.MethodPut,
            wantStatusCode: http.StatusMethodNotAllowed,
            wantBody:       false,
        },
        {
            name:           "DELETE trả về 405",
            method:         http.MethodDelete,
            wantStatusCode: http.StatusMethodNotAllowed,
            wantBody:       false,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, "/health", nil)
            rec := httptest.NewRecorder()

            handler.ServeHTTP(rec, req)

            if rec.Code != tt.wantStatusCode {
                t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatusCode)
            }

            if !tt.wantBody {
                return
            }

            // Kiểm tra Content-Type
            ct := rec.Header().Get("Content-Type")
            if ct != "application/json" {
                t.Errorf("Content-Type = %q, want application/json", ct)
            }

            // Decode và kiểm tra body
            var response struct {
                Status    string `json:"status"`
                Timestamp string `json:"timestamp"`
                Version   string `json:"version"`
            }
            if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
                t.Fatalf("không decode được response: %v", err)
            }

            if response.Status != "ok" {
                t.Errorf("status = %q, want ok", response.Status)
            }
            if response.Version != "1.2.3" {
                t.Errorf("version = %q, want 1.2.3", response.Version)
            }
            if response.Timestamp != "2024-01-15T10:30:00Z" {
                t.Errorf("timestamp = %q, want 2024-01-15T10:30:00Z", response.Timestamp)
            }
        })
    }
}
```

> Lưu ý: Để bài tập 2 compile được, bạn cần export `TimeNow` field trong `HealthHandler`. Thay `timeNow` thành `TimeNow` trong struct và các chỗ dùng nó trong `handlers/health.go`.

---

## Tổng kết nhanh

| Khái niệm | Go | C# tương đương |
|---|---|---|
| Test runner | `go test` (built-in) | `dotnet test` + test framework |
| Test discovery | Prefix `Test`, file `*_test.go` | Attributes `[Fact]`, `[Test]` |
| Assertion | `t.Errorf` / Testify | `Assert.Equal` / FluentAssertions |
| Mock | Fake struct implement interface | Moq / NSubstitute |
| Benchmark | `testing.B`, `go test -bench` | BenchmarkDotNet |
| Coverage | `go test -cover` | Coverlet / dotCover |
| HTTP test | `net/http/httptest` | `WebApplicationFactory` |
| Build separation | Build tags `//go:build` | Separate `.csproj` |
| Fuzzing | `testing.F`, `go test -fuzz` | SharpFuzz / FsCheck |

---

## Next

[07 — Defer, Panic & Recover →](07-defer-panic-recover.md)
