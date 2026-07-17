package astnormalization

import (
	"github.com/mwisner/graphql-go-tools/v2/pkg/ast"
	"github.com/mwisner/graphql-go-tools/v2/pkg/astvisitor"
)

func removeMergedTypeExtensions(walker *astvisitor.Walker) {
	visitor := removeMergedTypeExtensionsVisitor{
		Walker: walker,
	}
	walker.RegisterLeaveDocumentVisitor(&visitor)
}

type removeMergedTypeExtensionsVisitor struct {
	*astvisitor.Walker
}

func (r *removeMergedTypeExtensionsVisitor) LeaveDocument(operation, definition *ast.Document) {
	operation.RemoveMergedTypeExtensions()
}
