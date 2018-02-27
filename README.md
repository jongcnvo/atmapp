# ATMChain
Blockchain protocol for smart media

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

  console  Start an interactive Javascript environment

GLOBAL OPTIONS:
  --unlock value                    Comma separated list of accounts to unlock
  --mine                            Enable mining
  --port value                      Network listening port (default: 30303)