# Code Style and Conventions

## General Go Conventions
- Follow standard Go formatting (use `gofmt` or `go fmt`)
- Package names are lowercase, single words
- Exported symbols start with uppercase letters
- Private symbols start with lowercase letters

## Naming Conventions
- **Types/Structs**: PascalCase (e.g., `CPU`, `NES`, `Memory`)
- **Functions**: PascalCase for exported, camelCase for private (e.g., `New`, `Reset`, `handleNMI`)
- **Constants**: PascalCase with descriptive prefix (e.g., `FlagCarry`, `FlagZero`)
- **Variables**: camelCase (e.g., `cpu`, `mem`, `framebuffer`)

## Code Organization
- One package per directory under `pkg/`
- Test files are named `*_test.go` in the same package
- Constructor functions are named `New` or `New<Type>` (e.g., `New`, `NewNES`)
- Test helper functions use `create<Type>` pattern (e.g., `createTestCPU`)

## Comments
- **English comments**: All code comments, function documentation, and commit messages should be in English
- Use godoc-style comments for exported symbols
- Start with the symbol name (e.g., `// CPU represents the 6502 processor`)
- Document non-obvious implementation details inline

## Struct Patterns
```go
type CPU struct {
    // Group related fields with comments
    // Registers
    A  uint8  // Accumulator
    X  uint8  // X register
    
    // Memory interface
    Memory *memory.Memory
}
```

## Testing
- Use table-driven tests where appropriate
- Test file structure: helper functions first, then test functions
- Test function naming: `Test<FunctionName>` (e.g., `TestCPUReset`)
- Use descriptive error messages with expected vs actual values

## Error Handling
- Use standard Go error handling patterns
- Return errors explicitly rather than panicking (except for unrecoverable errors)

## Import Organization
- Standard library imports first
- Third-party imports second
- Local project imports last
- Grouped with blank lines between categories

Example:
```go
import (
    "fmt"
    "testing"
    
    "github.com/veandco/go-sdl2/sdl"
    
    "github.com/yoshiomiyamaegones/pkg/memory"
    "github.com/yoshiomiyamaegones/pkg/logger"
)
```