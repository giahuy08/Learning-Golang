# 32. Observability với OpenTelemetry — Traces, Metrics, và Log Correlation

> Liên quan: [16-logging-config.md](16-logging-config.md) (đã dạy `log/slog`), [13-http-server.md](13-http-server.md). Dành cho C# developer: OpenTelemetry (OTel) tương đương `System.Diagnostics.Activity` + `OpenTelemetry.Extensions.Hosting` trong .NET — cùng một chuẩn CNCF, API gần như song song giữa 2 ngôn ngữ.

---

## 1. Vì sao OpenTelemetry quan trọng hơn nhiều so với 1 dòng bullet trong roadmap

`log/slog` (bài 16) trả lời câu hỏi "chuyện gì đã xảy ra ở đây". Nhưng khi một request đi qua 5 service khác nhau (API gateway → auth service → order service → payment service → notification service), câu hỏi thực sự cần trả lời là: **"request này đi qua những đâu, service nào chậm, lỗi xảy ra ở bước nào?"** — log riêng lẻ từng service không trả lời được câu này, bạn cần **distributed tracing**.

Đến 2025-2026, OpenTelemetry gần như là **lựa chọn mặc định duy nhất** cho observability trong hệ thống Go production — không còn tình trạng "mỗi công ty một chuẩn riêng" như thời Jaeger/Zipkin/proprietary APM cạnh tranh nhau. Vì vậy đây không phải kiến thức "biết thêm cho vui" mà là kỹ năng gần như bắt buộc ở mức mid-senior trở lên.

## 2. Ba trụ cột: Traces, Metrics, Logs

| Trụ cột | Trả lời câu hỏi | Go package |
|---|---|---|
| **Traces** | Request này đi qua đâu, mất bao lâu ở mỗi bước? | `go.opentelemetry.io/otel/trace` |
| **Metrics** | Tổng thể hệ thống đang khỏe không (request rate, error rate, latency percentile)? | `go.opentelemetry.io/otel/metric` |
| **Logs** | Chi tiết gì đã xảy ra tại một thời điểm cụ thể? | `log/slog` + OTel log bridge |

Ba trụ cột này nên được **liên kết bằng trace ID** — đây là phần quan trọng nhất và hay bị bỏ qua: một dòng log không có trace ID đính kèm thì khi debug production, bạn không biết log đó thuộc về request nào trong hàng nghìn request đang chạy song song.

## 3. Khởi tạo Tracer — Setup tối thiểu

```go
package platform

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer khởi tạo global TracerProvider, gửi trace tới OTel Collector qua OTLP/HTTP.
// Gọi 1 lần duy nhất trong main.go, tương đương AddOpenTelemetry() trong Program.cs của .NET.
func InitTracer(ctx context.Context, serviceName, collectorEndpoint string) (func(context.Context) error, error) {
    exporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint(collectorEndpoint),
        otlptracehttp.WithInsecure(), // bỏ trong production, dùng TLS
    )
    if err != nil {
        return nil, err
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName(serviceName),
        ),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter), // gửi trace theo batch, không block request path
        sdktrace.WithResource(res),
    )
    otel.SetTracerProvider(tp)

    return tp.Shutdown, nil // gọi trong graceful shutdown để flush trace còn lại trước khi thoát
}
```

```go
// main.go
shutdown, err := platform.InitTracer(ctx, "order-service", "otel-collector:4318")
if err != nil {
    logger.Error("failed to init tracer", "error", err)
    os.Exit(1)
}
defer shutdown(context.Background())
```

## 4. Tạo Span — Đơn vị đo lường cơ bản

Một **span** đại diện cho một đơn vị công việc (một hàm, một DB query, một HTTP call). Span có thể lồng nhau (span cha-con), tạo thành một **trace** hoàn chỉnh.

```go
import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("order-service") // đặt tên theo package/component

func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
    // Bắt đầu span mới, ctx trả về mang theo span này — PHẢI truyền ctx tiếp cho các hàm con
    ctx, span := tracer.Start(ctx, "OrderService.CreateOrder")
    defer span.End() // luôn End() để đóng span, giống defer Dispose() của Activity trong .NET

    span.SetAttributes(
        attribute.String("order.customer_id", input.CustomerID),
        attribute.Int("order.item_count", len(input.Items)),
    )

    // Truyền ctx xuống repository — nếu repository cũng tạo span, nó sẽ là con của span này
    order, err := s.repo.Save(ctx, input)
    if err != nil {
        span.RecordError(err)                          // ghi lại lỗi vào span
        span.SetStatus(codes.Error, "failed to save order")
        return nil, err
    }

    return order, nil
}
```

**Điểm khác biệt cốt lõi so với .NET:** trong .NET, `Activity.Current` là ambient (thread-static/AsyncLocal), tự động propagate qua `await`. Trong Go, **bạn phải tự tay truyền `context.Context`** qua mọi lời gọi hàm — không có ambient context ẩn. Đây chính là lý do [05-context.md](05-context.md) nhấn mạnh "luôn nhận `ctx` làm tham số đầu tiên" — OpenTelemetry là ví dụ rõ nhất cho việc tại sao quy tắc đó tồn tại.

## 5. Tự động instrument HTTP server và HTTP client

Viết span thủ công cho mọi hàm là không thực tế. Dùng middleware/wrapper có sẵn cho các thư viện phổ biến:

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

