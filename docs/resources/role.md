# <resource name> Resource/Data Source

Creates a role.

## Example Usage

```hcl
resource "cassandra_role" "role" {
  name     = "app_user"
  password = "sup3rS3cr3tPa$$w0rd123343434345454545454"
}
```

## Argument Reference

- `name` - Name of the role. Must contain between 1 and 256 characters.

- `super_user` - Allow the role to create and manage other roles. It is __false__ by default.

- `login` - Enables the role to be able to login. It defaults to __true__.

- `password` - Password for user when using cassandra internal authentication.
  It is required. It has the restriction of being between 40 and 512 characters.
