package v1beta1

import "fmt"

func ManagedByKey(ressourceId string) string {
	return fmt.Sprintf("%s/%s", "managed-by", ressourceId)
}

const (
	ManagedByValueCapi     string = "capi"
	ManagedByValueExternal string = "external"
)
