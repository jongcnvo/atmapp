package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/atmchain/atmapp/log"
	"github.com/atmchain/atmapp/rpc"
	"github.com/atmchain/atmapp/utils/jsre"
	"github.com/atmchain/atmapp/utils/web3ext"
	"github.com/mattn/go-colorable"
	"github.com/peterh/liner"
	"github.com/robertkrimen/otto"
)

var (
	passwordRegexp = regexp.MustCompile(`personal.[nus]`)
	onlyWhitespace = regexp.MustCompile(`^\s*$`)
	exit           = regexp.MustCompile(`^\s*exit\s*;*\s*$`)
)

// HistoryFile is the file within the data directory to store input scrollback.
const HistoryFile = "history"

// DefaultPrompt is the default prompt line prefix to use for user input querying.
const DefaultPrompt = "> "

// Config is the collection of configurations to fine tune the behavior of the
// JavaScript console.
type Config struct {
	DataDir  string       // Data directory to store the console history at
	DocRoot  string       // Filesystem path from where to load JavaScript files from
	Client   *rpc.Client  // RPC client to execute Ethereum requests through
	Prompt   string       // Input prompt prefix string (defaults to DefaultPrompt)
	Prompter UserPrompter // Input prompter to allow interactive user feedback (defaults to TerminalPrompter)
	Printer  io.Writer    // Output writer to serialize any display strings to (defaults to os.Stdout)
	Preload  []string     // Absolute paths to JavaScript files to preload
}

// Console is a JavaScript interpreted runtime environment. It is a fully fleged
// JavaScript console attached to a running node via an external or in-process RPC
// client.
type Console struct {
	client   *rpc.Client  // RPC client to execute Ethereum requests through
	jsre     *jsre.JSRE   // JavaScript runtime environment running the interpreter
	prompt   string       // Input prompt prefix string
	prompter UserPrompter // Input prompter to allow interactive user feedback
	histPath string       // Absolute path to the console scrollback history
	history  []string     // Scroll history maintained by the console
	printer  io.Writer    // Output writer to serialize any display strings to
}

// bridge is a collection of JavaScript utility methods to bride the .js runtime
// environment and the Go RPC connection backing the remote method calls.
type bridge struct {
	client   *rpc.Client  // RPC client to execute Ethereum requests through
	prompter UserPrompter // Input prompter to allow interactive user feedback
	printer  io.Writer    // Output writer to serialize any display strings to
}

// newBridge creates a new JavaScript wrapper around an RPC client.
func newBridge(client *rpc.Client, prompter UserPrompter, printer io.Writer) *bridge {
	return &bridge{
		client:   client,
		prompter: prompter,
		printer:  printer,
	}
}

func New(config Config) (*Console, error) {
	// Handle unset config values gracefully
	if config.Prompter == nil {
		config.Prompter = Stdin
	}
	if config.Prompt == "" {
		config.Prompt = DefaultPrompt
	}
	if config.Printer == nil {
		config.Printer = colorable.NewColorableStdout()
	}
	// Initialize the console and return
	console := &Console{
		client:   config.Client,
		jsre:     jsre.New(config.DocRoot, config.Printer),
		prompt:   config.Prompt,
		prompter: config.Prompter,
		printer:  config.Printer,
		histPath: filepath.Join(config.DataDir, HistoryFile),
	}
	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return nil, err
	}
	if err := console.init(config.Preload); err != nil {
		return nil, err
	}
	return console, nil
}

type jsonrpcCall struct {
	Id     int64
	Method string
	Params []interface{}
}

