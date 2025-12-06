package instrument

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// CacheAnnotation represents a parsed cachewrap annotation
type CacheAnnotation struct {
	FuncName    string
	RecvType    string        // Receiver type for methods (e.g., "*repo")
	ParamNames  []string      // Parameters to use for cache key
	FuncDecl    *ast.FuncDecl // The function declaration
	PackageName string        // Package name
}

// ParseFile parses a Go source file and finds functions with cachewrap annotations
func ParseFile(filename string, src interface{}) ([]*CacheAnnotation, *ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}

	var annotations []*CacheAnnotation
	pkgName := node.Name.Name

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			return true
		}

		// Look for // cachewrap: annotation in comments
		for _, comment := range fn.Doc.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(text, "cachewrap:") {
				params := parseCacheParams(text)

				annotation := &CacheAnnotation{
					FuncName:    fn.Name.Name,
					ParamNames:  params,
					FuncDecl:    fn,
					PackageName: pkgName,
				}

				// Extract receiver type if it's a method
				if fn.Recv != nil && len(fn.Recv.List) > 0 {
					annotation.RecvType = getReceiverType(fn.Recv.List[0].Type)
				}

				annotations = append(annotations, annotation)
			}
		}

		return true
	})

	return annotations, node, fset, nil
}

// parseCacheParams extracts parameter names from cachewrap annotation
// Example: "cachewrap: id, email" -> ["id", "email"]
func parseCacheParams(comment string) []string {
	// Remove "cachewrap:" prefix
	paramsStr := strings.TrimPrefix(comment, "cachewrap:")
	paramsStr = strings.TrimSpace(paramsStr)

	if paramsStr == "" {
		return nil
	}

	// Split by comma
	parts := strings.Split(paramsStr, ",")
	var params []string
	for _, p := range parts {
		param := strings.TrimSpace(p)
		if param != "" {
			params = append(params, param)
		}
	}

	return params
}

// getReceiverType extracts the receiver type from AST
func getReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}
