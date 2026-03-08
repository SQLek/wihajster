package backend

import (
	"github.com/SQLek/wihajster/internal/tac"
	"github.com/SQLek/wihajster/internal/tac/cfg"
)

// FunctionView is a backend-oriented projection of TAC organized as CFG blocks.
//
// Code generators should iterate blocks rather than raw instruction slices.
type FunctionView struct {
	Name   string
	Blocks []cfg.BasicBlock
}

func BuildFunctionView(fn tac.Function) (FunctionView, error) {
	if err := tac.ValidateFunctionIR(fn); err != nil {
		return FunctionView{}, err
	}
	graph, err := cfg.Build(fn)
	if err != nil {
		return FunctionView{}, err
	}
	return FunctionView{Name: fn.Name, Blocks: graph.Blocks}, nil
}
