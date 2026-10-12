# 📚 Generics trong Go — Dành cho C# Developer

---

## 1. Tại sao Go thêm Generics muộn vậy? (Go 1.18, năm 2022)

Đây là một trong những câu hỏi hay nhất khi học Go. C# có generics từ phiên bản 2.0 (năm 2005). Java có generics từ Java 5 (cũng năm 2004). Vậy tại sao Go — ra đời năm 2009 — phải chờ đến tận 2022 mới có?

Câu trả lời không phải là "team Go lười biếng" hay "họ không nghĩ đến". Thực ra, Rob Pike, Ken Thompson và Robert Griesemer — ba người sáng lập Go — đã tranh luận về generics ngay từ những ngày đầu. Quyết định **không** thêm generics ban đầu là một quyết định **có chủ đích**, xuất phát từ triết lý thiết kế của Go: *đơn giản hơn, đọc được hơn, biên dịch nhanh hơn*.

Vấn đề với generics là độ phức tạp bùng phát theo cấp số nhân. Nhìn vào C#: bạn có `List<T>`, `Dictionary<TKey, TValue>`, `IEnumerable<T>`, `IQueryable<T>`, covariance (`out T`), contravariance (`in T`), generic constraints (`where T : class, IComparable<T>, new()`), và toàn bộ cơ chế reflection phải hiểu generics. Với Go, mỗi tính năng mới phải "trả" cho sự phức tạp của compiler, của tooling, của việc đọc code — và team Go cực kỳ thận trọng với cái giá đó.

Thay vào đó, trong suốt 13 năm, Go giải quyết vấn đề "code tổng quát" theo hai cách:

Cách thứ nhất là dùng `interface{}` (Go 1) hoặc `any` (Go 1.18 alias). Bạn nhận một giá trị bất kỳ, rồi type-assert khi cần. Cách này hoạt động nhưng mất type safety — compiler không bắt lỗi sai kiểu dữ liệu cho bạn, runtime mới bắt.

Cách thứ hai là code generation — dùng `go generate` với các tool như `stringer` hoặc viết template để sinh ra nhiều version của cùng một hàm cho các kiểu khác nhau. Cách này type-safe nhưng cồng kềnh và khó maintain.

Đến 2019, Russ Cox và Ian Lance Taylor bắt đầu một quá trình thiết kế kéo dài nhiều năm, trải qua nhiều proposal bị từ chối, để tìm ra cú pháp generics **phù hợp với triết lý Go** — đủ mạnh để giải quyết 95% use cases thực tế mà không mang vào sự phức tạp của C++ templates hay Haskell type classes. Kết quả là hệ thống generics trong Go 1.18: mạnh vừa đủ, không phức tạp hơn cần thiết.

---

## 2. So sánh với C# Generics

Về mặt khái niệm, generics trong Go và C# giải quyết cùng một vấn đề: viết code một lần, dùng được cho nhiều kiểu dữ liệu khác nhau, với type safety được kiểm tra lúc compile.

```csharp
// C# — Generic class
public class Stack<T>
{
    private List<T> _items = new();
    public void Push(T item) => _items.Add(item);
    public T Pop() { var last = _items[^1]; _items.RemoveAt(_items.Count - 1); return last; }
    public int Count => _items.Count;
}

// C# — Generic method
public static IEnumerable<TResult> Map<TSource, TResult>(
    IEnumerable<TSource> source,
    Func<TSource, TResult> selector)
{
    return source.Select(selector);
}

// C# — Constraint: T phải implement IComparable<T>
public static T Max<T>(T a, T b) where T : IComparable<T>
{
    return a.CompareTo(b) >= 0 ? a : b;
}
```

```go
// Go — tương đương
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(item T) {
    s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() T {
    last := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return last
}

// Go — Generic function
func Map[T, U any](s []T, f func(T) U) []U {
    result := make([]U, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}

// Go — Constraint: T phải "có thể so sánh thứ tự"
func Max[T interface{ int | float64 | string }](a, b T) T {
    if a >= b {
        return a
    }
    return b
}
```

Điểm khác biệt quan trọng nhất: trong C#, constraint dùng từ khóa `where` và tham chiếu đến interface hoặc class. Trong Go, constraint **là** interface — nhưng interface trong Go 1.18 được mở rộng để có thể chứa danh sách các kiểu cụ thể, không chỉ là tập hợp method signatures. Đây là một ý tưởng hoàn toàn mới.

---

## 3. Cú pháp Type Parameters

Type parameter trong Go được đặt trong dấu ngoặc vuông `[...]` ngay sau tên hàm hoặc tên kiểu. Mỗi type parameter cần có một **constraint** — constraint nói với compiler "T phải là kiểu gì đó có khả năng X".

