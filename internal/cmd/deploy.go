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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/homeport/gonvenience/pkg/v1/wait"
	"gopkg.in/yaml.v2"

	"github.com/homeport/havener/pkg/havener"
)

// cfgFile holds the related configuration of havener
var cfgFile string

// maxTimeOut holds the timeout in minutes for a helm init
var maxTimeOut int

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy to Kubernetes",
	Long:  `Deploy to Kubernetes based on havener configuration`,
	Run: func(cmd *cobra.Command, args []string) {
		havener.VerboseMessage("Looking for config file...")

		if cfgFile == "" && viper.GetString("havenerconfig") == "" {
			exitWithError("please provide configuration via --config or environment variable HAVENERCONFIG", fmt.Errorf("no havener configuration file set"))
		}

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			havener.InfoMessage("Using config file: %s", viper.ConfigFileUsed())
		}

		havener.VerboseMessage("Reading config file...")

		cfgdata, err := ioutil.ReadFile(viper.GetString("havenerconfig"))
		if err != nil {
			exitWithError("unable to read file", err)
		}

		havener.VerboseMessage("Unmarshaling config file...")

		var config havener.Config
		if err := yaml.Unmarshal(cfgdata, &config); err != nil {
			exitWithError("failed to unmarshal config file", err)
		}

		havener.VerboseMessage("Creating helm chart(s)...")

		for _, release := range config.Releases {
			overrides, err := havener.TraverseStructureAndProcessShellOperators(release.Overrides)

			havener.VerboseMessage("Processing overrides section...")

			if err != nil {
				exitWithError("failed to process overrides section", err)
			}

			havener.VerboseMessage("Marshaling overrides section...")

			overridesData, err := yaml.Marshal(overrides)
			if err != nil {
				exitWithError("failed to marshal overrides structure into bytes", err)
			}

			// Show a wait indicator ...
			pi := wait.NewProgressIndicator(fmt.Sprintf("Creating Helm Release for %s", release.ChartName))
			pi.Start()

			if _, err := havener.DeployHelmRelease(release.ChartName, release.ChartNamespace, release.ChartLocation, maxTimeOut, overridesData); err != nil {
				pi.Done()
				exitWithError("Error deploying chart", err)
			}

			pi.Done(fmt.Sprintf("Successfully created new helm chart for %s in namespace %s.", release.ChartName, release.ChartNamespace))
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (Mandatory argument)")
	deployCmd.PersistentFlags().IntVar(&maxTimeOut, "timeout", 40, "install timeout in minutes")

	viper.AutomaticEnv() // read in environment variables that match

	// Bind kubeconfig flag with viper, so that the contents can be accessible later
	viper.BindPFlag("havenerconfig", deployCmd.PersistentFlags().Lookup("config"))
}
