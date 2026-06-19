package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z". For
// `go install`ed binaries it is resolved from the module build info instead.
var version = "dev"

const repo = "ricardojparram/monkeytui"

// resolveVersion returns the best-known version string for this binary.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}

// dispatchCommand handles the `update`, `uninstall` and `version` subcommands.
// It returns true if a subcommand ran (and the caller should exit).
func dispatchCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "version", "-version", "--version":
		fmt.Println("monkeytui " + resolveVersion())
	case "update", "upgrade", "-update", "--update":
		runUpdate()
	case "uninstall", "remove", "-uninstall", "--uninstall":
		runUninstall()
	default:
		return false
	}
	return true
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "error: "+msg)
	os.Exit(1)
}

// assetName is the release asset matching the running OS/arch.
func assetName() string {
	n := "monkeytui_" + runtime.GOOS + "_" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		n += ".exe"
	}
	return n
}

// latestTag fetches the newest release tag from the GitHub API.
func latestTag() (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/" + repo + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var j struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		return "", err
	}
	if j.TagName == "" {
		return "", fmt.Errorf("no release tag found")
	}
	return j.TagName, nil
}

// runUpdate downloads the latest release binary and replaces this executable.
func runUpdate() {
	cur := resolveVersion()
	fmt.Println("current version: " + cur)

	latest, err := latestTag()
	if err != nil {
		fail("could not check for updates: " + err.Error())
	}
	if cur == latest {
		fmt.Println("already up to date.")
		return
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, latest, assetName())
	fmt.Printf("updating %s -> %s\n", cur, latest)
	if err := replaceSelf(url); err != nil {
		fail(err.Error())
	}
	fmt.Println("updated to " + latest + ". run: monkeytui")
}

// replaceSelf streams url into a temp file beside the executable, then renames
// it over the running binary (atomic on the same filesystem; allowed on Linux
// and macOS even while running).
func replaceSelf(url string) error {
	exe, err := selfPath()
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s (no prebuilt binary for %s/%s?)",
			resp.Status, runtime.GOOS, runtime.GOARCH)
	}

	tmp := exe + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("%v (try: sudo monkeytui update)", err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	out.Close()

	if err := os.Rename(tmp, exe); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("%v (try: sudo monkeytui update)", err)
	}
	return nil
}

// runUninstall removes this executable after confirmation.
func runUninstall() {
	exe, err := selfPath()
	if err != nil {
		fail(err.Error())
	}
	fmt.Printf("this will remove: %s\n", exe)
	if !confirm("continue? [y/N] ") {
		fmt.Println("cancelled.")
		return
	}
	if err := os.Remove(exe); err != nil {
		fail(err.Error() + " (try: sudo monkeytui uninstall)")
	}
	fmt.Println("uninstalled. thanks for typing!")
}

// selfPath returns the resolved path of the running executable.
func selfPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// confirm prompts on stdin; a non-interactive stdin defaults to yes.
func confirm(prompt string) bool {
	fi, _ := os.Stdin.Stat()
	if fi != nil && fi.Mode()&os.ModeCharDevice == 0 {
		return true // piped / non-interactive: proceed
	}
	fmt.Print(prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
