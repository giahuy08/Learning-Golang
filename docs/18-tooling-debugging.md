# 18. Tooling & Debugging trong Go

Nếu bạn đến từ C#, bạn đã quen với Visual Studio hoặc Rider — những IDE nặng ký tích hợp mọi thứ từ build, debug, profiling đến refactoring trong một giao diện đồ họa. Go có cách tiếp cận khác: **toolchain là tập hợp các công cụ nhỏ, làm tốt một việc, kết hợp với nhau qua command line**. Triết lý này giống Unix hơn — và khi bạn quen với nó, bạn sẽ thấy nó thực sự mạnh và nhất quán hơn.

Không có file `.sln`, không có MSBuild XML, không có NuGet GUI. Nhưng bạn sẽ có `go fmt`, `go vet`, `go test`, `dlv`, `pprof`, và `golangci-lint` — mỗi công cụ làm đúng việc của nó, và tất cả đều free, nhanh, và cross-platform.

---

## 1. `go fmt` — Formatter bắt buộc

Trong C#, định dạng code là tùy chọn. `.editorconfig` và Roslyn analyzer có thể enforce một số quy tắc, nhưng team vẫn có thể tranh luận về tabs vs spaces, brace placement, hay line length. Go đơn giản hóa điều này bằng cách **không cho phép tranh luận**: `go fmt` là formatter chính thức, không tùy chỉnh, và toàn bộ cộng đồng Go dùng nó.

Tại sao điều này quan trọng? Code review không còn bị tốn thời gian vào formatting comments. Diff trong git chỉ chứa thay đổi thực sự. Mọi file Go trên GitHub đều trông như nhau. Đây là một sự đánh đổi có chủ ý: mất quyền tùy chỉnh, được lại sự nhất quán tuyệt đối.

```go
// BAD: Code chưa qua go fmt — tab không đúng, spacing lộn xộn
package main
import "fmt"
func main(){
x:=10
if x>5{
fmt.Println("lớn hơn 5")
}
}
```

```go
// GOOD: Sau khi chạy go fmt — gofmt tự fix tất cả
package main

import "fmt"

func main() {
	x := 10
	if x > 5 {
		fmt.Println("lớn hơn 5")
	}
}
```

Chạy thủ công:

```bash
go fmt ./...          # format tất cả file trong project
gofmt -w -l ./...     # gofmt trực tiếp: -w ghi lại file, -l liệt kê file bị thay đổi
```

**Quan trọng**: Trong VS Code, bật format-on-save để không bao giờ phải nhớ chạy tay.

---

## 2. `go vet` — Phát hiện bug thực sự

`go vet` không phải linter thẩm mỹ — nó tìm **bug thực sự**: format string sai kiểu, mutex bị copy, unreachable code, và nhiều hơn nữa. Tương đương gần nhất trong C# là Roslyn analyzer hoặc một số rule trong StyleCop, nhưng `go vet` được tích hợp trong toolchain và không cần cấu hình.

```go
// BAD: Lỗi format string — go vet sẽ phát hiện ra
package main

import "fmt"

func main() {
	name := "Hùng"
	age := 25
	// WRONG: dùng %d cho string — go vet báo lỗi ngay
	fmt.Printf("Tên: %d, Tuổi: %s\n", name, age)
}
```

```go
// GOOD: Format string đúng kiểu
package main

import "fmt"

func main() {
	name := "Hùng"
	age := 25
	// Đúng: %s cho string, %d cho int
	fmt.Printf("Tên: %s, Tuổi: %d\n", name, age)
}
```

```go
// BAD: Mutex bị copy — lỗi nghiêm trọng, go vet phát hiện
package main

import "sync"

type Counter struct {
	mu    sync.Mutex
	count int
}

func process(c Counter) { // WRONG: truyền value — copy mutex!
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}
```

```go
// GOOD: Truyền pointer để tránh copy mutex
package main

import "sync"

type Counter struct {
	mu    sync.Mutex
	count int
}

func process(c *Counter) { // Đúng: dùng pointer
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}
```

```bash
go vet ./...    # chạy vet trên toàn bộ project
```

Luôn chạy `go vet` trong CI. Nó nhanh (dưới 1 giây cho project vừa) và bắt được những lỗi mà code review bỏ sót.

---

## 3. `go build` vs `go run` vs `go install`

Đây là điểm khác biệt rõ ràng so với C#, nơi bạn chỉ có `dotnet build`, `dotnet run`, và `dotnet publish`. Go tách biệt ba use case khác nhau:

**`go run`** biên dịch và chạy ngay, không tạo file binary. Dùng khi đang phát triển và muốn test nhanh — giống `dotnet run` nhưng không để lại artifact nào.

