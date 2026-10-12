# 📚 Go Database (SQL) — Làm việc với Database trong Go (dành cho C# developer)

---

## 1. Tổng quan — So sánh với C# ecosystem

Trong C#, bạn có nhiều lựa chọn để làm việc với database: ADO.NET là tầng thấp nhất (tương đương JDBC), Dapper là micro-ORM ánh xạ kết quả query vào object, và Entity Framework Core là full ORM với LINQ, migration, change tracking. Mỗi tầng càng cao thì càng tiện nhưng càng ẩn đi chi tiết thực thi, đôi khi sinh ra query tệ mà bạn không biết.

Go chọn một triết lý khác: **standard library `database/sql` đóng vai trò ADO.NET**, còn các thư viện bên ngoài thêm tiện ích chứ không che đi SQL. Trong thế giới Go production, bạn vẫn viết SQL thật — không có LINQ, không có magic. Lý do là Go coi SQL là domain language chuyên biệt, không nên replace bằng abstraction của ngôn ngữ lập trình. sqlc thậm chí đi ngược hoàn toàn: bạn viết SQL trước, nó generate Go code type-safe từ SQL đó.

Bảng so sánh nhanh:

| C# | Go tương đương |
|---|---|
| ADO.NET (`SqlConnection`, `SqlCommand`) | `database/sql` |
| Dapper | `sqlx`, `pgxscan` |
| Entity Framework Core | Không có tương đương thực sự; gần nhất là GORM (tránh dùng) |
| SQL Server Provider | `github.com/denisenkom/go-mssqldb` |
| Npgsql (PostgreSQL) | `github.com/lib/pq` hoặc `github.com/jackc/pgx/v5` |
| EF Migrations | `golang-migrate/migrate` |
| Stored Procedure via EF | `db.QueryContext()` gọi trực tiếp |
| `IDbTransaction` | `*sql.Tx` |
| `DbContext` (scoped per request) | `*sql.DB` (singleton, pool) |
| Dapper `Query<T>()` | `pgxscan.Select()` hoặc manual `rows.Scan()` |

Điểm khác biệt quan trọng nhất: trong C#, `SqlConnection` hay `DbContext` thường được DI inject theo scope (per-request). Trong Go, `*sql.DB` **là connection pool toàn cục, tồn tại suốt vòng đời app**. Bạn share một `*sql.DB` duy nhất cho toàn bộ app, không tạo mới theo request.

---

## 2. Driver Pattern — Tại sao phải `import _`?

`database/sql` là interface thuần túy — nó không biết gì về PostgreSQL, MySQL hay SQLite. Để kết nối được database cụ thể, bạn cần một **driver** implement interface đó.

Cơ chế hoạt động: package driver (ví dụ `github.com/lib/pq`) có một hàm `init()` tự động chạy khi package được import. Hàm `init()` đó gọi `sql.Register("postgres", &drv{})` để đăng ký driver vào registry của `database/sql`. Sau đó, khi bạn gọi `sql.Open("postgres", ...)`, standard library tìm driver đã đăng ký theo tên đó.

Vì bạn không cần dùng trực tiếp bất kỳ symbol nào từ package driver (chỉ cần nó chạy `init()`), Go bắt buộc phải dùng `import _ "..."` — dấu underscore báo cho compiler biết "tôi import package này chỉ để side effect, không dùng gì từ nó cả." Nếu bạn import mà không dùng symbol nào mà lại không có dấu `_`, Go compiler sẽ báo lỗi.

```go
// BAD — compiler báo lỗi "imported and not used"
import "github.com/lib/pq"

// GOOD — import để lấy side effect (đăng ký driver)
import _ "github.com/lib/pq"
```

So sánh với C#: trong C#, khi bạn cài `Microsoft.EntityFrameworkCore.SqlServer`, nó tự register qua DI bằng `.UseSqlServer(...)`. Không có equivalent của `import _` vì C# dùng reflection để scan assembly. Go không có reflection-based auto-discovery — mọi thứ phải explicit.

---

## 3. `sql.Open()` vs `sql.OpenDB()` — Không phải là kết nối thật!

Đây là điểm mà rất nhiều C# developer nhầm khi mới học Go.

`sql.Open("postgres", connectionString)` **KHÔNG kết nối tới database**. Nó chỉ validate cú pháp connection string và tạo ra một `*sql.DB` struct đại diện cho **connection pool**. Kết nối thật chỉ được tạo khi bạn thực sự gửi query đầu tiên.

Để kiểm tra kết nối có hoạt động không, bạn phải gọi `db.PingContext(ctx)` riêng:

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "time"

    _ "github.com/lib/pq" // đăng ký driver postgres
)

func main() {
    // sql.Open() trả về pool, chưa kết nối thật
    db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=disable")
    if err != nil {
        // Lỗi này chỉ là lỗi parse connection string, KHÔNG phải lỗi kết nối
        log.Fatal(err)
    }
    defer db.Close()

    // Phải Ping để kiểm tra kết nối thật
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        log.Fatalf("cannot connect to database: %v", err)
    }

    fmt.Println("Connected to database!")
}
```

`sql.OpenDB(connector)` là phiên bản low-level hơn, nhận vào một `driver.Connector` thay vì connection string. Dùng khi bạn cần custom behavior lúc tạo connection (ví dụ: pgx/v5 dùng cách này để bypass giới hạn của `database/sql` standard interface).

Trong C# với ADO.NET:
```csharp
// C# — SqlConnection thật sự kết nối khi gọi Open()
using var conn = new SqlConnection(connectionString);
await conn.OpenAsync(); // kết nối thật xảy ra ở đây
```

Trong Go, việc "open" và việc "connect" được tách biệt rõ ràng, và pool tự manage connections.

---

## 4. Connection Pool Settings — Tại sao quan trọng?

`*sql.DB` là connection pool, không phải single connection. Pool tự động tạo và tái sử dụng connections. Nếu không configure đúng, app có thể exhaust connections của database server hoặc ngược lại giữ quá nhiều idle connections tốn tài nguyên.

```go
package main

import (
    "database/sql"
    "log"
    "time"

    _ "github.com/lib/pq"
)

