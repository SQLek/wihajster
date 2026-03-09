package tac

import "fmt"

type BlockID int

type irBlock struct {
	id         BlockID
	start      int
	label      string
	terminator InstructionKind
	successors []BlockID
}

type irBlockIndex struct {
	blocks       []irBlock
	labelToBlock map[string]BlockID
}

func newIRBlockIndex(blocks []irBlock) *irBlockIndex {
	index := &irBlockIndex{blocks: blocks, labelToBlock: map[string]BlockID{}}
	for _, b := range blocks {
		if b.label == "" {
			continue
		}
		index.labelToBlock[b.label] = b.id
	}
	return index
}

func (idx *irBlockIndex) BlockByLabel(label string) (BlockID, bool) {
	id, ok := idx.labelToBlock[label]
	return id, ok
}

func (idx *irBlockIndex) EnsureLabel(id BlockID) string {
	if int(id) < 0 || int(id) >= len(idx.blocks) {
		return ""
	}
	if idx.blocks[id].label != "" {
		return idx.blocks[id].label
	}
	label := fmt.Sprintf(".B%d", id)
	idx.blocks[id].label = label
	idx.labelToBlock[label] = id
	return label
}

func ValidateFunctionIR(fn Function) error {
	if len(fn.Instructions) == 0 {
		return nil
	}

	labelDefs := map[string]int{}
	for i, inst := range fn.Instructions {
		if err := VerifyInstruction(inst); err != nil {
			return fmt.Errorf("function %s: invalid instruction at %d: %w", fn.Name, i, err)
		}
		if inst.Kind != InstructionLabel {
			continue
		}
		if _, exists := labelDefs[inst.Label]; exists {
			return fmt.Errorf("function %s: label %q defined multiple times", fn.Name, inst.Label)
		}
		labelDefs[inst.Label] = i
	}

	blocks, err := collectIRBlocks(fn, labelDefs)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return fmt.Errorf("function %s: expected exactly one entry block, found none", fn.Name)
	}
	if blocks[0].start != 0 {
		return fmt.Errorf("function %s: expected exactly one entry block at instruction 0", fn.Name)
	}

	for i, block := range blocks {
		for _, succ := range block.successors {
			if int(succ) < 0 || int(succ) >= len(blocks) {
				return fmt.Errorf("function %s: successor block %d is out of range", fn.Name, succ)
			}
		}
		if block.terminator == InstructionOp || block.terminator == InstructionLabel {
			if i+1 >= len(blocks) {
				return fmt.Errorf("function %s: block %q has no terminator and no fallthrough successor", fn.Name, displayBlockName(block, i))
			}
		}
	}

	return nil
}

func collectIRBlocks(fn Function, labelDefs map[string]int) ([]irBlock, error) {
	starts := []int{0}
	seenStarts := map[int]struct{}{0: {}}
	for i, inst := range fn.Instructions {
		switch inst.Kind {
		case InstructionJmp:
			idx, ok := labelDefs[inst.TrueLabel.Text]
			if !ok {
				return nil, fmt.Errorf("function %s: jump to undefined label %q", fn.Name, inst.TrueLabel.Text)
			}
			if _, exists := seenStarts[idx]; !exists {
				starts = append(starts, idx)
				seenStarts[idx] = struct{}{}
			}
			if i+1 < len(fn.Instructions) {
				if _, exists := seenStarts[i+1]; !exists {
					starts = append(starts, i+1)
					seenStarts[i+1] = struct{}{}
				}
			}
		case InstructionBr:
			trueIdx, ok := labelDefs[inst.TrueLabel.Text]
			if !ok {
				return nil, fmt.Errorf("function %s: branch to undefined label %q", fn.Name, inst.TrueLabel.Text)
			}
			falseIdx, ok := labelDefs[inst.FalseLabel.Text]
			if !ok {
				return nil, fmt.Errorf("function %s: branch to undefined label %q", fn.Name, inst.FalseLabel.Text)
			}
			if _, exists := seenStarts[trueIdx]; !exists {
				starts = append(starts, trueIdx)
				seenStarts[trueIdx] = struct{}{}
			}
			if _, exists := seenStarts[falseIdx]; !exists {
				starts = append(starts, falseIdx)
				seenStarts[falseIdx] = struct{}{}
			}
			if i+1 < len(fn.Instructions) {
				if _, exists := seenStarts[i+1]; !exists {
					starts = append(starts, i+1)
					seenStarts[i+1] = struct{}{}
				}
			}
		case InstructionRet:
			if i+1 < len(fn.Instructions) {
				if _, exists := seenStarts[i+1]; !exists {
					starts = append(starts, i+1)
					seenStarts[i+1] = struct{}{}
				}
			}
		}
	}

	for i := 0; i < len(starts)-1; i++ {
		for j := i + 1; j < len(starts); j++ {
			if starts[j] < starts[i] {
				starts[i], starts[j] = starts[j], starts[i]
			}
		}
	}

	blocks := make([]irBlock, 0, len(starts))
	startToID := map[int]BlockID{}
	for idx, start := range starts {
		end := len(fn.Instructions)
		if idx+1 < len(starts) {
			end = starts[idx+1]
		}
		if end <= start {
			return nil, fmt.Errorf("function %s: invalid block boundaries", fn.Name)
		}
		insts := fn.Instructions[start:end]
		last := insts[len(insts)-1]
		for i := 0; i < len(insts)-1; i++ {
			kind := insts[i].Kind
			if kind == InstructionJmp || kind == InstructionBr || kind == InstructionRet {
				return nil, fmt.Errorf("function %s: terminator must be last instruction in block starting at %d", fn.Name, start)
			}
		}
		b := irBlock{id: BlockID(idx), start: start, terminator: last.Kind}
		if insts[0].Kind == InstructionLabel {
			b.label = insts[0].Label
		}
		startToID[start] = b.id
		blocks = append(blocks, b)
	}

	lookup := newIRBlockIndex(blocks)
	for i := range blocks {
		end := len(fn.Instructions)
		if i+1 < len(blocks) {
			end = blocks[i+1].start
		}
		last := fn.Instructions[end-1]
		switch last.Kind {
		case InstructionJmp:
			targetStart := labelDefs[last.TrueLabel.Text]
			blocks[i].successors = []BlockID{startToID[targetStart]}
		case InstructionBr:
			trueTarget := startToID[labelDefs[last.TrueLabel.Text]]
			falseTarget := startToID[labelDefs[last.FalseLabel.Text]]
			if trueTarget == falseTarget {
				blocks[i].successors = []BlockID{trueTarget}
			} else {
				blocks[i].successors = []BlockID{trueTarget, falseTarget}
			}
		}

		if blocks[i].label != "" {
			if _, ok := lookup.BlockByLabel(blocks[i].label); !ok {
				return nil, fmt.Errorf("function %s: missing block mapping for label %q", fn.Name, blocks[i].label)
			}
		}
	}

	return blocks, nil
}

func displayBlockName(b irBlock, idx int) string {
	if b.label != "" {
		return b.label
	}
	return fmt.Sprintf("#%d", idx)
}
