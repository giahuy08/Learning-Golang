# 📚 Go Basics — Nền tảng Go (dành cho C# developer)

---

## 1. Go khác C# như thế nào?

Trước khi đi vào chi tiết, cần hiểu triết lý thiết kế của Go:

- **C#** sinh ra để lập trình viên làm việc nhanh — có IDE mạnh, nhiều abstraction, OOP đầy đủ.
- **Go** sinh ra để build **hệ thống lớn** với **team lớn** — ưu tiên đơn giản, đọc code dễ hơn viết code.

Hệ quả: Go **cố tình bỏ** nhiều thứ C# có (generics hạn chế, không có exception, không có class, không có inheritance). Không phải vì Go kém hơn — mà vì Google đã thấy những thứ đó gây ra vấn đề ở scale lớn.

---

## 2. Chương trình Go đầu tiên

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Go!")
}
```

So sánh với C#:
```csharp
// C#
using System;

class Program {
    static void Main() {
        Console.WriteLine("Hello, C#!");
    }
}
```

**`package main`** — Mọi file Go phải bắt đầu bằng khai báo package. Package `main` đặc biệt: nó là entry point của chương trình. Nếu thiếu dòng này, Go compiler sẽ báo lỗi ngay — đây là lý do lỗi `expected package, found import` xảy ra khi bạn quên dòng đầu tiên này.

**`import "fmt"`** — Tương đương `using System` trong C#. `fmt` là package chuẩn để format và in output.

**`func main()`** — Entry point, tương đương `static void Main()`. Không có class bọc ngoài vì Go không có class.

---

## 2.1. Cách chạy chương trình Go

Trước khi đi tiếp, cần biết cách thực sự chạy code trên máy — không chỉ đọc ví dụ.

### Bước 0 — Module (làm 1 lần cho mỗi project)

Go cần biết project của bạn thuộc "module" nào (giống khái niệm gần với `.csproj`/`.sln`, nhưng đơn giản hơn nhiều — chỉ là 1 file text). Trong thư mục project:

```bash
go mod init ten-module-cua-ban   # tạo file go.mod, chỉ chạy 1 lần khi bắt đầu project
```

Repo bạn đang học (`D:\Code\Golang\Go\Go`) đã có `go.mod` sẵn (`module learn-go`) — không cần chạy lại `go mod init`.

### `go run` — chạy thử nhanh, không tạo file binary

```bash
go run main.go          # chạy đúng 1 file
go run .                # chạy package (thư mục) hiện tại — dùng khi có nhiều file .go cùng package
go run ./cmd/shop        # chạy package nằm trong thư mục con — dùng khi project có nhiều "cmd/xxx"
```

`go run` **biên dịch ra 1 file tạm, chạy xong rồi xoá** — dùng khi đang code/thử nghiệm, tương đương bấm F5 debug-run trong Visual Studio nhưng không có debugger đính kèm.

### `go build` — tạo ra file thực thi (.exe trên Windows), không tự chạy

```bash
go build ./cmd/shop      # tạo ra file shop.exe trong thư mục hiện tại
./shop.exe                # tự chạy file đó (PowerShell: .\shop.exe)
```

Dùng khi muốn có 1 file binary để phân phối/deploy — đây chính là điểm mạnh của Go: 1 file `.exe` duy nhất, không cần cài .NET runtime hay bất kỳ dependency nào trên máy chạy nó (khác hẳn C# cần .NET runtime, trừ khi publish self-contained).

### `go install` — build rồi copy binary vào `$GOPATH/bin` (hoặc `$GOBIN`) để gọi như 1 lệnh CLI toàn cục

```bash
go install ./cmd/shop    # sau lệnh này, gõ "shop" (nếu $GOBIN nằm trong PATH) chạy được ở bất kỳ đâu
```

Đây là cách các tool bạn cài ở `20-setup-checklist.md` (`golangci-lint`, `dlv`, `gopls`...) hoạt động — chúng được `go install` vào máy bạn.

### Bảng tổng kết nhanh

| Lệnh | Tạo file binary? | Khi nào dùng |
|---|---|---|
| `go run .` | Không (file tạm, tự xoá) | Đang code, muốn chạy thử ngay |
| `go build ./cmd/x` | Có, ở thư mục hiện tại | Muốn có file để chạy nhiều lần hoặc phân phối |
| `go install ./cmd/x` | Có, vào `$GOBIN` | Muốn dùng như 1 lệnh CLI toàn cục trên máy |

### Truyền argument / flag

```bash
go run . --port=8080 arg1 arg2
```

Mọi thứ sau tên package được truyền nguyên vẹn vào `os.Args` của chương trình Go — `go run`/`go build` không "nuốt" các flag đó, chúng thuộc về chương trình của bạn, không phải của Go toolchain. Đọc chúng bằng package `flag` (xem `17-go-vs-csharp-gotchas.md` và roadmap Phase 5 — CLI Tools).

### Với repo học Go hiện tại của bạn

```bash
make run              # go run main.go — chạy file scratch ở root
go run ./cmd/shop      # chạy mini project e-commerce (docs/22-mini-ecommerce-project.md)
make ci                # fmt + vet + lint + test-race — chạy trước khi coi 1 bài là "xong"
```

---

## 3. Variables & Types

### Cách khai báo biến

Trong C#, bạn viết:
```csharp
int x = 10;
string name = "Alice";
var y = 20; // compiler tự infer type
```

Trong Go, có 2 cách:

```go
// Cách 1: var — tường minh, dùng ở package level hoặc khi cần zero value
var x int = 10
var name string = "Alice"

