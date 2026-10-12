# 20. Project Setup Checklist — Tooling Install Guide

Dùng doc này để tự setup máy cho Go development. Chạy từng lệnh trong terminal (PowerShell hoặc Git Bash đều được — các lệnh `go install` giống nhau).

---

## 0. Fix kiến trúc Go (bắt buộc trước tiên)

Máy bạn đang chạy Go bản **32-bit (386)** trên Windows 64-bit — nên đổi sang **64-bit (amd64)**:

1. Gỡ Go hiện tại: Settings → Apps → "Go Programming Language" → Uninstall.
2. Tải bản `go1.2x.x.windows-amd64.msi` tại https://go.dev/dl/ (chú ý đuôi `amd64`, không phải `386`).
3. Cài đặt (giữ mặc định, sẽ cài vào `C:\Program Files\Go`).
4. Mở terminal **mới** và kiểm tra:
   ```
   go version
   go env GOARCH GOROOT
   ```
   Kết quả mong đợi: `windows/amd64` và `GOROOT=C:\Program Files\Go`.

### 0.1. `go : The term 'go' is not recognized...` — PATH thiếu, không phải chưa cài

Gặp lỗi này nghĩa là **Go đã có trên máy nhưng thư mục `bin` của nó chưa nằm trong PATH** — khác với "chưa cài Go". Kiểm tra Go thực sự nằm ở đâu:

```powershell
Test-Path "C:\Program Files\Go\bin\go.exe"
Test-Path "C:\Program Files (x86)\Go\bin\go.exe"
Get-ChildItem -Path C:\ -Filter go.exe -Recurse -ErrorAction SilentlyContinue -Depth 3 | Select-Object FullName
```

Nếu tìm thấy ở `C:\Program Files (x86)\Go\bin\go.exe` (không phải `Program Files` không có `(x86)`) — đây chính là dấu hiệu của bản cài **32-bit** nói ở đầu mục này, nên xử lý luôn theo 4 bước phía trên (gỡ + cài lại bản `amd64`) thay vì chỉ sửa PATH, vì sửa PATH xong vẫn chạy nhầm bản 32-bit.

Nếu vẫn muốn test nhanh trước khi gỡ cài lại (không cần khởi động lại terminal):

```powershell
$env:PATH += ";C:\Program Files (x86)\Go\bin"   # đổi path đúng với nơi tìm thấy go.exe ở trên
go version
```

Cách này chỉ có hiệu lực trong cửa sổ PowerShell đang mở — đóng lại là mất. Để cố định:

1. `Win` → gõ **"Environment Variables"** → **"Edit the system environment variables"**
2. **Environment Variables** → chọn `Path` ở **User variables** → **Edit** → **New**
3. Thêm đường dẫn `bin` của Go (ví dụ `C:\Program Files\Go\bin` sau khi cài lại bản đúng)
4. OK hết, **đóng và mở lại** terminal/VS Code để PATH mới có hiệu lực

### 0.2. VS Code không autocomplete/suggest syntax Go — hệ quả trực tiếp của mục 0.1

`gopls` (language server) không khởi động được nếu extension không tìm ra `go` trong PATH — nên PATH thiếu không chỉ làm `go version` báo lỗi trong terminal, mà còn làm mất toàn bộ autocomplete/go-to-definition/suggest trong VS Code. Sau khi đã fix PATH ở mục 0.1:

1. **Đóng hẳn VS Code (không chỉ Reload Window) rồi mở lại** — VS Code chỉ đọc PATH của hệ điều hành lúc process khởi động.
2. Mở 1 file `.go` bất kỳ, nhìn status bar góc dưới-phải — phải thấy version Go hiện ra, nếu không nghĩa là extension vẫn chưa tìm thấy `go`.
3. Command Palette (`Ctrl+Shift+P`) → **"Go: Install/Update Tools"** → tick tất cả (đặc biệt `gopls`) → Install. Có `go` trong PATH không có nghĩa `gopls` đã được cài — đây là 2 thứ khác nhau (xem mục 3 phía dưới).
4. Nếu vẫn không có suggest: Command Palette → **"Go: Show Output"** để xem log lỗi thật từ `gopls`.

---

## 1. Project đã được setup sẵn (đã làm trong session này)

