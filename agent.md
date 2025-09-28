# agent.md

## Golang Agent Ruleset

### Always Do The Following

- Always use `go fmt` to format your code consistently.
- Always use functional coding paradigms where possible.
- Always try to eliminate the need for classes.
- Always use meaningful and descriptive variable, function, and package names.
- Always follow Go naming conventions: use camelCase for unexported names, PascalCase for exported names.
- Always use short variable names in limited scopes (e.g., `i` for loop index, `r` for Reader).
- Always declare variables close to where they are first used.
- Always use zero values when appropriate instead of explicit initialization.
- Always prefer composition over inheritance.
- Always use interfaces to define contracts and enable testing.
- Always define interfaces at the point of use, not the point of implementation.
- Always keep interfaces small and focused (prefer many small interfaces).
- Always use context.Context for cancellation, timeouts, and request-scoped values.
- Always pass context as the first parameter in function signatures.
- Always use defer for resource cleanup (files, connections, locks).
- Always use gofmt, go vet, and golint before committing code.
- Always handle the happy path first, then error cases.
- Always use early returns to reduce nesting and improve readability.
- Always use constants for magic numbers and strings.
- Always group related constants and variables in blocks.
- Always use receiver names consistently within a type (choose one 1-2 letter abbreviation).
- Always use pointer receivers for methods that modify the receiver or for large structs.
- Always use value receivers for small immutable data or when the method doesn't modify the receiver.

### Never Do The Following

- Never ignore errors returned by functions.
- Never use panic() except for truly unrecoverable situations.
- Never use init() functions unless absolutely necessary.
- Never use global variables unless they are constants or absolutely necessary.
- Never use empty catch-all interfaces (interface{}) unless working with reflection.
- Never mutate slices or maps passed as parameters without clear documentation.
- Never create goroutines without considering their lifecycle and termination.
- Never use naked returns in functions longer than a few lines.
- Never put business logic in main() function.
- Never ignore context cancellation in long-running operations.
- Never use strings for enums; use typed constants instead.
- Never use fmt.Sprintf for simple string concatenation.
- Never create packages that import their subpackages (circular imports).
- Never export types, functions, or variables that don't need to be public.
- Never use map[string]interface{} when you can use a struct.
- Never modify shared state without proper synchronization.
- Never use goroutines for simple sequential operations.
- Never hardcode configuration values; use environment variables or config files.

## Code Organization and Structure Guidelines

### Package Structure

- Always organize code into logical, cohesive packages.
- Always keep package names short, lowercase, and without underscores.
- Always make package names descriptive of their functionality.
- Always place the main package in cmd/ directory for applications.
- Always use internal/ directory for private packages that shouldn't be imported externally.
- Always group related functionality in the same package.
- Always avoid deep package hierarchies; prefer flat structure when possible.

### File Organization

- Always use meaningful file names that reflect their content.
- Always limit file size to maintain readability (generally under 500 lines).
- Always group related types and functions in the same file.
- Always place tests in the same package with _test.go suffix.
- Always use separate files for different concerns (handlers, models, services).

### Import Organization

- Always group imports into standard library, third-party, and local packages.
- Always use goimports tool to automatically organize imports.
- Always use blank imports only when necessary (for side effects).
- Always avoid dot imports except in test files when appropriate.

## Error Handling Patterns

### Error Creation and Wrapping

- Always create custom error types for domain-specific errors.
- Always use fmt.Errorf with %w verb to wrap errors and maintain the error chain.
- Always provide context in error messages (what operation failed, why it failed).
- Always use errors.Is() and errors.As() for error checking and unwrapping.
- Always return errors as the last return value in functions.

### Error Handling Strategy

- Always handle errors at the appropriate level of abstraction.
- Always log errors at the point where they can't be handled further up the call stack.
- Always use structured logging with appropriate log levels.
- Always include relevant context in error logs (request ID, user ID, etc.).
- Always fail fast when encountering unrecoverable errors.

### Validation and Input Handling

- Always validate input parameters at the beginning of functions.
- Always return specific error messages for validation failures.
- Always sanitize user input to prevent injection attacks.

## Testing Requirements

### Test Structure and Organization

- Always write tests for all public functions and methods.
- Always use table-driven tests for testing multiple scenarios.
- Always use descriptive test names that explain what is being tested.
- Always organize tests with Arrange, Act, Assert pattern.
- Always use testify/assert or similar libraries for readable assertions.
- Always create helper functions to reduce test code duplication.

### Test Coverage and Quality

