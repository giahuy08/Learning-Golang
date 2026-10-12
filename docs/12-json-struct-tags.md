# 📚 JSON & Struct Tags — Làm việc với JSON trong Go (dành cho C# developer)

---

## Tại sao JSON quan trọng đến mức cần một bài riêng?

Trong C#, bạn dùng `System.Text.Json` hoặc `Newtonsoft.Json` — đặt attribute `[JsonPropertyName("snake_case")]`, gọi `JsonSerializer.Serialize()`, xong. Go cũng làm được tất cả những thứ đó, nhưng cơ chế hoạt động khác hoàn toàn. Go dùng **struct tags** — một chuỗi metadata gắn vào field của struct — và thư viện `encoding/json` đọc metadata đó bằng reflection lúc runtime.

Điều này có nghĩa là lỗi trong struct tags (ví dụ: gõ sai dấu ngoặc kép, thêm khoảng trắng sai chỗ) **không gây compile error** — chúng âm thầm bị ignore, khiến JSON output sai mà bạn không nhận ra. Đây là nguồn gốc của nhiều bug khó chịu mà bài này sẽ chỉ rõ từng cái.

---

## 1. Cú pháp Struct Tags

Struct tag là một **raw string literal** (chuỗi bọc trong backtick `` ` ``) đặt ngay sau kiểu dữ liệu của field. Lý do dùng backtick thay vì dấu ngoặc kép là để bên trong có thể chứa dấu ngoặc kép mà không cần escape.

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email" validate:"required,email" db:"email_address"`
}
```

Cú pháp chung: `key:"value"`, nhiều tag cách nhau bằng **một khoảng trắng**. Đây là chuẩn do Go định nghĩa — thư viện `reflect` package đọc chuỗi này và parse theo format đó.

Trong C# bạn viết:
```csharp
[JsonPropertyName("name")]
[Required, EmailAddress]
public string Name { get; set; }
```

Go gộp tất cả vào một chuỗi duy nhất. Ít verbose hơn, nhưng dễ mắc lỗi hơn vì không có compile-time check.

**BAD — Các lỗi cú pháp âm thầm bị ignore:**

```go
// BAD: lỗi cú pháp struct tag — Go không báo lỗi compile, tag bị bỏ qua hoàn toàn
type BadUser struct {
    // Thiếu dấu ngoặc kép bên trong — tag này sẽ bị ignore
    Name string `json:name`           // SAI: phải là json:"name"

    // Khoảng trắng thừa sau dấu hai chấm — bị ignore
    Email string `json: "email"`      // SAI: không được có space sau dấu :

    // Nhiều tag nhưng dùng dấu phẩy thay vì space — chỉ tag đầu được đọc
    Age int `json:"age",db:"age"`     // SAI: phải dùng space, không dùng dấu phẩy
}
```

**GOOD — Cú pháp đúng:**

```go
// GOOD: cú pháp struct tag đúng chuẩn
type GoodUser struct {
    Name  string `json:"name"`
    Email string `json:"email" validate:"required,email" db:"email_address"`
    Age   int    `json:"age" db:"age"`
}
```

---

## 2. JSON Tags — Ánh xạ tên field

Go dùng quy ước **PascalCase** cho exported fields (field bắt đầu bằng chữ hoa) — vì đây là cách Go phân biệt public/private. Nhưng API thường dùng **snake_case**. Tag `json:"tên"` là cầu nối giữa hai thế giới này.

Tại sao Go lại design như vậy? Vì Go không có keyword `public`/`private` — thay vào đó, **chữ hoa/thường của ký tự đầu tiên** quyết định visibility. Field `name` (chữ thường) là unexported, `encoding/json` không thể đọc nó bằng reflection từ package khác. Vì vậy bạn **bắt buộc** phải viết `Name` (PascalCase) để JSON package có thể truy cập, rồi dùng tag để map sang `"name"` trong JSON.

```go
package main

import (
    "encoding/json"
    "fmt"
)

// Struct này đại diện cho một User trong hệ thống Go (PascalCase)
// nhưng API trả về snake_case — tag json: làm cầu nối
type User struct {
    ID        int    `json:"id"`
    FirstName string `json:"first_name"`  // PascalCase → snake_case
    LastName  string `json:"last_name"`
    Email     string `json:"email"`
    // Nếu không có tag, Go dùng tên field nguyên văn: "IsActive"
    IsActive  bool   `json:"is_active"`
}

func main() {
    u := User{
        ID:        1,
        FirstName: "Nguyen",
        LastName:  "Van A",
        Email:     "nguyenvana@example.com",
        IsActive:  true,
    }

    data, _ := json.Marshal(u)
    fmt.Println(string(data))
    // Output: {"id":1,"first_name":"Nguyen","last_name":"Van A","email":"nguyenvana@example.com","is_active":true}
}
```

**Không có tag — Go dùng tên field nguyên văn:**

```go
type WithoutTag struct {
    FirstName string  // → JSON key sẽ là "FirstName", không phải "first_name"
    isPrivate  string // → KHÔNG xuất hiện trong JSON vì unexported
}
```

---

## 3. `omitempty` — Bỏ qua field khi giá trị rỗng

`omitempty` bảo Go: "nếu field này có **zero value** thì đừng đưa vào JSON output". Zero value là `0` cho số, `""` cho string, `false` cho bool, `nil` cho pointer/slice/map.

Điều này tương đương `[JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]` trong C# .NET 6+.

```go
package main

