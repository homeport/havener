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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"

	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

const nodeDefaultCommand = "/bin/sh"

var (
	nodeExecTty     bool
	nodeExecImage   string
	nodeExecTimeout int
	defaultImage    = "alpine"
	defaultTimeout  = 10
)

// nodeExecCmd represents the node-exec command
var nodeExecCmd = &cobra.Command{
	Use:     "node-exec [flags] [<node>] [<command>]",
	Aliases: []string{"ne"},
	Short:   "Execute command on Kubernetes node",
	Long: `Execute a shell command on a node.

This executes a command directly on the node itself. Therefore, havener
creates a temporary pod which enables the user to access the shell
of the node. The pod is deleted automatically afterwards.

The command can be omitted which will result in the default command: ` + nodeDefaultCommand + `. For example 
'havener node-exec api-0' will search for a node named 'api-0' and open a shell if found.

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
	client, restconfig, err := havener.OutOfClusterAuthentication("")
	if err != nil {
		return &ErrorWithMsg{"failed to connect to Kubernetes cluster", err}
	}

	switch {
	case len(args) >= 2: //node name and command is given
		nodeName, command := args[0], strings.Join(args[1:], " ")
		node, err := lookupNodeByName(client, nodeName)
		if err != nil {
			return err
		}

		if err := havener.NodeExec(client, restconfig, node, nodeExecImage, nodeExecTimeout, command, os.Stdin, os.Stdout, os.Stderr, nodeExecTty); err != nil {
			return &ErrorWithMsg{"failed to execute command on node", err}
		}

	case len(args) == 1: //only node name is given
		nodeName, command := args[0], nodeDefaultCommand
		node, err := lookupNodeByName(client, nodeName)
		if err != nil {
			return err
		}

		if err := havener.NodeExec(client, restconfig, node, nodeExecImage, nodeExecTimeout, command, os.Stdin, os.Stdout, os.Stderr, nodeExecTty); err != nil {
			return &ErrorWithMsg{"failed to execute command on node", err}
		}

	default: //no arguments
		return availableNodesError(client, "no node name and command specified")
	}

	return nil
}

func lookupNodeByName(client kubernetes.Interface, input string) (*corev1.Node, error) {
	if node, err := client.CoreV1().Nodes().Get(input, metav1.GetOptions{}); err == nil {
		return node, nil
	}

	return nil, availableNodesError(client, "invalid node name specfied")
}

func availableNodesError(client kubernetes.Interface, title string) error {
	nodes, err := havener.ListNodes(client)
	if err != nil {
		return &ErrorWithMsg{"failed to list all nodes in cluster", err}
	}
	nodeList := []string{}
	for _, nodeName := range nodes {
		nodeList = append(nodeList, nodeName)
	}

	return &ErrorWithMsg{title,
		fmt.Errorf("> Usage:\nnode-exec [flags] <node> <command>\n> List of available nodes:\n%s",
			strings.Join(nodeList, "\n"),
		)}
}
