package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func throwError(err string) {
	fmt.Printf("Error: %v\n", err)
	os.Exit(0)
}

func requireExecutable(cmd string) {
	_, err := exec.LookPath(cmd)
	if err != nil {
		throwError(fmt.Sprintf("Missing executable \"%v\". ", cmd))
	}
}

var outputName string
var goos string
var goarch string
var goversion string = "latest"

const (
	outputNameDefault = "main"
	outputNameUsage   = "Name of the outputted file"
	osDefault         = "linux"
	osUsage           = "Choosing OS to compile to"
	osShorthand       = "s"
	archDefault       = "amd64"
	archUsage         = "Choosing architecture to compile to"
	archShorthand     = "a"
)

func init() {
	requireExecutable("docker")

	flag.StringVar(&outputName, "output", outputNameDefault, outputNameUsage)
	flag.StringVar(&outputName, "o", outputNameDefault, outputNameUsage)

	flag.StringVar(&goos, "goos", osDefault, osUsage)
	flag.StringVar(&goos, "gs", osDefault, osUsage)

	flag.StringVar(&goarch, "goarch", archDefault, archUsage)
	flag.StringVar(&goarch, "ga", archDefault, archUsage)

	flag.Usage = func() {
		flagSet := flag.CommandLine
		order := []string{"goarch", "goos", "output"}
		for _, name := range order {
			flag := flagSet.Lookup(name)
			switch name {
			case "goarch":
				fmt.Printf("-%s --%s\n", archShorthand, flag.Name)
			case "goos":
				fmt.Printf("-%s --%s\n", osShorthand, flag.Name)
			default:
				fmt.Printf("-%s --%s\n", string(flag.Name[0]), flag.Name)
			}
			fmt.Printf("  %s\n", flag.Usage)
		}
	}

	flag.Parse()

}

func main() {
	user, err := user.Current()
	if err != nil {
		throwError("Couldn't get current user.")
	}

	userGopath := os.Getenv("GOPATH")
	if userGopath == "" {
		userGopath = build.Default.GOPATH
	}

	pwd, err := os.Getwd()
	if err != nil {
		throwError("Couldn't get working directory.")
	}

	cmd := exec.Command("docker", "inspect", "golang:"+goversion)

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Pulling image for golang:%s\n", goversion)
		cmd = exec.Command("docker", "pull", "golang:"+goversion)
		err = cmd.Run()

	}

	repoName := strings.Replace(pwd, userGopath+"/src/", "", -1)
	volumeMount := fmt.Sprintf("%v:/go/src/%v", pwd, repoName)
	shellScript := fmt.Sprintf("cd /go/src/%v && go build -a -o %v && chown %v:%v %v", repoName, outputName, user.Uid, user.Gid, outputName)

	dockerArgs := []string{"run", "--rm", "-v", volumeMount, "-e", "GOOS=" + goos, "-e", "CGO_ENABLED=0", "-e", "GOARCH=" + goarch, "golang", "sh", "-c", shellScript}

	cmd = exec.Command("docker", dockerArgs...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	fmt.Printf("Compile for %v using architecture %v and output a file named \"%v\"\n", strings.Title(goos), goarch, outputName)
	err = cmd.Run()
	if err != nil {
		throwError(stderr.String())
	}

	fmt.Println("Compiled!")
}
