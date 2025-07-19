# kubectl-mutated

Show what resources have been mutated by a field manager that might be operated manually, like kubectl

Most GitOps or CD solutions ensure only fields specified in definitions are in sync, by the nature of "apply" operation. If you manually edit other fields that are absent in the golden definitions while debugging, they will stay and it is easy to forget to write them back into the definitions. This tool helps to find affected resources under this scenario.

## Usage

```sh
kubectl mutated [flags]
```

## Examples

```sh
# List such resources under current namespace
kubectl mutated

# List such resources under namespace "my-space"
kubectl mutated -n my-space

# List such resources of all types under any namespaces, including cluster-scoped resources
kubectl mutated --all-namespaces

# Output in YAML highlighting such fields
kubectl mutated -o hyaml

# Output in YAML filtered to such fields
kubectl mutated -o fyaml
```

## FAQs

- What if my CD scripts also use `kubectl`?

Set `--field-manager` of `kubectl` to something else in your scripts.
