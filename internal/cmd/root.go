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
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.ibm.com/hatch/havener/pkg/havener"
)

// cfgFile holds the related configuration of havener
var cfgFile string

// kubeCfgFile is the path to the KUBECONFIG yaml
var kubeCfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "havener",
	Short: "Proof of concept tool to help you manage Kubernetes releases",
	Long: `A proof of concept tool to verify if a Kubernetes releases
helper tool can be done in Golang
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (Mandatory argument)")

	// Add kubeconfig persistent flag with default value
	rootCmd.PersistentFlags().StringVar(&kubeCfgFile, "kubeconfig", "", "kubeconfig file (default is $HOME/.kube/config)")

	// Bind kubeconfig flag with viper, so that the contents can be accessible later
	viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindPFlag("havenerconfig", rootCmd.PersistentFlags().Lookup("config"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

	if cfgFile == "" && viper.GetString("havenerconfig") == "" {
		havener.ExitWithError("please provide configuration via --config or environment variable HAVENERCONFIG", fmt.Errorf("no havener configuration file set"))
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// If the kube config path flag is empty, set it to default
	if kubeCfgFile == "" {
		home, err := homedir.Dir()
		if err != nil {
			havener.ExitWithError("unable to get home directory", err)
		}
		viper.SetDefault("kubeconfig", home+"/.kube/config")
	}
}
