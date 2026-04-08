package tools

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// GoOutline lists top-level declarations in a Go source file (offline AST parse; not a full LSP).
type GoOutline struct{}

func (GoOutline) Name() string { return "GoOutline" }

func (GoOutline) IsDangerous() bool { return false }

func (GoOutline) Description() string {
	return "List top-level Go declarations (package, imports summary, types, consts, vars, funcs) in a file under the workspace. Uses go/parser only (no language server)."
}

func (GoOutline) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to a .go file relative to workspace or absolute under workspace",
			},
		},
		"required": []string{"path"},
	}
}

func (GoOutline) Execute(ctx context.Context, args map[string]any) (string, error) {
	rel := strings.TrimSpace(fmt.Sprint(args["path"]))
	if rel == "" {
		return "", fmt.Errorf("go_outline: path is required")
	}
	if !strings.HasSuffix(strings.ToLower(rel), ".go") {
		return "", fmt.Errorf("go_outline: path must be a .go file")
	}
	abs, err := resolveUnderWorkdir(ctx, rel)
	if err != nil {
		return "", err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, nil, parser.SkipObjectResolution)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("package ")
	b.WriteString(f.Name.Name)
	b.WriteByte('\n')
	if len(f.Imports) > 0 {
		b.WriteString(fmt.Sprintf("imports: %d\n", len(f.Imports)))
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name == nil {
				continue
			}
			recv := ""
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv = typeExprString(d.Recv.List[0].Type) + "."
			}
			b.WriteString(fmt.Sprintf("func %s%s\n", recv, d.Name.Name))
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name != nil {
						switch s.Type.(type) {
						case *ast.InterfaceType:
							b.WriteString(fmt.Sprintf("interface %s\n", s.Name.Name))
						case *ast.StructType:
							b.WriteString(fmt.Sprintf("struct %s\n", s.Name.Name))
						default:
							b.WriteString(fmt.Sprintf("type %s\n", s.Name.Name))
						}
					}
				case *ast.ValueSpec:
					for _, id := range s.Names {
						if id == nil {
							continue
						}
						tok := "var"
						if d.Tok == token.CONST {
							tok = "const"
						}
						b.WriteString(fmt.Sprintf("%s %s\n", tok, id.Name))
					}
				}
			}
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func typeExprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	default:
		return "?"
	}
}
