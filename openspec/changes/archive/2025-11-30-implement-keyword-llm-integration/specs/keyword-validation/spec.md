# Keyword Validation

## MODIFIED Requirements

### Requirement: Enforce Keyword Length

Keywords MUST be between 2 and 50 characters long.

#### Scenario: Keyword too short

Given a project creation request
When `brand_keywords` contains "a"
Then the request should be rejected with an error "keyword 'a' is too short (min 2 characters)"

#### Scenario: Keyword too long

Given a project creation request
When `brand_keywords` contains a string with 51 characters
Then the request should be rejected with an error "keyword is too long (max 50 characters)"

### Requirement: Enforce Character Set

Keywords MUST only contain alphanumeric characters, spaces, hyphens, and underscores.

#### Scenario: Invalid characters

Given a project creation request
When `brand_keywords` contains "hello@world"
Then the request should be rejected with an error "keyword 'hello@world' contains invalid characters"

### Requirement: Auto-Normalization

Keywords SHALL be trimmed of whitespace and converted to lowercase.

#### Scenario: Whitespace trimming

Given a project creation request
When `brand_keywords` contains " hello "
Then the keyword should be saved as "hello"

#### Scenario: Case insensitivity

Given a project creation request
When `brand_keywords` contains "Hello"
Then the keyword should be saved as "hello"

### Requirement: Deduplication

Duplicate keywords within the same list SHALL be merged.

#### Scenario: Duplicate keywords

Given a project creation request
When `brand_keywords` contains ["hello", "HELLO"]
Then only "hello" should be saved once

### Requirement: Semantic Validation

Keywords SHALL be checked for generic terms and ambiguity. The system MUST use stopword filtering for generic terms and MAY use LLM-based ambiguity detection for single-word keywords.

#### Scenario: Generic Keyword

Given a project creation request
When `brand_keywords` contains "xe"
Then the request should be rejected with an error "keyword 'xe' is too generic"

#### Scenario: Ambiguous Keyword - Single Word

Given a project creation request
When `brand_keywords` contains "Apple" (single word)
And the LLM service is available
Then the system calls LLM to check for ambiguity
And if LLM detects ambiguity, the system logs a warning with context (e.g., "may refer to fruit or technology company")
And the keyword is accepted (not rejected)
And the warning is included in the response or logs

#### Scenario: Ambiguous Keyword - Multi-Word

Given a project creation request
When `brand_keywords` contains "Apple iPhone" (multiple words)
Then the system does NOT call LLM for ambiguity check
And the keyword is validated using standard rules only
And the keyword is accepted if it passes standard validation

#### Scenario: LLM Ambiguity Check Unavailable

Given a project creation request
When `brand_keywords` contains "Apple" (single word)
And the LLM service is unavailable
Then the system logs a warning that LLM check was skipped
And the keyword is validated using standard rules only
And the keyword is accepted if it passes standard validation
And no error is returned to the user

#### Scenario: LLM Ambiguity Check Timeout

Given a project creation request
When `brand_keywords` contains "Apple" (single word)
And the LLM service times out
Then the system logs a warning that LLM check timed out
And the keyword is validated using standard rules only
And the keyword is accepted if it passes standard validation

#### Scenario: LLM Ambiguity Check Invalid Response

Given a project creation request
When `brand_keywords` contains "Apple" (single word)
And the LLM service returns an invalid response
Then the system logs a warning that LLM check failed
And the keyword is validated using standard rules only
And the keyword is accepted if it passes standard validation

## ADDED Requirements

### Requirement: LLM-Based Ambiguity Detection

The system SHALL use LLM service to detect ambiguous single-word keywords. The system MUST only perform LLM checks on single-word keywords to optimize performance and cost. The system SHALL log warnings for ambiguous keywords but SHALL NOT reject them automatically.

#### Scenario: Ambiguity Detection for Single Word

Given a single-word keyword "Apple"
When the system validates the keyword
Then the system calls LLM service with the keyword
And LLM returns whether the keyword is ambiguous and context
And if ambiguous, the system logs a warning with the context
And the keyword validation continues (does not fail)

#### Scenario: No Ambiguity Check for Multi-Word

Given a multi-word keyword "Apple iPhone"
When the system validates the keyword
Then the system does NOT call LLM service
And the keyword is validated using standard rules only

#### Scenario: Ambiguity Detection Configuration

Given the system is configured with LLM provider
When a single-word keyword is validated
Then the system uses the configured LLM provider for ambiguity check
And the check respects LLM timeout and retry settings
