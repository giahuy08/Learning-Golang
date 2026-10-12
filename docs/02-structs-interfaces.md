# 📚 Structs & Interfaces — OOP trong Go (dành cho C# developer)

---

## 1. Go không có Class — và đó là chủ ý

Trong C#, class là đơn vị cơ bản để tổ chức code: nó vừa chứa data, vừa chứa behavior, vừa hỗ trợ inheritance.

Go tách biệt hoàn toàn:
- **Struct** — chứa data
- **Method** — behavior, gắn vào struct nhưng khai báo bên ngoài
- **Interface** — định nghĩa contract, không có implementation

Lý do: inheritance trong OOP thường dẫn đến class hierarchy phức tạp, khó refactor. Go thay bằng **composition** — ghép các phần nhỏ lại với nhau.

---

## 2. Struct

Struct là tập hợp các fields, tương tự class trong C# nhưng không có behavior.

```go
type User struct {
    ID       int
    Name     string
    Email    string
    IsActive bool
}
```

So sánh với C#:
```csharp
// C# — class có thể vừa có data vừa có method
public class User {
    public int ID { get; set; }
    public string Name { get; set; }
    public string Email { get; set; }
    public bool IsActive { get; set; }
}
```

### Khởi tạo struct

```go
// GOOD: named fields — rõ ràng, không bị lỗi khi thêm field sau này
u := User{
    ID:       1,
    Name:     "Nguyen Van A",
    Email:    "a@example.com",
    IsActive: true,
}

// BAD: positional — dễ nhầm nếu đổi thứ tự field
u := User{1, "Nguyen Van A", "a@example.com", true}

// Zero value — tất cả field được set về zero value tương ứng
var u2 User
// u2.ID = 0, u2.Name = "", u2.Email = "", u2.IsActive = false

// Truy cập field — giống C#
fmt.Println(u.Name)
u.IsActive = false
```

### Struct là Value Type

Đây là điểm quan trọng khác C#. Trong C#, class là reference type — gán cho biến mới thì cả hai biến cùng trỏ vào một object. Trong Go, **struct là value type** — gán cho biến mới thì copy toàn bộ.

```go
u1 := User{Name: "Alice"}
u2 := u1     // copy toàn bộ struct
u2.Name = "Bob"

fmt.Println(u1.Name) // "Alice" — u1 không bị ảnh hưởng
fmt.Println(u2.Name) // "Bob"

// Muốn behavior như C# class (share reference), dùng pointer
u3 := &u1    // u3 là *User, trỏ vào u1
u3.Name = "Charlie"
fmt.Println(u1.Name) // "Charlie" — u1 bị ảnh hưởng
```

---

## 3. Methods

Method trong Go là function có thêm **receiver** — chỉ định nó thuộc về type nào.

```csharp
// C# — method khai báo bên trong class
public class Rectangle {
    public double Width { get; set; }
    public double Height { get; set; }

    public double Area() => Width * Height;
}
```

```go
// Go — method khai báo bên ngoài struct, nhưng gắn với struct qua receiver
type Rectangle struct {
    Width  float64
    Height float64
}

// (r Rectangle) là receiver — đặt tên ngắn theo convention (chữ cái đầu của type)
func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}
```

Cách dùng giống hệt C#:
```go
rect := Rectangle{Width: 10, Height: 5}
fmt.Println(rect.Area()) // 50
```

### Value Receiver vs Pointer Receiver

Đây là khái niệm không có trong C# và hay gây nhầm lẫn.

**Value receiver `(r Rectangle)`**: Go copy toàn bộ struct khi gọi method. Method nhận bản copy, không thể thay đổi original.

**Pointer receiver `(r *Rectangle)`**: Go truyền địa chỉ của struct. Method có thể thay đổi original.

```go
// Value receiver — KHÔNG thay đổi được original
func (r Rectangle) BadScale(factor float64) {
    r.Width *= factor   // chỉ thay đổi bản copy trong method
    r.Height *= factor  // original không bị ảnh hưởng
}

// Pointer receiver — CÓ THỂ thay đổi original
func (r *Rectangle) Scale(factor float64) {
    r.Width *= factor   // thay đổi original
    r.Height *= factor
}

rect := Rectangle{Width: 10, Height: 5}
rect.BadScale(2)
fmt.Println(rect.Width) // 10 — không thay đổi

rect.Scale(2)
fmt.Println(rect.Width) // 20 — đã thay đổi
```

**Khi nào dùng pointer receiver?**
1. Khi method cần modify struct — bắt buộc dùng pointer receiver
2. Khi struct lớn — tránh copy tốn kém
3. **Consistency**: nếu một method trong type dùng pointer receiver, tất cả nên dùng pointer receiver

---

## 4. Constructor Pattern

Go không có constructor (`new User()`). Convention là tạo function bắt đầu bằng `New`:

```csharp
// C# constructor
public class User {
    public User(int id, string name, string email) {
        if (string.IsNullOrEmpty(name))
            throw new ArgumentException("name cannot be empty");
        ID = id;
        Name = name;
        Email = email;
    }
}
var user = new User(1, "Alice", "alice@example.com");
```

