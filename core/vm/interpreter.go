package vm

// Config are the configuration options for the Interpreter
type Config struct {
	// Debug enabled debugging Interpreter options
	Debug bool
	// EnableJit enabled the JIT VM
	EnableJit bool
	// ForceJit forces the JIT VM
	ForceJit bool
	// Tracer is the op code logger
	//Tracer Tracer
	// NoRecursion disabled Interpreter call, callcode,
	// delegate call and create.
	NoRecursion bool
	// Disable gas metering
	DisableGasMetering bool
	// Enable recording of SHA3/keccak preimages
	EnablePreimageRecording bool
	// JumpTable contains the EVM instruction table. This
	// may be left uninitialised and will be set to the default
	// table.
	//JumpTable [256]operation
}
