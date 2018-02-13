package utils

import (
	"../atm"
	"../node"
	"../params"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

// NewApp creates an app with sane defaults.
func NewApp(gitCommit, usage string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	//app.Authors = nil
	app.Email = ""
	app.Version = params.Version
	if len(gitCommit) >= 8 {
		app.Version += "-" + gitCommit[:8]
	}
	app.Usage = usage
	return app
}

// RegisterATMService adds an Ethereum client to the stack.
func RegisterATMService(stack *node.Node, cfg *atm.Config) {
	var err error
	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		atm.New(ctx, cfg)
		return nil, nil
	})
	if err != nil {
		//Fatalf("Failed to register the ATM service: %v", err)
	}
}