**`go build`** tạo binary trong thư mục hiện tại (hoặc chỉ định). Dùng khi cần file thực thi để deploy, test performance, hay gửi cho người khác. Không có runtime cần thiết — file binary chứa mọi thứ (static linking).

**`go install`** biên dịch và đặt binary vào `$GOPATH/bin` (thường là `~/go/bin`). Dùng cho tool bạn muốn dùng globally như `dlv`, `golangci-lint`. Giống `npm install -g` hoặc `dotnet tool install -g`.

```bash
# Phát triển hàng ngày
go run main.go
go run ./cmd/server/          # chạy package trong thư mục

# Tạo binary để deploy
go build -o bin/myapp ./cmd/server/
GOOS=linux GOARCH=amd64 go build -o bin/myapp-linux ./cmd/server/  # cross-compile!

# Cài tool toàn cục
go install github.com/go-delve/delve/cmd/dlv@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Cross-compilation là một lợi thế lớn của Go so với C#: `GOOS=linux GOARCH=amd64` từ Windows tạo ra binary Linux không cần Docker hay WSL. Trong C#, cross-platform deployment phức tạp hơn nhiều.

---

## 4. `go test` flags — Kiểm soát việc chạy test

C# developer quen với Test Explorer trong Visual Studio — click để chạy test cụ thể. Với Go, tất cả làm qua command line nhưng không kém linh hoạt:

```bash
# Chạy tất cả test
go test ./...

# Verbose: xem tên từng test pass/fail (như Test Explorer output)
go test -v ./...

# Chạy test cụ thể bằng regex — tương đương "Run Test" trong VS
go test -run TestUserService ./internal/user/

# Chạy nhiều test match pattern
go test -run "TestCreate|TestUpdate" ./...

# Tắt cache — Go cache kết quả test, -count=1 buộc chạy lại
go test -count=1 ./...

# Timeout — mặc định 10 phút, set ngắn hơn cho CI
go test -timeout 30s ./...

# Kết hợp nhiều flag
go test -v -run TestUserCreate -count=1 -timeout 10s ./internal/user/

# Chạy benchmark (không phải unit test thông thường)
go test -bench=. -benchmem ./...
```

Lý do `-count=1` tồn tại: Go cache kết quả test thông minh — nếu code và test không thay đổi, Go không chạy lại. Điều này tốt cho CI nhưng đôi khi gây nhầm lẫn ("sao test vẫn pass dù tôi vừa break code?"). `-count=1` bypass cache hoàn toàn.

---

## 5. Race Detector — Tìm Data Race tại Runtime

Data race là một trong những bug khó nhất trong concurrent programming. Trong C#, `async/await` che giấu phần lớn threading complexity, nhưng bạn vẫn có thể bị race condition với `Task.Run` và shared state. Go với goroutines làm concurrent programming dễ hơn nhưng race condition vẫn có thể xảy ra.

Go có race detector tích hợp — nó instrument code tại runtime, theo dõi mọi memory access, và báo lỗi nếu hai goroutine truy cập cùng biến mà không có synchronization.

```go
// BAD: Data race — hai goroutine ghi vào cùng biến
package main

import (
	"fmt"
	"sync"
)

func main() {
	counter := 0
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter++ // RACE: không có lock!
		}()
	}

	wg.Wait()
	fmt.Println(counter) // kết quả không xác định
}
```

```go
// GOOD: Dùng atomic hoặc mutex để tránh race
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func main() {
	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1) // an toàn với concurrent access
		}()
	}

	wg.Wait()
	fmt.Println(counter) // luôn là 1000
}
```

```bash
# Chạy với race detector
go run -race main.go
go test -race ./...

# Output khi có race:
# WARNING: DATA RACE
# Write at 0x00c0000b4010 by goroutine 8:
#   main.main.func1()
# Previous write at 0x00c0000b4010 by goroutine 7:
#   main.main.func1()
```

Race detector làm chậm chương trình khoảng 5-10x và tốn nhiều RAM hơn, nên không dùng trong production. Nhưng **luôn chạy `go test -race ./...` trong CI**. Race condition thường không xuất hiện trong test thông thường nhưng crash production lúc tải cao.

---

## 6. `golangci-lint` — Meta-linter

`go vet` bắt bug, nhưng không bắt code smell, style issue, hay security problem. `golangci-lint` chạy 50+ linter song song và chỉ báo cáo vấn đề mới (dựa trên git diff). Đây là công cụ tương đương với SonarQube + StyleCop + Roslyn analyzer trong một lệnh.

```bash
# Cài đặt
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Chạy
golangci-lint run ./...
golangci-lint run --fix ./...  # tự fix một số vấn đề
```

Tạo file `.golangci.yml` ở root project:

```yaml
# .golangci.yml — cấu hình golangci-lint cho Go project thực tế
run:
  timeout: 5m
  # Bỏ qua generated code và vendor
  skip-dirs:
    - vendor
    - testdata
  skip-files:
    - ".*_generated\\.go$"
    - ".*\\.pb\\.go$"