import (
    "encoding/json"
    "fmt"
)

type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Description string  `json:"description,omitempty"` // bỏ qua nếu là ""
    Price       float64 `json:"price"`
    Discount    float64 `json:"discount,omitempty"` // bỏ qua nếu là 0
    Tags        []string `json:"tags,omitempty"`   // bỏ qua nếu là nil hoặc empty
}

func main() {
    p := Product{
        ID:    1,
        Name:  "Laptop",
        Price: 1500.00,
        // Description, Discount, Tags không set → zero value
    }

    data, _ := json.Marshal(p)
    fmt.Println(string(data))
    // Output: {"id":1,"name":"Laptop","price":1500}
    // Description, discount, tags bị bỏ qua vì là zero value
}
```

### GOTCHA: `omitempty` với Struct không hoạt động như bạn nghĩ

Đây là một trong những gotcha gây bug nhiều nhất. Khi field là **struct** (không phải pointer to struct), `omitempty` **không bao giờ bỏ qua** — vì Go không biết struct "rỗng" trông như thế nào. Zero value của struct là struct với tất cả fields là zero value — Go vẫn coi đó là "có giá trị".

**BAD — omitempty với struct không work:**

```go
// BAD: omitempty với struct value — KHÔNG bao giờ bỏ qua
type Order struct {
    ID      int     `json:"id"`
    Address Address `json:"address,omitempty"` // omitempty bị IGNORE với struct
}

type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
}

func main() {
    o := Order{ID: 1} // Address không set → zero value

    data, _ := json.Marshal(o)
    fmt.Println(string(data))
    // Output: {"id":1,"address":{"street":"","city":""}}
    // Address VẪN xuất hiện dù omitempty — đây là bug phổ biến!
}
```

**GOOD — Dùng pointer to struct để omitempty hoạt động:**

```go
package main

import (
    "encoding/json"
    "fmt"
)

type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
}

type Order struct {
    ID      int      `json:"id"`
    Address *Address `json:"address,omitempty"` // pointer: nil → bị bỏ qua
}

func main() {
    // Không có address
    o1 := Order{ID: 1}
    data1, _ := json.Marshal(o1)
    fmt.Println(string(data1))
    // Output: {"id":1}  ← address bị bỏ qua vì pointer là nil

    // Có address
    o2 := Order{
        ID:      2,
        Address: &Address{Street: "123 Le Loi", City: "HCMC"},
    }
    data2, _ := json.Marshal(o2)
    fmt.Println(string(data2))
    // Output: {"id":2,"address":{"street":"123 Le Loi","city":"HCMC"}}
}
```

**Bảng so sánh omitempty theo kiểu dữ liệu:**

| Kiểu       | Zero value       | omitempty bỏ qua?  |
|------------|------------------|--------------------|
| `string`   | `""`             | Có                 |
| `int`      | `0`              | Có                 |
| `bool`     | `false`          | Có                 |
| `float64`  | `0.0`            | Có                 |
| `[]T`      | `nil`            | Có (nil slice)     |
| `map[K]V`  | `nil`            | Có                 |
| `*T`       | `nil`            | Có                 |
| `struct`   | `T{}` (zero)     | **KHÔNG** — gotcha!|

---

## 4. `json:"-"` — Loại trừ field khỏi JSON

Đặt tag `json:"-"` để Go **hoàn toàn bỏ qua** field này khi Marshal và Unmarshal. Dùng cho các field nhạy cảm (password, internal state) hoặc field chỉ dùng nội bộ.

Khác với `omitempty` — `omitempty` chỉ bỏ qua khi rỗng, còn `json:"-"` bỏ qua **mọi lúc**.

```go
package main

import (
    "encoding/json"
    "fmt"
)

type UserAccount struct {
    ID           int    `json:"id"`
    Username     string `json:"username"`
    Email        string `json:"email"`
    PasswordHash string `json:"-"`           // KHÔNG BAO GIỜ xuất hiện trong JSON
    InternalNote string `json:"-"`           // field nội bộ, không expose ra API
    APIKey       string `json:"api_key,omitempty"` // chỉ bỏ qua khi rỗng
}

func main() {
    u := UserAccount{
        ID:           1,
        Username:     "nguyenvana",
        Email:        "a@example.com",
        PasswordHash: "$2a$10$abc...", // sẽ không xuất hiện trong JSON
        InternalNote: "test account",
        APIKey:       "sk-123",
    }

    data, _ := json.Marshal(u)
    fmt.Println(string(data))
    // Output: {"id":1,"username":"nguyenvana","email":"a@example.com","api_key":"sk-123"}
    // PasswordHash và InternalNote không xuất hiện
}
```

**Chú ý:** Nếu bạn muốn field tên thật sự là `-` trong JSON (hiếm gặp), dùng `json:"-,"` (có dấu phẩy phía sau).

---

## 5. `json.Marshal()` — Go struct → JSON bytes

`json.Marshal()` nhận vào bất kỳ giá trị Go nào và trả về `[]byte` (JSON bytes) cùng với `error`. Trong C# bạn dùng `JsonSerializer.Serialize()` trả về `string` — Go trả về `[]byte` vì Go ưu tiên hiệu năng (tránh allocate string không cần thiết).

```go
package main

import (
    "encoding/json"
    "fmt"
)

type Article struct {
    Title   string   `json:"title"`
    Content string   `json:"content"`
    Tags    []string `json:"tags"`
    Views   int      `json:"views"`
}