func setupDB(dsn string) *sql.DB {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal(err)
    }

    // Số connection tối đa được mở cùng lúc (cả idle + active)
    // PostgreSQL mặc định cho phép 100 connections tổng cộng
    // Nếu bạn chạy nhiều instance app, phải chia đều
    db.SetMaxOpenConns(25)

    // Số connection idle tối đa giữ trong pool
    // Nên <= MaxOpenConns
    // Quá cao: lãng phí tài nguyên; quá thấp: phải tạo connection mới thường xuyên
    db.SetMaxIdleConns(10)

    // Thời gian tối đa một connection được sống
    // Quan trọng để tránh dùng connection đã bị DB server đóng âm thầm
    // Nên thấp hơn timeout của database server (PostgreSQL mặc định: không giới hạn)
    db.SetConnMaxLifetime(30 * time.Minute)

    // Thời gian tối đa một connection nằm idle trong pool
    // Giúp giải phóng connection không cần thiết
    db.SetConnMaxIdleTime(10 * time.Minute)

    return db
}
```

Trong C#, các setting tương đương nằm trong connection string: `Max Pool Size=25;Min Pool Size=5;Connection Lifetime=1800`. Go tách thành methods riêng cho rõ ràng hơn.

Rule of thumb: `MaxOpenConns` đặt bằng số cores của DB server hoặc theo công thức `(số lõi CPU * 2) + số spindle disk`. Cho web API: bắt đầu với 25, monitor rồi tune.

---

## 5. Query Patterns — Ba hàm chính

Go có ba hàm query chính tương ứng với ba tình huống khác nhau. Tất cả đều phải có `Context` version — sẽ giải thích ở mục 6.

### 5.1 `db.QueryContext()` — Nhiều row

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    _ "github.com/lib/pq"
)

type User struct {
    ID    int
    Name  string
    Email string
}

func getActiveUsers(ctx context.Context, db *sql.DB) ([]User, error) {
    // QueryContext trả về *sql.Rows — một cursor, chưa load hết data vào memory
    rows, err := db.QueryContext(ctx, "SELECT id, name, email FROM users WHERE active = $1", true)
    if err != nil {
        return nil, fmt.Errorf("query users: %w", err)
    }
    // QUAN TRỌNG: luôn defer rows.Close() ngay sau khi check err
    // Nếu quên, connection sẽ không được trả về pool — pool exhaustion!
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        // Thứ tự trong Scan() phải khớp với SELECT column order
        if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
            return nil, fmt.Errorf("scan user: %w", err)
        }
        users = append(users, u)
    }

    // QUAN TRỌNG: kiểm tra rows.Err() sau vòng lặp
    // Lỗi có thể xảy ra trong quá trình đọc (network drop, timeout)
    // mà không bị bắt trong rows.Next()
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows iteration: %w", err)
    }

    return users, nil
}
```

So sánh C# với Dapper:
```csharp
// C# Dapper — tự scan vào object, tiện hơn nhưng ẩn đi mapping logic
var users = await conn.QueryAsync<User>("SELECT id, name, email FROM users WHERE active = @active", new { active = true });
```

### 5.2 `db.QueryRowContext()` — Một row

```go
func getUserByID(ctx context.Context, db *sql.DB, id int) (User, error) {
    var u User
    // QueryRowContext không trả err ngay — lỗi được defer đến lúc Scan()
    err := db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE id = $1", id).
        Scan(&u.ID, &u.Name, &u.Email)

    if err == sql.ErrNoRows {
        // Không tìm thấy row — đây là case bình thường, không phải lỗi hệ thống
        return User{}, fmt.Errorf("user %d not found: %w", id, sql.ErrNoRows)
    }
    if err != nil {
        return User{}, fmt.Errorf("get user %d: %w", id, err)
    }

    return u, nil
}
```

`sql.ErrNoRows` tương đương với kiểm tra `result == null` trong C#. Phải handle riêng case này.

### 5.3 `db.ExecContext()` — INSERT/UPDATE/DELETE

```go
func createUser(ctx context.Context, db *sql.DB, name, email string) (int64, error) {
    // ExecContext trả về sql.Result, không trả rows
    result, err := db.ExecContext(ctx,
        "INSERT INTO users (name, email, active) VALUES ($1, $2, $3)",
        name, email, true,
    )
    if err != nil {
        return 0, fmt.Errorf("create user: %w", err)
    }

    // LastInsertId() — PostgreSQL không hỗ trợ với lib/pq, dùng RETURNING thay thế
    // MySQL hỗ trợ LastInsertId()
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return 0, err
    }

    return rowsAffected, nil
}

// Với PostgreSQL, dùng RETURNING để lấy ID vừa insert
func createUserPG(ctx context.Context, db *sql.DB, name, email string) (int, error) {
    var newID int
    err := db.QueryRowContext(ctx,
        "INSERT INTO users (name, email, active) VALUES ($1, $2, $3) RETURNING id",
        name, email, true,
    ).Scan(&newID)
    if err != nil {
        return 0, fmt.Errorf("create user: %w", err)
    }
    return newID, nil
}
```

---

## 6. Luôn Dùng Context — Không Dùng `db.Query()` Không Có Context

Trong C#, cancellation được truyền qua `CancellationToken` — tham số optional, thường được truyền xuống từ controller qua DI. Trong Go, `context.Context` là tham số **bắt buộc đầu tiên** của mọi I/O operation, không phải optional.

`database/sql` có hai phiên bản của mỗi hàm: `db.Query()` (không có context) và `db.QueryContext(ctx, ...)` (có context). Luôn dùng phiên bản có context vì:

**Tại sao context quan trọng với database?** Khi HTTP request bị cancel (client disconnect, timeout), nếu không có context, query vẫn tiếp tục chạy trên DB server, tốn tài nguyên vô ích. Với `QueryContext`, khi context bị cancel, Go sẽ gửi lệnh cancel tới database driver, database server hủy query đang chạy.

```go
// BAD — không có context, không thể cancel, không thể timeout
rows, err := db.Query("SELECT * FROM orders WHERE user_id = $1", userID)

// GOOD — có context từ HTTP request, tự động cancel khi request bị cancel
func handleGetOrders(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // context này tự hủy khi client disconnect
    rows, err := db.QueryContext(ctx, "SELECT * FROM orders WHERE user_id = $1", userID)
    // ...
}

// GOOD — timeout riêng cho DB operation nếu muốn
func getOrders(ctx context.Context, db *sql.DB, userID int) ([]Order, error) {
    // Tạo timeout 3 giây cho query này
    queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()

    rows, err := db.QueryContext(queryCtx, "SELECT * FROM orders WHERE user_id = $1", userID)
    // ...
}
```

---

## 7. Scanning Rows — `rows.Err()` Sau Vòng Lặp Là Bắt Buộc

Đây là lỗi phổ biến nhất của Go developer mới: **quên kiểm tra `rows.Err()` sau vòng `for rows.Next()`**.

`rows.Next()` trả về `false` trong hai trường hợp: (1) đã đọc hết tất cả rows, (2) có lỗi xảy ra trong quá trình đọc. Bạn không phân biệt được hai case này nếu chỉ dừng vòng lặp. `rows.Err()` cho bạn biết lý do `Next()` dừng.