linters:
  # Tắt tất cả mặc định, bật chọn lọc
  disable-all: true
  enable:
    # Nhóm correctness — tương đương Roslyn error analyzer
    - errcheck        # kiểm tra error bị bỏ qua (rất quan trọng!)
    - gosimple        # suggest code đơn giản hơn
    - govet           # go vet tích hợp
    - ineffassign     # gán biến không dùng
    - staticcheck     # staticcheck analyzer (rất mạnh)
    - unused          # code không được dùng

    # Nhóm style — tương đương StyleCop
    - gofmt           # format check
    - goimports       # import order và unused imports
    - misspell        # typo trong comment và string
    - whitespace      # trailing whitespace

    # Nhóm bugs và security
    - gosec           # security issues (SQL injection, hardcoded credentials...)
    - bodyclose       # HTTP response body phải được đóng
    - contextcheck    # context được truyền đúng không
    - noctx           # HTTP request phải có context

    # Nhóm performance
    - prealloc        # suggest pre-allocate slice
    - gocritic        # nhiều check hữu ích về performance và style

    # Nhóm complexity
    - cyclop          # cyclomatic complexity
    - funlen          # function quá dài
    - gocognit        # cognitive complexity

linters-settings:
  errcheck:
    # Cho phép bỏ qua error của một số hàm (ví dụ: fmt.Println)
    exclude-functions:
      - fmt.Println
      - fmt.Printf

  funlen:
    lines: 80      # function không quá 80 dòng
    statements: 50 # không quá 50 statements

  cyclop:
    max-complexity: 15  # cyclomatic complexity tối đa

  gosec:
    excludes:
      - G104  # Unhandled errors in defer (thường là false positive)

  gocritic:
    enabled-tags:
      - diagnostic
      - performance
      - style

issues:
  # Không báo lỗi trong test file cho một số rule
  exclude-rules:
    - path: "_test\\.go$"
      linters:
        - funlen   # test function có thể dài
        - gosec    # test thường dùng hardcoded value

  # Số lượng issue tối đa (0 = không giới hạn)
  max-issues-per-linter: 0
  max-same-issues: 0
```

Tại sao không bật tất cả linter? Vì một số linter conflict với nhau, một số tạo quá nhiều false positive, và một số chỉ hữu ích trong context cụ thể. Config trên là điểm cân bằng tốt cho production project.

---

## 7. `gopls` — Go Language Server

`gopls` (Go Please) là language server chính thức của Go, implement Language Server Protocol (LSP). Nó cung cấp mọi tính năng IDE bạn quen từ Rider/Visual Studio: auto-complete, go-to-definition, find references, rename, inline error, signature help, hover documentation.

VS Code Go extension tự động quản lý `gopls` — bạn không cần cấu hình gì thêm. Nhưng hiểu `gopls` tồn tại giúp bạn debug khi extension không hoạt động: `Ctrl+Shift+P` → "Go: Restart Language Server".

`gopls` cũng chạy `go vet` và một số staticcheck trong nền và hiện lỗi inline — tương đương với Roslyn live error squiggles trong Visual Studio.

---

## 8. VS Code Go Setup

Đây là `settings.json` đầy đủ cho Go development trong VS Code:

```jsonc
// .vscode/settings.json — settings cho Go project
{
  // Format on save — bắt buộc cho Go developer
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.organizeImports": "explicit"  // tự sort và xóa import thừa
  },

  // Dùng gopls làm formatter (mặc định và tốt nhất)
  "[go]": {
    "editor.defaultFormatter": "golang.go",
    "editor.formatOnSave": true,
    "editor.insertSpaces": false,  // Go dùng tab, không phải space
    "editor.tabSize": 4
  },

  // gopls settings
  "gopls": {
    "ui.semanticTokens": true,           // syntax highlighting tốt hơn
    "ui.diagnostic.staticcheck": true,   // bật staticcheck trong gopls
    "ui.completion.usePlaceholders": true, // placeholder cho function args
    "formatting.gofumpt": false,          // dùng gofmt standard, không phải gofumpt
    "ui.documentation.hoverKind": "FullDocumentation"
  },

  // Go extension settings
  "go.lintTool": "golangci-lint",         // dùng golangci-lint thay vì golint
  "go.lintFlags": ["--fast"],             // chỉ chạy fast linter trong VS Code
  "go.vetOnSave": "package",             // chạy go vet khi save
  "go.buildOnSave": "package",           // build check khi save
  "go.testTimeout": "30s",

  // Test settings — hiện kết quả test rõ hơn
  "go.testFlags": ["-v", "-count=1"],
  "go.coverOnSave": false,
  "go.coverageDecorator": {
    "type": "gutter",
    "coveredHighlightColor": "rgba(64,128,128,0.5)",
    "uncoveredHighlightColor": "rgba(128,64,64,0.25)"
  },

  // Tắt telemetry nếu muốn
  "go.survey.prompt": false,

  // File associations
  "files.associations": {
    "*.go": "go",
    "go.sum": "go"
  },

  // Exclude generated và vendor từ search
  "search.exclude": {
    "**/vendor": true,
    "**/*.pb.go": true,
    "**/*_generated.go": true
  }
}
```

Extensions cần cài:
- **golang.go** — extension chính thức, cài `gopls` tự động
- **streetsidesoftware.code-spell-checker** — bắt typo trong comment
- **usernamehw.errorlens** — hiện error/warning inline cùng dòng (tương đương Rider inline error)

---

## 9. Delve Debugger

Delve là debugger chính thức cho Go — tương đương với Visual Studio Debugger hoặc JetBrains Debugger. Không giống gdb (không hiểu goroutine, không hiểu Go type system), Delve được xây dựng từ đầu cho Go.

```bash
# Cài đặt
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug chương trình — tương đương F5 trong Visual Studio
dlv debug ./cmd/server/

