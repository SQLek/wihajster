# wihajster

A toy C compiler written in Go, focused on a **finishable** subset of C99 and deterministic behavior.

## Vision

Build a compiler that is small, understandable, and reliable for embedded-style demos, while avoiding unnecessary complexity:

- single translation unit input (`.c`)
- assembly output (`.s`) only
- RISC-V target
- no linker implementation in the compiler itself
- strict subset of C99 with explicit "not supported" errors

## Scope decisions

### What we are building first (MVP)

- Frontend for a strict C99 subset
- Semantic checks for supported constructs
- Three-address code (TAC) intermediate representation
- RISC-V RV32 assembly emitter
- End-to-end compile flow:
  - parse one C file
  - produce one assembly file

### What we are explicitly not building (initially)

- full C99 support
- multi-file compilation and linking orchestration
- optimizer passes beyond basic canonicalization
- full libc/std headers support
- self-hosting

## Language subset (v0)

Supported in v0:

- scalar integer types: `int`, `char`
- pointers to supported scalar types
- expressions:
  - literals, identifiers
  - unary: `-`, `!`, `~`, `*`, `&`
  - binary: `+`, `-`, `*`, `/`, `%`, `<<`, `>>`, `&`, `|`, `^`, `&&`, `||`
  - comparisons and assignment
- statements:
  - block
  - `if` / `else`
  - `while`
  - `for`
  - `return`
- function definitions and direct calls (non-variadic)
- global and local scalar declarations

Rejected in v0 (must fail with clear diagnostics):

- `struct`, `union`, `enum`
- floating-point types
- function pointers
- variadic functions
- `switch`, `goto`
- advanced initializers and declarators
- most preprocessor features (except optional minimal constant defines if we add them deliberately)

## Architecture

Pipeline:

1. **Lexer/Parser** -> AST
2. **Semantic analysis** -> typed, validated program
3. **IR lowering** -> three-address code (TAC)
4. **Backend** -> RISC-V assembly

Why TAC:

- simple enough for a toy compiler
- easier unit testing vs direct AST-to-assembly
- stable interface between frontend and backend

## Target strategy

Primary bring-up target:

- QEMU `virt` profile (first-class during compiler bootstrap)

Secondary target (after stable MVP):

- CH32V003 profile with dedicated startup/runtime assumptions

Both targets share core frontend + TAC; only target profile and runtime glue differ.

## Milestones

### M1: Frontend + TAC foundation

- tokenizer and parser for v0 grammar
- semantic analyzer for supported types and expressions
- TAC data model and lowering for expressions/statements
- parser + sema + TAC golden tests

Exit criteria:

- test suite covers accepted/rejected v0 snippets with deterministic diagnostics

### M2: RISC-V backend and execution path

- map TAC to RV32 assembly
- function call/stack discipline for supported subset
- basic runtime entry assumptions for QEMU profile
- differential smoke checks against a reference compiler in dev tooling

Exit criteria:

- sample programs compile and execute under QEMU with expected output

### M3: CH32V003 profile and demos

- profile-specific startup/linker/runtime glue
- UART output demo
- LED blink demo
- optional HD44780 demo

Exit criteria:

- reproducible firmware demo on CH32V003 target profile

## Engineering principles

- Prefer explicit errors over partial/undefined behavior
- Keep feature gates and support matrix documented
- Add tests before broadening grammar coverage
- Minimize third-party dependencies
- Prioritize correctness and readability over premature optimization