```go
package main

import "fmt"

// Đọc cú pháp: "Map là generic function với type parameter T và U,
// cả hai đều thỏa constraint 'any' (nghĩa là bất kỳ kiểu nào).
// Nhận vào một slice []T và một function f từ T sang U.
// Trả về một slice []U."
func Map[T, U any](s []T, f func(T) U) []U {
    result := make([]U, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}

func Filter[T any](s []T, pred func(T) bool) []T {
    var result []T
    for _, v := range s {
        if pred(v) {
            result = append(result, v)
        }
    }
    return result
}

func Reduce[T, Acc any](s []T, initial Acc, f func(Acc, T) Acc) Acc {
    acc := initial
    for _, v := range s {
        acc = f(acc, v)
    }
    return acc
}

func main() {
    numbers := []int{1, 2, 3, 4, 5}

    // Map: nhân đôi từng phần tử
    doubled := Map(numbers, func(n int) int { return n * 2 })
    fmt.Println(doubled) // [2 4 6 8 10]

    // Map: chuyển int sang string
    strs := Map(numbers, func(n int) string { return fmt.Sprintf("item_%d", n) })
    fmt.Println(strs) // [item_1 item_2 item_3 item_4 item_5]

    // Filter: chỉ lấy số chẵn
    evens := Filter(numbers, func(n int) bool { return n%2 == 0 })
    fmt.Println(evens) // [2 4]

    // Reduce: tính tổng
    sum := Reduce(numbers, 0, func(acc, n int) int { return acc + n })
    fmt.Println(sum) // 15
}
```

Nếu bạn quen với LINQ trong C#, đây là các tương đương:

```csharp
// C# LINQ tương đương
var doubled = numbers.Select(n => n * 2).ToList();
var strs    = numbers.Select(n => $"item_{n}").ToList();
var evens   = numbers.Where(n => n % 2 == 0).ToList();
var sum     = numbers.Aggregate(0, (acc, n) => acc + n);
```

Sự khác biệt quan trọng: LINQ trong C# dùng extension methods trên `IEnumerable<T>` — đây là "phép màu" của C# không có trong Go. Go không có extension methods, nên Map/Filter/Reduce phải là **standalone functions**, không phải methods trên slice. Bạn viết `Map(numbers, ...)` thay vì `numbers.Map(...)`. Đây là một trong những hạn chế của Go generics mà nhiều người tiếc — nhưng đây cũng là quyết định có chủ đích để tránh phức tạp.

---

## 4. Type Constraints

Constraint là cách bạn nói với compiler "type parameter T này phải có những khả năng gì". Không có constraint, compiler không thể biết bạn được phép làm gì với giá trị của T.

### `any` — constraint thoải mái nhất

`any` là alias của `interface{}`, nghĩa là "bất kỳ kiểu nào". Với constraint này, bạn chỉ có thể làm những thứ áp dụng cho **mọi** kiểu: gán biến, truyền vào hàm, đặt vào slice. Bạn **không** thể cộng hai giá trị T, so sánh chúng với `<`, hay gọi method nếu không biết T có method đó.

```go
// BAD: Cố gắng cộng hai giá trị T với constraint 'any'
// Compiler sẽ báo lỗi: "invalid operation: a + b (operator + not defined on T)"
func Add[T any](a, b T) T {
    return a + b // ❌ Lỗi compile!
}
```

### `comparable` — dùng được với == và !=

`comparable` là built-in constraint, nghĩa là T hỗ trợ toán tử `==` và `!=`. Trong Go, không phải mọi kiểu đều comparable — slice, map, function **không** comparable. Nhưng int, string, struct (có các field đều comparable) thì có.

```go
// GOOD: Contains dùng == để tìm kiếm
func Contains[T comparable](s []T, target T) bool {
    for _, v := range s {
        if v == target {
            return true
        }
    }
    return false
}

// comparable là bắt buộc khi dùng T làm map key
func GroupBy[T comparable, V any](s []V, key func(V) T) map[T][]V {
    result := make(map[T][]V)
    for _, v := range s {
        k := key(v)
        result[k] = append(result[k], v)
    }
    return result
}
```

So sánh với C#: `comparable` trong Go gần giống với `where T : IEquatable<T>` trong C#, nhưng Go không dùng interface — đây là constraint đặc biệt được build vào ngôn ngữ.

### Custom constraints với interface

Bạn có thể định nghĩa constraint của riêng mình bằng cách dùng interface. Interface dùng làm constraint có thể chứa **danh sách kiểu** (type set) — đây là tính năng mới hoàn toàn trong Go 1.18.

```go
package main

import "fmt"

// Định nghĩa constraint: Integer là bất kỳ kiểu integer nào
type Integer interface {
    int | int8 | int16 | int32 | int64 |
        uint | uint8 | uint16 | uint32 | uint64
}

// Định nghĩa constraint: Number bao gồm cả integer và float
type Number interface {
    Integer | float32 | float64
}

// Dùng ~ để bao gồm cả các kiểu có underlying type là int
// Ví dụ: type MyInt int — kiểu này sẽ thỏa constraint ~int
type Ordered interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
        ~float32 | ~float64 | ~string
}

func Sum[T Number](s []T) T {
    var total T
    for _, v := range s {
        total += v
    }
    return total
}

func Min[T Ordered](a, b T) T {
    if a < b {
        return a
    }
    return b
}

func main() {
    ints := []int{1, 2, 3, 4, 5}
    fmt.Println(Sum(ints)) // 15

    floats := []float64{1.1, 2.2, 3.3}
    fmt.Println(Sum(floats)) // 6.6000...

    fmt.Println(Min(3, 7))     // 3
    fmt.Println(Min("a", "b")) // a
}
```