# Debug với arguments
dlv debug ./cmd/server/ -- --port=8080 --env=dev

# Debug test cụ thể
dlv test ./internal/user/ -- -test.run TestUserCreate

# Attach vào process đang chạy
dlv attach <pid>
```

Các lệnh trong Delve shell (sau khi vào debug session):

```
# Breakpoints
break main.main          # set breakpoint tại function
break user.go:42         # set breakpoint tại dòng 42 file user.go
breakpoints              # liệt kê tất cả breakpoint
clear 1                  # xóa breakpoint số 1

# Điều hướng — giống F10/F11/F5 trong Visual Studio
continue                 # tiếp tục đến breakpoint tiếp theo (F5)
next                     # step over — chạy dòng tiếp theo (F10)
step                     # step into — vào trong function (F11)
stepout                  # step out — ra khỏi function hiện tại (Shift+F11)

# Kiểm tra giá trị
print variableName       # in giá trị biến
print user.Email         # in field của struct
locals                   # in tất cả biến local
args                     # in arguments của function hiện tại
whatis variableName      # in kiểu của biến

# Goroutines — đây là điểm khác biệt so với C# debugger
goroutines               # liệt kê tất cả goroutine (như thread list)
goroutine 5              # switch sang goroutine số 5
goroutine 5 bt           # stack trace của goroutine 5

# Stack trace
stack                    # stack trace hiện tại
frame 2                  # switch sang frame số 2 trong stack

# Thoát
exit
```

### VS Code Debug Configuration

Tạo `.vscode/launch.json`:

```jsonc
{
  "version": "0.2.0",
  "configurations": [
    {
      // Debug main application — tương đương "Start Debugging" trong VS
      "name": "Debug Server",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/server",
      "env": {
        "APP_ENV": "development",
        "PORT": "8080"
      },
      "args": [],
      // Hiện output trong Debug Console
      "console": "integratedTerminal"
    },
    {
      // Debug test cụ thể
      "name": "Debug Test",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}/internal/user",
      "args": [
        "-test.run",
        "TestUserCreate",  // thay bằng test bạn muốn debug
        "-test.v"
      ]
    },
    {
      // Debug file hiện tại đang mở
      "name": "Debug Current File",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${file}"
    },
    {
      // Attach vào process đang chạy (remote debugging)
      "name": "Attach to Process",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "${workspaceFolder}",
      "port": 2345,
      "host": "127.0.0.1"
    }
  ]
}
```

Với config này, bạn có thể đặt breakpoint bằng cách click vào lề trái trong VS Code (giống Visual Studio), rồi nhấn F5 để start debug session. Tất cả familiar: watch window, call stack, variables panel — đều hoạt động.

---

## 10. `pprof` — Profiling CPU và Memory

Trong C#, bạn có Visual Studio Profiler hoặc dotMemory/dotTrace từ JetBrains. Go có `pprof` — mạnh không kém nhưng command-line driven. `pprof` đặc biệt hữu ích vì bạn có thể bật profiling endpoint trong production và lấy profile mà không cần restart service.

### Thêm pprof endpoint vào HTTP server

```go
// GOOD: Thêm pprof vào server với bảo vệ môi trường
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof" // side-effect import — đăng ký handlers vào DefaultServeMux
	"os"
)

