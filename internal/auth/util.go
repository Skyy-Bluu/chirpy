package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {

	apkiKeyHeader := headers.Get("Authorization")

	if apkiKeyHeader == "" {
		return "", fmt.Errorf("API key not found")
	}

	if apiKey, ok := strings.CutPrefix(apkiKeyHeader, "ApiKey "); ok {
		return apiKey, nil
	} else {
		return "", fmt.Errorf("Couldn't extract API key")
	}
}
