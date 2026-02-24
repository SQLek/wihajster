.tac v1

; OPTIONAL (M2+): memory-oriented TAC features.
; A strict M1 reader should reject these with explicit "not enabled in M1" diagnostics.
func @mem_demo() -> i32 {
.L0:
  %t0 = alloca i32
  store %t0, 7
  %t1 = load %t0
  ret %t1
}
