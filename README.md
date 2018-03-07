# ATMChain
Blockchain protocol for smart media

## Support OS

Mac/Linux

## Installation

1. Clone ATMChain source code from github 

```
$ go get github.com/atmchain/atmapp
```

2. Enter the root folder

```
$ cd ~go/src/github.com/atmchain/atmapp
```

3. Sync external dependencies

```
$ govendor sync
```

4. Make binary

```
$ make
```

## Run

Run ATMChain command line interface:

```
$ build/bin/atmapp
```

SUB COMMANDS:

  account  Manage accounts

  console  Start an interactive Javascript environment

  removedb Remove blockchain and state databases

GLOBAL OPTIONS:

  --datadir value               Data directory for the databases and keystore
  
  --identity value              Custom node name

  --unlock value                Comma separated list of accounts to unlock

  --mine                        Enable mining
  
  --port value                  Network listening port (default: 30303)

  --nodiscover                  Disables the peer discovery mechanism (manual peer addition)

  --verbosity value             Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail (default: 3)

  --rpc                         Enable the HTTP-RPC server
  
  --rpcaddr value               HTTP-RPC server listening interface (default: "localhost")
  
  --rpcport value               HTTP-RPC server listening port (default: 8545)
  
  --rpcapi value                API's offered over the HTTP-RPC interface