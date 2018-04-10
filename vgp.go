package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-ini/ini"
)

var (
	currentPath = ""
	rootPath    = ""
	configPath  = ""
	srcPath     = ""
	distPath    = ""

	packageName = ""
	packagePath = ""
	outName     = ""
	outPath     = ""
)

func Infof(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format, args...)
}

func Fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func main() {
	var err error

	currentPath, err = os.Getwd()
	if err != nil {
		Fatalf("%s", err)
	}
	rootPath = currentPath

	// find vgp.ini
	for {
		_, err = os.Stat(filepath.Join(rootPath, "vgp.ini"))
		if !os.IsNotExist(err) {
			// Infof("found %s", filepath.Join(rootPath, "vgp.ini"))
			break
		}

		parentPath := filepath.Dir(rootPath)

		if parentPath == rootPath {
			Fatalf("Not a project managed by vgp (or any of the parent directories): vgp.ini")
		}

		rootPath = parentPath
	}

	srcPath = filepath.Join(rootPath, "src")
	distPath = filepath.Join(rootPath, "dist")

	// Read configuration
	config, err := ini.Load(filepath.Join(rootPath, "vgp.ini"))
	if err != nil {
		Fatalf("%s", err)
	}
	packageName = config.Section("").Key("package_name").String()
	if packageName == "" {
		Fatalf("Configuration Error. package_name is required")
	}
	packagePath = filepath.Join(srcPath, packageName)

	outName = config.Section("").Key("out").String()
	if outName == "" {
		Fatalf("Configuration Error. out is required")
	}
	outPath = filepath.Join(distPath, outName)

	if len(os.Args) == 1 {
		return
	}

	switch os.Args[1] {
	case "init", "create":
		initCommand()
	default:
		proxyCommand()
	}
}

func initCommand() {
	var err error

	_, err = os.Stat(packagePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(packagePath, 0775)
		if err != nil {
			Fatalf("%s", err)
		}
	}

	// create vscode settings.json
	vscodePath := filepath.Join(rootPath, ".vscode")
	_, err = os.Stat(vscodePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(vscodePath, 0775)
		if err != nil {
			Fatalf("%s", err)
		}
	}
	originGopath := os.Getenv("GOPATH")
	vscodeSettingsPath := filepath.Join(vscodePath, "settings.json")
	settingsContent := []byte(fmt.Sprintf(`{
	"go.gopath": "${workspaceRoot}",
	"go.toolsGopath": "%s"
}`, originGopath))
	err = ioutil.WriteFile(vscodeSettingsPath, settingsContent, 0664)
	if err != nil {
		Fatalf("%s", err)
	}

	// Set GOPATH
	os.Setenv("GOPATH", rootPath)

	err = os.Chdir(packagePath)
	if err != nil {
		Fatalf("%s", err)
	}

	// glide init
	err = execute("/usr/bin/env", "glide", "create", "--skip-import", "--non-interactive")
	if err != nil {
		Fatalf("%s", err)
	}
	// glide install
	err = execute("/usr/bin/env", "glide", "install", "--skip-test", "--strip-vendor")
	if err != nil {
		Fatalf("%s", err)
	}
}

func proxyCommand() {
	var err error

	// Set GOPATH
	os.Setenv("GOPATH", rootPath)

	err = os.Chdir(packagePath)
	if err != nil {
		Fatalf("%s", err)
	}

	switch os.Args[1] {
	case "build":
		_, err = os.Stat(distPath)
		if os.IsNotExist(err) {
			err = os.MkdirAll(distPath, 0775)
			if err != nil {
				Fatalf("%s", err)
			}
		}

		replaced := false
		for i := range os.Args {
			if os.Args[i] != "-o" {
				continue
			}
			if i == len(os.Args)-1 {
				os.Args = append(os.Args, outPath)
			} else if strings.HasPrefix(os.Args[i+1], "-") {
				tmp := append([]string{outPath}, os.Args[i+1:]...)
				os.Args = append(os.Args[:i+1], tmp...)
			} else {
				if !filepath.IsAbs(os.Args[i+1]) {
					os.Args[i+1] = filepath.Join(currentPath, os.Args[i+1])
				}
			}
			replaced = true
			break
		}
		if !replaced {
			args := make([]string, len(os.Args))
			copy(args, os.Args)
			os.Args = append(os.Args[:2], "-o", outPath)
			os.Args = append(os.Args, args[2:]...)
		}

		os.Args[0] = "go"
		os.Args = append([]string{"CGO_ENABLED=0"}, os.Args...)
		err = execute("/usr/bin/env", os.Args...)
		if err != nil {
			Fatalf("%s", err)
		}
	case "config-wizard", "cw", "get", "update", "up", "remove", "rm", "info", "novendor", "nv", "tree":
		os.Args[0] = "glide"
		execute("/usr/bin/env", os.Args...)
	default:
		os.Args[0] = "go"
		execute("/usr/bin/env", os.Args...)
	}
}

func execute(cmdstr string, args ...string) error {
	cmd := exec.Command(cmdstr, args...)

	cmd.Stdin = os.Stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	err = cmd.Start()
	if err != nil {
		return err
	}

	cmd.Wait()

	if !cmd.ProcessState.Success() {
		return fmt.Errorf("executation error: %s %v", cmdstr, args)
	}

	return nil
}