func main() {
	// Chỉ bật pprof trong development — KHÔNG bao giờ expose production không bảo vệ
	if os.Getenv("APP_ENV") == "development" {
		go func() {
			// pprof chạy trên port riêng
			log.Println("pprof listening on :6060")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Application server bình thường
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### Lấy và đọc profile

```bash
# CPU profile — chạy trong 30 giây và lấy profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory (heap) profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine dump — xem tất cả goroutine đang làm gì
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Sau khi vào pprof shell:
(pprof) top 10          # top 10 function tốn CPU/memory nhất
(pprof) list FunctionName  # xem từng dòng của function cụ thể
(pprof) web             # mở flamegraph trong browser (cần graphviz)
(pprof) pdf             # xuất PDF
```

### Profiling trong test (không cần HTTP server)

```go
// Profiling trong benchmark test
package user_test

import (
	"os"
	"runtime/pprof"
	"testing"
)

func BenchmarkProcessUsers(b *testing.B) {
	// CPU profile
	f, _ := os.Create("cpu.prof")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	for i := 0; i < b.N; i++ {
		processUsers() // function cần benchmark
	}
}
```

```bash
# Chạy benchmark và lấy profile
go test -bench=BenchmarkProcessUsers -cpuprofile=cpu.prof ./internal/user/
go tool pprof cpu.prof

# Memory profile
go test -bench=BenchmarkProcessUsers -memprofile=mem.prof ./internal/user/
go tool pprof mem.prof
```

### Đọc Flamegraph

Flamegraph (khi dùng `web` trong pprof shell) hiện stack call hierarchy. Trục ngang là thời gian/memory, trục dọc là độ sâu call stack. **Ô rộng = function tốn nhiều tài nguyên**. Bạn tìm ô rộng nhất ở gần đỉnh (leaf functions) — đó là nơi cần optimize trước. Không phải function ở đáy stack (chúng rộng vì gọi nhiều thứ khác).

---

## 11. `go doc` — Đọc Documentation từ Terminal

```bash
# Documentation của package
go doc fmt
go doc net/http

# Documentation của type hoặc function cụ thể
go doc fmt.Printf
go doc http.Handler
go doc http.Server.ListenAndServe

# Documentation của package của bạn
go doc ./internal/user
go doc ./internal/user.UserService.Create

# Xem source code cùng documentation
go doc -src fmt.Sprintf
```

Đây là lý do Go enforce documentation comment phải đứng ngay trên declaration — `go doc` đọc và format chúng. Giống XML doc comment trong C# nhưng không cần XML tag.

---

## 12. Makefile cho Go

C# project dùng MSBuild hoặc script PowerShell. Go project thường dùng Makefile — không phải vì Makefile đặc biệt tốt, mà vì nó phổ biến, cross-platform (với Git Bash/WSL trên Windows), và không có build-in task runner nào tốt hơn trong ecosystem.

```makefile
# Makefile — standard targets cho Go project
# Biến
APP_NAME    := myapp
BUILD_DIR   := ./bin
CMD_DIR     := ./cmd/server
GO          := go
GOFLAGS     := -trimpath

# Lấy version từ git tag
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build test lint fmt vet clean run docker help

## all: build và test (default target)
all: fmt vet lint test build

## build: tạo binary
build:
	@echo "Building $(APP_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Binary: $(BUILD_DIR)/$(APP_NAME)"

## build-linux: cross-compile cho Linux (deploy lên server/Docker)
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) \
		-o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_DIR)

## run: chạy ứng dụng (development)
run:
	$(GO) run $(CMD_DIR) --env=development

## test: chạy tất cả test
test:
	$(GO) test -v -count=1 -timeout 60s ./...

## test-race: chạy test với race detector
test-race:
	$(GO) test -race -count=1 -timeout 60s ./...

## test-cover: chạy test với coverage report
test-cover:
	$(GO) test -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## bench: chạy benchmark
bench:
	$(GO) test -bench=. -benchmem -count=3 ./...

## fmt: format code
fmt:
	$(GO) fmt ./...

## vet: chạy go vet
vet:
	$(GO) vet ./...

## lint: chạy golangci-lint
lint:
	golangci-lint run ./...

## lint-fix: chạy golangci-lint và tự fix
lint-fix:
	golangci-lint run --fix ./...

## tidy: dọn dẹp go.mod và go.sum
tidy:
	$(GO) mod tidy

## generate: chạy go generate
generate:
	$(GO) generate ./...

## clean: xóa build artifacts
clean:
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Cleaned"

## docker: build Docker image
docker:
	docker build -t $(APP_NAME):$(VERSION) .

## docker-run: chạy Docker container
docker-run:
	docker run --rm -p 8080:8080 $(APP_NAME):$(VERSION)

## ci: tất cả checks cho CI (không build binary)
ci: fmt vet lint test-race

## help: hiện danh sách targets
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
```

```bash
# Sử dụng
make              # chạy all (fmt + vet + lint + test + build)
make test         # chỉ test
make build-linux  # cross-compile cho Linux
make ci           # CI pipeline
make help         # xem tất cả commands
```

---

## 13. GitHub Actions CI cho Go

C# developer thường quen với Azure DevOps hoặc GitHub Actions với .NET SDK. Đây là workflow hoàn chỉnh cho Go project:

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

env:
  GO_VERSION: "1.23"      # pin version cụ thể, không dùng "latest"
  GOLANGCI_LINT_VERSION: "v1.62"

jobs:
  # Job 1: Lint và vet
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true            # cache Go modules

      - name: Run go vet
        run: go vet ./...

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: ${{ env.GOLANGCI_LINT_VERSION }}
          args: --timeout=5m

  # Job 2: Test với race detector
  test:
    name: Test
    runs-on: ubuntu-latest
    # Chạy với nhiều OS (tương đương test matrix trong Azure DevOps)
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test -race -count=1 -timeout 120s -coverprofile=coverage.out ./...

      - name: Upload coverage
        # Chỉ upload từ Ubuntu để tránh trùng lặp
        if: matrix.os == 'ubuntu-latest'
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out
          flags: unittests

  # Job 3: Build — verify binary có thể compile
  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [lint, test]    # chỉ build nếu lint và test pass
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Build Linux binary
        run: |
          GOOS=linux GOARCH=amd64 go build \
            -trimpath \
            -ldflags "-X main.Version=${{ github.sha }}" \
            -o bin/myapp-linux-amd64 \
            ./cmd/server/

      - name: Build Windows binary
        run: |
          GOOS=windows GOARCH=amd64 go build \
            -trimpath \
            -o bin/myapp-windows-amd64.exe \
            ./cmd/server/

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: bin/
          retention-days: 7

  # Job 4: Security scan
  security:
    name: Security
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
```

Điểm khác biệt với .NET CI: không có `dotnet restore` riêng (Go tự xử lý), không cần `dotnet publish` cho artifacts (Go build tạo single binary), và `go test -race` là CI standard trong Go (không phổ biến trong C# CI).

---

## 14. `go generate` — Code Generation

`go generate` là cách Go làm code generation — tương đương T4 templates trong C# hoặc Source Generator trong Roslyn, nhưng đơn giản hơn: nó chỉ chạy các command được annotate trong comment.

```go
// internal/status/status.go
package status

// go:generate tạo String() method cho UserStatus — không cần viết tay
//
//go:generate stringer -type=UserStatus -linecomment

// UserStatus trạng thái của user account
type UserStatus int

const (
	StatusActive   UserStatus = iota // active
	StatusInactive                   // inactive
	StatusBanned                     // banned
)
```

```bash
# Cài stringer tool
go install golang.org/x/tools/cmd/stringer@latest

# Chạy generate — tạo file userstatus_string.go tự động
go generate ./internal/status/

# File được tạo ra có String() method:
# func (i UserStatus) String() string { ... }
# s := StatusActive.String() // "active"
```

### mockgen — Generate mock cho interface

Trong C#, bạn dùng Moq hoặc NSubstitute. Trong Go, `mockgen` từ `golang/mock` hoặc `mockery` generate mock từ interface:

```go
// internal/user/repository.go
package user

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

// UserRepository interface — mockgen sẽ generate mock cho interface này
type UserRepository interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
}
```

```bash
# Cài mockgen
go install go.uber.org/mock/mockgen@latest

# Generate mocks
go generate ./internal/user/

# Dùng trong test
# mockRepo := mocks.NewMockUserRepository(ctrl)
# mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(user, nil)
```

---

## Bảng Common Issues

| Vấn đề | Nguyên nhân | Cách fix |
|--------|-------------|----------|
| `undefined: PackageName` | Import thiếu hoặc sai package path | Dùng `goimports` để tự fix import |
| `go vet` báo lỗi format string | Dùng sai verb (`%d` cho string) | Kiểm tra kiểu của argument và verb |
| Test fail nhưng không biết tại sao | Kết quả test bị cache | Chạy với `-count=1` |
| Race condition xuất hiện trong CI nhưng không local | Race detector không bật local | Thêm `-race` vào local test command |
| `golangci-lint` chạy quá chậm | Chạy tất cả linter | Dùng `--fast` trong VS Code, đầy đủ trong CI |
| Delve không dừng ở breakpoint | Binary không có debug info | Đừng dùng `-ldflags="-s -w"` khi debug |
| `go: cannot find module` | `go.mod` chưa được init hoặc sai path | Chạy `go mod tidy` |
| pprof endpoint không phản hồi | `net/http/pprof` chưa được import | Thêm `import _ "net/http/pprof"` |
| `golangci-lint: deadline exceeded` | Timeout quá ngắn | Tăng timeout trong `.golangci.yml`: `run.timeout: 10m` |
| Binary size rất lớn | Debug symbols và DWARF data | Build production với `-ldflags="-s -w"` |

---

## Best Practices

**Format và vet là không thể thương lượng.** Trong codebase C#, người mới thường bỏ qua formatter vì IDE không enforce. Trong Go, bạn sẽ bị reject code review nếu chạy `go fmt` ra diff. Cách đơn giản nhất là bật format-on-save trong VS Code và thêm `go fmt ./... && go vet ./...` vào pre-commit hook.

**Race detector là CI requirement, không phải optional.** Data race là silent killer — chương trình chạy đúng trong 99% case và crash lúc production tải cao. `go test -race ./...` chậm hơn 5-10x nhưng CI không cần nhanh như local. Đây là trade-off đáng giá.

**Đừng bỏ qua error từ golangci-lint ngay từ đầu.** Khi project mới, fix tất cả warning. Khi project cũ, dùng `// nolint:lintername` có comment giải thích. `// nolint` không có lý do là technical debt.

**pprof endpoint phải có authentication trong production.** `import _ "net/http/pprof"` trên public port là security vulnerability — nó expose memory content của process. Dùng `localhost:6060` (chỉ accessible từ máy host) hoặc đặt sau authentication middleware.

**Makefile thay vì nhớ commands.** Câu hỏi "làm sao chạy test với race detector?" không nên phải hỏi người khác — `make test-race` là đủ. Makefile là documentation sống của project commands.

**Đặt version cụ thể trong CI.** `go-version: "1.23"` thay vì `"latest"` tránh CI bất ngờ fail sau khi Go release major version mới. Update version có chủ ý.

---

## Bài tập thực hành

### Bài tập 1: Setup hoàn chỉnh cho một module nhỏ

Tạo một Go module với đầy đủ tooling setup: `go fmt`, `go vet`, `golangci-lint`, test với race detector, và Makefile.

```go
// calculator/calculator.go
package calculator

// Calculator thực hiện các phép tính cơ bản
type Calculator struct {
	history []string
}

// Add cộng hai số
func (c *Calculator) Add(a, b float64) float64 {
	result := a + b
	c.history = append(c.history, fmt.Sprintf("%.2f + %.2f = %.2f", a, b, result))
	return result
}

// Divide chia hai số — trả về error nếu chia cho 0
func (c *Calculator) Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("không thể chia cho 0")
	}
	result := a / b
	c.history = append(c.history, fmt.Sprintf("%.2f / %.2f = %.2f", a, b, result))
	return result, nil
}

// History trả về lịch sử tính toán
func (c *Calculator) History() []string {
	// Trả về copy để tránh external modification
	result := make([]string, len(c.history))
	copy(result, c.history)
	return result
}
```

```go
// calculator/calculator_test.go
package calculator_test

import (
	"sync"
	"testing"
)

func TestAdd(t *testing.T) {
	c := &Calculator{}
	result := c.Add(3, 4)
	if result != 7 {
		t.Errorf("Add(3, 4) = %.2f; muốn 7", result)
	}
}

func TestDivide(t *testing.T) {
	t.Run("chia bình thường", func(t *testing.T) {
		c := &Calculator{}
		result, err := c.Divide(10, 2)
		if err != nil {
			t.Fatalf("không mong đợi lỗi: %v", err)
		}
		if result != 5 {
			t.Errorf("Divide(10, 2) = %.2f; muốn 5", result)
		}
	})

	t.Run("chia cho 0", func(t *testing.T) {
		c := &Calculator{}
		_, err := c.Divide(10, 0)
		if err == nil {
			t.Error("mong đợi lỗi khi chia cho 0")
		}
	})
}

// TestConcurrentAdd kiểm tra race condition — chạy với go test -race
func TestConcurrentAdd(t *testing.T) {
	c := &Calculator{}
	var wg sync.WaitGroup

	// 100 goroutine cùng gọi Add
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n float64) {
			defer wg.Done()
			c.Add(n, n)
		}(float64(i))
	}

	wg.Wait()
	// Nếu có race condition, -race flag sẽ báo lỗi
	// Calculator.history slice bị race — đây là điểm để thảo luận
}
```

Chạy bài tập:

```bash
# Khởi tạo module
go mod init github.com/yourname/calculator

# Chạy với race detector để thấy issue
go test -race -v ./...

# Xem go vet có bắt gì không
go vet ./...

# Chạy golangci-lint
golangci-lint run ./...
```

**Gợi ý fix race condition**: `Calculator` cần `sync.Mutex` để protect `history` slice khi có concurrent access.

---

### Bài tập 2: HTTP Server với pprof và debug endpoint

Tạo một HTTP server có pprof, thêm artificial CPU load, và profile nó.

**Solution hoàn chỉnh:**

```go
// main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof" // kích hoạt pprof endpoints
	"os"
	"runtime"
	"time"
)

// BuildInfo chứa thông tin build — sẽ được inject qua ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
)

// heavyComputation mô phỏng CPU-intensive work
func heavyComputation(n int) float64 {
	result := 0.0
	for i := 0; i < n; i++ {
		result += math.Sqrt(float64(i)) * math.Sin(float64(i))
	}
	return result
}

// memoryLeak mô phỏng memory allocation để demo heap profile
func allocateMemory(size int) []byte {
	return make([]byte, size)
}

func main() {
	// pprof server trên port riêng — chỉ localhost
	if os.Getenv("ENABLE_PPROF") == "true" {
		go func() {
			log.Println("pprof server listening on localhost:6060")
			log.Println("Try: go tool pprof http://localhost:6060/debug/pprof/heap")
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				log.Printf("pprof server error: %v", err)
			}
		}()
	}

	// Application server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"version":   Version,
			"buildTime": BuildTime,
			"goroutines": runtime.NumGoroutine(),
		})
	})

	// CPU-intensive endpoint — để demo CPU profiling
	mux.HandleFunc("/compute", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		result := heavyComputation(10_000_000) // 10 triệu iterations
		elapsed := time.Since(start)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result":  result,
			"elapsed": elapsed.String(),
		})
	})

	// Memory allocation endpoint — để demo heap profiling
	mux.HandleFunc("/allocate", func(w http.ResponseWriter, r *http.Request) {
		// Allocate 10MB
		data := allocateMemory(10 * 1024 * 1024)
		fmt.Fprintf(w, "Allocated %d bytes\n", len(data))
		// data sẽ bị GC sau khi function return
	})

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("Server starting on :%s", port)
	log.Printf("Health: http://localhost:%s/health", port)
	log.Printf("CPU load: http://localhost:%s/compute", port)
	log.Printf("Memory: http://localhost:%s/allocate", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

```makefile
# Makefile cho bài tập
.PHONY: run run-pprof profile-cpu profile-mem

run:
	go run main.go

# Chạy với pprof enabled
run-pprof:
	ENABLE_PPROF=true go run main.go &
	@echo "Server running. pprof at localhost:6060"
	@echo "Load test: for i in {1..5}; do curl -s localhost:8080/compute > /dev/null & done"

# Profile CPU — chạy sau khi start server với run-pprof
profile-cpu:
	@echo "Generating CPU load..."
	@for i in 1 2 3 4 5; do curl -s localhost:8080/compute > /dev/null & done
	go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=10

# Profile heap memory
profile-mem:
	go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap
```

Thực hành:

```bash
# Terminal 1: Start server với pprof
ENABLE_PPROF=true go run main.go

# Terminal 2: Tạo load
for i in {1..10}; do curl -s localhost:8080/compute & done

# Terminal 3: Profile CPU (30 giây)
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30

# Profile Heap
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap

# Browser sẽ mở tại localhost:8081 với flamegraph
```

---

## Tổng kết

So sánh nhanh với C# ecosystem:

| Go Tool | C# Equivalent |
|---------|---------------|
| `go fmt` | dotnet-format, editorconfig |
| `go vet` | Roslyn Analyzers |
| `golangci-lint` | SonarQube + StyleCop + Security scan |
| `gopls` | Roslyn (OmniSharp/C# Dev Kit) |
| `dlv` (Delve) | Visual Studio Debugger |
| `go tool pprof` | Visual Studio Profiler, dotTrace |
| `go test -race` | Không có tương đương trực tiếp |
| `go generate` | T4 Templates, Source Generators |
| Makefile | Azure DevOps pipelines, FAKE |
| GitHub Actions Go | GitHub Actions .NET |

Điểm khác biệt quan trọng nhất: **Go toolchain là một phần của ngôn ngữ, không phải addon**. `go fmt`, `go vet`, `go test` đều đến từ cùng một team và được update cùng với language. Không có versioning hell, không có "linter X không tương thích với compiler Y". Sự đơn giản này là một trong những lý do Go được yêu thích trong DevOps và backend infrastructure.

---

**Next:** [19. Performance Basics →](19-performance-basics.md) — Profiling với `pprof`, benchmark, escape analysis, và các sai lầm hiệu năng phổ biến.
