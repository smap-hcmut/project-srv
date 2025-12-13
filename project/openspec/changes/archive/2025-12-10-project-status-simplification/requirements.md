# Requirements Document

## Introduction

This document outlines the requirements for simplifying the project status system in the SMAP API. Currently, the system uses five project statuses (draft, active, completed, archived, cancelled), which creates unnecessary complexity. The goal is to reduce this to three essential statuses (draft, process, completed) that better align with the actual project lifecycle and execution flow.

## Glossary

- **Project Service**: The microservice component responsible for managing project entities and their lifecycle
- **ProjectStatus**: An enumeration representing the current state of a project in its lifecycle
- **Execute Operation**: The action that transitions a project from draft to processing state by initiating data collection
- **Redis State**: Runtime execution state stored in Redis for tracking project progress
- **PostgreSQL State**: Persistent project status stored in the PostgreSQL database

## Requirements

### Requirement 1

**User Story:** As a developer, I want a simplified project status model with only three states, so that the system is easier to understand and maintain.

#### Acceptance Criteria

1. THE Project Service SHALL support exactly three project statuses: "draft", "process", and "completed"
2. THE Project Service SHALL remove the "active", "archived", and "cancelled" status constants from the codebase
3. WHEN validating project status, THE Project Service SHALL accept only "draft", "process", or "completed" as valid values
4. THE Project Service SHALL update all status validation logic to reflect the three-status model
5. THE Project Service SHALL maintain backward compatibility by handling any existing projects with old status values

### Requirement 2

**User Story:** As a user, I want newly created projects to start in "draft" status, so that I can review and modify them before execution.

#### Acceptance Criteria

1. WHEN a user creates a new project, THE Project Service SHALL set the initial status to "draft"
2. WHILE a project has "draft" status, THE Project Service SHALL allow full modification of project properties
3. WHEN a project is in "draft" status, THE Project Service SHALL NOT have any associated Redis execution state
4. THE Project Service SHALL persist the "draft" status to PostgreSQL immediately upon project creation

### Requirement 3

**User Story:** As a user, I want to execute a draft project to begin data collection, so that the system starts processing my project requirements.

#### Acceptance Criteria

1. WHEN a user executes a draft project, THE Project Service SHALL transition the project status from "draft" to "process"
2. WHEN transitioning to "process" status, THE Project Service SHALL initialize Redis execution state
3. WHEN transitioning to "process" status, THE Project Service SHALL publish a project.created event to RabbitMQ
4. IF a project already has "process" status, THEN THE Project Service SHALL reject duplicate execution attempts with an appropriate error
5. THE Project Service SHALL update the PostgreSQL status to "process" when execution begins

### Requirement 4

**User Story:** As a system component, I want to mark projects as "completed" when all processing finishes, so that users know their project results are ready.

#### Acceptance Criteria

1. WHEN all project tasks finish successfully, THE Project Service SHALL transition the project status to "completed"
2. WHEN a project reaches "completed" status, THE Project Service SHALL maintain the Redis execution state for historical reference
3. THE Project Service SHALL persist the "completed" status to PostgreSQL
4. WHILE a project has "completed" status, THE Project Service SHALL allow users to view results but prevent re-execution
5. THE Project Service SHALL calculate and store final progress metrics when transitioning to "completed" status

### Requirement 5

**User Story:** As a developer, I want the usecase layer to handle status transitions correctly, so that the business logic enforces valid state changes.

#### Acceptance Criteria

1. THE Project Service SHALL enforce that only "draft" projects can be executed
2. THE Project Service SHALL prevent status transitions that skip intermediate states
3. WHEN updating a project, THE Project Service SHALL validate that the new status is one of the three allowed values
4. THE Project Service SHALL log all status transitions for audit purposes
5. IF an invalid status transition is attempted, THEN THE Project Service SHALL return a descriptive error message

### Requirement 6

**User Story:** As an API consumer, I want consistent status values in all API responses, so that I can reliably track project state.

#### Acceptance Criteria

1. WHEN returning project data via HTTP, THE Project Service SHALL include the current status as one of "draft", "process", or "completed"
2. WHEN filtering projects by status, THE Project Service SHALL accept only the three valid status values
3. THE Project Service SHALL reject API requests that specify invalid status values with a 400 Bad Request error
4. THE Project Service SHALL document the three valid status values in API documentation
5. WHEN querying project progress, THE Project Service SHALL return the current status along with execution metrics
