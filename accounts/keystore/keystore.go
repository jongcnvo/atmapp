package keystore

import (
	"bufio"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	atmchain "github.com/atmchain/atmapp"
	"github.com/atmchain/atmapp/accounts"
	"github.com/atmchain/atmapp/common"
	"github.com/atmchain/atmapp/core/types"
	"github.com/atmchain/atmapp/crypto"
	"github.com/atmchain/atmapp/event"
	"github.com/atmchain/atmapp/log"
	"github.com/pborman/uuid"
	"github.com/rjeczalik/notify"
	set "gopkg.in/fatih/set.v0"
)

// Minimum amount of time between cache reloads. This limit applies if the platform does
// not support change notifications. It also applies if the keystore directory does not
// exist yet, the code will attempt to create a watcher at most this often.
const minReloadInterval = 2 * time.Second

// Maximum time between wallet refreshes (if filesystem notifications don't work).
const walletRefreshCycle = 3 * time.Second

// KeyStoreScheme is the protocol scheme prefixing account and wallet URLs.
var KeyStoreScheme = "keystore"

var (
	ErrLocked  = accounts.NewAuthNeededError("password or unlock")
	ErrNoMatch = errors.New("no key for given address or file")
	ErrDecrypt = errors.New("could not decrypt key with given passphrase")
)

type Key struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}

type keyStore interface {
	// Loads and decrypts the key from disk.
	GetKey(addr common.Address, filename string, auth string) (*Key, error)
	// Writes and encrypts the key.
	StoreKey(filename string, k *Key, auth string) error
	// Joins filename with the key directory unless it is already absolute.
	JoinPath(filename string) string
}

type watcher struct {
	ac       *accountCache
	starting bool
	running  bool
	ev       chan notify.EventInfo
	quit     chan struct{}
}

// starts the watcher loop in the background.
// Start a watcher in the background if that's not already in progress.
// The caller must hold w.ac.mu.
func (w *watcher) start() {
	if w.starting || w.running {
		return
	}
	w.starting = true
	go w.loop()
}

func (w *watcher) loop() {
	defer func() {
		w.ac.mu.Lock()
		w.running = false
		w.starting = false
		w.ac.mu.Unlock()
	}()
	logger := log.New("path", w.ac.keydir)

	if err := notify.Watch(w.ac.keydir, w.ev, notify.All); err != nil {
		logger.Trace("Failed to watch keystore folder", "err", err)
		return
	}
	defer notify.Stop(w.ev)
	logger.Trace("Started watching keystore folder")
	defer logger.Trace("Stopped watching keystore folder")

	w.ac.mu.Lock()
	w.running = true
	w.ac.mu.Unlock()

	// Wait for file system events and reload.
	// When an event occurs, the reload call is delayed a bit so that
	// multiple events arriving quickly only cause a single reload.
	var (
		debounceDuration = 500 * time.Millisecond
		rescanTriggered  = false
		debounce         = time.NewTimer(0)
	)
	// Ignore initial trigger
	if !debounce.Stop() {
		<-debounce.C
	}
	defer debounce.Stop()
	for {
		select {
		case <-w.quit:
			return
		case <-w.ev:
			// Trigger the scan (with delay), if not already triggered
			if !rescanTriggered {
				debounce.Reset(debounceDuration)
				rescanTriggered = true
			}
		case <-debounce.C:
			w.ac.scanAccounts()
			rescanTriggered = false
		}
	}
}

func (w *watcher) close() {
	close(w.quit)
}

func newWatcher(*accountCache) *watcher { return new(watcher) }

type accountsByURL []accounts.Account

func (s accountsByURL) Len() int           { return len(s) }
func (s accountsByURL) Less(i, j int) bool { return s[i].URL.Cmp(s[j].URL) < 0 }
func (s accountsByURL) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// AmbiguousAddrError is returned when attempting to unlock
// an address for which more than one file exists.
type AmbiguousAddrError struct {
	Addr    common.Address
	Matches []accounts.Account
}

func (err *AmbiguousAddrError) Error() string {
	files := ""
	for i, a := range err.Matches {
		files += a.URL.Path
		if i < len(err.Matches)-1 {
			files += ", "
		}
	}
	return fmt.Sprintf("multiple keys match address (%s)", files)
}

// fileCache is a cache of files seen during scan of keystore.
type fileCache struct {
	all     *set.SetNonTS // Set of all files from the keystore folder
	lastMod time.Time     // Last time instance when a file was modified
	mu      sync.RWMutex
}

