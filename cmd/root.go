package cmd

import (
	"fireboom-server/pkg/common/consts"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fireboom",
	Short: "Fireboom is the next generation API development platform.",
	Long:  `Fireboom is the next generation API development platform, flexible and open, multi-language compatible, easy to learn, to Firebase, but no vendor locked. It helps you build production-level WEB apis without having to spend time repeating coding.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(fbVersion, fbCommit string) {
	if fbVersion == "" {
		fbVersion = consts.DevMode
	}
	if fbCommit == "" {
		fbCommit = consts.DevMode
	}
	viper.Set(consts.FbVersion, fbVersion)
	viper.Set(consts.FbCommit, fbCommit)
	if err := rootCmd.Execute(); err != nil {
		zap.S().Error(err)
		os.Exit(1)
	}
}
