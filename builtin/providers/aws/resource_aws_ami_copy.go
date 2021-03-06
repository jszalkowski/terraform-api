package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/xanzy/terraform-api/helper/schema"
)

func resourceAwsAmiCopy() *schema.Resource {
	// Inherit all of the common AMI attributes from aws_ami, since we're
	// implicitly creating an aws_ami resource.
	resourceSchema := resourceAwsAmiCommonSchema(true)

	// Additional attributes unique to the copy operation.
	resourceSchema["source_ami_id"] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	}
	resourceSchema["source_ami_region"] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	}

	return &schema.Resource{
		Create: resourceAwsAmiCopyCreate,

		Schema: resourceSchema,

		// The remaining operations are shared with the generic aws_ami resource,
		// since the aws_ami_copy resource only differs in how it's created.
		Read:   resourceAwsAmiRead,
		Update: resourceAwsAmiUpdate,
		Delete: resourceAwsAmiDelete,
	}
}

func resourceAwsAmiCopyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).ec2conn

	req := &ec2.CopyImageInput{
		Name:          aws.String(d.Get("name").(string)),
		Description:   aws.String(d.Get("description").(string)),
		SourceImageId: aws.String(d.Get("source_ami_id").(string)),
		SourceRegion:  aws.String(d.Get("source_ami_region").(string)),
	}

	res, err := client.CopyImage(req)
	if err != nil {
		return err
	}

	id := *res.ImageId
	d.SetId(id)
	d.Partial(true) // make sure we record the id even if the rest of this gets interrupted
	d.Set("id", id)
	d.Set("manage_ebs_snapshots", true)
	d.SetPartial("id")
	d.SetPartial("manage_ebs_snapshots")
	d.Partial(false)

	_, err = resourceAwsAmiWaitForAvailable(id, client)
	if err != nil {
		return err
	}

	return resourceAwsAmiUpdate(d, meta)
}