// Cách 2: := (short declaration) — compiler tự infer type, CHỈ dùng trong function
x := 10
name := "Alice"
```

**Tại sao Go có `:=`?** Vì Go muốn phân biệt rõ ràng giữa khai báo biến mới (`x := 10`) và gán giá trị cho biến đã có (`x = 10`). Trong C#, `=` làm cả hai việc nên đôi khi gây nhầm lẫn. Go tách ra để code rõ nghĩa hơn.

**Quan trọng:** Go **không cho phép khai báo biến mà không dùng**. Đoạn code sau sẽ không compile:

```go
x := 10
// không dùng x ở đâu cả
// → compile error: x declared and not used
```

Lý do: Go muốn loại bỏ dead code ngay từ compile time, tránh tích lũy rác trong codebase lớn.

### Các kiểu dữ liệu cơ bản

```go
// Số nguyên — Go có nhiều size, khác C# chỉ có int/long
var i int = 42          // C# int tương đương, size phụ thuộc platform (64-bit trên máy 64-bit)
var i8 int8 = 127       // C# sbyte
var i16 int16 = 32767   // C# short
var i32 int32 = 2147483647 // C# int (fixed 32-bit)
var i64 int64 = 9999999999 // C# long

// Số thực
var f32 float32 = 3.14  // C# float
var f64 float64 = 3.14159265358979 // C# double — dùng cái này mặc định

// String
s := "Hello"            // C# string, cũng immutable và UTF-8
s2 := `Multi
line`                   // C# @"..." verbatim string

// Boolean
b := true               // C# bool, giống hệt

// Không có char trong Go — dùng byte (ASCII) hoặc rune (Unicode)
ch := byte('A')         // C# byte — lưu giá trị ASCII
r := rune('Ắ')          // C# char — lưu Unicode code point (int32)
```

**Tại sao Go không có `char`?** Vì `char` trong C# thực ra là UTF-16, gây ra nhiều vấn đề với Unicode. Go dùng `rune` (int32) để đại diện Unicode code point, rõ ràng hơn về bản chất.

### Zero Values — Go không có null (theo nghĩa C#)

Trong C#, value type có giá trị mặc định, reference type thì null:
```csharp
int x;     // x = 0
string s;  // s = null — có thể gây NullReferenceException!
```

Trong Go, **mọi** biến đều có zero value, không bao giờ "uninitialized":
```go
var i int       // 0
var f float64   // 0.0
var s string    // "" (empty string, không phải null)
var b bool      // false
var p *int      // nil (pointer có thể nil)
```

Đây là lý do Go ít gặp null pointer exception hơn C# — bạn vẫn có thể có nil pointer, nhưng chỉ với pointer/interface/slice/map/channel, không phải với string hay int thông thường.

---

## 4. Functions

### Khai báo function

```csharp
// C#
int Add(int a, int b) {
    return a + b;
}
```

```go
// Go — từ khóa func đứng trước, type đứng sau tên
func add(a int, b int) int {
    return a + b
}

