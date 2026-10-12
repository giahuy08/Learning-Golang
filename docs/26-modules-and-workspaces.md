# 26. Modules nâng cao — `go:embed`, Workspaces, Versioning

> `08-packages-modules.md` đã cover `go.mod` cơ bản. Doc này bổ sung 3 kỹ năng cần khi làm việc ở công ty: nhúng file tĩnh vào binary, làm việc với monorepo nhiều module, và publish thư viện versioned đúng chuẩn.

---

## 1. `go:embed` — Nhúng file tĩnh vào binary

Một trong những lợi thế lớn nhất của Go là ship **một binary duy nhất**, không cần file rời đi kèm. `go:embed` (Go 1.16+) cho phép nhúng file/thư mục thẳng vào binary lúc compile — không cần đọc filesystem lúc runtime.

```go
package main

import (
    "embed"
    "fmt"
)

//go:embed templates/welcome.html
var welcomeTemplate string // đọc trực tiếp thành string nếu là 1 file text

//go:embed config/default.json
var defaultConfig []byte // đọc thành []byte nếu cần raw bytes

//go:embed static
var staticFiles embed.FS // đọc cả THƯ MỤC thành virtual filesystem

func main() {
    fmt.Println(welcomeTemplate)

    // Đọc file cụ thể từ embed.FS
    data, _ := staticFiles.ReadFile("static/logo.png")
    fmt.Println(len(data))
}
```

### Use case thực tế: Serve static files từ binary (không cần thư mục kèm theo)

```go
// GOOD: HTTP server serve static file — KHÔNG cần deploy thư mục static/ riêng
//go:embed static
var staticFS embed.FS

func main() {
    // http.FS convert embed.FS thành http.FileSystem
    sub, _ := fs.Sub(staticFS, "static") // bỏ prefix "static/" khỏi URL path
    http.Handle("/", http.FileServer(http.FS(sub)))
    http.ListenAndServe(":8080", nil)
}
```

Trước `go:embed`, bạn phải dùng thư viện bên thứ 3 như `packr` hoặc `go-bindata` để làm việc này — generate file `.go` chứa base64-encoded content. `go:embed` là native, không cần code generation, không cần dependency ngoài.

### Use case: CLI tool với default config nhúng sẵn

```go
//go:embed config/default.yaml
var defaultConfigYAML []byte

func LoadConfig(userConfigPath string) (*Config, error) {
    cfg := &Config{}
    if err := yaml.Unmarshal(defaultConfigYAML, cfg); err != nil {
        return nil, err
    }
    if userConfigPath != "" {
        // Override bằng config user cung cấp, nếu có
        userData, err := os.ReadFile(userConfigPath)
        if err == nil {
            yaml.Unmarshal(userData, cfg)
        }
    }
    return cfg, nil
}
```

### Giới hạn cần nhớ

- `//go:embed` phải ngay phía trên khai báo biến, **không có dòng trống ở giữa**.
- Chỉ nhúng được file **trong cùng module**, không đi ra ngoài (không thể `//go:embed ../other-project/file.txt`).
- File bắt đầu bằng `.` hoặc `_` bị bỏ qua mặc định khi embed cả thư mục (dùng pattern `all:static` để lấy luôn cả các file này).
- Binary sẽ **lớn hơn** tương ứng với dung lượng file embed — cân nhắc với asset lớn (video, dataset).

---

## 2. `go.work` — Go Workspaces cho Monorepo

Khi làm việc với **nhiều module Go trong cùng repo** (monorepo — pattern phổ biến ở công ty có nhiều service), bạn thường gặp vấn đề: module A muốn dùng code từ module B đang phát triển cùng lúc, nhưng B chưa được publish/tag version.

```
myrepo/
├── go.work           ← File điều phối, KHÔNG commit vào production build
├── service-a/
│   └── go.mod        ← module github.com/company/service-a
├── service-b/
│   └── go.mod        ← module github.com/company/service-b
└── shared-lib/
    └── go.mod        ← module github.com/company/shared-lib
```

```bash
# Tạo go.work tại root, liên kết cả 3 module lại
go work init ./service-a ./service-b ./shared-lib
```

File `go.work` được tạo ra:

```
go 1.22

use (
    ./service-a
    ./service-b
    ./shared-lib
)
```

Giờ đây, `service-a/go.mod` vẫn khai báo `require github.com/company/shared-lib v1.2.0` như bình thường (để build production dùng version đã publish), nhưng khi bạn `go build`/`go run` **trong workspace này**, Go sẽ tự động dùng code local ở `./shared-lib` thay vì tải version từ registry — không cần `replace` directive rối rắm trong `go.mod`.

```bash
go work use ./new-service   # thêm module mới vào workspace
go work sync                # đồng bộ version giữa các go.mod trong workspace
```

**Quan trọng:** `go.work` là công cụ **local development only** — không nên commit vào git nếu team không đồng thuận dùng chung cấu trúc monorepo này (nó ảnh hưởng tới build của mọi người dùng repo). Nhiều team thêm `go.work` vào `.gitignore` (như đã làm trong `C:\Learn\Go\.gitignore` của bạn) và mỗi dev tự tạo workspace theo nhu cầu.

---

## 3. Semantic Import Versioning — Cách Go xử lý version `v2+`

