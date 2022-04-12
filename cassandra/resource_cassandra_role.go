package cassandra

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gocql/gocql"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/crypto/bcrypt"
)

func resourceCassandraRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Name of role - must contain between 1 and 256 characters",
				ValidateFunc: validation.StringLenBetween(1, 256),
			},
			"super_user": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    false,
				Description: "Allow role to create and manage other roles",
			},
			"login": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    false,
				Description: "Enables role to be able to login",
			},
			"password": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     false,
				Description:  "Password for user when using Cassandra internal authentication",
				Sensitive:    true,
				ValidateFunc: validation.StringLenBetween(20, 512),
			},
		},
	}
}

func readRole(session *gocql.Session, name string) (string, bool, bool, string, error) {

	var (
		role        string
		canLogin    bool
		isSuperUser bool
		saltedHash  string
	)

	iter := session.Query(`select role, can_login, is_superuser, salted_hash from system_auth.roles where role = ?`, name).Iter()

	defer iter.Close()

	log.Printf("read role query returned %d", iter.NumRows())

	for iter.Scan(&role, &canLogin, &isSuperUser, &saltedHash) {
		return role, canLogin, isSuperUser, saltedHash, nil
	}

	return "", false, false, "", fmt.Errorf("cannot read role with name %s", name)
}

func resourceRoleCreateOrUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}, createRole bool) diag.Diagnostics {
	name := d.Get("name").(string)
	superUser := d.Get("super_user").(bool)
	login := d.Get("login").(bool)
	password := d.Get("password").(string)
	var diags diag.Diagnostics

	cluster := meta.(*gocql.ClusterConfig)
	start := time.Now()
	session, sessionCreateError := cluster.CreateSession()
	elapsed := time.Since(start)

	log.Printf("Getting a session took %s", elapsed)

	if sessionCreateError != nil {
		return diag.FromErr(sessionCreateError)
	}

	defer session.Close()

	createErr := session.Query(fmt.Sprintf(`%s ROLE '%s' WITH PASSWORD = '%s' AND LOGIN = %v AND SUPERUSER = %v`, boolToAction[createRole], name, password, login, superUser)).Exec()
	if createErr != nil {
		return diag.FromErr(createErr)
	}

	d.SetId(name)
	d.Set("name", name)
	d.Set("super_user", superUser)
	d.Set("login", login)
	d.Set("password", password)

	diags = append(diags, resourceRoleRead(ctx, d, meta)...)

	return diags
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceRoleCreateOrUpdate(ctx, d, meta, true)
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Id()
	password := d.Get("password").(string)
	var diags diag.Diagnostics

	cluster := meta.(*gocql.ClusterConfig)
	start := time.Now()
	session, sessionCreateError := cluster.CreateSession()
	elapsed := time.Since(start)

	log.Printf("Getting a session took %s", elapsed)

	if sessionCreateError != nil {
		return diag.FromErr(sessionCreateError)
	}

	defer session.Close()
	_name, login, superUser, saltedHash, readRoleErr := readRole(session, name)

	if readRoleErr != nil {
		return diag.FromErr(readRoleErr)
	}

	result := bcrypt.CompareHashAndPassword([]byte(saltedHash), []byte(password))

	d.SetId(_name)
	d.Set("name", _name)
	d.Set("super_user", superUser)
	d.Set("login", login)

	if result == nil {
		d.Set("password", password)
	} else {
		// password has changed between runs
		d.Set("password", saltedHash)
	}

	return diags
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	var diags diag.Diagnostics

	cluster := meta.(*gocql.ClusterConfig)
	start := time.Now()
	session, sessionCreateError := cluster.CreateSession()
	elapsed := time.Since(start)

	log.Printf("Getting a session took %s", elapsed)

	if sessionCreateError != nil {
		return diag.FromErr(sessionCreateError)
	}

	defer session.Close()

	err := session.Query(fmt.Sprintf(`DROP ROLE '%s'`, name)).Exec()
	if err != nil {
		diag.FromErr(err)
	}

	return diags
}

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceRoleCreateOrUpdate(ctx, d, meta, false)
}
