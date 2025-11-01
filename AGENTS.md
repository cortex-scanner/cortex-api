## Build, Lint, and Test

- **Build:** `go-task build`
- **Lint:** `go-task lint`
- **Lint (fix):** `go-task lint:fix`
- **Generate Mocks:** `go-task genmock`
- **Test:** `go-task test`
- **Test (single test):** `go test -v ./path/to/package -run TestName`
- **Run:** `go-task run`

## Code Style

- **Imports:** Use `goimports` to format imports.
- **Formatting:** Use `gofmt` for code formatting.
- **Types:** Use static typing wherever possible.
- **Naming Conventions:** Follow Go's idiomatic naming conventions (e.g., `camelCase` for variables, `PascalCase` for exported functions and types).
- **Error Handling:** Use `if err != nil` for error handling.
- **Logging:** Use the structured logger from the `logging` package. Provide context with the log message if applicable.
- **Mocks:** Use `mockery` with the `matryer` format to generate mocks for interfaces. Do not use any other mocking framework.
- **Testing**: Use `testify` for assertions. Before tests can be executed, mocks must be generated.
- **Documentation** Use `godoc` format to document code. Be precise and short. Assume the target audience is an experienced go developer.