```go
// BAD — quên rows.Err(), silent data loss nếu có lỗi network giữa chừng
func getProductsBad(ctx context.Context, db *sql.DB) ([]Product, error) {
    rows, err := db.QueryContext(ctx, "SELECT id, name, price FROM products")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var products []Product
    for rows.Next() {
        var p Product
        rows.Scan(&p.ID, &p.Name, &p.Price)
        products = append(products, p)
    }
    // Lỗi: không kiểm tra rows.Err()
    // Nếu network drop sau khi đọc 50 rows, hàm trả về 50 rows như bình thường
    // Caller nghĩ data đầy đủ nhưng thực ra thiếu
    return products, nil
}

// GOOD — kiểm tra đầy đủ
func getProductsGood(ctx context.Context, db *sql.DB) ([]Product, error) {
    rows, err := db.QueryContext(ctx, "SELECT id, name, price FROM products")
    if err != nil {
        return nil, fmt.Errorf("query products: %w", err)
    }
    defer rows.Close()

    var products []Product
    for rows.Next() {
        var p Product
        // Luôn check err từ Scan()
        if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
            return nil, fmt.Errorf("scan product: %w", err)
        }
        products = append(products, p)
    }

    // PHẢI check sau vòng lặp
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterating products: %w", err)
    }

    return products, nil
}
```

Xử lý nullable columns với `sql.NullString`, `sql.NullInt64`:

```go
type Product struct {
    ID          int
    Name        string
    Description sql.NullString // nullable VARCHAR
    Price       float64
}

// Scan nullable field
var p Product
err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price)

// Đọc giá trị nullable
if p.Description.Valid {
    fmt.Println(p.Description.String)
} else {
    fmt.Println("no description")
}
```

---

## 8. Prepared Statements — Khi Nào Cần?

Prepared statement là query được parse và lên kế hoạch thực thi trước bởi DB server, sau đó chỉ cần gửi parameters mỗi lần execute. Lợi ích: giảm overhead parse, server-side caching execution plan.

Trong `database/sql`, mỗi lần bạn gọi `db.QueryContext()` với placeholder, driver thực ra đã tự động prepare và execute. Nhưng `sql.Stmt` cho phép bạn **reuse** prepared statement đã tạo, tránh overhead prepare lặp lại khi gọi cùng query nhiều lần trong vòng lặp.

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    _ "github.com/lib/pq"
)

