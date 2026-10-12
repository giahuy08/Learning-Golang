# Go Developer Roadmap — From Zero to Production

> **Your profile:** Complete beginner → Backend APIs · CLI Tools · Cloud & DevOps · Open Source  
> **Estimated total time:** 12–18 months of consistent practice (1–2 hrs/day)

---

## How to Use This Roadmap

Each phase builds on the last. Don't skip phases — Go rewards understanding fundamentals deeply. At the end of each phase, complete the **milestone project** before moving on. Mark your progress, track what you build, and revisit earlier sections when something doesn't click later.

---

## Phase 1 — Environment & Go Basics (Weeks 1–3)

### Goals
Understand what Go is, why it's used, and write your first real programs.

### Setup
- Install Go: https://go.dev/dl/ (always install the latest stable version)
- Editor: **VS Code** with the official Go extension (`golang.go`) — gives you autocomplete, linting, formatting on save
- Learn the Go Playground: https://go.dev/play/ — run Go snippets in the browser without installing anything
- Understand the workspace: `GOPATH`, `GOROOT`, and modules (`go mod init`)

### Core Concepts
- **Variables & types:** `var`, `:=`, basic types (`int`, `float64`, `string`, `bool`)
- **Constants:** `const`, `iota`
- **Functions:** parameters, multiple return values, named returns
- **Control flow:** `if/else`, `for` (Go's only loop), `switch`
- **Arrays & slices:** the difference matters — slices are dynamic, arrays are fixed
- **Maps:** key-value store, zero values, deleting keys
- **Pointers:** `*` and `&`, when and why Go uses them
- **Structs:** custom types, field access, struct literals
- **Methods:** attaching behavior to types (`func (t Type) Method()`)
- **Interfaces:** implicit satisfaction, the power of `interface{}`
- **Packages:** `import`, `fmt`, `math`, `strings`, `strconv`
- **Error handling:** Go's `error` type, `errors.New()`, `fmt.Errorf()`

### Resources
- **A Tour of Go** (interactive, free): https://go.dev/tour/ — complete this first
- **Go by Example** (free): https://gobyexample.com/ — reference for every concept
- **Book:** *The Go Programming Language* by Donovan & Kernighan — the definitive text

### Milestone Project 🎯
Build a **CLI calculator** that takes two numbers and an operator (`+`, `-`, `*`, `/`) as command-line arguments and prints the result. Handle division by zero gracefully with a proper error.

---

## Phase 2 — Go's Unique Strengths (Weeks 4–6)

### Goals
Master the features that make Go different from every other language.

### Concurrency — Go's Killer Feature
- **Goroutines:** `go func()` — lightweight threads managed by the Go runtime
- **Channels:** `chan`, send/receive, buffered vs unbuffered channels
- **`select` statement:** wait on multiple channels
- **`sync` package:** `WaitGroup`, `Mutex`, `Once`
- **Common patterns:** fan-out/fan-in, pipelines, worker pools
- **Race conditions:** use `go run -race` to detect them

### Error Handling Done Right
- Wrapping errors: `fmt.Errorf("context: %w", err)`
- Unwrapping: `errors.Is()`, `errors.As()`
- Custom error types (implement the `error` interface)
- Sentinel errors vs. typed errors — when to use each

### Deeper Type System
- **Embedding:** struct composition over inheritance
- **Type assertions:** `v, ok := i.(Type)`
- **Type switches:** `switch v := i.(type)`
- **Generics (Go 1.18+):** `[T any]`, type constraints, when to actually use generics
- **Stringer interface:** `fmt.Stringer` for pretty printing

### Go Tooling
- `go fmt` — always run this (or enable format-on-save)
- `go vet` — catches common bugs
- `go test` — built-in testing framework
- `go build` / `go run`
- `golangci-lint` — industry-standard linter: https://golangci-lint.run/
- `go doc` — read documentation from the terminal

### Resources
- **Concurrency in Go** (book) by Katherine Cox-Buday — best resource on goroutines/channels
- Rob Pike's "Concurrency is not Parallelism" (talk): https://go.dev/blog/waza-talk

### Milestone Project 🎯
Build a **concurrent URL checker** — read a list of URLs from a file, check each one in parallel using goroutines and channels, and print which ones are up/down. Limit concurrency to N workers using a worker pool.

---

## Phase 3 — Standard Library & Testing (Weeks 7–9)

### Goals
Know the standard library well enough that you rarely need external packages for common tasks.

### Key Standard Library Packages
| Package | What it does |
|---|---|
| `fmt` | Formatted I/O |
| `os` | Files, environment variables, process control |
| `io` / `io/fs` | Readers, writers, filesystem abstraction |
| `bufio` | Buffered reading/writing |
| `strings` / `bytes` | String and byte manipulation |
| `strconv` | String ↔ number conversions |
| `time` | Time, duration, formatting |
| `math/rand` | Random numbers |
| `encoding/json` | JSON marshal/unmarshal |
| `encoding/csv` | CSV reading/writing |
| `net/http` | HTTP client and server (surprisingly powerful) |
| `net/url` | URL parsing |
| `log` / `log/slog` | Logging (`slog` is the modern structured logger) |
| `context` | Cancellation, deadlines, request-scoped values |
| `regexp` | Regular expressions |
| `sort` | Sorting slices and custom types |
| `flag` | CLI argument parsing |
| `filepath` | Cross-platform file path operations |
| `testing` | Unit tests, benchmarks, fuzzing |

### Testing in Go
- `_test.go` files and the `testing.T` type
- Table-driven tests — Go's idiomatic testing pattern
- `t.Run()` for subtests
- `testing.B` for benchmarks (`go test -bench=.`)
- `httptest` package for testing HTTP handlers
- Mocking with interfaces — no mocking library needed
- Test coverage: `go test -cover`

### Resources
- **Go standard library docs:** https://pkg.go.dev/std — bookmark this
- **Go blog:** https://go.dev/blog/ — official articles on idioms and best practices

### Milestone Project 🎯
Build a **file processing CLI tool** — reads a CSV of records, filters/transforms them based on flags, writes output as JSON. Include a full test suite with table-driven tests and >80% coverage.

---

## Phase 4 — Backend APIs (Weeks 10–14)

### Goals
Build production-quality HTTP APIs in Go.

### net/http Deep Dive
- `http.Handler` interface and `http.HandlerFunc`
- Routing with `http.ServeMux` (Go 1.22+ has pattern matching built-in)
- Middleware pattern: wrapping handlers
- Reading request body, query params, path params
- Writing JSON responses, setting status codes
- Graceful shutdown with `context`

### The Go Web Ecosystem
- **Router:** `chi` (https://github.com/go-chi/chi) — lightweight, idiomatic, great middleware support
- **Validation:** `go-playground/validator`
- **Database:** `database/sql` + `pgx` (PostgreSQL driver)
- **ORM/Query builder:** `sqlc` (generate type-safe Go from SQL) or `sqlx`
- **Migrations:** `golang-migrate`
- **Config:** `viper` or `godotenv`
- **Logging:** `log/slog` (stdlib) or `zerolog` / `zap` for high performance

### API Design Patterns
- RESTful resource design
- Request/response structs with JSON tags
- Input validation and sanitization
- Consistent error response format
- Pagination patterns
- Authentication: JWT with `golang-jwt/jwt`
- Middleware: logging, auth, rate limiting, CORS

### gRPC (for microservices)
- Protocol Buffers (protobuf): defining `.proto` files
- `google.golang.org/grpc` package
- Code generation with `protoc`
- Unary vs. streaming RPCs
- gRPC-Gateway for REST ↔ gRPC translation

### ConnectRPC — Modern alternative to raw gRPC (2024-2026 trend)
- **[connectrpc.com](https://connectrpc.com/)** (Buf's `connect-go`) has become the default choice for new internal services at many companies over raw `grpc-go`
- Same `.proto` definitions and codegen workflow as gRPC, but the generated server/client speak **gRPC, gRPC-Web, and plain HTTP/1.1+JSON** simultaneously — no separate gRPC-Gateway needed for REST/browser compatibility
- Simpler middleware model (plain `http.Handler`-compatible) — no special gRPC interceptor API to learn
- When to still choose raw `grpc-go`: you need bidirectional streaming performance tuning grpc-go exposes directly, or you're integrating with a polyglot system where every other service already speaks pure gRPC
- **Rule of thumb for 2026:** start new Go services with ConnectRPC unless you have a specific reason to need raw gRPC's lower-level control

### Resources
- **Let's Go** by Alex Edwards — best practical book for Go web dev: https://lets-go.alexedwards.net/
- **Let's Go Further** (advanced APIs, auth, deployment) — same author
- **Official gRPC Go tutorial:** https://grpc.io/docs/languages/go/

### Milestone Project 🎯
Build a **REST API for a task manager** — CRUD for tasks and users, JWT authentication, PostgreSQL storage with `sqlc`, structured logging, graceful shutdown, and a full integration test suite.

---

## Phase 5 — CLI Tools (Weeks 15–17)

### Goals
Build professional CLI tools — the kind you'd publish and use daily.

### CLI Fundamentals
- `flag` package (stdlib) for simple tools
- **Cobra** (https://github.com/spf13/cobra) — the standard for complex CLIs (used by kubectl, Hugo, Docker)
  - Commands, subcommands, flags, persistent flags
  - Cobra + Viper for config file support
- **urfave/cli** — simpler alternative to Cobra

### Great CLI UX
- **Bubbletea** (https://github.com/charmbracelet/bubbletea) — TUI framework for interactive terminal apps
- **Lipgloss** — terminal styling
- **Glamour** — render markdown in the terminal
- **Spinner / progress bars:** `briandowns/spinner`, `schollz/progressbar`
- **Table output:** `olekukonko/tablewriter`
- Color output with `fatih/color`

### CLI Best Practices
- `--help` should always work
- Exit codes: 0 = success, non-zero = error
- Read from stdin, write to stdout, errors to stderr
- Support `--json` / `--output` flags for machine-readable output
- Shell completion (`cobra` generates this automatically)
- Cross-compile: `GOOS=linux GOARCH=amd64 go build`
- Distribute as a single binary — Go's biggest advantage for CLI tools

### Resources
- **Charm CLI tools blog:** https://charm.sh/ — great inspiration and libraries
- `kubectl` source code — study how a complex CLI is structured

### Milestone Project 🎯
Build a **GitHub CLI tool** (`gh`-style) that lists your repos, opens issues, and shows PR status using the GitHub API. Include interactive mode with Bubbletea, JSON output flag, and shell completion.

### Optional — AI/LLM Engineering in Go (2025-2026 growth area)
Go has become one of the most common languages for building AI agent tooling and CLIs — the tool you're using right now to read this roadmap is itself a Go-ecosystem-adjacent CLI agent. This isn't required for the backend/CLI/DevOps track, but it's where Go is visibly gaining ground, and it reuses everything you just learned in this phase (CLI UX, HTTP clients, JSON, concurrency for streaming responses).
- Official/community Go SDKs for LLM providers (Anthropic, OpenAI) — typed request/response structs, streaming via channels
- **MCP (Model Context Protocol)** — writing an MCP server in Go (`mcp-go` or similar) that exposes tools/resources to an LLM client; a natural extension of the CLI tool patterns from this phase
- Streaming responses: server-sent events / chunked HTTP responses consumed with `bufio.Scanner`, fed into a channel for the UI layer to render incrementally
- See [33-ai-llm-engineering-go.md](33-ai-llm-engineering-go.md) for a hands-on walkthrough (optional milestone: extend your GitHub CLI tool above with an `ask` subcommand that summarizes an issue thread via an LLM API)

---

## Phase 6 — Cloud & DevOps (Weeks 18–22)

### Goals
Understand how Go code runs in cloud environments and write infrastructure tooling.

### Docker & Containers
- Write a multi-stage `Dockerfile` for a Go app (minimize image size — Go binaries are self-contained)
- Distroless / scratch base images
- `docker-compose` for local development
- Environment variable configuration

### Kubernetes (The Big One)
- Kubernetes concepts: Pods, Deployments, Services, ConfigMaps, Secrets
- **client-go** (https://github.com/kubernetes/client-go) — official K8s Go client
  - List, watch, and manage K8s resources from Go
  - Informers and the controller pattern
- **controller-runtime** — framework for building K8s operators
- **kubebuilder** — scaffold Kubernetes operators
- Custom Resource Definitions (CRDs): extend Kubernetes with your own types
- Writing admission webhooks

### Infrastructure as Code
- **Pulumi** — infrastructure as code using real Go (not YAML)
- Terraform provider development with the Go SDK
- AWS/GCP/Azure SDKs for Go

### Observability
- **OpenTelemetry Go SDK** — the de facto standard for tracing/metrics/logs by 2025-2026, near-mandatory at production scale. See [32-observability-opentelemetry.md](32-observability-opentelemetry.md) for a full walkthrough (traces, spans, context propagation, exporters, correlating traces with `log/slog`)
- Prometheus metrics: `prometheus/client_golang`
- Structured logging that feeds into log aggregators — correlate `log/slog` records with OpenTelemetry trace IDs so a single request can be followed across logs, metrics, and traces

### Cloud-Native Patterns
- 12-factor app principles
- Health check endpoints (`/healthz`, `/readyz`)
- Graceful shutdown handling `SIGTERM`
- Circuit breakers with `sony/gobreaker`
- Retry logic with exponential backoff

### Resources
- **Programming Kubernetes** by Harshal Patil & Michael Hausenblas — deep dive on client-go and operators
- **Kubernetes docs:** https://kubernetes.io/docs/
- **Pulumi Go docs:** https://www.pulumi.com/docs/languages-sdks/go/

### Milestone Project 🎯
Build a **Kubernetes operator** that watches a custom resource (e.g., `WebApp`) and automatically creates a Deployment + Service for it. Deploy it to a local cluster with `kind` or `minikube`.

### Microservices Architecture Patterns
gRPC/ConnectRPC above cover the *how* of inter-service communication; this section is the *when* and *what breaks*. See [34-microservices-patterns.md](34-microservices-patterns.md) for the full walkthrough: when to split a monolith at all, splitting by business boundary vs. technical layer, sync (REST/gRPC) vs. async (Kafka/NATS/RabbitMQ) communication, resilience (timeouts, circuit breakers, retry with backoff), distributed data patterns (saga, outbox), and multi-service local dev with `docker-compose`. Best tackled after your Phase 4 milestone API is solid — splitting too early teaches the wrong lessons.

---

## Phase 7 — Open Source Contribution (Weeks 23–26)

### Goals
Read, understand, and contribute to real Go projects.

### Before You Contribute
- Study Go project conventions: `CONTRIBUTING.md`, issue labels, PR templates
- Understand Go module versioning (semver, `go.mod`, `go.sum`)
- Learn `git` workflows: forks, branches, rebasing
- Read: https://go.dev/doc/contribute

### Where to Start
- **Good first issue** labels on GitHub — filter by language: Go
- Go standard library: https://github.com/golang/go — even docs fixes count
- CNCF projects (Cloud Native): https://contribute.cncf.io/ — many are written in Go (Prometheus, Jaeger, Flux)
- `awesome-go` list: https://github.com/avelino/awesome-go — find projects you use and contribute back

### How to Read a Large Go Codebase
1. Start with `main.go` or `cmd/` — find the entry point
2. Read `go.mod` — understand dependencies
3. Look at the test files — they show intended behavior
4. Use `go doc` and `gopls` (the Go language server) to navigate
5. Run `grep -r "FunctionName"` to find usages

### Writing Your Own Library
- Package design: small, focused, composable
- Document every exported symbol with comments
- Semantic versioning with GitHub releases
- CI with GitHub Actions (`golangci-lint` + `go test`)
- Write a great `README.md` with examples

### Resources
- **100 Go Mistakes** by Teiva Harsanyi — read before contributing, avoid common pitfalls
- **Effective Go:** https://go.dev/doc/effective_go — official style guide
- **Go Code Review Comments:** https://go.dev/wiki/CodeReviewComments

### Milestone Project 🎯
Make at least **3 meaningful open source contributions** — a bug fix, a feature addition, and a documentation improvement — in projects you actually use.

---

## Phase 8 — Advanced Go (Ongoing)

Once you've completed phases 1–7, go deeper on these topics as needed:

### Performance
- Profiling with `pprof`: CPU, memory, goroutine profiles
- `go test -bench` and benchmark analysis
- Escape analysis: `go build -gcflags="-m"` — understand heap vs. stack
- Reducing allocations: `sync.Pool`, pre-allocated slices
- Memory layout: struct field ordering for alignment

### Advanced Concurrency
- `errgroup` for concurrent tasks with error collection
- `semaphore` for resource limiting
- Lock-free data structures
- The `atomic` package

### Advanced Patterns
- Functional options pattern
- Builder pattern in Go
- Event-driven architecture with channels
- Plugin system with Go plugins (`.so` files) or RPC

### Security
- Input sanitization, SQL injection prevention
- Secrets management (never hardcode, use Vault or cloud secrets)
- TLS configuration with `crypto/tls`
- `gosec` — security linter: https://github.com/securego/gosec

---

## Thực chiến & phỏng vấn (bổ sung — làm song song từ Phase 4)

Làm theo thứ tự, áp dụng thẳng vào mini-ecommerce ([22](22-mini-ecommerce-project.md)):
1. [35 — DB transaction, locking](35-database-transactions-locking.md) → [36 — idempotency, retry, rate limit, cache](36-idempotency-and-reliability.md)
2. [37 — Concurrency bug lab](37-concurrency-bug-lab.md) → [38 — Integration test với DB thật](38-integration-testing-testcontainers.md)
3. [39 — Docker, compose, CI](39-docker-deployment-ci.md)
4. [40 — Chuẩn bị phỏng vấn](40-interview-prep.md) (bắt đầu từ tuần ~10, ôn đều mỗi tuần)

---

## Learning Resources Summary

### Free
| Resource | URL | Best for |
|---|---|---|
| A Tour of Go | https://go.dev/tour/ | Phase 1 |
| Go by Example | https://gobyexample.com/ | Reference |
| Go Playground | https://go.dev/play/ | Quick experiments |
| Go Blog | https://go.dev/blog/ | Idiomatic patterns |
| Effective Go | https://go.dev/doc/effective_go | Style |
| pkg.go.dev | https://pkg.go.dev/ | All docs |

### Books (Paid)
| Book | Author | Best for |
|---|---|---|
| The Go Programming Language | Donovan & Kernighan | Fundamentals |
| Concurrency in Go | Katherine Cox-Buday | Phase 2 |
| Let's Go | Alex Edwards | Phase 4 |
| Let's Go Further | Alex Edwards | Phase 4+ |
| 100 Go Mistakes | Teiva Harsanyi | Phase 7+ |
| Programming Kubernetes | Patil & Hausenblas | Phase 6 |

### Communities
- **Gophers Slack:** https://gophers.slack.com/ — largest Go community
- **r/golang:** https://reddit.com/r/golang
- **Go Forum:** https://forum.golangbridge.org/
- **GopherCon talks** (YouTube) — conference talks from Go experts

---

## Suggested Weekly Schedule

| Week | Focus |
|---|---|
| 1–3 | Phase 1: Basics + Tour of Go |
| 4–6 | Phase 2: Concurrency + error handling |
| 7–9 | Phase 3: Standard library + testing |
| 10–14 | Phase 4: Backend APIs |
| 15–17 | Phase 5: CLI tools |
| 18–22 | Phase 6: Cloud & DevOps |
| 23–26 | Phase 7: Open source |
| 27+ | Phase 8: Advanced topics (ongoing) |

---

## The "Best Go Developer" Checklist

You're becoming an expert Go developer when you:

- [ ] Understand *why* Go makes the tradeoffs it does (no classes, no exceptions, explicit errors)
- [ ] Write idiomatic Go that other Gophers recognize as clean
- [ ] Know when to use goroutines and when they'd be overkill
- [ ] Can design a package API that's easy to use and hard to misuse
- [ ] Read error messages and stack traces without fear
- [ ] Profile and optimize Go code, not just write it
- [ ] Navigate a large unknown Go codebase within minutes
- [ ] Have shipped something people actually use

---

*Roadmap generated June 2026. Go version referenced: Go 1.22+*
