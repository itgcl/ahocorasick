// ahocorasick.go: A memory-efficient, rune-based implementation of the
// Aho-Corasick string matching algorithm.
//
// This version operates on Go strings and runes to correctly handle
// multi-byte characters (like Chinese in UTF-8), preventing false matches
// across character boundaries.
//
// The Aho-Corasick algorithm is a multi-pattern string matching algorithm
// that can search for multiple patterns simultaneously. It consists of
// several key components:
// 1. Trie tree: stores all pattern strings
// 2. Failure function (fail): quickly jumps to the next possible match position when matching fails
// 3. Output function: marks which nodes represent complete pattern strings
// 4. Suffix links: used to find all possible matches

package ahocorasick

import (
	"container/list"
	"sync"
	"sync/atomic"
)

// node represents a node in the trie tree, operating on runes
type node struct {
	root    bool   // whether this is the root node
	output  bool   // whether this is the end node of a pattern string
	index   int    // if this is an output node, the index of the pattern in the dictionary
	counter uint64 // counter used for deduplication

	// child node mapping, key is rune character, value is corresponding child node
	// using rune instead of byte ensures correct handling of multi-byte characters
	child map[rune]*node

	// suffix points to the longest proper suffix that is also a word in the dictionary
	// used to quickly find other possible matches when current node matches
	suffix *node

	// fail points to the failure function, the node to jump to when current character fails to match
	// this is the core of AC algorithm, enabling efficient pattern matching
	fail *node
}

// Matcher contains the main structure of the Aho-Corasick automaton
// returned by NewMatcher, contains the complete matching automaton
type Matcher struct {
	counter uint64    // global counter for thread-safe deduplication
	trie    []node    // array storing all nodes, improving memory locality
	extent  int       // number of nodes currently used
	root    *node     // root node pointer
	heap    sync.Pool // memory pool used for thread-safe matching
}

// getFreeNode gets a new node from the pre-allocated node array
// this design avoids frequent memory allocations and improves performance
func (m *Matcher) getFreeNode() *node {
	m.extent++
	if m.extent == 1 {
		// initialize root node on first call
		m.root = &m.trie[0]
		m.root.root = true
	}
	newNode := &m.trie[m.extent-1]
	// note: child map is lazily initialized when needed to save memory
	return newNode
}

// buildTrie builds the AC automaton from a dictionary of strings
// this method implements the core of AC algorithm: building trie tree and computing failure function
func (m *Matcher) buildTrie(dictionary []string) {
	// estimate the number of trie nodes needed
	// for rune-based implementation, calculate total number of runes
	max := 1
	for _, word := range dictionary {
		for range word { // iterating over a string yields runes
			max++
		}
	}
	m.trie = make([]node, max)

	m.getFreeNode() // allocate root node

	// phase 1: build basic trie tree structure
	// insert all pattern strings into the trie
	for i, word := range dictionary {
		n := m.root
		// process rune by rune to ensure correctness with multi-byte characters
		for _, r := range word {
			if n.child == nil {
				n.child = make(map[rune]*node)
			}
			c, ok := n.child[r]
			if !ok {
				// if child node for current rune doesn't exist, create new node
				c = m.getFreeNode()
				n.child[r] = c
			}
			n = c
		}
		// mark the end node of pattern string
		n.output = true
		n.index = i
	}

	// phase 2: build failure function and suffix links
	// use breadth-first search (BFS) to compute fail pointers
	l := new(list.List)

	// initialize fail pointers of first level nodes to point to root
	for _, c := range m.root.child {
		c.fail = m.root
		l.PushBack(c)
	}

	// BFS traversal to build fail pointers
	for l.Len() > 0 {
		n := l.Remove(l.Front()).(*node)
		for r, childNode := range n.child {
			l.PushBack(childNode)

			// compute fail pointer for childNode
			f := n.fail
			for {
				failChild, ok := f.child[r]
				if ok {
					// found matching character, set fail pointer
					childNode.fail = failChild
					break
				}
				if f.root {
					// reached root node, fail pointer points to root
					childNode.fail = m.root
					break
				}
				// continue searching up the fail chain
				f = f.fail
			}

			// compute suffix pointer: points to longest output suffix
			if childNode.fail.output {
				childNode.suffix = childNode.fail
			} else {
				childNode.suffix = childNode.fail.suffix
			}
		}
	}

	// root node's suffix points to itself
	m.root.suffix = m.root
	// compress trie array, release unused space
	m.trie = m.trie[:m.extent]
}

// NewMatcher creates a matcher from a dictionary of byte slices
// assumes UTF-8 encoding, converts byte slices to strings
func NewMatcher(dictionary [][]byte) *Matcher {
	sDict := make([]string, len(dictionary))
	for i, b := range dictionary {
		sDict[i] = string(b)
	}
	return NewStringMatcher(sDict)
}

