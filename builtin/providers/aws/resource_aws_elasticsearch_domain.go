package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/xanzy/terraform-api/helper/resource"
	"github.com/xanzy/terraform-api/helper/schema"
)

func resourceAwsElasticSearchDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticSearchDomainCreate,
		Read:   resourceAwsElasticSearchDomainRead,
		Update: resourceAwsElasticSearchDomainUpdate,
		Delete: resourceAwsElasticSearchDomainDelete,

		Schema: map[string]*schema.Schema{
			"access_policies": &schema.Schema{
				Type:      schema.TypeString,
				StateFunc: normalizeJson,
				Optional:  true,
			},
			"advanced_options": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if !regexp.MustCompile(`^[0-9A-Za-z]+`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must start with a letter or number", k))
					}
					if !regexp.MustCompile(`^[0-9A-Za-z][0-9a-z-]+$`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q can only contain lowercase characters, numbers and hyphens", k))
					}
					return
				},
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"domain_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ebs_options": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ebs_enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"volume_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"cluster_config": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dedicated_master_count": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"dedicated_master_enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"dedicated_master_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"instance_count": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  1,
						},
						"instance_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "m3.medium.elasticsearch",
						},
						"zone_awareness_enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"snapshot_options": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"automated_snapshot_start_hour": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsElasticSearchDomainCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn

	input := elasticsearch.CreateElasticsearchDomainInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	}

	if v, ok := d.GetOk("access_policies"); ok {
		input.AccessPolicies = aws.String(v.(string))
	}

	if v, ok := d.GetOk("advanced_options"); ok {
		input.AdvancedOptions = stringMapToPointers(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("ebs_options"); ok {
		options := v.([]interface{})

		if len(options) > 1 {
			return fmt.Errorf("Only a single ebs_options block is expected")
		} else if len(options) == 1 {
			if options[0] == nil {
				return fmt.Errorf("At least one field is expected inside ebs_options")
			}

			s := options[0].(map[string]interface{})
			input.EBSOptions = expandESEBSOptions(s)
		}
	}

	if v, ok := d.GetOk("cluster_config"); ok {
		config := v.([]interface{})

		if len(config) > 1 {
			return fmt.Errorf("Only a single cluster_config block is expected")
		} else if len(config) == 1 {
			if config[0] == nil {
				return fmt.Errorf("At least one field is expected inside cluster_config")
			}
			m := config[0].(map[string]interface{})
			input.ElasticsearchClusterConfig = expandESClusterConfig(m)
		}
	}

	if v, ok := d.GetOk("snapshot_options"); ok {
		options := v.([]interface{})

		if len(options) > 1 {
			return fmt.Errorf("Only a single snapshot_options block is expected")
		} else if len(options) == 1 {
			if options[0] == nil {
				return fmt.Errorf("At least one field is expected inside snapshot_options")
			}

			o := options[0].(map[string]interface{})

			snapshotOptions := elasticsearch.SnapshotOptions{
				AutomatedSnapshotStartHour: aws.Int64(int64(o["automated_snapshot_start_hour"].(int))),
			}

			input.SnapshotOptions = &snapshotOptions
		}
	}

	log.Printf("[DEBUG] Creating ElasticSearch domain: %s", input)
	out, err := conn.CreateElasticsearchDomain(&input)
	if err != nil {
		return err
	}

	d.SetId(*out.DomainStatus.ARN)

	log.Printf("[DEBUG] Waiting for ElasticSearch domain %q to be created", d.Id())
	err = resource.Retry(15*time.Minute, func() error {
		out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(d.Get("domain_name").(string)),
		})
		if err != nil {
			return resource.RetryError{Err: err}
		}

		if !*out.DomainStatus.Processing && out.DomainStatus.Endpoint != nil {
			return nil
		}

		return fmt.Errorf("%q: Timeout while waiting for the domain to be created", d.Id())
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] ElasticSearch domain %q created", d.Id())

	return resourceAwsElasticSearchDomainRead(d, meta)
}

func resourceAwsElasticSearchDomainRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn

	out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Received ElasticSearch domain: %s", out)

	ds := out.DomainStatus

	if ds.AccessPolicies != nil && *ds.AccessPolicies != "" {
		d.Set("access_policies", normalizeJson(*ds.AccessPolicies))
	}
	err = d.Set("advanced_options", pointersMapToStringList(ds.AdvancedOptions))
	if err != nil {
		return err
	}
	d.Set("domain_id", *ds.DomainId)
	d.Set("domain_name", *ds.DomainName)
	if ds.Endpoint != nil {
		d.Set("endpoint", *ds.Endpoint)
	}

	err = d.Set("ebs_options", flattenESEBSOptions(ds.EBSOptions))
	if err != nil {
		return err
	}
	err = d.Set("cluster_config", flattenESClusterConfig(ds.ElasticsearchClusterConfig))
	if err != nil {
		return err
	}
	if ds.SnapshotOptions != nil {
		d.Set("snapshot_options", map[string]interface{}{
			"automated_snapshot_start_hour": *ds.SnapshotOptions.AutomatedSnapshotStartHour,
		})
	}

	d.Set("arn", *ds.ARN)

	return nil
}

func resourceAwsElasticSearchDomainUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn

	input := elasticsearch.UpdateElasticsearchDomainConfigInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	}

	if d.HasChange("access_policies") {
		input.AccessPolicies = aws.String(d.Get("access_policies").(string))
	}

	if d.HasChange("advanced_options") {
		input.AdvancedOptions = stringMapToPointers(d.Get("advanced_options").(map[string]interface{}))
	}

	if d.HasChange("ebs_options") {
		options := d.Get("ebs_options").([]interface{})

		if len(options) > 1 {
			return fmt.Errorf("Only a single ebs_options block is expected")
		} else if len(options) == 1 {
			s := options[0].(map[string]interface{})
			input.EBSOptions = expandESEBSOptions(s)
		}
	}

	if d.HasChange("cluster_config") {
		config := d.Get("cluster_config").([]interface{})

		if len(config) > 1 {
			return fmt.Errorf("Only a single cluster_config block is expected")
		} else if len(config) == 1 {
			m := config[0].(map[string]interface{})
			input.ElasticsearchClusterConfig = expandESClusterConfig(m)
		}
	}

	if d.HasChange("snapshot_options") {
		options := d.Get("snapshot_options").([]interface{})

		if len(options) > 1 {
			return fmt.Errorf("Only a single snapshot_options block is expected")
		} else if len(options) == 1 {
			o := options[0].(map[string]interface{})

			snapshotOptions := elasticsearch.SnapshotOptions{
				AutomatedSnapshotStartHour: aws.Int64(int64(o["automated_snapshot_start_hour"].(int))),
			}

			input.SnapshotOptions = &snapshotOptions
		}
	}

	_, err := conn.UpdateElasticsearchDomainConfig(&input)
	if err != nil {
		return err
	}

	err = resource.Retry(25*time.Minute, func() error {
		out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(d.Get("domain_name").(string)),
		})
		if err != nil {
			return resource.RetryError{Err: err}
		}

		if *out.DomainStatus.Processing == false {
			return nil
		}

		return fmt.Errorf("%q: Timeout while waiting for changes to be processed", d.Id())
	})
	if err != nil {
		return err
	}

	return resourceAwsElasticSearchDomainRead(d, meta)
}

func resourceAwsElasticSearchDomainDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn

	log.Printf("[DEBUG] Deleting ElasticSearch domain: %q", d.Get("domain_name").(string))
	_, err := conn.DeleteElasticsearchDomain(&elasticsearch.DeleteElasticsearchDomainInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for ElasticSearch domain %q to be deleted", d.Get("domain_name").(string))
	err = resource.Retry(15*time.Minute, func() error {
		out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(d.Get("domain_name").(string)),
		})

		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if !ok {
				return resource.RetryError{Err: err}
			}

			if awsErr.Code() == "ResourceNotFoundException" {
				return nil
			}

			return resource.RetryError{Err: awsErr}
		}

		if !*out.DomainStatus.Processing {
			return nil
		}

		return fmt.Errorf("%q: Timeout while waiting for the domain to be deleted", d.Id())
	})

	d.SetId("")

	return err
}
