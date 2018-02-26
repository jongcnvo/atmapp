package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// These flags override values in build env.
	GitCommitFlag   = flag.String("git-commit", "", `Overrides git commit hash embedded into executables`)
	GitBranchFlag   = flag.String("git-branch", "", `Overrides git branch being built`)
	GitTagFlag      = flag.String("git-tag", "", `Overrides git tag being built`)
	BuildnumFlag    = flag.String("buildnum", "", `Overrides CI build number`)
	PullRequestFlag = flag.Bool("pull-request", false, `Overrides pull request status of the build`)
	CronJobFlag     = flag.Bool("cron-job", false, `Overrides cron job status of the build`)
)

// Environment contains metadata provided by the build environment.
type Environment struct {
	Name                string // name of the environment
	Repo                string // name of GitHub repo
	Commit, Branch, Tag string // Git info
	Buildnum            string
	IsPullRequest       bool
	IsCronJob           bool
}

// Env returns build environment metadata gathered from git.
func Env() Environment {
	env := applyEnvFlags(Environment{Name: "local", Repo: "atmchain/atmapp"})

	head := readGitFile("HEAD")
	if splits := strings.Split(head, " "); len(splits) == 2 {
		head = splits[1]
	} else {
		return env
	}
	if env.Commit == "" {
		env.Commit = readGitFile(head)
	}
	if env.Branch == "" {
		if head != "HEAD" {
			env.Branch = strings.TrimLeft(head, "refs/heads/")
		}
	}
	if info, err := os.Stat(".git/objects"); err == nil && info.IsDir() && env.Tag == "" {
		env.Tag = firstLine(RunGit("tag", "-l", "--points-at", "HEAD"))
	}
	return env
}

// readGitFile returns content of file in .git directory.
func readGitFile(file string) string {
	content, err := ioutil.ReadFile(path.Join(".git", file))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

var warnedAboutGit bool

// RunGit runs a git subcommand and returns its output.
// The command must complete successfully.
func RunGit(args ...string) string {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err == exec.ErrNotFound {
		if !warnedAboutGit {
			log.Println("Warning: can't find 'git' in PATH")
			warnedAboutGit = true
		}
		return ""
	} else if err != nil {
		log.Fatal(strings.Join(cmd.Args, " "), ": ", err, "\n", stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

func applyEnvFlags(env Environment) Environment {
	if !flag.Parsed() {
		panic("you need to call flag.Parse before Env or LocalEnv")
	}
	if *GitCommitFlag != "" {
		env.Commit = *GitCommitFlag
	}
	if *GitBranchFlag != "" {
		env.Branch = *GitBranchFlag
	}
	if *GitTagFlag != "" {
		env.Tag = *GitTagFlag
	}
	if *BuildnumFlag != "" {
		env.Buildnum = *BuildnumFlag
	}
	if *PullRequestFlag {
		env.IsPullRequest = true
	}
	if *CronJobFlag {
		env.IsCronJob = true
	}
	return env
}

func firstLine(s string) string {
	return strings.Split(s, "\n")[0]
}

// ExpandPackagesNoVendor expands a cmd/go import path pattern, skipping
// vendored packages.
func ExpandPackagesNoVendor(patterns []string) []string {
	expand := false
	for _, pkg := range patterns {
		if strings.Contains(pkg, "...") {
			expand = true
		}
	}
	if expand {
		cmd := GoTool("list", patterns...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("package listing failed: %v\n%s", err, string(out))
		}
		var packages []string
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, "/vendor/") {
				packages = append(packages, strings.TrimSpace(line))
			}
		}
		return packages
	}
	return patterns
}

// GoTool returns the command that runs a go tool. This uses go from GOROOT instead of PATH
// so that go commands executed by build use the same version of Go as the 'host' that runs
// build code. e.g.
//
//     /usr/lib/go-1.8/bin/go run build/ci.go ...
//
// runs using go 1.8 and invokes go 1.8 tools from the same GOROOT. This is also important
// because runtime.Version checks on the host should match the tools that are run.
func GoTool(tool string, args ...string) *exec.Cmd {
	args = append([]string{tool}, args...)
	return exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), args...)
}

var DryRunFlag = flag.Bool("n", false, "dry run, don't execute commands")

