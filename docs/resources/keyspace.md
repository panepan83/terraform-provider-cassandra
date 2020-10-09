# cassandra_keyspace

Creates a keyspace.

## Example Usage

```hcl
locals {
  stategy_options = {
    replication_factor = 1
  }
}

resource "cassandra_keyspace" "keyspace" {
  name                 = "some_keyspace_name"
  replication_strategy = "SimpleStrategy"
  strategy_options     = local.strategy_options
}
```

## Argument Reference

- `name` - Name of the keyspace, must be between 1 and 48 characters.

- `replication_strategy` - Name of the replication strategy, only the built in replication strategies are supported. That is either __SimpleStrategy__ or __NetworkTopologyStrategy__.

- `strategy_options` - A map containing any extra options that are required by the selected replication strategy.

  For simple strategy, **replication_factor** must be passed. While for network topology strategy must contain keys which corresspond to the data center names and values which match their desired replication factor.

- `durable_writes` - Enables or disables durable writes. The default value is __true__. It is not reccomend to turn this off.
