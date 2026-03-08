package cfg

import (
	"fmt"
	"sort"

	"github.com/SQLek/wihajster/internal/tac"
)

type Graph struct {
	Function tac.Function
	Blocks   []BasicBlock
}

type BasicBlock struct {
	ID           int
	Start        int
	Instructions []tac.Instruction
	Predecessors []int
	Successors   []int
}

func Build(fn tac.Function) (Graph, error) {
	if err := tac.ValidateFunctionIR(fn); err != nil {
		return Graph{}, err
	}
	if len(fn.Instructions) == 0 {
		return Graph{Function: fn}, nil
	}

	labelDefs := map[string]int{}
	for i, inst := range fn.Instructions {
		if inst.Kind != tac.InstructionLabel {
			continue
		}
		if _, exists := labelDefs[inst.Label]; exists {
			return Graph{}, fmt.Errorf("function %s: label %q defined multiple times", fn.Name, inst.Label)
		}
		labelDefs[inst.Label] = i
	}

	leaders := map[int]struct{}{0: {}}
	for i, inst := range fn.Instructions {
		switch inst.Kind {
		case tac.InstructionJmp:
			target, ok := labelDefs[inst.TrueLabel.Text]
			if !ok {
				return Graph{}, fmt.Errorf("function %s: jump to undefined label %q", fn.Name, inst.TrueLabel)
			}
			leaders[target] = struct{}{}
			if i+1 < len(fn.Instructions) {
				leaders[i+1] = struct{}{}
			}
		case tac.InstructionBr:
			trueTarget, ok := labelDefs[inst.TrueLabel.Text]
			if !ok {
				return Graph{}, fmt.Errorf("function %s: branch to undefined label %q", fn.Name, inst.TrueLabel)
			}
			falseTarget, ok := labelDefs[inst.FalseLabel.Text]
			if !ok {
				return Graph{}, fmt.Errorf("function %s: branch to undefined label %q", fn.Name, inst.FalseLabel)
			}
			leaders[trueTarget] = struct{}{}
			leaders[falseTarget] = struct{}{}
			if i+1 < len(fn.Instructions) {
				leaders[i+1] = struct{}{}
			}
		}
	}

	order := make([]int, 0, len(leaders))
	for idx := range leaders {
		order = append(order, idx)
	}
	sort.Ints(order)

	blocks := make([]BasicBlock, 0, len(order))
	startToBlock := map[int]int{}
	for i, start := range order {
		end := len(fn.Instructions)
		if i+1 < len(order) {
			end = order[i+1]
		}
		b := BasicBlock{
			ID:           i,
			Start:        start,
			Instructions: append([]tac.Instruction(nil), fn.Instructions[start:end]...),
		}
		if len(b.Instructions) == 0 {
			return Graph{}, fmt.Errorf("function %s: empty basic block at instruction %d", fn.Name, start)
		}
		blocks = append(blocks, b)
		startToBlock[start] = i
	}

	for i := range blocks {
		last := blocks[i].Instructions[len(blocks[i].Instructions)-1]
		switch last.Kind {
		case tac.InstructionJmp:
			targetStart := labelDefs[last.TrueLabel.Text]
			targetID := startToBlock[targetStart]
			blocks[i].Successors = append(blocks[i].Successors, targetID)
		case tac.InstructionBr:
			trueID := startToBlock[labelDefs[last.TrueLabel.Text]]
			falseID := startToBlock[labelDefs[last.FalseLabel.Text]]
			blocks[i].Successors = append(blocks[i].Successors, trueID)
			if falseID != trueID {
				blocks[i].Successors = append(blocks[i].Successors, falseID)
			}
		case tac.InstructionRet:
			// no successors
		default:
			if i+1 < len(blocks) {
				blocks[i].Successors = append(blocks[i].Successors, i+1)
			}
		}

		for j := 0; j+1 < len(blocks[i].Instructions); j++ {
			kind := blocks[i].Instructions[j].Kind
			if kind == tac.InstructionJmp || kind == tac.InstructionBr || kind == tac.InstructionRet {
				return Graph{}, fmt.Errorf("function %s: terminator must be last instruction in block starting at %d", fn.Name, blocks[i].Start)
			}
		}
	}

	for i := range blocks {
		for _, succ := range blocks[i].Successors {
			blocks[succ].Predecessors = append(blocks[succ].Predecessors, i)
		}
	}

	return Graph{Function: fn, Blocks: blocks}, nil
}
