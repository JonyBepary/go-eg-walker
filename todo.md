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
        - [ ] `IntersectWithSummary` / `IntersectWithSummaryFull`
        - [ ] `AddRaw` specific scenarios (overlap, multi-parent explicit tests)
        - [ ] Edge cases for all functions
- [ ] Port `index.ts` Core Logic to Go (`egwalker` package)
    - [ ] Implement unit tests for `egwalker`
- [ ] Extensive Testing Infrastructure & Validation (Go)
    - [ ] Develop Go utilities for test data parsing
    - [ ] Implement Conformance Tests
    - [ ] Port/Re-implement Fuzzer
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
