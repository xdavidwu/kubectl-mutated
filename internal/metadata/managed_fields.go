package metadata

import (
	"strings"
)

func IsManualManager(m string) bool {
	return strings.HasPrefix(m, "kubectl") && m != "kubectl-rollout"
}
