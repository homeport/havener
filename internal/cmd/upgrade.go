// Copyright © 2018 The Havener
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"time"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/wait"
	"github.com/gonvenience/wrap"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	envVarUpgradeConfig     = "upgrade_config"
	envVarUpgradeTimeout    = "upgrade_timeout"
	envVarUpgradeValueReuse = "upgrade_reuse_values"
)

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:           "upgrade [havener config file]",
	Short:         "Upgrades Helm Charts using a havener configuration",
	Long:          `Upgrades Helm Charts using a havener configuration file.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		havenerConfig := viper.GetString(envVarUpgradeConfig)

		hvnr, err := havener.NewHavener()
		if err != nil {
			return err
		}

		switch {
		case len(args) == 1:
			return UpgradeViaHavenerConfig(hvnr, args[0])
		case len(havenerConfig) > 0:
			return UpgradeViaHavenerConfig(hvnr, havenerConfig)
		default:
			cmd.Usage()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	upgradeCmd.PersistentFlags().String("config", "", "havener configuration file")
	upgradeCmd.PersistentFlags().Int("timeout", 40, "upgrade timeout in minutes")
	upgradeCmd.PersistentFlags().Bool("reuse-values", false, "reuse the last release's values and merge in any overrides")

	viper.AutomaticEnv()
	viper.BindPFlag(envVarUpgradeConfig, upgradeCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag(envVarUpgradeTimeout, upgradeCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag(envVarUpgradeValueReuse, upgradeCmd.PersistentFlags().Lookup("reuse-values"))
}

// UpgradeViaHavenerConfig override an existing helm chart
func UpgradeViaHavenerConfig(hvnr havener.Havener, havenerConfig string) error {
	timeoutInMin := viper.GetInt(envVarUpgradeTimeout)
	reuseValues := viper.GetBool(envVarUpgradeValueReuse)

	config, err := havener.ProcessConfigFile(havenerConfig)
	if err != nil {
		return err
	}

	if err := processTask("Pre-upgrade Steps", config.Before); err != nil {
		return wrap.Error(err, "failed to evaluate pre-upgrade steps")
	}

	for _, release := range config.Releases {
		overridesData, err := marshallOverrideSection(release)
		if err != nil {
			return err
		}

		if err := processTask("Before Chart "+release.ChartName, release.Before); err != nil {
			return wrap.Error(err, "failed to evaluate before release steps")
		}

		if err := showHelmReleaseDiff(hvnr, release.ChartName, release.ChartLocation, overridesData, reuseValues); err != nil {
			return wrap.Error(err, "failed to show differences before upgrade")
		}

		pi := wait.NewProgressIndicator(fmt.Sprintf("Upgrading Helm Release for %s", release.ChartName))
		pi.SetTimeout(time.Duration(timeoutInMin) * time.Minute)
		setCurrentProgressIndicator(pi)
		pi.Start()

		err = hvnr.UpdateHelmRelease(
			release.ChartName,
			release.ChartLocation,
			overridesData,
			reuseValues)

		setCurrentProgressIndicator(nil)
		pi.Stop()

		if err != nil {
			return wrap.Error(err, "failed to upgrade via havener configuration")
		}

		message := bunt.Sprintf("Successfully upgraded helm chart *%s* in namespace *_%s_*.",
			release.ChartName,
			release.ChartNamespace,
		)

		message, err = hvnr.GetReleaseMessage(release, message)
		if err != nil {
			return err
		}

		printStatusMessage("Upgrade", message, bunt.Gray)

		if err := processTask("After Chart "+release.ChartName, release.After); err != nil {
			return wrap.Error(err, "failed to evaluate after release steps")
		}
	}

	if err := processTask("Post-upgrade Steps", config.After); err != nil {
		return wrap.Error(err, "failed to evaluate post-upgrade steps")
	}
	return nil
}
