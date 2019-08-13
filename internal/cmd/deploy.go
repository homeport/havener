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
	envVarDeployTimeout = "deployment_timeout"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:           "deploy [havener config file]",
	Short:         "Deploy to Kubernetes",
	Long:          `Deploy to Kubernetes based on havener configuration`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case len(args) == 1:
			return DeployViaHavenerConfig(args[0])
		default:
			cmd.Usage()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.PersistentFlags().Int("timeout", 40, "deployment timeout in minutes")

	viper.AutomaticEnv()
	viper.BindPFlag(envVarDeployTimeout, deployCmd.PersistentFlags().Lookup("timeout"))
}

// DeployViaHavenerConfig entry function for running a helm install
func DeployViaHavenerConfig(havenerConfig string) error {
	timeoutInMin := viper.GetInt(envVarDeployTimeout)

	config, err := havener.ParseConfigFile(havenerConfig)
	if err != nil {
		return err
	}

	err = havener.SetConfigEnv(config)
	if err != nil {
		return err
	}

	if err := processTask("Predeployment Steps", config.Before); err != nil {
		return wrap.Error(err, "failed to evaluate predeployment steps")
	}

	for _, release := range config.Releases {
		overridesData, err := processOverrideSection(release)
		if err != nil {
			return err
		}

		if err := processTask("Before Chart "+release.ChartName, release.Before); err != nil {
			return wrap.Error(err, "failed to evaluate before release steps")
		}

		pi := wait.NewProgressIndicator(fmt.Sprintf("Creating Helm Release for %s", release.ChartName))
		pi.SetTimeout(time.Duration(timeoutInMin) * time.Minute)
		setCurrentProgressIndicator(pi)
		pi.Start()
		err = havener.DeployHelmRelease(
			release.ChartName,
			release.ChartNamespace,
			release.ChartLocation,
			timeoutInMin,
			overridesData)
		pi.Stop()
		setCurrentProgressIndicator(nil)

		if err != nil {
			return wrap.Error(err, "failed to deploy via havener configuration")
		}

		message := bunt.Sprintf("Successfully created new helm chart *%s* in namespace *_%s_*.",
			release.ChartName,
			release.ChartNamespace,
		)

		message, err = getReleaseMessage(release, message)
		if err != nil {
			return err
		}

		printStatusMessage("Upgrade", message, bunt.Gray)

		if err := processTask("After Chart "+release.ChartName, release.After); err != nil {
			return wrap.Error(err, "failed to evaluate after release steps")
		}
	}

	if err := processTask("Postdeployment Steps", config.After); err != nil {
		return wrap.Error(err, "failed to evaluate postdeployment steps")
	}
	return nil
}
