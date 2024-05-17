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
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/bunt"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

const (
	nodeExecDefaultImage       = "docker.io/library/alpine"
	nodeExecDefaultTimeout     = 30 * time.Second
	nodeExecDefaultMaxParallel = 5
	nodeExecDefaultCommand     = "/bin/sh"
)

var nodeExecCmdSettings struct {
	stdin        bool
	tty          bool
	notty        bool
	image        string
	maxParallel  int
	timeout      time.Duration
	printAsBlock bool
}

// nodeExecCmd represents the node-exec command
var nodeExecCmd = &cobra.Command{
	Use:     "node-exec [flags] [<node>[,<node>,...]] [<command>]",
	Aliases: []string{"ne"},
	Short:   "Execute command on Kubernetes node",
	Long: bunt.Sprintf(`*Execute a command on a node*

Execute a command directly on the node itself. For this, *havener* creates a
temporary pod, which enables the user to access the shell of the node. The pod
is deleted automatically afterwards.

The command can be omitted which will result in the default command: _%s_. For
example _havener node-exec foo_ will search for a node named 'foo' and open a
shell if the node can be found.

When more than one node is specified, it will execute the command on all nodes.
In this distributed mode, both passing the StdIn as well as TTY mode are not
available. By default, the number of parallel node executions is limited to %d
in parallel in order to not create to many requests at the same time. This
value can be overwritten. Handle with care.

If you run the _node-exec_ without any additional arguments, it will print a
list of available nodes in the cluster.

For convenience, if the target node name _all_ is used, *havener* will look up
all nodes automatically.

`, nodeExecDefaultCommand, nodeExecDefaultMaxParallel),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check edge case for deprecated command-line flag
		if cmd.Flags().Changed("no-tty") {
			// Bail out if both the new and the old flag are used at the same time
			if cmd.Flags().Changed("tty") {
				return fmt.Errorf("cannot use --no-tty and --tty at the same time")
			}

			// If only --no-tty is used, continue to accept its input
			nodeExecCmdSettings.tty = !nodeExecCmdSettings.notty
		}

		hvnr, err := havener.NewHavener(havener.WithContext(cmd.Context()), havener.WithKubeConfigPath(kubeConfig))
		if err != nil {
			return fmt.Errorf("unable to get access to cluster: %w", err)
		}

		return execInClusterNodes(hvnr, args)
	},
}

func init() {
	rootCmd.AddCommand(nodeExecCmd)

	nodeExecCmd.Flags().SortFlags = false
	nodeExecCmd.Flags().BoolVarP(&nodeExecCmdSettings.stdin, "stdin", "i", false, "Pass stdin to the container")
	nodeExecCmd.Flags().BoolVarP(&nodeExecCmdSettings.tty, "tty", "t", false, "Stdin is a TTY")
	nodeExecCmd.Flags().StringVar(&nodeExecCmdSettings.image, "image", nodeExecDefaultImage, "Container image used for helper pod (from which the root-shell is accessed)")
	nodeExecCmd.Flags().DurationVar(&nodeExecCmdSettings.timeout, "timeout", nodeExecDefaultTimeout, "Timeout for the setup of the helper pod")
	nodeExecCmd.Flags().IntVar(&nodeExecCmdSettings.maxParallel, "max-parallel", nodeExecDefaultMaxParallel, "Number of parallel executions (value less or equal than zero means unlimited)")
	nodeExecCmd.Flags().BoolVar(&nodeExecCmdSettings.printAsBlock, "block", false, "Show distributed shell output as block for each node")

	// Deprecated/old flags
	nodeExecCmd.Flags().BoolVar(&nodeExecCmdSettings.notty, "no-tty", false, "do not allocate pseudo-terminal for command execution")
	_ = nodeExecCmd.Flags().MarkDeprecated("no-tty", "use --tty flag instead")
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
		input, command = args[0], []string{nodeExecDefaultCommand}
		nodes, err = lookupNodesByName(hvnr, input)
		if err != nil {
			return err
		}

	default: // no arguments
		return availableNodesError(hvnr, "no node name and command specified")
	}

	if !isStdinTerminal() {
		nodeExecCmdSettings.tty = false
	}

	var in io.Reader
	if nodeExecCmdSettings.stdin {
		in = os.Stdin
	}

	var nodeExecHelperPodConfig = havener.NodeExecHelperPodConfig{
		Annotations:    map[string]string{},
		ContainerImage: nodeExecCmdSettings.image,
		ContainerCmd:   []string{"/bin/sleep", "8h"},
		WaitTimeout:    nodeExecCmdSettings.timeout,
	}

	nodeExecHelperPodConfig.Annotations["originator"] = originator()

	// Single node mode, use default streams and run node execute function
	if len(nodes) == 1 {
		return hvnr.NodeExec(
			nodes[0],
			nodeExecHelperPodConfig,
			havener.ExecConfig{
				Command: command,
				Stdin:   in,
				Stdout:  os.Stdout,
				Stderr:  os.Stderr,
				TTY:     nodeExecCmdSettings.tty,
			},
		)
	}

	// In distributed shell mode, stdin is not piped through and TTY is forced to be disabled
	nodeExecCmdSettings.stdin = false
	nodeExecCmdSettings.tty = false

	// In case the user wants everything done in parallel, increase the max value
	if nodeExecCmdSettings.maxParallel <= 0 {
		nodeExecCmdSettings.maxParallel = len(nodes)
	}

	type task struct {
		node corev1.Node
	}

	var (
		wg      = &sync.WaitGroup{}
		tasks   = make(chan task, len(nodes))
		output  = make(chan OutputMsg)
		errors  = make(chan error, len(nodes))
		printer = make(chan bool, 1)
	)

	// Fill task queue with the list of nodes to be processed
	for i := range nodes {
		tasks <- task{node: nodes[i]}
	}
	close(tasks)

	// Start n task workers to work on task queue
	wg.Add(nodeExecCmdSettings.maxParallel)
	for i := 0; i < nodeExecCmdSettings.maxParallel; i++ {
		go func() {
			defer wg.Done()
			for task := range tasks {
				errors <- hvnr.NodeExec(
					task.node,
					nodeExecHelperPodConfig,
					havener.ExecConfig{
						Command: command,
						Stdin:   nil, // Disabled for now until reliable input duplication works
						Stdout:  chanWriter("StdOut", task.node.Name, output),
						Stderr:  chanWriter("StdErr", task.node.Name, output),
						TTY:     nodeExecCmdSettings.tty,
					},
				)
			}
		}()
	}

	// Start the respective output printer in a separate Go routine
	go func() {
		if nodeExecCmdSettings.printAsBlock {
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

func originator() string {
	if version == "" {
		version = "development version"
	}

	var username = "unknown"
	if user, err := user.Current(); err == nil {
		username = user.Name
	}

	var hostname = "unknown"
	if name, err := os.Hostname(); err == nil {
		hostname = name
	}

	return fmt.Sprintf("havener %s (%s/%s), %s@%s",
		version,
		runtime.GOOS, runtime.GOARCH,
		username, hostname,
	)
}

func lookupNodesByName(h havener.Havener, input string) ([]corev1.Node, error) {
	if input == "all" {
		return h.ListNodes()
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
