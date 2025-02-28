package cmd

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	engineServer "fireboom-server/pkg/engine/server"
	"fireboom-server/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "Start fireboom in production mode",
	Long:    `Start the fireboom application in production mode without watching`,
	Example: `./fireboom start`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlags(cmd.Flags())
		viper.Set(consts.EngineFirstStatus, consts.EngineStarting)
		utils.ExecuteInitMethods()
		utils.SetWithLockViper(consts.EnableAuth, true, true)
		if build.GeneratedJsonLoadErrored() {
			return
		}
		if err := server.GenerateAuthenticationKey(); err != nil {
			return
		}
		server.Run(func() {
			if utils.GetBoolWithLockViper(consts.EnableRebuild) && !engineBuildWithWaitGroup() {
				return
			}
			engineServer.EngineStarter.StartNodeServer()
		})
	},
}

func init() {
	startCmd.Flags().String(consts.WebPort, consts.DefaultWebPort, "Web port to listen on")
	startCmd.Flags().String(consts.ActiveMode, consts.DefaultProdActive, "Mode active to run in different environment")
	startCmd.Flags().Bool(consts.EnableLogicDelete, false, "Whether enable logic delete for multiple data")
	startCmd.Flags().Bool(consts.IgnoreMergeEnvironment, true, "Whether Ignore merge environment")

	startCmd.Flags().Bool(consts.EnableSwagger, false, "Whether enable swagger in production")
	startCmd.Flags().Bool(consts.EnableRebuild, false, "Whether enable rebuild in production")
	startCmd.Flags().Bool(consts.EnableWebConsole, true, "Whether enable web console page in production")
	startCmd.Flags().Bool(consts.EnableDebugPprof, false, "Whether enable /debug/pprof router in production")
	startCmd.Flags().Bool(consts.RegenerateKey, false, "Whether to renew authentication key in production")
	rootCmd.AddCommand(startCmd)
}
