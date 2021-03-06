package utils

import (
	"github.com/atmchain/atmapp/log"
	colorable "github.com/mattn/go-colorable"
	"gopkg.in/urfave/cli.v1"
	"io"
	"os"
	"syscall"
	"unsafe"
)

var (
	verbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
)

// Flags holds all command-line flags required for debugging.
var Flags = []cli.Flag{
	verbosityFlag,
}

type Termios syscall.Termios

const ioctlReadTermios = syscall.TIOCGETA

var glogger *log.GlogHandler

// IsTty returns true if the given file descriptor is a terminal.
func IsTty(fd uintptr) bool {
	var termios Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func init() {
	usecolor := IsTty(os.Stderr.Fd()) && os.Getenv("TERM") != "dumb"
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	glogger = log.NewGlogHandler(log.StreamHandler(output, log.TerminalFormat(usecolor)))
}

// Setup initializes profiling and logging based on the CLI flags.
// It should be called as early as possible in the program.
func Setup(ctx *cli.Context) error {
	// logging
	//log.PrintOrigins(ctx.GlobalBool(debugFlag.Name))
	glogger.Verbosity(log.Lvl(ctx.GlobalInt(verbosityFlag.Name)))
	//glogger.Vmodule(ctx.GlobalString(vmoduleFlag.Name))
	//glogger.BacktraceAt(ctx.GlobalString(backtraceAtFlag.Name))
	log.Root().SetHandler(glogger)
	return nil
}
