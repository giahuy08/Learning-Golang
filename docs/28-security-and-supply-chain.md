# 28. Security & Supply Chain — Thói quen, không chỉ CI check

> `18-tooling-debugging.md` đã nhắc `govulncheck` và `gosec` như một phần CI. Doc này biến chúng thành **thói quen chủ động** thay vì chỉ là bước tự động chạy nền — vì bạn cần biết đọc kết quả và quyết định hành động, không chỉ nhìn thấy CI xanh.

---

## 1. `govulncheck` — Quét lỗ hổng biết trước trong dependency

Khác với các scanner chỉ so khớp version với CVE database (nhiều false positive vì báo lỗi cả khi bạn không dùng đến function bị lỗi), `govulncheck` phân tích **call graph thực tế** của chương trình bạn — chỉ báo lỗi nếu code của bạn thực sự gọi tới function có lỗ hổng.

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest

# Quét toàn bộ module
govulncheck ./...
```

Output mẫu:

```
Vulnerability #1: GO-2023-1571
    Denial of service via crafted Content-Type header in net/http
  More info: https://pkg.go.dev/vuln/GO-2023-1571
  Module: stdlib
    Found in: net/http@go1.19.4
    Fixed in: net/http@go1.19.9
  Call stacks in your code:
    main.go:23:2: main.main calls net/http.ListenAndServe
```

**Cách đọc:** phần "Call stacks in your code" cho biết chính xác dòng code nào dẫn tới hàm bị lỗi. Nếu phần này **trống** (không có call stack), nghĩa là dependency có lỗ hổng nhưng code bạn **không thực sự gọi** tới đường dẫn bị ảnh hưởng — mức độ ưu tiên fix thấp hơn nhiều so với khi có call stack rõ ràng.

### Khi nào chạy

- Trước mỗi release/tag version — không chỉ dựa vào CI chạy tự động hàng ngày.
- Sau khi `go get -u` để update dependency — kiểm tra ngay xem version mới có tạo ra hoặc fix lỗ hổng nào.
- Định kỳ hàng tuần cho project đang chạy production lâu dài, kể cả không có code thay đổi (lỗ hổng mới được công bố liên tục).

```bash
# Kết hợp vào Makefile — như đã có sẵn khái niệm 'ci' target trong Makefile của bạn
security:
	govulncheck ./...
```

---

## 2. `gosec` — Tìm pattern code không an toàn

`gosec` khác `govulncheck`: nó không quét dependency mà quét **code của chính bạn** để tìm pattern nguy hiểm phổ biến (SQL injection, hardcoded credentials, weak crypto...).

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...
```

```go
// BAD: SQL injection — gosec sẽ báo G201
func GetUser(db *sql.DB, id string) (*User, error) {
    query := "SELECT * FROM users WHERE id = " + id // NGUY HIỂM: string concat trực tiếp
    return db.Query(query)
}
```

```go
// GOOD: parameterized query — an toàn
func GetUser(db *sql.DB, id string) (*User, error) {
    query := "SELECT * FROM users WHERE id = $1"
    return db.Query(query, id) // driver tự escape, không thể inject
}
```

```go
// BAD: hardcoded credential — gosec báo G101
const dbPassword = "SuperSecret123" // KHÔNG BAO GIỜ hardcode secret trong code
```

```go
// GOOD: đọc từ environment hoặc secret manager
dbPassword := os.Getenv("DB_PASSWORD")
if dbPassword == "" {
    log.Fatal("DB_PASSWORD is required")
}
```

Danh sách rule quan trọng nhất `gosec` bắt được:

| Rule | Vấn đề |
|---|---|
| G101 | Hardcoded credentials |
| G104 | Bỏ qua error return (thường false positive cho `defer f.Close()`, có thể exclude) |
| G201/G202 | SQL query xây dựng bằng string concat thay vì parameterized |
| G304 | Path traversal — mở file với path đến từ user input không validate |
| G401/G402/G403 | Dùng thuật toán crypto yếu (MD5, DES, RSA key < 2048 bit) |
| G501-G505 | Import package crypto không an toàn (`crypto/md5`, `crypto/des`) |

---

## 3. Path Traversal — Lỗi thực tế hay gặp khi serve file từ user input

```go
// BAD: path traversal — user có thể truyền "../../etc/passwd" để đọc file bất kỳ
func downloadFile(w http.ResponseWriter, r *http.Request) {
    filename := r.URL.Query().Get("file")
    data, _ := os.ReadFile("uploads/" + filename) // NGUY HIỂM
    w.Write(data)
}
```

```go
// GOOD: dùng filepath.Clean + kiểm tra path vẫn nằm trong thư mục cho phép
func downloadFile(w http.ResponseWriter, r *http.Request) {
    filename := r.URL.Query().Get("file")

    baseDir := "uploads"
    fullPath := filepath.Join(baseDir, filepath.Clean("/"+filename)) // Clean loại bỏ "../"

    // Double-check: đảm bảo path cuối cùng vẫn nằm trong baseDir
    if !strings.HasPrefix(fullPath, filepath.Clean(baseDir)+string(os.PathSeparator)) {
        http.Error(w, "invalid file path", http.StatusBadRequest)
        return
    }

    data, err := os.ReadFile(fullPath)
    if err != nil {
        http.Error(w, "file not found", http.StatusNotFound)
        return
    }
    w.Write(data)
}
```

