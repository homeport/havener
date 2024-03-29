## havener node-exec

Execute command on Kubernetes node

### Synopsis

Execute a command on a node.

This executes a command directly on the node itself. Therefore, havener creates
a temporary pod which enables the user to access the shell of the node. The pod
is deleted automatically afterwards.

The command can be omitted which will result in the default command: /bin/sh. For
example 'havener node-exec foo' will search for a node named 'foo' and open a
shell if found.

Typically, the TTY flag does have to be specified. By definition, if one one
target node is provided, it is assumed that TTY is desired and STDIN is attached
to the remote process. Analog, for the distributed mode with multiple nodes,
no TTY is set, and the STDIN is multiplexed into each remote process.

If you run the 'node-exec' without any additional arguments, it will print a
list of available nodes.

For convenience, if the target node name all is used, havener will look up
all nodes automatically.



```
havener node-exec [flags] [<node>[,<node>,...]] [<command>]
```

### Options

```
      --block              show distributed shell output as block for each node
  -h, --help               help for node-exec
      --image string       set image for helper pod from which the root-shell is accessed (default "alpine")
      --max-parallel int   number of parallel executions (defaults to number of nodes)
      --no-tty             do not allocate pseudo-terminal for command execution
      --timeout int        set timout in seconds for the setup of the helper pod (default 10)
```

### Options inherited from parent commands

```
      --debug                 debug output - level 5
      --error                 error output - level 2
      --fatal                 fatal output - level 1
      --kubeconfig string     Kubernetes configuration file (default "~/.kube/config")
      --terminal-height int   disable autodetection and specify an explicit terminal height (default -1)
      --terminal-width int    disable autodetection and specify an explicit terminal width (default -1)
      --trace                 trace output - level 6
  -v, --verbose               verbose output - level 4
      --warn                  warn output - level 3
```

### SEE ALSO

* [havener](havener.md)	 - Convenience wrapper around some kubectl commands

