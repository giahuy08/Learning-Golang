# 33. AI/LLM Engineering bằng Go — MCP Servers, Streaming, và Agent Tooling

> Optional, không bắt buộc cho track backend API/CLI/DevOps chính. Liên quan: Phase 5 (CLI Tools) trong [golang-roadmap.md](golang-roadmap.md), [13-http-server.md](13-http-server.md), [04-goroutines-channels.md](04-goroutines-channels.md), [31-iterators-and-range-over-func.md](31-iterators-and-range-over-func.md). Đây là mảng Go đang giành thị phần rõ rệt nhất 2025-2026 — nhiều CLI/agent tool phổ biến (bao gồm cả công cụ bạn dùng để đọc tài liệu này) thuộc hệ sinh thái Go/CLI-tooling.

---

## 1. Vì sao Go phù hợp cho mảng này

LLM application code thường bị mặc định viết bằng Python (do hệ sinh thái ML/data science), nhưng phần **hạ tầng xung quanh LLM** — CLI tool gọi API, agent orchestration, MCP server expose tool cho LLM dùng, proxy/gateway giữa client và LLM provider — lại là đúng thế mạnh truyền thống của Go: single binary dễ deploy, concurrency tốt cho streaming response, type safety khi parse JSON response phức tạp. Đây là lý do nhiều công cụ AI-agent/CLI mới (2024-2026) chọn Go thay vì Python hay Node.

## 2. Gọi LLM API — HTTP client cơ bản trước, SDK sau

Trước khi dùng SDK, hiểu được request/response thô giúp bạn debug dễ hơn khi SDK "làm phép" gì đó không rõ ràng:

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type Message struct {
    Role    string `json:"role"`    // "user" | "assistant" | "system"
    Content string `json:"content"`
}

type ChatRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
    Stream   bool      `json:"stream"`
}

type ChatResponse struct {
    Content []struct {
        Text string `json:"text"`
    } `json:"content"`
}

func Chat(ctx context.Context, apiKey string, req ChatRequest) (*ChatResponse, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
        "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("x-api-key", apiKey)
    httpReq.Header.Set("anthropic-version", "2023-06-01")
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("call LLM API: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("LLM API returned status %d", resp.StatusCode)
    }

    var chatResp ChatResponse
    if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    return &chatResp, nil
}
```

Trong thực tế, dùng SDK chính thức (`github.com/anthropics/anthropic-sdk-go`, `github.com/openai/openai-go`) thay vì tự viết — chúng xử lý retry, rate limit backoff, streaming parsing đúng chuẩn. Nhưng hiểu được lớp HTTP thô ở trên giúp bạn đọc lỗi SDK trả về mà không cần đoán mò.

## 3. Streaming response — Nơi Go's concurrency tỏa sáng

LLM API thường trả kết quả dạng **Server-Sent Events (SSE)** — từng chunk nhỏ gửi liên tục thay vì đợi toàn bộ response. Đây là chỗ pattern `bufio.Scanner` + channel (đã học ở [04-goroutines-channels.md](04-goroutines-channels.md)) áp dụng trực tiếp:

```go
func ChatStream(ctx context.Context, apiKey string, req ChatRequest) (<-chan string, <-chan error) {
    chunks := make(chan string)
    errs := make(chan error, 1)

    go func() {
        defer close(chunks)
        defer close(errs)

        req.Stream = true
        body, _ := json.Marshal(req)

        httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
            "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
        if err != nil {
            errs <- err
            return
        }
        httpReq.Header.Set("x-api-key", apiKey)
        httpReq.Header.Set("anthropic-version", "2023-06-01")
        httpReq.Header.Set("Content-Type", "application/json")
        httpReq.Header.Set("Accept", "text/event-stream")

        resp, err := http.DefaultClient.Do(httpReq)
        if err != nil {
            errs <- err
            return
        }
        defer resp.Body.Close()

        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            if !strings.HasPrefix(line, "data: ") {
                continue
            }
            data := strings.TrimPrefix(line, "data: ")

            var event struct {
                Delta struct {
                    Text string `json:"text"`
                } `json:"delta"`
            }
            if err := json.Unmarshal([]byte(data), &event); err != nil {
                continue // bỏ qua event không parse được (heartbeat, event type khác)
            }

            select {
            case chunks <- event.Delta.Text:
            case <-ctx.Done(): // caller huỷ giữa chừng — dừng ngay, không leak goroutine
                return
            }
        }
    }()

    return chunks, errs
}
```

```go
// Sử dụng: in từng chunk ra terminal ngay khi nhận được, giống hiệu ứng "typing" của ChatGPT
chunks, errs := ChatStream(ctx, apiKey, req)
for chunk := range chunks {
    fmt.Print(chunk)
}
if err := <-errs; err != nil {
    log.Fatal(err)
}
```

**Ghi chú:** đây cũng là ứng dụng thực tế của range-over-func iterators (bài 31) nếu bạn muốn API sạch hơn — viết `ChatStream` trả về `iter.Seq[string]` thay vì channel, loại bỏ hoàn toàn nhu cầu quản lý channel/goroutine thủ công phía caller.

## 4. MCP (Model Context Protocol) — Expose "tool" cho LLM dùng

MCP là chuẩn mở (do Anthropic khởi xướng, giờ được nhiều LLM client hỗ trợ) để LLM gọi ra các "tool" bên ngoài — đọc file, query database, gọi API nội bộ — một cách chuẩn hoá thay vì mỗi ứng dụng tự định nghĩa function-calling riêng. Viết MCP server bằng Go tận dụng đúng thế mạnh: single binary, dễ deploy như một subprocess mà LLM client spawn ra.

```go
package main

