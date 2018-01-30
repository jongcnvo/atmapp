package main

import (
	"../node"
	"../p2p"
	"../p2p/nat"
	"fmt"
	"gopkg.in/urfave/cli.v2"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

const (
	clientIdentifier = "atmapp" // Client identifier to advertise over the network
)

func main() {
	app := &cli.App{
		Name:      "greet",
		Usage:     "say a greeting",
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
	//node := makeFullNode(ctx)
	//startNode(ctx, node)
	//node.Wait()
	return nil
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "ATMChain")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "ATMChain")
		} else {
			return filepath.Join(home, ".atmchain")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

const (
	DefaultHTTPHost = "localhost" // Default host interface for the HTTP RPC server
	DefaultHTTPPort = 8545        // Default TCP port for the HTTP RPC server
	DefaultWSHost   = "localhost" // Default host interface for the websocket RPC server
	DefaultWSPort   = 8546        // Default TCP port for the websocket RPC server
)

// DefaultConfig contains reasonable default settings.
var DefaultConfig = node.Config{
	DataDir:     DefaultDataDir(),
	HTTPPort:    DefaultHTTPPort,
	HTTPModules: []string{"net", "web3"},
	WSPort:      DefaultWSPort,
	WSModules:   []string{"net", "web3"},
	P2P: p2p.Config{
		ListenAddr:      ":30303",
		DiscoveryV5Addr: ":30304",
		MaxPeers:        25,
		NAT:             nat.Any(),
	},
}

func defaultNodeConfig() node.Config {
	cfg := DefaultConfig
	cfg.Name = clientIdentifier
	//cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "eth", "shh")
	cfg.WSModules = append(cfg.WSModules, "eth", "shh")
	cfg.IPCPath = "geth.ipc"
	return cfg
}
