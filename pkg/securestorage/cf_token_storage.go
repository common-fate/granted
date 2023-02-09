package securestorage

type CFTokenStorage struct {
	Storage *SecureStorage
}

func NewCF() CFTokenStorage {
	return CFTokenStorage{
		Storage: &SecureStorage{
			StorageSuffix: "cf",
		},
	}
}