// scan performs a new scan on the given directory, compares against the already
// cached filenames, and returns file sets: creates, deletes, updates.
func (fc *fileCache) scan(keyDir string) (set.Interface, set.Interface, set.Interface, error) {
	t0 := time.Now()

	// List all the failes from the keystore folder
	files, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return nil, nil, nil, err
	}
	t1 := time.Now()

	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Iterate all the files and gather their metadata
	all := set.NewNonTS()
	mods := set.NewNonTS()

	var newLastMod time.Time
	for _, fi := range files {
		// Skip any non-key files from the folder
		path := filepath.Join(keyDir, fi.Name())
		if skipKeyFile(fi) {
			log.Trace("Ignoring file on account scan", "path", path)
			continue
		}
		// Gather the set of all and fresly modified files
		all.Add(path)

		modified := fi.ModTime()
		if modified.After(fc.lastMod) {
			mods.Add(path)
		}
		if modified.After(newLastMod) {
			newLastMod = modified
		}
	}
	t2 := time.Now()

	// Update the tracked files and return the three sets
	deletes := set.Difference(fc.all, all)   // Deletes = previous - current
	creates := set.Difference(all, fc.all)   // Creates = current - previous
	updates := set.Difference(mods, creates) // Updates = modified - creates

	fc.all, fc.lastMod = all, newLastMod
	t3 := time.Now()

	// Report on the scanning stats and return
	log.Debug("FS scan times", "list", t1.Sub(t0), "set", t2.Sub(t1), "diff", t3.Sub(t2))
	return creates, deletes, updates, nil
}

// skipKeyFile ignores editor backups, hidden files and folders/symlinks.
func skipKeyFile(fi os.FileInfo) bool {
	// Skip editor backups and UNIX-style hidden files.
	if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
		return true
	}
	// Skip misc special files, directories (yes, symlinks too).
	if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
		return true
	}
	return false
}

// accountCache is a live index of all accounts in the keystore.
type accountCache struct {
	keydir   string
	watcher  *watcher
	mu       sync.Mutex
	all      accountsByURL
	byAddr   map[common.Address][]accounts.Account
	throttle *time.Timer
	notify   chan struct{}
	fileC    fileCache
}

func newAccountCache(keydir string) (*accountCache, chan struct{}) {
	ac := &accountCache{
		keydir: keydir,
		byAddr: make(map[common.Address][]accounts.Account),
		notify: make(chan struct{}, 1),
		fileC:  fileCache{all: set.NewNonTS()},
	}
	ac.watcher = newWatcher(ac)
	return ac, ac.notify
}

func (ac *accountCache) accounts() []accounts.Account {
	ac.maybeReload()
	ac.mu.Lock()
	defer ac.mu.Unlock()
	cpy := make([]accounts.Account, len(ac.all))
	copy(cpy, ac.all)
	return cpy
}

func (ac *accountCache) add(newAccount accounts.Account) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	i := sort.Search(len(ac.all), func(i int) bool { return ac.all[i].URL.Cmp(newAccount.URL) >= 0 })
	if i < len(ac.all) && ac.all[i] == newAccount {
		return
	}
	// newAccount is not in the cache.
	ac.all = append(ac.all, accounts.Account{})
	copy(ac.all[i+1:], ac.all[i:])
	ac.all[i] = newAccount
	ac.byAddr[newAccount.Address] = append(ac.byAddr[newAccount.Address], newAccount)
}

func (ac *accountCache) maybeReload() {
	ac.mu.Lock()

	if ac.watcher.running {
		ac.mu.Unlock()
		return // A watcher is running and will keep the cache up-to-date.
	}
	if ac.throttle == nil {
		ac.throttle = time.NewTimer(0)
	} else {
		select {
		case <-ac.throttle.C:
		default:
			ac.mu.Unlock()
			return // The cache was reloaded recently.
		}
	}
	// No watcher running, start it.
	ac.watcher.start()
	ac.throttle.Reset(minReloadInterval)
	ac.mu.Unlock()
	ac.scanAccounts()
}

