package completion

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/completion"
)

// basically kubectl/pkg/cmd.registerCompletionFuncForGlobalFlags
func RegisterConfigFlagsCompletion(c *cobra.Command, flags *genericclioptions.ConfigFlags) error {
	f := util.NewFactory(flags)
	completion.SetFactoryForCompletion(f)

	for name, fn := range map[string]cobra.CompletionFunc{
		"namespace": func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completion.CompGetResource(f, "namespace", toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		"context": func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completion.ListContextsInConfig(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		"cluster": func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completion.ListClustersInConfig(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		"user": func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completion.ListUsersInConfig(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
	} {
		if err := c.RegisterFlagCompletionFunc(name, fn); err != nil {
			return fmt.Errorf("register completion for %s: %s", name, err)
		}
	}
	return nil
}
