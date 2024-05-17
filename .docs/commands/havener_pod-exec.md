## havener pod-exec

Execute command on Kubernetes pod

### Synopsis

Execute a command on a pod

This is similar to the kubectl exec command with just a slightly different
syntax. In contrast to kubectl, you do not have to specify the namespace
of the pod.

If no namespace is given, havener will search all namespaces for a pod that
matches the name.

Also, you can omit the command which will result in the default command: /bin/sh.
For example havener pod-exec api-0 will search for a pod named api-0 in all
namespaces and open a shell if found.

In case no container name is given, havener will assume you want to execute the
command in the first container found in the pod.

If you run the 'pod-exec' without any additional arguments, it will print a
list of available pods.

For convenience, if the target pod name all is used, havener will look up
all pods in all namespaces automatically.


```
havener pod-exec [flags] [[<namespace>/]<pod>[/container]] [<command>]
```

### Options

```
  -i, --stdin   Pass stdin to the container
  -t, --tty     Stdin is a TTY
      --block   show distributed shell output as block for each pod
  -h, --help    help for pod-exec
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

