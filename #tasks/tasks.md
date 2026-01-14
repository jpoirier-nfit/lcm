# Tasks

This file tracks development tasks for this project.

## Tasks

1. [x] [JIRA-NONE] Add Container Port Display and Browser Launch Feature to LCM <!-- added: 2026-01-13 --> <!-- model: global.anthropic.claude-haiku-4-5-20251001-v1:0 --> <!-- completed: 2026-01-13 --> <!-- completed-by: global.anthropic.claude-haiku-4-5-20251001-v1:0 -->

  - [x] Display open ports for running containers and add button to launch browser for each exposed port

2. [x] [JIRA-NONE] Implement Multi-Platform Container Support for Colima, Podman, and Orbstack <!-- added: 2026-01-13 --> <!-- model: global.anthropic.claude-haiku-4-5-20251001-v1:0 --> <!-- completed: 2026-01-14 --> <!-- completed-by: global.anthropic.claude-haiku-4-5-20251001-v1:0 -->

  - [x] Refactor container platform logic into configurable provider interface/pattern to support multiple platforms
  - [x] Implement platform adapter implementations for Colima, Podman, and Orbstack with platform-specific command handling
  - [x] Write unit and integration tests for each new platform provider to ensure compatibility
  - [x] Update README and documentation with platform detection and usage examples for each container platform

3. [x] [JIRA-NONE] Fix Duplicate Actions Control Display in Bottom Panel <!-- added: 2026-01-14 --> <!-- model: global.anthropic.claude-haiku-4-5-20251001-v1:0 --> <!-- completed: 2026-01-14 --> <!-- completed-by: global.anthropic.claude-haiku-4-5-20251001-v1:0 -->

  - [x] Locate and remove duplicate rendering logic for ACTIONS, INFO, FILTERS controls in bottom box

4. [x] [JIRA-NONE] Reorder Container List Columns to ID, NAME, IMAGE, STATUS, OPENPORTS, STATE <!-- added: 2026-01-14 --> <!-- model: global.anthropic.claude-haiku-4-5-20251001-v1:0 -->

  - [x] Verify column order displays correctly in container list table

5. [x] [JIRA-NONE] Implement Fuzzy Search Hotkey with / Trigger <!-- added: 2026-01-14 --> <!-- model: global.anthropic.claude-haiku-4-5-20251001-v1:0 --> <!-- completed: 2026-01-14 --> <!-- completed-by: global.anthropic.claude-haiku-4-5-20251001-v1:0 -->

  - [x] Add keyboard listener for '/' to trigger fuzzy search overlay with global search across containers and commands

