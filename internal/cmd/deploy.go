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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/homeport/gonvenience/pkg/v1/wait"
	"gopkg.in/yaml.v2"

	"github.com/homeport/havener/pkg/havener"
)

const (
	envVarDeployConfig  = "deployment_config"
	envVarDeployTimeout = "deployment_timeout"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy to Kubernetes",
	Long:  `Deploy to Kubernetes based on havener configuration`,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case len(viper.GetString(envVarDeployConfig)) > 0:
			deployViaHavenerConfig()

		default:
			cmd.Usage()
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.PersistentFlags().String("config", "", "havener configuration file")
	deployCmd.PersistentFlags().Int("timeout", 40, "deployment timeout in minutes")

	viper.AutomaticEnv()
	viper.BindPFlag(envVarDeployConfig, deployCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag(envVarDeployTimeout, deployCmd.PersistentFlags().Lookup("timeout"))
}

func deployViaHavenerConfig() {
	havenerConfig := viper.GetString(envVarDeployConfig)
	timeoutInMin := viper.GetInt(envVarDeployTimeout)

	cfgdata, err := ioutil.ReadFile(havenerConfig)
	if err != nil {
		exitWithError("unable to read havener configuration", err)
	}

	var config havener.Config
	if err := yaml.Unmarshal(cfgdata, &config); err != nil {
		exitWithError("failed to unmarshal havener configuration", err)
	}

	for _, release := range config.Releases {
		overrides, err := havener.TraverseStructureAndProcessShellOperators(release.Overrides)
		if err != nil {
			exitWithError("failed to process overrides section", err)
		}

		overridesData, err := yaml.Marshal(overrides)
		if err != nil {
			exitWithError("failed to marshal overrides structure into bytes", err)
		}

		pi := wait.NewProgressIndicator(fmt.Sprintf("Creating Helm Release for %s", release.ChartName))
		pi.SetTimeout(time.Duration(timeoutInMin) * time.Minute)
		pi.Start()

		_, err = havener.DeployHelmRelease(
			release.ChartName,
			release.ChartNamespace,
			release.ChartLocation,
			timeoutInMin,
			overridesData)

		if err != nil {
			pi.Stop()
			exitWithError("failed to deploy via havener configuration", err)
		}

		pi.Done("Successfully created new helm chart *%s* in namespace *_%s_*.", release.ChartName, release.ChartNamespace)
	}
}