// Tự động tạo span cho mọi incoming HTTP request
handler := otelhttp.NewHandler(router, "http-server")
server := &http.Server{Addr: cfg.Addr, Handler: handler}
```

```go
// Tự động tạo span cho mọi outgoing HTTP call, và tự động propagate trace context
// qua HTTP header (traceparent) sang service tiếp theo — đây là cách distributed tracing
// "nối" các service lại thành 1 trace hoàn chỉnh
client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}
```

Không có bước propagate qua HTTP header này, mỗi service sẽ tạo ra trace riêng biệt không liên kết được với nhau — đây là lỗi setup phổ biến nhất khi mới làm quen OTel.

## 6. Correlate `log/slog` với Trace ID

Đây là phần thường bị bỏ sót nhưng mang lại giá trị thực tế lớn nhất: khi debug production, bạn muốn từ 1 dòng log nhảy thẳng sang xem toàn bộ trace của request đó.

```go
// Custom slog.Handler wrapper: tự động thêm trace_id/span_id vào mọi log record
// nếu context truyền vào có active span
type otelHandler struct {
    slog.Handler
}

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
    if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
        r.AddAttrs(
            slog.String("trace_id", span.SpanContext().TraceID().String()),
            slog.String("span_id", span.SpanContext().SpanID().String()),
        )
    }
    return h.Handler.Handle(ctx, r)
}

// Dùng ở mọi nơi có ctx: logger.InfoContext(ctx, "order created", "order_id", order.ID)
// -> log JSON output sẽ tự có "trace_id": "...", click vào trace_id đó trong Grafana/Jaeger
//    để xem toàn bộ hành trình request qua các service
```

## 7. Metrics — Đo lường tổng thể hệ thống

```go
import "go.opentelemetry.io/otel/metric"

var (
    meter          = otel.Meter("order-service")
    ordersCreated, _ = meter.Int64Counter("orders.created",
        metric.WithDescription("Tổng số order đã tạo"))
    orderLatency, _  = meter.Float64Histogram("orders.create.duration",
        metric.WithDescription("Thời gian xử lý tạo order"), metric.WithUnit("ms"))
)

func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
    start := time.Now()
    defer func() {
        orderLatency.Record(ctx, float64(time.Since(start).Milliseconds()))
    }()

    order, err := s.repo.Save(ctx, input)
    if err == nil {
        ordersCreated.Add(ctx, 1, metric.WithAttributes(
            attribute.String("status", "success"),
        ))
    }
    return order, err
}
```

Metrics rẻ hơn traces (không lưu chi tiết từng request) và phù hợp cho dashboard/alerting (Grafana + Prometheus), trong khi traces phù hợp cho debug từng request cụ thể — dùng cả hai, không thay thế lẫn nhau.

## 8. Sampling — Không trace 100% request trong production

Trace 100% traffic ở hệ thống có traffic lớn sẽ tốn chi phí lưu trữ khổng lồ. Dùng sampling:

```go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithResource(res),
    sdktrace.WithSampler(sdktrace.ParentBased(
        sdktrace.TraceIDRatioBased(0.1), // chỉ trace 10% request, giữ nguyên nếu request cha đã được trace
    )),
)
```

**Lưu ý quan trọng:** luôn trace 100% request bị lỗi (error) bất kể sampling rate — nhiều setup dùng "tail-based sampling" ở tầng Collector để quyết định giữ/bỏ trace *sau khi* đã biết request đó có lỗi hay không, thay vì quyết định ngẫu nhiên từ đầu.

---

## Bảng tổng kết

| Khái niệm | Vai trò | Package Go |
|---|---|---|
| `TracerProvider` | Khởi tạo 1 lần, cấu hình exporter + sampling | `sdktrace` |
| `Span` | Đơn vị đo lường 1 đơn vị công việc | `trace.Tracer.Start()` |
| `otelhttp` | Auto-instrument HTTP server/client, tự propagate trace context | `contrib/instrumentation/net/http/otelhttp` |
| Trace-log correlation | Thêm `trace_id`/`span_id` vào mọi log record | Custom `slog.Handler` wrapper |
| `Meter` | Đo counter/histogram cho dashboard, alerting | `otel/metric` |
| Sampling | Kiểm soát chi phí lưu trữ khi traffic lớn | `sdktrace.WithSampler` |

---

## Bài tập

### Exercise 1
Thêm OpenTelemetry vào REST API đã xây ở [22-mini-ecommerce-project.md](22-mini-ecommerce-project.md) Giai đoạn 4: setup `TracerProvider` xuất trace ra console (`stdouttrace` exporter, không cần Collector thật), tạo span cho mỗi request ở handler và mỗi query ở repository.

### Exercise 2
Viết `otelHandler` wrapper cho `log/slog` như mục 6, xác nhận bằng cách gọi 1 request và kiểm tra log JSON output có field `trace_id` khớp với trace hiển thị trên console exporter.

### Exercise 3
Chạy [Jaeger](https://www.jaegertracing.io/) bằng Docker Compose (`docker run -p 16686:16686 -p 4318:4318 jaegertracing/all-in-one`), trỏ OTLP exporter tới nó, tạo vài request có lỗi giả lập (`return errors.New(...)`) và tìm chúng trên Jaeger UI qua filter theo tag `error=true`.

---

**Liên quan:** [16-logging-config.md](16-logging-config.md), [05-context.md](05-context.md), [13-http-server.md](13-http-server.md), [30-language-trends-review-2026.md](30-language-trends-review-2026.md)
</content>
</invoke>
