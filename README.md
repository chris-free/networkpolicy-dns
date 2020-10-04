# NetworkPolicy-DNS
##### Poor mans DNS to NetworkPolicy controller for Kubernetes


## Rationale

The aim of this project is to restrict external access to a whitelist set of domains. At the moment Kubernetes doesn't have the functionality to do this natively. Though the NetworkPolicy resource allows the whitelisting of CIDR, it doesn't extend to DNS. 

## Design

This follows a very simple Kubernetes controller pattern. net-dns is deployed with RBAC rights to modify network policies and routinely checks for DNS changes on the whitelisted domains.

## Configuration

This requires a podSelector structure and a list of domains, set through the "netdns-settings" configmap.

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: netdns-settings
data:
  settings.yml: |
    podSelector:
      matchLabels:
        role: mysql-client
    domain:
      - "aws.com"
      - "chrisfreeman.uk"
    interval: 60
```

## Example 

Example resource manifest found in example-resources.yml

 
## Backlog

1. clean up run func, get list of policies check if in there, then do an if else on create/update
2. name in yml
3. use: https://github.com/kubernetes/apimachinery/blob/master/pkg/util/yaml/decoder.go for unmarshalling settings
4. create integration tests?
5. setup github actions to build docker images
6. push to docker etc
