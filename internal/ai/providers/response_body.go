package providers

import (
	"fmt"
	"io"
)

const maxProviderResponseBytes int64 = 16 << 20

func readResponseBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, maxProviderResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxProviderResponseBytes {
		return nil, fmt.Errorf("provider response exceeds %d MiB", maxProviderResponseBytes>>20)
	}
	return data, nil
}
