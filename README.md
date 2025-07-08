# Aho-Corasick String Matching Algorithm

A memory-efficient, rune-based implementation of the Aho-Corasick string matching algorithm in Go.

## Features

- **Multi-byte character support**: Correctly handles UTF-8 encoded text (including Chinese, Japanese, etc.)
- **High performance**: Memory-efficient implementation with pre-allocated node arrays
- **Thread-safe**: Concurrent matching support with atomic operations
- **Consistent API**: Clean design with []byte as primary input type and explicit String variants
- **Zero dependencies**: Pure Go implementation

## Installation

```bash
go get github.com/itgcl/ahocorasick
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/itgcl/ahocorasick"
)

func main() {
    // Create a matcher with dictionary of patterns
    patterns := []string{"he", "she", "his", "hers"}
    matcher := ahocorasick.NewStringMatcher(patterns)
    
    // Search for patterns in text
    text := "she sells seashells by the seashore"
    matches := matcher.MatchString(text)
    
    fmt.Printf("Found %d matches: %v\n", len(matches), matches)
    // Output: Found 2 matches: [1 0] (indices in dictionary)
}
```

## API Reference

### Creating Matchers

```go
// Create from string slice (recommended)
matcher := ahocorasick.NewMatcher([]string{"pattern1", "pattern2"})

// Create from byte slice (converted to strings assuming UTF-8)
matcher := ahocorasick.NewByteMatcher([][]byte{[]byte("pattern1"), []byte("pattern2")})

// Backward compatibility alias
matcher := ahocorasick.NewStringMatcher([]string{"pattern1", "pattern2"})
```

### Core API - Consistent Naming

The API follows a clear pattern: **base methods accept `[]byte`, String variants have explicit suffix**

#### Basic Matching
```go
// Primary methods (accept []byte)
matches := matcher.Match([]byte("search text"))
found := matcher.Contains([]byte("search text"))
index, found := matcher.MatchFirst([]byte("search text"))

// String variants (explicit suffix)
matches := matcher.MatchString("search text")
found := matcher.ContainsString("search text")
index, found := matcher.MatchFirstString("search text")
```

#### Thread-Safe Matching
```go
// Primary methods (accept []byte)  
matches := matcher.MatchThreadSafe([]byte("search text"))

// String variants (explicit suffix)
matches := matcher.MatchThreadSafeString("search text")
```

### Deprecated Methods (for backward compatibility)

```go
// These methods are deprecated but still available
matcher.MatchBytes([]byte("text"))           // use Match() instead
matcher.MatchBytesThreadSafe([]byte("text")) // use MatchThreadSafe() instead  
matcher.ContainsBytes([]byte("text"))        // use Contains() instead
```

## Performance

The Aho-Corasick algorithm provides:
- **O(n + m + z)** time complexity where:
  - n = length of text
  - m = total length of all patterns  
  - z = number of matches
- **O(m)** space complexity for the automaton

This makes it ideal for searching many patterns simultaneously in large texts.

## Examples

### Case-Sensitive Matching
```go
patterns := []string{"Go", "go", "golang"}
matcher := ahocorasick.NewMatcher(patterns)
matches := matcher.Match([]byte("Go is a great language, golang rocks!"))
// Will find both "Go" and "golang"
```

### Multi-byte Character Support
```go
patterns := []string{"中文", "测试", "编程"}
matcher := ahocorasick.NewMatcher(patterns)
text := []byte("这是一个中文测试程序")
matches := matcher.Match(text)
```

### String vs Byte Slice Usage
```go
matcher := ahocorasick.NewMatcher([]string{"hello", "world"})

// Working with byte slices (primary API)
data := []byte("hello world")
matches := matcher.Match(data)                    // []byte input
found := matcher.Contains(data)                   // []byte input
first, ok := matcher.MatchFirst(data)             // []byte input

// Working with strings (explicit String suffix)
text := "hello world"  
matches = matcher.MatchString(text)               // string input
found = matcher.ContainsString(text)              // string input
first, ok = matcher.MatchFirstString(text)        // string input
```

### Concurrent Usage
```go
var wg sync.WaitGroup
data := []byte("search text")

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        matches := matcher.MatchThreadSafe(data)  // thread-safe
        // Process matches...
    }()
}
wg.Wait()
```

## Algorithm Details

The implementation consists of:

1. **Trie Construction**: Build a trie (prefix tree) from all patterns
2. **Failure Function**: Compute failure links for efficient backtracking  
3. **Output Function**: Mark nodes that represent complete patterns
4. **Suffix Links**: Find all possible pattern endings at each position

## API Design Principles

- **Primary methods accept `[]byte`**: Most efficient for binary data processing
- **String variants have explicit suffix**: Clear distinction for string inputs
- **Consistent naming**: Easy to remember and predict method names
- **Backward compatibility**: Deprecated methods still work for existing code

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
