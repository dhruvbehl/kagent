/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RemoteAgentSpec defines the desired state of RemoteAgent.
//
// A RemoteAgent represents an external A2A (Agent-to-Agent) protocol endpoint
// that an in-cluster Agent can reference as a sub-agent tool. This is the
// counterpart to RemoteMCPServer for A2A peers (e.g., a kagent Agent running
// in a different cluster, exposed via an agentgateway A2A frontend).
type RemoteAgentSpec struct {
	// Description of the remote agent. Surfaced to the calling agent's runtime
	// alongside the URL and is useful as context in the caller's system prompt.
	// +optional
	Description string `json:"description,omitempty"`

	// URL is the A2A endpoint of the remote agent. The remote endpoint is
	// expected to speak JSON-RPC A2A (e.g., message/send) and serve an agent
	// card at /.well-known/agent.json on the same origin.
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`

	// HeadersFrom specifies a list of configuration values to be added as
	// headers to A2A requests sent to this remote agent. Values are resolved
	// from Secrets or ConfigMaps in the same namespace as the RemoteAgent
	// resource. Headers specified at the Tool level on the consuming Agent
	// will override any header of the same name configured here.
	// +optional
	HeadersFrom []ValueRef `json:"headersFrom,omitempty"`

	// Timeout for A2A calls to this remote agent. If unset, the runtime's
	// default A2A client timeout is used.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// AllowedNamespaces defines which namespaces are allowed to reference this
	// RemoteAgent. This follows the Gateway API pattern for cross-namespace
	// route attachments. If not specified, only Agents in the same namespace
	// can reference this RemoteAgent.
	// See: https://gateway-api.sigs.k8s.io/guides/multiple-ns/#cross-namespace-routing
	// +optional
	AllowedNamespaces *AllowedNamespaces `json:"allowedNamespaces,omitempty"`
}

// RemoteAgentStatus defines the observed state of RemoteAgent.
type RemoteAgentStatus struct {
	ObservedGeneration int64              `json:"observedGeneration"`
	Conditions         []metav1.Condition `json:"conditions"`

	// AgentCard contains the fetched agent card JSON from the remote endpoint.
	// +optional
	AgentCard string `json:"agentCard,omitempty"`

	// AgentName is the name extracted from the agent card.
	// +optional
	AgentName string `json:"agentName,omitempty"`

	// AgentDescription is the description extracted from the agent card.
	// +optional
	AgentDescription string `json:"agentDescription,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=ras,categories=kagent
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.url"
// +kubebuilder:printcolumn:name="Accepted",type="string",JSONPath=".status.conditions[?(@.type=='Accepted')].status"

// RemoteAgent is the Schema for the RemoteAgents API.
type RemoteAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemoteAgentSpec   `json:"spec,omitempty"`
	Status RemoteAgentStatus `json:"status,omitempty"`
}

// ResolveHeaders resolves all HeadersFrom entries using the object's namespace.
func (r *RemoteAgent) ResolveHeaders(ctx context.Context, client client.Client) (map[string]string, error) {
	result := map[string]string{}

	for _, h := range r.Spec.HeadersFrom {
		k, v, err := h.Resolve(ctx, client, r.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve header: %v", err)
		}

		result[k] = v
	}

	return result, nil
}

// +kubebuilder:object:root=true
// RemoteAgentList contains a list of RemoteAgent.
type RemoteAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemoteAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemoteAgent{}, &RemoteAgentList{})
}
