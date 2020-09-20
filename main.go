package main

import (
	"github.com/bartoszj/terraform-provider-cassandra/cassandra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{ProviderFunc: cassandra.Provider})
}
