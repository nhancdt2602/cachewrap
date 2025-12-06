package instrument

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/nhancdt2602/cachewrap/tool/rules"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// InjectCaching modifies DST to add caching logic based on the cache rules
func InjectCaching(dstFile *dst.File, dec *decorator.Decorator, cacheRules []*rules.CachewrapRule) error {
	if len(cacheRules) == 0 {
		return nil
	}

	ctx := &InjectionContext{
		dec: dec,
	}

	for _, rule := range cacheRules {
		if err := injectCachingForFunc(dstFile, rule, ctx); err != nil {
			return fmt.Errorf("failed to inject caching for %s: %w", rule.FuncName, err)
		}
	}

	return nil
}

// InjectionContext holds contextual information for code injection
type InjectionContext struct {
	dec *decorator.Decorator
}

// injectCachingForFunc injects caching logic into a specific function
func injectCachingForFunc(dstFile *dst.File, rule *rules.CachewrapRule, ctx *InjectionContext) error {
	var targetFunc *dst.FuncDecl

	dst.Inspect(dstFile, func(n dst.Node) bool {
		if funcDecl, ok := n.(*dst.FuncDecl); ok {
			if funcDecl.Name.Name == rule.FuncName {
				if rule.ReceiverType != "" {
					if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
						recvType := getReceiverTypeName(funcDecl.Recv.List[0].Type)
						if recvType == rule.ReceiverType {
							targetFunc = funcDecl
							return false
						}
					}
				} else {
					if funcDecl.Recv == nil {
						targetFunc = funcDecl
						return false
					}
				}
			}
		}
		return true
	})

	if targetFunc == nil {
		return fmt.Errorf("function %s not found", rule.FuncName)
	}

	return injectCacheLogic(targetFunc, rule, ctx)
}

// injectCacheLogic adds cache checking code at the start of function
func injectCacheLogic(funcDecl *dst.FuncDecl, rule *rules.CachewrapRule, injCtx *InjectionContext) error {
	// First, convert function to use named return values (needed for defer to work)
	if err := ensureNamedReturns(funcDecl); err != nil {
		return err
	}

	// Handle different annotation types
	if rule.IsClear {
		return injectClearLogic(funcDecl, rule, injCtx)
	} else if rule.IsEvict {
		return injectEvictLogic(funcDecl, rule, injCtx)
	}

	return injectCachingLogic(funcDecl, rule, injCtx)
}

