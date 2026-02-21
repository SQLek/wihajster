# Language v0 Contract

This document defines the explicit C subset accepted by v0. Anything outside this contract must fail deterministically with a clear parser or semantic diagnostic.

| Supported | Rejected (must produce explicit parser/semantic errors) |
|---|---|
| **Types**: `int` (32-bit, `int32_t`-equivalent), `char`, and pointers to those (`int*`, `char*`, nested pointers). | `short`, `long`, `long long`, unsigned/signed variants beyond `char`/`int`, `_Bool`, `void` objects, `struct`, `union`, `enum`, floating-point types (`float`, `double`, `long double`), complex/imaginary types. |
| **Declarations**: local/global scalar declarations for supported types. | Arrays (local/global), VLAs, aggregate/object initializers beyond scalar basics, designated initializers, bit-fields, storage-class/qualifier complexity not in v0. |
| **Expressions**: integer/char literals, identifiers, unary `- ! ~ * &`, binary `+ - * / % << >> & | ^ && ||`, comparisons (`== != < <= > >=`), assignment (`=`). | Increment/decrement (`++ --`), comma operator, ternary `?:`, compound assignment (`+=` etc.), casts (unless explicitly added later), `sizeof`, member access (`.` `->`), subscripting `[]` (since arrays are unsupported). |
| **Statements**: expression statements, block statements, `if/else`, `while`, `for`, `return`. | `switch/case/default`, `goto`/labels, `do/while`, `break`/`continue` (until explicitly specified), empty declaration+statement extensions not in grammar. |
| **Functions**: function definitions and calls, non-variadic only; no function pointer support. | Variadic functions (`...`), function pointer declarators/types/calls, old-style K&R declarations, nested functions. |
| **Preprocessor**: minimal object-like `#define NAME value` constants only (or no preprocessor support in strict mode). | Function-like macros, token pasting/stringification, conditional compilation (`#if`, `#ifdef`, ...), `#include`, `#pragma`, macro recursion semantics. |
| **Diagnostics policy**: unsupported syntax/features are rejected deterministically at parse or semantic phase with stable messages. | Silent acceptance, best-effort fallback, or deferred “backend-only” failure for unsupported front-end constructs. |

## Deterministic rejection requirements

For every rejected feature category above, implement an explicit diagnostic path:

- **Parser errors** for syntactic forms that are out of grammar in v0 (for example `switch`, `goto`, `struct` declarations, designated initializer tokens).
- **Semantic errors** for syntactically parseable but unsupported constructs (for example type categories or declarator forms temporarily parsed but not implemented).
- Include feature-specific wording, e.g. `error: v0 does not support switch statements` rather than generic `unexpected token` when feasible.
- Keep diagnostics stable and testable (golden tests/snapshots preferred).

## Notes

- This is an explicit contract for Milestone M1 scope discipline.
- New features must be added here before they are accepted by default.
