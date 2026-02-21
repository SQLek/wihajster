# AGENTS.md

Guidance for contributors and coding agents working in this repository.

## Project intent

This repository aims to build a toy C compiler in Go with a strict, documented scope:

- single translation unit compilation
- assembly output only
- RISC-V target
- C99 subset first
- no self-hosting requirement

## Required planning constraints

When proposing or implementing changes, preserve the following priorities:

1. Finishability over breadth.
2. Deterministic diagnostics over permissive parsing.
3. Testability through a clear IR boundary (TAC).
4. Minimize external dependencies.
5. Keep QEMU `virt` as the first bring-up profile; treat CH32V003 as a later profile.

## Compiler architecture direction

Preferred pipeline:

1. Lexer + parser
2. Semantic analysis
3. TAC (three-address code) IR
4. RISC-V backend

Do not skip IR by tightly coupling AST and backend logic for non-trivial features.

## Language policy

- Treat the language subset as an explicit contract.
- If a C feature is unsupported, return a clear error.
- Avoid silently accepting syntax that is not semantically implemented.

## Milestone policy

Use this order unless there is a documented reason to deviate:

- M1: parser/sema/TAC with tests
- M2: RV32 backend + QEMU execution path
- M3: CH32V003 profile + embedded demos

## Pull request expectations

Every non-trivial PR should include:

- What subset/features changed
- Which milestone(s) it advances
- How behavior was validated (tests or command outputs)
- Any unsupported constructs explicitly deferred
