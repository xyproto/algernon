package generator

import (
	"fmt"
	"github.com/mamaar/risotto/ast"
	"reflect"
)

func (g *generator) generateStatement(stmt ast.Statement, dcls []ast.Declaration) error {
	defer func() {
		if g.isElseStatement {
			g.isElseStatement = false
		}
	}()
	switch stmt.(type) {
	case *ast.VariableStatement:
		return g.variableStatement(stmt.(*ast.VariableStatement))
	case *ast.ExpressionStatement:
		return g.expressionStatement(stmt.(*ast.ExpressionStatement))
	case *ast.BlockStatement:
		return g.blockStatement(stmt.(*ast.BlockStatement), dcls)
	case *ast.ReturnStatement:
		return g.returnStatement(stmt.(*ast.ReturnStatement))
	case *ast.EmptyStatement:
		return g.emptyStatement(stmt.(*ast.EmptyStatement))
	case *ast.IfStatement:
		return g.ifStatement(stmt.(*ast.IfStatement))
	case *ast.ThrowStatement:
		return g.throwStatement(stmt.(*ast.ThrowStatement))
	case *ast.ForStatement:
		return g.forStatement(stmt.(*ast.ForStatement))
	case *ast.ForInStatement:
		return g.forInStatement(stmt.(*ast.ForInStatement))
	case *ast.BranchStatement:
		return g.branchStatement(stmt.(*ast.BranchStatement))
	case *ast.TryStatement:
		return g.tryStatement(stmt.(*ast.TryStatement))
	case *ast.CatchStatement:
		return g.catchStatement(stmt.(*ast.CatchStatement))
	case *ast.WhileStatement:
		return g.whileStatement(stmt.(*ast.WhileStatement))
	case *ast.DoWhileStatement:
		return g.doWhileStatement(stmt.(*ast.DoWhileStatement))
	default:
		return fmt.Errorf("Statement is not defined <%v>", reflect.TypeOf(stmt))
	}
}

func (g *generator) doWhileStatement(d *ast.DoWhileStatement) error {
	g.writeLine("do ")
	if err := g.generateStatement(d.Body, nil); err != nil {
		return err
	}
	g.write(" while(")
	g.descentExpression()
	if err := g.generateExpression(d.Test); err != nil {
		return err
	}
	g.ascentExpression()
	g.write(");")
	return nil
}

func (g *generator) whileStatement(w *ast.WhileStatement) error {
	g.writeLine("while(")
	if err := g.generateExpression(w.Test); err != nil {
		return err
	}
	g.write(") ")
	return g.generateStatement(w.Body, nil)
}

func (g *generator) catchStatement(c *ast.CatchStatement) error {
	g.write(" catch (")
	if err := g.identifier(c.Parameter); err != nil {
		return err
	}
	g.write(") ")
	return g.generateStatement(c.Body, nil)
}

func (g *generator) tryStatement(t *ast.TryStatement) error {
	g.writeLine("try ")
	if err := g.generateStatement(t.Body, nil); err != nil {
		return err
	}
	if t.Catch != nil {
		if err := g.generateStatement(t.Catch, nil); err != nil {
			return err
		}
	}
	if t.Finally != nil {
		g.write(" finally ")
		if err := g.generateStatement(t.Finally, nil); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) branchStatement(b *ast.BranchStatement) error {
	g.writeLine(b.Token.String())
	if b.Label != nil {
		if err := g.generateExpression(b.Label); err != nil {
			return err
		}
	}
	g.write(";")
	return nil
}

func (g *generator) forInStatement(f *ast.ForInStatement) error {
	g.writeLine("for(var ")
	if err := g.generateExpression(f.Into); err != nil {
		return err
	}
	g.write(" in ")
	if err := g.generateExpression(f.Source); err != nil {
		return err
	}
	g.write(") ")
	return g.generateStatement(f.Body, nil)
}

func (g *generator) forStatement(f *ast.ForStatement) error {
	g.writeLine("for(")
	g.isInInitializer = true
	if err := g.generateExpression(f.Initializer); err != nil {
		return err
	}
	g.isInInitializer = false
	g.write("; ")
	if err := g.generateExpression(f.Test); err != nil {
		return err
	}
	g.write("; ")
	if err := g.generateExpression(f.Update); err != nil {
		return nil
	}
	g.write(") ")
	return g.generateStatement(f.Body, nil)
}

func (g *generator) throwStatement(t *ast.ThrowStatement) error {
	g.writeLine("throw ")

	if err := g.generateExpression(t.Argument); err != nil {
		return err
	}
	g.write(";")
	return nil
}

func (g *generator) ifStatement(i *ast.IfStatement) error {
	if !g.isElseStatement {
		g.write("\n")
	}
	g.write("if (")
	g.descentExpression()
	if err := g.generateExpression(i.Test); err != nil {
		return err
	}
	g.ascentExpression()

	g.write(") ")

	if err := g.generateStatement(i.Consequent, nil); err != nil {
		return err
	}

	if i.Alternate != nil {
		g.write(" else ")
		g.isElseStatement = true
		return g.generateStatement(i.Alternate, nil)
	}

	return nil
}

func (g *generator) emptyStatement(r *ast.EmptyStatement) error {
	return nil
}

func (g *generator) returnStatement(r *ast.ReturnStatement) error {
	g.writeIndentation("return ")
	g.descentExpression()
	if err := g.generateExpression(r.Argument); err != nil {
		return err
	}
	g.ascentExpression()
	g.write(";")
	return nil
}

func (g *generator) blockStatement(b *ast.BlockStatement, dcls []ast.Declaration) error {
	g.write("{\n")
	g.indentLevel++

	for _, stmt := range b.List {
		if err := g.generateStatement(stmt, nil); err != nil {
			return err
		}
	}
	for _, dcl := range dcls {
		if err := g.generateDeclaration(dcl); err != nil {
			return err
		}
	}

	g.indentLevel--
	g.writeAlone("}")
	return nil
}

func (g *generator) expressionStatement(e *ast.ExpressionStatement) error {
	g.writeAlone("")
	if err := g.generateExpression(e.Expression); err != nil {
		return err
	}
	g.write(";")
	return nil
}

func (g *generator) variableStatement(v *ast.VariableStatement) error {
	g.writeLine("var ")

	for i, vexp := range v.List {
		if len(v.List) > 1 {
			g.writeIndentation("")
		}

		if err := g.generateExpression(vexp); err != nil {
			return err
		}

		if i < len(v.List)-1 {
			g.write(",")
			g.write("\n")
		} else {
			g.write(";")
		}
	}
	return nil
}
