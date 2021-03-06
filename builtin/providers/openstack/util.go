package openstack

import (
	"fmt"

	"github.com/rackspace/gophercloud"
	"github.com/xanzy/terraform-api/helper/schema"
)

// CheckDeleted checks the error to see if it's a 404 (Not Found) and, if so,
// sets the resource ID to the empty string instead of throwing an error.
func CheckDeleted(d *schema.ResourceData, err error, msg string) error {
	errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
	if !ok {
		return fmt.Errorf("%s: %s", msg, err)
	}
	if errCode.Actual == 404 {
		d.SetId("")
		return nil
	}
	return fmt.Errorf("%s: %s", msg, err)
}