Go module dùng **Semantic Import Versioning**: từ version `v2.0.0` trở đi, **module path phải có suffix `/v2`** — đây là điểm khác biệt lớn nhất so với npm/NuGet (nơi version không nằm trong import path).

```go
// go.mod của module bạn, khi lên version 2+
module github.com/company/mylib/v2

go 1.22
```

```go
// Code import PHẢI có /v2 trong path
import "github.com/company/mylib/v2/handler"
```

**Tại sao Go thiết kế vậy?** Để cho phép **2 major version cùng tồn tại trong cùng 1 build** — dependency A dùng `mylib/v1`, dependency B dùng `mylib/v2`, cả hai đều compile được vì Go coi chúng là **2 package hoàn toàn khác nhau** (khác import path). Đây là cách Go giải quyết "diamond dependency problem" mà nhiều hệ sinh thái khác (npm, đặc biệt trước có version resolution mạnh) gặp khó khăn.

### Quy trình release version mới

```bash
# Release v1.0.0 — không cần đổi module path
git tag v1.0.0
git push origin v1.0.0

# Release v1.1.0 — patch/minor, vẫn không đổi module path
git tag v1.1.0
git push origin v1.1.0

# Release v2.0.0 — BREAKING CHANGE, phải đổi module path trong go.mod
# 1. Sửa go.mod: module github.com/company/mylib/v2
# 2. Sửa mọi import nội bộ trỏ sang /v2
# 3. Tag và push
git tag v2.0.0
git push origin v2.0.0
```

---

## 4. Công cụ điều tra dependency: `go mod why`, `go mod graph`, `go list`

```bash
# Vì sao module X nằm trong dependency tree của tôi?
go mod why github.com/some/indirect-dependency

# In toàn bộ dependency graph (dạng "A -> B")
go mod graph

# Liệt kê mọi dependency kèm version đang dùng
go list -m all

# Kiểm tra version mới nhất available cho từng dependency
go list -m -u all

# Tìm dependency có lỗ hổng bảo mật đã biết (kết hợp với govulncheck)
govulncheck ./...
```

`go mod why` đặc biệt hữu ích khi `go.sum` có một package lạ bạn không nhớ đã thêm — nó cho biết chính xác chain import nào kéo package đó vào.

---

## 5. `tool` directive trong `go.mod` (Go 1.24+) — Quản lý dev tool đúng chuẩn

**Vấn đề trước Go 1.24:** bạn cần cài một CLI tool viết bằng Go (linter, code generator như `mockgen`, `sqlc`, `protoc-gen-go`) và muốn version của nó được lock trong `go.mod`/`go.sum` giống mọi dependency khác — để cả team và CI dùng đúng 1 version, tránh "máy tôi chạy version khác máy bạn". Trước 1.24, cộng đồng dùng một hack gọi là **tools.go pattern**:

```go
//go:build tools
// +build tools

// tools.go — file giả, không bao giờ build vào binary thật,
// chỉ tồn tại để ép go.mod ghi nhận các tool này là dependency
package tools

import (
    _ "github.com/golang/mock/mockgen"
    _ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
```

Hack này hoạt động nhưng khó hiểu với người mới (tại sao lại `import _` một CLI tool?), và `go build`/`go vet` vẫn phải quét qua file này dù nó không compile ra gì.

**Go 1.24 chính thức hoá bằng `tool` directive:**

```bash
# Thêm tool vào go.mod, tự động cài đặt version hiện tại làm dependency
go get -tool github.com/golang/mock/mockgen@latest
```

Kết quả trong `go.mod`:

```go
module github.com/mycompany/myapi

go 1.24

require (
    github.com/golang/mock v1.6.0
)

tool github.com/golang/mock/mockgen // khai báo tường minh: đây là dev tool, không phải runtime dependency
```

Chạy tool đã khai báo, luôn đúng version đã lock trong `go.mod` — không cần `go install` riêng lẻ ngoài module:

```bash
go tool mockgen -source=user_repo.go -destination=mocks/user_repo_mock.go
```

Ưu điểm so với `tools.go`: không cần file giả, không cần `//go:build` hack, `go mod tidy` hiểu và quản lý `tool` directive như một loại dependency riêng biệt (không lẫn với dependency runtime của chính ứng dụng).

---

## Bảng tổng kết

| Công cụ | Dùng khi nào |
|---|---|
| `go:embed` | Cần ship file tĩnh (template, config default, static assets) trong 1 binary duy nhất |
| `go.work` | Phát triển nhiều module liên quan cùng lúc trong monorepo, chưa muốn publish version mới |
| `module/v2` suffix | Release breaking change (major version bump) cho thư viện public |
| `go mod why` | Điều tra vì sao 1 dependency lạ xuất hiện trong `go.sum` |
| `go mod graph` | Visualize toàn bộ dependency tree |
| `tool` directive (Go 1.24+) | Lock version của dev tool (linter, codegen) trong go.mod thay vì hack `tools.go` |

---

## Bài tập

1. Tạo một CLI tool nhỏ có file `banner.txt` chứa ASCII art, dùng `go:embed` để nhúng và in ra khi chạy `--version`.
2. Tạo 2 module riêng (`lib-a`, `app-b` dùng `lib-a`) trong cùng thư mục, dùng `go work init` để link chúng, sửa code trong `lib-a` và verify `app-b` thấy thay đổi ngay mà không cần `go mod tidy` lại.
