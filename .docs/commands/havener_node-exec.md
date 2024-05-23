## havener node-exec

Execute command on Kubernetes node

### Synopsis

Execute a command on a node

Execute a command directly on the node itself. For this, havener creates a
temporary pod, which enables the user to access the shell of the node. The pod
is deleted automatically afterwards.

The command can be omitted which will result in the default command: /bin/sh. For
example havener node-exec foo will search for a node named 'foo' and open a
shell if the node can be found.

When more than one node is specified, it will execute the command on all nodes.
In this distributed mode, both passing the StdIn as well as TTY mode are not
available. By default, the number of parallel node executions is limited to 5
in parallel in order to not create to many requests at the same time. This
value can be overwritten. Handle with care.

If you run the node-exec without any additional arguments, it will print a
list of available nodes in the cluster.

For convenience, if the target node name all is used, havener will look up
all nodes automatically.



```
havener node-exec [flags] [<node>[,<node>,...]] [<command>]
```

### Options

```
  -i, --stdin              Pass stdin to the container
  -t, --tty                Stdin is a TTY
      --image string       Container image used for helper pod (from which the root-shell is accessed) (default "docker.io/library/alpine")
      --timeout duration   Timeout for the setup of the helper pod (default 30s)
      --max-parallel int   Number of parallel executions (value less or equal than zero means unlimited) (default 5)
      --block              Show distributed shell output as block for each node
  -h, --help               help for node-exec
```

### Options inherited from parent commands

```
      --debug                 debug output - level 5
      --error                 error output - level 2
      --fatal                 fatal output - level 1
      --kubeconfig string     Kubernetes configuration (default "~/.kube/config")
      --terminal-height int   disable autodetection and specify an explicit terminal height (default -1)
      --terminal-width int    disable autodetection and specify an explicit terminal width (default -1)
      --trace                 trace output - level 6
  -v, --verbose               verbose output - level 4
      --warn                  warn output - level 3
```

### SEE ALSO

* [havener](havener.md)	 - Convenience wrapper around some kubectl commands

