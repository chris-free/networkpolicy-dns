# NetworkPolicy-DNS
##### Poor mans DNS NetworkPolicy controller for Kubernetes


## Rationale

Not all software has a perfect security record (*cough* Wordpress). Wouldn't it be great to know that your site can't reach any external services apart from the ones you whitelist? Kubernetes doesn't any native functionality to restrict outgoing access to a set of domains, only a CIDR range; this is where this controller comes in.

## Design

This is a very simple Kubernetes controller, net-dns is deployed with RBAC rights to modify NetworkPolicy's and periodically checks for DNS changes on the whitelisted domains and if necessary updates the NetworkPolicy. 

So simple it uses: ~16mb docker image ~1mb RAM usage

## Alternatives

This is the simplest solution to the problem without a doubt :100:, however you might need greater control and reassurances. This is likely the case if you want to audit the network data (what is your website even doing?), or if you want to modify the connections (e.g load balancing, why would you even be reading this! :confused:). If this is the case it is likely you will want something in-between a simple reverse proxy (e.g Tinyproxy or Squid) to a Service Mesh à la Istio. 

## Configuration

As you have probably figured, configuration is extremely simple. 

There are 3 values: 
- podselector (full k8s 1.18 spec) 
- domain list (you figured it out, the domains that are whitelisted)
- interval (period in seconds to check for DNS changes)

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: netdns-settings
data:
  settings.yml: |
    podSelector: # Kubernetes 1.18 spec
      matchLabels:
        role: mysql-client
    domain:
      - "aws.com"
      - "chrisfreeman.uk"
    interval: 60 # seconds
```

The default path for the configmap is "/configmap/settings.yml"

## Example 

Example resource manifest found in example-resources.yml

 
## TODO

1. Multiple generated networkpolicies
2. Use: https://github.com/kubernetes/apimachinery/blob/master/pkg/util/yaml/decoder.go for unmarshalling settings
3. Integration tests