```go
// Go — unexported fields để bảo vệ dữ liệu
type User struct {
    id    int    // chữ thường = chỉ dùng trong package này
    name  string
    email string
}

// Constructor function — trả về pointer và error
func NewUser(id int, name, email string) (*User, error) {
    if name == "" {
        return nil, errors.New("name cannot be empty")
    }
    if !strings.Contains(email, "@") {
        return nil, errors.New("invalid email format")
    }
    return &User{id: id, name: name, email: email}, nil
}

// Getter methods vì fields unexported
func (u *User) Name() string { return u.name }
func (u *User) Email() string { return u.email }

// Gọi
user, err := NewUser(1, "Alice", "alice@example.com")
if err != nil {
    log.Fatal(err)
}
fmt.Println(user.Name()) // "Alice"
```

Tại sao trả về `*User` thay vì `User`? Hai lý do:
1. Tránh copy struct lớn
2. Cho phép trả về `nil` khi có lỗi (value type không thể nil)

---

## 5. Struct Embedding — thay thế Inheritance

Go không có `extends`. Thay vào đó, bạn **embed** struct này vào struct khác.

```csharp
// C# inheritance
public class Animal {
    public string Name { get; set; }
    public virtual string Speak() => $"{Name} makes a sound";
}

public class Dog : Animal {
    public string Breed { get; set; }
    public override string Speak() => $"{Name} says: Woof!";
}
```

```go
// Go embedding
type Animal struct {
    Name string
}

func (a Animal) Speak() string {
    return a.Name + " makes a sound"
}

type Dog struct {
    Animal        // embedded — không có tên field, chỉ có type
    Breed  string
}

// Override method của Animal
func (d Dog) Speak() string {
    return d.Name + " says: Woof!"
}
```

Khi embed, Dog tự động có tất cả fields và methods của Animal:

```go
d := Dog{
    Animal: Animal{Name: "Rex"},
    Breed:  "Labrador",
}

fmt.Println(d.Name)         // "Rex" — truy cập trực tiếp field của Animal
fmt.Println(d.Breed)        // "Labrador"
fmt.Println(d.Speak())      // "Rex says: Woof!" — method của Dog (override)
fmt.Println(d.Animal.Speak()) // "Rex makes a sound" — gọi tường minh Animal.Speak
```

**Embedding không phải inheritance.** Sự khác biệt quan trọng: Dog không "là một" Animal theo nghĩa polymorphism. Bạn không thể pass Dog vào nơi nhận Animal. Để có polymorphism, dùng interface.

---

## 6. Interfaces

Interface trong Go định nghĩa **một tập hợp methods** mà một type phải có. Khác C#/Java ở một điểm mấu chốt: **implicit implementation** — type không cần khai báo `implements`.

```csharp
// C# — phải khai báo tường minh
public interface IShape {
    double Area();
    double Perimeter();
}

public class Rectangle : IShape { // phải khai báo ": IShape"
    public double Area() => Width * Height;
    public double Perimeter() => 2 * (Width + Height);
}
```

```go
// Go — implicit
type Shape interface {
    Area() float64
    Perimeter() float64
}

type Rectangle struct {
    Width, Height float64
}

// Rectangle tự động implement Shape chỉ vì có đủ 2 methods này
// Không cần "implements Shape" hay gì cả
func (r Rectangle) Area() float64      { return r.Width * r.Height }
func (r Rectangle) Perimeter() float64 { return 2 * (r.Width + r.Height) }
```

**Tại sao implicit?** Trong C#/Java, nếu interface thêm method mới, tất cả class implement nó phải cập nhật. Với implicit implementation, bạn có thể định nghĩa interface *sau* khi đã có các types — rất hữu ích khi viết code, sau đó nhận ra "ồ các types này có điểm chung, tạo interface để abstract".

### Dùng Interface để viết code linh hoạt

```go
type Circle struct {
    Radius float64
}

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }

// Hàm nhận interface — hoạt động với bất kỳ Shape nào
func printShapeInfo(s Shape) {
    fmt.Printf("Area: %.2f, Perimeter: %.2f\n", s.Area(), s.Perimeter())
}

// Slice của interface — chứa nhiều types khác nhau
shapes := []Shape{
    Rectangle{10, 5},
    Circle{7},
    Rectangle{3, 4},
}

for _, s := range shapes {
    printShapeInfo(s) // Go gọi đúng method của từng type
}
```

---

## 7. Interface Composition

Trong Go, nên tạo **interface nhỏ** rồi ghép lại — thay vì một interface lớn.

