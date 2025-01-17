package cxparsering

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/skycoin/cx/cx/ast"
	"github.com/skycoin/cx/cx/constants"
	globals2 "github.com/skycoin/cx/cx/globals"
	"github.com/skycoin/cx/cx/types"
	"github.com/skycoin/cx/cxparser/actions"
	constants2 "github.com/skycoin/cx/cxparser/constants"
	cxpartialparsing "github.com/skycoin/cx/cxparser/cxpartialparsing"
	"github.com/skycoin/cx/cxparser/util/profiling"
)

// preliminarystage performs a first pass for the CX cxgo. Globals, packages and
// custom types are added to `cxpartialparsing.Program`.
func Preliminarystage(srcStrs, srcNames []string) int {
	var prePkg *ast.CXPackage
	parseErrors := 0

	rePkg := regexp.MustCompile("package")
	rePkgName := regexp.MustCompile(`(^|[\s])package\s+([_a-zA-Z][_a-zA-Z0-9]*)`)
	reStrct := regexp.MustCompile("type")
	reStrctName := regexp.MustCompile(`(^|[\s])type\s+([_a-zA-Z][_a-zA-Z0-9]*)?\s`)

	reGlbl := regexp.MustCompile("var")
	reGlblName := regexp.MustCompile(`(^|[\s])var\s([_a-zA-Z][_a-zA-Z0-9]*)`)

	reBodyOpen := regexp.MustCompile("{")
	reBodyClose := regexp.MustCompile("}")

	reImp := regexp.MustCompile("import")
	reImpName := regexp.MustCompile(`(^|[\s])import\s+"([_a-zA-Z][_a-zA-Z0-9/-]*)"`)

	profiling.StartProfile("1. packages/structs")
	// 1. Identify all the packages and structs
	for srcI, srcStr := range srcStrs {
		srcName := srcNames[srcI]
		profiling.StartProfile(srcName)

		reader := strings.NewReader(srcStr)
		scanner := bufio.NewScanner(reader)
		var lineno = 0
		for scanner.Scan() {
			line := scanner.Bytes()
			lineno++

			// 1-a. Identify all the packages
			if loc := rePkg.FindIndex(line); loc != nil {
				 
				if match := rePkgName.FindStringSubmatch(string(line)); match != nil {
					if pkg, err := cxpartialparsing.Program.GetPackage(match[len(match)-1]); err != nil {
						// then it hasn't been added
						newPkg := ast.MakePackage(match[len(match)-1])
						cxpartialparsing.Program.AddPackage(newPkg)
						prePkg = newPkg
					} else {
						prePkg = pkg
					}
				}
			}

			// 1-b. Identify all the structs
			if loc := reStrct.FindIndex(line); loc != nil {
				 
				if match := reStrctName.FindStringSubmatch(string(line)); match != nil {
					if prePkg == nil {
						println(ast.CompilationError(srcName, lineno),
							"No package defined")
					} else if _, err := cxpartialparsing.Program.GetStruct(match[len(match)-1], prePkg.Name); err != nil {
						// then it hasn't been added
						strct := ast.MakeStruct(match[len(match)-1])
						prePkg.AddStruct(strct)
					}
				}
			}
		}
		profiling.StopProfile(srcName)
	} // for range srcStrs
	profiling.StopProfile("1. packages/structs")

	profiling.StartProfile("2. globals")
	// 2. Identify all global variables
	//    We also identify packages again, so we know to what
	//    package we're going to add the variable declaration to.
	for i, source := range srcStrs {
		profiling.StartProfile(srcNames[i])
		// inBlock needs to be 0 to guarantee that we're in the global scope
		var inBlock int

		scanner := bufio.NewScanner(strings.NewReader(source))
		for scanner.Scan() {
			line := scanner.Bytes()

			// Identify all the package imports.
			if loc := reImp.FindIndex(line); loc != nil {

				if match := reImpName.FindStringSubmatch(string(line)); match != nil {
					pkgName := match[len(match)-1]
					// Checking if `pkgName` already exists and if it's not a standard library package.
					if _, err := cxpartialparsing.Program.GetPackage(pkgName); err != nil && !constants2.IsCorePackage(pkgName) {
						// _, sourceCode, srcNames := ParseArgsForCX([]string{fmt.Sprintf("%s%s", SRCPATH, pkgName)}, false)
						_, sourceCode, fileNames := ast.ParseArgsForCX([]string{filepath.Join(globals2.SRCPATH, pkgName)}, false)
						ParseSourceCode(sourceCode, fileNames)
					}
				}
			}

			// we search for packages at the same time, so we can know to what package to add the global
			if loc := rePkg.FindIndex(line); loc != nil {
				 
				if match := rePkgName.FindStringSubmatch(string(line)); match != nil {
					if pkg, err := cxpartialparsing.Program.GetPackage(match[len(match)-1]); err != nil {
						// then it hasn't been added
						prePkg = ast.MakePackage(match[len(match)-1])
						cxpartialparsing.Program.AddPackage(prePkg)
					} else {
						prePkg = pkg
					}
				}
			}

			if locs := reBodyOpen.FindAllIndex(line, -1); locs != nil {
				inBlock++
			}
			if locs := reBodyClose.FindAllIndex(line, -1); locs != nil{
				inBlock--
			}

			// we could have this situation: {var local i32}
			// but we don't care about this, as the later passes will throw an error as it's invalid syntax

			if loc := rePkg.FindIndex(line); loc != nil {
				 
				if match := rePkgName.FindStringSubmatch(string(line)); match != nil {
					if pkg, err := cxpartialparsing.Program.GetPackage(match[len(match)-1]); err != nil {
						// it should be already present
						panic(err)
					} else {
						prePkg = pkg
					}
				}
			}

			// finally, if we read a "var" and we're in global scope, we add the global without any type
			// the type will be determined later on
			if loc := reGlbl.FindIndex(line); loc != nil && inBlock == 0{
				 
				if match := reGlblName.FindStringSubmatch(string(line)); match != nil {
					if _, err := prePkg.GetGlobal(match[len(match)-1]); err != nil {
						// then it hasn't been added
						arg := ast.MakeArgument(match[len(match)-1], "", 0)
						arg.Offset = types.InvalidPointer
						arg.Package = prePkg
						prePkg.AddGlobal(arg)
					}
				}
			}
		}
		profiling.StopProfile(srcNames[i])
	}
	profiling.StopProfile("2. globals")

	profiling.StartProfile("3. cxpartialparsing")

	for i, source := range srcStrs {
		profiling.StartProfile(srcNames[i])
		source = source + "\n"
		if len(srcNames) > 0 {
			cxpartialparsing.CurrentFileName = srcNames[i]
		}
		/*
			passone
		*/
		parseErrors += Passone(source)
		profiling.StopProfile(srcNames[i])
	}

	profiling.StopProfile("3. cxpartialparsing")
	return parseErrors
}

func AddInitFunction(prgrm *ast.CXProgram) error {
	mainPkg, err := prgrm.GetPackage(constants.MAIN_PKG)
	if err != nil {
		return err
	}

	initFn := ast.MakeFunction(constants.SYS_INIT_FUNC, actions.CurrentFile, actions.LineNo)
	mainPkg.AddFunction(initFn)

	//Init Expressions
	actions.FunctionDeclaration(prgrm, initFn, nil, nil, prgrm.SysInitExprs)

	if _, err := mainPkg.SelectFunction(constants.MAIN_FUNC); err != nil {
		return err
	}
	return nil
}