// injectCachingLogic injects cache get/set logic
func injectCachingLogic(funcDecl *dst.FuncDecl, rule *rules.CachewrapRule, injCtx *InjectionContext) error {
	prefix := rule.KeyPrefix
	if prefix == "" {
		// Fallback to function name
		prefix = rule.FuncName
	}

	// Generate cache key expression with param name-value pairs
	// MakeCacheKey("User", "id", id, "email", email)
	var cacheKeyParts []string
	cacheKeyParts = append(cacheKeyParts, fmt.Sprintf(`"%s"`, prefix))
	for _, paramName := range rule.ParamNames {
		cacheKeyParts = append(cacheKeyParts, fmt.Sprintf(`"%s"`, paramName))
		cacheKeyParts = append(cacheKeyParts, paramName)
	}

	var cacheGetter, cacheSetter string
	var needsCacheLookup bool

	if rule.CacheInstance == "" {
		// No cache instance specified, fallback to global cache
		cacheGetter = "cache.Get"
		cacheSetter = "cache.Set"
	} else if strings.HasPrefix(rule.CacheInstance, "@") {
		// Named cache: @redisCache
		// Generate: cache.GetCache("redisCache")
		needsCacheLookup = true
		cacheName := strings.TrimPrefix(rule.CacheInstance, "@")
		cacheGetter = fmt.Sprintf("cache.GetCache(%q)", cacheName)
		cacheSetter = fmt.Sprintf("cache.GetCache(%q)", cacheName)
	} else {
		cacheGetter = rule.CacheInstance + ".Get"
		cacheSetter = rule.CacheInstance + ".Set"
	}

	// Get the return type for type assertion
	var returnType string
	if funcDecl.Type != nil && funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		// Get first return type as string for type assertion
		firstReturn := funcDecl.Type.Results.List[0]
		returnType = getTypeString(firstReturn.Type)
	}
	if returnType == "" {
		returnType = "interface{}"
	}

	// Parse cache logic from code snippet
	// We inject: cache check + defer to store result
	// Use named return values to capture result in defer
	var snippet string

	if needsCacheLookup {
		// For named caches, we need to look up the cache instance first
		cacheName := strings.TrimPrefix(rule.CacheInstance, "@")
		snippet = fmt.Sprintf(`
package main
import "fmt"
func _() (retVal0 interface{}, retVal1 error) {
	cacheInst := cache.GetCache(%q)
	if cacheInst != nil {
		cacheKey := cache.MakeCacheKey(%s)
		if cached, ok := cacheInst.Get(cacheKey); ok {
			if typedValue, ok := cached.(%s); ok {
				fmt.Printf("[CACHE HIT] key: %%s\n", cacheKey)
				return typedValue, nil
			}
		}
		defer func() {
			if retVal1 == nil && retVal0 != nil {
				fmt.Printf("[CACHE SET] key: %%s\n", cacheKey)
				cacheInst.Set(cacheKey, retVal0)
			}
		}()
	}
}
`, cacheName, strings.Join(cacheKeyParts, ", "), returnType)
	} else {
		// For global or direct field reference
		snippet = fmt.Sprintf(`
package main
import "fmt"
func _() (retVal0 interface{}, retVal1 error) {
	cacheKey := cache.MakeCacheKey(%s)
	if cached, ok := %s(cacheKey); ok {
		if typedValue, ok := cached.(%s); ok {
			fmt.Printf("[CACHE HIT] key: %%s\n", cacheKey)
			return typedValue, nil
		}
	}
	defer func() {
		if retVal1 == nil && retVal0 != nil {
			fmt.Printf("[CACHE SET] key: %%s\n", cacheKey)
			%s(cacheKey, retVal0)
		}
	}()
}
`, strings.Join(cacheKeyParts, ", "), cacheGetter, returnType, cacheSetter)
	}

	dec := decorator.NewDecorator(nil)
	file, err := dec.Parse(snippet)
	if err != nil {
		return err
	}

	var fn *dst.FuncDecl
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*dst.FuncDecl); ok {
			fn = funcDecl
			break
		}
	}

	if fn != nil {
		if fn.Body != nil && len(fn.Body.List) > 0 && funcDecl.Body != nil {
			clonedStmts := make([]dst.Stmt, 0, len(fn.Body.List))
			for _, stmt := range fn.Body.List {
				clonedStmts = append(clonedStmts, dst.Clone(stmt).(dst.Stmt))
			}

			// Add line directive before injected code
			if len(clonedStmts) > 0 {
				clonedStmts[0].Decorations().Before = dst.NewLine
				clonedStmts[0].Decorations().Start.Append("//line <generated>:1")
			}

			// Add line directive before original code to restore line numbers
			if len(funcDecl.Body.List) > 0 {
				firstOriginal := funcDecl.Body.List[0]

				// Get the position of the first original statement
				pos := findStatementPosition(funcDecl.Body.List[0], injCtx)
				if pos.IsValid() {
					lineDirective := fmt.Sprintf("//line %s:%d:%d", pos.Filename, pos.Line, pos.Column)
					firstOriginal.Decorations().Before = dst.NewLine
					firstOriginal.Decorations().Start.Prepend(lineDirective)
				}
			}

			newStatements := make([]dst.Stmt, 0, len(clonedStmts)+len(funcDecl.Body.List))
			newStatements = append(newStatements, clonedStmts...)
			newStatements = append(newStatements, funcDecl.Body.List...)
			funcDecl.Body.List = newStatements
		}
	}

	return nil
}

