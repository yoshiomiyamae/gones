# Task Completion Checklist

When completing a development task in GoNES, follow these steps:

## 1. Code Quality
- [ ] Code follows Go conventions and project style
- [ ] All comments are in English
- [ ] Exported functions have godoc comments
- [ ] Code is formatted with `gofmt` or `go fmt`

## 2. Testing
- [ ] Run tests to ensure no regressions:
  ```bash
  go test ./...
  # or
  make test
  ```
- [ ] Add new tests for new functionality
- [ ] Ensure all tests pass

## 3. Build Verification
- [ ] Verify the code builds successfully:
  ```bash
  go build ./cmd/gones
  # or
  make build
  ```

## 4. Dependencies
- [ ] If new dependencies were added, update go.mod:
  ```bash
  go mod tidy
  ```

## 5. Integration Testing (Optional)
- [ ] Test with actual ROM files if applicable
- [ ] Verify emulator functionality for affected components

## 6. Documentation
- [ ] Update README.md if user-facing features changed
- [ ] Update code comments for public APIs
- [ ] Document any breaking changes

## Notes
- No linter configuration file (`.golangci.yml`) was found, but standard Go conventions apply
- Use `go vet` for additional static analysis if needed
- Cross-compilation tests can be done with `make build-all` if applicable