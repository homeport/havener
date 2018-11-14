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

	"github.com/caarlos0/spin"
	"github.com/homeport/havener/pkg/havener"

	yaml "gopkg.in/yaml.v2"
)

// cfgFile holds the related configuration of havener
var cfgFile string

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Helm Charts to Kubernetes",
	Long:  `TODO please do this later`,
	Run: func(cmd *cobra.Command, args []string) {
		if cfgFile == "" && viper.GetString("havenerconfig") == "" {
			havener.ExitWithError("please provide configuration via --config or environment variable HAVENERCONFIG", fmt.Errorf("no havener configuration file set"))
		}

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}

		cfgdata, err := ioutil.ReadFile(viper.GetString("havenerconfig"))
		if err != nil {
			havener.ExitWithError("unable to read file", err)
		}

		var config havener.Config
		if err := yaml.Unmarshal(cfgdata, &config); err != nil {
			havener.ExitWithError("failed to unmarshal config file", err)
		}

		for _, release := range config.Releases {
			overrides, err := havener.TraverseStructureAndProcessShellOperators(release.Overrides)
			if err != nil {
				havener.ExitWithError("failed to process overrides section", err)
			}

			overridesData, err := yaml.Marshal(overrides)
			if err != nil {
				havener.ExitWithError("failed to marshal overrides structure into bytes", err)
			}

			// Show a wait indicator ...
			s := spin.New("%s " + fmt.Sprintf("Creating Helm Release for %s", release.ChartName))
			s.Start()
			if _, err := havener.DeployHelmRelease(release.ChartName, release.ChartNamespace, release.ChartLocation, overridesData); err != nil {
				havener.ExitWithError("Error deploying chart", err)
			}
			s.Stop()
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (Mandatory argument)")

	viper.AutomaticEnv() // read in environment variables that match

	// Bind kubeconfig flag with viper, so that the contents can be accessible later
	viper.BindPFlag("havenerconfig", deployCmd.PersistentFlags().Lookup("config"))
}
