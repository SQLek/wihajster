# Parser v0 Coverage Matrix (M1)

Status values:
- `Done`: implemented in parser with tests.
- `Deferred`: intentionally rejected with explicit parser diagnostics.

| Area | Feature | Status | Validation |
|---|---|---|---|
| Types | `int`, `char`, `void` | Done | parser unit + integration tests |
| Types | Pointer declarators (`*`) | Done | params/global/local declaration tests |
| Types | `struct`, `union`, `enum`, floating point | Deferred | explicit unsupported diagnostics tests |
| Declarations | Global scalar declarations | Done | integration tests |
| Declarations | Local scalar declarations | Done | integration + block recovery tests |
| Declarations | Arrays / multi-declarator extensions | Deferred | explicit unsupported diagnostics tests |
| Expressions | literals (`int`, `char`) | Done | parser tests |
| Expressions | unary `+ - ! ~ * &` | Done | expression tests |
| Expressions | binary ops from v0 subset | Done | precedence tests |
| Expressions | assignment `=` | Done | assignment associativity test |
| Expressions | function calls | Done | call parsing test |
| Expressions | ternary/comma/casts/compound-assign | Deferred | explicit unsupported diagnostics tests |
| Statements | block / expr / `if` / `while` / `return` | Done | integration tests |
| Statements | `for` | Done | integration tests |
| Statements | `switch`, `goto`, `do`, `break`, `continue` | Deferred | explicit unsupported diagnostics tests |
| Functions | non-variadic definitions + params | Done | integration tests |
| Functions | variadic functions | Deferred | explicit unsupported diagnostics tests |
| Diagnostics | fatal lexer precedence | Done | parser unit test |
| Diagnostics | statement-level recovery | Done | parser unit + integration tests |
