# TODO: Go EG Walker Library Development

## Phase 0: Foundation & Infrastructure Setup
- [x] GitHub Repository Creation
- [x] Go Module Initialization
- [x] Initial Directory Structure
- [x] Basic CI/CD Setup (GitHub Actions)
- [ ] `pkg.go.dev` Consideration (pending first tagged release)

## Phase 1: Core Logic Porting & Correctness
- [x] Data Structure Definition (Go)
- [x] Port `causal-graph.ts` to Go (`causalgraph` package) (core functions ported)
    - [x] Implement unit tests for `causalgraph`
        - [x] `Diff`
        - [x] `FindDominators`
        - [x] `FindConflicting`
        - [x] `CompareVersions`
        - [x] `IterVersionsBetween`
        - [x] `IntersectWithSummary` / `IntersectWithSummaryFull`
        - [x] `AddRaw` basic and advanced scenarios (e.g., `TestAddRaw_AdvancedScenarios`)
        - [ ] `AddRaw` more exhaustive tests for overlap, multi-parent, and edge cases if needed
        - [ ] Edge cases for all functions
- [~] Port `index.ts` Core Logic to Go (`egwalker` package) (basic structure and some functions ported, key CRDT/replay logic pending)
    - [ ] Implement `integrate` function (YjsMod/FugueMax CRDT logic for inserts) from `index.ts`'s `apply1`
    - [ ] Implement full `apply1` logic (using `integrate` and correct positioning) from `index.ts` (currently `egwalker.applyOp` is simplified)
    - [ ] Implement full `retreat1` logic from `index.ts` (currently `egwalker.retreatOp` is simplified)
    - [ ] Implement full `traverseAndApply` logic from `index.ts` (core history replay and state synchronization logic)
    - [ ] Implement `mergeOplogInto` function from `index.ts`
    - [ ] Refine `Walker.merge` to correctly use the full `traverseAndApply` logic for `mergeChangesIntoBranch` equivalent behavior
    - [ ] Refine `Walker.Checkout` to use the full `traverseAndApply` logic for accurate state generation
    - [ ] Implement unit tests for `egwalker`
        - [x] Basic `LocalInsert`, `LocalDelete` (via `Walker.LocalInsert`, `Walker.LocalDelete`)
        - [ ] Tests for `integrate` and full `apply1` logic with concurrent inserts
        - [ ] Tests for full `retreat1` logic
        - [ ] Tests for `traverseAndApply` with various historical sequences and branches
        - [ ] Tests for `Walker.merge` (complex merge scenarios, equivalent to `mergeChangesIntoBranch`)
        - [ ] Tests for `Walker.Checkout` (various versions, complex histories)
        - [ ] Tests for `mergeOplogInto`
- [ ] Extensive Testing Infrastructure & Validation (Go)
    - [ ] Develop Go utilities for test data parsing (from `eg-walker-reference/testdata/`)
    - [ ] Implement Conformance Tests (using parsed `ff-raw.json`, `git-makefile-raw.json`, `conformance.json`, etc.)
    - [ ] Port/Re-implement Fuzzer (`ListFugueSimple.ts` to Go, then fuzzer logic from `fuzzer.ts`)
    - [ ] CI Integration for all tests
- [ ] Goal: Achieve 100% pass rate on all ported conformance and fuzz tests.

## Phase 2: API Design, Initial Benchmarking, and "Production Grade" Groundwork
- [ ] Public API Definition
- [ ] Basic Benchmarking (Go Native)
- [ ] Code Refinement & Documentation

## Phase 3: Concurrency Design and Implementation
- [ ] Identify Concurrency Opportunities & Constraints
- [ ] Concurrency Strategy
- [ ] Implementation & Refinement
- [ ] Concurrency Testing (Go Native with `-race`)

## Phase 4: Performance Optimization
- [ ] In-depth Profiling (Go Native)
- [ ] Optimization Iteration (Algorithmic & Go-specific)
- [ ] Re-benchmark and Validate

## Phase 5: Production Hardening & Documentation
- [ ] Robustness & Error Handling
- [ ] Comprehensive Documentation (GoDoc, examples)
- [ ] API Stability Review
- [ ] Serialization (Initial Design & Implementation)
- [ ] Tagging a Release (e.g., v0.1.0)