```go
// Từng interface nhỏ, focused
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

// Ghép lại khi cần
type ReadWriter interface {
    Reader
    Writer
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

**Tại sao interface nhỏ tốt hơn?** Càng ít method trong interface, càng nhiều types có thể implement nó. `io.Reader` chỉ có 1 method — file, network connection, HTTP response, string reader đều implement được. Nếu thêm method vào, nhiều types sẽ không đáp ứng được.

---

## 8. Type Assertion & Type Switch

Khi có một interface, đôi khi cần lấy lại concrete type bên dưới:

```go
var s Shape = Rectangle{10, 5}

// Type assertion — lấy Rectangle từ Shape
// BAD: nếu sai type → panic
r := s.(Circle)  // panic: interface conversion: *Rectangle is not Circle

// GOOD: dùng ok pattern — giống C# "as" operator
r, ok := s.(Rectangle)
if ok {
    fmt.Println("Là rectangle, width:", r.Width)
} else {
    fmt.Println("Không phải rectangle")
}
```

Khi cần xử lý nhiều types khác nhau, dùng **type switch**:

```go
// Tương đương C# switch với pattern matching
func describe(i interface{}) string {
    switch v := i.(type) {
    case int:
        return fmt.Sprintf("số nguyên: %d", v)
    case string:
        return fmt.Sprintf("chuỗi: %q", v)
    case bool:
        return fmt.Sprintf("boolean: %v", v)
    case Rectangle:
        return fmt.Sprintf("hình chữ nhật: %v x %v", v.Width, v.Height)
    default:
        return fmt.Sprintf("loại không biết: %T", v)
    }
}
```

---

## 9. Nil Interface — Cạm bẫy quan trọng

Một interface trong Go thực ra chứa 2 thành phần bên trong: `(type, value)`. Interface chỉ nil khi **cả type và value đều nil**.

```go
// Cạm bẫy phổ biến
func getUser(id int) error {
    var err *MyError = nil   // typed nil pointer

    if id < 0 {
        err = &MyError{Message: "invalid id"}
    }

    return err  // BAD: trả về (*MyError)(nil) — interface có type nhưng value nil
                // người gọi kiểm tra err != nil sẽ thấy TRUE dù không có lỗi!
}

// GOOD: trả về nil tường minh
func getUser(id int) error {
    if id < 0 {
        return &MyError{Message: "invalid id"}
    }
    return nil  // nil interface thực sự
}
```

---

## 10. Best Practices

**Accept interfaces, return structs** — Hàm nên nhận interface (linh hoạt) và trả về concrete type (rõ ràng):
```go
// GOOD
func NewService(db Database) *UserService { ... }

// BAD — trả về interface, caller không biết capabilities thực sự
func NewService(db Database) Service { ... }
```

**Không define interface trước khi cần** — Trong C#, thường tạo interface trước rồi mới implement. Go khuyến khích ngược lại: viết concrete type trước, chỉ tạo interface khi có ít nhất 2 types cần dùng chung.

**Pointer receiver nhất quán** — Nếu một method trong type dùng pointer receiver, tất cả method nên dùng pointer receiver để tránh nhầm lẫn.

---

## 11. Bài tập

### Bài 1: Shape Calculator
Implement Triangle và Square, sau đó tính tổng diện tích:

```go
package main

import (
    "fmt"
    "math"
)

type Shape interface {
    Area() float64
}

type Rectangle struct{ Width, Height float64 }
type Circle struct{ Radius float64 }
type Triangle struct{ Base, Height float64 }
type Square struct{ Side float64 }

func (r Rectangle) Area() float64 { return r.Width * r.Height }
func (c Circle) Area() float64    { return math.Pi * c.Radius * c.Radius }
func (t Triangle) Area() float64  { return 0.5 * t.Base * t.Height }
func (s Square) Area() float64    { return s.Side * s.Side }

func totalArea(shapes []Shape) float64 {
    total := 0.0
    for _, s := range shapes {
        total += s.Area()
    }
    return total
}

func main() {
    shapes := []Shape{
        Rectangle{10, 5},
        Circle{7},
        Triangle{6, 4},
        Square{3},
    }
    fmt.Printf("Total area: %.2f\n", totalArea(shapes))
}
```

### Bài 2: Logger Interface
```go
package main

import "fmt"

type Logger interface {
    Log(message string)
}

type ConsoleLogger struct{}

func (l ConsoleLogger) Log(message string) {
    fmt.Println("[LOG]", message)
}

type MemoryLogger struct {
    logs []string
}

func (l *MemoryLogger) Log(message string) {
    l.logs = append(l.logs, message)
}

func (l *MemoryLogger) GetLogs() []string {
    return l.logs
}

func processOrder(id int, logger Logger) {
    logger.Log(fmt.Sprintf("Processing order %d", id))
    logger.Log(fmt.Sprintf("Order %d completed", id))
}

func main() {
    // Dùng ConsoleLogger
    processOrder(1, ConsoleLogger{})

    // Dùng MemoryLogger để test (không cần print ra màn hình)
    ml := &MemoryLogger{}
    processOrder(2, ml)
    fmt.Println("Logged:", ml.GetLogs())
}
```

---

## Next
➡️ [`03-error-handling.md`](03-error-handling.md) — Error patterns trong Go