// MustRun executes the given command and exits the host process for
// any error.
func MustRun(cmd *exec.Cmd) {
	fmt.Println(">>>", strings.Join(cmd.Args, " "))
	if !*DryRunFlag {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

// GOPATH returns the value that the GOPATH environment
// variable should be set to.
func GOPATH() string {
	if os.Getenv("GOPATH") == "" {
		log.Fatal("GOPATH is not set")
	}
	return os.Getenv("GOPATH")
}

var GOBIN, _ = filepath.Abs(filepath.Join("build", "bin"))

func executablePath(name string) string {
	return filepath.Join(GOBIN, name)
}

func main() {
	log.SetFlags(log.Lshortfile)

	if _, err := os.Stat(filepath.Join("build", "ci.go")); os.IsNotExist(err) {
		log.Fatal("this script must be run from the root of the repository")
	}
	if len(os.Args) < 2 {
		log.Fatal("need subcommand as first argument")
	}
	switch os.Args[1] {
	case "install":
		doInstall(os.Args[2:])
	default:
		log.Fatal("unknown command ", os.Args[1])
	}
}

// Compiling

func doInstall(cmdline []string) {
	var (
		arch = flag.String("arch", "", "Architecture to cross build for")
		cc   = flag.String("cc", "", "C compiler to cross build with")
	)
	flag.CommandLine.Parse(cmdline)
	env := Env()

	// Check Go version. People regularly open issues about compilation
	// failure with outdated Go. This should save them the trouble.
	if !strings.Contains(runtime.Version(), "devel") {
		// Figure out the minor version number since we can't textually compare (1.10 < 1.7)
		var minor int
		fmt.Sscanf(strings.TrimPrefix(runtime.Version(), "go1."), "%d", &minor)

		if minor < 7 {
			log.Println("You have Go version", runtime.Version())
			log.Println("atmapp requires at least Go version 1.7 and cannot")
			log.Println("be compiled with an earlier version. Please upgrade your Go installation.")
			os.Exit(1)
		}
	}
	// Compile packages given as arguments, or everything if there are no arguments.
	packages := []string{"./..."}
	if flag.NArg() > 0 {
		packages = flag.Args()
	}
	packages = ExpandPackagesNoVendor(packages)

	if *arch == "" || *arch == runtime.GOARCH {
		goinstall := goTool("install", buildFlags(env)...)
		goinstall.Args = append(goinstall.Args, "-v")
		goinstall.Args = append(goinstall.Args, packages...)
		MustRun(goinstall)
		return
	}
	// If we are cross compiling to ARMv5 ARMv6 or ARMv7, clean any previous builds
	if *arch == "arm" {
		os.RemoveAll(filepath.Join(runtime.GOROOT(), "pkg", runtime.GOOS+"_arm"))
		for _, path := range filepath.SplitList(GOPATH()) {
			os.RemoveAll(filepath.Join(path, "pkg", runtime.GOOS+"_arm"))
		}
	}
	// Seems we are cross compiling, work around forbidden GOBIN
	goinstall := goToolArch(*arch, *cc, "install", buildFlags(env)...)
	goinstall.Args = append(goinstall.Args, "-v")
	goinstall.Args = append(goinstall.Args, []string{"-buildmode", "archive"}...)
	goinstall.Args = append(goinstall.Args, packages...)
	MustRun(goinstall)

	if cmds, err := ioutil.ReadDir("cmd"); err == nil {
		for _, cmd := range cmds {
			pkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Join(".", "cmd", cmd.Name()), nil, parser.PackageClauseOnly)
			if err != nil {
				log.Fatal(err)
			}
			for name := range pkgs {
				if name == "main" {
					gobuild := goToolArch(*arch, *cc, "build", buildFlags(env)...)
					gobuild.Args = append(gobuild.Args, "-v")
					gobuild.Args = append(gobuild.Args, []string{"-o", executablePath(cmd.Name())}...)
					gobuild.Args = append(gobuild.Args, "."+string(filepath.Separator)+filepath.Join("cmd", cmd.Name()))
					MustRun(gobuild)
					break
				}
			}
		}
	}
}

func goTool(subcmd string, args ...string) *exec.Cmd {
	return goToolArch(runtime.GOARCH, os.Getenv("CC"), subcmd, args...)
}

func goToolArch(arch string, cc string, subcmd string, args ...string) *exec.Cmd {
	cmd := GoTool(subcmd, args...)
	if subcmd == "build" || subcmd == "install" || subcmd == "test" {
		// Go CGO has a Windows linker error prior to 1.8 (https://github.com/golang/go/issues/8756).
		// Work around issue by allowing multiple definitions for <1.8 builds.
		var minor int
		fmt.Sscanf(strings.TrimPrefix(runtime.Version(), "go1."), "%d", &minor)

		if runtime.GOOS == "windows" && minor < 8 {
			cmd.Args = append(cmd.Args, []string{"-ldflags", "-extldflags -Wl,--allow-multiple-definition"}...)
		}
	}
	cmd.Env = []string{"GOPATH=" + GOPATH()}
	if arch == "" || arch == runtime.GOARCH {
		cmd.Env = append(cmd.Env, "GOBIN="+GOBIN)
	} else {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
		cmd.Env = append(cmd.Env, "GOARCH="+arch)
	}
	if cc != "" {
		cmd.Env = append(cmd.Env, "CC="+cc)
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOPATH=") || strings.HasPrefix(e, "GOBIN=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	return cmd
}

func buildFlags(env Environment) (flags []string) {
	var ld []string
	if env.Commit != "" {
		ld = append(ld, "-X", "main.gitCommit="+env.Commit)
	}
	if runtime.GOOS == "darwin" {
		ld = append(ld, "-s")
	}

	if len(ld) > 0 {
		flags = append(flags, "-ldflags", strings.Join(ld, " "))
	}
	return flags
}
