package azvm

import "gitlab.com/blender-institute/flamenco-deploy-azure/flamenco"

const (
	// used by default for VM operations
	publisher = "Canonical"
	offer     = "UbuntuServer"
	sku       = "18.04-LTS"

	adminUsername = flamenco.AdminUsername
)
