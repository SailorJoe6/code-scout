# Test Coverage Improvement Plan

## Goal

Increase test coverage across all Code Scout components to achieve at least 90% coverage per component, ensuring robust testing of core functionality, error paths, and edge cases.

## Current State (Baseline: 82.2% overall)

| Component | Current Coverage | Target | Gap |
|-----------|-----------------|--------|-----|
| `internal/scanner` | 100.0% | 90% | ✅ Already achieved |
| `internal/embeddings` | 98.0% | 90% | ✅ Already achieved |
| `internal/chunker` | 95.1% | 90% | ✅ Already achieved |
| `internal/parser` | 89.0% | 90% | +1.0% needed |
| `internal/config` | 52.2% | 90% | +37.8% needed |
| `cmd/tei-wrapper` | 60.1% | 90% | +29.9% needed |
| `cmd/code-scout` | No unit tests | 90% | New tests needed |
| `internal/storage` | Not in report | 90% | Investigation needed |

## Implementation Strategy

This plan follows the **Elephant Carpaccio** approach: each slice is a complete, vertical improvement that delivers value independently. Each slice will be tested, committed, and pushed before moving to the next.

### Slice 1: Parser Package (89.0% → 90%+)

**Priority**: Highest (lowest effort, quickest win)

**Target Functions** (identified from detailed coverage):
- `extractMethodSpecName` (66.7%) - Missing edge cases for method spec parsing
- `extractImportPath` (75.0%) - Incomplete coverage of import path variations
- `extractPackageName` (75.0%) - Missing test cases for package name extraction
- `containsOnlyCMarkers` (90.9%) - One uncovered branch

**Approach**:
- Add table-driven tests for edge cases
- Test error conditions and malformed input
- Verify behavior with unusual but valid Go syntax

**Expected Effort**: 1-2 hours
**Deliverable**: Parser coverage at 90%+

---

### Slice 2: Config Package (52.2% → 90%+)

**Priority**: High (small, focused component)

**Likely Gaps** (based on config.go analysis needed):
- Error handling in config file parsing
- Missing config file scenarios
- Invalid JSON handling
- Config merging logic (user-level overrides project-level)
- Endpoint validation (URL parsing, normalization)
- API key handling edge cases
- Default value fallback behavior

**Approach**:
- Test all config loading scenarios:
  - No config files
  - User-level only
  - Project-level only
  - Both (merge behavior)
- Test invalid config scenarios:
  - Malformed JSON
  - Invalid field values
  - Missing required fields
- Test endpoint normalization and validation
- Test CLI flag override behavior

**Expected Effort**: 2-3 hours
**Deliverable**: Config coverage at 90%+

---

### Slice 3: TEI Wrapper (60.1% → 90%+)

**Priority**: Medium (isolated component, doesn't affect core functionality)

**Likely Gaps** (based on server.go analysis needed):
- HTTP handler error paths
- Request validation (malformed requests, missing fields)
- Health check endpoint coverage
- Graceful shutdown scenarios
- Concurrent request handling
- Timeout handling
- Error response formatting

**Approach**:
- Use `httptest` for testing HTTP handlers
- Test all endpoint variations:
  - Valid requests
  - Malformed JSON
  - Missing required fields
  - Invalid model names
- Test server lifecycle:
  - Startup
  - Graceful shutdown
  - Forced shutdown
- Test error propagation from mock backend

**Expected Effort**: 2-3 hours
**Deliverable**: TEI Wrapper coverage at 90%+

---

### Slice 4: Storage Package (? → 90%+)

**Priority**: High (core component, but requires investigation first)

**Current Issue**: Storage package not appearing in coverage report
- Possible cause: CGO build failures in test environment
- Need to investigate and fix build configuration first

**Investigation Steps**:
1. Determine why storage tests aren't running in coverage
2. Fix CGO/LanceDB test environment issues
3. Assess current actual coverage
4. Identify gaps

**Likely Gaps** (anticipated):
- Error recovery scenarios
- Edge cases in chunk storage/retrieval
- Database corruption handling
- Concurrent access patterns
- Schema migration scenarios
- Index creation and optimization
- Query error handling
- Connection lifecycle management

**Approach** (after investigation):
- Use table-driven tests for CRUD operations
- Test error scenarios with mock failures
- Test concurrent access patterns
- Verify cleanup and resource management
- Test incremental update scenarios

**Expected Effort**: 3-5 hours (including investigation)
**Deliverable**: Storage coverage at 90%+

---

### Slice 5: CLI Commands (cmd/code-scout) (0% → 90%+)

**Priority**: Lower (integration-heavy, depends on all components)

**Philosophy**: Focus on unit-testable business logic, not full integration tests
- Integration tests already exist in `integration_test.go`
- Focus on logic that can be tested in isolation

**Testable Components**:

**embeddings_factory.go**:
- Model selection logic based on file type
- Configuration override behavior
- Error handling for missing/invalid models

**Flag validation and parsing**:
- Worker count validation
- Batch size validation
- Limit/threshold validation
- Conflicting flag combinations

**Error handling**:
- User-friendly error messages
- Exit code consistency
- Logging behavior

**Approach**:
- Extract testable logic into pure functions where possible
- Mock external dependencies (storage, embeddings, scanner)
- Use table-driven tests for flag validation
- Test error message formatting and clarity
- Keep integration tests focused on happy path

**Expected Effort**: 3-4 hours
**Deliverable**: CLI command coverage at 90%+

---

## Testing Principles

### 1. Test Behavior, Not Implementation
- Focus on public interfaces and contracts
- Test edge cases and error conditions
- Don't test internal implementation details

### 2. Keep Tests Fast
- Mock external dependencies (Ollama, filesystem when reasonable)
- Use in-memory storage where possible
- Parallelize independent tests

### 3. Table-Driven Tests
- Use Go's table-driven test pattern for multiple scenarios
- Makes it easy to add new test cases
- Improves readability and maintenance

### 4. Don't Over-Engineer
- Target 90-95% coverage, not 100%
- Some code paths (panic recovery, truly impossible conditions) don't need tests
- Focus on valuable test coverage, not metrics gaming

### 5. One Slice Per Commit
- Complete each component fully before moving to next
- Commit with message: "test: Increase [component] coverage to XX%"
- Update this plan document after each slice

## Success Criteria

- [ ] All components at 90%+ coverage
- [ ] Overall project coverage at 90%+
- [ ] All tests passing
- [ ] No flaky tests
- [ ] Fast test suite (< 30 seconds for full run)
- [ ] Clear, maintainable test code

## Tracking Progress

Update this table as each slice is completed:

| Slice | Component | Status | Final Coverage | Commit SHA | Date |
|-------|-----------|--------|----------------|------------|------|
| 1 | `internal/parser` | ✅ Complete | 90.5% | Pending | 2026-01-16 |
| 2 | `internal/config` | ⏳ Not started | - | - | - |
| 3 | `cmd/tei-wrapper` | ⏳ Not started | - | - | - |
| 4 | `internal/storage` | ⏳ Not started | - | - | - |
| 5 | `cmd/code-scout` | ⏳ Not started | - | - | - |

Status Legend:
- ⏳ Not started
- 🔄 In progress
- ✅ Complete
- 🚧 Blocked

## Notes

- Storage package investigation is a prerequisite for Slice 4
- CLI command testing may reveal need for refactoring to improve testability
- If any component requires significant refactoring to reach 90%, consider creating a separate issue in beads to track the refactoring work