Các file sau đã được tạo tại `C:\Learn\Go\`:

| File | Mục đích |
|---|---|
| `go.mod` | Module `learn-go`, khởi tạo bằng `go mod init learn-go` |
| `.git/` | Git repo đã init (`git init`) |
| `.gitignore` | Bỏ qua build artifacts, coverage, `.exe` |
| `.golangci.yml` | Config cho `golangci-lint` — bật errcheck, staticcheck, gosec, gocritic... |
| `Makefile` | Chạy `make help` để xem: `run`, `fmt`, `vet`, `lint`, `test`, `test-race`, `test-cover`, `bench`, `ci` |
| `.vscode/settings.json` | Format-on-save, gopls, golangci-lint làm linter mặc định |
| `.vscode/launch.json` | 3 debug config: file hiện tại, `cmd/playground`, test package |
| `cmd/playground/` | Thư mục trống — nơi thực hành project structure (Phase Backend APIs) |

Bạn chưa cần commit gì — file `main.go` ở root vẫn giữ nguyên làm scratch file cho luyện tập Phase 1.

---

## 2. Cài VS Code extension

Mở VS Code trong `C:\Learn\Go`, cài các extension sau (search trong Extensions panel):

- **`golang.go`** — extension chính thức, tự động cài `gopls` khi mở file `.go` đầu tiên (VS Code sẽ hỏi "Install missing tools", bấm **Install All**)
- **`usernamehw.errorlens`** — hiện lỗi/warning inline
- **`streetsidesoftware.code-spell-checker`** — bắt typo trong comment

Khi extension `golang.go` hỏi "Install missing tools", nó sẽ tự cài phần lớn danh sách bên dưới — nhưng nên biết cách cài tay để hiểu công cụ nào đang chạy.

---

## 3. Cài tools qua terminal (nếu muốn cài tay thay vì để VS Code tự cài)

Chạy từng dòng — mỗi lệnh tải và build tool vào `%USERPROFILE%\go\bin` (đã có trong PATH nếu cài Go đúng cách):

```bash
# Language server — bắt buộc cho autocomplete/go-to-definition trong VS Code
go install golang.org/x/tools/gopls@latest

# Debugger — tương đương Visual Studio Debugger
go install github.com/go-delve/delve/cmd/dlv@latest

# Meta-linter — chạy 50+ linter (errcheck, staticcheck, gosec...)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Security vulnerability scanner cho dependencies
go install golang.org/x/vuln/cmd/govulncheck@latest

# Tự động thêm/xóa import khi save (VS Code dùng cái này)
go install golang.org/x/tools/cmd/goimports@latest

# Generate String() method cho enum-like types (dùng ở Phase 3+)
go install golang.org/x/tools/cmd/stringer@latest

# Generate mock từ interface — dùng khi viết unit test có dependency
go install go.uber.org/mock/mockgen@latest
```

Kiểm tra cài thành công:

```bash
gopls version
dlv version
golangci-lint version
govulncheck -version
goimports -h
stringer -h
mockgen -version
```

Nếu lệnh báo "not found", kiểm tra `%USERPROFILE%\go\bin` đã nằm trong `PATH` chưa:

```bash
go env GOPATH   # thường là C:\Users\<user>\go — bin nằm trong GOPATH\bin
```

---

## 4. Verify toàn bộ setup

```bash
cd /c/Learn/Go

go build ./...        # phải compile được (không có gì trong cmd/playground là OK)
go vet ./...           # bắt bug thật
make fmt               # format code
golangci-lint run ./... # full lint pass
go test ./...           # chưa có test thì sẽ báo "no test files" — bình thường
```

---

## 5. (Tùy chọn) Git commit đầu tiên

Repo đã `git init` nhưng chưa có commit nào. Khi sẵn sàng:

```bash
git add .
git commit -m "chore: init learn-go module with tooling setup"
```

---

## 6. Optional nhưng nên có khi lên Phase 4+ (Backend APIs)

- **Docker Desktop** — để chạy PostgreSQL local và test containerize app
- **`golang-migrate` CLI** — quản lý DB migration: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- **`sqlc` CLI** — generate type-safe Go từ SQL, tải binary tại https://github.com/sqlc-dev/sqlc/releases (không dùng `go install` vì cần CGO-free build)
- **Postman hoặc `curl`/`httpie`** — test API thủ công
- **`grpcurl`** — test gRPC endpoint không cần client code
