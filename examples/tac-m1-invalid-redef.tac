.tac v1

; INVALID example: destination %t0 is defined twice in one function.
func @bad() -> i32 {
.L0:
  %t0 = const.i32 1
  %t0 = add %t0, 2
  ret %t0
}