// Nếu các tham số cùng type, viết gọn
func add(a, b int) int {
    return a + b
}
```

### Multiple Return Values — thứ C# không có

Đây là một trong những điểm khác biệt lớn nhất. Trong C#, để trả về nhiều giá trị bạn phải dùng `Tuple`, `out` parameter, hoặc tạo class riêng:

```csharp
// C# — cồng kềnh
(int result, bool success) Divide(int a, int b) {
    if (b == 0) return (0, false);
    return (a / b, true);
}
```

Trong Go, multiple return là tính năng built-in, được dùng **ở khắp nơi**:

```go
func divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("cannot divide by zero")
    }
    return a / b, nil
}

// Gọi hàm — phải nhận đủ return values
result, err := divide(10, 2)
if err != nil {
    log.Fatal(err) // xử lý lỗi ngay tại chỗ
}
fmt.Println(result) // 5
```

**Tại sao Go không dùng exception?** Đây là quyết định thiết kế quan trọng nhất của Go. Exception trong C# (và Java) có vấn đề: bạn không biết hàm nào có thể throw exception trừ khi đọc documentation hoặc source code. Go buộc lỗi phải là một phần của function signature — khi thấy `(result, error)` bạn biết ngay hàm này có thể thất bại. Điều này giúp code ở scale lớn dễ maintain hơn nhiều.

### Cách bỏ qua return value

Trong Go, bạn **phải** nhận tất cả return values. Nếu không cần, dùng `_`:

```go
result, _ := divide(10, 2) // bỏ qua error — chỉ làm khi chắc chắn không có lỗi
```

---

## 5. Control Flow

### if-else

```go
// Không cần dấu ngoặc () — khác C#
if x > 10 {
    fmt.Println("lớn hơn 10")
} else if x == 10 {
    fmt.Println("bằng 10")
} else {
    fmt.Println("nhỏ hơn 10")
}
```

**Init statement trong if** — Go cho phép chạy một lệnh ngay trước điều kiện, và biến đó chỉ tồn tại trong scope của if:

```go
// Cực kỳ phổ biến trong Go — gọi hàm rồi kiểm tra error ngay
if err := doSomething(); err != nil {
    log.Fatal(err)
}
// err không tồn tại ở đây — tránh ô nhiễm scope

// Tương đương C# (verbose hơn):
// var err = DoSomething();
// if (err != null) { ... }
```

### for — Go chỉ có một vòng lặp

Go không có `while`, `do-while`, `foreach` riêng biệt. Tất cả đều là `for`:

```go
// Giống C# for
for i := 0; i < 5; i++ {
    fmt.Println(i)
}

// Giống C# while
count := 0
for count < 5 {
    count++
}

// Giống C# foreach — dùng range
nums := []int{1, 2, 3, 4, 5}
for i, v := range nums {
    fmt.Printf("index=%d, value=%d\n", i, v)
}

// Chỉ lấy value, bỏ index
for _, v := range nums {
    fmt.Println(v)
}

