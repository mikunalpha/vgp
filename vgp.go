package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-ini/ini"
)

var (
	rootPath = ""
	srcPath  = ""
	distPath = ""

	packageName = ""
	packagePath = ""
	outName     = ""
	outPath     = ""
)

func main() {
	var err error

	rootPath, err = os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	srcPath = filepath.Join(rootPath, "src")
	distPath = filepath.Join(rootPath, "dist")

	// Read configuration
	config, err := ini.Load(filepath.Join(rootPath, "vgp.ini"))
	if err != nil {
		log.Fatalln(err)
	}
	packageName = config.Section("").Key("package_name").String()
	if packageName == "" {
		log.Fatalln(errors.New("Configuration Error. package_name is required"))
	}
	packagePath = filepath.Join(srcPath, packageName)

	outName = config.Section("").Key("out").String()
	if outName == "" {
		log.Fatalln(errors.New("Configuration Error. out is required"))
	}
	outPath = filepath.Join(distPath, outName)

	if len(os.Args) == 1 {
		return
	}

	switch os.Args[1] {
	case "init":
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
			log.Fatalln(err)
		}
	}

	// create vscode settings.json
	vscodePath := filepath.Join(rootPath, ".vscode")
	_, err = os.Stat(vscodePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(vscodePath, 0775)
		if err != nil {
			log.Fatalln(err)
		}
	}
	originGopath := os.Getenv("GOPATH")
	vscodeSettingsPath := filepath.Join(vscodePath, "settings.json")
	settingsContent := []byte(fmt.Sprintf("{\n  \"go.path\": \"${workspaceRoot}\",\n  \"go.toolsGopath\": \"%s\"\n}", originGopath))
	err = ioutil.WriteFile(vscodeSettingsPath, settingsContent, 0664)
	if err != nil {
		log.Fatalln(err)
	}

	// Set GOPATH
	os.Setenv("GOPATH", rootPath)

	err = os.Chdir(packagePath)
	if err != nil {
		log.Fatalln(err)
	}

	// glide init
	err = execute("/usr/bin/env", "glide", "create", "--skip-import", "--non-interactive")
	if err != nil {
		log.Fatalln(err)
	}
	// glide install
	err = execute("/usr/bin/env", "glide", "install", "--skip-test", "--strip-vendor")
	if err != nil {
		log.Fatalln(err)
	}
}

func proxyCommand() {
	var err error

	// Set GOPATH
	os.Setenv("GOPATH", rootPath)

	err = os.Chdir(packagePath)
	if err != nil {
		log.Fatalln(err)
	}

	switch os.Args[1] {
	case "build":
		_, err = os.Stat(distPath)
		if os.IsNotExist(err) {
			err = os.MkdirAll(distPath, 0775)
			if err != nil {
				log.Fatalln(err)
			}
		}

		replaced := false
		for i := range os.Args {
			if os.Args[i] != "-o" {
				continue
			}
			if i == len(os.Args)-1 {
				os.Args = append(os.Args, distPath)
			} else if strings.HasPrefix(os.Args[i+1], "-") {
				tmp := append([]string{distPath}, os.Args[i+1:]...)
				os.Args = append(os.Args[:i+1], tmp...)
			} else {
				os.Args[i+1] = distPath
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
		err = execute("/usr/bin/env", os.Args...)
		if err != nil {
			log.Fatalln(err)
		}
		err = execute("/usr/bin/env", "strip", "-s", outPath)
		if err != nil {
			log.Fatalln(err)
		}
	case "get", "up", "rm":
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
		return errors.New("Executation Error")
	}

	return nil
}
