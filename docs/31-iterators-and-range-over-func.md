# 31. Range-over-func Iterators (Go 1.23+)

> Dành cho C# developer: đây chính là `IEnumerable<T>`/`yield return` của Go, nhưng đến muộn hơn nhiều (Go 1.23, giữa 2024) và dùng generics + function type thay vì một interface đặc biệt trong ngôn ngữ. Liên quan: [09-generics.md](09-generics.md), [04-goroutines-channels.md](04-goroutines-channels.md) (so sánh với cách "giả iterator" bằng channel trước đây).

---

## 1. Vấn đề trước Go 1.23

Trước Go 1.23, `for range` chỉ hoạt động với 4 loại: slice, array, map, channel, string, và (từ 1.22) số nguyên (`for i := range 10`). Nếu bạn muốn viết một kiểu dữ liệu tuỳ chỉnh (ví dụ cây nhị phân, linked list, hoặc một nguồn dữ liệu lazy đọc từ file) và cho phép người dùng `for range` qua nó, bạn có 2 lựa chọn tồi:

1. **Trả về slice** — phải load toàn bộ dữ liệu vào memory trước, không lazy được.
2. **Dùng channel + goroutine** — lazy được, nhưng phải quản lý goroutine leak nếu caller `break` giữa chừng (giống C# `IEnumerable` không có vấn đề này vì nó không cần thread riêng).

```go
// Cách cũ: "giả iterator" bằng channel — có nguy cơ leak goroutine
func TreeValuesChan(t *Tree) <-chan int {
    ch := make(chan int)
    go func() {
        defer close(ch)
        var walk func(*Tree)
        walk = func(n *Tree) {
            if n == nil {
                return
            }
            walk(n.Left)
            ch <- n.Value // nếu caller break sớm, goroutine này BỊ KẸT MÃI MÃI ở đây
            walk(n.Right)
        }
        walk(t)
    }()
    return ch
}
```

## 2. Range-over-func — giải pháp chính thức

Go 1.23 cho phép `for range` nhận một **hàm** có 1 trong các chữ ký sau, thay vì chỉ nhận slice/map/channel:

```go
func(yield func() bool)
func(yield func(V) bool)
func(yield func(K, V) bool)
```

`yield` là hàm do runtime của `for range` truyền vào — gọi `yield(v)` tương đương "trả về giá trị `v` cho vòng lặp", và giá trị boolean trả về cho biết caller có muốn tiếp tục hay đã `break`.

```go
package main

import "fmt"

type Tree struct {
    Left, Right *Tree
    Value       int
}

// TreeValues là một iterator: nhận yield, tự gọi yield(v) cho từng phần tử.
// Chữ ký func(yield func(int) bool) khớp iter.Seq[int].
func (t *Tree) All(yield func(int) bool) {
    if t == nil {
        return
    }
    if !t.Left.All(yield) {
        return // caller đã break, dừng đệ quy ngay — không leak gì cả
    }
    if !yield(t.Value) {
        return
    }
    t.Right.All(yield)
}

func main() {
    t := &Tree{
        Left:  &Tree{Value: 1},
        Value: 2,
        Right: &Tree{Value: 3},
    }

    for v := range t.All {
        fmt.Println(v)
        if v == 2 {
            break // dừng lazy, không có goroutine nào bị leak
        }
    }
}
```

So với bản channel: không có goroutine, không có channel, không có nguy cơ leak khi `break` — vì toàn bộ chạy đồng bộ trên cùng 1 goroutine của caller, giống hệt cách `yield return` hoạt động trong C#.

## 3. Package `iter` — type hoá iterator

Package chuẩn `iter` (stdlib, Go 1.23+) định nghĩa 2 type để bạn không phải viết chữ ký hàm dài dòng mỗi lần:

```go
package iter

type Seq[V any]     func(yield func(V) bool)
type Seq2[K, V any]  func(yield func(K, V) bool)
```

Viết lại ví dụ trên dùng `iter.Seq`:

```go
import "iter"

func (t *Tree) All() iter.Seq[int] {
    return func(yield func(int) bool) {
        // ... như trên
    }
}

// Cách gọi giống hệt: for v := range t.All() { ... }
```

`iter.Seq2[K, V]` dùng cho trường hợp cần cặp key-value, ví dụ duyệt map hoặc index+value:

```go
func Enumerate[T any](s []T) iter.Seq2[int, T] {
    return func(yield func(int, T) bool) {
        for i, v := range s {
            if !yield(i, v) {
                return
            }
        }
    }
}

// for i, v := range Enumerate(myList) { ... }
```

## 4. `slices` và `maps` đã được cập nhật để dùng `iter.Seq`

Đây là thay đổi API quan trọng bạn cần biết vì nó ảnh hưởng trực tiếp code đang dùng `slices`/`maps` (đã học ở [09-generics.md](09-generics.md)):

```go
import (
    "fmt"
    "maps"
    "slices"
)

m := map[string]int{"a": 1, "b": 2, "c": 3}

// maps.Keys/Values trả về iter.Seq, KHÔNG phải slice — dùng trực tiếp với range:
for k := range maps.Keys(m) {
    fmt.Println(k)
}

// Muốn có slice thật (vd. để sort), convert bằng slices.Collect hoặc slices.Sorted:
keys := slices.Sorted(maps.Keys(m)) // []string đã sort sẵn
fmt.Println(keys) // [a b c]

// slices.Values làm chiều ngược lại: []T -> iter.Seq[T]
nums := []int{1, 2, 3}
for v := range slices.Values(nums) {
    fmt.Println(v)
}

// slices.Collect: iter.Seq[T] -> []T
doubled := slices.Collect(func(yield func(int) bool) {
    for v := range slices.Values(nums) {
        if !yield(v * 2) {
            return
        }
    }
})
fmt.Println(doubled) // [2 4 6]
```

**Bẫy hay gặp:** code cũ viết theo tutorial trước Go 1.23 giả định `maps.Keys(m)` trả về `[]K` — code đó **không còn compile** trên Go 1.23+ vì kiểu trả về đã đổi sang `iter.Seq[K]`. Nếu gặp lỗi kiểu `cannot range over maps.Keys(m) (variable of type []K)` khi đọc code cũ, hoặc ngược lại lỗi "cannot use []K as iter.Seq[K]" khi copy code mới vào Go cũ — kiểm tra lại version.

## 5. Khi nào nên tự viết iterator thay vì trả slice

- **Dữ liệu lớn hoặc vô hạn** (đọc file theo dòng, stream từ network, cây/graph lớn) — iterator chỉ tính giá trị tiếp theo khi cần, không load hết vào memory.
- **Muốn caller có thể `break` sớm mà không tốn chi phí tính toán phần còn lại** — ví dụ tìm phần tử đầu tiên thỏa điều kiện trong danh sách rất dài.
- **Không** nên dùng cho danh sách nhỏ, cố định — trả `[]T` đơn giản và dễ đọc hơn nhiều so với việc tạo một closure iterator không cần thiết.

## 6. Bài tập

### Exercise 1
Viết hàm `Filter[T any](seq iter.Seq[T], pred func(T) bool) iter.Seq[T]` — nhận một iterator và một predicate, trả về iterator mới chỉ yield những phần tử thỏa `pred`. Test bằng cách filter số chẵn từ `slices.Values([]int{1,2,3,4,5,6})`.

### Exercise 2
Viết một iterator `Lines(path string) iter.Seq2[int, string]` đọc file theo từng dòng (dùng `bufio.Scanner`), yield `(số dòng, nội dung dòng)`. So sánh với cách bạn sẽ làm việc này bằng channel — chỉ ra chỗ code ngắn hơn và không cần lo goroutine leak.

### Exercise 3
Refactor `TreeValuesChan` (ví dụ channel ở mục 1) trong một codebase giả định của bạn thành `iter.Seq[int]` dùng range-over-func, rồi viết benchmark so sánh 2 cách (`go test -bench`) — channel version sẽ chậm hơn đáng kể do overhead tạo goroutine + channel cho mỗi lần gọi.

---

## Tóm tắt

Range-over-func là cách Go chính thức hoá "lazy iteration" mà không cần channel/goroutine — tương đương `yield return` của C# nhưng dùng closure thay vì cú pháp ngôn ngữ đặc biệt. `iter.Seq[V]`/`iter.Seq2[K,V]` là 2 type chuẩn để không phải viết chữ ký hàm dài dòng. `slices`/`maps` package đã migrate sang dùng `iter.Seq` từ Go 1.23 — đây là thay đổi breaking API cần biết khi đọc code cũ hoặc nâng cấp Go version.

**Liên quan:** [09-generics.md](09-generics.md), [30-language-trends-review-2026.md](30-language-trends-review-2026.md)
</content>
</invoke>
