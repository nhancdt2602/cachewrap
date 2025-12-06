package instrument

import (
	"go/parser"
	"go/token"
	"strings"

	"github.com/nhancdt2602/cachewrap/tool/rules"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// ParseCachewrapAnnotations scans a file for // cachewrap: annotations
func ParseCachewrapAnnotations(filePath string) ([]*rules.CachewrapRule, *dst.File, *decorator.Decorator, error) {
	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)
	f, err := dec.ParseFile(filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}

	var cacheRules []*rules.CachewrapRule

	dst.Inspect(f, func(n dst.Node) bool {
		funcDecl, ok := n.(*dst.FuncDecl)
		if !ok {
			return true
		}

		// Check if function has cache annotation in its doc comments
		if funcDecl.Decs.Start.All() != nil {
			for _, comment := range funcDecl.Decs.Start.All() {
				// Match cachewrap, cacheevict, or cacheclear
				if strings.Contains(comment, "cachewrap") ||
					strings.Contains(comment, "cacheevict") ||
					strings.Contains(comment, "cacheclear") {
					rule := parseCachewrapComment(comment, funcDecl, f)
					if rule != nil {
						cacheRules = append(cacheRules, rule)
					}
				}
			}
		}

		return true
	})

	return cacheRules, f, dec, nil
}

// parseCachewrapComment extracts cache rule from comment
// Supports:
//
//	"// cachewrap: id, email"           -> uses global cache
//	"// cachewrap[@redisCache]: id"     -> uses named cache
func parseCachewrapComment(comment string, funcDecl *dst.FuncDecl, file *dst.File) *rules.CachewrapRule {
	var annotationType string
	var idx int
	// trim prefix "// " until the first non-space character
	comment = strings.TrimPrefix(comment, "//")
	comment = strings.TrimSpace(comment)

	if i := strings.Index(comment, "cacheevict"); i == 0 {
		annotationType = "cacheevict"
		idx = i
	} else if i := strings.Index(comment, "cacheclear"); i == 0 {
		annotationType = "cacheclear"
		idx = i
	} else if i := strings.Index(comment, "cachewrap"); i == 0 {
		annotationType = "cachewrap"
		idx = i
	} else {
		return nil
	}

	// Parse format: cachewrap[@cacheName, prefix=X]: params
	remaining := strings.TrimSpace(comment[idx+len(annotationType):])

	var cacheInstance string
	var prefix string
	var paramsStr string

	if strings.HasPrefix(remaining, "[") {
		endBracket := strings.Index(remaining, "]")
		if endBracket == -1 {
			return nil
		}

		optionsStr := strings.TrimSpace(remaining[1:endBracket])
		options := strings.Split(optionsStr, ",")

		for _, opt := range options {
			opt = strings.TrimSpace(opt)

			if strings.HasPrefix(opt, "prefix=") {
				prefix = strings.TrimSpace(strings.TrimPrefix(opt, "prefix="))
			} else if strings.HasPrefix(opt, "@") {
				cacheInstance = opt
			} else if strings.Contains(opt, ".") {
				cacheInstance = opt
			}
		}

		remaining = strings.TrimSpace(remaining[endBracket+1:])
	}

	if strings.HasPrefix(remaining, ":") {
		paramsStr = strings.TrimSpace(remaining[1:])
	} else if annotationType != "cacheclear" {
		return nil
	}

	var paramNames []string
	if paramsStr != "" {
		parts := strings.Split(paramsStr, ",")
		for _, p := range parts {
			param := strings.TrimSpace(p)
			if param != "" {
				paramNames = append(paramNames, param)
			}
		}
	}

	rule := &rules.CachewrapRule{
		FuncName:      funcDecl.Name.Name,
		ParamNames:    paramNames,
		PackageName:   file.Name.Name,
		CacheInstance: cacheInstance,
		KeyPrefix:     prefix,
		IsEvict:       annotationType == "cacheevict",
		IsClear:       annotationType == "cacheclear",
	}

	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		rule.ReceiverType = getReceiverTypeName(funcDecl.Recv.List[0].Type)
	}

	return rule
}

func getReceiverTypeName(expr dst.Expr) string {
	switch t := expr.(type) {
	case *dst.StarExpr:
		if ident, ok := t.X.(*dst.Ident); ok {
			return "*" + ident.Name
		}
	case *dst.Ident:
		return t.Name
	}
	return ""
}
