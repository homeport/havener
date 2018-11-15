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

	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

// reuseValues holds the bool of --reuse-values
var reuseValues bool

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Kubernetes with new Helm Charts",
	Long:  `TODO please do this later`,
	Run: func(cmd *cobra.Command, args []string) {
		havener.VerboseMessage(verbose, "Looking for config file...")

		if cfgFile == "" && viper.GetString("havenerconfig") == "" {
			havener.ExitWithError("please provide configuration via --config or environment variable HAVENERCONFIG", fmt.Errorf("no havener configuration file set"))
		}

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			havener.InfoMessage(fmt.Sprintf("Using config file: %s", viper.ConfigFileUsed()))
		}

		havener.VerboseMessage(verbose, "Reading config file...")

		cfgdata, err := ioutil.ReadFile(viper.GetString("havenerconfig"))
		if err != nil {
			havener.ExitWithError("unable to read file", err)
		}

		havener.VerboseMessage(verbose, "Unmarshaling config file...")

		var config havener.Config
		if err := yaml.Unmarshal(cfgdata, &config); err != nil {
			havener.ExitWithError("failed to unmarshal config file", err)
		}

		for _, release := range config.Releases {
			overrides, err := havener.TraverseStructureAndProcessShellOperators(release.Overrides)

			havener.VerboseMessage(verbose, "Processing overrides section...")

			if err != nil {
				havener.ExitWithError("failed to process overrides section", err)
			}

			havener.VerboseMessage(verbose, "Marshaling overrides section...")

			overridesData, err := yaml.Marshal(overrides)
			if err != nil {
				havener.ExitWithError("failed to marshal overrides structure into bytes", err)
			}

			havener.InfoMessage(fmt.Sprintf("Going to upgrade existing %s chart...", release.ChartName))

			if _, err := havener.UpdateHelmRelease(release.ChartName, release.ChartLocation, overridesData, reuseValues); err != nil {
				havener.ExitWithError("Error updating chart", err)
			}

			havener.InfoMessage(fmt.Sprintf("Successfully upgraded existing %s chart.", release.ChartName))
		}

	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	upgradeCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (Mandatory argument)")
	upgradeCmd.PersistentFlags().BoolVar(&reuseValues, "reuse-values", false, "reuse the last release's values and merge in any overrides")

	viper.AutomaticEnv() // read in environment variables that match

	// Bind kubeconfig flag with viper, so that the contents can be accessible later
	viper.BindPFlag("havenerconfig", upgradeCmd.PersistentFlags().Lookup("config"))

}
