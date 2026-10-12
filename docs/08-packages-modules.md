# 📦 Packages & Modules — Tổ chức code Go (dành cho C# developer)

---

## 1. Package vs Module — Tại sao Go tách hai khái niệm này?

Khi mới học Go, nhiều C# developer bị nhầm lẫn giữa **package** và **module** vì Go dùng hai từ này với ý nghĩa cụ thể và tách biệt. Trong C#, bạn quen với `namespace` (để tổ chức code) và `project`/`solution` (để tổ chức file). Go có hệ thống tương tự nhưng với triết lý khác.

**Package** trong Go là một **thư mục chứa các file `.go`**. Tất cả các file trong cùng một thư mục phải khai báo cùng một `package name` ở đầu file. Package là đơn vị **compilation** và **visibility** — mọi thứ không được export (chữ thường) chỉ nhìn thấy được trong cùng package. Đây tương đương với `namespace` trong C#, nhưng có thêm tính chất **encapsulation** mà C# chỉ có thể mô phỏng qua `internal`.

**Module** trong Go là một **tập hợp các package** có liên quan, được định nghĩa bởi một file `go.mod` ở root directory. Module là đơn vị **versioning** và **distribution** — khi bạn publish một thư viện lên GitHub, cả module đó được version cùng nhau. Đây tương đương với `project` hoặc cả `solution` trong C#, và tương đương `NuGet package` khi nói về distribution.

Cấu trúc điển hình trông như sau:

```
myapp/                    ← module root (có go.mod)
├── go.mod
├── go.sum
├── main.go               ← package main
├── config/
│   └── config.go         ← package config
├── internal/
│   └── database/
│       └── db.go         ← package database (chỉ dùng nội bộ)
└── api/
    ├── handler.go        ← package api
    └── middleware.go     ← package api (cùng thư mục = cùng package)
```

So sánh trực tiếp:

| Go | C# |
|---|---|
| `package` (thư mục) | `namespace` |
| `module` (go.mod) | `project` (.csproj) |
| Module trên GitHub/pkg.go.dev | NuGet package |
| `go get` | `dotnet add package` |

Lý do Go tách biệt package và module như vậy là vì Go muốn **mỗi thư mục là một đơn vị độc lập**, không cần file project phức tạp như `.csproj`. File `go.mod` chỉ cần ở root của module, còn các thư mục con tự động trở thành package dựa trên vị trí của chúng.

---

## 2. File `go.mod` — Trái tim của module

File `go.mod` là thứ Go dùng thay cho `.csproj` hoặc `packages.config` trong C#. Khi bạn chạy `go mod init`, Go tạo file này. Bạn hiếm khi cần sửa tay — các lệnh `go get` và `go mod tidy` sẽ tự cập nhật.

Một file `go.mod` thực tế trông như sau:

```
module github.com/yourname/myapp

go 1.22

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/lib/pq v1.10.9
    go.uber.org/zap v1.27.0
)

require (
    // indirect: dependencies của dependencies (transitive)
    github.com/bytedance/sonic v1.11.3 // indirect
    github.com/gabriel-vasile/mimetype v1.4.3 // indirect
    golang.org/x/net v0.23.0 // indirect
)
```

**`module github.com/yourname/myapp`** — Đây là **module path**, đồng thời là **import path prefix** cho tất cả các package trong module của bạn. Khi code trong module này import package nội bộ, bạn viết `import "github.com/yourname/myapp/config"`. Không phải đường dẫn tương đối, mà là đường dẫn module đầy đủ — đây là điểm khác biệt lớn so với C# (không cần `using` với đường dẫn tuyệt đối như vậy).

**`go 1.22`** — Version Go tối thiểu để build module này. Khác với C# (target framework ảnh hưởng đến API có sẵn), trong Go đây chủ yếu ảnh hưởng đến **language features** được dùng. Luôn giữ version này gần với version Go bạn đang dùng trong production.

**`require`** block — Danh sách dependencies kèm version cụ thể, giống `PackageReference` trong `.csproj`. Go **pin exact version** theo mặc định (Minimum Version Selection — MVS), khác với NuGet có thể resolve lên version mới hơn. Đây là lý do Go builds **reproducible** hơn.

Để khởi tạo module mới:

```bash
# Tạo module mới — tương đương dotnet new + đặt tên project
go mod init github.com/yourname/myapp
```

---

## 3. File `go.sum` — Lock file của Go

Nếu bạn quen với `package-lock.json` (npm) hoặc `packages.lock.json` (NuGet), thì `go.sum` có chức năng tương tự nhưng **mạnh hơn**. File này chứa **cryptographic hash** của từng dependency và từng version được download, đảm bảo rằng code bạn download hôm nay giống hệt code team member khác download tuần trước.

```
github.com/gin-gonic/gin v1.9.1 h1:4idEAncQnU5cB7BeOkPtxjfCSye0AAm1R0RVIqJ+Jmg=
github.com/gin-gonic/gin v1.9.1/go.mod h1:hPrL7YrpYKXt5YId3A/Tnip5kqbEAP+KLuI3SUcPTeU=
```

Quy tắc quan trọng: **không bao giờ sửa tay file `go.sum`**. File này được `go` tool tự động quản lý. Nếu file này bị corrupt hoặc mất, bạn chạy `go mod tidy` để tái tạo. Luôn **commit cả `go.mod` lẫn `go.sum`** vào git — đây là requirement, không phải tùy chọn. Thiếu `go.sum` trong repository nghĩa là builds không reproducible và đây là lỗi thường gặp khi team mới bắt đầu với Go.

---

## 4. `go get` — Thêm và cập nhật dependencies

`go get` là tương đương của `dotnet add package` trong .NET. Cú pháp cơ bản:

```bash
# Thêm dependency mới với version mới nhất
go get github.com/gin-gonic/gin

# Thêm với version cụ thể
go get github.com/gin-gonic/gin@v1.9.1

# Cập nhật lên version minor/patch mới nhất
go get github.com/gin-gonic/gin@latest

# Cập nhật tất cả dependencies (cẩn thận trong production)
go get -u ./...

# Xem tất cả dependencies có thể update
go list -u -m all
```

Một điểm khác biệt quan trọng so với NuGet: `go get` trong các phiên bản Go mới (1.17+) **chỉ update go.mod**, không tự động download code. Code được download khi bạn build hoặc chạy `go mod download`. Đây là thiết kế cố ý để tách biệt dependency declaration khỏi download.

---

## 5. `go mod tidy` — Dọn dẹp dependencies

Đây là lệnh bạn sẽ chạy thường xuyên nhất sau khi thay đổi code. `go mod tidy` làm hai việc: **thêm** các dependencies bị thiếu (mà code đang dùng nhưng chưa có trong `go.mod`) và **xóa** các dependencies không còn dùng nữa. Tương đương không hoàn toàn là `dotnet restore` kết hợp với tool dọn dẹp package thừa.

```bash
# Chạy sau khi thêm/xóa import trong code
go mod tidy

# Kiểm tra go.mod có sạch không (dùng trong CI/CD)
go mod tidy && git diff --exit-code go.mod go.sum
```

Workflow chuẩn khi làm việc với dependencies trong Go:

Bước 1: Thêm `import` vào code như bình thường (hoặc xóa import không dùng).
Bước 2: Chạy `go mod tidy` để sync `go.mod` và `go.sum` với code thực tế.
Bước 3: Commit cả code lẫn `go.mod` và `go.sum` cùng nhau.

Nếu bạn quên chạy `go mod tidy` trước khi commit, CI sẽ fail với lỗi về `go.mod` không đồng bộ — đây là lỗi phổ biến trong team mới làm quen Go.

---

## 6. Semantic Versioning và v2 modules

