package main

import (
	"github.com/xanzy/terraform-api/builtin/providers/mysql"
	"github.com/xanzy/terraform-api/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: mysql.Provider,
	})
}
