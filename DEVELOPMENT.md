Copy this to capi project:

tilt-settings.yaml
```
default_registry: "" # change if you use a remote image registry
provider_repos:
  # This refers to your provider directory and loads settings
  # from `tilt-provider.yaml`
  - ../cluster-api-provider-ionoscloud
enable_providers:
  - ionoscloud
```

ctlptl create cluster kind --name kind-capi-test --registry=ctlptl-registry

tilt up in capi project