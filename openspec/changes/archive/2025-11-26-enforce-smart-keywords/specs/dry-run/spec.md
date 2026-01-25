# Keyword Dry Run

## ADDED Requirements

### Requirement: Fetch Sample Data

The system SHALL fetch a sample of real data (5-10 items) based on the provided keywords.

#### Scenario: Dry Run Execution

Given a list of keywords ["VinFast", "VF3"]
When the user requests a dry run
Then the system should return a list of 5-10 recent posts matching these keywords

### Requirement: Visualize Results

The system SHALL display the fetched sample data to the user for verification.

#### Scenario: Display Sample

Given a successful dry run response
When the data is received
Then the UI should display the content of the posts to the user
