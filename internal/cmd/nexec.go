// Copyright © 2021 The Homeport Team
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
	"io"
	"os"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/term"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

const nodeDefaultCommand = "/bin/sh"

var (
	nodeExecNoTty       bool
	nodeExecImage       string
	nodeExecTimeout     int
	nodeExecBlock       bool
	defaultImage        = "alpine"
	defaultTimeout      = 10
	nodeExecMaxParallel = 10
)

// nodeExecCmd represents the node-exec command
var nodeExecCmd = &cobra.Command{
	Use:     "node-exec [flags] [<node>[,<node>,...]] [<command>]",
	Aliases: []string{"ne"},
	Short:   "Execute command on Kubernetes node",
	Long: bunt.Sprintf(`Execute a command on a node.

This executes a command directly on the node itself. Therefore, havener creates
a temporary pod which enables the user to access the shell of the node. The pod
is deleted automatically afterwards.

The command can be omitted which will result in the default command: _%s_. For
example 'havener node-exec foo' will search for a node named 'foo' and open a
shell if found.

Typically, the TTY flag does have to be specified. By definition, if one one
target node is provided, it is assumed that TTY is desired and STDIN is attached
to the remote process. Analog, for the distributed mode with multiple nodes,
no TTY is set, and the STDIN is multiplexed into each remote process.

If you run the 'node-exec' without any additional arguments, it will print a
list of available nodes.

For convenience, if the target node name _all_ is used, havener will look up
all nodes automatically.

`, nodeDefaultCommand),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		hvnr, err := havener.NewHavener(havener.WithContext(cmd.Context()), havener.WithKubeConfigPath(kubeConfig))
		if err != nil {
			return fmt.Errorf("unable to get access to cluster: %w", err)
		}

		return execInClusterNodes(hvnr, args)
	},
}

func init() {
	rootCmd.AddCommand(nodeExecCmd)

	nodeExecCmd.PersistentFlags().BoolVar(&nodeExecNoTty, "no-tty", false, "do not allocate pseudo-terminal for command execution")
	nodeExecCmd.PersistentFlags().StringVar(&nodeExecImage, "image", defaultImage, "set image for helper pod from which the root-shell is accessed")
	nodeExecCmd.PersistentFlags().IntVar(&nodeExecTimeout, "timeout", defaultTimeout, "set timout in seconds for the setup of the helper pod")
	nodeExecCmd.PersistentFlags().BoolVar(&nodeExecBlock, "block", false, "show distributed shell output as block for each node")
	nodeExecCmd.PersistentFlags().IntVar(&nodeExecMaxParallel, "max-parallel", 0, "number of parallel executions (defaults to number of nodes)")
}

func execInClusterNodes(hvnr havener.Havener, args []string) error {
	var (
		nodes   []corev1.Node
		input   string
		command []string
		err     error
	)

	switch {
	case len(args) >= 2: // node name and command is given
		input, command = args[0], args[1:]
		nodes, err = lookupNodesByName(hvnr, input)
		if err != nil {
			return err
		}

	case len(args) == 1: // only node name is given
		input, command = args[0], []string{nodeDefaultCommand}
		nodes, err = lookupNodesByName(hvnr, input)
		if err != nil {
			return err
		}

	default: // no arguments
		return availableNodesError(hvnr, "no node name and command specified")
	}

	// In case the current process does not run in a terminal, disable the
	// default TTY behavior.
	if !term.IsTerminal() {
		nodeExecNoTty = true
	}

	// Single node mode, use default streams and run node execute function
	if len(nodes) == 1 {
		return hvnr.NodeExec(
			nodes[0],
			nodeExecImage,
			nodeExecTimeout,
			command,
			os.Stdin,
			os.Stdout,
			os.Stderr,
			!nodeExecNoTty,
		)
	}

	// In distributed shell mode, TTY is forced to be disabled
	nodeExecNoTty = true

	// In case nothing is configured, all nodes will be executed concurrently
	if nodeExecMaxParallel <= 0 || nodeExecMaxParallel > len(nodes) {
		nodeExecMaxParallel = len(nodes)
	}

	type task struct {
		node   corev1.Node
		reader io.Reader
	}

	var (
		wg      = &sync.WaitGroup{}
		tasks   = make(chan task, len(nodes))
		readers = duplicateReader(os.Stdin, len(nodes))
		output  = make(chan OutputMsg)
		errors  = make(chan error, len(nodes))
		printer = make(chan bool, 1)
	)

	// Fill task queue with the list of nodes to be processed
	for i := range nodes {
		tasks <- task{reader: readers[i], node: nodes[i]}
	}
	close(tasks)

	// Start n task workers to work on task queue
	wg.Add(nodeExecMaxParallel)
	for i := 0; i < nodeExecMaxParallel; i++ {
		go func() {
			defer wg.Done()
			for task := range tasks {
				errors <- hvnr.NodeExec(
					task.node,
					nodeExecImage,
					nodeExecTimeout,
					command,
					task.reader,
					chanWriter("StdOut", task.node.Name, output),
					chanWriter("StdErr", task.node.Name, output),
					!nodeExecNoTty,
				)
			}
		}()
	}

	// Start the respective output printer in a separate Go routine
	go func() {
		if nodeExecBlock {
			PrintOutputMessageAsBlock(output)

		} else {
			PrintOutputMessage(output)
		}

		printer <- true
	}()

	wg.Wait()
	close(errors)
	close(output)

	<-printer
	return combineErrorsFromChannel("node command execution failed", errors)
}

func lookupNodesByName(h havener.Havener, input string) ([]corev1.Node, error) {
	if input == "all" {
		list, err := h.Client().CoreV1().Nodes().List(h.Context(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		return list.Items, nil
	}

	var nodeList []corev1.Node
	for _, nodeName := range strings.Split(input, ",") {
		node, err := h.Client().CoreV1().Nodes().Get(h.Context(), nodeName, metav1.GetOptions{})
		if err != nil {
			return nil, availableNodesError(h, "node '%s' does not exist", nodeName)
		}

		nodeList = append(nodeList, *node)
	}

	return nodeList, nil
}

func availableNodesError(h havener.Havener, title string, fArgs ...interface{}) error {
	nodes, err := h.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list all nodes in cluster: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("failed to find any node in cluster")
	}

	names := make([]string, len(nodes))
	for i, node := range nodes {
		names[i] = node.Name
	}

	return fmt.Errorf("%s: %w",
		fmt.Sprintf(title, fArgs...),
		bunt.Errorf("*list of available nodes:*\n%s\n\nor, use _all_ to target all nodes", strings.Join(names, "\n")),
	)
}
