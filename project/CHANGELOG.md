# Dry-Run Keywords Optimization - Change Log

## Overview

This document describes the changes made to the SMAP Project API service as part of the **Dry-Run Keywords Optimization** proposal. The optimization introduces keyword sampling to ensure dry-run operations complete within WebSocket timeout constraints.

## Version: 1.0.0

**Release Date**: December 2024  
**Status**: Implementation Complete

---

## Summary of Changes

### New Feature: Keyword Sampling for Dry-Run

The dry-run keywords endpoint now samples keywords before processing to ensure consistent execution times within WebSocket timeout limits (90 seconds).

**Key Benefits:**

- Predictable dry-run execution times (< 85 seconds)
- Reduced WebSocket timeout failures
- Better user experience with faster feedback
- Emergency fallback for edge cases

---

## API Changes

### Endpoint: `POST /projects/dryrun`

**Request**: No changes to request format.

**Response**: New fields added (backward compatible):

```json
{
  "job_id": "uuid",
  "status": "processing",
  "sampled_keywords": ["keyword1", "keyword2", "keyword3"],
  "total_keywords": 50,
  "sampling_ratio": 0.06,
  "estimated_duration": "48s"
}
```

| Field                | Type       | Description                                     |
| -------------------- | ---------- | ----------------------------------------------- |
| `sampled_keywords`   | `string[]` | Keywords selected for dry-run (subset of input) |
| `total_keywords`     | `int`      | Total keywords provided in request              |
| `sampling_ratio`     | `float`    | Ratio of sampled to total keywords (0.0 - 1.0)  |
| `estimated_duration` | `string`   | Estimated processing time                       |

---

## Configuration Changes

### New Environment Variables

| Variable                        | Default      | Description                                      |
| ------------------------------- | ------------ | ------------------------------------------------ |
| `DRY_RUN_PERCENTAGE`            | `10`         | Percentage of keywords to sample (0-100)         |
| `DRY_RUN_MIN_KEYWORDS`          | `3`          | Minimum keywords to ensure representative sample |
| `DRY_RUN_MAX_KEYWORDS`          | `5`          | Maximum keywords to cap performance              |
| `DRY_RUN_KEYWORD_TIME_ESTIMATE` | `16s`        | Estimated processing time per keyword            |
| `DRY_RUN_SAMPLING_STRATEGY`     | `percentage` | Strategy: percentage, fixed, or tiered           |
| `DRY_RUN_EMERGENCY_THRESHOLD`   | `70s`        | Threshold to trigger emergency fallback          |
| `DRY_RUN_EMERGENCY_KEYWORDS`    | `3`          | Keywords count for emergency fallback            |

### Kubernetes ConfigMap Updates

Add to `k8s/configmap.yaml`:

```yaml
# Dry-Run Keyword Sampling Configuration
DRY_RUN_PERCENTAGE: "10"
DRY_RUN_MIN_KEYWORDS: "3"
DRY_RUN_MAX_KEYWORDS: "5"
DRY_RUN_KEYWORD_TIME_ESTIMATE: "16s"
DRY_RUN_SAMPLING_STRATEGY: "percentage"
DRY_RUN_EMERGENCY_THRESHOLD: "70s"
DRY_RUN_EMERGENCY_KEYWORDS: "3"
```

---

## Architecture Changes

### New Module: `internal/sampling/`

Independent sampling module with clean architecture:

```
internal/sampling/
├── interface.go      # UseCase interface definition
├── type.go           # SampleInput, SampleOutput, StrategyType
├── error.go          # Domain errors
└── usecase/
    ├── new.go        # Constructor with config parsing
    ├── sampling.go   # Core sampling logic
    └── selector.go   # Random keyword selection
```

### Integration Points

1. **Project UseCase** (`internal/project/usecase/project.go`)

   - `DryRunKeywords()` now uses `sampling.UseCase` interface
   - Sampling applied before publishing to RabbitMQ

2. **HTTP Handler** (`internal/project/delivery/http/`)

   - Response includes sampling metadata
   - Swagger documentation updated

3. **Dependency Injection** (`internal/httpserver/handler.go`)
   - `sampling.UseCase` injected into project usecase

---

## Sampling Algorithm

### Percentage Strategy (Default)