// Vòng lặp vô hạn (giống C# while(true))
for {
    // break để thoát
}
```

**Tại sao chỉ có `for`?** Go theo triết lý "một cách duy nhất để làm một việc". Có `while`, `do-while`, `for`, `foreach` trong C# cơ bản làm cùng một việc — dễ gây tranh luận trong team nên dùng cái nào. Go loại bỏ vấn đề đó.

### switch

```go
day := "Monday"
switch day {
case "Monday", "Tuesday", "Wednesday", "Thursday", "Friday":
    fmt.Println("Ngày làm việc")
case "Saturday", "Sunday":
    fmt.Println("Cuối tuần")
default:
    fmt.Println("Không hợp lệ")
}
```

**Khác C#:** Go switch không cần `break` — mặc định không fall-through. Nếu muốn fall-through (hiếm khi dùng), dùng từ khóa `fallthrough`.

---

## 6. Arrays và Slices

### Array — ít dùng trong Go

```go
// Array có size cố định — là VALUE TYPE giống C# struct
arr := [3]int{1, 2, 3}

// Khi gán cho biến khác, nó COPY toàn bộ — giống C# value type
arr2 := arr
arr2[0] = 99
fmt.Println(arr[0])  // 1 — arr không thay đổi
```

### Slice — dùng hàng ngày

Slice trong Go tương tự `List<T>` trong C# nhưng mạnh hơn. Slice là **reference type**.

Bên trong, một slice gồm 3 thành phần:
```
┌─────────────────────────────────┐
│ Slice header                    │
│  pointer → [underlying array]   │
│  len = 3  (số phần tử hiện tại) │
│  cap = 5  (dung lượng tối đa)   │
└─────────────────────────────────┘
```

```go
// Tạo slice
nums := []int{1, 2, 3}           // literal
s := make([]int, 3, 5)           // make(type, len, cap) — len=3, cap=5
var s2 []int                     // nil slice — len=0, cap=0

// append — nếu vượt cap, Go tự cấp phát array mới lớn hơn
nums = append(nums, 4)           // phải gán lại vì append trả về slice mới
nums = append(nums, 5, 6, 7)     // thêm nhiều phần tử

// Slicing — lấy một phần
s := []int{0, 1, 2, 3, 4, 5}
fmt.Println(s[1:3])   // [1 2] — từ index 1 đến 2 (không bao gồm 3)
fmt.Println(s[:3])    // [0 1 2] — từ đầu đến index 2
fmt.Println(s[3:])    // [3 4 5] — từ index 3 đến hết
```

**Cạm bẫy quan trọng:** Vì slice chia sẻ underlying array, thay đổi qua slice này có thể ảnh hưởng slice kia:

```go
a := []int{1, 2, 3, 4, 5}
b := a[1:3]   // b = [2 3], nhưng chia sẻ array với a

b[0] = 99
fmt.Println(a) // [1 99 3 4 5] — a cũng bị thay đổi!

// Fix: dùng copy() nếu cần slice độc lập
b = make([]int, 2)
copy(b, a[1:3])
b[0] = 99
fmt.Println(a) // [1 2 3 4 5] — a không bị ảnh hưởng
```

### Map — tương đương Dictionary<K,V> trong C#

```go
// Tạo map
m := map[string]int{
    "alice": 90,
    "bob":   85,
}

// HOẶC dùng make
m := make(map[string]int)

// Thêm/cập nhật — giống C# Dictionary
m["charlie"] = 95
m["alice"] = 92  // cập nhật

// Đọc
score := m["alice"]  // 92

// Vấn đề: nếu key không tồn tại, trả về zero value (0 cho int)
// Không biết được đây là 0 thật hay key không tồn tại
missing := m["dave"]  // 0 — không rõ có key "dave" không

// GOOD: dùng ok pattern — giống C# TryGetValue
score, ok := m["dave"]
if !ok {
    fmt.Println("dave không tồn tại")
}

// Xóa — không có method, dùng built-in delete
delete(m, "bob")

// Iterate — thứ tự KHÔNG ĐẢM BẢO (khác C# Dictionary cũng vậy)
for key, value := range m {
    fmt.Printf("%s: %d\n", key, value)
}
```

---

## 7. Packages

Go tổ chức code bằng **package**, tương tự namespace trong C# nhưng cũng kiêm luôn vai trò visibility.

**Rule duy nhất về visibility:** Tên bắt đầu bằng chữ HOA = exported (public). Chữ thường = unexported (internal to package).

```go
// Trong package "math"
func Add(a, b int) int { ... }    // exported — package khác dùng được
func helper(x int) int { ... }    // unexported — chỉ trong package "math"

