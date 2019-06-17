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
	"io/ioutil"
	"time"

	"github.com/homeport/havener/internal/hvnr"

	"github.com/homeport/gonvenience/pkg/v1/bunt"
	"github.com/homeport/gonvenience/pkg/v1/wait"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

const (
	envVarUpgradeConfig     = "upgrade_config"
	envVarUpgradeTimeout    = "upgrade_timeout"
	envVarUpgradeValueReuse = "upgrade_reuse_values"
)

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:           "upgrade",
	Short:         "Upgrade Helm Release in Kubernetes",
	Long:          `Upgrade Helm Release based on havener configuration`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		havenerUpgradeConfig := viper.GetString(envVarUpgradeConfig)

		switch {
		case len(havenerUpgradeConfig) > 0:
			return UpgradeViaHavenerConfig(havenerUpgradeConfig)

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
func UpgradeViaHavenerConfig(havenerConfig string) error {
	timeoutInMin := viper.GetInt(envVarUpgradeTimeout)
	reuseValues := viper.GetBool(envVarUpgradeValueReuse)

	cfgdata, err := ioutil.ReadFile(havenerConfig)
	if err != nil {
		return &ErrorWithMsg{"unable to read havener configuration", err}
	}

	var config havener.Config
	if err := yaml.Unmarshal(cfgdata, &config); err != nil {
		return &ErrorWithMsg{"failed to unmarshal havener configuration", err}
	}

	if err := processTask("Pre-upgrade Steps", config.Before); err != nil {
		return &ErrorWithMsg{"failed to evaluate pre-upgrade steps", err}
	}

	for _, release := range config.Releases {
		overrides, err := havener.TraverseStructureAndProcessOperators(release.Overrides)
		if err != nil {
			return &ErrorWithMsg{"failed to process overrides section", err}
		}

		if err := processTask("Before Chart "+release.ChartName, release.Before); err != nil {
			return &ErrorWithMsg{"failed to evaluate before release steps", err}
		}

		overridesData, err := yaml.Marshal(overrides)
		if err != nil {
			return &ErrorWithMsg{"failed to marshal overrides structure into bytes", err}
		}

		if err := hvnr.ShowHelmReleaseDiff(release.ChartName, release.ChartLocation, overridesData, reuseValues); err != nil {
			return &ErrorWithMsg{"failed to show differences before upgrade", err}
		}

		pi := wait.NewProgressIndicator(fmt.Sprintf("Upgrading Helm Release for %s", release.ChartName))
		pi.SetTimeout(time.Duration(timeoutInMin) * time.Minute)
		pi.Start()

		err = havener.UpdateHelmRelease(
			release.ChartName,
			release.ChartLocation,
			overridesData,
			reuseValues)

		pi.Stop()

		if err != nil {
			return &ErrorWithMsg{"failed to upgrade via havener configuration", err}
		}

		message := bunt.Sprintf("Successfully upgraded helm chart *%s* in namespace *_%s_*.",
			release.ChartName,
			release.ChartNamespace,
		)

		releaseNotes, err := havener.RunHelmBinary("get", "notes", release.ChartName)
		if err != nil {
			return &ErrorWithMsg{"failed to get notes of release", err}
		}

		if releaseNotes != nil {
			message = message + "\n\n" + string(releaseNotes)
		}

		printStatusMessage("Upgrade", message, bunt.Gray)

		if err := processTask("After Chart "+release.ChartName, release.After); err != nil {
			return &ErrorWithMsg{"failed to evaluate after release steps", err}
		}
	}

	if err := processTask("Post-upgrade Steps", config.After); err != nil {
		return &ErrorWithMsg{"failed to evaluate post-upgrade steps", err}
	}
	return nil
}