// Send implements the web3 provider "send" method.
func (b *bridge) Send(call otto.FunctionCall) (response otto.Value) {
	// Remarshal the request into a Go value.
	JSON, _ := call.Otto.Object("JSON")
	reqVal, err := JSON.Call("stringify", call.Argument(0))
	if err != nil {
		throwJSException(err.Error())
	}
	var (
		rawReq = reqVal.String()
		dec    = json.NewDecoder(strings.NewReader(rawReq))
		reqs   []jsonrpcCall
		batch  bool
	)
	dec.UseNumber() // avoid float64s
	if rawReq[0] == '[' {
		batch = true
		dec.Decode(&reqs)
	} else {
		batch = false
		reqs = make([]jsonrpcCall, 1)
		dec.Decode(&reqs[0])
	}

	// Execute the requests.
	resps, _ := call.Otto.Object("new Array()")
	for _, req := range reqs {
		resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
		resp.Set("id", req.Id)
		var result json.RawMessage
		err = b.client.Call(&result, req.Method, req.Params...)
		switch err := err.(type) {
		case nil:
			if result == nil {
				// Special case null because it is decoded as an empty
				// raw message for some reason.
				resp.Set("result", otto.NullValue())
			} else {
				resultVal, err := JSON.Call("parse", string(result))
				if err != nil {
					setError(resp, -32603, err.Error())
				} else {
					resp.Set("result", resultVal)
				}
			}
		case rpc.Error:
			setError(resp, err.ErrorCode(), err.Error())
		default:
			setError(resp, -32603, err.Error())
		}
		resps.Call("push", resp)
	}

	// Return the responses either to the callback (if supplied)
	// or directly as the return value.
	if batch {
		response = resps.Value()
	} else {
		response, _ = resps.Get("0")
	}
	if fn := call.Argument(1); fn.Class() == "Function" {
		fn.Call(otto.NullValue(), otto.NullValue(), response)
		return otto.UndefinedValue()
	}
	return response
}

// NewAccount is a wrapper around the personal.newAccount RPC method that uses a
// non-echoing password prompt to acquire the passphrase and executes the original
// RPC method (saved in jeth.newAccount) with it to actually execute the RPC call.
func (b *bridge) NewAccount(call otto.FunctionCall) (response otto.Value) {
	var (
		password string
		confirm  string
		err      error
	)
	switch {
	// No password was specified, prompt the user for it
	case len(call.ArgumentList) == 0:
		if password, err = b.prompter.PromptPassword("Passphrase: "); err != nil {
			throwJSException(err.Error())
		}
		if confirm, err = b.prompter.PromptPassword("Repeat passphrase: "); err != nil {
			throwJSException(err.Error())
		}
		if password != confirm {
			throwJSException("passphrases don't match!")
		}

	// A single string password was specified, use that
	case len(call.ArgumentList) == 1 && call.Argument(0).IsString():
		password, _ = call.Argument(0).ToString()

	// Otherwise fail with some error
	default:
		throwJSException("expected 0 or 1 string argument")
	}
	// Password acquired, execute the call and return
	ret, err := call.Otto.Call("jeth.newAccount", nil, password)
	if err != nil {
		throwJSException(err.Error())
	}
	return ret
}

// OpenWallet is a wrapper around personal.openWallet which can interpret and
// react to certain error messages, such as the Trezor PIN matrix request.
func (b *bridge) OpenWallet(call otto.FunctionCall) (response otto.Value) {
	// Make sure we have an wallet specified to open
	if !call.Argument(0).IsString() {
		throwJSException("first argument must be the wallet URL to open")
	}
	wallet := call.Argument(0)

	var passwd otto.Value
	if call.Argument(1).IsUndefined() || call.Argument(1).IsNull() {
		passwd, _ = otto.ToValue("")
	} else {
		passwd = call.Argument(1)
	}
	// Open the wallet and return if successful in itself
	val, err := call.Otto.Call("jeth.openWallet", nil, wallet, passwd)
	if err == nil {
		return val
	}

	// Trezor PIN matrix input requested, display the matrix to the user and fetch the data
	fmt.Fprintf(b.printer, "Look at the device for number positions\n\n")
	fmt.Fprintf(b.printer, "7 | 8 | 9\n")
	fmt.Fprintf(b.printer, "--+---+--\n")
	fmt.Fprintf(b.printer, "4 | 5 | 6\n")
	fmt.Fprintf(b.printer, "--+---+--\n")
	fmt.Fprintf(b.printer, "1 | 2 | 3\n\n")

	if input, err := b.prompter.PromptPassword("Please enter current PIN: "); err != nil {
		throwJSException(err.Error())
	} else {
		passwd, _ = otto.ToValue(input)
	}
	if val, err = call.Otto.Call("jeth.openWallet", nil, wallet, passwd); err != nil {
		throwJSException(err.Error())
	}
	return val
}

