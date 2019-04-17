package flamenco

// Global constants for Flamenco Manager deployment.
// Other packages may also have constants; if they are shared
// between packages, they should go here.
const (
	AdminUsername = "flamencoadmin"
	UnixGroupName = "flamenco"

	// The VM installation script; it is named locally the same as remotely.
	InstallScriptName = "flamenco-manager-setup-vm.sh"
)