// NewStringMatcher is an alias for NewMatcher for backward compatibility
func NewStringMatcher(dictionary []string) *Matcher {
	m := new(Matcher)
	m.buildTrie(dictionary)
	return m
}

// Match searches input byte slice for all matching dictionary words, returns indices of matches in dictionary
// uses simple counter mechanism to prevent duplicate reporting of same match
func (m *Matcher) Match(text []byte) []int {
	return m.MatchString(string(text))
}

// MatchString searches input string for all matching dictionary words, returns indices of matches in dictionary
// uses simple counter mechanism to prevent duplicate reporting of same match
func (m *Matcher) MatchString(text string) []int {
	m.counter++
	return match(text, m.root, func(f *node) bool {
		if f.counter != m.counter {
			f.counter = m.counter
			return true
		}
		return false
	})
}

// match is the core matching logic, operating on runes
// unique function is used for deduplication, preventing same match from being reported multiple times
func match(text string, n *node, unique func(f *node) bool) []int {
	hits := make([]int, 0, 8)

	// process input text rune by rune
	for _, r := range text {
		child, ok := n.child[r]

		// if current node doesn't have child for this rune, follow fail chain
		for !ok && !n.root {
			n = n.fail
			child, ok = n.child[r]
		}

		// if found matching child node, move to that node
		if ok {
			n = child
		}

		// check if current node is an output node (complete pattern match)
		if n.output {
			if unique(n) {
				hits = append(hits, n.index)
			}
		}

		// check all possible suffix matches
		// suffix chain contains all patterns ending at current position
		f := n.suffix
		for f != nil && !f.root {
			if unique(f) {
				hits = append(hits, f.index)
			} else {
				break // if this suffix already reported, no need to check subsequent ones
			}
			f = f.suffix
		}
	}
	return hits
}

// MatchThreadSafe is the thread-safe version of Match, searches input byte slice
// uses atomic operations and thread-local storage to ensure concurrency safety
func (m *Matcher) MatchThreadSafe(text []byte) []int {
	return m.MatchThreadSafeString(string(text))
}

// MatchThreadSafeString is the thread-safe version of MatchString, searches input string
// uses atomic operations and thread-local storage to ensure concurrency safety
func (m *Matcher) MatchThreadSafeString(text string) []int {
	var heap map[int]uint64

	// use atomic operation to get unique generation identifier
	generation := atomic.AddUint64(&m.counter, 1)
	n := m.root

	// get or create deduplication map from memory pool
	item := m.heap.Get()
	if item == nil {
		heap = make(map[int]uint64, len(m.trie))
	} else {
		heap = item.(map[int]uint64)
	}

	// use thread-local heap for deduplication
	hits := match(text, n, func(f *node) bool {
		g := heap[f.index]
		if g != generation {
			heap[f.index] = generation
			return true
		}
		return false
	})

	// return heap to memory pool
	m.heap.Put(heap)
	return hits
}

// Contains checks if any dictionary word exists in the input byte slice
// more efficient than Match as it only needs to determine existence without collecting all matches
func (m *Matcher) Contains(text []byte) bool {
	return m.ContainsString(string(text))
}

// ContainsString checks if any dictionary word exists in the input string
// more efficient than Match as it only needs to determine existence without collecting all matches
func (m *Matcher) ContainsString(text string) bool {
	n := m.root
	for _, r := range text {
		child, ok := n.child[r]

		// follow fail chain to find match
		for !ok && !n.root {
			n = n.fail
			child, ok = n.child[r]
		}
		if ok {
			n = child
		}

		// check if match found (current node or any suffix)
		if n.output || (n.suffix != nil && !n.suffix.root) {
			return true
		}
	}
	return false
}

// MatchFirst searches input byte slice for the first matching dictionary word
// returns index of matching word in dictionary and boolean indicating if match was found
// returns immediately upon finding first match, more efficient than Match()
func (m *Matcher) MatchFirst(text []byte) (index int, ok bool) {
	return m.MatchFirstString(string(text))
}

// MatchFirstString searches input string for the first matching dictionary word
// returns index of matching word in dictionary and boolean indicating if match was found
// returns immediately upon finding first match, more efficient than Match()
func (m *Matcher) MatchFirstString(text string) (index int, ok bool) {
	n := m.root
	for _, r := range text {
		child, exists := n.child[r]

		// follow fail chain to find match
		for !exists && !n.root {
			n = n.fail
			child, exists = n.child[r]
		}
		if exists {
			n = child
		}

		// check if current node is a complete match
		if n.output {
			return n.index, true // found match, exit immediately!
		}

		// check for suffix match
		f := n.suffix
		if f != nil && !f.root {
			// note: we only need to check first suffix, as it represents
			// the longest possible suffix match at this position
			// suffix chain is already flattened during build
			return f.index, true // found suffix match, exit immediately!
		}
	}

	return -1, false // no match found in entire text
}
