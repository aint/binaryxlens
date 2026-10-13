---
name: golang-dev
description: Trigger this skill when creating, refactoring, linting, building, or testing Go (.go) code, or adjusting go.mod.
globs: **/*.go
alwaysApply: true
---

# Golang Development Skill

## 1. Core Language Rules
- This project utilizes Go 1.2x.
- Code must be idiomatic, leveraging standard library packages wherever possible.

## 2. Verification & Definition of Done
- A task involving Go code is only considered complete ("Done") when the project build passed:
- If any step fails, do not proceed. Refer back to the Escalation Rules in `AGENTS.md`.

## 3. Do not create tests
Feel free to use tests for verifying something, but do not create test files without direct ask to do it.

