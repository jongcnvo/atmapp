package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"os"
	"runtime"

	"github.com/atmchain/atmapp/atm"
	"github.com/atmchain/atmapp/log"
	"github.com/atmchain/atmapp/node"
	"github.com/atmchain/atmapp/utils"
)

const (
	clientIdentifier = "atmapp" // Client identifier to advertise over the network
)

var (
	gitCommit = ""
	app       = utils.NewApp(gitCommit, "the ATMChain command line interface")
)

func init() {
	// Initialize the CLI app and start Geth
	app.Action = atmapp
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2017-2018 The ATMChain Foundation"
	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := utils.Setup(ctx); err != nil {
			return err
		}
		// Start system runtime metrics collection
		//go metrics.CollectProcessMetrics(3 * time.Second)
		return nil
	}
}

func main() {
	fmt.Println("")
	fmt.Println("        ___   ______ __  ___ ______ __            _ ")
	fmt.Println("       /   | /_  __//  |/  // ____// /_   ____ _ (_)____ ")
	fmt.Println("      / /| |  / /  / /|_/ // /    / __ \\ / __ `// // __ \\")
	fmt.Println("     / ___ | / /  / /  / // /___ / / / // /_/ // / // / /")
	fmt.Println("    /_/  |_|/_/  /_/  /_/ \\____//_/ /_/ \\__,_//_//_/ /_/ ")
	fmt.Println("")

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// atmapp is the main entry point into the system
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func atmapp(ctx *cli.Context) error {
	node := makeFullNode(ctx)
	startNode(ctx, node)
	node.Wait()
	return nil
}

func makeFullNode(ctx *cli.Context) *node.Node {
	// Load defaults.
	cfg := ATMConfig{
		ATM:  atm.DefaultConfig,
		Node: DefaultNodeConfig(),
	}

	// Create node instance
	stack, err := node.New(&cfg.Node)
	if err != nil {
		return nil
	}

	// Set config from command line
	utils.SetATMConfig(ctx, stack, &cfg.ATM)

	// Register ATMChain service
	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		fullNode, err := atm.New(ctx, &cfg.ATM)
		return fullNode, err
	})

	return stack
}

func startNode(ctx *cli.Context, stack *node.Node) {
	if err := stack.Start(); err != nil {
		log.Info("Error starting protocol stack: %v", err)
	}
}
