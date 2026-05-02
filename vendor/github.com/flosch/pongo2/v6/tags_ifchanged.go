package pongo2

import (
	"bytes"
)

// ifchangedState holds the per-execution mutable state for an {% ifchanged %} tag.
type ifchangedState struct {
	lastValues  []*Value
	lastContent []byte
}

type tagIfchangedNode struct {
	watchedExpr []IEvaluator
	thenWrapper *NodeWrapper
	elseWrapper *NodeWrapper
}

// getState returns the per-execution ifchanged state for this node,
// creating it on first access. Each template execution gets its own
// independent state via ctx.tagState.
func (node *tagIfchangedNode) getState(ctx *ExecutionContext) *ifchangedState {
	if s, ok := ctx.tagState[node].(*ifchangedState); ok {
		return s
	}
	s := &ifchangedState{}
	ctx.tagState[node] = s
	return s
}

func (node *tagIfchangedNode) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	state := node.getState(ctx)

	if len(node.watchedExpr) == 0 {
		// Check against own rendered body

		buf := bytes.NewBuffer(make([]byte, 0, 1024)) // 1 KiB
		err := node.thenWrapper.Execute(ctx, buf)
		if err != nil {
			return err
		}

		bufBytes := buf.Bytes()

		changed := !bytes.Equal(state.lastContent, bufBytes)
		if changed {
			state.lastContent = bufBytes
		}

		if changed {
			// Rendered content changed, output it
			writer.Write(bufBytes)
		} else if node.elseWrapper != nil {
			// Content hasn't changed, render else block if present
			err := node.elseWrapper.Execute(ctx, writer)
			if err != nil {
				return err
			}
		}
	} else {
		nowValues := make([]*Value, 0, len(node.watchedExpr))
		for _, expr := range node.watchedExpr {
			val, err := expr.Evaluate(ctx)
			if err != nil {
				return err
			}
			nowValues = append(nowValues, val)
		}

		// Compare old to new values now
		changed := len(state.lastValues) == 0

		for idx, oldVal := range state.lastValues {
			if !oldVal.EqualValueTo(nowValues[idx]) {
				changed = true
				break // we can stop here because ONE value changed
			}
		}

		state.lastValues = nowValues

		if changed {
			// Render thenWrapper
			err := node.thenWrapper.Execute(ctx, writer)
			if err != nil {
				return err
			}
		} else if node.elseWrapper != nil {
			// Render elseWrapper
			err := node.elseWrapper.Execute(ctx, writer)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func tagIfchangedParser(doc *Parser, start *Token, arguments *Parser) (INodeTag, *Error) {
	ifchangedNode := &tagIfchangedNode{}

	for arguments.Remaining() > 0 {
		// Parse condition
		expr, err := arguments.ParseExpression()
		if err != nil {
			return nil, err
		}
		ifchangedNode.watchedExpr = append(ifchangedNode.watchedExpr, expr)
	}

	if arguments.Remaining() > 0 {
		return nil, arguments.Error("Ifchanged-arguments are malformed.", nil)
	}

	// Wrap then/else-blocks
	wrapper, endargs, err := doc.WrapUntilTag("else", "endifchanged")
	if err != nil {
		return nil, err
	}
	ifchangedNode.thenWrapper = wrapper

	if endargs.Count() > 0 {
		return nil, endargs.Error("Arguments not allowed here.", nil)
	}

	if wrapper.Endtag == "else" {
		// if there's an else in the if-statement, we need the else-Block as well
		wrapper, endargs, err = doc.WrapUntilTag("endifchanged")
		if err != nil {
			return nil, err
		}
		ifchangedNode.elseWrapper = wrapper

		if endargs.Count() > 0 {
			return nil, endargs.Error("Arguments not allowed here.", nil)
		}
	}

	return ifchangedNode, nil
}

func init() {
	RegisterTag("ifchanged", tagIfchangedParser)
}