- Always aim for high test coverage (>80%) but prioritize critical paths.
- Always test error conditions and edge cases.
- Always use dependency injection to enable proper unit testing.
- Always mock external dependencies in unit tests.
- Always write integration tests for important workflows.
- Always use build tags to separate unit tests from integration tests.

### Benchmarking and Performance Testing

- Always write benchmarks for performance-critical code.
- Always use testing.B for benchmark tests.
- Always use go test -bench to run performance tests.
- Always profile code when performance issues are suspected.

## Performance Considerations

### Memory Management

- Always prefer value types over pointer types when appropriate.
- Always reuse slices and maps when possible to reduce allocations.
- Always use sync.Pool for frequently allocated objects.
- Always be mindful of slice capacity to avoid unnecessary allocations.
- Always use strings.Builder for efficient string concatenation.

### Algorithm Efficiency

- Always choose appropriate data structures for the use case.
- Always consider time and space complexity of algorithms.
- Always use buffered I/O operations for better performance.
- Always batch operations when dealing with external systems.
- Always use goroutines judiciously; they're not always faster.

### Resource Management

- Always close resources (files, connections, etc.) using defer.
- Always use connection pooling for database and network operations.
- Always set appropriate timeouts for network operations.
- Always implement circuit breakers for external service calls.

## Security Guidelines

### Input Validation and Sanitization

- Always validate and sanitize all user inputs.
- Always use parameterized queries to prevent SQL injection.
- Always validate file uploads and restrict file types.
- Always implement proper authentication and authorization.
- Always use HTTPS for all network communications.

### Data Protection

- Always hash passwords using bcrypt or similar secure algorithms.
- Always use cryptographically secure random number generators.
- Always store sensitive data encrypted.
- Always avoid logging sensitive information (passwords, tokens, etc.).
- Always use environment variables for secrets, never hardcode them.

### Security Headers and Practices

- Always implement proper CORS policies.
- Always use security headers (CSP, HSTS, etc.).
- Always validate JWT tokens properly.
- Always implement rate limiting for API endpoints.
- Always use secure session management.

## Dependency Management

### Module Management

- Always use Go modules for dependency management.
- Always pin dependency versions in go.mod file.
- Always regularly update dependencies to get security fixes.
- Always use go mod tidy to keep dependencies clean.
- Always verify dependency checksums with go.sum.

### Vendor Management

- Always vendor dependencies for production deployments when necessary.
- Always avoid dependencies with known security vulnerabilities.
- Always prefer standard library over third-party libraries when possible.
- Always evaluate the maintenance status of dependencies before adding them.

## Documentation Standards

### Code Documentation

- Always write clear and concise comments for exported functions and types.
- Always follow Go doc comment conventions.
- Always include examples in documentation when helpful.
- Always update documentation when code changes.
- Always use godoc to generate and review documentation.

### API Documentation

- Always document API endpoints with clear descriptions.
- Always provide request and response examples.
- Always document error codes and their meanings.
- Always keep API documentation up to date with implementation.

### README and Project Documentation

- Always include a comprehensive README file.
- Always document installation and setup instructions.
- Always provide usage examples and getting started guides.
- Always document environment variables and configuration options.

## Concurrency Best Practices

### Goroutine Management

- Always ensure goroutines have a clear termination condition.
- Always use context for goroutine cancellation and timeout control.
- Always avoid creating goroutines in loops without bounds or control.
- Always use sync.WaitGroup or channels to coordinate goroutine completion.
- Always handle goroutine panics to prevent program crashes.

### Channel Usage

- Always prefer channels for communication between goroutines.
- Always close channels when no more values will be sent.
- Always use buffered channels when appropriate to prevent blocking.
- Always use select statements for non-blocking channel operations.
- Always avoid sharing memory by communicating instead of communicating by sharing memory.

### Synchronization Primitives

- Always use sync.Mutex for protecting shared state.
- Always use sync.RWMutex when reads significantly outnumber writes.
- Always use sync.Once for one-time initialization.
- Always use atomic operations for simple shared counters and flags.
- Always be consistent with locking order to avoid deadlocks.

### Race Condition Prevention

- Always run tests with -race flag to detect race conditions.
- Always design concurrent code to minimize shared mutable state.
- Always use proper synchronization for all shared data access.
- Always be careful with closure variables in goroutines.

### Worker Pool Patterns

- Always implement worker pools for controlling resource usage.
- Always use semaphores or limited channels to control concurrency.
- Always gracefully shut down worker pools using context cancellation.
- Always monitor goroutine counts to prevent goroutine leaks.