func main() {
    a := Article{
        Title:   "Go for C# Developers",
        Content: "Go is awesome...",
        Tags:    []string{"go", "programming"},
        Views:   1000,
    }

    // Marshal trả về []byte, không phải string
    data, err := json.Marshal(a)
    if err != nil {
        fmt.Println("Lỗi:", err)
        return
    }

    // Chuyển sang string để in ra
    fmt.Println(string(data))

    // Muốn JSON đẹp (pretty print) — tương đương JsonSerializerOptions.WriteIndented = true
    prettyData, _ := json.MarshalIndent(a, "", "  ")
    fmt.Println(string(prettyData))
}
```

Output của `MarshalIndent`:
```json
{
  "title": "Go for C# Developers",
  "content": "Go is awesome...",
  "tags": ["go", "programming"],
  "views": 1000
}
```

---

## 6. `json.Unmarshal()` — JSON bytes → Go struct

`json.Unmarshal()` parse JSON bytes vào một Go struct. Bạn phải truyền **pointer** đến struct (tương tự `ref` trong C#) để hàm có thể modify giá trị.

**Unknown fields bị ignore mặc định** — đây là hành vi được thiết kế có chủ đích. Go muốn code của bạn chịu đựng được khi API thêm fields mới (backward compatibility). Nếu bạn muốn strict mode, xem mục 10.

```go
package main

import (
    "encoding/json"
    "fmt"
)

type Config struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Database string `json:"database"`
}

func main() {
    // JSON có field "unknown_field" mà struct không có → bị ignore
    raw := `{
        "host": "localhost",
        "port": 5432,
        "database": "mydb",
        "unknown_field": "this will be ignored"
    }`

    var cfg Config
    // Truyền &cfg (pointer) — bắt buộc, không thể truyền cfg
    err := json.Unmarshal([]byte(raw), &cfg)
    if err != nil {
        fmt.Println("Lỗi:", err)
        return
    }

    fmt.Printf("Host: %s, Port: %d, DB: %s\n", cfg.Host, cfg.Port, cfg.Database)
    // Output: Host: localhost, Port: 5432, DB: mydb

    // Partial unmarshal: chỉ một số field có trong JSON → các field còn lại là zero value
    partialRaw := `{"host": "db.example.com"}`
    var cfg2 Config
    json.Unmarshal([]byte(partialRaw), &cfg2)
    fmt.Printf("Host: %s, Port: %d\n", cfg2.Host, cfg2.Port)
    // Output: Host: db.example.com, Port: 0  (zero value)
}
```

---

## 7. `json.NewEncoder` / `json.NewDecoder` — Streaming JSON

Với HTTP, file lớn, hoặc stream data, dùng `Encoder`/`Decoder` thay vì `Marshal`/`Unmarshal`. Lý do: `Marshal` đọc toàn bộ dữ liệu vào memory trước, còn `Encoder` ghi trực tiếp vào `io.Writer` — hiệu quả hơn cho HTTP response bodies.

Trong C# bạn thường dùng `JsonSerializer.SerializeAsync(stream, obj)` — đây là pattern tương đương trong Go.

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "strings"
)

type Response struct {
    Status  string `json:"status"`
    Message string `json:"message"`
}

func main() {
    // === ENCODER: ghi JSON ra io.Writer ===
    var buf bytes.Buffer
    encoder := json.NewEncoder(&buf)
    // Tùy chỉnh: indent cho đẹp (thay thế MarshalIndent)
    encoder.SetIndent("", "  ")

    resp := Response{Status: "ok", Message: "Thành công"}
    if err := encoder.Encode(resp); err != nil {
        fmt.Println("Lỗi encode:", err)
        return
    }
    // Encode tự thêm newline ở cuối — khác với Marshal
    fmt.Print(buf.String())

    // === DECODER: đọc JSON từ io.Reader ===
    raw := `{"status": "error", "message": "Not found"}`
    decoder := json.NewDecoder(strings.NewReader(raw))

    var result Response
    if err := decoder.Decode(&result); err != nil {
        fmt.Println("Lỗi decode:", err)
        return
    }
    fmt.Printf("Status: %s, Message: %s\n", result.Status, result.Message)

    // === Dùng trong HTTP handler (pattern phổ biến nhất) ===
    // func handleRequest(w http.ResponseWriter, r *http.Request) {
    //     var req MyRequest
    //     json.NewDecoder(r.Body).Decode(&req)   // đọc request body
    //     // ... xử lý ...
    //     json.NewEncoder(w).Encode(myResponse)   // ghi response
    // }
}
```

**Khi nào dùng gì:**

| Tình huống | Dùng |
|---|---|
| HTTP request/response body | `NewDecoder` / `NewEncoder` |
| File JSON lớn | `NewDecoder` / `NewEncoder` |
| Lưu vào `[]byte` hoặc `string` | `Marshal` / `Unmarshal` |
| Test, debug nhanh | `Marshal` / `Unmarshal` |

---

## 8. CRITICAL GOTCHA: nil slice vs empty slice trong JSON

Đây là gotcha gây bug production phổ biến nhất khi làm API. Trong Go, **nil slice** và **empty slice** là hai thứ khác nhau, và JSON marshal chúng khác nhau:

- `nil` slice → JSON `null`
- Empty slice (`[]T{}` hoặc `make([]T, 0)`) → JSON `[]`

Frontend JavaScript thường không phân biệt được `null` và `[]` và crash theo cách khó debug.

