# Tasks

## Phase 1: Initial Implementation (Completed)

- [x] **Layer 1: AI Suggestion** <!-- id: 0 -->
  - [x] Implement `KeywordSuggester` service using LLM provider.
  - [x] Create API endpoint `POST /projects/keywords/suggest`.
- [x] **Layer 2: Semantic Validator** <!-- id: 1 -->
  - [x] Update `KeywordValidator` with length and stopword checks.
  - [x] Integrate LLM check for ambiguity.
  - [x] Update `POST /projects` to enforce validation.
- [x] **Layer 3: Dry Run** <!-- id: 2 -->
  - [x] Implement `KeywordTester` service.
  - [x] Create API endpoint `POST /projects/keywords/dry-run`.
  - [x] Mock Collector Service response for dry run (or integrate if available).
- [x] **Data Model** <!-- id: 3 -->
  - [x] Add `ExcludeKeywords` to `Project` model and database migration.
  - [x] Update Create/Update APIs to accept `exclude_keywords`.

## Phase 2: Module Refactoring (Completed)

- [x] **Create Keyword Module** <!-- id: 4 -->
  - [x] Create `internal/keyword` directory structure.
  - [x] Define `keyword.Service` interface in `interface.go`.
  - [x] Move `KeywordSuggester` to `internal/keyword/suggester.go`.
  - [x] Move `KeywordValidator` to `internal/keyword/validator.go`.
  - [x] Move `KeywordTester` to `internal/keyword/tester.go`.
  - [x] Implement `keyword.Service` in `service.go` that orchestrates all three components.
- [x] **Update Project UseCase** <!-- id: 5 -->
  - [x] Remove keyword logic from `internal/project/usecase/project.go`.
  - [x] Inject `keyword.Service` into `ProjectUseCase` via constructor.
  - [x] Update `Create` and `Update` methods to use injected `keywordService`.
  - [x] Update `SuggestKeywords` and `DryRunKeywords` to delegate to `keywordService`.
- [x] **Update Dependency Injection** <!-- id: 6 -->
  - [x] Update `internal/httpserver/handler.go` to initialize `keyword.Service`.
  - [x] Pass `keywordService` to `projectusecase.New()`.
- [x] **Testing** <!-- id: 7 -->
  - [x] Create mock implementations in `internal/keyword/mock/`.
  - [x] Update existing tests to use mocked `keyword.Service`.
  - [x] Add unit tests for `keyword.Service`.
