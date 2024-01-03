	// The instrumentation code relies on there being 8 bytes of space available on the stack which we can steal to
	// cache the pointer to the telemetry data.
	// In EBPF there cannot be any dynamic allocations on the stack. We utilize this guarantee to get the stack usage
	// of each function; to determine whether there are 8 bytes free for us to use. If not available, we cannot have telemetry
	// and so patch nops at our trampolines.
	//
	// The stack size of each function is obtained from the `.stack_sizes` section in the ELF file. This section is emitted
	// when the `-stack-size-section` argument is passed to llc. Details of this section are documented here:
	// - https://releases.llvm.org/9.0.0/docs/CodeGenerator.html#emitting-function-stack-size-information
	//
	// Confusingly the docs mention:
	// "The stack size values only include the space allocated in the function prologue. Functions with dynamic stack allocations are not included."
	// This seems to suggest that this option relies on an explicit prologue to get the stack space. There is no prologue present in EBPF bytecode
	// since the frame pointer is read-only.
	// According to the LLVM source code however, there is no explicit requirement of the prologue. This is deduced in two way:
	// 1. Looking at the LLVM source code which tracks BPF stack size and ensures it is less than 512 bytes, and the source code
	//    for calculating stack sizes. As can be seen in the following links, both functions track fixed size allocations on the
	//    stack to determine the size. The ebpf code checks each allocation to make sure it does not cross the 512 byte limit,
	//    and the function calculating the `StackSize` loops over all stack allocations to calculate the stack size.
	//    - BPF stack size check: https://github.com/llvm/llvm-project/blob/llvmorg-12.0.1/llvm/lib/Target/BPF/BPFRegisterInfo.cpp#L96-L102
	//    - `StackSize` calculation: https://github.com/llvm/llvm-project/blob/llvmorg-12.0.1/llvm/lib/CodeGen/PrologEpilogInserter.cpp#L1104
	//    - Emit stack_sizes section: https://github.com/llvm/llvm-project/blob/llvmorg-12.0.1/llvm/lib/CodeGen/AsmPrinter/AsmPrinter.cpp#L1137
	//
	// 2. By parsing the `.stack_sizes` sections in the object files and validating that it reports the correct stack usage.


			// If the compiler argument '-pg' is passed, then the compiler will instrument a `call -1` instruction
			// somewhere (see [2] below) in the beginning of the bytecode. We will patch this instruction as a trampoline to
			// -------------------------
			// jump to our instrumentation code.
			//
			// Checks:
			// [1] We cannot use the helper `IsBuiltinCall()` for the entry trampoline since the compiler does not correctly generate
			//     the instrumentation instruction. The loader expects the source and destination register to be set to R0
			//     along with the correct opcode, for it to be recognized as a built-in call. However, the `call -1` does
			//     not satisfy this requirement. Therefore, we use a looser check relying on the constant to be `-1` to correctly
			//     identify the patch point.
			//
			// [2] The instrumentation instruction may not be the first instruction in the bytecode.
			//     Since R1 is allowed to be clobbered, the compiler adds the instrumentation instruction after `RX = R1`, when
			//     R1, i.e. the context pointer, is used in the code. `RX` will be a callee saved register.
			//     If R1, i.e. the context pointer, is not used then the instrumented instruction will be the first instruction.

