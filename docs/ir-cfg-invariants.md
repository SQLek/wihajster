# IR/CFG invariants

This document defines required invariants for function-level IR validation.

The current implementation still uses the `internal/tac` package name while the project migrates toward `internal/ir`. The invariants below apply to this IR form regardless of package naming.

## Required invariants

1. **Exactly one entry block**
   - A function has one entry block and it starts at instruction index `0`.

2. **Block label uniqueness**
   - Every block label is unique within a function.

3. **All successor labels/IDs resolve**
   - Every `jmp`/`br` successor label must point to a defined block label.

4. **Terminator is last in every block**
   - If a block contains `jmp`, `br`, or `ret`, that terminator must be the final instruction of the block.

5. **Non-terminator block edge policy**
   - A block without a terminator must have exactly one deterministic fallthrough successor (the next block in deterministic order), or an explicit edge policy if fallthrough is disabled.

6. **PHI edge cardinality (when PHI is enabled)**
   - For each PHI node, there must be exactly one incoming value for each predecessor edge.

7. **Deterministic block ordering**
   - Block ordering is deterministic and based on increasing instruction start index.

## Validation entry points

Function IR validation is required in:

- builder finalize paths (`AddFunction` and block setters)
- parser/decoder import path
- backend/evaluator entry points

This keeps diagnostics deterministic and prevents downstream stages from operating on malformed CFG structure.