// UnlockAccount is a wrapper around the personal.unlockAccount RPC method that
// uses a non-echoing password prompt to acquire the passphrase and executes the
// original RPC method (saved in jeth.unlockAccount) with it to actually execute
// the RPC call.
func (b *bridge) UnlockAccount(call otto.FunctionCall) (response otto.Value) {
	// Make sure we have an account specified to unlock
	if !call.Argument(0).IsString() {
		throwJSException("first argument must be the account to unlock")
	}
	account := call.Argument(0)

	// If password is not given or is the null value, prompt the user for it
	var passwd otto.Value

	if call.Argument(1).IsUndefined() || call.Argument(1).IsNull() {
		fmt.Fprintf(b.printer, "Unlock account %s\n", account)
		if input, err := b.prompter.PromptPassword("Passphrase: "); err != nil {
			throwJSException(err.Error())
		} else {
			passwd, _ = otto.ToValue(input)
		}
	} else {
		if !call.Argument(1).IsString() {
			throwJSException("password must be a string")
		}
		passwd = call.Argument(1)
	}
	// Third argument is the duration how long the account must be unlocked.
	duration := otto.NullValue()
	if call.Argument(2).IsDefined() && !call.Argument(2).IsNull() {
		if !call.Argument(2).IsNumber() {
			throwJSException("unlock duration must be a number")
		}
		duration = call.Argument(2)
	}
	// Send the request to the backend and return
	val, err := call.Otto.Call("jeth.unlockAccount", nil, account, passwd, duration)
	if err != nil {
		throwJSException(err.Error())
	}
	return val
}

// Sign is a wrapper around the personal.sign RPC method that uses a non-echoing password
// prompt to acquire the passphrase and executes the original RPC method (saved in
// jeth.sign) with it to actually execute the RPC call.
func (b *bridge) Sign(call otto.FunctionCall) (response otto.Value) {
	var (
		message = call.Argument(0)
		account = call.Argument(1)
		passwd  = call.Argument(2)
	)

	if !message.IsString() {
		throwJSException("first argument must be the message to sign")
	}
	if !account.IsString() {
		throwJSException("second argument must be the account to sign with")
	}

	// if the password is not given or null ask the user and ensure password is a string
	if passwd.IsUndefined() || passwd.IsNull() {
		fmt.Fprintf(b.printer, "Give password for account %s\n", account)
		if input, err := b.prompter.PromptPassword("Passphrase: "); err != nil {
			throwJSException(err.Error())
		} else {
			passwd, _ = otto.ToValue(input)
		}
	}
	if !passwd.IsString() {
		throwJSException("third argument must be the password to unlock the account")
	}

	// Send the request to the backend and return
	val, err := call.Otto.Call("jeth.sign", nil, message, account, passwd)
	if err != nil {
		throwJSException(err.Error())
	}
	return val
}

// Sleep will block the console for the specified number of seconds.
func (b *bridge) Sleep(call otto.FunctionCall) (response otto.Value) {
	if call.Argument(0).IsNumber() {
		sleep, _ := call.Argument(0).ToInteger()
		time.Sleep(time.Duration(sleep) * time.Second)
		return otto.TrueValue()
	}
	return throwJSException("usage: sleep(<number of seconds>)")
}

// SleepBlocks will block the console for a specified number of new blocks optionally
// until the given timeout is reached.
func (b *bridge) SleepBlocks(call otto.FunctionCall) (response otto.Value) {
	var (
		blocks = int64(0)
		sleep  = int64(9999999999999999) // indefinitely
	)
	// Parse the input parameters for the sleep
	nArgs := len(call.ArgumentList)
	if nArgs == 0 {
		throwJSException("usage: sleepBlocks(<n blocks>[, max sleep in seconds])")
	}
	if nArgs >= 1 {
		if call.Argument(0).IsNumber() {
			blocks, _ = call.Argument(0).ToInteger()
		} else {
			throwJSException("expected number as first argument")
		}
	}
	if nArgs >= 2 {
		if call.Argument(1).IsNumber() {
			sleep, _ = call.Argument(1).ToInteger()
		} else {
			throwJSException("expected number as second argument")
		}
	}
	// go through the console, this will allow web3 to call the appropriate
	// callbacks if a delayed response or notification is received.
	blockNumber := func() int64 {
		result, err := call.Otto.Run("eth.blockNumber")
		if err != nil {
			throwJSException(err.Error())
		}
		block, err := result.ToInteger()
		if err != nil {
			throwJSException(err.Error())
		}
		return block
	}
	// Poll the current block number until either it ot a timeout is reached
	targetBlockNr := blockNumber() + blocks
	deadline := time.Now().Add(time.Duration(sleep) * time.Second)

	for time.Now().Before(deadline) {
		if blockNumber() >= targetBlockNr {
			return otto.TrueValue()
		}
		time.Sleep(time.Second)
	}
	return otto.FalseValue()
}

