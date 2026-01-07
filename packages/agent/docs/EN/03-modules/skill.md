# Skill Module

> Persistable reusable capabilities

## Goals
- Capture successful agent behaviors as reusable prompts/procedures.
- Versioned skills stored under `skills/` with metadata and test cases.

## Structure
- Metadata: name, version, description, tags, owner.
- Content: reference prompts, expected inputs/outputs, evaluation examples.
- Loading: registry scans `skills/` on startup and registers available skills.

## Usage
- Agents can reference skills to bootstrap prompts or delegate subtasks.
- Evaluation harness runs skill test cases to prevent regressions.
