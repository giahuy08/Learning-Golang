# 29. Typed Nil Interface Trap — Bẫy interface có type nhưng value nil

> Liên quan: [02-structs-interfaces.md](02-structs-interfaces.md#L413), [03-error-handling.md](03-error-handling.md). Dành cho C# developer: đây là bug runtime kinh điển của Go, không có tương đương trực tiếp trong C# vì C# chỉ có một khái niệm `null` duy nhất.

---

## 1. Vấn đề cốt lõi

Một biến kiểu `interface` (ví dụ `error`) trong Go được lưu ở runtime dưới dạng cặp:

```
(Type, Value)
```

`interface == nil` chỉ đúng khi **cả Type lẫn Value đều nil**. Nếu Type đã có giá trị cụ thể (dù Value bên trong là nil), interface đó **không bằng nil**.

So sánh C#: `object obj = null` chỉ có một tầng null. Go tách "con trỏ nil" (`*T` bằng nil) và "interface nil" (cả type lẫn value đều nil) thành hai khái niệm khác nhau — đây chính là nguồn gốc của bug.

---

## 2. Ví dụ minh hoạ đầy đủ

```go
package main

import "fmt"

type MyError struct {
    Message string
}

func (e *MyError) Error() string {
    return e.Message
}

// BAD: trả về typed nil pointer qua interface
func getUserBad(id int) error {
    var err *MyError = nil // typed nil pointer, kiểu tĩnh *MyError

    if id < 0 {
        err = &MyError{Message: "invalid id"}
    }

    return err // đóng gói (*MyError)(nil) vào interface error
    // interface trả về = (Type: *MyError, Value: nil) -> KHÔNG bằng nil
}

// GOOD: trả về nil tường minh
func getUserGood(id int) error {
    if id < 0 {
        return &MyError{Message: "invalid id"}
    }
    return nil // interface = (Type: nil, Value: nil) -> nil thật
}

func main() {
    err := getUserBad(5) // id hợp lệ, không có lỗi thật sự
    if err != nil {
        fmt.Println("BAD: có lỗi báo sai:", err) // in ra: BAD: có lỗi báo sai: <nil>
    }

    err2 := getUserGood(5)
    if err2 != nil {
        fmt.Println("sẽ không bao giờ chạy tới đây")
    } else {
        fmt.Println("GOOD: không có lỗi, đúng như mong đợi")
    }
}
```

Chạy thử trên Go Playground (https://go.dev/play/) để tự thấy `BAD` in ra dòng lỗi giả dù `id = 5` hợp lệ.

---

## 3. Vì sao dễ xảy ra trong code thực tế

Bẫy này thường xuất hiện khi bạn dùng **con trỏ struct làm error type** kết hợp với biến trung gian có kiểu cụ thể, ví dụ pattern rất hay gặp:

```go
func validate(input Input) *ValidationError {
    if input.Name == "" {
        return &ValidationError{Field: "name"}
    }
    return nil
}

func process(input Input) error {
    err := validate(input) // err có kiểu *ValidationError, không phải error
    return err              // BUG: nếu validate trả nil, err vẫn "typed nil" khi box vào error
}
```

Ngay cả khi `validate` trả về `nil` đúng, dòng `return err` trong `process` vẫn đóng gói `(*ValidationError)(nil)` vào interface `error` — bug xảy ra ở lớp bọc ngoài, không phải ở hàm gốc.

---

## 4. Quy tắc phòng tránh

1. **Không bao giờ trả kiểu con trỏ cụ thể (`*T`) từ một hàm có chữ ký trả về interface** nếu bên trong có đường dẫn trả `nil` qua biến đó. Luôn kiểm tra rõ tại điểm bàn giao.
2. Nếu hàm nội bộ trả `*ValidationError`, hàm bọc ngoài phải kiểm tra tường minh trước khi gán vào `error`:

```go
func process(input Input) error {
    verr := validate(input)
    if verr != nil {
        return verr
    }
    return nil // luôn nil thật, không đóng gói typed nil
}
```

3. Khi khai báo biến trung gian dùng làm error, khai báo bằng kiểu `error` thay vì kiểu con trỏ cụ thể:

```go
var err error = nil // interface nil thật ngay từ đầu
if id < 0 {
    err = &MyError{Message: "invalid id"}
}
return err
```

4. `go vet` và một số linter (`nilness` trong `golangci-lint`) có thể phát hiện một số trường hợp, nhưng không bắt được hết — đây là lý do phải hiểu rõ cơ chế thay vì chỉ dựa vào tool.

---

## 5. Exercises

### Exercise 1 — Tái hiện bug

Viết một hàm `divide(a, b float64) (*CalcError, float64)` trả về con trỏ lỗi tùy chỉnh khi `b == 0`. Sau đó viết hàm bọc `safeDivide(a, b float64) error` gọi `divide` và trả kết quả lỗi ra ngoài mà **không** kiểm tra tường minh. Chứng minh bằng `if err != nil` rằng lỗi bị báo sai ngay cả khi `b != 0`.

### Exercise 2 — Sửa bug

Sửa lại `safeDivide` ở Exercise 1 theo đúng quy tắc mục 4 để `err != nil` chỉ đúng khi thực sự có lỗi chia cho 0.

### Exercise 3 — Viết unit test bắt bug này

Dùng `testing` package (xem [06-testing.md](06-testing.md)) viết test-case gọi `safeDivide(10, 2)` (không có lỗi) và assert `err == nil`. Test này phải **fail** với bản BAD và **pass** với bản GOOD — mục tiêu là luyện viết test phát hiện đúng loại bug này trong review code thật.

---

## Tóm tắt

`interface != nil` không tương đương "con trỏ khác nil" — nó tương đương "hoặc type khác nil, hoặc value khác nil". Bug này không phải do bạn code sai logic, mà do hiểu lầm cách Go biểu diễn interface ở runtime. Quy tắc sống còn: **đừng để một kiểu con trỏ cụ thể bị đóng gói ngầm vào interface tại điểm return** — luôn kiểm tra hoặc dùng biến kiểu interface ngay từ đầu.
</content>
</invoke>
