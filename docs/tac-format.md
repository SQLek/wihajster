# TAC Text Format v1

This document defines a deterministic text format for the compiler's Three-Address Code (TAC) IR.
It is intended for parser/sema/TAC testing in M1 and for backend verification in later milestones.

TAC is intentionally simple and linear, but this project requires a stricter contract so that snapshots,
golden tests, and diagnostics remain stable.

## Version header (required)

Every TAC file must start with exactly one version line:

```text
.tac v1
```

A reader must reject files with:
- missing header,
- duplicated header,
- unknown major version.

Minor/feature negotiation is out of scope for v1.

## File structure and ordering rules

A TAC file is a **module** with zero or more function definitions:

1. Version header.
2. Optional module metadata lines (future use; currently unused).
3. Function blocks.

Function ordering must be deterministic:
- Primary rule: preserve source declaration order.
- Tie-break for generated/synthetic functions: lexicographic by function name.

Within each function:
1. Function header (`func @name(...) -> type {`).
2. Zero or more labels and instructions.
3. Closing brace (`}`).

Label ordering must follow control-flow linearization order chosen by the TAC builder. The builder must be deterministic for the same input.

## Naming rules

### Identifiers

- Function names: `@` prefix + C identifier-like name, e.g. `@main`, `@sum_i32`.
- Parameters: `%` prefix + identifier, e.g. `%a`, `%argc`.
- Temporaries: `%t` prefix + decimal index, e.g. `%t0`, `%t1`, `%t2`.
- Labels: `.L` prefix + decimal index, e.g. `.L0`, `.L1`.

### Uniqueness and single-definition rule

Within one function:
- Every temporary destination must be defined exactly once.
- A destination name cannot be redefined.
- Labels must be unique.

This is a strict SSA-like constraint for destinations and is required in v1 for easier testing and analysis.
(Full SSA constructs such as phi are optional and deferred.)

## Types (v1)

Core types used in M1:
- `i32`
- `i8`
- `ptr`
- `void`

Optional for later milestones:
- explicit pointer-to-type spelling (e.g. `ptr<i32>`),
- backend-specific widened integer types.

## Grammar (EBNF)

```ebnf
file            = header, newline, { module_line, newline }, { function, newline } ;
header          = ".tac v1" ;
module_line     = comment | metadata ;
metadata        = ".meta", ws, ident, "=", value ;

function        = "func", ws, func_name, "(", [ params ], ")", ws, "->", ws, type_name, ws, "{", newline,
                  { block_line, newline },
                  "}" ;

params          = param, { ",", ws, param } ;
param           = value_name, ":", type_name ;

block_line      = comment | label_def | instruction ;
label_def       = label, ":" ;

instruction     = simple_instr
                | term_instr ;

simple_instr    = [ dest, ws, "=", ws ], opcode, [ ws, operands ] ;
term_instr      = "jmp", ws, label
                | "br", ws, value, ",", ws, label, ",", ws, label
                | "ret", [ ws, value ] ;

operands        = operand, { ",", ws, operand } ;
operand         = value | label | type_name ;

value           = value_name | integer | string_lit | "null" ;
dest            = value_name ;
value_name      = "%", ident | "%t", digits ;
func_name       = "@", ident ;
label           = ".L", digits ;

type_name       = "i32" | "i8" | "ptr" | "void" | ident ;
opcode          = ident ;

comment         = ";", { any_char_except_newline } ;

ident           = ( letter | "_" ), { letter | digit | "_" } ;
digits          = digit, { digit } ;
integer         = [ "-" ], digits ;
string_lit      = '"', { string_char }, '"' ;

ws              = { " " | "\t" } ;
newline         = "\n" ;
```

Notes:
- `opcode` is constrained further by the opcode table below.
- Whitespace is insignificant except as separator.
- Comments start with `;` and run to end of line.

## Opcode set and operand signatures

Readers must reject unknown opcodes unless the parser is explicitly in extension mode.

### Opcode grammar (normative)

