# PermissionSet

A permission set represents a Kubernetes RBAC `Role` or `ClusterRole`, which contain rules that represent a set of permissions that has been bound to an identity via a `RoleBinding` or `ClusterRoleBinding`. Permissions are purely additive (there are no "deny" rules).

## Properties

| Property            | Type      | Description |
| ----------------| --------- |----------------------------------------|
| name | `string` |  Name of the underlying role in Kubernetes |  
| rules | `[]string` |  List of strings representing the access granted by the role (see generator function [flattenPolicyRules](../../pkg/kubehound/models/converter/graph.go))|  

## Common Properties

+ [storeID](./COMMON.md#store-information)
+ [app](./COMMON.md#ownership-information)
+ [team](./COMMON.md#ownership-information)
+ [service](./COMMON.md#ownership-information)
+ [critical](./COMMON.md#risk-information)
+ [namespace](./COMMON.md#namespace-information)
+ [isNamespaced](./COMMON.md#namespace-information)

## Definition

[vertex.PermissionSet](../../pkg/kubehound/models/graph/permission_set.go)

## References

+ [Official Kubernetes documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) 
