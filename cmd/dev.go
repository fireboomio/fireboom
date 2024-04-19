package cmd

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/server"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var devCmd = &cobra.Command{
	Use:     "dev",
	Short:   "Start fireboom in development mode",
	Long:    `Start the fireboom application in development mode and watch for changes`,
	Example: `./fireboom dev`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlags(cmd.Flags())
		viper.Set(consts.DevMode, true)
		viper.Set(consts.EnableSwagger, true)
		viper.Set(consts.EnableWebConsole, true)
		viper.Set(consts.EnableDebugPprof, true)
		viper.Set(consts.EngineFirstStatus, consts.EngineBuilding)
		utils.ExecuteInitMethods()
		if err := server.GenerateAuthenticationKey(); err != nil {
			return
		}

		server.Run(utils.BuildAndStart)
	},
}

func init() {
	devCmd.Flags().String(consts.WebPort, consts.DefaultWebPort, "Web port to listen on")
	devCmd.Flags().String(consts.ActiveMode, "", "Mode active to run in different environment")
	devCmd.Flags().Bool(consts.IgnoreMergeEnvironment, false, "Whether Ignore merge environment")

	devCmd.Flags().Bool(consts.EnableHookReport, true, "Whether enable hook report on dev mode")
	devCmd.Flags().Bool(consts.EnableAuth, false, "Whether enable auth key on dev mode")
	rootCmd.AddCommand(devCmd)
}
