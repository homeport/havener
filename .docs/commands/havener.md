## havener

Convenience wrapper around some kubectl commands

### Synopsis

Convenience wrapper around some kubectl commands.

Think of it as a swiss army knife for Kubernetes tasks. Possible use cases are
for example executing a command on multiple pods at the same time, or
retrieving usage details.

See the individual commands to get the complete overview.



### Options

```
      --kubeconfig string     Kubernetes configuration file (default "~/.kube/config")
      --terminal-width int    disable autodetection and specify an explicit terminal width (default -1)
      --terminal-height int   disable autodetection and specify an explicit terminal height (default -1)
      --fatal                 fatal output - level 1
      --error                 error output - level 2
      --warn                  warn output - level 3
  -v, --verbose               verbose output - level 4
      --debug                 debug output - level 5
      --trace                 trace output - level 6
  -h, --help                  help for havener
```

### SEE ALSO

* [havener events](havener_events.md)	 - Show Kubernetes cluster events
* [havener logs](havener_logs.md)	 - Retrieve log files from all pods
* [havener node-exec](havener_node-exec.md)	 - Execute command on Kubernetes node
* [havener pod-exec](havener_pod-exec.md)	 - Execute command on Kubernetes pod
* [havener top](havener_top.md)	 - Shows CPU and Memory usage
* [havener version](havener_version.md)	 - Shows the version
* [havener watch](havener_watch.md)	 - Watch status of all pods in all namespaces

