---
name: go-expert
version: 1.0
description: |
  Provides expert knowledge about Go programming language including
  best practices, idioms, concurrency patterns, and common pitfalls.
tags:
  - go
  - golang
  - programming
---

# Go Expert Skill

This skill provides expert knowledge about the Go programming language.

## Key Concepts

### Go Philosophy
- **Simplicity**: Go favors simple, readable code over clever abstractions
- **Composition over inheritance**: Go uses interfaces and composition
- **Explicit is better than implicit**: Error handling is explicit, no exceptions

### Concurrency Patterns

#### Goroutines
```go
// Start a goroutine
go func() {
    // Do work concurrently
}()
```

#### Channels
```go
// Buffered channel
ch := make(chan int, 10)

// Send and receive
ch <- value
result := <-ch
```

#### Select Pattern
```go
select {
case val := <-ch1:
    // Handle ch1
case val := <-ch2:
    // Handle ch2
case <-time.After(timeout):
    // Handle timeout
}
}
```

### Common Pitfalls to Avoid

1. **Not checking errors**: Always check error returns
2. **Goroutine leaks**: Always ensure goroutines can exit
3. **Channel misuse**: Understand when to use buffered vs unbuffered channels
4. **Interface pollution**: Define interfaces where they're used, not in advance
5. **Ignoring context**: Always pass context for cancellable operations

### Error Handling Best Practices

```go
// Wrap errors with context
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Use custom errors for expected error conditions
var ErrNotFound = errors.New("not found")
```

### Performance Tips

1. **Use `strings.Builder`** for string concatenation in loops
2. **Pre-allocate slices** when size is known
3. **Use `sync.Pool`** for frequently allocated objects
4. **Profile before optimizing** - use `pprof`

### Testing Patterns

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"basic", "input", "output", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go by Example](https://gobyexample.com/)