Ký hiệu `~` rất quan trọng và không có tương đương trực tiếp trong C#. `~int` nghĩa là "kiểu nào có **underlying type** là int" — điều này cho phép bạn viết `type UserID int` và vẫn dùng được `Sum[UserID]`. Không có `~`, bạn phải explicit list `UserID` trong constraint.

---

## 5. Constraint từ package `golang.org/x/exp/constraints`

Go 1.21 đã thêm package `slices` và `maps` vào standard library với các generic function. Trước đó, community dùng `golang.org/x/exp/constraints`. Quan trọng nhất là `constraints.Ordered`:

```go
// Định nghĩa Ordered trong standard library (Go 1.21+)
// Tương đương với:
type Ordered interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
        ~float32 | ~float64 |
        ~string
}
```

Từ Go 1.21, bạn có thể dùng thẳng `slices.Sort`, `slices.Contains` từ standard library — những hàm này được implement bằng generics:

```go
package main

import (
    "fmt"
    "slices" // Go 1.21+
)

func main() {
    s := []int{3, 1, 4, 1, 5, 9, 2, 6}
    slices.Sort(s)
    fmt.Println(s) // [1 1 2 3 4 5 6 9]

    fmt.Println(slices.Contains(s, 5)) // true
    fmt.Println(slices.Index(s, 9))    // 6
}
```

> **Cập nhật Go 1.23+:** `maps.Keys(m)` và `maps.Values(m)` **không trả về slice** mà trả về `iter.Seq[K]`/`iter.Seq[V]` — một iterator lazy, dùng trực tiếp với `for range` hoặc convert sang slice bằng `slices.Collect(maps.Keys(m))`. Đây là một phần của tính năng **range-over-func iterators** — xem chi tiết ở [31-iterators-and-range-over-func.md](31-iterators-and-range-over-func.md), một trong những thay đổi ngôn ngữ lớn nhất kể từ khi generics ra mắt.

> **Cập nhật Go 1.24+:** ngoài generic type **definition** (`type Set[T comparable] struct{...}`) đã học ở mục 6 bên dưới, Go giờ hỗ trợ **generic type alias**: `type StringSet = Set[string]` với alias cũng có thể nhận type parameter: `type Pair[A, B any] = struct{ First A; Second B }`. Trước 1.24, alias không được phép có type parameter.

---

## 6. Generic Types — Generic Struct

Bạn không chỉ có thể viết generic functions mà còn có thể định nghĩa generic types. Đây tương đương với generic class trong C#.

```go
package main

import (
    "errors"
    "fmt"
)

// Stack[T any] — generic struct
// Tương đương C#: public class Stack<T> where T : class? — không, trong Go
// 'any' bao gồm cả value types (int, struct) và pointer types
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(item T) {
    s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, error) {
    if len(s.items) == 0 {
        var zero T // zero value của T — tương đương default(T) trong C#
        return zero, errors.New("stack is empty")
    }
    last := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return last, nil
}

func (s *Stack[T]) Peek() (T, bool) {
    if len(s.items) == 0 {
        var zero T
        return zero, false
    }
    return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int {
    return len(s.items)
}

func (s *Stack[T]) IsEmpty() bool {
    return len(s.items) == 0
}

func main() {
    // Stack của int
    intStack := &Stack[int]{}
    intStack.Push(1)
    intStack.Push(2)
    intStack.Push(3)

    for !intStack.IsEmpty() {
        val, _ := intStack.Pop()
        fmt.Println(val) // 3, 2, 1
    }

    // Stack của string — cùng code, khác kiểu
    strStack := &Stack[string]{}
    strStack.Push("hello")
    strStack.Push("world")

    top, ok := strStack.Peek()
    fmt.Printf("top=%s, ok=%v\n", top, ok) // top=world, ok=true
}
```

Lưu ý quan trọng: khi khai báo method trên generic type, bạn phải viết `func (s *Stack[T])` chứ **không** phải `func (s *Stack[T any])`. Type parameter `T` đã được khai báo ở `type Stack[T any]`, các methods chỉ dùng lại nó.

---

## 7. Type Inference — Go tự suy ra kiểu

Một trong những tính năng tiện dụng nhất: Go thường có thể tự suy ra type arguments từ các arguments thông thường, nên bạn không cần viết tường minh.

