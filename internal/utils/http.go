package utils

import (
	"net/http"
)

func Retryable(resp *http.Response) bool {
	if resp != nil {
		if resp.StatusCode > 500 || resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusTooManyRequests {
			return true
		}
	}
	return false
}