func setError(resp *otto.Object, code int, msg string) {
	resp.Set("error", map[string]interface{}{"code": code, "message": msg})
}

// throwJSException panics on an otto.Value. The Otto VM will recover from the
// Go panic and throw msg as a JavaScript error.
func throwJSException(msg interface{}) otto.Value {
	val, err := otto.ToValue(msg)
	if err != nil {
		log.Error("Failed to serialize JavaScript exception", "exception", msg, "err", err)
	}
	panic(val)
}

// init retrieves the available APIs from the remote RPC provider and initializes
// the console's JavaScript namespaces based on the exposed modules.
func (c *Console) init(preload []string) error {
	// Initialize the JavaScript <-> Go RPC bridge
	bridge := newBridge(c.client, c.prompter, c.printer)
	c.jsre.Set("jeth", struct{}{})

	jethObj, _ := c.jsre.Get("jeth")
	jethObj.Object().Set("send", bridge.Send)
	jethObj.Object().Set("sendAsync", bridge.Send)

	consoleObj, _ := c.jsre.Get("console")
	consoleObj.Object().Set("log", c.consoleOutput)
	consoleObj.Object().Set("error", c.consoleOutput)

	// Load all the internal utility JavaScript libraries
	if err := c.jsre.Compile("bignumber.js", jsre.BigNumber_JS); err != nil {
		return fmt.Errorf("bignumber.js: %v", err)
	}
	if err := c.jsre.Compile("web3.js", jsre.Web3_JS); err != nil {
		return fmt.Errorf("web3.js: %v", err)
	}
	if _, err := c.jsre.Run("var Web3 = require('web3');"); err != nil {
		return fmt.Errorf("web3 require: %v", err)
	}
	if _, err := c.jsre.Run("var web3 = new Web3(jeth);"); err != nil {
		return fmt.Errorf("web3 provider: %v", err)
	}
	// Load the supported APIs into the JavaScript runtime environment
	apis, err := c.client.SupportedModules()
	if err != nil {
		return fmt.Errorf("api modules: %v", err)
	}
	flatten := "var eth = web3.eth; var personal = web3.personal; "
	for api := range apis {
		if api == "web3" {
			continue // manually mapped or ignore
		}
		if file, ok := web3ext.Modules[api]; ok {
			// Load our extension for the module.
			if err = c.jsre.Compile(fmt.Sprintf("%s.js", api), file); err != nil {
				return fmt.Errorf("%s.js: %v", api, err)
			}
			flatten += fmt.Sprintf("var %s = web3.%s; ", api, api)
		} else if obj, err := c.jsre.Run("web3." + api); err == nil && obj.IsObject() {
			// Enable web3.js built-in extension if available.
			flatten += fmt.Sprintf("var %s = web3.%s; ", api, api)
		}
	}
	if _, err = c.jsre.Run(flatten); err != nil {
		return fmt.Errorf("namespace flattening: %v", err)
	}
	// Initialize the global name register (disabled for now)
	//c.jsre.Run(`var GlobalRegistrar = eth.contract(` + registrar.GlobalRegistrarAbi + `);   registrar = GlobalRegistrar.at("` + registrar.GlobalRegistrarAddr + `");`)

	// If the console is in interactive mode, instrument password related methods to query the user
	if c.prompter != nil {
		// Retrieve the account management object to instrument
		personal, err := c.jsre.Get("personal")
		if err != nil {
			return err
		}
		// Override the openWallet, unlockAccount, newAccount and sign methods since
		// these require user interaction. Assign these method in the Console the
		// original web3 callbacks. These will be called by the jeth.* methods after
		// they got the password from the user and send the original web3 request to
		// the backend.
		if obj := personal.Object(); obj != nil { // make sure the personal api is enabled over the interface
			if _, err = c.jsre.Run(`jeth.openWallet = personal.openWallet;`); err != nil {
				return fmt.Errorf("personal.openWallet: %v", err)
			}
			if _, err = c.jsre.Run(`jeth.unlockAccount = personal.unlockAccount;`); err != nil {
				return fmt.Errorf("personal.unlockAccount: %v", err)
			}
			if _, err = c.jsre.Run(`jeth.newAccount = personal.newAccount;`); err != nil {
				return fmt.Errorf("personal.newAccount: %v", err)
			}
			if _, err = c.jsre.Run(`jeth.sign = personal.sign;`); err != nil {
				return fmt.Errorf("personal.sign: %v", err)
			}
			obj.Set("openWallet", bridge.OpenWallet)
			obj.Set("unlockAccount", bridge.UnlockAccount)
			obj.Set("newAccount", bridge.NewAccount)
			obj.Set("sign", bridge.Sign)
		}
	}
	// The admin.sleep and admin.sleepBlocks are offered by the console and not by the RPC layer.
	admin, err := c.jsre.Get("admin")
	if err != nil {
		return err
	}
	if obj := admin.Object(); obj != nil { // make sure the admin api is enabled over the interface
		obj.Set("sleepBlocks", bridge.SleepBlocks)
		obj.Set("sleep", bridge.Sleep)
		obj.Set("clearHistory", c.clearHistory)
	}
	// Preload any JavaScript files before starting the console
	for _, path := range preload {
		if err := c.jsre.Exec(path); err != nil {
			failure := err.Error()
			if ottoErr, ok := err.(*otto.Error); ok {
				failure = ottoErr.String()
			}
			return fmt.Errorf("%s: %v", path, failure)
		}
	}
	// Configure the console's input prompter for scrollback and tab completion
	if c.prompter != nil {
		if content, err := ioutil.ReadFile(c.histPath); err != nil {
			c.prompter.SetHistory(nil)
		} else {
			c.history = strings.Split(string(content), "\n")
			c.prompter.SetHistory(c.history)
		}
		c.prompter.SetWordCompleter(c.AutoCompleteInput)
	}
	return nil
}

