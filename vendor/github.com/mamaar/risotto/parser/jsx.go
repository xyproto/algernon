package parser

import (
	"bytes"
	"github.com/mamaar/risotto/ast"
	"github.com/mamaar/risotto/token"
)

func (self *_parser) parseJSX() ast.Expression {
	switch self.token {
	case token.LESS:
		return self.parseJSXElement()
	case token.LEFT_BRACE:
		return self.parseJSXExpression()
	case token.SEMICOLON:
		self.next()
		return self.parsePrimaryExpression()
	}
	return self.parseJSXText()
}

func (self *_parser) parseJSXExpression() *ast.JSXExpression {
	variable := &ast.JSXExpression{}
	variable.Pos = self.expect(token.LEFT_BRACE)
	variable.Identifier = self.parseExpression()
	self.expect(token.RIGHT_BRACE)

	return variable
}

func (self *_parser) parseJSXText() *ast.JSXText {
	text := &ast.JSXText{
		Pos: self.idx,
	}
	buf := &bytes.Buffer{}

	for self.token != token.EOF && self.token != token.LEFT_BRACE && self.token != token.LESS {
		buf.WriteString(self.literal)
		if self.literal == "" {
			buf.WriteString(self.token.String())
		}
		self.rawNext()
	}
	text.Literal = buf.String()
	return text
}

func (self *_parser) parseJSXBlock() *ast.JSXBlock {
	jsx := &ast.JSXBlock{}

	jsx.OpeningElement = self.parseOpeningElement()
	if jsx.OpeningElement.SelfClosing {
		return jsx
	}

	for {
		j := self.parseJSX()
		if elem, ok := j.(*ast.JSXElement); ok && !elem.IsOpening {
			jsx.ClosingElement = elem
			break
		}
		jsx.Body = append(jsx.Body, j)
		if self.token == token.EOF {
			break
		}
	}
	return jsx
}

func (self *_parser) parseJSXElement() ast.Expression {
	self.expect(token.LESS)
	if self.token == token.SLASH {
		return self.parseClosingElement()
	}
	return self.parseJSXBlock()
}

func (self *_parser) parseClosingElement() *ast.JSXElement {
	closing := &ast.JSXElement{IsOpening: false}
	closing.LeftTag = self.expect(token.SLASH)

	if self.token == token.IDENTIFIER {
		closing.Name = self.parseIdentifier()
	}

	closing.RightTag = self.expect(token.GREATER)
	return closing
}

func (self *_parser) parseOpeningElement() *ast.JSXElement {
	open := &ast.JSXElement{IsOpening: true}
	open.LeftTag = self.idx

	if self.token == token.IDENTIFIER {
		open.Name = self.parseIdentifier()
	}
	for self.token == token.IDENTIFIER {
		open.PropertyList = append(open.PropertyList, self.parseJSXProperty())
	}

	if self.token == token.SLASH {
		self.next()
		open.RightTag = self.expect(token.GREATER)
		open.SelfClosing = true
		return open
	}
	open.RightTag = self.expect(token.GREATER)
	return open
}

func (self *_parser) parseJSXProperty() ast.Property {
	p := ast.Property{}

	p.Key = self.parseIdentifier().Name
	self.expect(token.ASSIGN)
	p.Value = self.parseJSXValue()

	return p
}

func (self *_parser) parseJSXValue() ast.Expression {
	if self.token == token.STRING {
		t := &ast.JSXText{
			Pos:     self.idx,
			Literal: self.literal[1 : len(self.literal)-1],
		}
		self.next()
		return t
	}

	return self.parseJSXExpression()
}
