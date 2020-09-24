package cassandra

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	keyspaceLiteralPattern = `^[a-zA-Z0-9][a-zA-Z0-9_]{0,48}$`
	strategyLiteralPatten  = `^SimpleStrategy|NetworkTopologyStrategy$`
)

var (
	keyspaceRegex, _ = regexp.Compile(keyspaceLiteralPattern)
	strategyRegex, _ = regexp.Compile(strategyLiteralPatten)
	boolToAction     = map[bool]string{
		true:  "CREATE",
		false: "ALTER",
	}
)

func resourceCassandraKeyspace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKeyspaceCreate,
		ReadContext:   resourceKeyspaceRead,
		UpdateContext: resourceKeyspaceUpdate,
		DeleteContext: resourceKeyspaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of keyspace",
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					name := i.(string)

					if !keyspaceRegex.MatchString(name) {
						return diag.Diagnostics{
							{
								Severity:      diag.Error,
								Summary:       "Invalid keyspace",
								Detail:        fmt.Sprintf("%s: invalid keyspace name - must match %s", name, keyspaceLiteralPattern),
								AttributePath: path,
							},
						}
					}

					if name == "system" {
						return diag.Diagnostics{
							{
								Severity:      diag.Error,
								Summary:       "Cannot manage system keyspace",
								Detail:        fmt.Sprintf("cannot manage system keyspace, it is internal to Cassandra"),
								AttributePath: path,
							},
						}
					}

					return nil
				},
			},
			"replication_strategy": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
				Description: "Keyspace replication strategy - must be one of SimpleStrategy or NetworkTopologyStrategy",
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					strategy := i.(string)

					if !strategyRegex.MatchString(strategy) {
						return diag.Diagnostics{
							{
								Severity:      diag.Error,
								Summary:       "Invalid replication strategy",
								Detail:        fmt.Sprintf("%s: invalid replication strategy - must match %s", strategy, strategyLiteralPatten),
								AttributePath: path,
							},
						}
					}

					return nil
				},
			},
			"strategy_options": &schema.Schema{
				Type:        schema.TypeMap,
				Required:    true,
				ForceNew:    false,
				Description: "strategy options used with replication strategy",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				StateFunc: func(v interface{}) string {
					strategyOptions := v.(map[string]interface{})

					keys := make([]string, len(strategyOptions))

					for key, value := range strategyOptions {

						strValue := value.(string)

						keys = append(keys, fmt.Sprintf("%q=%q", key, strValue))
					}

					sort.Strings(keys)

					return hash(strings.Join(keys, ", "))
				},
			},
			"durable_writes": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    false,
				Description: "Enable or disable durable writes - disabling is not recommended",
				Default:     true,
			},
		},
	}
}

func generateCreateOrUpdateKeyspaceQueryString(name string, create bool, replicationStrategy string, strategyOptions map[string]interface{}, durableWrites bool) (string, error) {

	numberOfStrategyOptions := len(strategyOptions)

	if numberOfStrategyOptions == 0 {
		return "", fmt.Errorf("must specify stratgey options - see https://docs.datastax.com/en/cql/3.3/cql/cql_reference/cqlCreateKeyspace.html")
	}

	query := fmt.Sprintf(`%s KEYSPACE %s WITH REPLICATION = { 'class' : '%s'`, boolToAction[create], name, replicationStrategy)

	for key, value := range strategyOptions {
		query += fmt.Sprintf(`, '%s' : '%s'`, key, value.(string))
	}

	query += fmt.Sprintf(` } AND DURABLE_WRITES = %t`, durableWrites)

	log.Println("query", query)

	return query, nil
}

func resourceKeyspaceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	replicationStrategy := d.Get("replication_strategy").(string)
	strategyOptions := d.Get("strategy_options").(map[string]interface{})
	durableWrites := d.Get("durable_writes").(bool)

	query, err := generateCreateOrUpdateKeyspaceQueryString(name, true, replicationStrategy, strategyOptions, durableWrites)

	if err != nil {
		return diag.FromErr(err)
	}

	cluster := meta.(*gocql.ClusterConfig)

	start := time.Now()

	session, sessionCreateError := cluster.CreateSession()

	elapsed := time.Since(start)

	log.Printf("Getting a session took %s", elapsed)

	if sessionCreateError != nil {
		return diag.FromErr(sessionCreateError)
	}

	defer session.Close()

	err = session.Query(query).Exec()

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(name)

	return resourceKeyspaceRead(ctx, d, meta)
}

func resourceKeyspaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Id()

	cluster := meta.(*gocql.ClusterConfig)

	start := time.Now()

	session, sessionCreateError := cluster.CreateSession()

	elapsed := time.Since(start)

	log.Printf("Getting a session took %s", elapsed)

	if sessionCreateError != nil {
		return diag.FromErr(sessionCreateError)
	}

	defer session.Close()

	keyspaceMetadata, err := session.KeyspaceMetadata(name)

	if err == gocql.ErrKeyspaceDoesNotExist {
		d.SetId("")
		return nil
	} else if err != nil {
		return diag.FromErr(err)
	}

	strategyOptions := make(map[string]string)

	for key, value := range keyspaceMetadata.StrategyOptions {
		strategyOptions[key] = value.(string)
	}

	strategyClass := strings.TrimPrefix(keyspaceMetadata.StrategyClass, "org.apache.cassandra.locator.")

	d.Set("name", name)
	d.Set("replication_strategy", strategyClass)
	d.Set("durable_writes", keyspaceMetadata.DurableWrites)
	d.Set("strategy_options", strategyOptions)

	return nil
}

func resourceKeyspaceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("name").(string)

	cluster := meta.(*gocql.ClusterConfig)

	start := time.Now()

	session, sessionCreateError := cluster.CreateSession()

	elapsed := time.Since(start)

	log.Printf("Getting a session took %s", elapsed)

	if sessionCreateError != nil {
		return diag.FromErr(sessionCreateError)
	}

	defer session.Close()

	err := session.Query(fmt.Sprintf(`DROP KEYSPACE %s`, name)).Exec()
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKeyspaceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	replicationStrategy := d.Get("replication_strategy").(string)
	strategyOptions := d.Get("strategy_options").(map[string]interface{})
	durableWrites := d.Get("durable_writes").(bool)

	query, err := generateCreateOrUpdateKeyspaceQueryString(name, false, replicationStrategy, strategyOptions, durableWrites)

	if err != nil {
		return diag.FromErr(err)
	}

	cluster := meta.(*gocql.ClusterConfig)

	start := time.Now()

	session, sessionCreateError := cluster.CreateSession()

	elapsed := time.Since(start)

	log.Printf("Getting a session took %s", elapsed)

	if sessionCreateError != nil {
		return diag.FromErr(sessionCreateError)
	}

	defer session.Close()

	err = session.Query(query).Exec()

	if err != nil {
		return diag.FromErr(err)
	}

	return resourceKeyspaceRead(ctx, d, meta)
}
