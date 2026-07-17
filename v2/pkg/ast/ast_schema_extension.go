package ast

import "github.com/mwisner/graphql-go-tools/v2/pkg/lexer/position"

type SchemaExtension struct {
	SchemaDefinition

	ExtendLiteral position.Position
}
