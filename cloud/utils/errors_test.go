package utils_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractOAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		fmt.Fprintln(w, `{"Errors":[{"Type":"InternalError","Details":"Outscale faced an internal error while processing your request","Code":"2000"}],"ResponseContext":{"RequestId":"7c0d8260-0fe2-4acd-af4a-31dc2245eaf8"}}`)
	}))
	defer srv.Close()
	cl := osc.NewAPIClient(&osc.Configuration{
		DefaultHeader:    make(map[string]string),
		Servers:          osc.ServerConfigurations{{URL: srv.URL}},
		OperationServers: map[string]osc.ServerConfigurations{},
	})
	ctx := context.TODO()
	_, httpRes, err := cl.NetApi.CreateNet(ctx).CreateNetRequest(osc.CreateNetRequest{}).Execute()
	require.Error(t, err)
	err = utils.LogAndExtractError(ctx, "CreateNet", osc.CreateNetRequest{}, httpRes, err)
	assert.EqualError(t, err, "2000/InternalError (Outscale faced an internal error while processing your request)")
}
