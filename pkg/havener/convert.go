package havener

import (
	"fmt"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //from https://github.com/kubernetes/client-go/issues/345
)

func HumanReadableSize(bytes int64) string {
	mods := []string{"", "KiB", "MiB", "GiB", "TiB"}

	value := float64(bytes)
	i := 0
	for value > 1024.0 {
		value /= 1024.0
		i++
	}

	return fmt.Sprintf("%.1f %s", value, mods[i])
}
