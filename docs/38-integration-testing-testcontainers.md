# 38. Integration Test với DB thật — testcontainers, test đồng thời, golden file

> Bổ sung cho [06-testing.md](06-testing.md). Mock DB không bắt được lỗi SQL, constraint, transaction hay race thật. Test repository/service quan trọng nên chạy trên Postgres thật.

---

## 1. Tách integration test khỏi unit test

```go
//go:build integration

package repo_test
```
```makefile
test:             ; go test -race ./...
test-integration: ; go test -race -tags=integration ./...
```
Unit test chạy trong vài giây, không cần Docker; integration chạy ở CI (hoặc trước khi push).

## 2. Postgres bằng `testcontainers-go`

```go
// go get github.com/testcontainers/testcontainers-go/modules/postgres
import (
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("shop"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(), // chờ DB thật sự sẵn sàng
	)
	if err != nil { log.Fatal(err) }

	dsn, _ := ctr.ConnectionString(ctx, "sslmode=disable")
	testDB, err = sql.Open("pgx", dsn)
	if err != nil { log.Fatal(err) }
	if err := migrate(testDB); err != nil { log.Fatal(err) } // chạy schema thật của dự án

	code := m.Run()
	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code) // lưu ý: os.Exit bỏ qua defer, nên terminate gọi tường minh ở trên
}
```
Khởi động 1 container cho cả package (nhanh), **cô lập từng test** bằng một trong hai cách:
- Mỗi test chạy trong transaction rồi `Rollback` ở `t.Cleanup` (nhanh nhất, nhưng không test được code tự mở transaction/commit).
- `TRUNCATE ... RESTART IDENTITY CASCADE` đầu mỗi test (đơn giản, chắc chắn).

```go
func reset(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(`TRUNCATE orders, order_items, products RESTART IDENTITY CASCADE`)
	if err != nil { t.Fatal(err) }
}
```
(Test dùng chung DB thì **không** dùng `t.Parallel()` giữa các test cùng bảng, hoặc dùng schema/DB riêng cho từng test.)

## 3. Test chứng minh không oversell (liên kết bài 35)

```go
func TestPlaceOrder_NoOversell(t *testing.T) {
	reset(t)
	testDB.Exec(`INSERT INTO products(id, stock) VALUES ('p1', 10)`)
	svc := NewService(testDB)

	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.Buy(context.Background(), "p1", 1); err == nil {
				ok.Add(1)
			} else if !errors.Is(err, ErrOutOfStock) {
				t.Errorf("lỗi lạ: %v", err)
			}
		}()
	}
	wg.Wait()

	if ok.Load() != 10 { t.Fatalf("bán %d, kỳ vọng 10", ok.Load()) }
	var stock int
	testDB.QueryRow(`SELECT stock FROM products WHERE id='p1'`).Scan(&stock)
	if stock != 0 { t.Fatalf("kho còn %d", stock) }
}
```
Chạy với `-race -count=20`. Đây là loại test khiến người phỏng vấn tin bạn hiểu concurrency + DB.

## 4. `httptest` cho handler (không cần mạng)

```go
func TestCreateOrder_Idempotent(t *testing.T) {
	srv := httptest.NewServer(newRouter(testDB))
	defer srv.Close()

	post := func() *http.Response {
		req, _ := http.NewRequest("POST", srv.URL+"/orders", strings.NewReader(`{"sku":"p1","qty":1}`))
		req.Header.Set("Idempotency-Key", "abc")
		resp, err := http.DefaultClient.Do(req)
		if err != nil { t.Fatal(err) }
		return resp
	}
	a, b := post(), post()
	if a.StatusCode != 201 || b.StatusCode != 201 { t.Fatal("kỳ vọng 201 cả hai lần") }
	// rồi COUNT(*) FROM orders == 1
}
```

## 5. Golden file (so sánh output phức tạp)

```go
var update = flag.Bool("update", false, "ghi lại golden")

func TestRenderInvoice(t *testing.T) {
	got := RenderInvoice(sample)
	path := filepath.Join("testdata", t.Name()+".golden")
	if *update { os.WriteFile(path, got, 0o644) }
	want, _ := os.ReadFile(path)
	if !bytes.Equal(got, want) { t.Errorf("khác golden; chạy -update nếu thay đổi là chủ ý\n%s", got) }
}
```
`go test -run TestRenderInvoice -update` để tạo/ghi lại, **review diff trong git** trước khi commit.

## 6. Mock bằng interface (không cần thư viện)

Định nghĩa interface nhỏ ở phía consumer, viết fake thủ công:
```go
type mailer interface{ Send(ctx context.Context, to, body string) error }
type fakeMailer struct{ sent []string; err error }
func (f *fakeMailer) Send(_ context.Context, to, _ string) error { f.sent = append(f.sent, to); return f.err }
```
Chỉ dùng `mockery`/`gomock` khi interface lớn và có nhiều test cần verify tương tác. Ưu tiên **fake có hành vi** hơn mock kiểm tra "hàm X được gọi mấy lần".

## Bài tập
1. Dựng `TestMain` với testcontainers cho repository của mini-ecommerce.
2. Viết test oversell ở mục 3 — chạy với cách trừ kho BAD để thấy fail, rồi sửa.
3. Thêm test idempotency ở mục 4.
4. Đo thời gian: unit test vs integration test; đặt integration sau build tag.

**Liên quan:** [06-testing.md](06-testing.md), [35-database-transactions-locking.md](35-database-transactions-locking.md), [36-idempotency-and-reliability.md](36-idempotency-and-reliability.md), [39-docker-deployment-ci.md](39-docker-deployment-ci.md)
