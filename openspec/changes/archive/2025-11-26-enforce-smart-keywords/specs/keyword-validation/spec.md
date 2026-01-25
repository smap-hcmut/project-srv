# Keyword Validation

## ADDED Requirements

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

Keywords SHALL be checked for generic terms and ambiguity.

#### Scenario: Generic Keyword

Given a project creation request
When `brand_keywords` contains "xe"
Then the request should be rejected with an error "keyword 'xe' is too generic"

#### Scenario: Ambiguous Keyword

Given a project creation request
When `brand_keywords` contains "Apple"
Then the system should warn the user about potential ambiguity with "fruit"
