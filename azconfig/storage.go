package azconfig

// StorageCredentials has everything you need to mount a file share from a storage account.
type StorageCredentials struct {
	Username string // the storage account name
	Password string // the storage account key
}