```go
package main

import "fmt"

func Map[T, U any](s []T, f func(T) U) []U {
    result := make([]U, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}

func Contains[T comparable](s []T, target T) bool {
    for _, v := range s {
        if v == target {
            return true
        }
    }
    return false
}

func main() {
    numbers := []int{1, 2, 3}

    // GOOD: Go tự suy ra T=int, U=string từ arguments
    // Không cần viết Map[int, string](...)
    result := Map(numbers, func(n int) string {
        return fmt.Sprintf("%d", n)
    })
    fmt.Println(result)

    // GOOD: Go tự suy ra T=int từ kiểu của slice và target
    fmt.Println(Contains(numbers, 2)) // true

    // Đôi khi phải viết tường minh khi Go không thể suy ra
    // Ví dụ khi return type khác với argument types hoàn toàn
    _ = make([]string, 0) // Go biết kiểu từ literal
}
```

Type inference trong Go không mạnh bằng C# — có một số trường hợp Go không tự suy ra được và bạn phải viết tường minh `Func[ConcreteType](...)`. Khi gặp lỗi compile về type inference, cứ thêm type arguments vào là xong.

---

## 8. Real-World Example: Generic `Result[T]` Type

Trong C#, pattern Result/Either rất phổ biến — thay vì throw exception, trả về một object chứa either giá trị thành công hoặc lỗi. Go vốn dùng multiple return values `(T, error)` cho mục đích này, nhưng đôi khi bạn cần truyền "kết quả có thể lỗi" như một giá trị (ví dụ: slice của results, hoặc results từ concurrent goroutines).

```go
package main

import (
    "errors"
    "fmt"
    "strconv"
)

// Result[T] — đóng gói một giá trị hoặc một lỗi
// Tương đương C#: Result<T> pattern, hoặc OneOf<T, Error>
type Result[T any] struct {
    value T
    err   error
    ok    bool
}

// Ok tạo một Result thành công
func Ok[T any](value T) Result[T] {
    return Result[T]{value: value, ok: true}
}

// Err tạo một Result lỗi
func Err[T any](err error) Result[T] {
    return Result[T]{err: err, ok: false}
}

// IsOk kiểm tra có phải là success không
func (r Result[T]) IsOk() bool {
    return r.ok
}

// Unwrap trả về giá trị, panic nếu là lỗi (giống C# .Value trên null)
func (r Result[T]) Unwrap() T {
    if !r.ok {
        panic(fmt.Sprintf("called Unwrap on an Err result: %v", r.err))
    }
    return r.value
}

// UnwrapOr trả về giá trị, hoặc defaultValue nếu là lỗi
func (r Result[T]) UnwrapOr(defaultValue T) T {
    if !r.ok {
        return defaultValue
    }
    return r.value
}

// Error trả về lỗi (nil nếu là Ok)
func (r Result[T]) Error() error {
    return r.err
}

// MapResult áp dụng function lên giá trị nếu là Ok
func MapResult[T, U any](r Result[T], f func(T) U) Result[U] {
    if !r.ok {
        return Err[U](r.err)
    }
    return Ok(f(r.value))
}

// parseNumber thử parse string thành int, trả về Result
func parseNumber(s string) Result[int] {
    n, err := strconv.Atoi(s)
    if err != nil {
        return Err[int](fmt.Errorf("cannot parse %q as number: %w", s, err))
    }
    return Ok(n)
}

func main() {
    // Parse thành công
    r1 := parseNumber("42")
    fmt.Println(r1.IsOk())        // true
    fmt.Println(r1.Unwrap())      // 42
    fmt.Println(r1.UnwrapOr(0))   // 42

    // Parse thất bại
    r2 := parseNumber("abc")
    fmt.Println(r2.IsOk())        // false
    fmt.Println(r2.Error())       // cannot parse "abc" as number: ...
    fmt.Println(r2.UnwrapOr(-1)) // -1

    // MapResult: nếu parse được, nhân đôi
    r3 := MapResult(parseNumber("21"), func(n int) int { return n * 2 })
    fmt.Println(r3.Unwrap()) // 42

    // Xử lý một batch — rất hữu ích khi collect results từ goroutines
    inputs := []string{"1", "bad", "3", "also_bad", "5"}
    var results []Result[int]
    for _, input := range inputs {
        results = append(results, parseNumber(input))
    }

    var successes []int
    var errMessages []string
    for _, r := range results {
        if r.IsOk() {
            successes = append(successes, r.Unwrap())
        } else {
            errMessages = append(errMessages, r.Error().Error())
        }
    }

    fmt.Println("Successes:", successes)    // [1 3 5]
    fmt.Println("Errors:", errMessages)     // [cannot parse "bad"... cannot parse "also_bad"...]

    // Dùng errors.Is để kiểm tra lỗi cụ thể
    var numErr *strconv.NumError
    if r2.Error() != nil && errors.As(r2.Error(), &numErr) {
        fmt.Println("NumError func:", numErr.Func) // Atoi
    }
}
```

---

## 9. Khi nào NÊN dùng Generics

Generics phù hợp nhất trong ba tình huống:

