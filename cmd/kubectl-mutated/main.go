package main

import (
	"os"
	"path"

	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	if path.Base(os.Args[0]) == "kubectl_complete-mutated" {
		mutatedCmd.SetArgs(append([]string{cobra.ShellCompRequestCmd}, os.Args[1:]...))
	}
	mutatedCmd.Execute()
}