// find returns the cached account for address if there is a unique match.
// The exact matching rules are explained by the documentation of accounts.Account.
// Callers must hold ac.mu.
func (ac *accountCache) find(a accounts.Account) (accounts.Account, error) {
	// Limit search to address candidates if possible.
	matches := ac.all
	if (a.Address != common.Address{}) {
		matches = ac.byAddr[a.Address]
	}
	if a.URL.Path != "" {
		// If only the basename is specified, complete the path.
		if !strings.ContainsRune(a.URL.Path, filepath.Separator) {
			a.URL.Path = filepath.Join(ac.keydir, a.URL.Path)
		}
		for i := range matches {
			if matches[i].URL == a.URL {
				return matches[i], nil
			}
		}
		if (a.Address == common.Address{}) {
			return accounts.Account{}, ErrNoMatch
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return accounts.Account{}, ErrNoMatch
	default:
		err := &AmbiguousAddrError{Addr: a.Address, Matches: make([]accounts.Account, len(matches))}
		copy(err.Matches, matches)
		sort.Sort(accountsByURL(err.Matches))
		return accounts.Account{}, err
	}
}

// scanAccounts checks if any changes have occurred on the filesystem, and
// updates the account cache accordingly
func (ac *accountCache) scanAccounts() error {
	// Scan the entire folder metadata for file changes
	creates, deletes, updates, err := ac.fileC.scan(ac.keydir)
	if err != nil {
		log.Debug("Failed to reload keystore contents", "err", err)
		return err
	}
	if creates.Size() == 0 && deletes.Size() == 0 && updates.Size() == 0 {
		return nil
	}
	// Create a helper method to scan the contents of the key files
	var (
		buf = new(bufio.Reader)
		key struct {
			Address string `json:"address"`
		}
	)
	readAccount := func(path string) *accounts.Account {
		fd, err := os.Open(path)
		if err != nil {
			log.Trace("Failed to open keystore file", "path", path, "err", err)
			return nil
		}
		defer fd.Close()
		buf.Reset(fd)
		// Parse the address.
		key.Address = ""
		err = json.NewDecoder(buf).Decode(&key)
		addr := common.HexToAddress(key.Address)
		switch {
		case err != nil:
			log.Debug("Failed to decode keystore key", "path", path, "err", err)
		case (addr == common.Address{}):
			log.Debug("Failed to decode keystore key", "path", path, "err", "missing or zero address")
		default:
			return &accounts.Account{Address: addr, URL: accounts.URL{Scheme: KeyStoreScheme, Path: path}}
		}
		return nil
	}
	// Process all the file diffs
	start := time.Now()

	for _, p := range creates.List() {
		if a := readAccount(p.(string)); a != nil {
			ac.add(*a)
		}
	}
	for _, p := range deletes.List() {
		ac.deleteByFile(p.(string))
	}
	for _, p := range updates.List() {
		path := p.(string)
		ac.deleteByFile(path)
		if a := readAccount(path); a != nil {
			ac.add(*a)
		}
	}
	end := time.Now()

	select {
	case ac.notify <- struct{}{}:
	default:
	}
	log.Trace("Handled keystore changes", "time", end.Sub(start))
	return nil
}

// deleteByFile removes an account referenced by the given path.
func (ac *accountCache) deleteByFile(path string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	i := sort.Search(len(ac.all), func(i int) bool { return ac.all[i].URL.Path >= path })

	if i < len(ac.all) && ac.all[i].URL.Path == path {
		removed := ac.all[i]
		ac.all = append(ac.all[:i], ac.all[i+1:]...)
		if ba := removeAccount(ac.byAddr[removed.Address], removed); len(ba) == 0 {
			delete(ac.byAddr, removed.Address)
		} else {
			ac.byAddr[removed.Address] = ba
		}
	}
}

func (ac *accountCache) close() {
	ac.mu.Lock()
	ac.watcher.close()
	if ac.throttle != nil {
		ac.throttle.Stop()
	}
	if ac.notify != nil {
		close(ac.notify)
		ac.notify = nil
	}
	ac.mu.Unlock()
}

func removeAccount(slice []accounts.Account, elem accounts.Account) []accounts.Account {
	for i := range slice {
		if slice[i] == elem {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// keystoreWallet implements the accounts.Wallet interface for the original
// keystore.
type keystoreWallet struct {
	account  accounts.Account // Single account contained in this wallet
	keystore *KeyStore        // Keystore where the account originates from
}

// URL implements accounts.Wallet, returning the URL of the account within.
func (w *keystoreWallet) URL() accounts.URL {
	return w.account.URL
}

// Status implements accounts.Wallet, returning whether the account held by the
// keystore wallet is unlocked or not.
func (w *keystoreWallet) Status() (string, error) {
	w.keystore.mu.RLock()
	defer w.keystore.mu.RUnlock()

	if _, ok := w.keystore.unlocked[w.account.Address]; ok {
		return "Unlocked", nil
	}
	return "Locked", nil
}

// Accounts implements accounts.Wallet, returning an account list consisting of
// a single account that the plain kestore wallet contains.
func (w *keystoreWallet) Accounts() []accounts.Account {
	return []accounts.Account{w.account}
}

// Open implements accounts.Wallet, but is a noop for plain wallets since there
// is no connection or decryption step necessary to access the list of accounts.
func (w *keystoreWallet) Open(passphrase string) error { return nil }

// Close implements accounts.Wallet, but is a noop for plain wallets since is no
// meaningful open operation.
func (w *keystoreWallet) Close() error { return nil }

// Contains implements accounts.Wallet, returning whether a particular account is
// or is not wrapped by this wallet instance.
func (w *keystoreWallet) Contains(account accounts.Account) bool {
	return account.Address == w.account.Address && (account.URL == (accounts.URL{}) || account.URL == w.account.URL)
}

// Derive implements accounts.Wallet, but is a noop for plain wallets since there
// is no notion of hierarchical account derivation for plain keystore accounts.
func (w *keystoreWallet) Derive(path accounts.DerivationPath, pin bool) (accounts.Account, error) {
	return accounts.Account{}, accounts.ErrNotSupported
}

// SelfDerive implements accounts.Wallet, but is a noop for plain wallets since
// there is no notion of hierarchical account derivation for plain keystore accounts.
func (w *keystoreWallet) SelfDerive(base accounts.DerivationPath, chain atmchain.ChainStateReader) {}

// SignHash implements accounts.Wallet, attempting to sign the given hash with
// the given account. If the wallet does not wrap this particular account, an
// error is returned to avoid account leakage (even though in theory we may be
// able to sign via our shared keystore backend).
func (w *keystoreWallet) SignHash(account accounts.Account, hash []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if account.Address != w.account.Address {
		return nil, accounts.ErrUnknownAccount
	}
	if account.URL != (accounts.URL{}) && account.URL != w.account.URL {
		return nil, accounts.ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignHash(account, hash)
}

// SignTx implements accounts.Wallet, attempting to sign the given transaction
// with the given account. If the wallet does not wrap this particular account,
// an error is returned to avoid account leakage (even though in theory we may
// be able to sign via our shared keystore backend).
func (w *keystoreWallet) SignTx(account accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// Make sure the requested account is contained within
	if account.Address != w.account.Address {
		return nil, accounts.ErrUnknownAccount
	}
	if account.URL != (accounts.URL{}) && account.URL != w.account.URL {
		return nil, accounts.ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignTx(account, tx, chainID)
}

// SignHashWithPassphrase implements accounts.Wallet, attempting to sign the
// given hash with the given account using passphrase as extra authentication.
func (w *keystoreWallet) SignHashWithPassphrase(account accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if account.Address != w.account.Address {
		return nil, accounts.ErrUnknownAccount
	}
	if account.URL != (accounts.URL{}) && account.URL != w.account.URL {
		return nil, accounts.ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignHashWithPassphrase(account, passphrase, hash)
}

// SignTxWithPassphrase implements accounts.Wallet, attempting to sign the given
// transaction with the given account using passphrase as extra authentication.
func (w *keystoreWallet) SignTxWithPassphrase(account accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// Make sure the requested account is contained within
	if account.Address != w.account.Address {
		return nil, accounts.ErrUnknownAccount
	}
	if account.URL != (accounts.URL{}) && account.URL != w.account.URL {
		return nil, accounts.ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignTxWithPassphrase(account, passphrase, tx, chainID)
}

// KeyStoreType is the reflect type of a keystore backend.
var KeyStoreType = reflect.TypeOf(&KeyStore{})

// KeyStore manages a key storage directory on disk.
type KeyStore struct {
	storage  keyStore                     // Storage backend, might be cleartext or encrypted
	cache    *accountCache                // In-memory account cache over the filesystem storage
	changes  chan struct{}                // Channel receiving change notifications from the cache
	unlocked map[common.Address]*unlocked // Currently unlocked account (decrypted private keys)

	wallets     []accounts.Wallet       // Wallet wrappers around the individual key files
	updateFeed  event.Feed              // Event feed to notify wallet additions/removals
	updateScope event.SubscriptionScope // Subscription scope tracking current live listeners
	updating    bool                    // Whether the event notification loop is running

	mu sync.RWMutex
}

type unlocked struct {
	*Key
	abort chan struct{}
}

// NewKeyStore creates a keystore for the given directory.
func NewKeyStore(keydir string, scryptN, scryptP int) *KeyStore {
	keydir, _ = filepath.Abs(keydir)
	ks := &KeyStore{storage: &keyStorePassphrase{keydir, scryptN, scryptP}}
	ks.init(keydir)
	return ks
}

func (ks *KeyStore) init(keydir string) {
	// Lock the mutex since the account cache might call back with events
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Initialize the set of unlocked keys and the account cache
	ks.unlocked = make(map[common.Address]*unlocked)
	ks.cache, ks.changes = newAccountCache(keydir)

	// TODO: In order for this finalizer to work, there must be no references
	// to ks. addressCache doesn't keep a reference but unlocked keys do,
	// so the finalizer will not trigger until all timed unlocks have expired.
	runtime.SetFinalizer(ks, func(m *KeyStore) {
		m.cache.close()
	})
	// Create the initial list of wallets from the cache
	accs := ks.cache.accounts()
	ks.wallets = make([]accounts.Wallet, len(accs))
	for i := 0; i < len(accs); i++ {
		ks.wallets[i] = &keystoreWallet{account: accs[i], keystore: ks}
	}
}

// Subscribe implements accounts.Backend, creating an async subscription to
// receive notifications on the addition or removal of keystore wallets.
func (ks *KeyStore) Subscribe(sink chan<- accounts.WalletEvent) event.Subscription {
	// We need the mutex to reliably start/stop the update loop
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Subscribe the caller and track the subscriber count
	sub := ks.updateScope.Track(ks.updateFeed.Subscribe(sink))

	// Subscribers require an active notification loop, start it
	if !ks.updating {
		ks.updating = true
		go ks.updater()
	}
	return sub
}

// updater is responsible for maintaining an up-to-date list of wallets stored in
// the keystore, and for firing wallet addition/removal events. It listens for
// account change events from the underlying account cache, and also periodically
// forces a manual refresh (only triggers for systems where the filesystem notifier
// is not running).
func (ks *KeyStore) updater() {
	for {
		// Wait for an account update or a refresh timeout
		select {
		case <-ks.changes:
		case <-time.After(walletRefreshCycle):
		}
		// Run the wallet refresher
		ks.refreshWallets()

		// If all our subscribers left, stop the updater
		ks.mu.Lock()
		if ks.updateScope.Count() == 0 {
			ks.updating = false
			ks.mu.Unlock()
			return
		}
		ks.mu.Unlock()
	}
}

// refreshWallets retrieves the current account list and based on that does any
// necessary wallet refreshes.
func (ks *KeyStore) refreshWallets() {
	// Retrieve the current list of accounts
	ks.mu.Lock()
	accs := ks.cache.accounts()

	// Transform the current list of wallets into the new one
	wallets := make([]accounts.Wallet, 0, len(accs))
	events := []accounts.WalletEvent{}

	for _, account := range accs {
		// Drop wallets while they were in front of the next account
		for len(ks.wallets) > 0 && ks.wallets[0].URL().Cmp(account.URL) < 0 {
			events = append(events, accounts.WalletEvent{Wallet: ks.wallets[0], Kind: accounts.WalletDropped})
			ks.wallets = ks.wallets[1:]
		}
		// If there are no more wallets or the account is before the next, wrap new wallet
		if len(ks.wallets) == 0 || ks.wallets[0].URL().Cmp(account.URL) > 0 {
			wallet := &keystoreWallet{account: account, keystore: ks}

			events = append(events, accounts.WalletEvent{Wallet: wallet, Kind: accounts.WalletArrived})
			wallets = append(wallets, wallet)
			continue
		}
		// If the account is the same as the first wallet, keep it
		if ks.wallets[0].Accounts()[0] == account {
			wallets = append(wallets, ks.wallets[0])
			ks.wallets = ks.wallets[1:]
			continue
		}
	}
	// Drop any leftover wallets and set the new batch
	for _, wallet := range ks.wallets {
		events = append(events, accounts.WalletEvent{Wallet: wallet, Kind: accounts.WalletDropped})
	}
	ks.wallets = wallets
	ks.mu.Unlock()

	// Fire all wallet events and return
	for _, event := range events {
		ks.updateFeed.Send(event)
	}
}

// SignHash calculates a ECDSA signature for the given hash. The produced
// signature is in the [R || S || V] format where V is 0 or 1.
func (ks *KeyStore) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	// Look up the key to sign with and abort if it cannot be found
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}
	// Sign the hash using plain ECDSA operations
	return crypto.Sign(hash, unlockedKey.PrivateKey)
}

// Find resolves the given account into a unique entry in the keystore.
func (ks *KeyStore) Find(a accounts.Account) (accounts.Account, error) {
	ks.cache.maybeReload()
	ks.cache.mu.Lock()
	a, err := ks.cache.find(a)
	ks.cache.mu.Unlock()
	return a, err
}

func (ks *KeyStore) getDecryptedKey(a accounts.Account, auth string) (accounts.Account, *Key, error) {
	a, err := ks.Find(a)
	if err != nil {
		return a, nil, err
	}
	key, err := ks.storage.GetKey(a.Address, a.URL.Path, auth)
	return a, key, err
}

// SignHashWithPassphrase signs hash if the private key matching the given address
// can be decrypted with the given passphrase. The produced signature is in the
// [R || S || V] format where V is 0 or 1.
func (ks *KeyStore) SignHashWithPassphrase(a accounts.Account, passphrase string, hash []byte) (signature []byte, err error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	defer zeroKey(key.PrivateKey)
	return crypto.Sign(hash, key.PrivateKey)
}

// SignTx signs the given transaction with the requested account.
func (ks *KeyStore) SignTx(a accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// Look up the key to sign with and abort if it cannot be found
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}
	// Depending on the presence of the chain ID, sign with EIP155 or homestead
	if chainID != nil {
		return types.SignTx(tx, types.NewEIP155Signer(chainID), unlockedKey.PrivateKey)
	}
	return types.SignTx(tx, types.HomesteadSigner{}, unlockedKey.PrivateKey)
}

// SignTxWithPassphrase signs the transaction if the private key matching the
// given address can be decrypted with the given passphrase.
func (ks *KeyStore) SignTxWithPassphrase(a accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	defer zeroKey(key.PrivateKey)

	// Depending on the presence of the chain ID, sign with EIP155 or homestead
	if chainID != nil {
		return types.SignTx(tx, types.NewEIP155Signer(chainID), key.PrivateKey)
	}
	return types.SignTx(tx, types.HomesteadSigner{}, key.PrivateKey)
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

// Wallets implements accounts.Backend, returning all single-key wallets from the
// keystore directory.
func (ks *KeyStore) Wallets() []accounts.Wallet {
	// Make sure the list of wallets is in sync with the account cache
	ks.refreshWallets()

	ks.mu.RLock()
	defer ks.mu.RUnlock()

	cpy := make([]accounts.Wallet, len(ks.wallets))
	copy(cpy, ks.wallets)
	return cpy
}

// Accounts returns all key files present in the directory.
func (ks *KeyStore) Accounts() []accounts.Account {
	return ks.cache.accounts()
}

// Unlock unlocks the given account indefinitely.
func (ks *KeyStore) Unlock(a accounts.Account, passphrase string) error {
	return ks.TimedUnlock(a, passphrase, 0)
}

// TimedUnlock unlocks the given account with the passphrase. The account
// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
// until the program exits. The account must match a unique key file.
//
// If the account address is already unlocked for a duration, TimedUnlock extends or
// shortens the active unlock timeout. If the address was previously unlocked
// indefinitely the timeout is not altered.
func (ks *KeyStore) TimedUnlock(a accounts.Account, passphrase string, timeout time.Duration) error {
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()
	u, found := ks.unlocked[a.Address]
	if found {
		if u.abort == nil {
			// The address was unlocked indefinitely, so unlocking
			// it with a timeout would be confusing.
			zeroKey(key.PrivateKey)
			return nil
		}
		// Terminate the expire goroutine and replace it below.
		close(u.abort)
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{})}
		go ks.expire(a.Address, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	ks.unlocked[a.Address] = u
	return nil
}

func (ks *KeyStore) expire(addr common.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		ks.mu.Lock()
		// only drop if it's still the same key instance that dropLater
		// was launched with. we can check that using pointer equality
		// because the map stores a new pointer every time the key is
		// unlocked.
		if ks.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(ks.unlocked, addr)
		}
		ks.mu.Unlock()
	}
}
