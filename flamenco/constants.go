package flamenco

// Global constants for Flamenco Manager deployment.
// Other packages may also have constants; if they are shared
// between packages, they should go here.
const (
	AdminUsername = "flamencoadmin"

	// When changing this, be sure to also change the installation script.
	// See flamenco-manager-setup-vm.sh.
	UnixGroupName = "flamenco"

	// The VM installation script; it is named locally the same as remotely.
	InstallScriptName = "flamenco-manager-setup-vm.sh"
)
