package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	//"io/ioutil"
	//"github.com/maximilien/i18n4go/common"
)

type Translation struct {
	ID          string `json:"id"`
	Translation string `json:"translation"`
	Modified    bool   `json:"modified"`
}

type builder struct {
	str []string
}

func main() {
	var pkg = flag.String("pkg", "", "help message for flagname")
	var dir = flag.String("dir", "", "help message for dir")
	var out = flag.String("o", "", "help message for out")
	flag.Parse()

	var pkgName = *pkg

	var absDirPath = *dir
	if !filepath.IsAbs(absDirPath) {
		absDirPath = filepath.Join(os.Getenv("PWD"), absDirPath)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, absDirPath, nil, parser.ParseComments|parser.AllErrors)

	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	var conf types.Config
	conf.Importer = importer.Default()

	files := make([]*ast.File, 0, len(pkgs[pkgName].Files))
	for _, file := range pkgs[pkgName].Files {
		files = append(files, file)
	}
	_, err = conf.Check(pkgName, fset, files, &info)
	if err != nil {
		fmt.Println("checked", err)
		return
	}

	b := builder{}

	for _, astFile := range pkgs[pkgName].Files {
		err = b.extractString(astFile, &info, fset)
		if err != nil {
			fmt.Println(err)
		}
	}

	err = b.write(*out)
	if err != nil {
		fmt.Errorf("got error during write (%v)\n", err)
	}
}

func (b *builder) extractString(f *ast.File, info *types.Info, fset *token.FileSet) error {
	ast.Inspect(f, func(n ast.Node) bool {

		switch x := n.(type) {
		case *ast.CallExpr:
			if len(x.Args) > 0 {
				tc := info.TypeOf(x.Fun)
				//fmt.Println(tc.String())
				if strings.HasSuffix(tc.String(), "github.com/nicksnyder/go-i18n/i18n.TranslateFunc") {
					str := x.Args[0].(*ast.BasicLit)
					b.str = append(b.str, str.Value[1:len(str.Value)-1])
					//fmt.Printf("got string %s\n", str.Value[1:len(str.Value)-1])
				}
			}
		}
		return true
	})

	return nil
}

func (b *builder) write(out string) error {
	tr := []Translation{}
	for _, str := range b.str {
		tr = append(tr, Translation{
			ID:          str,
			Translation: str,
		})
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	m, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		return err
	}
	f.Write(m)
	return nil
}