```
targetCount = totalKeywords * percentage / 100
targetCount = max(targetCount, minKeywords)
targetCount = min(targetCount, maxKeywords)
targetCount = min(targetCount, totalKeywords)
```

### Emergency Fallback

When estimated time exceeds threshold:

```
if (targetCount * keywordTimeEstimate) > emergencyThreshold:
    targetCount = emergencyKeywords
```

### Example Calculations

| Input Keywords | Percentage | Min | Max | Result | Reason                             |
| -------------- | ---------- | --- | --- | ------ | ---------------------------------- |
| 50             | 10%        | 3   | 5   | 5      | 10% of 50 = 5, capped at max       |
| 10             | 10%        | 3   | 5   | 3      | 10% of 10 = 1, minimum applied     |
| 2              | 10%        | 3   | 5   | 2      | Cannot exceed available            |
| 100            | 10%        | 3   | 10  | 3      | Emergency fallback (10\*16s > 70s) |

---

## Performance Metrics

### Benchmark Results

```
BenchmarkUsecase_Sample-12    1851210    645.5 ns/op    2016 B/op    10 allocs/op
```

- **Sampling overhead**: ~645ns per operation (well under 5ms target)
- **Memory allocation**: 2016 bytes per operation
- **Test coverage**: 93.0%

---

## Migration Guide

### For API Consumers

1. **No breaking changes** - existing integrations continue to work
2. **New response fields** are additive - ignore if not needed
3. **Sampled keywords** are returned in response for transparency

### For Operators

1. Update `template.env` with new `DRY_RUN_*` variables
2. Update Kubernetes ConfigMaps with sampling configuration
3. Monitor logs for sampling decisions:
   ```
   INFO: Keyword sampling completed for user X: 5/50 keywords selected (10.0%), method=random, estimated_time=80s
   ```

### Rollback Procedure

To disable sampling (emergency rollback):

1. Set `DRY_RUN_PERCENTAGE=100` (sample all keywords)
2. Or set `DRY_RUN_MAX_KEYWORDS` to a very high value

---

## Files Changed

### New Files

- `internal/sampling/interface.go`
- `internal/sampling/type.go`
- `internal/sampling/error.go`
- `internal/sampling/usecase/new.go`
- `internal/sampling/usecase/sampling.go`
- `internal/sampling/usecase/selector.go`
- `internal/sampling/usecase/sampling_test.go`

### Modified Files

- `config/config.go` - Added `DryRunSamplingConfig`
- `template.env` - Added DRY*RUN*\* variables
- `k8s/configmap.yaml` - Added DRY*RUN*\* entries
- `internal/project/usecase/project.go` - Updated `DryRunKeywords()`
- `internal/project/usecase/new.go` - Added sampler dependency
- `internal/project/type.go` - Updated `DryRunKeywordsOutput`
- `internal/project/delivery/http/presenter.go` - Updated `DryRunJobResp`
- `internal/httpserver/handler.go` - Inject sampling usecase
- `docs/swagger.yaml` - Updated API documentation

---

## Testing

### Unit Tests

- `TestUsecase_Sample` - Basic sampling scenarios
- `TestUsecase_Sample_EmergencyFallback` - Emergency fallback
- `TestUsecase_FallbackToDefaults` - Config fallback
- `TestUsecase_Sample_ValidationErrors` - Input validation
- `TestUsecase_Sample_FixedStrategy` - Fixed strategy
- `TestUsecase_Sample_TieredStrategy` - Tiered strategy
- `TestUsecase_ConfigParsingErrors` - Config parsing
- `TestSelector_EdgeCases` - Selector edge cases

### Benchmark Tests

- `BenchmarkUsecase_Sample` - Performance benchmark

---

## Known Limitations

1. **Random selection** - Keywords are randomly selected, not prioritized
2. **No persistence** - Sampling decisions are not stored
3. **Single strategy** - Only percentage strategy is fully implemented

## Future Improvements

1. **Priority-based selection** - Select keywords by importance
2. **Diversity selection** - Ensure keyword variety
3. **A/B testing** - Compare different strategies
4. **Metrics collection** - Track sampling effectiveness

---

## References

- [Proposal Document](./proposal.md)
- [Design Document](./design.md)
- [Implementation Tasks](./tasks.md)