import (
    "context"
    "fmt"

    "github.com/modelcontextprotocol/go-sdk/mcp" // tên package minh hoạ — kiểm tra SDK chính thức mới nhất
)

type WeatherArgs struct {
    City string `json:"city" jsonschema:"required,description=Tên thành phố"`
}

func main() {
    server := mcp.NewServer("weather-server", "1.0.0")

    // Đăng ký một tool: LLM có thể "nhìn thấy" và gọi hàm này khi cần thông tin thời tiết
    server.AddTool(mcp.Tool{
        Name:        "get_weather",
        Description: "Lấy thông tin thời tiết hiện tại của một thành phố",
    }, func(ctx context.Context, args WeatherArgs) (string, error) {
        // Gọi API thời tiết thật, hoặc query DB nội bộ...
        return fmt.Sprintf("Thời tiết ở %s: 28°C, nắng nhẹ", args.City), nil
    })

    server.Run() // giao tiếp qua stdio với LLM client (Claude Desktop, IDE, v.v.)
}
```

Cấu trúc này rất giống pattern `handler → service` đã học ở [10-project-structure.md](10-project-structure.md): mỗi tool là 1 "handler" nhận input đã validate (qua JSON schema), gọi business logic, trả kết quả dạng string/structured content cho LLM diễn giải tiếp.

## 5. Agent loop cơ bản — Vòng lặp gọi tool

Một "agent" về cơ bản là vòng lặp: gọi LLM → LLM yêu cầu gọi tool → chạy tool → gửi kết quả lại cho LLM → lặp lại đến khi LLM trả lời cuối cùng.

```go
func RunAgent(ctx context.Context, userPrompt string, tools map[string]ToolFunc) (string, error) {
    messages := []Message{{Role: "user", Content: userPrompt}}

    for range 10 { // giới hạn số vòng lặp — tránh agent chạy vô hạn nếu LLM cứ gọi tool liên tục (Go 1.22+ range over int)
        resp, err := Chat(ctx, apiKey, ChatRequest{Model: "claude-sonnet", Messages: messages})
        if err != nil {
            return "", err
        }

        toolCall, ok := extractToolCall(resp)
        if !ok {
            return resp.Content[0].Text, nil // LLM trả lời cuối cùng, không cần gọi tool nữa
        }

        fn, exists := tools[toolCall.Name]
        if !exists {
            return "", fmt.Errorf("unknown tool: %s", toolCall.Name)
        }

        result, err := fn(ctx, toolCall.Args)
        if err != nil {
            result = fmt.Sprintf("error: %v", err) // trả lỗi cho LLM tự quyết định retry hay báo user
        }

        messages = append(messages,
            Message{Role: "assistant", Content: toolCall.Raw},
            Message{Role: "user", Content: result}, // "tool result" gửi lại dưới dạng message tiếp theo
        )
    }

    return "", fmt.Errorf("agent exceeded max iterations")
}
```

Đây chính là lý do error handling (bài 03), context cancellation (bài 05), và JSON parsing (bài 12) đều là nền tảng bắt buộc trước khi làm mảng này — agent loop chỉ là ghép nối những gì bạn đã học, không phải kiến thức hoàn toàn mới.

---

## 6. Milestone Project (optional)

Mở rộng GitHub CLI tool ở Phase 5 milestone: thêm subcommand `mycli ask "tóm tắt issue #42"` — tool gọi GitHub API lấy nội dung issue, gửi vào LLM API (streaming), in kết quả tóm tắt ra terminal theo hiệu ứng "typing" (dùng pattern streaming ở mục 3). Nâng cao: biến chính GitHub CLI tool đó thành một MCP server để bạn có thể hỏi Claude Desktop/IDE "liệt kê PR đang mở của tôi" và nó tự gọi tool này.

---

## Tóm tắt

Go không phải ngôn ngữ để viết model hay training pipeline — đó vẫn là sân của Python. Nhưng cho **hạ tầng xung quanh LLM** (CLI tools, MCP servers, agent orchestration, API gateway/proxy), Go đang là lựa chọn ngày càng phổ biến nhờ đúng những gì bạn đã học trong roadmap này: HTTP client/server (bài 13), streaming qua channel (bài 04), JSON (bài 12), context cancellation (bài 05), và single-binary deployment (bài 10). Đây không phải một track riêng biệt cần học lại từ đầu — nó là ứng dụng thực tế của toàn bộ nền tảng Go bạn đang xây.

**Liên quan:** [golang-roadmap.md](golang-roadmap.md) Phase 5, [31-iterators-and-range-over-func.md](31-iterators-and-range-over-func.md), [30-language-trends-review-2026.md](30-language-trends-review-2026.md)
</content>
</invoke>