```go
package main

import (
    "encoding/json"
    "fmt"
)

type SearchResult struct {
    Query string   `json:"query"`
    Items []string `json:"items"`
}

func main() {
    // BAD: nil slice → JSON null — frontend nhận được null thay vì []
    r1 := SearchResult{Query: "golang"}
    // r1.Items là nil vì chưa gán
    data1, _ := json.Marshal(r1)
    fmt.Println(string(data1))
    // Output: {"query":"golang","items":null}  ← BUG: frontend expect []

    // GOOD: khởi tạo empty slice → JSON []
    r2 := SearchResult{
        Query: "golang",
        Items: make([]string, 0), // hoặc []string{}
    }
    data2, _ := json.Marshal(r2)
    fmt.Println(string(data2))
    // Output: {"query":"golang","items":[]}  ← ĐÚNG

    // Sau khi append, slice tự động không còn nil
    r3 := SearchResult{Query: "golang"}
    r3.Items = append(r3.Items, "result1")
    data3, _ := json.Marshal(r3)
    fmt.Println(string(data3))
    // Output: {"query":"golang","items":["result1"]}  ← Đúng vì append đã init
}
```

**Rule of thumb:** Khi một function trả về danh sách cho API, **luôn luôn** khởi tạo slice với `make([]T, 0)` trước khi trả về, ngay cả khi danh sách rỗng.

```go
// Pattern đúng cho API handler trả về danh sách
func getUsers() []User {
    users := make([]User, 0) // khởi tạo empty, không phải nil

    rows, err := db.Query("SELECT * FROM users")
    if err != nil {
        return users // trả về [] chứ không phải null
    }
    // ... scan rows vào users ...
    return users
}
```

---

## 9. Custom Marshaling — Implement Interface

Đôi khi bạn cần kiểm soát hoàn toàn cách JSON được tạo ra hoặc parse. Go dùng **interface** (không phải override method hay attribute như C#). Implement `json.Marshaler` và `json.Unmarshaler`:

```go
// json.Marshaler interface:
type Marshaler interface {
    MarshalJSON() ([]byte, error)
}

// json.Unmarshaler interface:
type Unmarshaler interface {
    UnmarshalJSON([]byte) error
}
```

Ví dụ thực tế: API trả về `status` dạng int (0, 1, 2) nhưng bạn muốn Go code dùng string ("pending", "active", "inactive"):

```go
package main

import (
    "encoding/json"
    "fmt"
)

type Status int

const (
    StatusPending  Status = 0
    StatusActive   Status = 1
    StatusInactive Status = 2
)

// Implement Marshaler: Status → JSON string
func (s Status) MarshalJSON() ([]byte, error) {
    names := map[Status]string{
        StatusPending:  "pending",
        StatusActive:   "active",
        StatusInactive: "inactive",
    }
    name, ok := names[s]
    if !ok {
        return nil, fmt.Errorf("unknown status: %d", s)
    }
    // Marshal string "pending" → `"pending"` (có dấu ngoặc kép)
    return json.Marshal(name)
}

// Implement Unmarshaler: JSON string → Status
func (s *Status) UnmarshalJSON(data []byte) error {
    var name string
    if err := json.Unmarshal(data, &name); err != nil {
        return err
    }
    codes := map[string]Status{
        "pending":  StatusPending,
        "active":   StatusActive,
        "inactive": StatusInactive,
    }
    code, ok := codes[name]
    if !ok {
        return fmt.Errorf("unknown status name: %s", name)
    }
    *s = code
    return nil
}

type User struct {
    ID     int    `json:"id"`
    Name   string `json:"name"`
    Status Status `json:"status"` // tự động dùng custom marshaling
}

func main() {
    u := User{ID: 1, Name: "An", Status: StatusActive}

    data, _ := json.Marshal(u)
    fmt.Println(string(data))
    // Output: {"id":1,"name":"An","status":"active"}

    // Unmarshal
    raw := `{"id":2,"name":"Binh","status":"pending"}`
    var u2 User
    json.Unmarshal([]byte(raw), &u2)
    fmt.Printf("Status value: %d\n", u2.Status) // Output: Status value: 0
}
```

---

## 10. Handling Unknown Fields — `DisallowUnknownFields()`

Mặc định Go ignore unknown fields. Đôi khi bạn muốn strict: nếu client gửi field lạ → trả về lỗi (bảo vệ API khỏi typo, phát hiện versioning mismatch).

```go
package main

import (
    "encoding/json"
    "fmt"
    "strings"
)

type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

func strictDecode(raw string) (*LoginRequest, error) {
    decoder := json.NewDecoder(strings.NewReader(raw))
    decoder.DisallowUnknownFields() // bật strict mode

    var req LoginRequest
    if err := decoder.Decode(&req); err != nil {
        return nil, err
    }
    return &req, nil
}

func main() {
    // Request đúng
    good := `{"username": "an", "password": "secret"}`
    req, err := strictDecode(good)
    fmt.Println(req, err) // &{an secret} <nil>

    // Request có field lạ — sẽ bị reject
    bad := `{"username": "an", "password": "secret", "admin": true}`
    req2, err2 := strictDecode(bad)
    fmt.Println(req2, err2)
    // Output: <nil> json: unknown field "admin"
}
```

---

## 11. `json.RawMessage` — Defer Parsing hoặc Embed Raw JSON

`json.RawMessage` là `[]byte` nhưng Go biết nó là JSON hợp lệ. Dùng khi:

1. Bạn nhận một JSON object nhưng chưa biết structure của một field cho đến khi đọc field khác (polymorphic JSON)
2. Bạn muốn chuyển tiếp raw JSON mà không parse

```go
package main

import (
    "encoding/json"
    "fmt"
)

// API event có "type" xác định structure của "data"
type Event struct {
    Type string          `json:"type"`
    Data json.RawMessage `json:"data"` // defer parsing
}

type UserCreatedData struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
}

type OrderPlacedData struct {
    OrderID string  `json:"order_id"`
    Amount  float64 `json:"amount"`
}

func processEvent(raw string) {
    var event Event
    json.Unmarshal([]byte(raw), &event)

    fmt.Printf("Event type: %s\n", event.Type)

    // Bây giờ mới parse data dựa trên type
    switch event.Type {
    case "user_created":
        var data UserCreatedData
        json.Unmarshal(event.Data, &data)
        fmt.Printf("New user: %s (%s)\n", data.UserID, data.Email)
    case "order_placed":
        var data OrderPlacedData
        json.Unmarshal(event.Data, &data)
        fmt.Printf("Order %s: $%.2f\n", data.OrderID, data.Amount)
    }
}

func main() {
    processEvent(`{"type":"user_created","data":{"user_id":"u123","email":"a@b.com"}}`)
    processEvent(`{"type":"order_placed","data":{"order_id":"o456","amount":99.99}}`)
}
```

---

## 12. Time Marshaling — `time.Time` và Custom Format

`time.Time` mặc định marshal sang **RFC3339** (ví dụ: `"2024-01-15T10:30:00Z"`). Điều này khác C# vốn dùng ISO 8601 với format linh hoạt hơn.

**Vấn đề phổ biến:** API của bạn nhận date dạng `"2024-01-15"` (không có time component), nhưng `time.Time` expect RFC3339 → lỗi parse.

```go
package main

import (
    "encoding/json"
    "fmt"
    "time"
)

// === Mặc định: RFC3339 ===
type Event struct {
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"` // tự động RFC3339
}

