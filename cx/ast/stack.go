package ast

import (
	"fmt"

	"github.com/skycoin/cx/cx/types"
)

func stackValueHeader(fileName string, fileLine int) string {
	return fmt.Sprintf("%s:%d", fileName, fileLine)
}

// PrintStack ...
func (cxprogram *CXProgram) PrintStack() {
	fmt.Println()
	fmt.Println("===Callstack===")

	// we're going backwards in the stack
	fp := cxprogram.Stack.Pointer

	for c := cxprogram.CallCounter; c != types.InvalidPointer; c-- {
		op := cxprogram.CallStack[c].Operator
		fp -= op.Size

		var dupNames []string

		fmt.Printf(">>> %s()\n", op.Name)

		for _, inp := range op.Inputs {
			fmt.Println("ProgramInput")
			fmt.Printf("\t%s : %s() : %s\n", stackValueHeader(inp.ArgDetails.FileName, inp.ArgDetails.FileLine), op.Name, GetPrintableValue(cxprogram, fp, inp))

			dupNames = append(dupNames, inp.Package.Name+inp.Name)
		}

		for _, out := range op.Outputs {
			fmt.Println("ProgramOutput")
			fmt.Printf("\t%s : %s() : %s\n", stackValueHeader(out.ArgDetails.FileName, out.ArgDetails.FileLine), op.Name, GetPrintableValue(cxprogram, fp, out))

			dupNames = append(dupNames, out.Package.Name+out.Name)
		}

		// fmt.Println("Expressions")
		exprs := ""
		for _, expr := range op.Expressions {
			cxAtomicOp, _, _, err := cxprogram.GetOperation(expr)
			if err != nil {
				panic(err)
			}
			for _, inp := range cxAtomicOp.Inputs {
				if inp.Name == "" || cxAtomicOp.Operator == nil {
					continue
				}
				var dup bool
				for _, name := range dupNames {
					if name == inp.Package.Name+inp.Name {
						dup = true
						break
					}
				}
				if dup {
					continue
				}

				// fmt.Println("\t", inp.Name, "\t", ":", "\t", GetPrintableValue(fp, inp))
				// exprs += fmt.Sprintln("\t", stackValueHeader(inp.FileName, inp.FileLine), "\t", ":", "\t", GetPrintableValue(fp, inp))

				exprs += fmt.Sprintf("\t%s : %s() : %s\n", stackValueHeader(inp.ArgDetails.FileName, inp.ArgDetails.FileLine), cxAtomicOp.GetOperatorName(), GetPrintableValue(cxprogram, fp, inp))

				dupNames = append(dupNames, inp.Package.Name+inp.Name)
			}

			for _, out := range cxAtomicOp.Outputs {
				if out.Name == "" || cxAtomicOp.Operator == nil {
					continue
				}
				var dup bool
				for _, name := range dupNames {
					if name == out.Package.Name+out.Name {
						dup = true
						break
					}
				}
				if dup {
					continue
				}

				// fmt.Println("\t", out.Name, "\t", ":", "\t", GetPrintableValue(fp, out))
				// exprs += fmt.Sprintln("\t", stackValueHeader(out.FileName, out.FileLine), ":", GetPrintableValue(fp, out))

				exprs += fmt.Sprintf("\t%s : %s() : %s\n", stackValueHeader(out.ArgDetails.FileName, out.ArgDetails.FileLine), cxAtomicOp.GetOperatorName(), GetPrintableValue(cxprogram, fp, out))

				dupNames = append(dupNames, out.Package.Name+out.Name)
			}
		}

		if len(exprs) > 0 {
			fmt.Println("Expressions\n", exprs)
		}
	}
}
