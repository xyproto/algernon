package pongo2

type tagCycleValue struct {
	node  *tagCycleNode
	value *Value
}

type tagCycleNode struct {
	position *Token
	args     []IEvaluator
	asName   string
	silent   bool
}

func (cv *tagCycleValue) String() string {
	return cv.value.String()
}

// cycleIdx returns the current cycle index for this node from the execution
// context and advances it. Each template execution gets its own independent
// cycle state via ctx.tagState.
func (node *tagCycleNode) cycleIdx(ctx *ExecutionContext) int {
	idx, _ := ctx.tagState[node].(int)
	ctx.tagState[node] = idx + 1
	return idx
}

func (node *tagCycleNode) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	idx := node.cycleIdx(ctx)
	item := node.args[idx%len(node.args)]

	val, err := item.Evaluate(ctx)
	if err != nil {
		return err
	}

	if t, ok := val.Interface().(*tagCycleValue); ok {
		// {% cycle cycleitem %} — advance the referenced cycle node
		refIdx := t.node.cycleIdx(ctx)
		item := t.node.args[refIdx%len(t.node.args)]

		val, err := item.Evaluate(ctx)
		if err != nil {
			return err
		}

		t.value = val

		if !t.node.silent {
			writer.WriteString(val.String())
		}
	} else {
		// Regular call

		cycleValue := &tagCycleValue{
			node:  node,
			value: val,
		}

		if node.asName != "" {
			ctx.Private[node.asName] = cycleValue
		}
		if !node.silent {
			writer.WriteString(val.String())
		}
	}

	return nil
}

// HINT: We're not supporting the old comma-separated list of expressions argument-style
func tagCycleParser(doc *Parser, start *Token, arguments *Parser) (INodeTag, *Error) {
	cycleNode := &tagCycleNode{
		position: start,
	}

	for arguments.Remaining() > 0 {
		node, err := arguments.ParseExpression()
		if err != nil {
			return nil, err
		}
		cycleNode.args = append(cycleNode.args, node)

		if arguments.MatchOne(TokenKeyword, "as") != nil {
			// as

			nameToken := arguments.MatchType(TokenIdentifier)
			if nameToken == nil {
				return nil, arguments.Error("Name (identifier) expected after 'as'.", nil)
			}
			cycleNode.asName = nameToken.Val

			if arguments.MatchOne(TokenIdentifier, "silent") != nil {
				cycleNode.silent = true
			}

			// Now we're finished
			break
		}
	}

	if arguments.Remaining() > 0 {
		return nil, arguments.Error("Malformed cycle-tag.", nil)
	}

	return cycleNode, nil
}

func init() {
	RegisterTag("cycle", tagCycleParser)
}
