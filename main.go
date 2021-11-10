package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/reusee/e4"
	"golang.org/x/tools/go/ast/astutil"
)

func main() {

	var err error
	defer func() {
		if err != nil {
			pt("%s\n", err.Error())
		}
	}()
	defer e4.Handle(&err)

	if len(os.Args) <= 1 {
		fmt.Printf("usage: %s [profile file]", os.Args[0])
		return
	}

	coverFilePath := os.Args[1]
	content, err := ioutil.ReadFile(coverFilePath)
	ce(err)

	lineByFile := make(map[string]map[int]bool)
	for _, line := range strings.Split(string(content), "\n")[1:] {
		line = strings.TrimSpace(line)
		if !strings.HasSuffix(line, " 0") {
			continue
		}

		parts := strings.Split(line, ":")
		file := parts[0]

		parts = strings.SplitN(parts[1], ".", 2)
		lineNum, err := strconv.Atoi(parts[0])
		ce(err)

		m, ok := lineByFile[file]
		if !ok {
			m = make(map[int]bool)
			lineByFile[file] = m
		}
		m[lineNum] = true
	}

	pkg, err := build.ImportDir(filepath.Dir(coverFilePath), build.FindOnly)
	ce(err)
	srcDir := pkg.Dir

	var files []string
	for path := range lineByFile {
		files = append(files, path)
	}
	sort.Strings(files)
	for _, path := range files {
		lineNums := lineByFile[path]
		path = filepath.Join(srcDir, filepath.Base(path))
		content, err := os.ReadFile(path)
		ce(err)
		fset := new(token.FileSet)
		file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
		ce(err)

		// exclude NOCOVER marked blocks
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if !nocoverPattern.MatchString(comment.Text) {
					continue
				}
				position := fset.Position(comment.Pos())
				delete(lineNums, position.Line)
				nodes, _ := astutil.PathEnclosingInterval(file, comment.Pos(), comment.End())
				blockStmt, ok := nodes[0].(*ast.BlockStmt)
				if !ok {
					continue
				}
				start := fset.Position(blockStmt.Lbrace).Line
				end := fset.Position(blockStmt.Rbrace).Line
				for num := range lineNums {
					if num >= start && num <= end {
						delete(lineNums, num)
					}
				}
			}
		}

		// exclude error checking blocks
		ast.Inspect(file, func(node ast.Node) bool {
			// if
			ifStmt, ok := node.(*ast.IfStmt)
			if !ok {
				return true
			}
			// bin expr
			cond, ok := ifStmt.Cond.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			// err
			ident, ok := cond.X.(*ast.Ident)
			if !ok || ident.Name != "err" {
				return true
			}
			// !=
			if cond.Op != token.NEQ {
				return true
			}
			// nil
			ident, ok = cond.Y.(*ast.Ident)
			if !ok || ident.Name != "nil" {
				return true
			}
			// single statement
			numStmts := len(ifStmt.Body.List)
			if numStmts > 1 {
				return true
			}
			// exclude
			start := fset.Position(ifStmt.Body.Lbrace).Line
			end := fset.Position(ifStmt.Body.Rbrace).Line
			for num := range lineNums {
				if num >= start && num <= end {
					delete(lineNums, num)
				}
			}
			return true
		})

		var nums []int
		for num := range lineNums {
			nums = append(nums, num)
		}
		sort.Ints(nums)
		for _, num := range nums {
			pt("%s # %d\n", path, num)
		}

	}

}

var nocoverPattern = regexp.MustCompile(`NOCOVER\s*$`)
