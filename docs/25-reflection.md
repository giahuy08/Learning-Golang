# 25. Reflection — `reflect` Package

> Trong C#, bạn dùng `System.Reflection` để inspect type lúc runtime — Attribute-based serialization, DI container, ORM đều dựa vào nó. Go có tương đương: package `reflect`. Bạn hiếm khi tự viết code dùng `reflect` trực tiếp trong business logic, nhưng **hiểu nó giúp bạn biết `encoding/json`, `database/sql` Scan, hay validator library hoạt động thế nào bên dưới** — và biết khi nào một bug lạ là do reflection-based library, không phải do bạn.

---

## 1. Hai khái niệm cốt lõi: `Type` và `Value`

```go
package main

import (
    "fmt"
    "reflect"
)

type User struct {
    Name string
    Age  int
}

func main() {
    u := User{Name: "Alice", Age: 30}

    t := reflect.TypeOf(u)  // Thông tin về TYPE: tên, kind, các field
    v := reflect.ValueOf(u) // Thông tin về VALUE: giá trị thực tế

    fmt.Println(t.Name())      // "User"
    fmt.Println(t.Kind())      // "struct"
    fmt.Println(v.NumField())  // 2

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        value := v.Field(i)
        fmt.Printf("%s (%s) = %v\n", field.Name, field.Type, value)
    }
    // Output:
    // Name (string) = Alice
    // Age (int) = 30
}
```

**`Kind()` vs `Type()`:** `Type()` là tên type cụ thể (`User`, `MyInt`), `Kind()` là category cơ bản (`struct`, `int`, `slice`, `map`...). Một `type MyInt int` có `Type()` là `MyInt` nhưng `Kind()` là `reflect.Int`.

---

## 2. Đọc struct tag — Cách `encoding/json` hoạt động bên dưới

Đây chính là cơ chế đứng sau `json:"name"` mà bạn dùng hàng ngày (xem thêm `12-json-struct-tags.md`):

```go
type Product struct {
    Name  string `json:"name" validate:"required"`
    Price float64 `json:"price" validate:"gt=0"`
}

func main() {
    t := reflect.TypeOf(Product{})

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        jsonTag := field.Tag.Get("json")
        validateTag := field.Tag.Get("validate")
        fmt.Printf("%s: json=%q validate=%q\n", field.Name, jsonTag, validateTag)
    }
    // Name: json="name" validate="required"
    // Price: json="price" validate="gt=0"
}
```

`encoding/json.Marshal` thực chất làm đúng việc này: dùng reflection để duyệt qua struct fields, đọc tag `json`, rồi quyết định tên key và cách encode. Đây cũng là lý do **struct tag sai cú pháp không gây compile error** — nó chỉ là string, và bug chỉ lộ ra lúc runtime khi thư viện đọc tag.

---

## 3. Set giá trị qua reflection — Cần `Elem()` và pointer

Đây là phần dễ gây confuse nhất: để **sửa** giá trị qua reflection, bạn phải truyền pointer, rồi gọi `.Elem()` để lấy giá trị mà pointer trỏ tới.

```go
// BAD: reflect.ValueOf(u) không addressable — panic khi gọi Set
u := User{Name: "Alice"}
v := reflect.ValueOf(u)
v.Field(0).SetString("Bob") // panic: reflect: reflect.Value.SetString using unaddressable value
```

```go
// GOOD: truyền pointer, dùng Elem() để lấy value có thể set
u := User{Name: "Alice"}
v := reflect.ValueOf(&u).Elem() // Elem() "bóc" pointer ra giá trị bên trong
v.Field(0).SetString("Bob")
fmt.Println(u.Name) // "Bob"
```

Đây chính xác là cơ chế `json.Unmarshal(data, &target)` yêu cầu bạn truyền `&target` — không có pointer, reflection không thể ghi ngược giá trị vào struct gốc.

---

## 4. Type Switch vs Reflection — Khi nào dùng cái nào

