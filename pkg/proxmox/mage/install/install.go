package install

import (
	"fmt"
	"os/exec"

	"github.com/magefile/mage/mg"
)

const (
	GolangCILintVersion = "v2.8.0"
)

func Dependencies() error {
	fmt.Println("Installing Dependencies...")
	mg.Deps(Golangcilint)

	return nil
}

func Golangcilint() error {
	fmt.Println("Installing GolangCI Lint...")
	cmd := exec.Command("go", "install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"+GolangCILintVersion)
	return cmd.Run()
}
