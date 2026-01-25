# AI Keyword Suggestion

## ADDED Requirements

### Requirement: Suggest Niche Keywords

The system SHALL suggest niche keywords based on a seed keyword using an LLM.

#### Scenario: Suggestion for Brand

Given a seed keyword "VinFast"
When the user requests suggestions
Then the system should return a list including "VinFast VF3", "VF Wild", "trạm sạc VinFast"

### Requirement: Suggest Negative Keywords

The system SHALL suggest negative keywords to exclude irrelevant results.

#### Scenario: Negative Keywords for Brand

Given a seed keyword "VinFast"
When the user requests suggestions
Then the system should return a list of negative keywords including "sim", "xổ số", "tuyển dụng"
