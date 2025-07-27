# kubectl-mutated

Show what resources have been mutated by a field manager that might be operated manually, like kubectl

Some GitOps or CD solutions ensure only fields specified in definitions are in sync, by the nature of "apply" operation. If you manually edit other fields that are absent in the golden definitions while debugging, they will stick and it is easy to forget to write them back into the definitions.

Operator pattern products may have a similar problems too. In Kubernetes, it is common to use "apply" to manage only the fields of interest, and make it possible for other entities, including humans, to manage other fields.

This tool helps to find what fields of resources are managed by human, to make sure that resources are fully managed by machine, keeping them deterministic and reproducible.

There are other ways to prevent manual editing of resources. `kustomize-controller` of Flux v2 [revokes kubectl managed fields ownership](https://github.com/fluxcd/kustomize-controller/pull/527) to undo changes from kubectl, even on fields not specified on GitOps. The managers-to-revoke lists can be further extended via `--override-manager`.

On identifying manually managed fields, this tool is still a more comprehensive solution. It inspects all resources, not just the ones directly managed by some controllers.

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

- What if I do want some resources, such as secrets, to be managed by hand?

Use a different field manager like above, or set a label like `managed-by: hand` and run `kubectl mutated` with `--selector 'managed-by!=hand'`.
