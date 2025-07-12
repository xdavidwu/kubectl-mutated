package metadata

import (
	"strings"
)

// Either explicitly specified, or from user-agent before '/'
// see k8s.io/apiserver/pkg/endpoints/handlers.managerOrUserAgent
func IsManualManager(m string) bool {
	return (strings.HasPrefix(m, "kubectl") && m != "kubectl-rollout") ||
		// helm cli managed resources (via its generic client)
		// (flux helm-controller uses "helm-controller")
		m == "helm" ||
		// helm cli storage (secrets, configmaps) implicitly via user-agent Helm/<version>
		// (flux helm-controller users "helm-controller")
		m == "Helm"
}
