# 27. Performance Engineering — Đo đúng trước khi tối ưu

> `19-performance-basics.md` đã cover profiling (`pprof`), benchmark cơ bản, escape analysis, struct alignment lý thuyết. Doc này tập trung vào **quy trình đo lường đáng tin cậy** — vì "tối ưu dựa trên 1 lần chạy benchmark" là sai lầm phổ biến dẫn đến kết luận sai.

---

## 1. Vấn đề: Benchmark 1 lần không đáng tin

```bash
go test -bench=BenchmarkParse ./...
```

Chạy lệnh trên 2 lần liên tiếp, bạn thường thấy số khác nhau — do CPU throttling, GC timing, OS scheduler, cache warm-up... Nếu bạn tối ưu code rồi so sánh với **1 lần chạy benchmark cũ**, bạn có thể kết luận sai: "nhanh hơn 5%" trong khi đó chỉ là nhiễu đo lường (noise).

```go
// calculator_bench_test.go
func BenchmarkStringConcat(b *testing.B) {
    for i := 0; i < b.N; i++ {
        s := ""
        for j := 0; j < 100; j++ {
            s += "x"
        }
        _ = s
    }
}

func BenchmarkStringBuilder(b *testing.B) {
    for i := 0; i < b.N; i++ {
        var sb strings.Builder
        for j := 0; j < 100; j++ {
            sb.WriteString("x")
        }
        _ = sb.String()
    }
}
```

---

## 2. `benchstat` — So sánh benchmark đúng cách bằng thống kê

`benchstat` (từ `golang.org/x/perf/cmd/benchstat`) chạy nhiều lần, tính trung bình và độ lệch, rồi cho biết sự khác biệt có **ý nghĩa thống kê** hay chỉ là nhiễu.

```bash
# Cài đặt
go install golang.org/x/perf/cmd/benchstat@latest

# Bước 1: Chạy benchmark NHIỀU LẦN (-count=10) TRƯỚC khi thay đổi code, lưu kết quả
go test -bench=. -benchmem -count=10 ./... > old.txt

# Bước 2: Thay đổi code (tối ưu), chạy lại benchmark tương tự
go test -bench=. -benchmem -count=10 ./... > new.txt

# Bước 3: So sánh
benchstat old.txt new.txt
```

Output mẫu:

```
name              old time/op    new time/op    delta
StringConcat-8      8.42µs ± 2%    8.51µs ± 3%   ~     (p=0.315 n=10+10)
StringBuilder-8     512ns ± 1%     498ns ± 2%   -2.73%  (p=0.001 n=10+10)

name              old alloc/op   new alloc/op   delta
StringConcat-8      3.61kB ± 0%    3.61kB ± 0%   ~     (all equal)
StringBuilder-8       112B ± 0%      112B ± 0%   ~     (all equal)
```

Cách đọc:
- **`~`** nghĩa là thay đổi **không có ý nghĩa thống kê** (`p` value cao) — đừng kết luận "nhanh hơn/chậm hơn" dựa vào nó.
- **`-2.73%`** với `p=0.001` (rất thấp) — thay đổi này **thực sự đáng tin**, không phải nhiễu.
- `-count=10` là tối thiểu hợp lý; với benchmark noisy (nhiều I/O, GC), có thể cần `-count=20` trở lên.

**Quy tắc thực chiến:** Không bao giờ nói "code này nhanh hơn X%" dựa trên 1 lần chạy `go test -bench`. Luôn dùng `-count=10+` và `benchstat` trước khi đưa số liệu vào PR description hoặc quyết định kiến trúc.

---

## 3. `benchmem` — Không chỉ tốc độ, còn là số lần allocate

```bash
go test -bench=. -benchmem ./...
```

```
BenchmarkStringConcat-8      141234    8420 ns/op    3610 B/op    99 allocs/op
BenchmarkStringBuilder-8    2341123     512 ns/op     112 B/op     1 allocs/op
```

`allocs/op` thường quan trọng hơn `ns/op` trong hệ thống có GC — nhiều allocation nhỏ tạo áp lực GC lớn hơn số liệu tốc độ đơn thuần thể hiện. `StringConcat` chậm gấp 16 lần và tốn 99 lần allocate cho mỗi lần chạy — mỗi `+=` tạo string mới (string trong Go immutable), trong khi `strings.Builder` tái sử dụng buffer nội bộ.

---

## 4. Quy trình tối ưu chuẩn (không đoán mò)

```
1. Đo baseline (benchmark + pprof) trước khi đổi bất cứ gì
2. Xác định bottleneck THỰC SỰ qua profile — đừng đoán
3. Sửa MỘT thay đổi tại một thời điểm
4. Đo lại với benchstat, so sánh có ý nghĩa thống kê không
5. Nếu có cải thiện thật, giữ lại. Nếu không, revert và thử hướng khác
6. Lặp lại
```

```go
// Ví dụ: dùng pprof để xác nhận bottleneck TRƯỚC khi optimize (không đoán)
func BenchmarkProcessOrders(b *testing.B) {
    f, _ := os.Create("cpu.prof")
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()

    orders := generateTestOrders(10000)
    b.ResetTimer() // loại bỏ thời gian setup khỏi kết quả đo

    for i := 0; i < b.N; i++ {
        ProcessOrders(orders)
    }
}
```