```ebnf
instr_const_i32   = dest, ws, "=", ws, "const.i32", ws, integer ;
instr_const_i8    = dest, ws, "=", ws, "const.i8", ws, integer ;
instr_copy        = dest, ws, "=", ws, "copy", ws, value ;

instr_binop       = dest, ws, "=", ws, binop, ws, value, ",", ws, value ;
binop             = "add" | "sub" | "mul" | "div_s" | "mod_s"
                  | "and" | "or" | "xor" | "shl" | "shr_s"
                  | "eq" | "ne" | "lt_s" | "le_s" | "gt_s" | "ge_s" ;

instr_unop        = dest, ws, "=", ws, unop, ws, value ;
unop              = "neg" | "not" | "logic_not" ;

instr_call        = [ dest, ws, "=", ws ], "call", ws, func_name, "(", [ arg_list ], ")" ;
arg_list          = value, { ",", ws, value } ;

instr_jmp         = "jmp", ws, label ;
instr_br          = "br", ws, value, ",", ws, label, ",", ws, label ;
instr_ret         = "ret", [ ws, value ] ;
```

Any opcode not matched by the grammar above is non-core and must be treated as optional/extension behavior.

### Core opcodes (required for M1)

| Opcode | Form | Meaning |
|---|---|---|
| `const.i32` | `%dst = const.i32 <int>` | Materialize 32-bit integer constant. |
| `const.i8` | `%dst = const.i8 <int>` | Materialize 8-bit integer constant. |
| `copy` | `%dst = copy <value>` | Copy value to new destination. |
| `add` | `%dst = add <a>, <b>` | Integer add. |
| `sub` | `%dst = sub <a>, <b>` | Integer subtract. |
| `mul` | `%dst = mul <a>, <b>` | Integer multiply. |
| `div_s` | `%dst = div_s <a>, <b>` | Signed integer division. |
| `mod_s` | `%dst = mod_s <a>, <b>` | Signed remainder. |
| `and` | `%dst = and <a>, <b>` | Bitwise and. |
| `or` | `%dst = or <a>, <b>` | Bitwise or. |
| `xor` | `%dst = xor <a>, <b>` | Bitwise xor. |
| `shl` | `%dst = shl <a>, <b>` | Left shift. |
| `shr_s` | `%dst = shr_s <a>, <b>` | Arithmetic right shift. |
| `eq` | `%dst = eq <a>, <b>` | Equality comparison (`i32` boolean result 0/1). |
| `ne` | `%dst = ne <a>, <b>` | Inequality comparison. |
| `lt_s` | `%dst = lt_s <a>, <b>` | Signed less-than. |
| `le_s` | `%dst = le_s <a>, <b>` | Signed less-or-equal. |
| `gt_s` | `%dst = gt_s <a>, <b>` | Signed greater-than. |
| `ge_s` | `%dst = ge_s <a>, <b>` | Signed greater-or-equal. |
| `neg` | `%dst = neg <a>` | Arithmetic negate. |
| `not` | `%dst = not <a>` | Bitwise not. |
| `logic_not` | `%dst = logic_not <a>` | Logical not (`0 -> 1`, non-zero -> 0). |
| `call` | `%dst = call @fn(<args>)` or `call @fn(<args>)` | Call function, optionally capturing result. |
| `jmp` | `jmp .Lx` | Unconditional branch. |
| `br` | `br <cond>, .Ltrue, .Lfalse` | Conditional branch on non-zero condition. |
| `ret` | `ret` or `ret <value>` | Return from function. |

### Optional opcodes (defer until needed)

These are valid extension points but not required for M1 parser/sema/TAC milestone:

- Memory ops: `alloca`, `load`, `store`, `gep`.
- Cast/convert ops: `zext`, `sext`, `trunc`, `bitcast`.
- SSA merge op: `phi`.
- Backend lowering helpers for M2+ if proven necessary.

If these appear before implementation, emit explicit deterministic errors such as:
`error: opcode 'phi' is recognized but not enabled in milestone M1`.

## Determinism requirements

For test stability:
- Preserve destination numbering in creation order (`%t0`, `%t1`, ...).
- Preserve label numbering in creation order (`.L0`, `.L1`, ...).
- Emit canonical opcode spellings listed above.
- Emit one instruction per line.
- Do not reorder instructions inside a block.

## Validation checklist for TAC writers/readers

- Header present and valid.
- Function names unique in module.
- Labels unique per function.
- Destination single-definition rule enforced.
- Referenced labels exist.
- `ret` appears in all exit paths (or explicit verifier error).
- Unsupported opcode/type emits clear error.

## Relationship to milestones

- **M1 (required):** format parser, verifier basics, core arithmetic/control-flow opcodes, deterministic text emission.
- **M2 (optional extension):** richer pointer/memory and lowering helpers needed for RV32 backend.
- **M3 (optional extension):** profile-specific conventions for embedded targets may add metadata fields without breaking v1 core grammar.