type Rectangle struct {
    Width  float64  // exported field
    height float64  // unexported field
}
```

```go
// Trong package "main"
import "math"

math.Add(1, 2)    // OK
math.helper(1)    // Error: cannot refer to unexported name
```

**Tại sao không dùng `public`/`private` như C#?** Go muốn visibility được nhìn thấy ngay từ tên — không cần đọc thêm keyword. Khi thấy tên viết hoa, bạn biết ngay nó là public API.

---

## 8. Pointers

Go có pointers như C/C++, nhưng đơn giản hơn (không có pointer arithmetic).

Trong C#:
- Value types (int, struct) được pass by value mặc định
- Reference types (class) được pass by reference mặc định

Trong Go: **mọi thứ đều pass by value**. Kể cả slice, map (nhưng chúng chứa pointer bên trong nên vẫn share data). Muốn thực sự pass by reference, dùng pointer.

```go
x := 42
p := &x        // & lấy địa chỉ của x, p có type *int
fmt.Println(p)  // 0xc0000b4008 — địa chỉ bộ nhớ
fmt.Println(*p) // 42 — * dereference: lấy giá trị tại địa chỉ đó

*p = 100        // thay đổi giá trị tại địa chỉ p trỏ tới
fmt.Println(x)  // 100 — x đã thay đổi
```

Khi nào cần pointer:

```go
// BAD: muốn modify struct trong function nhưng pass by value
func birthday(u User) {
    u.Age++  // chỉ thay đổi bản copy, original không đổi
}

user := User{Name: "Alice", Age: 30}
birthday(user)
fmt.Println(user.Age)  // vẫn là 30

// GOOD: pass pointer
func birthday(u *User) {
    u.Age++  // thay đổi original qua pointer
}

birthday(&user)
fmt.Println(user.Age)  // 31
```

---

## 9. Common Errors

**`declared and not used`**
Go không cho phép khai báo biến mà không dùng. Xóa biến đó hoặc dùng `_` nếu cần bỏ qua return value.

**`imported and not used`**
Tương tự — không cho phép import package không dùng. Xóa import thừa.

**`expected package, found import`**
File thiếu dòng `package main` ở đầu tiên. Đây là lỗi hay gặp nhất khi mới học.

**`cannot use X (type A) as type B`**
Go không tự động convert giữa các numeric type, kể cả `int32` và `int64`. Phải convert tường minh: `int64(myInt32)`.

**`index out of range`**
Truy cập slice ngoài bounds. Kiểm tra `len(s)` trước khi truy cập.

---

## 10. Bài tập

### Bài 1: FizzBuzz
Viết hàm `fizzBuzz(n int) []string` trả về slice strings từ 1 đến n:
- "FizzBuzz" nếu chia hết cho cả 3 và 5
- "Fizz" nếu chia hết cho 3
- "Buzz" nếu chia hết cho 5
- Số dạng string nếu không chia hết

```go
package main

import (
    "fmt"
    "strconv"
)

func fizzBuzz(n int) []string {
    result := make([]string, 0, n)
    for i := 1; i <= n; i++ {
        switch {
        case i%15 == 0:
            result = append(result, "FizzBuzz")
        case i%3 == 0:
            result = append(result, "Fizz")
        case i%5 == 0:
            result = append(result, "Buzz")
        default:
            result = append(result, strconv.Itoa(i))
        }
    }
    return result
}

func main() {
    fmt.Println(fizzBuzz(15))
}
```

### Bài 2: Word Count
```go
func wordCount(s string) map[string]int {
    counts := make(map[string]int)
    for _, word := range strings.Fields(s) {
        counts[strings.ToLower(word)]++
    }
    return counts
}
```

---

## Next
➡️ [`02-structs-interfaces.md`](02-structs-interfaces.md) — Structs, Methods, Interfaces
