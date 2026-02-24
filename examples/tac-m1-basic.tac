.tac v1

func @add(%a:i32, %b:i32) -> i32 {
.L0:
  %t0 = add %a, %b
  ret %t0
}

func @main() -> i32 {
.L0:
  %t0 = const.i32 2
  %t1 = const.i32 40
  %t2 = call @add(%t0, %t1)
  %t3 = eq %t2, 42
  br %t3, .L1, .L2
.L1:
  ret %t2
.L2:
  %t4 = const.i32 1
  ret %t4
}
