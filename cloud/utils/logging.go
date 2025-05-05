package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"k8s.io/klog/v2"
)

const maxResponseLength = 500

func clean(buf []byte) string {
	return strings.ReplaceAll(string(buf), `"`, ``)
}

func truncatedBody(httpResp *http.Response) string {
	body, err := io.ReadAll(httpResp.Body)
	if err == nil {
		str := []rune(clean(body))
		if len(str) > maxResponseLength {
			return string(str[:maxResponseLength/2]) + " [truncated] " + string(str[len(str)-maxResponseLength/2:])
		}
		return string(str)
	}
	return "(unable to fetch body)"
}

func errorBody(err error, httpResp *http.Response) (string, error) {
	body, rerr := io.ReadAll(httpResp.Body)
	if rerr == nil {
		return clean(body), extractOAPIError(err, body)
	}
	return "(unable to fetch body)", err
}

func LogAndExtractError(ctx context.Context, call string, request any, httpResp *http.Response, err error) error {
	logger := klog.FromContext(ctx).WithCallDepth(1)
	if logger.V(5).Enabled() {
		buf, _ := json.Marshal(request)
		logger.Info("OAPI request: "+clean(buf), "OAPI", call)
	}
	switch {
	case err != nil && httpResp == nil:
		logger.V(3).Error(err, "OAPI error", "OAPI", call)
	case httpResp == nil:
	case httpResp.StatusCode > 299:
		var body string
		body, err = errorBody(err, httpResp)
		err = fmt.Errorf("%s returned %w", call, err)
		logger.V(3).Info("OAPI error response: "+body, "OAPI", call, "http_status", httpResp.Status)
	case logger.V(5).Enabled(): // no error
		logger.Info("OAPI response: "+truncatedBody(httpResp), "OAPI", call)
	}
	return err
}
