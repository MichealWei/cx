package ast

import (
	"github.com/skycoin/cx/cx/constants"
	"github.com/skycoin/cx/cx/types"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func deserializeRaw(byts []byte, offset types.Pointer, size types.Pointer, item interface{}) {
	_, err := encoder.DeserializeRaw(byts[offset:offset+size], item)
	if err != nil {
		panic(err)
	}
}

func serializeString(name string, s *SerializedCXProgram) (int64, int64) {
	if name == "" {
		return int64(-1), int64(-1)
	}

	size := encoder.Size(name)

	off, found := s.StringsMap[name]
	if found {
		return int64(off), int64(size)
	}
	off = int64(len(s.Strings))
	s.Strings = append(s.Strings, encoder.Serialize(name)...)
	s.StringsMap[name] = off

	return int64(off), int64(size)
}

func indexPackage(pkg *CXPackage, s *SerializedCXProgram) {
	if _, found := s.PackagesMap[pkg.Name]; !found {
		s.PackagesMap[pkg.Name] = int64(len(s.PackagesMap))
	} else {
		panic("duplicated package in serialization process")
	}
}

func indexStruct(strct *CXStruct, s *SerializedCXProgram) {
	strctName := strct.Package.Name + "." + strct.Name
	if _, found := s.StructsMap[strctName]; !found {
		s.StructsMap[strctName] = int64(len(s.StructsMap))
	} else {
		panic("duplicated struct in serialization process")
	}
}

func indexFunction(fn *CXFunction, s *SerializedCXProgram) {
	fnName := fn.Package.Name + "." + fn.Name
	if _, found := s.FunctionsMap[fnName]; !found {
		s.FunctionsMap[fnName] = int64(len(s.FunctionsMap))
	} else {
		panic("duplicated function in serialization process")
	}
}

func serializeBoolean(val bool) int64 {
	if val {
		return 1
	}
	return 0
}

func serializePointers(pointers []types.Pointer, s *SerializedCXProgram) (int64, int64) {
	if len(pointers) == 0 {
		return int64(-1), int64(-1)
	}
	off := len(s.Integers)
	l := len(pointers)

	ints := make([]int64, l)
	for i, pointer := range pointers {
		ints[i] = int64(pointer)
	}

	s.Integers = append(s.Integers, ints...)

	return int64(off), int64(l)
}

func serializeIntegers(ints []int, s *SerializedCXProgram) (int64, int64) {
	if len(ints) == 0 {
		return int64(-1), int64(-1)
	}
	off := len(s.Integers)
	l := len(ints)

	ints32 := make([]int64, l)
	for i, int := range ints {
		ints32[i] = int64(int)
	}

	s.Integers = append(s.Integers, ints32...)

	return int64(off), int64(l)
}

func serializeArgument(arg *CXArgument, s *SerializedCXProgram) int {
	s.Arguments = append(s.Arguments, serializedArgument{})
	argOff := len(s.Arguments) - 1

	sNil := int64(-1)

	s.Arguments[argOff].NameOffset, s.Arguments[argOff].NameSize = serializeString(arg.Name, s)

	s.Arguments[argOff].Type = int64(arg.Type)

	if arg.StructType == nil {
		s.Arguments[argOff].StructTypeOffset = sNil
	} else {
		strctName := arg.StructType.Package.Name + "." + arg.StructType.Name
		if strctOff, found := s.StructsMap[strctName]; found {
			s.Arguments[argOff].StructTypeOffset = int64(strctOff)
		} else {
			panic("struct reference not found")
		}
	}

	s.Arguments[argOff].Size = int64(arg.Size)
	s.Arguments[argOff].TotalSize = int64(arg.TotalSize)
	s.Arguments[argOff].Offset = int64(arg.Offset)
	// s.Arguments[argOff].IndirectionLevels = int64(arg.IndirectionLevels)
	s.Arguments[argOff].DereferenceLevels = int64(arg.DereferenceLevels)

	s.Arguments[argOff].DeclarationSpecifiersOffset,
		s.Arguments[argOff].DeclarationSpecifiersSize = serializeIntegers(arg.DeclarationSpecifiers, s)

	s.Arguments[argOff].IsSlice = serializeBoolean(arg.IsSlice)
	// s.Arguments[argOff].IsPointer = serializeBoolean(arg.IsPointer)
	// s.Arguments[argOff].IsReference = serializeBoolean(arg.IsReference)

	s.Arguments[argOff].IsStruct = serializeBoolean(arg.IsStruct)
	s.Arguments[argOff].IsInnerArg = serializeBoolean(arg.IsInnerArg)
	s.Arguments[argOff].IsLocalDeclaration = serializeBoolean(arg.IsLocalDeclaration)
	s.Arguments[argOff].IsShortDeclaration = serializeBoolean(arg.IsShortAssignmentDeclaration)
	s.Arguments[argOff].PreviouslyDeclared = serializeBoolean(arg.PreviouslyDeclared)

	s.Arguments[argOff].PassBy = int64(arg.PassBy)
	s.Arguments[argOff].DoesEscape = serializeBoolean(arg.DoesEscape)

	s.Arguments[argOff].LengthsOffset, s.Arguments[argOff].LengthsSize = serializePointers(arg.Lengths, s)
	s.Arguments[argOff].IndexesOffset, s.Arguments[argOff].IndexesSize = serializeSliceOfArguments(arg.Indexes, s)
	s.Arguments[argOff].FieldsOffset, s.Arguments[argOff].FieldsSize = serializeSliceOfArguments(arg.Fields, s)
	s.Arguments[argOff].InputsOffset, s.Arguments[argOff].InputsSize = serializeSliceOfArguments(arg.Inputs, s)
	s.Arguments[argOff].OutputsOffset, s.Arguments[argOff].OutputsSize = serializeSliceOfArguments(arg.Outputs, s)

	if _, found := s.PackagesMap[arg.Package.Name]; found {
		s.Arguments[argOff].PackageName = arg.Package.Name
	} else {
		panic("package reference not found")
	}

	return argOff
}

func serializeSliceOfArguments(args []*CXArgument, s *SerializedCXProgram) (int64, int64) {
	if len(args) == 0 {
		return int64(-1), int64(-1)
	}
	idxs := make([]int, len(args))
	for i, arg := range args {
		idxs[i] = serializeArgument(arg, s)
	}
	return serializeIntegers(idxs, s)
}

func serializeCalls(calls []CXCall, s *SerializedCXProgram) (int64, int64) {
	if len(calls) == 0 {
		return int64(-1), int64(-1)
	}
	idxs := make([]int, len(calls))
	for i, call := range calls {
		idxs[i] = serializeCall(&call, s)
	}
	return serializeIntegers(idxs, s)

}

func serializeExpression(prgrm *CXProgram, expr *CXExpression, s *SerializedCXProgram) int {
	s.Expressions = append(s.Expressions, serializedExpression{})
	exprOff := len(s.Expressions) - 1
	sExpr := &s.Expressions[exprOff]

	sNil := int64(-1)

	sExpr.Type = int64(expr.Type)
	switch expr.Type {
	case CX_LINE:
		_, _, cxLine, err := prgrm.GetOperation(expr)
		if err != nil {
			panic(err)
		}

		sExpr.ExpressionType = int64(expr.ExpressionType)
		sExpr.FileName = cxLine.FileName
		sExpr.LineNumber = int64(cxLine.LineNumber)
		sExpr.LineStr = cxLine.LineStr

	case CX_ATOMIC_OPERATOR:
		cxAtomicOp, _, _, err := prgrm.GetOperation(expr)
		if err != nil {
			panic(err)
		}

		if cxAtomicOp.Operator == nil {
			// then it's a declaration
			sExpr.OperatorOffset = sNil
			sExpr.IsNative = serializeBoolean(false)
			sExpr.OpCode = int64(-1)
		} else if cxAtomicOp.Operator.IsBuiltIn() {
			sExpr.OperatorOffset = sNil
			sExpr.IsNative = serializeBoolean(true)
			sExpr.OpCode = int64(cxAtomicOp.Operator.AtomicOPCode)
		} else {
			sExpr.IsNative = serializeBoolean(false)
			sExpr.OpCode = sNil

			opName := cxAtomicOp.Operator.Package.Name + "." + cxAtomicOp.Operator.Name
			if opOff, found := s.FunctionsMap[opName]; found {
				sExpr.OperatorOffset = int64(opOff)
			}
		}

		sExpr.InputsOffset, sExpr.InputsSize = serializeSliceOfArguments(cxAtomicOp.Inputs, s)
		sExpr.OutputsOffset, sExpr.OutputsSize = serializeSliceOfArguments(cxAtomicOp.Outputs, s)

		sExpr.LabelOffset, sExpr.LabelSize = serializeString(cxAtomicOp.Label, s)
		sExpr.ThenLines = int64(cxAtomicOp.ThenLines)
		sExpr.ElseLines = int64(cxAtomicOp.ElseLines)

		sExpr.ExpressionType = int64(expr.ExpressionType)

		fnName := cxAtomicOp.Function.Package.Name + "." + cxAtomicOp.Function.Name
		if fnOff, found := s.FunctionsMap[fnName]; found {
			sExpr.FunctionOffset = int64(fnOff)
		} else {
			panic("function reference not found")
		}

		if _, found := s.PackagesMap[cxAtomicOp.Package.Name]; found {
			sExpr.PackageName = cxAtomicOp.Package.Name
		} else {
			panic("package reference not found")
		}

	}
	return exprOff
}

func serializeCall(call *CXCall, s *SerializedCXProgram) int {
	s.Calls = append(s.Calls, serializedCall{})
	callOff := len(s.Calls) - 1
	serializedCall := &s.Calls[callOff]

	opName := call.Operator.Package.Name + "." + call.Operator.Name
	if opOff, found := s.FunctionsMap[opName]; found {
		serializedCall.OperatorOffset = int64(opOff)
		serializedCall.Line = int64(call.Line)
		serializedCall.FramePointer = int64(call.FramePointer)
	} else {
		panic("function reference not found")
	}

	return callOff
}

func serializeStructArguments(strct *CXStruct, s *SerializedCXProgram) {
	strctName := strct.Package.Name + "." + strct.Name
	if strctOff, found := s.StructsMap[strctName]; found {
		sStrct := &s.Structs[strctOff]
		sStrct.FieldsOffset, sStrct.FieldsSize = serializeSliceOfArguments(strct.Fields, s)
	} else {
		panic("struct reference not found")
	}
}

func serializeFunctionArguments(fn *CXFunction, s *SerializedCXProgram) {
	fnName := fn.Package.Name + "." + fn.Name
	if fnOff, found := s.FunctionsMap[fnName]; found {
		sFn := &s.Functions[fnOff]

		sFn.InputsOffset, sFn.InputsSize = serializeSliceOfArguments(fn.Inputs, s)
		sFn.OutputsOffset, sFn.OutputsSize = serializeSliceOfArguments(fn.Outputs, s)
		sFn.ListOfPointersOffset, sFn.ListOfPointersSize = serializeSliceOfArguments(fn.ListOfPointers, s)
	} else {
		panic("function reference not found")
	}
}

func serializePackageName(pkg *CXPackage, s *SerializedCXProgram) {
	sPkg := &s.Packages[s.PackagesMap[pkg.Name]]
	sPkg.NameOffset, sPkg.NameSize = serializeString(pkg.Name, s) //Change Name to String
}

func serializeStructName(strct *CXStruct, s *SerializedCXProgram) {
	strctName := strct.Package.Name + "." + strct.Name
	sStrct := &s.Structs[s.StructsMap[strctName]]
	sStrct.NameOffset, sStrct.NameSize = serializeString(strct.Name, s) //Change Name to String
}

func serializeFunctionName(fn *CXFunction, s *SerializedCXProgram) {
	fnName := fn.Package.Name + "." + fn.Name
	if off, found := s.FunctionsMap[fnName]; found {
		sFn := &s.Functions[off]
		sFn.NameOffset, sFn.NameSize = serializeString(fn.Name, s) //Change Name to String
	} else {
		panic("function reference not found")
	}
}

func serializePackageGlobals(pkg *CXPackage, s *SerializedCXProgram) {
	if pkgOff, found := s.PackagesMap[pkg.Name]; found {
		sPkg := &s.Packages[pkgOff]
		sPkg.GlobalsOffset, sPkg.GlobalsSize = serializeSliceOfArguments(pkg.Globals, s)
	} else {
		panic("package reference not found")
	}
}

func serializePackageImports(pkg *CXPackage, s *SerializedCXProgram) {
	l := len(pkg.Imports)
	if l == 0 {
		s.Packages[s.PackagesMap[pkg.Name]].ImportsOffset = int64(-1)
		s.Packages[s.PackagesMap[pkg.Name]].ImportsSize = int64(-1)
		return
	}
	imps := make([]int64, l)
	count := 0
	for _, imp := range pkg.Imports {
		if idx, found := s.PackagesMap[imp.Name]; found {
			imps[count] = int64(idx)
		} else {
			panic("import package reference not found")
		}

		count++
	}

	s.Packages[s.PackagesMap[pkg.Name]].ImportsOffset = int64(len(s.Integers))
	s.Packages[s.PackagesMap[pkg.Name]].ImportsSize = int64(l)
	s.Integers = append(s.Integers, imps...)
}

func serializeStructPackage(strct *CXStruct, s *SerializedCXProgram) {
	strctName := strct.Package.Name + "." + strct.Name
	if _, found := s.PackagesMap[strct.Package.Name]; found {
		if off, found := s.StructsMap[strctName]; found {
			sStrct := &s.Structs[off]
			sStrct.PackageName = strct.Package.Name
		} else {
			panic("struct reference not found")
		}
	} else {
		panic("struct's package reference not found")
	}
}

func serializeFunctionPackage(fn *CXFunction, s *SerializedCXProgram) {
	fnName := fn.Package.Name + "." + fn.Name
	if _, found := s.PackagesMap[fn.Package.Name]; found {
		if off, found := s.FunctionsMap[fnName]; found {
			sFn := &s.Functions[off]
			sFn.PackageName = fn.Package.Name
		} else {
			panic("function reference not found")
		}
	} else {
		panic("function's package reference not found")
	}
}

func serializePackageIntegers(pkg *CXPackage, s *SerializedCXProgram) {
	if pkgOff, found := s.PackagesMap[pkg.Name]; found {
		sPkg := &s.Packages[pkgOff]

		if pkg.CurrentFunction == nil {
			// package has no functions
			sPkg.CurrentFunctionName = ""
		} else {
			currFnName := pkg.CurrentFunction.Package.Name + "." + pkg.CurrentFunction.Name

			if _, found := s.FunctionsMap[currFnName]; found {
				sPkg.CurrentFunctionName = currFnName
			} else {
				panic("function reference not found")
			}
		}

		if pkg.CurrentStruct == nil {
			// package has no structs
			sPkg.CurrentStructName = ""
		} else {
			currStrctName := pkg.CurrentStruct.Package.Name + "." + pkg.CurrentStruct.Name

			if _, found := s.StructsMap[currStrctName]; found {
				sPkg.CurrentStructName = currStrctName
			} else {
				panic("struct reference not found")
			}
		}
	} else {
		panic("package reference not found")
	}
}

func serializeStructIntegers(strct *CXStruct, s *SerializedCXProgram) {
	strctName := strct.Package.Name + "." + strct.Name
	if off, found := s.StructsMap[strctName]; found {
		sStrct := &s.Structs[off]
		sStrct.Size = int64(strct.Size)
	} else {
		panic("struct reference not found")
	}
}

func serializeFunctionIntegers(fn *CXFunction, s *SerializedCXProgram) {
	fnName := fn.Package.Name + "." + fn.Name
	if off, found := s.FunctionsMap[fnName]; found {
		sFn := &s.Functions[off]
		sFn.Size = int64(fn.Size)
		sFn.Length = int64(fn.LineCount)
	} else {
		panic("function reference not found")
	}
}

// initSerialization initializes the
// container for our serialized cx program.
// Program memory is also added here to our container
// if memory is to be included.
func initSerialization(prgrm *CXProgram, s *SerializedCXProgram, includeDataMemory, useCompression bool) {
	s.PackagesMap = make(map[string]int64)
	s.StructsMap = make(map[string]int64)
	s.FunctionsMap = make(map[string]int64)
	s.StringsMap = make(map[string]int64)

	s.Calls = make([]serializedCall, prgrm.CallCounter)
	s.Packages = make([]serializedPackage, len(prgrm.Packages))

	// If use compression, whole memory will be included
	// If not and if includeDataMemory, only data segment memory will be included
	if useCompression {
		s.Memory = prgrm.Memory
	} else if includeDataMemory && len(prgrm.Memory) != 0 {
		s.DataSegmentMemory = prgrm.Memory[prgrm.Data.StartsAt : prgrm.Data.StartsAt+prgrm.Data.Size]
	}

	var numStrcts int
	var numFns int

	for _, pkg := range prgrm.Packages {
		numStrcts += len(pkg.Structs)
		numFns += len(pkg.Functions)
	}

	s.Structs = make([]serializedStruct, numStrcts)
	s.Functions = make([]serializedFunction, numFns)
	// args and exprs need to be appended as they are found
}

// serializeProgram serializes
// program of cx program.
func serializeProgram(prgrm *CXProgram, s *SerializedCXProgram) {
	s.Program = serializedProgram{}
	sPrgrm := &s.Program
	sPrgrm.PackagesOffset = int64(0)
	sPrgrm.PackagesSize = int64(len(prgrm.Packages))

	if _, found := s.PackagesMap[prgrm.CurrentPackage.Name]; found {
		sPrgrm.CurrentPackageName = prgrm.CurrentPackage.Name
	} else {
		panic("package reference not found")
	}

	sPrgrm.InputsOffset, sPrgrm.InputsSize = serializeSliceOfArguments(prgrm.ProgramInput, s)
	//sPrgrm.OutputsOffset, sPrgrm.OutputsSize = serializeSliceOfArguments(prgrm.ProgramOutput, s)

	sPrgrm.CallStackOffset, sPrgrm.CallStackSize = serializeCalls(prgrm.CallStack[:prgrm.CallCounter], s)

	sPrgrm.CallCounter = int64(prgrm.CallCounter)

	sPrgrm.MemoryOffset = int64(0)
	sPrgrm.MemorySize = int64(len(prgrm.Memory))

	sPrgrm.HeapPointer = int64(prgrm.Heap.Pointer)
	sPrgrm.StackPointer = int64(prgrm.Stack.Pointer)
	sPrgrm.StackSize = int64(prgrm.Stack.Size)
	sPrgrm.DataSegmentSize = int64(prgrm.Data.Size)
	sPrgrm.DataSegmentStartsAt = int64(prgrm.Data.StartsAt)
	sPrgrm.HeapSize = int64(prgrm.Heap.Size)
	sPrgrm.HeapStartsAt = int64(prgrm.Heap.StartsAt)

	sPrgrm.Terminated = serializeBoolean(prgrm.Terminated)
	sPrgrm.VersionOffset, sPrgrm.VersionSize = serializeString(prgrm.Version, s)
}

// serializeCXProgramElements is used serializing CX program's
// elements (packages, structs, functions, etc.).
func serializeCXProgramElements(prgrm *CXProgram, s *SerializedCXProgram) {
	var fnCounter int64
	var strctCounter int64

	// indexing packages and serializing their names
	for _, pkg := range prgrm.Packages {
		indexPackage(pkg, s)
		serializePackageName(pkg, s)
	}
	// we first needed to populate references to all packages
	// now we add the imports' references
	for _, pkg := range prgrm.Packages {
		serializePackageImports(pkg, s)
	}

	// structs
	for _, pkg := range prgrm.Packages {
		for _, strct := range pkg.Structs {
			indexStruct(strct, s)
			serializeStructName(strct, s)
			serializeStructPackage(strct, s)
			serializeStructIntegers(strct, s)
		}
	}
	// we first needed to populate references to all structs
	// now we add fields
	for _, pkg := range prgrm.Packages {
		for _, strct := range pkg.Structs {
			serializeStructArguments(strct, s)
		}
	}

	// globals
	for _, pkg := range prgrm.Packages {
		serializePackageGlobals(pkg, s)
	}

	// functions
	for _, pkg := range prgrm.Packages {
		for _, fn := range pkg.Functions {
			indexFunction(fn, s)
			serializeFunctionName(fn, s)
			serializeFunctionPackage(fn, s)
			serializeFunctionIntegers(fn, s)
			serializeFunctionArguments(fn, s)
		}
	}

	// package elements' offsets and sizes
	for _, pkg := range prgrm.Packages {
		if pkgOff, found := s.PackagesMap[pkg.Name]; found {
			sPkg := &s.Packages[pkgOff]

			if len(pkg.Structs) == 0 {
				sPkg.StructsOffset = int64(-1)
				sPkg.StructsSize = int64(-1)
			} else {
				sPkg.StructsOffset = strctCounter
				lenStrcts := int64(len(pkg.Structs))
				sPkg.StructsSize = lenStrcts
				strctCounter += lenStrcts
			}

			if len(pkg.Functions) == 0 {
				sPkg.FunctionsOffset = int64(-1)
				sPkg.FunctionsSize = int64(-1)
			} else {
				sPkg.FunctionsOffset = fnCounter
				lenFns := int64(len(pkg.Functions))
				sPkg.FunctionsSize = lenFns
				fnCounter += lenFns
			}
		} else {
			panic("package reference not found")
		}
	}

	// package integers
	// we needed the references to all functions and structs first
	for _, pkg := range prgrm.Packages {
		serializePackageIntegers(pkg, s)
	}

	// expressions
	for _, pkg := range prgrm.Packages {
		for _, fn := range pkg.Functions {
			fnName := fn.Package.Name + "." + fn.Name
			if fnOff, found := s.FunctionsMap[fnName]; found {
				sFn := &s.Functions[fnOff]

				if len(fn.Expressions) == 0 {
					sFn.ExpressionsOffset = int64(-1)
					sFn.ExpressionsSize = int64(-1)
					sFn.CurrentExpressionOffset = int64(-1)
				} else {
					exprs := make([]int, len(fn.Expressions))
					for i, expr := range fn.Expressions {
						exprIdx := serializeExpression(prgrm, expr, s)
						if fn.CurrentExpression == expr {
							// sFn.CurrentExpressionOffset = int32(exprIdx)
							sFn.CurrentExpressionOffset = int64(i)
						}
						exprs[i] = exprIdx
					}

					sFn.ExpressionsOffset, sFn.ExpressionsSize = serializeIntegers(exprs, s)
				}
			} else {
				panic("function reference not found")
			}
		}
	}
}

// SerializeCXProgram translates cx program to slice of bytes that we can save.
// These slice of bytes can then be deserialize in the future and
// be translated back to cx program.
func SerializeCXProgram(prgrm *CXProgram, includeDataMemory, useCompression bool) (b []byte) {
	s := SerializedCXProgram{}
	initSerialization(prgrm, &s, includeDataMemory, useCompression)

	// serialize cx program's packages,
	// structs, functions, etc.
	serializeCXProgramElements(prgrm, &s)

	// serialize cx program's program
	serializeProgram(prgrm, &s)

	// serializing everything
	b = encoder.Serialize(s)

	if useCompression {
		// Compress using LZ4
		CompressBytesLZ4(&b)
	}
	return b
}

// SerializeDebugInfo prints the name of the serialized segment and byte size.
func SerializeDebugInfo(prgrm *CXProgram, includeMemory, useCompression bool) SerializedDataSize {
	idxSize := encoder.Size(serializedCXProgramIndex{})
	var s SerializedCXProgram

	bytes := SerializeCXProgram(prgrm, includeMemory, useCompression)
	deserializeRaw(bytes, 0, types.Cast_ui64_to_ptr(idxSize), &s.Index)

	data := &SerializedDataSize{
		Program:     len(bytes[s.Index.ProgramOffset:s.Index.CallsOffset]),
		Calls:       len(bytes[s.Index.CallsOffset:s.Index.PackagesOffset]),
		Packages:    len(bytes[s.Index.PackagesOffset:s.Index.StructsOffset]),
		Structs:     len(bytes[s.Index.StructsOffset:s.Index.FunctionsOffset]),
		Functions:   len(bytes[s.Index.FunctionsOffset:s.Index.ExpressionsOffset]),
		Expressions: len(bytes[s.Index.ExpressionsOffset:s.Index.ArgumentsOffset]),
		Arguments:   len(bytes[s.Index.ArgumentsOffset:s.Index.IntegersOffset]),
		Integers:    len(bytes[s.Index.IntegersOffset:s.Index.StringsOffset]),
		Strings:     len(bytes[s.Index.StringsOffset:s.Index.MemoryOffset]),
		Memory:      len(bytes[s.Index.MemoryOffset:]),
	}

	return *data
}

func deserializeString(off int64, size int64, s *SerializedCXProgram) string {
	if size < 1 {
		return ""
	}

	var name string
	deserializeRaw(s.Strings, types.Cast_i64_to_ptr(off), types.Cast_i64_to_ptr(size), &name)

	return name
}

func deserializePackages(s *SerializedCXProgram, prgrm *CXProgram) {
	var fnCounter int64
	var strctCounter int64

	for _, sPkg := range s.Packages {
		// initializing packages with their names,
		// empty functions, structs, imports and globals
		// and current function and struct
		pkg := CXPackage{}
		pkg.Name = deserializeString(sPkg.NameOffset, sPkg.NameSize, s)
		prgrm.Packages[pkg.Name] = &pkg

		if sPkg.ImportsSize > 0 {
			prgrm.Packages[pkg.Name].Imports = make(map[string]*CXPackage, sPkg.ImportsSize)
		}

		if sPkg.FunctionsSize > 0 {
			prgrm.Packages[pkg.Name].Functions = make(map[string]*CXFunction, sPkg.FunctionsSize)

			for _, sFn := range s.Functions[sPkg.FunctionsOffset : sPkg.FunctionsOffset+sPkg.FunctionsSize] {
				var fn CXFunction
				fn.Name = deserializeString(sFn.NameOffset, sFn.NameSize, s)
				prgrm.Packages[pkg.Name].Functions[fn.Name] = &fn
			}
		}

		if sPkg.StructsSize > 0 {
			prgrm.Packages[pkg.Name].Structs = make(map[string]*CXStruct, sPkg.StructsSize)

			for _, sStrct := range s.Structs[sPkg.StructsOffset : sPkg.StructsOffset+sPkg.StructsSize] {
				var strct CXStruct
				strct.Name = deserializeString(sStrct.NameOffset, sStrct.NameSize, s)
				prgrm.Packages[pkg.Name].Structs[strct.Name] = &strct
			}
		}

		if sPkg.GlobalsSize > 0 {
			prgrm.Packages[pkg.Name].Globals = make([]*CXArgument, sPkg.GlobalsSize)
		}

		// CurrentFunction
		if sPkg.FunctionsSize > 0 {
			prgrm.Packages[pkg.Name].CurrentFunction = prgrm.Packages[pkg.Name].Functions[sPkg.CurrentFunctionName]
		}

		// CurrentStruct
		if sPkg.StructsSize > 0 {
			prgrm.Packages[pkg.Name].CurrentStruct = prgrm.Packages[pkg.Name].Structs[sPkg.CurrentStructName]
		}

		fnCounter += sPkg.FunctionsSize
		strctCounter += sPkg.StructsSize

		// imports
		if sPkg.ImportsSize > 0 {
			// getting indexes of imports
			idxs := deserializeIntegers(sPkg.ImportsOffset, sPkg.ImportsSize, s)

			for _, idx := range idxs {
				impPkg := deserializePackageImport(&s.Packages[idx], s, prgrm)
				prgrm.Packages[pkg.Name].Imports[impPkg.Name] = impPkg
			}
		}

		// globals
		if sPkg.GlobalsSize > 0 {
			prgrm.Packages[pkg.Name].Globals = deserializeArguments(sPkg.GlobalsOffset, sPkg.GlobalsSize, s, prgrm)
		}

		// structs
		if sPkg.StructsSize > 0 {
			for _, sStrct := range s.Structs[sPkg.StructsOffset : sPkg.StructsOffset+sPkg.StructsSize] {
				strctName := deserializeString(sStrct.NameOffset, sStrct.NameSize, s)
				deserializeStruct(&sStrct, prgrm.Packages[pkg.Name].Structs[strctName], s, prgrm)
			}
		}

		// functions
		if sPkg.FunctionsSize > 0 {
			for _, sFn := range s.Functions[sPkg.FunctionsOffset : sPkg.FunctionsOffset+sPkg.FunctionsSize] {
				fnName := deserializeString(sFn.NameOffset, sFn.NameSize, s)
				deserializeFunction(&sFn, prgrm.Packages[pkg.Name].Functions[fnName], s, prgrm)
			}
		}
	}

	// current package
	prgrm.CurrentPackage = prgrm.Packages[s.Program.CurrentPackageName]
}

func deserializeStruct(sStrct *serializedStruct, strct *CXStruct, s *SerializedCXProgram, prgrm *CXProgram) {
	strct.Name = deserializeString(sStrct.NameOffset, sStrct.NameSize, s)
	strct.Fields = deserializeArguments(sStrct.FieldsOffset, sStrct.FieldsSize, s, prgrm)
	strct.Size = types.Cast_i64_to_ptr(sStrct.Size)
	strct.Package = prgrm.Packages[sStrct.PackageName]
}

func deserializeArguments(off int64, size int64, s *SerializedCXProgram, prgrm *CXProgram) []*CXArgument {
	if size < 1 {
		return nil
	}

	// getting indexes of arguments
	idxs := deserializeIntegers(off, size, s)

	// sArgs := s.Arguments[off : off + size]
	args := make([]*CXArgument, size)
	for i, idx := range idxs {
		args[i] = deserializeArgument(&s.Arguments[idx], s, prgrm)
	}
	return args
}

// func getStructType(sArg *serializedArgument, s *SerializedCXProgram, prgrm *CXProgram) *CXStruct {
// 	if sArg.StructTypeOffset < 0 {
// 		return nil
// 	}

// 	//structTypePkg := prgrm.Packages[s.Structs[sArg.StructTypeOffset].PackageOffset]
// 	sStrct := s.Structs[sArg.StructTypeOffset]
// 	structTypeName := deserializeString(sStrct.NameOffset, sStrct.NameSize, s)

// 	for _, strct := range structTypePkg.Structs {
// 		if strct.Name == structTypeName {
// 			return strct
// 		}
// 	}

// 	return nil
// }

func deserializeArgument(sArg *serializedArgument, s *SerializedCXProgram, prgrm *CXProgram) *CXArgument {
	var arg CXArgument
	arg.ArgDetails = &CXArgumentDebug{}
	arg.Name = deserializeString(sArg.NameOffset, sArg.NameSize, s)
	arg.Type = types.Code(sArg.Type)

	//arg.StructType = getStructType(sArg, s, prgrm)

	arg.Size = types.Cast_i64_to_ptr(sArg.Size)
	arg.TotalSize = types.Cast_i64_to_ptr(sArg.TotalSize)
	arg.Offset = types.Cast_i64_to_ptr(sArg.Offset)
	// arg.IndirectionLevels = int(sArg.IndirectionLevels)
	arg.DereferenceLevels = int(sArg.DereferenceLevels)
	arg.PassBy = int(sArg.PassBy)

	arg.DeclarationSpecifiers = deserializeIntegers(sArg.DeclarationSpecifiersOffset, sArg.DeclarationSpecifiersSize, s)

	arg.IsSlice = deserializeBool(sArg.IsSlice)
	// arg.IsPointer = deserializeBool(sArg.IsPointer)
	// arg.IsReference = deserializeBool(sArg.IsReference)
	arg.IsStruct = deserializeBool(sArg.IsStruct)
	arg.IsInnerArg = deserializeBool(sArg.IsInnerArg)
	arg.IsLocalDeclaration = deserializeBool(sArg.IsLocalDeclaration)
	arg.IsShortAssignmentDeclaration = deserializeBool(sArg.IsShortDeclaration)
	arg.PreviouslyDeclared = deserializeBool(sArg.PreviouslyDeclared)
	arg.DoesEscape = deserializeBool(sArg.DoesEscape)

	arg.Lengths = deserializePointers(sArg.LengthsOffset, sArg.LengthsSize, s)
	arg.Indexes = deserializeArguments(sArg.IndexesOffset, sArg.IndexesSize, s, prgrm)
	arg.Fields = deserializeArguments(sArg.FieldsOffset, sArg.FieldsSize, s, prgrm)
	arg.Inputs = deserializeArguments(sArg.InputsOffset, sArg.InputsSize, s, prgrm)
	arg.Outputs = deserializeArguments(sArg.OutputsOffset, sArg.OutputsSize, s, prgrm)

	arg.Package = prgrm.Packages[sArg.PackageName]

	return &arg
}

func deserializeOperator(sExpr *serializedExpression, s *SerializedCXProgram, prgrm *CXProgram) *CXFunction {
	if sExpr.OperatorOffset < 0 {
		return nil
	}

	opPkg := prgrm.Packages[s.Functions[sExpr.OperatorOffset].PackageName]
	sOp := s.Functions[sExpr.OperatorOffset]
	opName := deserializeString(sOp.NameOffset, sOp.NameSize, s)

	for _, fn := range opPkg.Functions {
		if fn.Name == opName {
			return fn
		}
	}

	return nil
}

func deserializePackageImport(sImp *serializedPackage, s *SerializedCXProgram, prgrm *CXProgram) *CXPackage {
	impName := deserializeString(sImp.NameOffset, sImp.NameSize, s)

	for _, pkg := range prgrm.Packages {
		if pkg.Name == impName {
			return pkg
		}
	}

	return nil
}

func deserializeExpressionFunction(sExpr *serializedExpression, s *SerializedCXProgram, prgrm *CXProgram) *CXFunction {
	if sExpr.FunctionOffset < 0 {
		return nil
	}

	fnPkg := prgrm.Packages[s.Functions[sExpr.FunctionOffset].PackageName]
	sFn := s.Functions[sExpr.FunctionOffset]
	fnName := deserializeString(sFn.NameOffset, sFn.NameSize, s)

	for _, fn := range fnPkg.Functions {
		if fn.Name == fnName {
			return fn
		}
	}

	return nil
}

func deserializeExpressions(off int64, size int64, s *SerializedCXProgram, prgrm *CXProgram) []*CXExpression {
	if size < 1 {
		return nil
	}

	// getting indexes of expressions
	idxs := deserializeIntegers(off, size, s)

	// sExprs := s.Expressions[off : off + size]
	exprs := make([]*CXExpression, size)
	for i, idx := range idxs {
		exprs[i] = deserializeExpression(&s.Expressions[idx], s, prgrm)
	}
	return exprs
}

func deserializeExpression(sExpr *serializedExpression, s *SerializedCXProgram, prgrm *CXProgram) *CXExpression {
	var expr CXExpression

	expr.ExpressionType = CXEXPR_TYPE(sExpr.ExpressionType)
	switch sExpr.Type {
	case int64(CX_LINE):
		cxLine := &CXLine{}

		cxLine.FileName = sExpr.FileName
		cxLine.LineNumber = int(sExpr.LineNumber)
		cxLine.LineStr = sExpr.LineStr
		index := prgrm.AddCXLine(cxLine)
		expr.Index = index
		expr.Type = CX_LINE
	case int64(CX_ATOMIC_OPERATOR):
		cxAtomicOp := &CXAtomicOperator{}
		if deserializeBool(sExpr.IsNative) {
			cxAtomicOp.Operator = Natives[int(sExpr.OpCode)]
		} else {
			cxAtomicOp.Operator = deserializeOperator(sExpr, s, prgrm)
		}

		cxAtomicOp.Inputs = deserializeArguments(sExpr.InputsOffset, sExpr.InputsSize, s, prgrm)
		cxAtomicOp.Outputs = deserializeArguments(sExpr.OutputsOffset, sExpr.OutputsSize, s, prgrm)

		cxAtomicOp.Label = deserializeString(sExpr.LabelOffset, sExpr.LabelSize, s)

		cxAtomicOp.ThenLines = int(sExpr.ThenLines)
		cxAtomicOp.ElseLines = int(sExpr.ElseLines)

		cxAtomicOp.Function = deserializeExpressionFunction(sExpr, s, prgrm)
		cxAtomicOp.Package = prgrm.Packages[sExpr.PackageName]

		index := prgrm.AddCXAtomicOp(cxAtomicOp)
		expr.Index = index
		expr.Type = CX_ATOMIC_OPERATOR
	}

	return &expr
}

func deserializeFunction(sFn *serializedFunction, fn *CXFunction, s *SerializedCXProgram, prgrm *CXProgram) {
	fn.Name = deserializeString(sFn.NameOffset, sFn.NameSize, s)
	fn.Inputs = deserializeArguments(sFn.InputsOffset, sFn.InputsSize, s, prgrm)
	fn.Outputs = deserializeArguments(sFn.OutputsOffset, sFn.OutputsSize, s, prgrm)
	fn.ListOfPointers = deserializeArguments(sFn.ListOfPointersOffset, sFn.ListOfPointersSize, s, prgrm)
	fn.Expressions = deserializeExpressions(sFn.ExpressionsOffset, sFn.ExpressionsSize, s, prgrm)
	fn.Size = types.Cast_i64_to_ptr(sFn.Size)
	fn.LineCount = int(sFn.Length)

	if sFn.CurrentExpressionOffset > 0 {
		fn.CurrentExpression = fn.Expressions[sFn.CurrentExpressionOffset]
	}

	fn.Package = prgrm.Packages[sFn.PackageName]
}

func deserializeBool(val int64) bool {
	return val == 1
}

func deserializePointers(off int64, size int64, s *SerializedCXProgram) []types.Pointer {
	if size < 1 {
		return nil
	}
	ints := s.Integers[off : off+size]
	res := make([]types.Pointer, len(ints))
	for i, in := range ints {
		res[i] = types.Cast_i64_to_ptr(in)
	}

	return res
}

func deserializeIntegers(off int64, size int64, s *SerializedCXProgram) []int {
	if size < 1 {
		return nil
	}
	ints := s.Integers[off : off+size]
	res := make([]int, len(ints))
	for i, in := range ints {
		res[i] = int(in)
	}

	return res
}

// initDeserialization initializes the CXProgram fields that represent a CX program. This should be refactored, as the names Deserialize and initDeserialization create some naming conflict.
func initDeserialization(prgrm *CXProgram, s *SerializedCXProgram) {
	prgrm.Memory = s.Memory
	prgrm.Packages = make(map[string]*CXPackage, len(s.Packages))
	prgrm.CallStack = make([]CXCall, constants.CALLSTACK_SIZE)
	prgrm.Heap.StartsAt = types.Cast_i64_to_ptr(s.Program.HeapStartsAt)
	prgrm.Heap.Pointer = types.Cast_i64_to_ptr(s.Program.HeapPointer)
	prgrm.Stack.Size = types.Cast_i64_to_ptr(s.Program.StackSize)
	prgrm.Data.Size = types.Cast_i64_to_ptr(s.Program.DataSegmentSize)
	prgrm.Data.StartsAt = types.Cast_i64_to_ptr(s.Program.DataSegmentStartsAt)
	prgrm.Heap.Size = types.Cast_i64_to_ptr(s.Program.HeapSize)
	prgrm.Version = deserializeString(s.Program.VersionOffset, s.Program.VersionSize, s)

	// This means reinstantiate memory and add DataSegmentMemory
	if len(s.DataSegmentMemory) > 0 && len(s.Memory) == 0 {
		minHeapSize := minHeapSize()
		prgrm.Memory = make([]byte, constants.STACK_SIZE+minHeapSize)
		y := 0
		for i := prgrm.Data.StartsAt; i < prgrm.Data.StartsAt+prgrm.Data.Size; i++ {
			prgrm.Memory[i] = s.DataSegmentMemory[y]
			y++
		}
	}
	deserializePackages(s, prgrm)
}

// Deserialize deserializes a serialized CX program back to its golang struct representation.
func Deserialize(b []byte, useCompression bool) (prgrm *CXProgram) {
	prgrm = &CXProgram{}
	var s SerializedCXProgram

	if useCompression {
		// Uncompress using LZ4
		UncompressBytesLZ4(&b)
	}

	deserializeRaw(b, 0, types.Cast_int_to_ptr(len(b)), &s)
	initDeserialization(prgrm, &s)

	return prgrm
}

// CopyProgramState copies the program state from `prgrm1` to `prgrm2`.
func CopyProgramState(sPrgrm1, sPrgrm2 *[]byte) {
	idxSize := types.Cast_ui64_to_ptr(encoder.Size(serializedCXProgramIndex{}))

	var index1 serializedCXProgramIndex
	var index2 serializedCXProgramIndex

	deserializeRaw((*sPrgrm1), 0, idxSize, &index1)
	deserializeRaw((*sPrgrm2), 0, idxSize, &index2)

	var prgrm1Info serializedProgram
	deserializeRaw((*sPrgrm1),
		types.Cast_i64_to_ptr(index1.ProgramOffset),
		types.Cast_i64_to_ptr(index1.CallsOffset-index1.ProgramOffset),
		&prgrm1Info)

	var prgrm2Info serializedProgram
	deserializeRaw((*sPrgrm2),
		types.Cast_i64_to_ptr(index2.ProgramOffset),
		types.Cast_i64_to_ptr(index2.CallsOffset-index2.ProgramOffset),
		&prgrm2Info)

	// the stack segment should be 0 for prgrm1, but just in case
	var prgrmState []byte
	prgrmState = append(prgrmState, make([]byte, prgrm2Info.StackSize)...)
	// We are only interested on extracting the data segment
	prgrmState = append(prgrmState, (*sPrgrm1)[index1.StringsOffset+prgrm1Info.StackSize:index1.StringsOffset+prgrm1Info.StackSize+(prgrm2Info.HeapStartsAt-prgrm2Info.StackSize)]...)

	for i, byt := range prgrmState {
		(*sPrgrm2)[i+int(index2.MemoryOffset)] = byt
	}
}

// GetSerializedStackSize returns the stack size of a serialized CX program starts.
func GetSerializedStackSize(sPrgrm []byte) int {
	idxSize := types.Cast_ui64_to_ptr(encoder.Size(serializedCXProgramIndex{}))
	var index serializedCXProgramIndex
	deserializeRaw(sPrgrm, 0, idxSize, &index)

	var prgrmInfo serializedProgram
	deserializeRaw(sPrgrm,
		types.Cast_i64_to_ptr(index.ProgramOffset),
		types.Cast_i64_to_ptr(index.CallsOffset-index.ProgramOffset),
		&prgrmInfo)

	return int(prgrmInfo.StackSize)
}

// GetSerializedDataSize returns the size of the data segment of a serialized CX program.
func GetSerializedDataSize(sPrgrm []byte) int {
	idxSize := types.Cast_ui64_to_ptr(encoder.Size(serializedCXProgramIndex{}))
	var index serializedCXProgramIndex
	deserializeRaw(sPrgrm, 0, idxSize, &index)

	var prgrmInfo serializedProgram
	deserializeRaw(sPrgrm,
		types.Cast_i64_to_ptr(index.ProgramOffset),
		types.Cast_i64_to_ptr(index.CallsOffset-index.ProgramOffset),
		&prgrmInfo)

	return int(prgrmInfo.HeapStartsAt - prgrmInfo.StackSize)
}