**Thứ nhất, data structures tổng quát.** Bất cứ khi nào bạn xây dựng một container (Stack, Queue, Heap, LinkedList, Tree, Set, Cache), generics là lựa chọn đúng đắn. Bạn muốn `Stack[Order]`, `Queue[Task]`, `Cache[string, User]` — tất cả type-safe, không cần type assertion.

**Thứ hai, utility functions trên collections.** Map, Filter, Reduce, GroupBy, Partition, Zip, Flatten — những hàm này hoạt động theo cùng logic bất kể kiểu dữ liệu là gì. Viết một lần với generics tốt hơn nhiều so với copy-paste cho từng kiểu.

**Thứ ba, helper functions hoạt động trên nhiều kiểu tương tự.** Ví dụ: `Ptr[T any](v T) *T` (trả về pointer đến giá trị), `Must[T any](v T, err error) T` (unwrap hoặc panic), `Coalesce[T comparable](values ...T) T` (trả về giá trị non-zero đầu tiên).

```go
package main

import "fmt"

// Ptr — rất hữu ích khi cần pointer đến literal value
// Thay vì: x := 42; ptr := &x — giờ chỉ cần: Ptr(42)
func Ptr[T any](v T) *T {
    return &v
}

// Must — unwrap (T, error), panic nếu có lỗi
// Tương đương C#: .Value trên Nullable hoặc guard clause
func Must[T any](v T, err error) T {
    if err != nil {
        panic(err)
    }
    return v
}

// Coalesce — trả về giá trị non-zero đầu tiên
// Tương đương C#: ?? operator nhưng cho bất kỳ comparable type
func Coalesce[T comparable](values ...T) T {
    var zero T
    for _, v := range values {
        if v != zero {
            return v
        }
    }
    return zero
}

func main() {
    p := Ptr(42)
    fmt.Println(*p) // 42

    // Coalesce với string: trả về string không rỗng đầu tiên
    name := Coalesce("", "", "Alice", "Bob")
    fmt.Println(name) // Alice

    // Coalesce với int: trả về int != 0 đầu tiên
    port := Coalesce(0, 0, 8080, 9090)
    fmt.Println(port) // 8080
}
```

---

## 10. Khi nào KHÔNG NÊN dùng Generics

Đây là phần nhiều người bỏ qua và sau đó viết code phức tạp không cần thiết.

**Đừng dùng generics khi interface thường là đủ.** Nếu bạn cần gọi các methods cụ thể trên giá trị, và những methods đó có thể được định nghĩa trong interface, hãy dùng interface thường. Generics giải quyết vấn đề *khác kiểu nhưng cùng logic cấu trúc*; interface giải quyết vấn đề *khác kiểu nhưng cùng behavior*.

```go
// BAD: Dùng generic khi interface là đủ và rõ hơn
// Nếu bạn cần "gọi Speak() trên T", đừng dùng generic
type Animal interface {
    Speak() string
}

// BAD version — generic không cần thiết
func MakeNoise[T interface{ Speak() string }](animals []T) {
    for _, a := range animals {
        fmt.Println(a.Speak())
    }
}

// GOOD version — dùng interface thường, đơn giản hơn, flexible hơn
// (bạn có thể truyền mixed slice của Dog và Cat)
func MakeNoise(animals []Animal) {
    for _, a := range animals {
        fmt.Println(a.Speak())
    }
}
```

**Đừng dùng generics khi chỉ có một kiểu dữ liệu.** Nếu hàm của bạn chỉ bao giờ được gọi với `int`, đừng làm nó generic chỉ để trông "đẹp". Premature generalization cũng nguy hiểm như premature optimization.

**Đừng dùng generics để avoid interface{}/any khi code thực sự cần dynamic typing.** Một số use cases thực sự cần tính linh động của `any` — JSON serialization, plugin systems, reflection-based code. Cố ép generics vào đây sẽ tạo ra code phức tạp hơn nhiều mà không giải quyết được vấn đề.

---

## 11. Hạn chế của Go Generics (điều C# làm được nhưng Go không làm)

### Không có generic methods — chỉ có generic functions

Đây là hạn chế lớn nhất và gây bất ngờ nhiều nhất cho C# developer.

```go
// BAD: Cố thêm type parameter vào method — KHÔNG HỢP LỆ trong Go
type MySlice[T any] struct {
    items []T
}

// ❌ Lỗi compile: "method cannot have type parameters"
func (s *MySlice[T]) MapTo[U any](f func(T) U) []U {
    // ...
}

// GOOD: Dùng standalone generic function thay thế
func MapSlice[T, U any](s *MySlice[T], f func(T) U) []U {
    result := make([]U, len(s.items))
    for i, v := range s.items {
        result[i] = f(v)
    }
    return result
}
```

Trong C#, bạn có thể viết `myList.Select(x => x * 2)` vì `Select` là generic method trên `IEnumerable<T>`. Trong Go, bạn phải viết `Map(mySlice, func(x int) int { return x * 2 })`. Đây là lý do tại sao LINQ-style chaining không tự nhiên trong Go.

