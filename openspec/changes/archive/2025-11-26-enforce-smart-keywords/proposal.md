# Smart Keyword Enforcer

## Summary

Implement a 3-layer "Smart Keyword Enforcer" architecture to prevent "Garbage In, Garbage Out" by ensuring high-quality keyword inputs through AI suggestion, semantic validation, and dry-run testing.

## Problem Statement

Currently, the system accepts keywords without strict validation. If a user enters poor-quality keywords (e.g., generic terms, typos), the system crawls irrelevant data ("garbage"), wasting database storage and AI processing resources. Users are not SEO experts and often fail to provide sufficient or specific keywords.

## Goals

- **Eliminate Garbage Data**: Prevent generic or irrelevant keywords from entering the system.
- **Assist Users**: Proactively suggest niche and negative keywords using AI.
- **Verify Before Commit**: Allow users to "Dry Run" keywords to see actual results before creating a project.

## Non-Goals

- Full SEO suite replacement.
- Real-time monitoring (this is configuration time).

## Architectural Decision

To maintain clean separation of concerns and improve testability, all keyword-related functionality will be extracted into a dedicated **Keyword Module** (`internal/keyword`). This module will be injected into the Project UseCase as a dependency, following the Dependency Injection pattern already established in the codebase.
