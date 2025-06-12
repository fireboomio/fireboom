package cmd

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/server"
	"go.uber.org/zap"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var buildCmd = &cobra.Command{
	Use:     "build",
	Short:   "Build fireboom application",
	Long:    `Build fireboom application to apply the new configuration`,
	Example: `./fireboom build`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlags(cmd.Flags())
		viper.Set(consts.EngineFirstStatus, consts.EngineBuilding)
		utils.ExecuteInitMethods()
		engineBuildWithWaitGroup()
	},
}

func engineBuildWithWaitGroup() bool {
	var group sync.WaitGroup
	if err := server.EngineBuilder.GenerateGraphqlConfig(&group); err != nil {
		return false
	}

	group.Wait()
	zap.L().Info("build success")
	return true
}

func init() {
	buildCmd.Flags().String(consts.ActiveMode, consts.DefaultProdActive, "Mode active to run in different environment")
	buildCmd.Flags().String(consts.Workdir, "", "Working directory to build the application")
	buildCmd.Flags().Bool(consts.IgnoreMergeEnvironment, false, "Whether Ignore merge environment")
	rootCmd.AddCommand(buildCmd)
}