### Không có specialization

Trong C++ templates hay trong C# (thông qua overloading), bạn có thể viết implementation khác nhau cho từng kiểu cụ thể. Go không hỗ trợ điều này — một generic function có một implementation cho tất cả types.

### Không có covariance/contravariance

C# có `IEnumerable<out T>` (covariant) và `Action<in T>` (contravariant). Go không có khái niệm này. `[]Dog` không thể được dùng như `[]Animal` kể cả khi `Dog` implements `Animal`.

---

## 12. Bảng: Vấn Đề Thường Gặp

| Lỗi / Vấn đề | Nguyên nhân | Giải pháp |
|---|---|---|
| `cannot use [] as type parameter` | Slice không comparable | Dùng `any` thay `comparable` cho slice |
| `operator + not defined on T` | Constraint `any` không cho phép arithmetic | Dùng constraint `Number` hoặc `Integer` |
| `method cannot have type parameters` | Go không cho generic methods | Chuyển thành standalone generic function |
| `cannot infer type argument` | Go không đủ thông tin để suy ra type | Thêm type argument tường minh: `Func[int](...)` |
| `T does not implement interface X` | T thỏa constraint nhưng không có method cụ thể | Thêm method vào constraint interface |
| `undefined: constraints.Ordered` | Package `constraints` không còn trong stdlib | Dùng `cmp.Ordered` (Go 1.21+) hoặc tự định nghĩa |
| Generic code chậm hơn expected | GC shape stenciling — Go dùng shared implementation | Profile trước khi lo lắng; thường không thành vấn đề |
| Muốn làm `[]Dog` thành `[]Animal` | Go không có covariance | Copy sang `[]Animal` bằng loop, hoặc thiết kế lại |

---

## 13. Best Practices

Khi viết generic code trong Go, nguyên tắc đầu tiên là **bắt đầu với concrete types, refactor sang generics khi cần**. Đừng generic hóa ngay từ đầu. Viết `func ProcessOrders(orders []Order)` trước; chỉ khi bạn thấy mình copy-paste cùng logic cho `[]Product`, `[]User`, mới refactor sang `func Process[T OrderLike](items []T)`.

Nguyên tắc thứ hai: **constraint càng cụ thể càng tốt**. Dùng `comparable` thay vì `any` khi bạn cần `==`. Dùng constraint có method signature khi bạn cần gọi method. Constraint rộng (như `any`) có nghĩa compiler giúp bạn ít hơn.

Nguyên tắc thứ ba: **đặt tên type parameter có ý nghĩa** cho code phức tạp. `T` và `U` ổn cho hàm đơn giản, nhưng `[Key comparable, Value any]` rõ ràng hơn `[K comparable, V any]` trong context của một cache hoặc map.

Nguyên tắc thứ tư: **ưu tiên interface thường khi behavior quan trọng hơn structure**. Câu hỏi test: "Tôi cần code này hoạt động trên nhiều *kiểu dữ liệu khác nhau về cấu trúc* (int, string, float64)?" — đây là use case của generics. "Tôi cần code này hoạt động trên nhiều *đối tượng có behavior chung* (Dog, Cat, Bird đều Speak())?" — đây là use case của interface thường.

---

## 14. Ví dụ Thực Hành

### Bài tập: Implement Generic `Set[T comparable]`

Trong C#, bạn có `HashSet<T>`. Go không có built-in Set, nhưng với generics bạn có thể xây dựng một generic Set type. Implement type `Set[T comparable]` với các operations:

- `NewSet[T]()` — tạo set mới
- `Add(item T)` — thêm phần tử
- `Remove(item T)` — xóa phần tử
- `Contains(item T) bool` — kiểm tra tồn tại
- `Len() int` — số phần tử
- `Union(other Set[T]) Set[T]` — hợp hai set
- `Intersection(other Set[T]) Set[T]` — giao hai set
- `Difference(other Set[T]) Set[T]` — hiệu hai set (phần tử có trong set này nhưng không có trong set kia)
- `ToSlice() []T` — chuyển sang slice

**Giải pháp đầy đủ:**

