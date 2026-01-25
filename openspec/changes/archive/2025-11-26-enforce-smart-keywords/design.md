# Design: Smart Keyword Enforcer

## Overview

The Smart Keyword Enforcer introduces a 3-layer architecture as a **separate Keyword Module** that is injected into the **Project Service** usecase. This separation ensures better modularity, testability, and reusability across different services.

## Module Structure

### New Module: `internal/keyword`

All keyword-related components will be moved to a dedicated module:

```
internal/keyword/
├── interface.go          # Defines KeywordService interface
├── service.go            # Implements KeywordService
├── suggester.go          # KeywordSuggester implementation
├── validator.go          # KeywordValidator implementation
├── tester.go             # KeywordTester implementation
└── mock/                 # Mock implementations for testing
    ├── suggester.go
    ├── validator.go
    └── tester.go
```

### Dependency Injection

The `KeywordService` will be injected into the `ProjectUseCase`:

```go
type ProjectUseCase struct {
    repo           repository.Repository
    keywordService keyword.Service  // Injected dependency
    // ... other fields
}
```

## Architecture Layers

### Layer 1: AI Suggestion & Expansion (The Helper)

- **Component**: `keyword.Suggester`
- **Function**: Connects to an LLM (e.g., Gemini Flash/GPT-4o-mini).
- **Input**: Brand Name (e.g., "VinFast").
- **Output**:
  - **Niche Keywords**: Specific variations (e.g., "VinFast VF3", "VF Wild").
  - **Negative Keywords**: Terms to exclude (e.g., "sim vinfast", "xổ số").
- **Benefit**: Reduces user effort and improves keyword coverage.

### Layer 2: Semantic Validator (The Gatekeeper)

- **Component**: `keyword.Validator`
- **Function**: Checks for generic terms and potential ambiguities.
- **Rules**:
  - **Length Check**: Warn if too short (1 word).
  - **Stopwords**: Block common words (e.g., "xe", "mua").
  - **LLM Check**: Ask LLM if the keyword is ambiguous (e.g., "Apple" -> fruit vs tech).

### Layer 3: Dry Run (The Reality Check)

- **Component**: `keyword.Tester`
- **Function**: Fetches a small sample of real data to show the user what the keyword returns.
- **Flow**:
  1. User clicks "Test Keyword".
  2. Project Service calls `keywordService.TestKeywords()`.
  3. Keyword Service calls **Collector Service** with `dry_run=true`.
  4. Collector fetches 5-10 recent posts.
  5. UI displays posts.
  6. User verifies relevance.

## User Flow Update

1.  **Basic Info**: Name, Date Range.
2.  **Smart Configuration**:
    - User enters Seed Keyword.
    - System suggests Niche & Negative Keywords (Layer 1).
    - User selects/deselects.
3.  **Dry Run**:
    - System fetches sample posts (Layer 3).
    - User reviews. If "garbage", User adds Negative Keywords (Layer 2 warns if still generic).
4.  **Submit**:
    - Final payload includes `Include_Keywords` and `Exclude_Keywords`.

## Data Model Changes

Update `Project` struct to include `ExcludeKeywords`.

```json
{
  "brand_keywords": ["VinFast", "VF3"],
  "exclude_keywords": ["sim", "xổ số"]
}
```
