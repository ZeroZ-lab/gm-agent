# Scheme Module

> Strict procedural interpreter

## Purpose
- Define deterministic workflows in YAML/JSON that Agents must follow.
- Provide stricter guardrails than free-form prompting.

## Components
- Scheme parser that validates schema and steps.
- Step executor that runs actions in order, emitting events for each step.
- Error handling modes: `BLOCKED` (halt and wait) or `FALLBACK` (alternate path).

## Use Cases
- Compliance-critical flows that cannot rely solely on LLM discretion.
- Repeatable pipelines such as documentation generation or code review checklists.
