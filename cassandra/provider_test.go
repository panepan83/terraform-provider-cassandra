package cassandra

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"testing"
)

var (
	testAccProviders map[string]terraform.ResourceProvider
	testAccProvider  *schema.Provider
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"cassandra": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func TestProvider_configure(t *testing.T) {
	hosts := []interface{}{"asdf"}
	rc := terraform.NewResourceConfigRaw(map[string]interface{}{
		"username": "cassanrda",
		"password": "cassanrda",
		"port":     9042,
		"hosts":    hosts,
	})
	p := Provider()
	err := p.Configure(rc)
	if err != nil {
		t.Fatal(err)
	}
}