// ensureNamedReturns converts a function to use named return values
// This allows defer statements to access and modify return values
func ensureNamedReturns(funcDecl *dst.FuncDecl) error {
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) == 0 {
		return nil // No return values
	}

	hasNames := false
	for _, field := range funcDecl.Type.Results.List {
		if len(field.Names) > 0 {
			hasNames = true
			break
		}
	}

	if hasNames {
		return nil
	}

	// Add names to return values: retVal0, retVal1, etc.
	idx := 0
	for _, field := range funcDecl.Type.Results.List {
		// Each field might have multiple types (e.g., func() (int, int))
		numTypes := 1
		if len(field.Names) == 0 && field.Type != nil {
			numTypes = 1
		}

		for i := 0; i < numTypes; i++ {
			name := &dst.Ident{
				Name: fmt.Sprintf("retVal%d", idx),
			}
			field.Names = append(field.Names, name)
			idx++
		}
	}

	return nil
}

// injectEvictLogic injects cache eviction logic
func injectEvictLogic(funcDecl *dst.FuncDecl, rule *rules.CachewrapRule, injCtx *InjectionContext) error {
	prefix := rule.KeyPrefix
	if prefix == "" {
		prefix = rule.FuncName
	}

	// Generate cache key expression
	var cacheKeyParts []string
	cacheKeyParts = append(cacheKeyParts, fmt.Sprintf(`"%s"`, prefix))
	for _, paramName := range rule.ParamNames {
		cacheKeyParts = append(cacheKeyParts, fmt.Sprintf(`"%s"`, paramName))
		cacheKeyParts = append(cacheKeyParts, paramName)
	}

	// Determine cache instance
	cacheDeleter := "cache.Delete"
	var needsCacheLookup bool
	var cacheName string

	if rule.CacheInstance != "" && strings.HasPrefix(rule.CacheInstance, "@") {
		needsCacheLookup = true
		cacheName = strings.TrimPrefix(rule.CacheInstance, "@")
	} else if rule.CacheInstance != "" {
		cacheDeleter = rule.CacheInstance + ".Delete"
	}

	// Generate eviction code
	// Note: retVal0 is the error return value (most evict functions return error)
	var snippet string
	if needsCacheLookup {
		snippet = fmt.Sprintf(`
package main
func _() (retVal0 error) {
	defer func() {
		if retVal0 == nil {
			cacheInst := cache.GetCache(%q)
			if cacheInst != nil {
				cacheKey := cache.MakeCacheKey(%s)
				cacheInst.Delete(cacheKey)
			}
		}
	}()
}
`, cacheName, strings.Join(cacheKeyParts, ", "))
	} else {
		snippet = fmt.Sprintf(`
package main
func _() (retVal0 error) {
	defer func() {
		if retVal0 == nil {
			cacheKey := cache.MakeCacheKey(%s)
			%s(cacheKey)
		}
	}()
}
`, strings.Join(cacheKeyParts, ", "), cacheDeleter)
	}

	return injectSnippet(funcDecl, snippet, injCtx)
}

// injectClearLogic injects cache clear logic
func injectClearLogic(funcDecl *dst.FuncDecl, rule *rules.CachewrapRule, injCtx *InjectionContext) error {
	prefix := rule.KeyPrefix
	if prefix == "" {
		return fmt.Errorf("cacheclear requires prefix parameter")
	}

	// Determine cache instance
	cacheClearer := "cache.ClearPrefix"
	var needsCacheLookup bool
	var cacheName string

	if rule.CacheInstance != "" && strings.HasPrefix(rule.CacheInstance, "@") {
		needsCacheLookup = true
		cacheName = strings.TrimPrefix(rule.CacheInstance, "@")
	} else if rule.CacheInstance != "" {
		cacheClearer = rule.CacheInstance + ".ClearPrefix"
	}

	// Generate clear code
	var snippet string
	if needsCacheLookup {
		snippet = fmt.Sprintf(`
package main
func _() {
	cacheInst := cache.GetCache(%q)
	if cacheInst != nil {
		cacheInst.ClearPrefix(%q)
	}
}
`, cacheName, prefix)
	} else {
		snippet = fmt.Sprintf(`
package main
func _() {
	%s(%q)
}
`, cacheClearer, prefix)
	}

	return injectSnippet(funcDecl, snippet, injCtx)
}