```go
package main

import (
    "fmt"
    "strings"
)

// Set[T comparable] — generic set backed by a map
// T phải comparable vì cần làm map key
// Tương đương C#: HashSet<T> where T : IEquatable<T>
type Set[T comparable] struct {
    items map[T]struct{}
}

// NewSet tạo một Set rỗng
func NewSet[T comparable]() Set[T] {
    return Set[T]{items: make(map[T]struct{})}
}

// NewSetFrom tạo Set từ slice — tiện lợi cho khởi tạo
func NewSetFrom[T comparable](items []T) Set[T] {
    s := NewSet[T]()
    for _, item := range items {
        s.Add(item)
    }
    return s
}

// Add thêm một phần tử vào set
// Nếu đã tồn tại — không làm gì (đúng với set semantics)
func (s *Set[T]) Add(item T) {
    s.items[item] = struct{}{}
}

// Remove xóa một phần tử khỏi set
// Nếu không tồn tại — không làm gì
func (s *Set[T]) Remove(item T) {
    delete(s.items, item)
}

// Contains kiểm tra phần tử có trong set không
func (s Set[T]) Contains(item T) bool {
    _, ok := s.items[item]
    return ok
}

// Len trả về số phần tử trong set
func (s Set[T]) Len() int {
    return len(s.items)
}

// ToSlice chuyển set thành slice (thứ tự không đảm bảo)
func (s Set[T]) ToSlice() []T {
    result := make([]T, 0, len(s.items))
    for item := range s.items {
        result = append(result, item)
    }
    return result
}

// Union trả về tập hợp tất cả phần tử có trong s HOẶC other
// A ∪ B
func (s Set[T]) Union(other Set[T]) Set[T] {
    result := NewSet[T]()
    for item := range s.items {
        result.Add(item)
    }
    for item := range other.items {
        result.Add(item)
    }
    return result
}

// Intersection trả về tập hợp các phần tử có trong cả s VÀ other
// A ∩ B
func (s Set[T]) Intersection(other Set[T]) Set[T] {
    result := NewSet[T]()
    for item := range s.items {
        if other.Contains(item) {
            result.Add(item)
        }
    }
    return result
}

// Difference trả về tập hợp các phần tử có trong s nhưng KHÔNG có trong other
// A \ B
func (s Set[T]) Difference(other Set[T]) Set[T] {
    result := NewSet[T]()
    for item := range s.items {
        if !other.Contains(item) {
            result.Add(item)
        }
    }
    return result
}

// IsSubsetOf kiểm tra s có phải là tập con của other không
func (s Set[T]) IsSubsetOf(other Set[T]) bool {
    for item := range s.items {
        if !other.Contains(item) {
            return false
        }
    }
    return true
}

func main() {
    // ---- Test với int ----
    a := NewSetFrom([]int{1, 2, 3, 4, 5})
    b := NewSetFrom([]int{3, 4, 5, 6, 7})

    fmt.Println("a.Contains(3):", a.Contains(3)) // true
    fmt.Println("a.Contains(6):", a.Contains(6)) // false

    union := a.Union(b)
    fmt.Println("Union len:", union.Len()) // 7 (1,2,3,4,5,6,7)

    inter := a.Intersection(b)
    fmt.Println("Intersection len:", inter.Len()) // 3 (3,4,5)
    fmt.Println("Intersection contains 3:", inter.Contains(3)) // true
    fmt.Println("Intersection contains 1:", inter.Contains(1)) // false

    diff := a.Difference(b)
    fmt.Println("Difference len:", diff.Len()) // 2 (1,2)
    fmt.Println("Difference contains 1:", diff.Contains(1)) // true
    fmt.Println("Difference contains 3:", diff.Contains(3)) // false

    // Test Add/Remove
    a.Add(10)
    fmt.Println("After Add(10):", a.Contains(10)) // true
    a.Remove(10)
    fmt.Println("After Remove(10):", a.Contains(10)) // false

    // ---- Test với string ----
    langs1 := NewSetFrom([]string{"Go", "Python", "Java"})
    langs2 := NewSetFrom([]string{"Go", "Rust", "Java"})

    common := langs1.Intersection(langs2)
    fmt.Println("Common languages:", strings.Join(common.ToSlice(), ", ")) // Go, Java (thứ tự có thể khác)

    goOnly := langs1.Difference(langs2)
    fmt.Println("Only in langs1:", goOnly.ToSlice()) // [Python]

    // IsSubsetOf
    small := NewSetFrom([]string{"Go", "Java"})
    fmt.Println("small is subset of langs1:", small.IsSubsetOf(langs1)) // true
    fmt.Println("langs1 is subset of small:", langs1.IsSubsetOf(small)) // false

    // ---- Dùng với custom type ----
    type UserID int
    userSet := NewSet[UserID]()
    userSet.Add(UserID(1))
    userSet.Add(UserID(2))
    userSet.Add(UserID(1)) // duplicate — bị ignore
    fmt.Println("User set len:", userSet.Len()) // 2
}
```

### Bài tập 2: Generic Pipeline

Implement một `Pipeline[T any]` type cho phép chain các transformation functions — tương tự như LINQ method chaining trong C#.