// === Custom format: chỉ ngày, không có giờ ===
type DateOnly struct {
    time.Time
}

const dateFormat = "2006-01-02" // Go dùng ngày MAGIC này để định format

func (d DateOnly) MarshalJSON() ([]byte, error) {
    return json.Marshal(d.Format(dateFormat))
}

func (d *DateOnly) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return err
    }
    t, err := time.Parse(dateFormat, s)
    if err != nil {
        return err
    }
    d.Time = t
    return nil
}

type Booking struct {
    RoomID    int      `json:"room_id"`
    CheckIn   DateOnly `json:"check_in"`
    CheckOut  DateOnly `json:"check_out"`
}

func main() {
    // RFC3339 mặc định
    e := Event{
        Name:      "Conference",
        CreatedAt: time.Now(),
    }
    data, _ := json.Marshal(e)
    fmt.Println(string(data))
    // Output: {"name":"Conference","created_at":"2024-01-15T10:30:00+07:00"}

    // Custom date format
    raw := `{"room_id":101,"check_in":"2024-03-01","check_out":"2024-03-05"}`
    var b Booking
    json.Unmarshal([]byte(raw), &b)
    fmt.Printf("Room %d: %s → %s\n",
        b.RoomID,
        b.CheckIn.Format("02/01/2006"),  // "01/03/2024"
        b.CheckOut.Format("02/01/2006"), // "05/03/2024"
    )
}
```

**Lưu ý quan trọng về "magic date" của Go:** Go dùng ngày `Mon Jan 2 15:04:05 MST 2006` (hay `2006-01-02T15:04:05`) làm reference time. `2006` = year, `01` = month, `02` = day, `15` = hour, `04` = minute, `05` = second. Đây là ngày Go được announce — khác hoàn toàn với C# dùng `"yyyy-MM-dd"`.

---

## 13. Validation Sau Unmarshal — `go-playground/validator`

`encoding/json` chỉ parse — không validate. JSON `{"age": -5}` sẽ unmarshal thành công dù age không hợp lệ. Cần thư viện riêng để validate, phổ biến nhất là `github.com/go-playground/validator/v10`.

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/go-playground/validator/v10"
)

type RegisterRequest struct {
    Username string `json:"username" validate:"required,min=3,max=20"`
    Email    string `json:"email"    validate:"required,email"`
    Age      int    `json:"age"      validate:"required,min=18,max=120"`
    Password string `json:"password" validate:"required,min=8"`
}

var validate = validator.New()

func parseAndValidate(raw string) (*RegisterRequest, error) {
    var req RegisterRequest
    if err := json.Unmarshal([]byte(raw), &req); err != nil {
        return nil, fmt.Errorf("invalid JSON: %w", err)
    }
    if err := validate.Struct(req); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    return &req, nil
}

func main() {
    // Valid request
    good := `{"username":"nguyenvana","email":"a@example.com","age":25,"password":"secret123"}`
    req, err := parseAndValidate(good)
    fmt.Println(req, err)

    // Invalid: email sai, age < 18
    bad := `{"username":"an","email":"not-an-email","age":15,"password":"short"}`
    req2, err2 := parseAndValidate(bad)
    fmt.Println(req2, err2)
    // Output: <nil> validation failed: Key: 'RegisterRequest.Email' Error:...
}
```

**Để chạy được ví dụ này:** `go get github.com/go-playground/validator/v10`

