package main

import (
	"../core"
	"../log"
	"../node"
	"fmt"
	"gopkg.in/urfave/cli.v2"
	"os"
)

const (
	clientIdentifier = "atmapp" // Client identifier to advertise over the network
)

func main() {
	app := &cli.App{
		Name:      "ATMChain Core Layer Application",
		Usage:     "Run as a ATM full node",
		Action:    atmapp,
		Copyright: "Copyright 2017-2018 The ATMChain Foundation",
	}

	app.Run(os.Args)

	fmt.Println("    ___   ______ __  ___ ______ __            _ ")
	fmt.Println("   /   | /_  __//  |/  // ____// /_   ____ _ (_)____ ")
	fmt.Println("  / /| |  / /  / /|_/ // /    / __ \\ / __ `// // __ \\")
	fmt.Println(" / ___ | / /  / /  / // /___ / / / // /_/ // / // / /")
	fmt.Println("/_/  |_|/_/  /_/  /_/ \\____//_/ /_/ \\__,_//_//_/ /_/ ")
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
		ATM:  core.DefaultConfig,
		Node: DefaultNodeConfig(),
	}

	stack, err := node.New(&cfg.Node)
	if err != nil {
		return nil
	}
	return stack
}

func startNode(ctx *cli.Context, stack *node.Node) {
	if err := stack.Start(); err != nil {
		log.Info("Error starting protocol stack: %v", err)
	}
}