```go
package main

import "fmt"

// Pipeline[T] — wrapper cho []T với fluent API
// Tương đương C#: IEnumerable<T> với LINQ
// Lưu ý: đây là cách Go giải quyết method chaining
// bằng cách wrap slice trong struct
type Pipeline[T any] struct {
    data []T
}

// From tạo Pipeline từ slice
func From[T any](data []T) Pipeline[T] {
    // Copy để tránh mutation của slice gốc
    copied := make([]T, len(data))
    copy(copied, data)
    return Pipeline[T]{data: copied}
}

// Filter giữ lại các phần tử thỏa điều kiện
// Tương đương C#: .Where(pred)
func (p Pipeline[T]) Filter(pred func(T) bool) Pipeline[T] {
    var result []T
    for _, v := range p.data {
        if pred(v) {
            result = append(result, v)
        }
    }
    return Pipeline[T]{data: result}
}

// Limit chỉ lấy n phần tử đầu tiên
// Tương đương C#: .Take(n)
func (p Pipeline[T]) Limit(n int) Pipeline[T] {
    if n > len(p.data) {
        n = len(p.data)
    }
    return Pipeline[T]{data: p.data[:n]}
}

// Skip bỏ qua n phần tử đầu tiên
// Tương đương C#: .Skip(n)
func (p Pipeline[T]) Skip(n int) Pipeline[T] {
    if n > len(p.data) {
        n = len(p.data)
    }
    return Pipeline[T]{data: p.data[n:]}
}

// ForEach thực thi action lên từng phần tử
// Tương đương C#: .ToList().ForEach(action)
func (p Pipeline[T]) ForEach(action func(T)) {
    for _, v := range p.data {
        action(v)
    }
}

// Collect trả về kết quả dưới dạng slice
// Tương đương C#: .ToList() hoặc .ToArray()
func (p Pipeline[T]) Collect() []T {
    result := make([]T, len(p.data))
    copy(result, p.data)
    return result
}

// Count trả về số phần tử
func (p Pipeline[T]) Count() int {
    return len(p.data)
}

// First trả về phần tử đầu tiên thỏa điều kiện, và bool cho biết có tìm được không
// Tương đương C#: .FirstOrDefault(pred)
func (p Pipeline[T]) First(pred func(T) bool) (T, bool) {
    for _, v := range p.data {
        if pred(v) {
            return v, true
        }
    }
    var zero T
    return zero, false
}

// MapPipeline là standalone function vì Go không cho generic method
// Tương đương C#: .Select(f)
func MapPipeline[T, U any](p Pipeline[T], f func(T) U) Pipeline[U] {
    result := make([]U, len(p.data))
    for i, v := range p.data {
        result[i] = f(v)
    }
    return Pipeline[U]{data: result}
}

type Product struct {
    Name     string
    Price    float64
    Category string
}

func main() {
    products := []Product{
        {Name: "Laptop", Price: 999.99, Category: "Electronics"},
        {Name: "Phone", Price: 599.99, Category: "Electronics"},
        {Name: "Book", Price: 29.99, Category: "Education"},
        {Name: "Desk", Price: 299.99, Category: "Furniture"},
        {Name: "Chair", Price: 199.99, Category: "Furniture"},
        {Name: "Monitor", Price: 499.99, Category: "Electronics"},
    }

    // Pipeline chaining — tương đương C# LINQ:
    // products.Where(p => p.Category == "Electronics")
    //         .OrderBy(p => p.Price) -- Go không có sort trong pipeline này
    //         .Take(2)
    //         .Select(p => p.Name)
    //         .ToList()
    electronicsNames := MapPipeline(
        From(products).
            Filter(func(p Product) bool { return p.Category == "Electronics" }).
            Limit(2),
        func(p Product) string { return p.Name },
    ).Collect()

    fmt.Println("Top 2 Electronics:", electronicsNames) // [Laptop Phone]

    // Tìm sản phẩm rẻ nhất dưới 500
    cheapest, found := From(products).
        Filter(func(p Product) bool { return p.Price < 500 }).
        First(func(p Product) bool { return true })

    if found {
        fmt.Printf("Cheapest under 500: %s ($%.2f)\n", cheapest.Name, cheapest.Price)
    }

    // Đếm số sản phẩm Electronics
    count := From(products).
        Filter(func(p Product) bool { return p.Category == "Electronics" }).
        Count()

    fmt.Println("Electronics count:", count) // 3
}
```

---

## Tóm tắt

Generics trong Go được thêm vào muộn nhưng được thiết kế cẩn thận để phù hợp với triết lý của Go: đủ mạnh để giải quyết các use case thực tế (data structures, utility functions), nhưng đủ đơn giản để không làm phức tạp hóa code. Sự thiếu vắng của generic methods là hạn chế rõ ràng nhất so với C#, và đó là lý do bạn sẽ thấy nhiều standalone functions trong code Go thay vì method chaining.

So với C#: nếu bạn dùng generics chủ yếu cho `List<T>`, `Dictionary<K,V>`, và một số LINQ operations — Go generics đủ mạnh để làm tất cả những điều đó. Nếu bạn dùng generic methods phức tạp với covariance/contravariance và specialization — bạn sẽ phải tìm cách khác trong Go.

Nguyên tắc thực hành: sử dụng generics khi bạn thấy mình copy-paste cùng một đoạn logic cho nhiều kiểu dữ liệu khác nhau. Đó là dấu hiệu rõ ràng nhất rằng generics đang giải quyết vấn đề thực sự.

---

**Tiếp theo:** [10. Project Structure trong Go](10-project-structure.md) — Cách tổ chức layout thư mục `cmd/`, `internal/`, layered architecture (handler → service → repository).
