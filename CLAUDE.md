# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## How to work with the user on this project

This is a learning project: the user is building a database from scratch to learn database
internals, and is also learning Go. They want to write the hard parts themselves and use Claude
as a design/code-review companion, not an implementer. Follow these rules strictly:

- **Never write or edit code unless explicitly asked to.** Do not "helpfully" fix, refactor, or
  complete code you're reviewing — including the known build break noted below. Point it out;
  don't fix it unless told to.
- **Never modify a file without an explicit instruction to do so**, even if the change seems
  obviously correct or small.
- **Default mode is code review, style feedback, and testing feedback** — point out bugs, races,
  unidiomatic Go, missing edge cases/tests, API design issues, etc., but let the user write the
  fix.
- **When answering design/code questions, don't hand over complete working code.** Explain the
  concept, point at the relevant tradeoff or Go idiom, ask guiding questions, or sketch things
  at a high level — the way a mentor would, not an autocomplete. Only give a full, ready-to-use
  implementation if the user says they're stuck/struggling and explicitly want the answer.
- Go-specific feedback is especially valuable here (idioms, stdlib usage, error handling
  conventions, testing patterns with `testing`/table tests) since Go itself is new to the user.

## What this is

A from-scratch, educational implementation of database internals in Go, module
`github.com/felipedreis/dbinternals`. The focus is B+Tree data structures — first an
in-memory pointer-based version, now being reworked into a page-based (slotted page) version
closer to how a real disk-backed storage engine would work.

## Commands

```sh
go build ./...          # build everything
go test ./...           # run all tests
go test ./ds/...        # run tests in the ds package only
go test ./ds/ -run TestBTree_Add   # run a single test by name
go vet ./...             # static checks
go run .                 # run main.go
```

There is no separate lint config, CI, or Makefile — `go build`/`go vet`/`go test` are the whole
toolchain.

**Known state:** the repo is mid-refactor from the in-memory B-Tree to the page-based one, and
`ds/btree.go` currently does not compile on its own (`go build ./...` / `go test ./...` will
fail with declared-and-not-used / undefined-symbol errors in `checkUnderflow`, specifically the
right-merge branch). This is expected pre-existing WIP, not something you broke — check
`git status`/`git diff` before assuming a build failure is caused by your own edit.

## Architecture

Everything lives in the `ds` package (`ds/`). There are two parallel, mostly independent B+Tree
implementations sharing some interfaces and helpers:

### 1. In-memory tree (`ds/node.go`, `ds/inMemoryBTree.go`) — working
- `Node` is a plain in-memory struct with `Keys []Key`, `Values []Value` (leaves) or
  `Child []*Node` (internal nodes), plus `Parent`, `Left`/`Right` sibling pointers.
- `InMemoryBTree` (constructed via `NewBTree(nodeSize)`) drives `Add`/`Remove`/`Find` on top of
  `Node`, using `find` (recursive descent via `binarySearch`/`upperBound`) plus
  `checkOverflow`/`checkUnderflow` to split/merge/rebalance nodes back up to the root.
- This implementation is functionally complete (including rebalance-with-sibling and
  merge-with-sibling) and is what `ds/btree_test.go` exercises.

### 2. Page-based tree (`ds/page.go`, `ds/btree.go`, `ds/types.go`) — in progress
- `Page` (`ds/page.go`) is a slotted-page byte-buffer implementation: a fixed `PAGE_SIZE` (4096)
  `[]byte` with a header (flags, version, key count, data size, fragmentation, free-block
  offset, parent/sibling/left-child page pointers) followed by a growing-inward array of cell
  pointers and a growing-backward area of cell payloads. See the comment block at the top of
  `page.go` for the exact byte layout. Leaf cells encode `[keySize|dataSize|key|data]`; internal
  cells encode `[keySize|childPointer|key]`. Keys/values are raw `[]byte` at this layer —
  `Page` has no notion of the `Key`/`Value` types used elsewhere.
- `PutKeyValue`/`PutKeyChild` write into free space, compacting via `defrag()` when needed;
  `RemoveValueAtIdx`/`RemoveChildAtIdx` remove a cell and mark its space fragmented rather than
  reclaiming it immediately. `Split`/`Merge` operate directly on the byte layout.
- `BTNode` (`ds/types.go`) is the interface the tree logic (`ds/btree.go`) programs against —
  it's the typed (`Key`/`Value`/`NodeId`) counterpart of the raw `Page` API, so a future
  `Page`-backed implementer can satisfy it. `BTNodeProvider` is the pluggable node
  store/allocator (`GetNewNode`, `GetNode`, `Release`, `Tombstone`) — i.e. the seam where a real
  buffer pool / pager would plug in.
- `PagedBTree` (`ds/btree.go`) implements `BTree` (`Add`/`Remove`/`Find`) against `BTNode` +
  `BTNodeProvider` + a `MasterPage` (holds/updates the root `NodeId`). `find` walks root-to-leaf
  collecting the full ancestor stack (reversed, leaf-first) so `checkOverflow`/`checkUnderflow`
  can propagate splits/merges/rebalances upward without re-walking the tree. `Find` (point
  lookup) is not implemented yet (`errors.New("not implemented")`), and the underflow
  right-merge branch is incomplete (that's the current build break, see above).

### Shared pieces
- `Key` (`ds/types.go`) is `fmt.Stringer` + `Compare(other Key) int` (java-style tri-state
  comparator); `Value` wraps a `[]byte`. Both implementations are generic over any `Key`.
- `ds/util.go` has small generic slice helpers, `InsertAt[T]`/`Remove[T]`, used by both
  implementations for in-place-ish shifting of `Keys`/`Values`/`Child` slices.
- Tests define a local `TestKey{k int}` implementing `Key` (see `ds/btree_test.go`).

When extending the page-based tree, prefer wiring new logic through `BTNode`/`BTNodeProvider`
rather than calling `Page` methods directly from `PagedBTree`, so the tree logic stays storage
agnostic (in-memory vs. page-backed).
