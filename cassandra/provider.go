package cassandra

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"time"

	"github.com/gocql/gocql"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	allowedTLSProtocols = map[string]uint16{
		"SSL3.0": tls.VersionSSL30,
		"TLS1.0": tls.VersionTLS10,
		"TLS1.1": tls.VersionTLS11,
		"TLS1.2": tls.VersionTLS12,
		"TLS1.3": tls.VersionTLS13,
	}
)

// Provider returns a terraform.ResourceProvider
func Provider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"cassandra_keyspace": resourceCassandraKeyspace(),
			"cassandra_role":     resourceCassandraRole(),
			"cassandra_grant":    resourceCassandraGrant(),
		},
		ConfigureContextFunc: configureProvider,
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CASSANDRA_USERNAME", ""),
				Description: "Cassandra username",
				Sensitive:   true,
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CASSANDRA_PASSWORD", ""),
				Description: "Cassandra password",
				Sensitive:   true,
			},
			"port": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CASSANDRA_PORT", 9042),
				Description: "Cassandra CQL Port",
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					port := i.(int)

					if port <= 0 || port >= 65535 {
						return diag.Diagnostics{
							{
								Severity:      diag.Error,
								Summary:       "Invalid port number",
								Detail:        fmt.Sprintf("%d: invalid value - must be between 1 and 65535", port),
								AttributePath: path,
							},
						}
					}

					return nil
				},
			},
			"host": &schema.Schema{
				Type:         schema.TypeString,
				DefaultFunc:  schema.EnvDefaultFunc("CASSANDRA_HOST", nil),
				Description:  "Cassandra host",
				Optional:     true,
				ExactlyOneOf: []string{"host", "hosts"},
			},
			"hosts": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				MinItems:    1,
				Optional:    true,
				Description: "Cassandra hosts",
			},
			"host_filter": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Filter all incoming events for host. Hosts have to existing before using this provider",
			},
			"connection_timeout": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1000,
				Description: "Connection timeout in milliseconds",
			},
			"root_ca": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Use root CA to connect to Cluster. Applies only when useSSL is enabled",
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					rootCA := i.(string)

					if rootCA == "" {
						return nil
					}

					caPool := x509.NewCertPool()
					ok := caPool.AppendCertsFromPEM([]byte(rootCA))

					if !ok {
						return diag.Diagnostics{
							{
								Severity:      diag.Error,
								Summary:       "Invalid PEM",
								Detail:        fmt.Sprintf("%s: invalid PEM", rootCA),
								AttributePath: path,
							},
						}
					}

					return nil
				},
			},
			"use_ssl": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Use SSL when connecting to cluster",
			},
			"min_tls_version": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "TLS1.2",
				Description: "Minimum TLS Version used to connect to the cluster - allowed values are SSL3.0, TLS1.0, TLS1.1, TLS1.2. Applies only when useSSL is enabled",
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					minTLSVersion := i.(string)

					if allowedTLSProtocols[minTLSVersion] == 0 {
						return diag.Diagnostics{
							{
								Severity:      diag.Error,
								Summary:       "Invalid TLS",
								Detail:        fmt.Sprintf("%s: invalid value - must be one of SSL3.0, TLS1.0, TLS1.1, TLS1.2", minTLSVersion),
								AttributePath: path,
							},
						}
					}

					return nil
				},
			},
			"protocol_version": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     4,
				Description: "CQL Binary Protocol Version",
			},
		},
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {

	log.Printf("Creating provider")

	useSSL := d.Get("use_ssl").(bool)
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	port := d.Get("port").(int)
	connectionTimeout := d.Get("connection_timeout").(int)
	protocolVersion := d.Get("protocol_version").(int)
	diags := diag.Diagnostics{}

	log.Printf("Using port %d", port)
	log.Printf("Using use_ssl %v", useSSL)
	log.Printf("Using username %s", username)

	var rawHosts []interface{}

	if rawHost, getHost := d.GetOk("host"); getHost == true {
		rawHosts = []interface{}{rawHost}
	} else {
		rawHosts = d.Get("hosts").([]interface{})
	}

	hosts := make([]string, len(rawHosts))
	hostFilter := d.Get("host_filter").(bool)

	for _, value := range rawHosts {
		hosts = append(hosts, value.(string))

		log.Printf("Using host %v", value.(string))
	}

	cluster := gocql.NewCluster()

	cluster.Hosts = hosts

	cluster.Port = port

	cluster.Authenticator = &gocql.PasswordAuthenticator{
		Username: username,
		Password: password,
	}

	cluster.ConnectTimeout = time.Millisecond * time.Duration(connectionTimeout)

	cluster.Timeout = time.Minute * time.Duration(1)

	cluster.CQLVersion = "3.0.0"

	cluster.Keyspace = "system"

	cluster.ProtoVersion = protocolVersion

	if hostFilter {
		cluster.HostFilter = gocql.WhiteListHostFilter(hosts...)
	}

	cluster.DisableInitialHostLookup = true

	if useSSL {

		rootCA := d.Get("root_ca").(string)
		minTLSVersion := d.Get("min_tls_version").(string)

		tlsConfig := &tls.Config{
			MinVersion: allowedTLSProtocols[minTLSVersion],
		}

		if rootCA != "" {
			caPool := x509.NewCertPool()
			ok := caPool.AppendCertsFromPEM([]byte(rootCA))

			if !ok {
				diags = append(diags, diag.Diagnostic{
					Severity:      diag.Error,
					Summary:       "Unable to load rootCA",
					AttributePath: cty.Path{cty.GetAttrStep{Name: "root_ca"}},
				})
				return nil, diags
			}

			tlsConfig.RootCAs = caPool
		}

		cluster.SslOpts = &gocql.SslOptions{
			Config: tlsConfig,
		}
	}

	return cluster, diags
}
