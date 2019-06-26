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
	"strings"

	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

var (
	nodeExecTty     bool
	nodeExecImage   string
	nodeExecTimeout int
	defaultImage    = "alpine"
	defaultTimeout  = 10
)

// nodeExecCmd represents the node-exec command
var nodeExecCmd = &cobra.Command{
	Use:     "node-exec [flags] <node> <command>",
	Aliases: []string{"ne"},
	Short:   "Execute command on Kubernetes node",
	Long: `Execute a shell command on a node.

This will create a job to get a pod with the right settings to execute a
command on the pod that is executed in the root namespace. Technically, this
is like running the command as you would run it on the node itself. The job
and respective pod will be deleted after the command was executed.

`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return execInClusterNode(args)
	},
}

func init() {
	rootCmd.AddCommand(nodeExecCmd)

	nodeExecCmd.PersistentFlags().BoolVar(&nodeExecTty, "tty", false, "allocate pseudo-terminal for command execution")
	nodeExecCmd.PersistentFlags().StringVar(&nodeExecImage, "image", defaultImage, "set image for helper pod from which the root-shell is accessed")
	nodeExecCmd.PersistentFlags().IntVar(&nodeExecTimeout, "timeout", defaultTimeout, "set timout in seconds for the setup of the helper pod")
}

func execInClusterNode(args []string) error {
	havener.VerboseMessage("Connecting to Kubernetes cluster...")
	client, restconfig, err := havener.OutOfClusterAuthentication("")
	if err != nil {
		return &ErrorWithMsg{"failed to connect to Kubernetes cluster", err}
	}

	switch {
	case len(args) >= 2: //node name and command is given
		nodeName, command := args[0], strings.Join(args[1:], " ")

		havener.VerboseMessage("Executing command on node...")
		if err := havener.NodeExec(client, restconfig, nodeName, nodeExecImage, nodeExecTimeout, command, os.Stdin, os.Stdout, os.Stderr, nodeExecTty); err != nil {
			return &ErrorWithMsg{"failed to execute command on node", err}
		}

		havener.VerboseMessage("Successfully executed command.")

	case len(args) == 1: //only node name is given
		return &ErrorWithMsg{"no command specified", fmt.Errorf(
			"Usage:\nnode-exec [flags] <node> <command>",
		)}

	default: //no arguments
		nodes, err := havener.ListNodes(client)
		if err != nil {
			return err
		}

		return &ErrorWithMsg{"no node name and command specified",
			fmt.Errorf(
				"Usage:\nnode-exec [flags] <node> <command>\n\nAvailable nodes:\n%s",
				strings.Join(nodes, "\n"),
			)}
	}

	return nil
}
