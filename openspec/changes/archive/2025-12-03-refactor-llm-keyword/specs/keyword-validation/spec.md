## MODIFIED Requirements

### Requirement: Semantic Validation

Keywords SHALL be checked for generic terms and ambiguity. The system MUST use stopword filtering for generic terms and MAY use LLM-based ambiguity detection for single-word keywords. LLM-based checks are optional enhancements - when LLM fails, the system SHALL continue with the original keyword without blocking the request.

#### Scenario: Generic Keyword

Given a project creation request
When `brand_keywords` contains "xe"
Then the request should be rejected with an error "keyword 'xe' is too generic"

#### Scenario: Ambiguous Keyword - LLM Available and Working

Given a project creation request
When `brand_keywords` contains "Apple" (single word)
And the LLM service is available and responds successfully
Then the system calls LLM to check for ambiguity
And if LLM detects ambiguity, the system logs a warning with context
And the keyword "apple" (normalized) is accepted
And the project creation succeeds with 200 OK
And the keyword is saved to database

#### Scenario: Ambiguous Keyword - LLM Returns Error

Given a project creation request
When `brand_keywords` contains "Apple" (single word)
And the LLM service returns an error (404, timeout, invalid response, etc.)
Then the system logs a warning about LLM failure
And the keyword "apple" (normalized) is accepted
And the project creation succeeds with 200 OK
And the keyword is saved to database
And no error is returned to the user

#### Scenario: Ambiguous Keyword - Multi-Word (No LLM Check)

Given a project creation request
When `brand_keywords` contains "Apple iPhone" (multiple words)
Then the system does NOT call LLM for ambiguity check
And the keyword is validated using standard rules only
And the keyword is accepted if it passes standard validation

### Requirement: LLM-Based Ambiguity Detection

The system SHALL use LLM service to detect ambiguous single-word keywords. The system MUST only perform LLM checks on single-word keywords. The system SHALL log warnings for ambiguous keywords but SHALL NOT reject them. LLM failures SHALL NOT cause request failures - the original keyword SHALL be used.

#### Scenario: Ambiguity Detection Success

Given a single-word keyword "Apple"
When the system validates the keyword
And LLM service responds successfully
Then the system receives ambiguity result
And if ambiguous, logs a warning with context
And the keyword validation succeeds
And the normalized keyword is returned

#### Scenario: Ambiguity Detection Failure - Continue with Original

Given a single-word keyword "Apple"
When the system validates the keyword
And LLM service fails (any error)
Then the system logs a warning about LLM failure
And the keyword validation succeeds
And the normalized keyword "apple" is returned
And project creation continues normally

#### Scenario: No Ambiguity Check for Multi-Word

Given a multi-word keyword "Apple iPhone"
When the system validates the keyword
Then the system does NOT call LLM service
And the keyword is validated using standard rules only
And the normalized keyword is returned if valid
