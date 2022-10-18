## havener top

Shows CPU and Memory usage

### Synopsis

Shows more detailed CPU and Memory usage details

The top command shows Load, CPU, and Memory usage details for all nodes.

Based on the pod usage, aggregated usage details per namespace are generated
and displayed to show CPU and Memory usage.

Furthermore, the list of top pod consumers is displayed, both for the whole
cluster as well as a list per node.


```
havener top [flags]
```

### Options

```
  -c, --cycles int     number of cycles to run, negative numbers means infinite cycles (default -1)
  -i, --interval int   interval between measurements in seconds (default 4)
  -h, --help           help for top
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

