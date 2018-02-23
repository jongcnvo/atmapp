package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/atmchain/atmapp/accounts"
	"github.com/atmchain/atmapp/accounts/keystore"
	"github.com/atmchain/atmapp/atm"
	"github.com/atmchain/atmapp/log"
	"github.com/atmchain/atmapp/node"
	"github.com/atmchain/atmapp/utils"
	"gopkg.in/urfave/cli.v1"
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

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go stack.Stop()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		//debug.Exit() // ensure trace and CPU profile data is flushed.
		//debug.LoudPanic("boom")
	}()

	// Unlock any account specifically requested
	//ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	//passwords := utils.MakePasswordList(ctx)

	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)
}

// tries unlocking the specified account a few times.
func unlockAccount(ctx *cli.Context, ks *keystore.KeyStore, address string, i int, passwords []string) (accounts.Account, string) {
	account, err := utils.MakeAddress(ks, address)
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := getPassPhrase(prompt, false, i, passwords)
		err = ks.Unlock(account, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address)
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address)
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	utils.Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

// getPassPhrase retrieves the password associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := utils.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := utils.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError, auth string) accounts.Account {
	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.URL)
	}
	fmt.Println("Testing your passphrase against all of them...")
	var match *accounts.Account
	for _, a := range err.Matches {
		if err := ks.Unlock(a, auth); err == nil {
			match = &a
			break
		}
	}
	if match == nil {
		utils.Fatalf("None of the listed files could be unlocked.")
	}
	fmt.Printf("Your passphrase unlocked %s\n", match.URL)
	fmt.Println("In order to avoid this warning, you need to remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.URL)
		}
	}
	return *match
}