### 3.1. `os.Root` (Go 1.24+) — Cách chuẩn thay thế `filepath.Clean` + `strings.HasPrefix`

Pattern `filepath.Clean` + kiểm tra prefix ở trên **vẫn đúng và vẫn hoạt động**, nhưng dễ viết sai (quên `+ string(os.PathSeparator)` sẽ cho phép bypass bằng thư mục tên `uploads-evil/`), và mỗi nơi trong codebase phải tự nhớ lặp lại logic này. Go 1.24 thêm `os.Root` — một handle filesystem bị **giới hạn cứng** trong một thư mục gốc, không thể thoát ra ngoài dù dùng `../` hay symlink trỏ ra ngoài:

```go
// GOOD (Go 1.24+): os.Root giới hạn filesystem access trong "uploads", không cần tự viết logic chống traversal
func downloadFile(w http.ResponseWriter, r *http.Request) {
    filename := r.URL.Query().Get("file")

    root, err := os.OpenRoot("uploads")
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    defer root.Close()

    // root.Open TỰ TỪ CHỐI mọi path thoát ra ngoài "uploads", kể cả "../../etc/passwd"
    // hoặc symlink trỏ ra ngoài root — trả về lỗi thay vì đọc nhầm file
    f, err := root.Open(filename)
    if err != nil {
        http.Error(w, "file not found", http.StatusNotFound)
        return
    }
    defer f.Close()

    io.Copy(w, f)
}
```

`os.Root` là lựa chọn nên dùng mặc định cho mọi tính năng "serve file theo tên do user cung cấp" khi module của bạn ở Go 1.24+ — nó loại bỏ cả lớp bug do tự viết logic validate path sai. Với module còn ở Go < 1.24, pattern `filepath.Clean` + `strings.HasPrefix` ở trên vẫn là cách đúng để dùng.

---

## 4. Secrets Management — Không bao giờ commit secret vào git

```bash
# GOOD: dùng git-secrets hoặc gitleaks để CHẶN commit có secret TRƯỚC khi nó vào history
go install github.com/gitleaks/gitleaks@latest
gitleaks detect --source . -v
```

Nếu secret đã lỡ commit: **đổi secret ngay lập tức** (rotate credential) — xóa khỏi git history (`git filter-repo`) không đủ, vì secret đã có thể bị đọc bởi bất kỳ ai từng clone hoặc bởi các bot quét GitHub public repo tự động trong vài giây sau khi push.

Trong production, dùng secret manager thay vì environment variable thuần khi có thể:

| Cloud | Secret Manager |
|---|---|
| AWS | AWS Secrets Manager / Parameter Store |
| GCP | Secret Manager |
| Azure | Key Vault |
| Kubernetes | Sealed Secrets / External Secrets Operator, không commit raw `Secret` YAML |

---

## 5. Pin version chính xác — Tránh "dependency confusion" và supply chain attack

```bash
# BAD: @latest trong CI có thể kéo về version độc hại mới publish (supply chain attack)
go install some/tool@latest

# GOOD: pin version cụ thể đã audit, đặc biệt trong CI/production build
go install some/tool@v1.4.2
```

`go.sum` đã tự động pin **hash checksum** của mọi dependency — nếu ai đó publish lại cùng version nhưng nội dung khác (compromised registry), `go build` sẽ **fail** với lỗi checksum mismatch thay vì âm thầm dùng code độc hại. Đây là lý do **không bao giờ** xóa hoặc bypass `go.sum` bằng `GONOSUMCHECK` hay `GOFLAGS=-insecure` trừ khi bạn hiểu rõ hậu quả.

```bash
# Kiểm tra go.sum có bị thiếu entry nào không (nên chạy trong CI)
go mod verify
```

---

## Bảng tổng kết — Checklist bảo mật thực chiến

| Việc cần làm | Tần suất |
|---|---|
| `govulncheck ./...` | Trước mỗi release, sau mỗi `go get -u`, định kỳ hàng tuần |
| `gosec ./...` | Trong CI mỗi PR (đã có trong `.golangci.yml` của bạn qua linter `gosec`) |
| `gitleaks detect` | Pre-commit hook, trước mỗi push |
| `go mod verify` | Trong CI, đảm bảo dependency không bị tamper |
| Review `go.sum` diff trong PR | Mỗi khi thêm/update dependency mới — hỏi "tại sao cần thêm package này?" |
| Rotate secret nếu lỡ commit | Ngay lập tức, không chờ xóa git history |

---

## Bài tập

1. Cài `gitleaks`, chạy trên project `C:\Learn\Go` để verify không có secret nào bị commit.
2. Viết một hàm `SaveUploadedFile` nhận filename từ HTTP multipart form, cố tình thử tấn công path traversal (`../../windows/system32/config`) để xác nhận code của bạn chặn được, sau đó fix bằng pattern ở mục 3.