**Các validate tag phổ biến:**

| Tag | Ý nghĩa |
|-----|---------|
| `required` | Không được là zero value |
| `min=N,max=N` | Độ dài string hoặc giá trị số |
| `email` | Email hợp lệ |
| `url` | URL hợp lệ |
| `oneof=a b c` | Chỉ chấp nhận một trong các giá trị |
| `gt=0` | Lớn hơn 0 |
| `len=10` | Độ dài chính xác |

---

## 14. Number Gotcha — JSON số unmarshal vào `interface{}`

Đây là gotcha gây panic runtime khó debug. Khi unmarshal vào `interface{}` (tương đương `object` trong C#), Go map tất cả JSON numbers thành `float64` — ngay cả khi đó là số nguyên.

```go
package main

import (
    "encoding/json"
    "fmt"
)

func main() {
    raw := `{"id": 123, "score": 98.5, "name": "An"}`

    // Unmarshal vào map[string]interface{} — type không xác định trước
    var data map[string]interface{}
    json.Unmarshal([]byte(raw), &data)

    // BAD: cố cast id thành int — sẽ PANIC
    // id := data["id"].(int)  // PANIC: interface conversion: float64 is not int

    // GOOD: phải cast sang float64 trước
    id := data["id"].(float64)
    fmt.Printf("ID type: %T, value: %v\n", data["id"], id) // float64, 123
    fmt.Printf("ID as int: %d\n", int(id))                  // 123

    // BETTER: dùng json.Number để giữ nguyên type
    var data2 map[string]interface{}
    decoder := json.NewDecoder(strings.NewReader(raw))
    decoder.UseNumber() // numbers giữ nguyên là json.Number
    json.Unmarshal([]byte(raw), &data2) // dùng decoder thay

    // Với UseNumber, bạn phải dùng decoder, không phải Unmarshal
    // Xem pattern đúng bên dưới
}
```

**Pattern đúng với `UseNumber`:**

```go
package main

import (
    "encoding/json"
    "fmt"
    "strings"
)

func main() {
    raw := `{"id": 123, "score": 98.5}`

    decoder := json.NewDecoder(strings.NewReader(raw))
    decoder.UseNumber() // JSON numbers được giữ dạng json.Number

    var data map[string]interface{}
    decoder.Decode(&data)

    // json.Number có thể convert sang int64 hoặc float64
    id, _ := data["id"].(json.Number).Int64()
    score, _ := data["score"].(json.Number).Float64()
    fmt.Printf("ID: %d, Score: %.1f\n", id, score) // ID: 123, Score: 98.5

    // Tốt nhất: dùng typed struct để tránh hoàn toàn vấn đề này
    type Stats struct {
        ID    int     `json:"id"`
        Score float64 `json:"score"`
    }
    var s Stats
    json.Unmarshal([]byte(raw), &s)
    fmt.Printf("Typed — ID: %d, Score: %.1f\n", s.ID, s.Score) // luôn đúng type
}
```

**Nguyên tắc:** Tránh unmarshal vào `interface{}` khi biết trước structure. Dùng typed struct. Chỉ dùng `interface{}` khi JSON structure thực sự không biết trước.

---

## 15. Struct Tags Cho Các Thư Viện Khác

Struct tags không chỉ dành cho JSON. Các thư viện Go phổ biến đều dùng struct tags với key riêng:

```go
package main

// Ví dụ struct dùng nhiều loại tag cùng lúc — pattern phổ biến trong real project
type Employee struct {
    // json: tên trong JSON API
    // db: tên cột trong database (dùng với sqlx)
    // validate: validation rules (go-playground/validator)
    // form: tên field trong HTML form (dùng với gorilla/schema)
    // yaml: tên trong YAML config file (dùng với gopkg.in/yaml.v3)

    ID         int    `json:"id"          db:"id"           validate:"required"`
    FirstName  string `json:"first_name"  db:"first_name"   validate:"required,min=2" form:"first_name" yaml:"first_name"`
    LastName   string `json:"last_name"   db:"last_name"    validate:"required,min=2" form:"last_name"  yaml:"last_name"`
    Email      string `json:"email"       db:"email"        validate:"required,email" form:"email"      yaml:"email"`
    Department string `json:"department"  db:"dept_name"    validate:"required"       form:"department" yaml:"department"`
    Salary     float64 `json:"salary"    db:"salary"        validate:"required,gt=0"  yaml:"salary"`

    // Không expose password ra JSON, không lưu vào DB dạng plain text
    Password   string `json:"-"           db:"-"            validate:"required,min=8" form:"password"`

    // Internal field, không expose, không store
    CacheKey   string `json:"-"           db:"-"`
}
```

**Các thư viện và key phổ biến:**

| Thư viện | Tag key | Dùng để |
|----------|---------|---------|
| `encoding/json` | `json` | JSON serialization |
| `github.com/jmoiern/sqlx` | `db` | SQL query result mapping |
| `github.com/go-playground/validator` | `validate` | Input validation |
| `gopkg.in/yaml.v3` | `yaml` | YAML config parsing |
| `github.com/gorilla/schema` | `schema` | HTML form parsing |
| `github.com/BurntSushi/toml` | `toml` | TOML config parsing |
| `github.com/mitchellh/mapstructure` | `mapstructure` | map → struct conversion |

---

## Common Issues — Bảng Tổng Hợp

| Vấn đề | Triệu chứng | Nguyên nhân | Fix |
|--------|-------------|-------------|-----|
| Field không xuất hiện trong JSON | Field bị mất hoàn toàn | Field bắt đầu bằng chữ thường (unexported) | Đổi thành PascalCase |
| Tag bị ignore | JSON dùng tên field Go thay vì tên tag | Sai cú pháp tag (space, thiếu `"`) | Kiểm tra cú pháp backtick |
| `omitempty` không work với struct | Struct rỗng vẫn xuất hiện trong JSON | `omitempty` không biết struct "rỗng" | Dùng pointer `*Struct` |
| `[]` thành `null` trong JSON | Frontend nhận `null` thay vì `[]` | Slice là `nil` | `make([]T, 0)` hoặc `[]T{}` |
| Panic khi type assert `int` từ `interface{}` | `interface conversion failed` | JSON number → `float64` | Cast sang `float64` rồi convert, hoặc dùng typed struct |
| Time parse error | `cannot parse "2024-01-15"` | RFC3339 expect full datetime | Custom `MarshalJSON`/`UnmarshalJSON` |
| Unknown field không bị reject | Client typo field không bị phát hiện | Default: ignore unknown fields | `decoder.DisallowUnknownFields()` |
| Validate không chạy | Invalid data qua được | `json.Unmarshal` không validate | Thêm `validate.Struct()` sau unmarshal |

---

## Best Practices

Khi design API trong Go, hãy luôn tách biệt **API struct** (dùng cho JSON) và **domain struct** (dùng trong business logic). Đây là pattern tương đương DTO (Data Transfer Object) trong C#. API struct có thể thay đổi mà không ảnh hưởng đến domain logic, và ngược lại.

Với mọi API endpoint trả về danh sách, khởi tạo slice với `make([]T, 0)` trước khi populate — tránh việc trả về `null` cho frontend khi không có data. Một frontend developer sẽ cảm ơn bạn vì điều này.

Dùng `json.NewDecoder` / `json.NewEncoder` cho HTTP handlers thay vì `Marshal`/`Unmarshal`. Nó efficient hơn (streaming, không cần load toàn bộ body vào memory) và là Go idiom chuẩn.

Đặt `DisallowUnknownFields()` ở các endpoint nhận input từ client (POST, PUT, PATCH) — giúp phát hiện typo sớm và tăng bảo mật. Nhưng đừng dùng ở các endpoint call external API — API ngoài có thể thêm field mới bất cứ lúc nào.

Custom marshaling (`MarshalJSON`/`UnmarshalJSON`) nên được test kỹ, đặc biệt là edge cases: zero value, nil pointer, invalid input. Lỗi trong custom marshaling thường chỉ phát hiện ở runtime.

---

## Bài Tập Thực Hành

### Bài 1: API Response Builder

Xây dựng một hệ thống trả về API response chuẩn. Yêu cầu:
- Response wrapper với `success`, `data`, `error`, `meta`
- Danh sách users phải luôn là `[]` dù không có data, không phải `null`
- Password không được xuất hiện trong JSON
- `created_at` format `"DD/MM/YYYY HH:MM"`

```go
package main

import (
    "encoding/json"
    "fmt"
    "time"
)

// === Bài giải ===

// CustomTime cho phép format time theo ý muốn
type CustomTime struct {
    time.Time
}

func (ct CustomTime) MarshalJSON() ([]byte, error) {
    return json.Marshal(ct.Format("02/01/2006 15:04"))
}

func (ct *CustomTime) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return err
    }
    t, err := time.Parse("02/01/2006 15:04", s)
    if err != nil {
        return err
    }
    ct.Time = t
    return nil
}

// User domain struct
type User struct {
    ID           int        `json:"id"`
    Username     string     `json:"username"`
    Email        string     `json:"email"`
    PasswordHash string     `json:"-"`          // không expose
    CreatedAt    CustomTime `json:"created_at"` // custom format
}

// Meta thông tin phân trang
type Meta struct {
    Total   int `json:"total"`
    Page    int `json:"page"`
    PerPage int `json:"per_page"`
}

// APIResponse wrapper chuẩn
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Error   string      `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"` // pointer: nil → bỏ qua
}

