package tenant

import (
	"context"

	"github.com/outscale/osc-sdk-go/v2"
)

type Tenant interface {
	Region() string
	ContextWithAuth(context.Context) context.Context
	Client() *osc.APIClient
}