```go
// Type switch (idiomatic Go) — ưu tiên dùng khi CÓ THỂ liệt kê hết các type
func describe(v any) string {
    switch x := v.(type) {
    case int:
        return fmt.Sprintf("int: %d", x)
    case string:
        return fmt.Sprintf("string: %q", x)
    default:
        return "unknown"
    }
}
```

```go
// Reflection — cần khi xử lý type KHÔNG biết trước (generic library, serialization)
func genericDescribe(v any) string {
    rv := reflect.ValueOf(v)
    return fmt.Sprintf("kind=%s, type=%s, value=%v", rv.Kind(), rv.Type(), rv)
}
```

**Quy tắc thực tế (Rob Pike):** *"Clear is better than clever. Reflection is never clear."* — chỉ dùng `reflect` khi viết infrastructure/library code (serializer, ORM, validator, DI framework). Trong business logic thông thường, nếu bạn thấy mình cần `reflect`, thường có cách thiết kế lại bằng interface hoặc generics (xem `09-generics.md`) sạch hơn.

---

## 5. Generics đã thay thế nhiều use-case của reflection

Trước Go 1.18 (chưa có generics), một số thư viện dùng `reflect` để viết hàm "generic" kiểu `func Max(a, b interface{}) interface{}`. Từ Go 1.18+, phần lớn các case này nên dùng generics thay vì reflection — nhanh hơn (compile-time, không cần runtime type inspection) và type-safe hơn.

```go
// CŨ: dùng reflect để viết hàm generic trước Go 1.18 — chậm, không type-safe
func OldMax(a, b interface{}) interface{} {
    va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
    if va.Kind() == reflect.Int && va.Int() > vb.Int() {
        return a
    }
    return b
}
```

```go
// MỚI: generics — nhanh hơn, compile-time type check, không cần reflect
func Max[T cmp.Ordered](a, b T) T {
    if a > b {
        return a
    }
    return b
}
```

**Reflection vẫn cần thiết khi:** bạn không biết type tại compile-time — ví dụ serialization framework nhận `interface{}` từ JSON/DB và phải quyết định cấu trúc lúc runtime. Generics giải quyết "polymorphism với type đã biết ở compile-time", reflection giải quyết "type hoàn toàn động".

---

## 6. Performance — Reflection không miễn phí

Reflection chậm hơn direct code access đáng kể (thường 10-100x tuỳ operation) vì:
- Type information phải được tra cứu lúc runtime thay vì compile-time
- Không có type-checking từ compiler → nhiều bounds-check, type-assertion runtime
- Việc `Set` giá trị qua reflection cần thêm layer indirection

```go
// Benchmark thực tế — reflection field access chậm hơn nhiều so với direct access
func BenchmarkDirectAccess(b *testing.B) {
    u := User{Name: "Alice"}
    for i := 0; i < b.N; i++ {
        _ = u.Name
    }
}

func BenchmarkReflectAccess(b *testing.B) {
    u := User{Name: "Alice"}
    v := reflect.ValueOf(u)
    for i := 0; i < b.N; i++ {
        _ = v.Field(0).String()
    }
}
// Kết quả điển hình: reflect access chậm hơn ~20-50x direct access
```

Đây là lý do các thư viện JSON hiệu năng cao (`json-iterator/go`, `easyjson`) tránh reflection bằng cách **code generation** — generate code Marshal/Unmarshal cụ thể cho từng struct tại build-time thay vì dùng reflection lúc runtime (tương tự cách `stringer` generate code, xem `18-tooling-debugging.md` §14).

---

## Bảng tổng kết

| Use case | Nên dùng |
|---|---|
| Biết trước tất cả type cần xử lý | Type switch |
| Cần polymorphism với type parameter cụ thể | Generics (`[T any]`) |
| Viết serializer/ORM/validator cho type bất kỳ | Reflection (`reflect` package) |
| Cần performance cao cho serialization | Code generation (`go generate`) thay vì reflection runtime |

---

## Bài tập

Viết một hàm `ToMap(v any) map[string]any` dùng reflection để convert bất kỳ struct nào thành `map[string]interface{}`, dùng tên field làm key (hoặc đọc tag `json` nếu có). Xử lý case struct chứa nested struct (đệ quy).