func successResponse(data interface{}, meta *Meta) APIResponse {
    return APIResponse{Success: true, Data: data, Meta: meta}
}

func errorResponse(msg string) APIResponse {
    return APIResponse{Success: false, Data: nil, Error: msg}
}

func main() {
    // Giả lập database trả về 0 users — phải là [] chứ không phải null
    users := make([]User, 0)

    resp := successResponse(users, &Meta{Total: 0, Page: 1, PerPage: 20})
    data, _ := json.MarshalIndent(resp, "", "  ")
    fmt.Println("=== Empty list response ===")
    fmt.Println(string(data))

    // Response có data
    users2 := []User{
        {
            ID:           1,
            Username:     "nguyenvana",
            Email:        "a@example.com",
            PasswordHash: "$2a$10$secret_hash", // sẽ bị ẩn
            CreatedAt:    CustomTime{time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)},
        },
    }
    resp2 := successResponse(users2, &Meta{Total: 1, Page: 1, PerPage: 20})
    data2, _ := json.MarshalIndent(resp2, "", "  ")
    fmt.Println("\n=== With data response ===")
    fmt.Println(string(data2))

    // Error response
    errResp := errorResponse("User not found")
    data3, _ := json.MarshalIndent(errResp, "", "  ")
    fmt.Println("\n=== Error response ===")
    fmt.Println(string(data3))
}
```

**Kết quả mong đợi:**
```json
=== Empty list response ===
{
  "success": true,
  "data": [],
  "meta": {"total": 0, "page": 1, "per_page": 20}
}