// injectSnippet is a helper to inject parsed snippet into function
func injectSnippet(funcDecl *dst.FuncDecl, snippet string, injCtx *InjectionContext) error {
	dec := decorator.NewDecorator(nil)
	file, err := dec.Parse(snippet)
	if err != nil {
		return err
	}

	// Extract and CLONE statements from parsed snippet
	if len(file.Decls) > 0 {
		if fn, ok := file.Decls[0].(*dst.FuncDecl); ok {
			if fn.Body != nil && len(fn.Body.List) > 0 && funcDecl.Body != nil {
				// Clone each statement to avoid DST node duplication
				clonedStmts := make([]dst.Stmt, 0, len(fn.Body.List))
				for _, stmt := range fn.Body.List {
					clonedStmts = append(clonedStmts, dst.Clone(stmt).(dst.Stmt))
				}

				// Add line directive before injected code
				if len(clonedStmts) > 0 {
					clonedStmts[0].Decorations().Before = dst.NewLine
					clonedStmts[0].Decorations().Start.Append("//line <generated>:1")
				}

				// Add line directive before original code to restore line numbers
				if len(funcDecl.Body.List) > 0 {
					firstOriginal := funcDecl.Body.List[0]
					pos := findStatementPosition(funcDecl.Body.List[0], injCtx)
					if pos.IsValid() {
						lineDirective := fmt.Sprintf("//line %s:%d:%d", pos.Filename, pos.Line, pos.Column)
						firstOriginal.Decorations().Before = dst.NewLine
						firstOriginal.Decorations().Start.Prepend(lineDirective)
					}
				}

				// Prepend injected statements
				newStatements := make([]dst.Stmt, 0, len(clonedStmts)+len(funcDecl.Body.List))
				newStatements = append(newStatements, clonedStmts...)
				newStatements = append(newStatements, funcDecl.Body.List...)
				funcDecl.Body.List = newStatements
			}
		}
	}

	return nil
}

// getTypeString converts a DST type expression to string
func getTypeString(expr dst.Expr) string {
	switch t := expr.(type) {
	case *dst.Ident:
		return t.Name
	case *dst.StarExpr:
		return "*" + getTypeString(t.X)
	case *dst.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *dst.MapType:
		return "map[" + getTypeString(t.Key) + "]" + getTypeString(t.Value)
	case *dst.SelectorExpr:
		return getTypeString(t.X) + "." + t.Sel.Name
	case *dst.InterfaceType:
		return "interface{}"
	default:
		return "interface{}"
	}
}

// findStatementPosition tries to find the position of a statement
func findStatementPosition(stmt dst.Stmt, injCtx *InjectionContext) token.Position {
	astNode := injCtx.dec.Ast.Nodes[stmt]
	if astNode == nil {
		return token.Position{}
	}

	pos := injCtx.dec.Fset.Position(astNode.Pos())

	// Check if there are decorations (comments) before the statement
	// If so, we want to point to where those decorations start
	if stmt.Decorations() != nil && len(stmt.Decorations().Start.All()) > 0 {
		// There are comments/decorations - the line directive should point
		// to where they start, not where the statement starts
		// Approximate by going back the number of decoration lines
		numDecoLines := len(stmt.Decorations().Start.All())
		if pos.Line > numDecoLines {
			pos.Line -= numDecoLines
		}
	}

	return pos
}
