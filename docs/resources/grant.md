# `grant` Resource

Grants permissions to a role.

## Example Usage

```hcl
resource "cassandra_grant" "all_access_to_keyspace" {
  privilege     = "all"
  resource_type = "keyspace"
  keyspace_name = "test"
  grantee       = "migration"
}
```

## Argument Reference

- `privilege` - Type of access we are granting against a resource

  One of either `all`, `create`, `alter`, `drop`, `select`, `modify`, `authorize`, `describe` and `execute`

  See official cassandra docs for more [information](https://docs.datastax.com/en/cql/3.3/cql/cql_reference/cqlGrant.html)

- `grantee` - The name of the cassandra role which we are granting privileges to

- `resource_type` -  Enables one to qualify/restrict the grant to a particular resource(s)

  This can take any of the following values

    - `all functions`
    - `all functions in keyspace`
    - `function`
    - `all keyspaces`
    - `keyspace`
    - `table`
    - `all roles`
    - `role`
    - `roles`
    - `mbean`
    - `mbeans`
    - `all mbeans`

  For more info please see official [docs](https://docs.datastax.com/en/cql/3.3/cql/cql_reference/cqlGrant.html).

- `keyspace_name` - Keyspace qualifier to the resource, only applicable when resource_type takes the following values

    - `all functions in keyspace`
    - `function`
    - `keyspace`
    - `table`

- `function_name` - Represents name of the function we are granting access to. Its only applicable when resource_type is function

- `table_name` - Represents name of the table we are granting access to. Its only applicable when resource_type is table

- `role_name` - Represents name of the role we are granting access to. Only applicable for resource_type is role

- `mbean_name` - Represents name of the mbean we are granting access to. Only applicable for resource_type is mbean

- `mbean_pattern` - Represents a pattern, which will grant access to all mbeans which satisfy this pattern. Only works when resource_type is mbeans