=== With data response ===
{
  "success": true,
  "data": [
    {
      "id": 1,
      "username": "nguyenvana",
      "email": "a@example.com",
      "created_at": "15/01/2024 10:30"
    }
  ],
  "meta": {"total": 1, "page": 1, "per_page": 20}
}
```

---

### Bài 2: Polymorphic Event System

Xây dựng event processor có thể xử lý nhiều loại event khác nhau từ một JSON stream, sử dụng `json.RawMessage`.

```go
package main

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"
)

// === Bài giải ===

type EventType string

const (
    EventUserRegistered EventType = "user_registered"
    EventOrderPlaced    EventType = "order_placed"
    EventPaymentFailed  EventType = "payment_failed"
)

// Base event với type và timestamp
type BaseEvent struct {
    Type      EventType       `json:"type"`
    Timestamp time.Time       `json:"timestamp"`
    Data      json.RawMessage `json:"data"` // parse sau
}

type UserRegisteredData struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    Plan   string `json:"plan"`
}

type OrderPlacedData struct {
    OrderID  string  `json:"order_id"`
    UserID   string  `json:"user_id"`
    Amount   float64 `json:"amount"`
    Currency string  `json:"currency"`
}

type PaymentFailedData struct {
    OrderID string `json:"order_id"`
    Reason  string `json:"reason"`
    Retry   bool   `json:"retry"`
}

// Processor interface — mỗi event type implement riêng
type EventProcessor interface {
    Process(data json.RawMessage) error
}

type UserRegisteredProcessor struct{}
func (p UserRegisteredProcessor) Process(data json.RawMessage) error {
    var d UserRegisteredData
    if err := json.Unmarshal(data, &d); err != nil {
        return err
    }
    fmt.Printf("  [USER] New user %s (%s) on plan '%s'\n", d.UserID, d.Email, d.Plan)
    return nil
}

type OrderPlacedProcessor struct{}
func (p OrderPlacedProcessor) Process(data json.RawMessage) error {
    var d OrderPlacedData
    if err := json.Unmarshal(data, &d); err != nil {
        return err
    }
    fmt.Printf("  [ORDER] User %s placed order %s: %.2f %s\n",
        d.UserID, d.OrderID, d.Amount, d.Currency)
    return nil
}

type PaymentFailedProcessor struct{}
func (p PaymentFailedProcessor) Process(data json.RawMessage) error {
    var d PaymentFailedData
    if err := json.Unmarshal(data, &d); err != nil {
        return err
    }
    retryStr := "no retry"
    if d.Retry {
        retryStr = "will retry"
    }
    fmt.Printf("  [PAYMENT] Order %s failed: %s (%s)\n", d.OrderID, d.Reason, retryStr)
    return nil
}

// Registry map event type → processor
var processors = map[EventType]EventProcessor{
    EventUserRegistered: UserRegisteredProcessor{},
    EventOrderPlaced:    OrderPlacedProcessor{},
    EventPaymentFailed:  PaymentFailedProcessor{},
}

func processStream(stream string) {
    decoder := json.NewDecoder(strings.NewReader(stream))

    for decoder.More() {
        var event BaseEvent
        if err := decoder.Decode(&event); err != nil {
            fmt.Printf("  [ERROR] decode: %v\n", err)
            continue
        }

        fmt.Printf("Event at %s:\n", event.Timestamp.Format("15:04:05"))

        processor, ok := processors[event.Type]
        if !ok {
            fmt.Printf("  [WARN] Unknown event type: %s\n", event.Type)
            continue
        }

        if err := processor.Process(event.Data); err != nil {
            fmt.Printf("  [ERROR] process: %v\n", err)
        }
    }
}

func main() {
    // Giả lập nhiều events liên tiếp trong một stream (NDJSON)
    stream := `
{"type":"user_registered","timestamp":"2024-01-15T10:00:00Z","data":{"user_id":"u001","email":"an@example.com","plan":"pro"}}
{"type":"order_placed","timestamp":"2024-01-15T10:05:00Z","data":{"order_id":"ord001","user_id":"u001","amount":99.99,"currency":"USD"}}
{"type":"payment_failed","timestamp":"2024-01-15T10:05:30Z","data":{"order_id":"ord001","reason":"insufficient funds","retry":true}}
{"type":"unknown_event","timestamp":"2024-01-15T10:06:00Z","data":{}}
`
    processStream(stream)
}
```

---

## Tổng Kết

JSON trong Go đơn giản ở bề mặt nhưng có nhiều gotcha quan trọng. Ba điều cần nhớ nhất:

**Một:** `nil` slice → `null` trong JSON, `make([]T, 0)` → `[]`. Luôn khởi tạo slice trước khi trả về từ API handler.

**Hai:** `omitempty` với struct value không hoạt động — dùng pointer `*Struct` nếu muốn optional struct field.

**Ba:** JSON không validate — sau `Unmarshal` phải chạy validator thủ công nếu cần validate business rules.

Hiểu được những gotcha này sẽ giúp bạn tránh được phần lớn bugs JSON trong real project mà không cần debug mò.

---

**Next:** [13-http-server.md](13-http-server.md) — Xây dựng HTTP server trong Go: `net/http`, routing, middleware, và các pattern phổ biến dùng trong production API.