Go tuân thủ [Semantic Versioning](https://semver.org/) một cách nghiêm ngặt hơn hầu hết hệ sinh thái khác. Version có dạng `v{MAJOR}.{MINOR}.{PATCH}`:

- **PATCH** (v1.0.**1**): Bug fix, không thay đổi API — update an toàn.
- **MINOR** (v1.**1**.0): Thêm feature mới, backward compatible — update an toàn.
- **MAJOR** (**v2**.0.0): **Breaking change** — đây là điểm Go xử lý khác hoàn toàn.

Khi một module release **v2.0.0** (major version bump), Go yêu cầu **module path phải thay đổi** bằng cách thêm `/v2` vào cuối. Đây là quy tắc **Import Compatibility Rule** — nếu import path giống nhau thì phải backward compatible.

```go
// v1.x.x — import path bình thường
import "github.com/some/library"

// v2.x.x — phải có /v2 trong import path
import "github.com/some/library/v2"
```

Điều này trông kỳ lạ với C# developer, nhưng lý do rất thực tế: Go cho phép bạn **dùng v1 và v2 cùng lúc trong một project** mà không có conflict. Trong C#, upgrade NuGet package lên major version mới thường là all-or-nothing. Go giải quyết vấn đề này bằng cách encode version vào import path.

```go
// Hoàn toàn hợp lệ trong Go — dùng cả hai version cùng lúc
import (
    libv1 "github.com/some/library"
    libv2 "github.com/some/library/v2"
)
```

Khi bạn tạo một module và chuẩn bị release v2, bạn phải cập nhật `go.mod`:

```
// go.mod cho v2
module github.com/yourname/myapp/v2

go 1.22
```

---

## 7. `internal/` package — Encapsulation mạnh hơn C#

Go có một cơ chế encapsulation đặc biệt không có trong C#: thư mục `internal/`. Bất kỳ package nào nằm trong hoặc dưới một thư mục `internal/` chỉ có thể được import bởi code trong **cùng module** (hoặc cụ thể hơn: code trong parent directory của `internal/` đó).

Trong C#, `internal` là access modifier và chỉ giới hạn ở assembly level. Go's `internal/` mạnh hơn vì nó **enforced bởi compiler** và có phạm vi rõ ràng theo cấu trúc thư mục.

```
myapp/
├── go.mod
├── main.go
├── internal/
│   ├── database/
│   │   └── db.go         ← KHÔNG thể import từ module khác
│   └── validator/
│       └── validate.go   ← KHÔNG thể import từ module khác
└── api/
    └── handler.go        ← CÓ THỂ import internal/database
```

```go
// Trong api/handler.go — OK: cùng module
import "github.com/yourname/myapp/internal/database"

// Trong một module KHÁC — COMPILE ERROR
// import "github.com/yourname/myapp/internal/database"
// Error: use of internal package not allowed
```

Dùng `internal/` cho implementation details bạn không muốn người dùng thư viện của mình phụ thuộc vào — database queries, helper functions, business logic nội bộ. Đây là cách Go khuyến khích viết thư viện có **stable public API** và **flexible internal implementation**.

---

## 8. Hàm `init()` — Chạy trước `main()`

Mỗi package trong Go có thể có một hoặc nhiều hàm `init()` — chạy tự động khi package được load, **trước** khi `main()` chạy. Không có equivalent trực tiếp trong C#, gần nhất là static constructors (`static ClassName()`) nhưng behavior khác.

```go
// Trong một package
package config

import "fmt"

var DefaultTimeout int

func init() {
    DefaultTimeout = 30
    fmt.Println("config package initialized")
}
```

**Thứ tự thực thi** quan trọng: Go đảm bảo package dependencies được init trước, rồi mới đến package phụ thuộc vào chúng. Trong một package, các hàm `init()` chạy theo thứ tự xuất hiện trong file, và file được xử lý theo thứ tự alphabetical của tên file.

```go
// main.go
package main

import (
    _ "myapp/config"  // init() của config chạy TRƯỚC main()
    "fmt"
)

func main() {
    fmt.Println("main chạy sau init()")
}
// Output:
// config package initialized
// main chạy sau init()
```

**Anti-pattern cần tránh:** Dùng `init()` cho logic có thể fail hoặc có side effects khó debug. Vì `init()` không thể trả về error, bạn buộc phải `panic()` nếu có lỗi — và panic trong init rất khó debug hơn lỗi thông thường.

```go
// BAD — init() với logic phức tạp có thể fail
func init() {
    db, err := sql.Open("postgres", os.Getenv("DB_URL"))
    if err != nil {
        panic(err) // Khó debug, không thể recover gracefully
    }
    globalDB = db
}

// GOOD — dùng hàm khởi tạo tường minh
func NewDB(connStr string) (*sql.DB, error) {
    return sql.Open("postgres", connStr)
}
// Gọi từ main() và xử lý error bình thường
```

Dùng `init()` chủ yếu cho: đăng ký drivers (xem phần tiếp theo), thiết lập giá trị mặc định đơn giản, và những thứ **không thể fail**.

---

## 9. Blank import `_` — Import chỉ để chạy `init()`

Đây là một trong những cú pháp trông kỳ lạ nhất với C# developer. Khi bạn viết `import _ "some/package"`, bạn đang import package đó **chỉ để trigger hàm `init()` của nó**, không dùng bất kỳ symbol nào từ package đó.

Ví dụ điển hình nhất là database drivers trong Go. Package `database/sql` định nghĩa interface, còn các driver (postgres, mysql, sqlite) được **register** vào `database/sql` thông qua `init()`. Bạn phải import driver để nó được register:

```go
// BAD — quên import driver
package main

import (
    "database/sql"
    "fmt"
)

func main() {
    // RUNTIME ERROR: unknown driver "postgres"
    db, err := sql.Open("postgres", "postgresql://localhost/mydb")
    if err != nil {
        fmt.Println(err)
    }
    _ = db
}
```

```go
// GOOD — blank import để register driver
package main

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq" // init() của pq gọi sql.Register("postgres", ...)
)

func main() {
    // OK — driver đã được register bởi init() của pq
    db, err := sql.Open("postgres", "postgresql://localhost/mydb")
    if err != nil {
        fmt.Println(err)
    }
    _ = db
}
```

Bên trong `github.com/lib/pq`, có code như thế này:

```go
// Đây là code bên trong package pq (bạn không viết cái này)
func init() {
    sql.Register("postgres", &Driver{})
}
```

Ngoài database drivers, blank import còn dùng cho: image format decoders (`image/png`, `image/jpeg`), testing frameworks, và bất kỳ pattern nào cần "register" handler vào global registry.

Quan trọng: **hạn chế dùng blank import** ngoài các trường hợp chuẩn như trên. Nếu bạn thấy mình cần blank import cho code của chính mình, đó thường là dấu hiệu của design có vấn đề.

---

## 10. Package Naming — Ngắn, rõ, không phải "utils"

Naming convention trong Go khác hẳn C#. Go ưu tiên tên package **ngắn, lowercase, không có underscore, không có camelCase**, và phải là **danh từ số ít có nghĩa cụ thể**.

Lý do Go nghiêm khắc về điều này: tên package xuất hiện trong mọi lần bạn dùng nó — `http.Get()`, `json.Marshal()`, `fmt.Println()`. Tên dài hoặc vô nghĩa làm code khó đọc hơn.

```go
// BAD — tên package sai style Go
package utils         // Quá mơ hồ — utils là gì?
package helpers       // Tương tự, không rõ nghĩa
package common        // Không có nghĩa gì
package myHelpers     // camelCase không dùng trong package name
package string_utils  // underscore không dùng trong package name
package Validator     // PascalCase không dùng trong package name
```

```go
// GOOD — tên package đúng style Go
package validator     // Rõ ràng: package này validate thứ gì đó
package auth          // Ngắn, rõ
package middleware    // Một từ, mô tả chức năng
package postgres      // Tên cụ thể của implementation
package config        // Rõ ràng
```

Một quy tắc nữa: **tên hàm trong package không nên lặp lại tên package**. Nếu package là `validator`, hàm nên là `validator.Check()` không phải `validator.ValidateCheck()`. Khi dùng, bạn viết `validator.Check(input)` — rõ ràng mà không dư thừa.

```go
// BAD — tên hàm lặp lại tên package
package validator

func ValidateEmail(s string) bool { ... }    // Dùng: validator.ValidateEmail()
func ValidatePassword(s string) bool { ... } // Dùng: validator.ValidatePassword()
```

```go
// GOOD — tên hàm bổ sung cho tên package, không lặp lại
package validator

func Email(s string) bool { ... }    // Dùng: validator.Email()
func Password(s string) bool { ... } // Dùng: validator.Password()
```

Thay vì tạo package `utils` chứa mọi thứ lẫn lộn, hãy tách ra các package có tên cụ thể: `timeutil`, `httputil`, `strconv` — hoặc tốt hơn là đặt tên theo domain business: `invoice`, `payment`, `notification`.

---

## 11. Circular Imports — Go cấm hoàn toàn

Go **không cho phép circular imports** ở compiler level. Nếu package A import package B, và package B cũng import package A, Go compiler sẽ báo lỗi ngay lập tức. Trong C#, circular references trong cùng assembly cũng bị cấm, nhưng bạn ít gặp hơn vì namespace không enforce điều này.

```
// Cấu trúc gây circular import — sẽ compile error
api/
  handler.go     imports "myapp/service"
service/
  service.go     imports "myapp/api"  ← CIRCULAR! Compiler báo lỗi
```

Lỗi bạn sẽ thấy:

```
import cycle not allowed:
    myapp/api
    myapp/service
    myapp/api
```

**Cách fix:** Tách phần bị chia sẻ ra thành một package thứ ba mà cả hai đều import, hoặc dùng interface để đảo ngược dependency.

```
// Cách 1: Package chia sẻ (shared types)
types/
  types.go     ← Định nghĩa shared interfaces và structs
api/
  handler.go   imports "myapp/types"
service/
  service.go   imports "myapp/types"
// api và service không import nhau nữa

// Cách 2: Dependency Inversion — api định nghĩa interface
// service implement interface đó mà không cần biết về api
```

Sự cứng nhắc này của Go thực ra là **tính năng** chứ không phải hạn chế. Nó buộc bạn phải thiết kế dependency graph rõ ràng ngay từ đầu, tránh được spaghetti code mà nhiều C# project lớn hay gặp.

---

## 12. `go work` — Workspace Mode cho Multi-Module Development

Tình huống phổ biến: bạn đang phát triển cả `myapp` lẫn thư viện `mylib` mà `myapp` phụ thuộc vào. Bạn muốn test thay đổi trong `mylib` với `myapp` mà không cần publish `mylib` lên GitHub mỗi lần. Đây là lúc `go work` (workspace mode, introduced Go 1.18) hữu ích.

```bash
# Cấu trúc thư mục
workspace/
├── myapp/
│   ├── go.mod      (module github.com/yourname/myapp)
│   └── main.go
└── mylib/
    ├── go.mod      (module github.com/yourname/mylib)
    └── lib.go

# Tạo workspace ở root
cd workspace
go work init ./myapp ./mylib
# Tạo ra file go.work
```

File `go.work` trông như sau:

```
go 1.22

use (
    ./myapp
    ./mylib
)
```

Với `go.work` tại chỗ, khi build `myapp`, Go tự động dùng local copy của `mylib` thay vì version trên GitHub. Bạn không cần sửa `go.mod` của `myapp`.

**Khi nào dùng `go work`:** Phát triển multi-module trong local environment, monorepo với nhiều module phụ thuộc nhau. **Khi nào không dùng:** Trong CI/CD — hãy dùng `replace` directive thay thế (xem phần tiếp theo). **Quan trọng:** Không commit file `go.work` vào repository trong hầu hết trường hợp — thêm `go.work` và `go.work.sum` vào `.gitignore`.

---

## 13. Vendoring — Offline Builds cho Enterprise

Vendoring là copy tất cả dependencies vào thư mục `vendor/` trong project của bạn. Thay vì download dependencies từ internet khi build, Go dùng code local trong `vendor/`.

```bash
# Tạo vendor directory
go mod vendor

# Build sử dụng vendor (Go tự động dùng vendor/ nếu tồn tại)
go build ./...

# Hoặc explicit
go build -mod=vendor ./...
```

Sau khi chạy `go mod vendor`, cấu trúc trông như:

```
myapp/
├── go.mod
├── go.sum
├── vendor/
│   ├── modules.txt
│   ├── github.com/gin-gonic/gin/
│   └── github.com/lib/pq/
└── main.go
```

**Khi nào dùng vendoring:**
- Môi trường build không có internet access (air-gapped enterprise environments).
- Cần guarantee 100% reproducible builds kể cả khi upstream repository bị xóa.
- Audit security — bạn muốn kiểm soát hoàn toàn code nào vào production.

**Khi nào không nên dùng:** Project cá nhân hoặc startup — `go.sum` đã đủ để reproducible builds. Vendoring tăng kích thước repository đáng kể.

**Quy trình vendor:** Sau khi thêm/cập nhật dependency, chạy `go mod tidy` trước rồi mới chạy `go mod vendor`. Commit cả thư mục `vendor/` vào git.

---

## 14. Replace Directive — Dùng Local Copy của Dependency

`replace` directive trong `go.mod` cho phép bạn redirect một module path sang một location khác — có thể là local directory hoặc một repository khác. Đây là giải pháp thay thế cho `go work` khi bạn cần commit vào repository.

```
// go.mod
module github.com/yourname/myapp

go 1.22

require (
    github.com/yourname/mylib v0.1.0
)

replace (
    // Dùng local copy trong quá trình phát triển
    github.com/yourname/mylib => ../mylib

    // Hoặc redirect sang fork của bạn
    github.com/original/library => github.com/yourname/library v1.2.3-patched

    // Hoặc fix security vulnerability khi chờ upstream patch
    golang.org/x/crypto v0.21.0 => golang.org/x/crypto v0.22.0
)
```

**Lưu ý quan trọng:** `replace` directive ảnh hưởng đến **toàn bộ module build**, không chỉ package của bạn. Nếu thư viện của bạn có `replace` directive và người khác dùng thư viện đó, `replace` directive của bạn sẽ **không** có hiệu lực trong build của họ (Go chỉ áp dụng `replace` của main module). Điều này có nghĩa là: không dùng `replace` directive với local paths trong thư viện bạn publish — chỉ dùng trong applications/main modules.

---

## Common Issues

| Lỗi | Nguyên nhân | Cách fix |
|---|---|---|
| `cannot find module providing package X` | Package chưa có trong `go.mod` | Chạy `go get X` hoặc `go mod tidy` |
| `import cycle not allowed` | Circular import giữa các packages | Tạo package thứ ba cho shared types, hoặc dùng interface |
| `go.mod file not found` | Chạy go command ngoài module directory | Chạy `go mod init <name>` trước |
| `use of internal package not allowed` | Import `internal/` từ module ngoài | Design lại, expose API qua public package |
| `verifying module: checksum mismatch` | `go.sum` không khớp với file download | Xóa `go.sum`, chạy `go mod tidy` |
| Module không update sau `go get` | Đang trong `vendor/` mode | Chạy `go mod vendor` lại sau update |
| `ambiguous import` | Hai module khác nhau cung cấp cùng package path | Kiểm tra `go.mod`, dùng `replace` directive |
| Build chậm trong CI | Download dependency mỗi lần | Cache `GOMODCACHE` trong CI hoặc dùng vendoring |
| `init()` gây panic khó debug | Logic phức tạp trong `init()` | Chuyển sang hàm khởi tạo tường minh |

---

## Best Practices

Hãy luôn chạy `go mod tidy` trước khi commit. Trong team, nếu `go.mod` và `go.sum` không được commit đồng thời với code thay đổi dependencies, CI sẽ fail và mọi người trong team sẽ gặp lỗi khi pull code. Đây là nguồn gốc của nhiều "tại sao build của tao bị lỗi mà của mày không" trong team mới dùng Go.

Cấu trúc project theo domain, không theo technical layer. Thay vì `models/`, `controllers/`, `services/` (kiểu MVC C#), hãy nghĩ đến `user/`, `payment/`, `notification/`. Package theo domain tự mô tả business logic và tránh được các package "utils" vô nghĩa.

Dùng `internal/` một cách chủ động cho implementation details. Bất cứ thứ gì bạn không chắc có muốn expose như public API không, hãy đặt vào `internal/`. Dễ dàng chuyển từ internal sang public hơn là ngược lại (breaking change).

Không tạo package `utils`, `helpers`, hay `common`. Nếu thấy mình định tạo những package này, đó là dấu hiệu code cần được refactor. Hỏi: "Cái này thực sự làm gì?" và đặt tên theo chức năng cụ thể đó.

Với thư viện sẽ publish: lên kế hoạch cho v2 ngay từ đầu. Nếu API của bạn có thể cần breaking change trong tương lai, thiết kế để interface đủ stable, và khi cần breaking change, đừng ngại tạo v2 với `/v2` trong module path.

Trong CI/CD pipeline, luôn kiểm tra `go mod tidy` không tạo ra diff: `go mod tidy && git diff --exit-code go.mod go.sum`. Bước này ngăn developer quên sync `go.mod` sau khi thay đổi imports.

---

## Bài tập thực hành

### Bài tập 1: Xây dựng multi-package calculator module

Tạo một module với cấu trúc package hợp lý, bao gồm internal package và public API.

**Yêu cầu:**
- Module `github.com/yourname/calculator`
- Package `math` với các phép tính cơ bản (public API)
- Package `internal/validate` để validate input (chỉ dùng nội bộ)
- Package `main` để demo

**Solution:**

Tạo cấu trúc thư mục và chạy `go mod init github.com/yourname/calculator` trước.

```
calculator/
├── go.mod
├── main.go
├── math/
│   └── math.go
└── internal/
    └── validate/
        └── validate.go
```

```go
// internal/validate/validate.go
package validate

import "errors"

// IsValidDivisor kiểm tra số chia hợp lệ
// Hàm này KHÔNG thể dùng từ module bên ngoài — chỉ dùng nội bộ
func IsValidDivisor(b float64) error {
    if b == 0 {
        return errors.New("chia cho 0 không hợp lệ")
    }
    return nil
}
```

```go
// math/math.go
package math

import (
    "github.com/yourname/calculator/internal/validate"
)

// Add cộng hai số
func Add(a, b float64) float64 {
    return a + b
}

// Subtract trừ hai số
func Subtract(a, b float64) float64 {
    return a - b
}

// Multiply nhân hai số
func Multiply(a, b float64) float64 {
    return a * b
}

// Divide chia hai số — trả về error nếu chia cho 0
// Dùng internal/validate để kiểm tra input
func Divide(a, b float64) (float64, error) {
    if err := validate.IsValidDivisor(b); err != nil {
        return 0, err
    }
    return a / b, nil
}
```

```go
// main.go
package main

import (
    "fmt"
    // Dùng public package math — OK
    "github.com/yourname/calculator/math"
    // Nếu uncomment dòng dưới sẽ COMPILE ERROR:
    // _ "github.com/yourname/calculator/internal/validate"
)

func main() {
    fmt.Println("=== Calculator Demo ===")

    fmt.Printf("Add(10, 5) = %.2f\n", math.Add(10, 5))
    fmt.Printf("Subtract(10, 5) = %.2f\n", math.Subtract(10, 5))
    fmt.Printf("Multiply(10, 5) = %.2f\n", math.Multiply(10, 5))

    result, err := math.Divide(10, 5)
    if err != nil {
        fmt.Printf("Divide error: %v\n", err)
    } else {
        fmt.Printf("Divide(10, 5) = %.2f\n", result)
    }

    // Thử chia cho 0
    _, err = math.Divide(10, 0)
    if err != nil {
        fmt.Printf("Divide(10, 0) error: %v\n", err) // chia cho 0 không hợp lệ
    }
}
```

```
// go.mod
module github.com/yourname/calculator

go 1.22
```

Chạy: `go run main.go`

Output:
```
=== Calculator Demo ===
Add(10, 5) = 10.00
Subtract(10, 5) = 5.00
Multiply(10, 5) = 50.00
Divide(10, 5) = 2.00
Divide(10, 0) error: chia cho 0 không hợp lệ
```

---

### Bài tập 2: Module với external dependency và blank import

Tạo một ứng dụng nhỏ dùng external dependency (`github.com/google/uuid`) và mô phỏng blank import pattern.

**Yêu cầu:**
- Dùng `go get` để thêm dependency
- Tạo package `registry` simulate driver registration pattern
- Package `store` dùng registry thông qua blank import

```
myapp/
├── go.mod
├── go.sum
├── main.go
├── registry/
│   └── registry.go
├── store/
│   ├── store.go
│   └── memory/
│       └── memory.go
```

```bash
# Bước 1: Khởi tạo
go mod init github.com/yourname/myapp
go get github.com/google/uuid
```

```go
// registry/registry.go
package registry

import "fmt"

// Store là interface cho storage backends
type Store interface {
    Save(key, value string) error
    Get(key string) (string, error)
}

// stores chứa tất cả registered backends
var stores = map[string]Store{}

// Register đăng ký một Store implementation với tên
// Được gọi từ init() của các implementation packages
func Register(name string, s Store) {
    if _, exists := stores[name]; exists {
        panic("store đã được register: " + name)
    }
    stores[name] = s
    fmt.Printf("[registry] Registered store: %s\n", name)
}

// Get lấy Store đã register
func Get(name string) (Store, bool) {
    s, ok := stores[name]
    return s, ok
}
```

```go
// store/memory/memory.go
package memory

import (
    "fmt"
    "github.com/yourname/myapp/registry"
)

// MemoryStore là in-memory implementation
type MemoryStore struct {
    data map[string]string
}

func (m *MemoryStore) Save(key, value string) error {
    m.data[key] = value
    return nil
}

func (m *MemoryStore) Get(key string) (string, error) {
    v, ok := m.data[key]
    if !ok {
        return "", fmt.Errorf("key không tồn tại: %s", key)
    }
    return v, nil
}

// init() tự động register khi package được import
// Pattern này giống hệt github.com/lib/pq register postgres driver
func init() {
    registry.Register("memory", &MemoryStore{data: map[string]string{}})
}
```

```go
// main.go
package main

import (
    "fmt"

    "github.com/google/uuid"
    "github.com/yourname/myapp/registry"

    // Blank import — chỉ để trigger init() của memory package
    // Không dùng bất kỳ symbol nào từ memory package trực tiếp
    _ "github.com/yourname/myapp/store/memory"
)

func main() {
    // Tạo ID unique dùng external dependency
    id := uuid.New()
    fmt.Printf("Generated ID: %s\n", id)

    // Lấy store đã được register bởi init() của memory package
    store, ok := registry.Get("memory")
    if !ok {
        fmt.Println("ERROR: memory store chưa được register")
        return
    }

    // Dùng store
    key := id.String()
    err := store.Save(key, "Hello from Go!")
    if err != nil {
        fmt.Printf("Save error: %v\n", err)
        return
    }

    value, err := store.Get(key)
    if err != nil {
        fmt.Printf("Get error: %v\n", err)
        return
    }

    fmt.Printf("Saved and retrieved: %s\n", value)

    // Thử key không tồn tại
    _, err = store.Get("nonexistent-key")
    if err != nil {
        fmt.Printf("Expected error: %v\n", err)
    }
}
```

Chạy:
```bash
go mod tidy   # Sync go.mod và go.sum
go run main.go
```

Output:
```
[registry] Registered store: memory
Generated ID: 550e8400-e29b-41d4-a716-446655440000
Saved and retrieved: Hello from Go!
Expected error: key không tồn tại: nonexistent-key
```

Lưu ý thứ tự output: `[registry] Registered store: memory` xuất hiện **trước** bất kỳ thứ gì trong `main()` — đây là bằng chứng `init()` chạy trước `main()`.

Sau khi chạy, kiểm tra `go.mod` đã tự động thêm `github.com/google/uuid`:
```bash
cat go.mod
```

---

## Tóm tắt nhanh

- **Package** = thư mục `.go` files, đơn vị compilation và visibility.
- **Module** = tập hợp packages + `go.mod`, đơn vị versioning và distribution.
- **`go.mod`** = định nghĩa module, dependencies — tương đương `.csproj`.
- **`go.sum`** = hash verification, không edit tay, luôn commit vào git.
- **`go mod tidy`** = dọn dẹp dependencies, chạy thường xuyên.
- **`internal/`** = encapsulation mạnh hơn C# `internal`, compiler enforced.
- **`init()`** = chạy trước `main()`, tránh logic phức tạp hay có thể fail.
- **Blank import `_`** = trigger `init()` để register side effects.
- **Package naming** = ngắn, lowercase, singular noun, không `utils`/`helpers`.
- **Circular import** = Go cấm hoàn toàn, giải pháp là shared types package hoặc interface.
- **`go work`** = local multi-module development, không commit `go.work`.
- **Vendoring** = offline builds, enterprise environments.
- **`replace` directive** = local dev hoặc fork dependency, không dùng trong publishable libraries.

---

**Tiếp theo:** [09. Generics trong Go](09-generics.md) — `[T any]`, type constraints, và khi nào nên/không nên dùng generics.