func (c *Console) clearHistory() {
	c.history = nil
	c.prompter.ClearHistory()
	if err := os.Remove(c.histPath); err != nil {
		fmt.Fprintln(c.printer, "can't delete history file:", err)
	} else {
		fmt.Fprintln(c.printer, "history file deleted")
	}
}

// consoleOutput is an override for the console.log and console.error methods to
// stream the output into the configured output stream instead of stdout.
func (c *Console) consoleOutput(call otto.FunctionCall) otto.Value {
	output := []string{}
	for _, argument := range call.ArgumentList {
		output = append(output, fmt.Sprintf("%v", argument))
	}
	fmt.Fprintln(c.printer, strings.Join(output, " "))
	return otto.Value{}
}

// AutoCompleteInput is a pre-assembled word completer to be used by the user
// input prompter to provide hints to the user about the methods available.
func (c *Console) AutoCompleteInput(line string, pos int) (string, []string, string) {
	// No completions can be provided for empty inputs
	if len(line) == 0 || pos == 0 {
		return "", nil, ""
	}
	// Chunck data to relevant part for autocompletion
	// E.g. in case of nested lines eth.getBalance(eth.coinb<tab><tab>
	start := pos - 1
	for ; start > 0; start-- {
		// Skip all methods and namespaces (i.e. including the dot)
		if line[start] == '.' || (line[start] >= 'a' && line[start] <= 'z') || (line[start] >= 'A' && line[start] <= 'Z') {
			continue
		}
		// Handle web3 in a special way (i.e. other numbers aren't auto completed)
		if start >= 3 && line[start-3:start] == "web3" {
			start -= 3
			continue
		}
		// We've hit an unexpected character, autocomplete form here
		start++
		break
	}
	return line[:start], c.jsre.CompleteKeywords(line[start:pos]), line[pos:]
}

// Welcome show summary of current Geth instance and some metadata about the
// console's available modules.
func (c *Console) Welcome() {
	// Print some generic Geth metadata
	fmt.Fprintf(c.printer, "Welcome to the Geth JavaScript console!\n\n")
	c.jsre.Run(`
		console.log("instance: " + web3.version.node);
		console.log("coinbase: " + eth.coinbase);
		console.log("at block: " + eth.blockNumber + " (" + new Date(1000 * eth.getBlock(eth.blockNumber).timestamp) + ")");
		console.log(" datadir: " + admin.datadir);
	`)
	// List all the supported modules for the user to call
	if apis, err := c.client.SupportedModules(); err == nil {
		modules := make([]string, 0, len(apis))
		for api, version := range apis {
			modules = append(modules, fmt.Sprintf("%s:%s", api, version))
		}
		sort.Strings(modules)
		fmt.Fprintln(c.printer, " modules:", strings.Join(modules, " "))
	}
	fmt.Fprintln(c.printer)
}