```bash
go test -bench=BenchmarkProcessOrders -cpuprofile=cpu.prof
go tool pprof -http=:8081 cpu.prof
# Xem "top" và "list ProcessOrders" — tìm chính xác dòng nào tốn CPU nhất
# TRƯỚC khi quyết định tối ưu phần nào
```

**Lỗi phổ biến:** tối ưu phần code "trông có vẻ chậm" (intuition) thay vì phần thực sự chiếm thời gian theo profile. Rất nhiều lần, bottleneck thực sự nằm ở chỗ không ai ngờ (ví dụ: `fmt.Sprintf` trong hot path, hay reflection ẩn trong thư viện log).

---

## 5. Struct Field Alignment — Thực hành với `fieldalignment`

`19-performance-basics.md` đã giải thích lý thuyết về memory alignment. Đây là cách kiểm tra thực tế:

```go
// BAD: struct này tốn 24 bytes trên 64-bit do padding
type BadOrder struct {
    Active bool    // 1 byte + 7 bytes padding (để Price align đúng 8-byte boundary)
    Price  float64 // 8 bytes
    Count  int32   // 4 bytes + 4 bytes padding (struct tổng phải là bội số của alignment lớn nhất)
}

// GOOD: sắp xếp field lớn → nhỏ, giảm padding — chỉ còn 16 bytes
type GoodOrder struct {
    Price  float64 // 8 bytes
    Count  int32   // 4 bytes
    Active bool    // 1 byte + 3 bytes padding
}
```

```bash
# Cài tool kiểm tra alignment tự động
go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest

# Chạy trên package của bạn
fieldalignment ./...

# Output mẫu:
# order.go:5:16: struct with 24 pointer bytes could be 16

# Tự động fix (cẩn thận: đổi thứ tự field có thể ảnh hưởng tới struct literal dùng positional field)
fieldalignment -fix ./...
```

**Khi nào việc này thực sự quan trọng:** struct được tạo **hàng triệu lần** (ví dụ: xử lý stream sự kiện, mỗi record là 1 struct). Với struct tạo vài chục lần trong toàn bộ vòng đời chương trình, tiết kiệm vài byte không đáng để hy sinh độ dễ đọc code (đặt field theo nhóm logic thường quan trọng hơn).

---

## 5.1. Runtime tự cải thiện: Swiss Tables map (Go 1.24+) và package `unique` (Go 1.23+)

Không phải mọi performance win đều đến từ việc bạn tự tối ưu code — đôi khi runtime Go tự thay đổi implementation bên dưới mà API không đổi:

- **Map implementation đổi sang Swiss Tables** (Go 1.24): cấu trúc dữ liệu bên dưới `map[K]V` được viết lại hoàn toàn (dựa trên thiết kế "Swiss Tables" của Abseil/Google), nhanh hơn và tốn ít memory hơn cho hầu hết workload — **không cần đổi một dòng code nào**. Đây là lý do luôn đáng để cập nhật Go version mới ngay cả khi codebase không đổi: một phần "tối ưu performance" tốt nhất chỉ đơn giản là `go get -tool` version compiler mới hơn.
- **Package `unique`** (Go 1.23+): giải quyết vấn đề "hàng triệu string giống hệt nhau tốn memory trùng lặp" (ví dụ: field `Country string` lặp lại "Vietnam" hàng triệu lần trong slice record). `unique.Make(v)` trả về một `unique.Handle[T]` — nếu gọi với cùng giá trị nhiều lần, chỉ 1 bản copy thực sự được giữ trong memory (interning), so sánh 2 handle cũng nhanh hơn so sánh string (so sánh con trỏ thay vì so sánh byte-by-byte):

```go
import "unique"

type Record struct {
    Country unique.Handle[string] // thay vì string thường
    Value   int
}

h1 := unique.Make("Vietnam")
h2 := unique.Make("Vietnam")
fmt.Println(h1 == h2)        // true — cùng interned value
fmt.Println(h1.Value())      // "Vietnam" — lấy lại string gốc
```

Chỉ dùng `unique` khi đã đo bằng `pprof`/benchmark thấy string trùng lặp thực sự là bottleneck memory — đây không phải thứ dùng mặc định cho mọi string field.

---

## 6. Checklist trước khi merge một PR "tối ưu performance"

- [ ] Có `benchstat` output với `-count=10+` chứng minh cải thiện có ý nghĩa thống kê (không chỉ `~`)
- [ ] Có `allocs/op` không tăng lên (hoặc giảm) — tránh trade tốc độ CPU lấy nhiều GC pressure hơn
- [ ] Có pprof profile xác nhận đây thực sự là bottleneck, không phải đoán
- [ ] Code sau khi tối ưu vẫn đọc được — nếu tối ưu làm code khó hiểu, cân nhắc có đáng không (đa số code không cần tối ưu tới mức hy sinh readability)
- [ ] Test vẫn pass, đặc biệt `go test -race` nếu thay đổi liên quan tới concurrency

---

## Bài tập

Viết 2 phiên bản của hàm `SumSquares(nums []int) int`: một dùng vòng lặp thường, một dùng goroutine chia nhỏ công việc (worker pool). Dùng `benchstat` với `-count=10` để xác định với kích thước input bao nhiêu thì phiên bản concurrent thực sự nhanh hơn có ý nghĩa thống kê (gợi ý: với input nhỏ, overhead tạo goroutine thường làm phiên bản concurrent CHẬM hơn).