func bulkInsertUsers(ctx context.Context, db *sql.DB, users []User) error {
    // BAD — prepare lại mỗi lần trong vòng lặp, tốn kém
    for _, u := range users {
        _, err := db.ExecContext(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", u.Name, u.Email)
        if err != nil {
            return err
        }
    }
    return nil
}

func bulkInsertUsersBetter(ctx context.Context, db *sql.DB, users []User) error {
    // GOOD — prepare một lần, execute nhiều lần
    stmt, err := db.PrepareContext(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)")
    if err != nil {
        return fmt.Errorf("prepare insert: %w", err)
    }
    defer stmt.Close() // trả statement về pool khi done

    for _, u := range users {
        if _, err := stmt.ExecContext(ctx, u.Name, u.Email); err != nil {
            return fmt.Errorf("insert user %s: %w", u.Name, err)
        }
    }
    return nil
}
```

**Khi nào KHÔNG cần `sql.Stmt`?** Nếu bạn chỉ gọi query một lần, dùng `db.QueryContext()` trực tiếp là đủ và đơn giản hơn. Prepared statement chỉ có lợi khi execute cùng query nhiều lần liên tiếp.

So sánh C#: `SqlCommand` với `Parameters` cũng là prepared statement bên dưới. Dapper tự handle việc này.

---

## 9. Transactions — Pattern Đúng Với `defer`

Transaction trong Go tương đương `IDbTransaction` trong ADO.NET hay `using var transaction = await context.Database.BeginTransactionAsync()` trong EF Core. Điều quan trọng là đảm bảo transaction luôn được rollback nếu có lỗi, kể cả panic.

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    _ "github.com/lib/pq"
)

// transferMoney chuyển tiền giữa hai tài khoản — phải atomic
func transferMoney(ctx context.Context, db *sql.DB, fromID, toID int, amount float64) error {
    // BeginTx nhận TxOptions để set isolation level
    tx, err := db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable, // mạnh nhất, tránh phantom read
    })
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }

    // Pattern chuẩn: defer rollback ngay sau BeginTx
    // Nếu Commit() đã được gọi thành công, Rollback() là no-op (không làm gì)
    // Nếu hàm return sớm do lỗi hoặc panic, Rollback() đảm bảo cleanup
    defer tx.Rollback()

    // Trừ tiền từ tài khoản nguồn
    result, err := tx.ExecContext(ctx,
        "UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1",
        amount, fromID,
    )
    if err != nil {
        return fmt.Errorf("debit account %d: %w", fromID, err)
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        // Không đủ tiền hoặc account không tồn tại
        return fmt.Errorf("insufficient balance or account %d not found", fromID)
    }

    // Cộng tiền vào tài khoản đích
    if _, err := tx.ExecContext(ctx,
        "UPDATE accounts SET balance = balance + $1 WHERE id = $2",
        amount, toID,
    ); err != nil {
        return fmt.Errorf("credit account %d: %w", toID, err)
    }

    // Ghi audit log trong cùng transaction
    if _, err := tx.ExecContext(ctx,
        "INSERT INTO transfer_log (from_id, to_id, amount) VALUES ($1, $2, $3)",
        fromID, toID, amount,
    ); err != nil {
        return fmt.Errorf("log transfer: %w", err)
    }

    // Commit — nếu thành công, defer Rollback() phía trên là no-op
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transfer: %w", err)
    }

    return nil
}
```

**Tại sao `defer tx.Rollback()` trước khi Commit?** Vì nếu bất kỳ bước nào fail và hàm return error, `defer` đảm bảo rollback được gọi tự động. Nếu không có `defer`, bạn phải gọi `tx.Rollback()` ở mỗi error branch — dễ quên. Sau khi `tx.Commit()` thành công, gọi `tx.Rollback()` sẽ trả về `sql.ErrTxDone` nhưng `defer` ignore nó, nên an toàn.

So sánh C# pattern:
```csharp
// C# — using statement tự gọi Dispose() (= Rollback nếu chưa Commit)
await using var tx = await conn.BeginTransactionAsync();
try {
    // ... operations ...
    await tx.CommitAsync();
} catch {
    await tx.RollbackAsync();
    throw;
}
```

---

## 10. SQL Injection Prevention — Luôn Dùng Placeholder

SQL injection là lỗ hổng bảo mật nghiêm trọng nhất trong ứng dụng database. Go không tự động bảo vệ bạn — bạn phải chủ động.

Rule duy nhất: **không bao giờ dùng string formatting (`fmt.Sprintf`, string concatenation) để build SQL query có user input.**

```go
// BAD — SQL injection hoàn toàn có thể
// Attacker gửi email = "' OR '1'='1"
// Query trở thành: SELECT * FROM users WHERE email = '' OR '1'='1'
// Kết quả: trả về toàn bộ users!
func getUserByEmailBad(ctx context.Context, db *sql.DB, email string) (User, error) {
    query := fmt.Sprintf("SELECT id, name FROM users WHERE email = '%s'", email)
    var u User
    err := db.QueryRowContext(ctx, query).Scan(&u.ID, &u.Name)
    return u, err
}

// BAD — vẫn là SQL injection dù dùng "+" thay vì Sprintf
func getUserByEmailBad2(ctx context.Context, db *sql.DB, email string) (User, error) {
    query := "SELECT id, name FROM users WHERE email = '" + email + "'"
    var u User
    err := db.QueryRowContext(ctx, query).Scan(&u.ID, &u.Name)
    return u, err
}

// GOOD — dùng placeholder $1 (PostgreSQL) hoặc ? (MySQL/SQLite)
// Driver tự động escape giá trị, injection không thể xảy ra
func getUserByEmailGood(ctx context.Context, db *sql.DB, email string) (User, error) {
    var u User
    err := db.QueryRowContext(ctx,
        "SELECT id, name FROM users WHERE email = $1",
        email, // giá trị được truyền riêng, không embed vào query string
    ).Scan(&u.ID, &u.Name)
    if err == sql.ErrNoRows {
        return User{}, fmt.Errorf("user not found")
    }
    return u, err
}
```

**Điều gì về dynamic ORDER BY hay table name?** Placeholder chỉ dùng được cho values, không dùng được cho column names hay table names. Nếu cần dynamic, dùng whitelist:

```go
// GOOD — whitelist cho dynamic ORDER BY
func getUsers(ctx context.Context, db *sql.DB, sortBy string) ([]User, error) {
    // Whitelist các column được phép sort
    allowed := map[string]string{
        "name":       "name",
        "email":      "email",
        "created_at": "created_at",
    }
    col, ok := allowed[sortBy]
    if !ok {
        col = "id" // default
    }

    // Safe vì col luôn là giá trị từ whitelist, không phải user input
    query := fmt.Sprintf("SELECT id, name, email FROM users ORDER BY %s", col)
    // ...
    _ = query
    return nil, nil
}
```

---

## 11. pgx/v5 — Tại Sao Nên Dùng Thay lib/pq?

`lib/pq` là driver PostgreSQL truyền thống — stable, phổ biến, nhưng đã ngừng phát triển tính năng mới. `pgx/v5` là thế hệ tiếp theo với nhiều lợi thế:

Thứ nhất, **hiệu năng tốt hơn**: pgx dùng binary protocol thay vì text protocol, giảm overhead serialization. Thứ hai, **type support phong phú hơn**: native support cho `pgtype.UUID`, `pgtype.JSONB`, PostgreSQL arrays, `HSTORE`, `CIDR`, v.v. mà không cần cast thủ công. Thứ ba, **connection pool riêng** (`pgxpool.Pool`) được thiết kế tốt hơn `database/sql` pool. Thứ tư, **pgxscan** giúp scan rows vào struct tự động như Dapper.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()

    // pgxpool.New thay vì sql.Open
    // Connection string format khác: dùng URL-style
    pool, err := pgxpool.New(ctx, "postgres://postgres:secret@localhost:5432/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Configure pool
    config, _ := pgxpool.ParseConfig("postgres://postgres:secret@localhost:5432/mydb")
    config.MaxConns = 25
    config.MinConns = 5

    pool, err = pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Query với pgx native
    rows, err := pool.Query(ctx, "SELECT id, name FROM users WHERE active = $1", true)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    for rows.Next() {
        var id int
        var name string
        if err := rows.Scan(&id, &name); err != nil {
            log.Fatal(err)
        }
        fmt.Printf("User: %d - %s\n", id, name)
    }
}
```

Dùng pgx với `database/sql` interface (khi cần tương thích với thư viện dùng `*sql.DB`):

```go
import (
    "database/sql"
    "github.com/jackc/pgx/v5/stdlib"
)

// stdlib.OpenDB tạo *sql.DB backed bởi pgx driver
db := stdlib.OpenDB(*connConfig)
```

**pgxscan** — quét rows tự động như Dapper:

```go
import (
    "github.com/georgysavva/scany/v2/pgxscan"
)

var users []User
// Select tự động quét nhiều rows vào slice
err = pgxscan.Select(ctx, pool, &users, "SELECT id, name, email FROM users WHERE active = $1", true)

var user User
// Get tự động quét một row
err = pgxscan.Get(ctx, pool, &user, "SELECT id, name, email FROM users WHERE id = $1", 1)
```

---

## 12. sqlc — Generate Type-Safe Code Từ SQL

sqlc là công cụ đặc trưng của Go ecosystem: bạn viết SQL query trong file `.sql`, sqlc phân tích và generate Go code type-safe. Không cần ORM, không cần viết scan thủ công.

Workflow:
1. Định nghĩa schema database trong SQL
2. Viết queries trong file `.sql` với annotation đặc biệt
3. Chạy `sqlc generate` → nhận Go structs và functions tự động
4. Dùng generated code trong app

Cài đặt: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

File cấu hình `sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "schema/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        emit_json_tags: true
        emit_interface: true
```

File `schema/users.sql`:
```sql
CREATE TABLE users (
    id         SERIAL PRIMARY KEY,
    name       TEXT    NOT NULL,
    email      TEXT    NOT NULL UNIQUE,
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

File `queries/users.sql`:
```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: ListActiveUsers :many
SELECT * FROM users WHERE active = true ORDER BY name;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING *;

-- name: UpdateUserActive :exec
UPDATE users SET active = $2 WHERE id = $1;
```

Sau khi chạy `sqlc generate`, bạn nhận được code như thế này (tự động generated):
```go
// Code dưới đây được sqlc tự generate — bạn KHÔNG viết tay
type User struct {
    ID        int32
    Name      string
    Email     string
    Active    bool
    CreatedAt time.Time
}

func (q *Queries) GetUser(ctx context.Context, id int32) (User, error) {
    // ... implementation được generate ...
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
    // ... implementation được generate ...
}
```

Dùng generated code trong app:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jackc/pgx/v5"
    "myapp/internal/db" // generated package
)

func main() {
    ctx := context.Background()
    conn, _ := pgx.Connect(ctx, "postgres://postgres:secret@localhost/myapp")
    defer conn.Close(ctx)

    queries := db.New(conn)

    user, err := queries.CreateUser(ctx, db.CreateUserParams{
        Name:  "Alice",
        Email: "alice@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created user: %d\n", user.ID)

    users, err := queries.ListActiveUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, u := range users {
        fmt.Println(u.Name)
    }
}
```

**Ưu điểm của sqlc:** compile-time type safety (sai kiểu là lỗi compile, không phải runtime), không viết scan thủ công, SQL vẫn là SQL thật nên DBA có thể review, performance tốt vì không có reflection overhead.

---

## 13. Repository Pattern — Interface + Implementation

Repository pattern trong Go tương tự C# — interface định nghĩa contract, implementation chứa logic database cụ thể. Lợi ích: dễ mock trong test, dễ swap implementation (ví dụ đổi từ PostgreSQL sang in-memory).

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    _ "github.com/lib/pq"
)

// Domain types
type User struct {
    ID        int
    Name      string
    Email     string
    Active    bool
    CreatedAt time.Time
}

type CreateUserInput struct {
    Name  string
    Email string
}

// UserRepository interface — định nghĩa contract
// Tương đương IUserRepository trong C# + DI
type UserRepository interface {
    GetByID(ctx context.Context, id int) (User, error)
    GetByEmail(ctx context.Context, email string) (User, error)
    List(ctx context.Context, activeOnly bool) ([]User, error)
    Create(ctx context.Context, input CreateUserInput) (User, error)
    Deactivate(ctx context.Context, id int) error
}

// PostgresUserRepository implement UserRepository với PostgreSQL
type PostgresUserRepository struct {
    db *sql.DB
}

// NewPostgresUserRepository — constructor (tương đương DI registration)
func NewPostgresUserRepository(db *sql.DB) UserRepository {
    return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id int) (User, error) {
    var u User
    err := r.db.QueryRowContext(ctx,
        "SELECT id, name, email, active, created_at FROM users WHERE id = $1",
        id,
    ).Scan(&u.ID, &u.Name, &u.Email, &u.Active, &u.CreatedAt)

    if err == sql.ErrNoRows {
        return User{}, fmt.Errorf("user %d not found", id)
    }
    if err != nil {
        return User{}, fmt.Errorf("get user by id: %w", err)
    }
    return u, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (User, error) {
    var u User
    err := r.db.QueryRowContext(ctx,
        "SELECT id, name, email, active, created_at FROM users WHERE email = $1",
        email,
    ).Scan(&u.ID, &u.Name, &u.Email, &u.Active, &u.CreatedAt)

    if err == sql.ErrNoRows {
        return User{}, fmt.Errorf("user with email %s not found", email)
    }
    if err != nil {
        return User{}, fmt.Errorf("get user by email: %w", err)
    }
    return u, nil
}

func (r *PostgresUserRepository) List(ctx context.Context, activeOnly bool) ([]User, error) {
    query := "SELECT id, name, email, active, created_at FROM users"
    args := []any{}

    if activeOnly {
        query += " WHERE active = $1"
        args = append(args, true)
    }
    query += " ORDER BY name"

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("list users: %w", err)
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Active, &u.CreatedAt); err != nil {
            return nil, fmt.Errorf("scan user: %w", err)
        }
        users = append(users, u)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate users: %w", err)
    }

    return users, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, input CreateUserInput) (User, error) {
    var u User
    err := r.db.QueryRowContext(ctx,
        "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, active, created_at",
        input.Name, input.Email,
    ).Scan(&u.ID, &u.Name, &u.Email, &u.Active, &u.CreatedAt)
    if err != nil {
        return User{}, fmt.Errorf("create user: %w", err)
    }
    return u, nil
}

func (r *PostgresUserRepository) Deactivate(ctx context.Context, id int) error {
    result, err := r.db.ExecContext(ctx,
        "UPDATE users SET active = false WHERE id = $1",
        id,
    )
    if err != nil {
        return fmt.Errorf("deactivate user: %w", err)
    }
    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("user %d not found", id)
    }
    return nil
}

// UserService — business logic layer dùng repository
type UserService struct {
    users UserRepository
}

func NewUserService(users UserRepository) *UserService {
    return &UserService{users: users}
}

func (s *UserService) RegisterUser(ctx context.Context, name, email string) (User, error) {
    // Business logic: kiểm tra email tồn tại
    existing, err := s.users.GetByEmail(ctx, email)
    if err == nil && existing.ID > 0 {
        return User{}, fmt.Errorf("email %s already registered", email)
    }

    return s.users.Create(ctx, CreateUserInput{Name: name, Email: email})
}
```

---

## 14. Database Migrations — golang-migrate

Migration trong Go dùng `golang-migrate` tương đương EF Core migrations nhưng SQL thuần, không code-first.

Cài đặt CLI: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

Thêm vào go.mod:
```
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
```

Cấu trúc thư mục migrations:
```
migrations/
    000001_create_users.up.sql
    000001_create_users.down.sql
    000002_add_avatar_to_users.up.sql
    000002_add_avatar_to_users.down.sql
```

File `000001_create_users.up.sql`:
```sql
CREATE TABLE users (
    id         SERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    active     BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

File `000001_create_users.down.sql`:
```sql
DROP TABLE IF EXISTS users;
```

File `000002_add_avatar_to_users.up.sql`:
```sql
ALTER TABLE users ADD COLUMN avatar_url TEXT;
```

File `000002_add_avatar_to_users.down.sql`:
```sql
ALTER TABLE users DROP COLUMN avatar_url;
```

Chạy migration từ CLI:
```bash
# Up — chạy tất cả pending migrations
migrate -path migrations -database "postgres://postgres:secret@localhost/mydb?sslmode=disable" up

# Down — rollback 1 bước
migrate -path migrations -database "..." down 1

# Version hiện tại
migrate -path migrations -database "..." version
```

Chạy migration từ code Go (tự động migrate khi app start — pattern phổ biến):

```go
package main

import (
    "log"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(dbURL string) {
    m, err := migrate.New(
        "file://migrations", // thư mục chứa migration files
        dbURL,
    )
    if err != nil {
        log.Fatalf("creating migrator: %v", err)
    }
    defer m.Close()

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        log.Fatalf("running migrations: %v", err)
    }

    log.Println("Database migrations applied successfully")
}

func main() {
    runMigrations("postgres://postgres:secret@localhost/myapp?sslmode=disable")
    // ... rest of app
}
```

So sánh với EF Core: EF Core migrations generate C# code (`20231015_CreateUsers.cs`) và có thể rollback qua code. golang-migrate dùng SQL thuần — DBA có thể đọc, review, và chạy thủ công nếu cần. Không có "apply pending migrations" magic ẩn.

---

## 15. Testing với Database — testcontainers-go

Testing với database thật là integration test — không phải unit test. Để không phụ thuộc vào database server cố định, dùng `testcontainers-go` để spin up Docker container PostgreSQL trong test, chạy test, rồi xóa container.

Cài đặt: `go get github.com/testcontainers/testcontainers-go`

```go
package repository_test

import (
    "context"
    "database/sql"
    "fmt"
    "testing"
    "time"

    _ "github.com/lib/pq"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB khởi động PostgreSQL container và trả về *sql.DB
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    ctx := context.Background()

    // Spin up PostgreSQL container
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        t.Fatalf("start postgres container: %v", err)
    }

    // Cleanup: xóa container sau khi test xong
    t.Cleanup(func() {
        if err := pgContainer.Terminate(ctx); err != nil {
            t.Logf("terminate container: %v", err)
        }
    })

    // Lấy connection string
    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatalf("get connection string: %v", err)
    }

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        t.Fatalf("open db: %v", err)
    }

    // Apply schema
    _, err = db.ExecContext(ctx, `
        CREATE TABLE users (
            id         SERIAL PRIMARY KEY,
            name       TEXT    NOT NULL,
            email      TEXT    NOT NULL UNIQUE,
            active     BOOLEAN NOT NULL DEFAULT true,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )
    `)
    if err != nil {
        t.Fatalf("create schema: %v", err)
    }

    return db
}

func TestPostgresUserRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := NewPostgresUserRepository(db)
    ctx := context.Background()

    user, err := repo.Create(ctx, CreateUserInput{
        Name:  "Alice",
        Email: "alice@example.com",
    })
    if err != nil {
        t.Fatalf("Create() error: %v", err)
    }

    if user.ID == 0 {
        t.Error("expected non-zero ID")
    }
    if user.Name != "Alice" {
        t.Errorf("Name = %q, want %q", user.Name, "Alice")
    }
    if !user.Active {
        t.Error("expected user to be active by default")
    }
}

func TestPostgresUserRepository_GetByID_NotFound(t *testing.T) {
    db := setupTestDB(t)
    repo := NewPostgresUserRepository(db)
    ctx := context.Background()

    _, err := repo.GetByID(ctx, 99999)
    if err == nil {
        t.Error("expected error for non-existent user, got nil")
    }
}

func TestPostgresUserRepository_List_ActiveOnly(t *testing.T) {
    db := setupTestDB(t)
    repo := NewPostgresUserRepository(db)
    ctx := context.Background()

    // Seed data
    repo.Create(ctx, CreateUserInput{Name: "Alice", Email: "alice@example.com"})
    repo.Create(ctx, CreateUserInput{Name: "Bob", Email: "bob@example.com"})

    // Deactivate Bob
    users, _ := repo.List(ctx, false)
    for _, u := range users {
        if u.Name == "Bob" {
            repo.Deactivate(ctx, u.ID)
        }
    }

    // Chỉ lấy active users
    activeUsers, err := repo.List(ctx, true)
    if err != nil {
        t.Fatalf("List() error: %v", err)
    }
    if len(activeUsers) != 1 {
        t.Errorf("expected 1 active user, got %d", len(activeUsers))
    }
    if activeUsers[0].Name != "Alice" {
        t.Errorf("expected Alice, got %s", activeUsers[0].Name)
    }
}
```

---

## Bảng Common Issues

| Lỗi | Nguyên nhân | Cách fix |
|---|---|---|
| `pq: password authentication failed` | Sai credentials hoặc user không tồn tại | Kiểm tra lại DSN, đảm bảo Ping() trước khi dùng |
| `sql: no rows in result set` | `QueryRowContext().Scan()` không tìm thấy row | Check `err == sql.ErrNoRows` riêng biệt |
| Connection pool exhausted (app bị hang) | `MaxOpenConns` quá thấp hoặc quên `rows.Close()` | Luôn `defer rows.Close()`, tăng `MaxOpenConns` |
| `sql: expected N destination arguments in Scan, not M` | Số biến trong `Scan()` khác số cột trong SELECT | Đảm bảo SELECT columns và Scan args khớp nhau |
| Silent data loss khi đọc nhiều rows | Quên kiểm tra `rows.Err()` sau vòng lặp | Luôn check `rows.Err()` sau `for rows.Next()` |
| Transaction không rollback khi panic | Dùng `if err != nil { tx.Rollback() }` thủ công | Dùng `defer tx.Rollback()` ngay sau `BeginTx()` |
| SQL injection | Build query bằng `fmt.Sprintf` với user input | Luôn dùng placeholders `$1`, `$2`, ... |
| `pq: binary parameters are not supported` | Dùng `lib/pq` với kiểu binary phức tạp | Chuyển sang `pgx/v5` |
| `driver: bad connection` | Connection đã bị DB server đóng | Set `SetConnMaxLifetime` phù hợp |
| `context deadline exceeded` | Query chạy quá lâu, context timeout | Optimize query, tăng timeout nếu cần, thêm index |
| `pq: duplicate key value violates unique constraint` | Insert record trùng unique field | Handle error và return lỗi rõ ràng cho caller |
| `pgxscan: no column with name "X"` | Column name trong SQL khác với struct field tag | Thêm `db:"column_name"` tag vào struct field |

---

## Best Practices

**Singleton `*sql.DB`**: Tạo một `*sql.DB` duy nhất khi app khởi động và truyền nó xuống qua dependency injection. Không bao giờ tạo `sql.Open()` trong từng request handler — mỗi `Open()` tạo một pool mới, gây ra quá nhiều connections.

**Wrap errors với context**: Khi trả error từ database, luôn wrap thêm context bằng `fmt.Errorf("operation name: %w", err)`. Điều này giúp logging và debugging dễ hơn rất nhiều khi có lỗi production.

**Đặt timeout cho mọi query**: Không để query chạy vô thời hạn. Dùng `context.WithTimeout` hoặc dựa vào context của HTTP request (đã có timeout từ middleware). Query không có timeout có thể giữ connection mãi mãi và làm pool cạn.

**Prefer `RETURNING` thay vì `LastInsertId`**: Với PostgreSQL, `LastInsertId()` không hoạt động với `lib/pq`. Dùng `INSERT ... RETURNING id` và scan qua `QueryRowContext` để lấy ID vừa insert. Cách này cũng safer và không cần round-trip thứ hai.

**Không dùng `SELECT *` trong production code**: Luôn liệt kê columns cụ thể trong SELECT. `SELECT *` dễ gây lỗi khi schema thay đổi (thêm column mới), và trả về data không cần thiết làm tăng network overhead.

**Migration an toàn**: Mỗi migration nên backward compatible — tức là phiên bản code cũ vẫn chạy được với schema mới trong giai đoạn deploy rolling. Không drop column hay đổi kiểu column trong cùng migration với deploy code mới; làm hai bước riêng.

**Test với database thật**: Mock `UserRepository` interface cho unit test của service layer. Dùng testcontainers cho integration test của repository layer. Không dùng SQLite thay thế PostgreSQL trong test — behavior khác nhau (type casting, constraint enforcement, v.v.).

---

## Bài Tập Thực Hành

### Bài tập 1: Blog Post Repository

Xây dựng hệ thống quản lý bài viết blog với đầy đủ CRUD, transaction cho publish workflow, và proper error handling.

**Schema:**
```sql
CREATE TABLE authors (
    id         SERIAL PRIMARY KEY,
    username   TEXT NOT NULL UNIQUE,
    email      TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE posts (
    id           SERIAL PRIMARY KEY,
    author_id    INT  NOT NULL REFERENCES authors(id),
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    published    BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Yêu cầu:**
1. Implement `PostRepository` interface với: `Create`, `GetByID`, `ListByAuthor`, `Publish`
2. `Publish` phải dùng transaction: update `published = true`, set `published_at = NOW()`, insert vào `audit_log` table
3. Xử lý đúng `sql.ErrNoRows`
4. Tất cả methods đều nhận `context.Context`

**Giải pháp đầy đủ:**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// === Domain types ===

type Author struct {
	ID        int
	Username  string
	Email     string
	CreatedAt time.Time
}

type Post struct {
	ID          int
	AuthorID    int
	Title       string
	Content     string
	Published   bool
	PublishedAt sql.NullTime
	CreatedAt   time.Time
}

type CreatePostInput struct {
	AuthorID int
	Title    string
	Content  string
}

// === Repository Interface ===

type PostRepository interface {
	Create(ctx context.Context, input CreatePostInput) (Post, error)
	GetByID(ctx context.Context, id int) (Post, error)
	ListByAuthor(ctx context.Context, authorID int) ([]Post, error)
	Publish(ctx context.Context, postID int) error
}

// === PostgreSQL Implementation ===

type postgresPostRepository struct {
	db *sql.DB
}

func NewPostgresPostRepository(db *sql.DB) PostRepository {
	return &postgresPostRepository{db: db}
}

func (r *postgresPostRepository) Create(ctx context.Context, input CreatePostInput) (Post, error) {
	var p Post
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO posts (author_id, title, content) 
		 VALUES ($1, $2, $3) 
		 RETURNING id, author_id, title, content, published, published_at, created_at`,
		input.AuthorID, input.Title, input.Content,
	).Scan(&p.ID, &p.AuthorID, &p.Title, &p.Content, &p.Published, &p.PublishedAt, &p.CreatedAt)

	if err != nil {
		return Post{}, fmt.Errorf("create post: %w", err)
	}
	return p, nil
}

func (r *postgresPostRepository) GetByID(ctx context.Context, id int) (Post, error) {
	var p Post
	err := r.db.QueryRowContext(ctx,
		`SELECT id, author_id, title, content, published, published_at, created_at 
		 FROM posts WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.AuthorID, &p.Title, &p.Content, &p.Published, &p.PublishedAt, &p.CreatedAt)

	if err == sql.ErrNoRows {
		return Post{}, fmt.Errorf("post %d not found", id)
	}
	if err != nil {
		return Post{}, fmt.Errorf("get post %d: %w", id, err)
	}
	return p, nil
}

func (r *postgresPostRepository) ListByAuthor(ctx context.Context, authorID int) ([]Post, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, author_id, title, content, published, published_at, created_at 
		 FROM posts WHERE author_id = $1 ORDER BY created_at DESC`,
		authorID,
	)
	if err != nil {
		return nil, fmt.Errorf("list posts by author %d: %w", authorID, err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.AuthorID, &p.Title, &p.Content, &p.Published, &p.PublishedAt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}

	return posts, nil
}

func (r *postgresPostRepository) Publish(ctx context.Context, postID int) error {
	// Transaction: publish post + ghi audit log atomically
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() // no-op nếu Commit thành công

	// Kiểm tra post tồn tại và chưa publish
	var published bool
	err = tx.QueryRowContext(ctx,
		"SELECT published FROM posts WHERE id = $1 FOR UPDATE", // FOR UPDATE: lock row
		postID,
	).Scan(&published)
	if err == sql.ErrNoRows {
		return fmt.Errorf("post %d not found", postID)
	}
	if err != nil {
		return fmt.Errorf("check post status: %w", err)
	}
	if published {
		return fmt.Errorf("post %d already published", postID)
	}

	// Update post status
	if _, err := tx.ExecContext(ctx,
		"UPDATE posts SET published = true, published_at = NOW() WHERE id = $1",
		postID,
	); err != nil {
		return fmt.Errorf("publish post: %w", err)
	}

	// Ghi audit log (giả sử bảng audit_log tồn tại)
	// Trong demo này bỏ qua bảng audit, chỉ minh họa pattern transaction
	// if _, err := tx.ExecContext(ctx,
	// 	"INSERT INTO audit_log (action, entity_id) VALUES ('publish_post', $1)",
	// 	postID,
	// ); err != nil {
	// 	return fmt.Errorf("audit log: %w", err)
	// }

	return tx.Commit()
}

// === Main để demo ===

func setupSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS authors (
		id         SERIAL PRIMARY KEY,
		username   TEXT NOT NULL UNIQUE,
		email      TEXT NOT NULL UNIQUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS posts (
		id           SERIAL PRIMARY KEY,
		author_id    INT  NOT NULL REFERENCES authors(id),
		title        TEXT NOT NULL,
		content      TEXT NOT NULL,
		published    BOOLEAN NOT NULL DEFAULT false,
		published_at TIMESTAMPTZ,
		created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`
	_, err := db.Exec(schema)
	return err
}

func main() {
	dsn := "host=localhost port=5432 user=postgres password=secret dbname=blogdb sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}

	if err := setupSchema(db); err != nil {
		log.Fatalf("schema: %v", err)
	}

	repo := NewPostgresPostRepository(db)

	// Seed author
	var authorID int
	err = db.QueryRowContext(ctx,
		"INSERT INTO authors (username, email) VALUES ($1, $2) ON CONFLICT (email) DO UPDATE SET username=EXCLUDED.username RETURNING id",
		"alice", "alice@example.com",
	).Scan(&authorID)
	if err != nil {
		log.Fatalf("seed author: %v", err)
	}

	// Create post
	post, err := repo.Create(ctx, CreatePostInput{
		AuthorID: authorID,
		Title:    "Go Database Guide",
		Content:  "This is a comprehensive guide to database in Go...",
	})
	if err != nil {
		log.Fatalf("create: %v", err)
	}
	fmt.Printf("Created post ID=%d, Published=%v\n", post.ID, post.Published)

	// Publish
	if err := repo.Publish(ctx, post.ID); err != nil {
		log.Fatalf("publish: %v", err)
	}
	fmt.Println("Post published!")

	// Get by ID
	fetched, err := repo.GetByID(ctx, post.ID)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("Fetched: %q, Published=%v, PublishedAt=%v\n",
		fetched.Title, fetched.Published, fetched.PublishedAt.Time)

	// List by author
	posts, err := repo.ListByAuthor(ctx, authorID)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	fmt.Printf("Author has %d posts\n", len(posts))
}
```

---

### Bài tập 2: Product Inventory với Bulk Operations

Xây dựng inventory system hỗ trợ bulk insert và query với filtering.

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Product struct {
	ID        int
	SKU       string
	Name      string
	Price     float64
	Stock     int
	Category  string
	CreatedAt time.Time
}

type ProductFilter struct {
	Category  string  // lọc theo category, "" = tất cả
	MinPrice  float64 // 0 = không lọc
	MaxPrice  float64 // 0 = không lọc
	InStock   bool    // chỉ lấy hàng còn stock
}

type InventoryRepository interface {
	BulkCreate(ctx context.Context, products []Product) (int64, error)
	Search(ctx context.Context, filter ProductFilter) ([]Product, error)
	AdjustStock(ctx context.Context, sku string, delta int) error
}

type postgresInventoryRepo struct {
	db *sql.DB
}

func NewInventoryRepo(db *sql.DB) InventoryRepository {
	return &postgresInventoryRepo{db: db}
}

// BulkCreate dùng prepared statement để insert nhiều products hiệu quả
func (r *postgresInventoryRepo) BulkCreate(ctx context.Context, products []Product) (int64, error) {
	if len(products) == 0 {
		return 0, nil
	}

	stmt, err := r.db.PrepareContext(ctx,
		`INSERT INTO products (sku, name, price, stock, category) 
		 VALUES ($1, $2, $3, $4, $5) 
		 ON CONFLICT (sku) DO UPDATE SET name=EXCLUDED.name, price=EXCLUDED.price`,
	)
	if err != nil {
		return 0, fmt.Errorf("prepare bulk insert: %w", err)
	}
	defer stmt.Close()

	var totalAffected int64
	for _, p := range products {
		result, err := stmt.ExecContext(ctx, p.SKU, p.Name, p.Price, p.Stock, p.Category)
		if err != nil {
			return totalAffected, fmt.Errorf("insert product %s: %w", p.SKU, err)
		}
		n, _ := result.RowsAffected()
		totalAffected += n
	}

	return totalAffected, nil
}

// Search với dynamic WHERE clause — safe, không SQL injection
func (r *postgresInventoryRepo) Search(ctx context.Context, filter ProductFilter) ([]Product, error) {
	var conditions []string
	var args []any
	argIndex := 1 // PostgreSQL dùng $1, $2, ...

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, filter.Category)
		argIndex++
	}
	if filter.MinPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("price >= $%d", argIndex))
		args = append(args, filter.MinPrice)
		argIndex++
	}
	if filter.MaxPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("price <= $%d", argIndex))
		args = append(args, filter.MaxPrice)
		argIndex++
	}
	if filter.InStock {
		conditions = append(conditions, "stock > 0")
		// Không cần thêm arg vì không có placeholder
	}

	query := "SELECT id, sku, name, price, stock, category, created_at FROM products"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.Category, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}

	return products, nil
}

// AdjustStock dùng transaction với optimistic check
func (r *postgresInventoryRepo) AdjustStock(ctx context.Context, sku string, delta int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var currentStock int
	err = tx.QueryRowContext(ctx,
		"SELECT stock FROM products WHERE sku = $1 FOR UPDATE",
		sku,
	).Scan(&currentStock)
	if err == sql.ErrNoRows {
		return fmt.Errorf("product %s not found", sku)
	}
	if err != nil {
		return fmt.Errorf("get stock: %w", err)
	}

	newStock := currentStock + delta
	if newStock < 0 {
		return fmt.Errorf("insufficient stock: current=%d, adjustment=%d", currentStock, delta)
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE products SET stock = $1 WHERE sku = $2",
		newStock, sku,
	); err != nil {
		return fmt.Errorf("update stock: %w", err)
	}

	return tx.Commit()
}

func main() {
	dsn := "host=localhost port=5432 user=postgres password=secret dbname=inventorydb sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Setup schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id         SERIAL PRIMARY KEY,
			sku        TEXT    NOT NULL UNIQUE,
			name       TEXT    NOT NULL,
			price      NUMERIC(10,2) NOT NULL,
			stock      INT     NOT NULL DEFAULT 0,
			category   TEXT    NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("schema: %v", err)
	}

	ctx := context.Background()
	repo := NewInventoryRepo(db)

	// Bulk insert
	products := []Product{
		{SKU: "LAPTOP-001", Name: "Gaming Laptop", Price: 1299.99, Stock: 10, Category: "Electronics"},
		{SKU: "MOUSE-001", Name: "Wireless Mouse", Price: 29.99, Stock: 50, Category: "Electronics"},
		{SKU: "DESK-001", Name: "Standing Desk", Price: 599.99, Stock: 5, Category: "Furniture"},
		{SKU: "CHAIR-001", Name: "Ergonomic Chair", Price: 399.99, Stock: 0, Category: "Furniture"},
	}

	n, err := repo.BulkCreate(ctx, products)
	if err != nil {
		log.Fatalf("bulk create: %v", err)
	}
	fmt.Printf("Inserted/updated %d products\n", n)

	// Search với filter
	results, err := repo.Search(ctx, ProductFilter{
		Category: "Electronics",
		InStock:  true,
	})
	if err != nil {
		log.Fatalf("search: %v", err)
	}
	fmt.Printf("\nElectronics in stock (%d items):\n", len(results))
	for _, p := range results {
		fmt.Printf("  [%s] %s — $%.2f (stock: %d)\n", p.SKU, p.Name, p.Price, p.Stock)
	}

	// Adjust stock
	if err := repo.AdjustStock(ctx, "MOUSE-001", -5); err != nil {
		log.Fatalf("adjust stock: %v", err)
	}
	fmt.Println("\nStock adjusted for MOUSE-001 (-5)")

	// Try to over-deduct (should fail)
	err = repo.AdjustStock(ctx, "CHAIR-001", -1)
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}
}
```

---

## Tổng kết

Go approach với database có vẻ verbose hơn C# (EF Core với LINQ) nhưng đổi lại bạn có sự minh bạch hoàn toàn: bạn biết chính xác SQL nào được thực thi, biết khi nào connection được dùng, và không có magic N+1 query ẩn. Trong production, điều này dẫn đến app dễ debug, dễ optimize hơn.

Key takeaways: `*sql.DB` là pool không phải connection; luôn `defer rows.Close()` và check `rows.Err()`; luôn dùng context; dùng placeholder thay vì string formatting; `defer tx.Rollback()` ngay sau `BeginTx()`; prefer `pgx/v5` cho PostgreSQL; dùng sqlc khi muốn type-safety mà không muốn ORM.

---

**Tiếp theo:** [15. Advanced Concurrency Patterns](15-concurrency-patterns.md) — Worker pools, fan-out/fan-in, `errgroup`, pipeline pattern.
