# Patch Module

> Diff generation, application, and rollback

## Features
- Generate unified diffs for text files; detect binaries and fail fast.
- Dry-run apply to preview changes without touching disk.
- Apply with line-offset tolerance to handle drift.
- Automatic backup before apply; rollback restores from backup.

## Workflow
1. Reducer produces `ApplyPatch` command with diff and subject.
2. Executor validates target files and policies.
3. Apply or dry-run, returning structured patch result and artifact entries.
4. Emit events for success/failure so reducer can update state.
