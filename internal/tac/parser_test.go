package tac

import (
	"strings"
	"testing"
)

func TestParseModule_BasicExample(t *testing.T) {
	input := `.tac v1

func @add(%a:i32, %b:i32) -> i32 {
.L0:
  %t0 = add %a, %b
  ret %t0
}
`

	mod, err := ParseModule(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(mod.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(mod.Functions))
	}
	fn := mod.Functions[0]
	if fn.Name != "@add" {
		t.Fatalf("expected @add, got %s", fn.Name)
	}
	if len(fn.Instructions) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(fn.Instructions))
	}
}

func TestParseModule_DestinationRedefinition(t *testing.T) {
	input := `.tac v1

func @bad() -> i32 {
.L0:
  %t0 = const.i32 1
  %t0 = add %t0, 2
  ret %t0
}
`

	_, err := ParseModule(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected redefinition error")
	}
	if !strings.Contains(err.Error(), "redefined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseModule_OptionalOpcodeRejectedInM1(t *testing.T) {
	input := `.tac v1

func @mem_demo() -> i32 {
.L0:
  %t0 = alloca i32
  ret %t0
}
`

	_, err := ParseModule(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected optional opcode rejection")
	}
	if !strings.Contains(err.Error(), "not enabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteModule_RoundTrip(t *testing.T) {
	original := `.tac v1

func @main() -> i32 {
.L0:
  %t0 = const.i32 42
  ret %t0
}
`

	mod, err := ParseModule(strings.NewReader(original))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	var out strings.Builder
	if err := WriteModule(&out, mod); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	reparsed, err := ParseModule(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("unexpected reparse error: %v", err)
	}
	if len(reparsed.Functions) != 1 || reparsed.Functions[0].Name != "@main" {
		t.Fatalf("unexpected round-trip module: %+v", reparsed)
	}
}