// Evaluate executes code and pretty prints the result to the specified output
// stream.
func (c *Console) Evaluate(statement string) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(c.printer, "[native] error: %v\n", r)
		}
	}()
	return c.jsre.Evaluate(statement, c.printer)
}

// Interactive starts an interactive user session, where input is propted from
// the configured user prompter.
func (c *Console) Interactive() {
	var (
		prompt    = c.prompt          // Current prompt line (used for multi-line inputs)
		indents   = 0                 // Current number of input indents (used for multi-line inputs)
		input     = ""                // Current user input
		scheduler = make(chan string) // Channel to send the next prompt on and receive the input
	)
	// Start a goroutine to listen for promt requests and send back inputs
	go func() {
		for {
			// Read the next user input
			line, err := c.prompter.PromptInput(<-scheduler)
			if err != nil {
				// In case of an error, either clear the prompt or fail
				if err == liner.ErrPromptAborted { // ctrl-C
					prompt, indents, input = c.prompt, 0, ""
					scheduler <- ""
					continue
				}
				close(scheduler)
				return
			}
			// User input retrieved, send for interpretation and loop
			scheduler <- line
		}
	}()
	// Monitor Ctrl-C too in case the input is empty and we need to bail
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt)

	// Start sending prompts to the user and reading back inputs
	for {
		// Send the next prompt, triggering an input read and process the result
		scheduler <- prompt
		select {
		case <-abort:
			// User forcefully quite the console
			fmt.Fprintln(c.printer, "caught interrupt, exiting")
			return

		case line, ok := <-scheduler:
			// User input was returned by the prompter, handle special cases
			if !ok || (indents <= 0 && exit.MatchString(line)) {
				return
			}
			if onlyWhitespace.MatchString(line) {
				continue
			}
			// Append the line to the input and check for multi-line interpretation
			input += line + "\n"

			indents = countIndents(input)
			if indents <= 0 {
				prompt = c.prompt
			} else {
				prompt = strings.Repeat(".", indents*3) + " "
			}
			// If all the needed lines are present, save the command and run
			if indents <= 0 {
				if len(input) > 0 && input[0] != ' ' && !passwordRegexp.MatchString(input) {
					if command := strings.TrimSpace(input); len(c.history) == 0 || command != c.history[len(c.history)-1] {
						c.history = append(c.history, command)
						if c.prompter != nil {
							c.prompter.AppendHistory(command)
						}
					}
				}
				c.Evaluate(input)
				input = ""
			}
		}
	}
}

// countIndents returns the number of identations for the given input.
// In case of invalid input such as var a = } the result can be negative.
func countIndents(input string) int {
	var (
		indents     = 0
		inString    = false
		strOpenChar = ' '   // keep track of the string open char to allow var str = "I'm ....";
		charEscaped = false // keep track if the previous char was the '\' char, allow var str = "abc\"def";
	)

	for _, c := range input {
		switch c {
		case '\\':
			// indicate next char as escaped when in string and previous char isn't escaping this backslash
			if !charEscaped && inString {
				charEscaped = true
			}
		case '\'', '"':
			if inString && !charEscaped && strOpenChar == c { // end string
				inString = false
			} else if !inString && !charEscaped { // begin string
				inString = true
				strOpenChar = c
			}
			charEscaped = false
		case '{', '(':
			if !inString { // ignore brackets when in string, allow var str = "a{"; without indenting
				indents++
			}
			charEscaped = false
		case '}', ')':
			if !inString {
				indents--
			}
			charEscaped = false
		default:
			charEscaped = false
		}
	}

	return indents
}

// Execute runs the JavaScript file specified as the argument.
func (c *Console) Execute(path string) error {
	return c.jsre.Exec(path)
}

// Stop cleans up the console and terminates the runtime environment.
func (c *Console) Stop(graceful bool) error {
	if err := ioutil.WriteFile(c.histPath, []byte(strings.Join(c.history, "\n")), 0600); err != nil {
		return err
	}
	if err := os.Chmod(c.histPath, 0600); err != nil { // Force 0600, even if it was different previously
		return err
	}
	c.jsre.Stop(graceful)
	return nil
}
