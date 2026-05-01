package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	schemev1 "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	agenttranslator "github.com/kagent-dev/kagent/go/core/internal/controller/translator/agent"
)

// TestTranslateAgent_RemoteAgentTool exercises the tool.RemoteAgent translation
// path: a Tool entry referencing a RemoteAgent CRD should surface in the
// generated AgentConfig's RemoteAgents list using the URL declared on the CRD
// (not an in-cluster Service URL).
func TestTranslateAgent_RemoteAgentTool(t *testing.T) {
	ctx := context.Background()
	scheme := schemev1.Scheme
	require.NoError(t, v1alpha2.AddToScheme(scheme))

	modelConfig := &v1alpha2.ModelConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "default-model", Namespace: "test"},
		Spec: v1alpha2.ModelConfigSpec{
			Provider: "OpenAI",
			Model:    "gpt-4o",
		},
	}

	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "remote-peer", Namespace: "test"},
		Spec: v1alpha2.RemoteAgentSpec{
			Description: "Remote peer agent in cluster A",
			URL:         "https://gw-a.example.com/peer",
		},
	}

	agent := &v1alpha2.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "test"},
		Spec: v1alpha2.AgentSpec{
			Type: v1alpha2.AgentType_Declarative,
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				SystemMessage: "Test",
				ModelConfig:   "default-model",
				Tools: []*v1alpha2.Tool{
					{
						Type: v1alpha2.ToolProviderType_RemoteAgent,
						RemoteAgent: &v1alpha2.TypedReference{
							Name: "remote-peer",
						},
					},
				},
			},
		},
	}

	testNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	kagentNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kagent"}}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, remoteAgent, modelConfig, testNs, kagentNs).
		Build()

	translator := agenttranslator.NewAdkApiTranslator(
		kubeClient,
		types.NamespacedName{Name: "default-model", Namespace: "test"},
		nil,
		"", // no proxy
		nil,
	)

	result, err := agenttranslator.TranslateAgent(ctx, translator, agent)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Config)

	require.Len(t, result.Config.RemoteAgents, 1, "expected the remote agent tool to surface as a RemoteAgentConfig")
	got := result.Config.RemoteAgents[0]

	assert.Equal(t, "https://gw-a.example.com/peer", got.Url, "URL should be taken verbatim from the RemoteAgent CRD spec")
	assert.Equal(t, "Remote peer agent in cluster A", got.Description, "Description should be propagated from the RemoteAgent CRD spec")
	assert.NotEmpty(t, got.Name, "Name should be a stable Python identifier derived from the CRD object ref")
}

// TestTranslateAgent_RemoteAgentTool_HeaderMerge verifies that headers from the
// Tool entry override headers from the RemoteAgent CRD on a per-key basis,
// matching the documented HeadersFrom precedence.
func TestTranslateAgent_RemoteAgentTool_HeaderMerge(t *testing.T) {
	ctx := context.Background()
	scheme := schemev1.Scheme
	require.NoError(t, v1alpha2.AddToScheme(scheme))

	modelConfig := &v1alpha2.ModelConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "default-model", Namespace: "test"},
		Spec: v1alpha2.ModelConfigSpec{
			Provider: "OpenAI",
			Model:    "gpt-4o",
		},
	}

	// Two secrets, one referenced from the RemoteAgent CRD and one from the
	// Tool entry — they share a header key (Authorization) so we can confirm
	// the Tool-level value wins. They also each contribute a unique header
	// to confirm both sources are merged.
	crdSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "crd-secret", Namespace: "test"},
		Data: map[string][]byte{
			"crd-token": []byte("from-crd"),
			"crd-only":  []byte("crd-extra"),
		},
	}
	toolSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "tool-secret", Namespace: "test"},
		Data: map[string][]byte{
			"tool-token": []byte("from-tool"),
			"tool-only":  []byte("tool-extra"),
		},
	}

	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "remote-peer", Namespace: "test"},
		Spec: v1alpha2.RemoteAgentSpec{
			URL: "https://gw-a.example.com/peer",
			HeadersFrom: []v1alpha2.ValueRef{
				{
					Name: "Authorization",
					ValueFrom: &v1alpha2.ValueSource{
						Type: v1alpha2.SecretValueSource,
						Name: "crd-secret",
						Key:  "crd-token",
					},
				},
				{
					Name: "X-Crd-Only",
					ValueFrom: &v1alpha2.ValueSource{
						Type: v1alpha2.SecretValueSource,
						Name: "crd-secret",
						Key:  "crd-only",
					},
				},
			},
		},
	}

	agent := &v1alpha2.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "test"},
		Spec: v1alpha2.AgentSpec{
			Type: v1alpha2.AgentType_Declarative,
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				SystemMessage: "Test",
				ModelConfig:   "default-model",
				Tools: []*v1alpha2.Tool{
					{
						Type: v1alpha2.ToolProviderType_RemoteAgent,
						RemoteAgent: &v1alpha2.TypedReference{
							Name: "remote-peer",
						},
						HeadersFrom: []v1alpha2.ValueRef{
							{
								// Same key as the CRD — Tool-level should win.
								Name: "Authorization",
								ValueFrom: &v1alpha2.ValueSource{
									Type: v1alpha2.SecretValueSource,
									Name: "tool-secret",
									Key:  "tool-token",
								},
							},
							{
								Name: "X-Tool-Only",
								ValueFrom: &v1alpha2.ValueSource{
									Type: v1alpha2.SecretValueSource,
									Name: "tool-secret",
									Key:  "tool-only",
								},
							},
						},
					},
				},
			},
		},
	}

	testNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	kagentNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kagent"}}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, remoteAgent, modelConfig, crdSecret, toolSecret, testNs, kagentNs).
		Build()

	translator := agenttranslator.NewAdkApiTranslator(
		kubeClient,
		types.NamespacedName{Name: "default-model", Namespace: "test"},
		nil,
		"",
		nil,
	)

	result, err := agenttranslator.TranslateAgent(ctx, translator, agent)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Config.RemoteAgents, 1)

	got := result.Config.RemoteAgents[0]
	require.NotNil(t, got.Headers)

	assert.Equal(t, "from-tool", got.Headers["Authorization"],
		"Tool HeadersFrom should override RemoteAgent CRD HeadersFrom on shared keys")
	assert.Equal(t, "crd-extra", got.Headers["X-Crd-Only"],
		"Headers only declared on the RemoteAgent CRD should be present")
	assert.Equal(t, "tool-extra", got.Headers["X-Tool-Only"],
		"Headers only declared on the Tool entry should be present")
}

// makeBaseObjects returns the standard set of fixtures used by the negative
// tests below.
func makeBaseObjects() (*v1alpha2.ModelConfig, *corev1.Namespace, *corev1.Namespace) {
	return &v1alpha2.ModelConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "default-model", Namespace: "test"},
			Spec: v1alpha2.ModelConfigSpec{
				Provider: "OpenAI",
				Model:    "gpt-4o",
			},
		},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kagent"}}
}

// TestTranslateAgent_RemoteAgentTool_NotFound: referencing a non-existent
// RemoteAgent should surface a clear error rather than panic or silently
// succeed.
func TestTranslateAgent_RemoteAgentTool_NotFound(t *testing.T) {
	ctx := context.Background()
	scheme := schemev1.Scheme
	require.NoError(t, v1alpha2.AddToScheme(scheme))

	modelConfig, testNs, kagentNs := makeBaseObjects()

	agent := &v1alpha2.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "test"},
		Spec: v1alpha2.AgentSpec{
			Type: v1alpha2.AgentType_Declarative,
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				SystemMessage: "Test",
				ModelConfig:   "default-model",
				Tools: []*v1alpha2.Tool{
					{
						Type:        v1alpha2.ToolProviderType_RemoteAgent,
						RemoteAgent: &v1alpha2.TypedReference{Name: "missing-peer"},
					},
				},
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, modelConfig, testNs, kagentNs).
		Build()

	translator := agenttranslator.NewAdkApiTranslator(
		kubeClient,
		types.NamespacedName{Name: "default-model", Namespace: "test"},
		nil, "", nil,
	)

	_, err := agenttranslator.TranslateAgent(ctx, translator, agent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-peer")
}

// TestTranslateAgent_RemoteAgentTool_NilReference: type=RemoteAgent with no
// remoteAgent field set must fail validation, not panic.
func TestTranslateAgent_RemoteAgentTool_NilReference(t *testing.T) {
	ctx := context.Background()
	scheme := schemev1.Scheme
	require.NoError(t, v1alpha2.AddToScheme(scheme))

	modelConfig, testNs, kagentNs := makeBaseObjects()

	agent := &v1alpha2.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "test"},
		Spec: v1alpha2.AgentSpec{
			Type: v1alpha2.AgentType_Declarative,
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				SystemMessage: "Test",
				ModelConfig:   "default-model",
				Tools: []*v1alpha2.Tool{
					{Type: v1alpha2.ToolProviderType_RemoteAgent},
				},
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, modelConfig, testNs, kagentNs).
		Build()

	translator := agenttranslator.NewAdkApiTranslator(
		kubeClient,
		types.NamespacedName{Name: "default-model", Namespace: "test"},
		nil, "", nil,
	)

	_, err := agenttranslator.TranslateAgent(ctx, translator, agent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remoteAgent reference")
}

// TestTranslateAgent_RemoteAgentTool_BothAgentAndRemoteAgent: a Tool entry
// must not set both Agent and RemoteAgent — the consuming agent must be
// rejected up front.
func TestTranslateAgent_RemoteAgentTool_BothAgentAndRemoteAgent(t *testing.T) {
	ctx := context.Background()
	scheme := schemev1.Scheme
	require.NoError(t, v1alpha2.AddToScheme(scheme))

	modelConfig, testNs, kagentNs := makeBaseObjects()

	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "peer", Namespace: "test"},
		Spec:       v1alpha2.RemoteAgentSpec{URL: "https://gw.example.com/peer"},
	}

	subAgent := &v1alpha2.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "sub", Namespace: "test"},
		Spec: v1alpha2.AgentSpec{
			Type: v1alpha2.AgentType_Declarative,
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				SystemMessage: "sub",
				ModelConfig:   "default-model",
			},
		},
	}

	agent := &v1alpha2.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "test"},
		Spec: v1alpha2.AgentSpec{
			Type: v1alpha2.AgentType_Declarative,
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				SystemMessage: "Test",
				ModelConfig:   "default-model",
				Tools: []*v1alpha2.Tool{
					{
						Type:        v1alpha2.ToolProviderType_RemoteAgent,
						Agent:       &v1alpha2.TypedReference{Name: "sub"},
						RemoteAgent: &v1alpha2.TypedReference{Name: "peer"},
					},
				},
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, subAgent, remoteAgent, modelConfig, testNs, kagentNs).
		Build()

	translator := agenttranslator.NewAdkApiTranslator(
		kubeClient,
		types.NamespacedName{Name: "default-model", Namespace: "test"},
		nil, "", nil,
	)

	_, err := agenttranslator.TranslateAgent(ctx, translator, agent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both Agent and RemoteAgent")
}

// TestTranslateAgent_RemoteAgentTool_TimeoutPropagation verifies that a
// spec.timeout set on the RemoteAgent CRD is converted to float64 seconds and
// carried through to the generated RemoteAgentConfig.Timeout field.
func TestTranslateAgent_RemoteAgentTool_TimeoutPropagation(t *testing.T) {
	ctx := context.Background()
	scheme := schemev1.Scheme
	require.NoError(t, v1alpha2.AddToScheme(scheme))

	modelConfig, testNs, kagentNs := makeBaseObjects()

	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "remote-peer", Namespace: "test"},
		Spec: v1alpha2.RemoteAgentSpec{
			Description: "Timed peer agent",
			URL:         "https://gw-a.example.com/peer",
			Timeout:     &metav1.Duration{Duration: 30 * time.Second},
		},
	}

	agent := &v1alpha2.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "test"},
		Spec: v1alpha2.AgentSpec{
			Type: v1alpha2.AgentType_Declarative,
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				SystemMessage: "Test",
				ModelConfig:   "default-model",
				Tools: []*v1alpha2.Tool{
					{
						Type: v1alpha2.ToolProviderType_RemoteAgent,
						RemoteAgent: &v1alpha2.TypedReference{
							Name: "remote-peer",
						},
					},
				},
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, remoteAgent, modelConfig, testNs, kagentNs).
		Build()

	translator := agenttranslator.NewAdkApiTranslator(
		kubeClient,
		types.NamespacedName{Name: "default-model", Namespace: "test"},
		nil,
		"",
		nil,
	)

	result, err := agenttranslator.TranslateAgent(ctx, translator, agent)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Config.RemoteAgents, 1)

	got := result.Config.RemoteAgents[0]
	require.NotNil(t, got.Timeout, "Timeout should be propagated to RemoteAgentConfig")
	assert.Equal(t, 30.0, *got.Timeout, "Timeout should be 30.0 seconds")
}

// TestTranslateAgent_RemoteAgentTool_MissingHeaderSecret: header resolution
// failure on the RemoteAgent CRD must surface as a clear error.
func TestTranslateAgent_RemoteAgentTool_MissingHeaderSecret(t *testing.T) {
	ctx := context.Background()
	scheme := schemev1.Scheme
	require.NoError(t, v1alpha2.AddToScheme(scheme))

	modelConfig, testNs, kagentNs := makeBaseObjects()

	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "peer", Namespace: "test"},
		Spec: v1alpha2.RemoteAgentSpec{
			URL: "https://gw.example.com/peer",
			HeadersFrom: []v1alpha2.ValueRef{{
				Name: "Authorization",
				ValueFrom: &v1alpha2.ValueSource{
					Type: v1alpha2.SecretValueSource,
					Name: "missing-secret",
					Key:  "token",
				},
			}},
		},
	}

	agent := &v1alpha2.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "test"},
		Spec: v1alpha2.AgentSpec{
			Type: v1alpha2.AgentType_Declarative,
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				SystemMessage: "Test",
				ModelConfig:   "default-model",
				Tools: []*v1alpha2.Tool{
					{
						Type:        v1alpha2.ToolProviderType_RemoteAgent,
						RemoteAgent: &v1alpha2.TypedReference{Name: "peer"},
					},
				},
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, remoteAgent, modelConfig, testNs, kagentNs).
		Build()

	translator := agenttranslator.NewAdkApiTranslator(
		kubeClient,
		types.NamespacedName{Name: "default-model", Namespace: "test"},
		nil, "", nil,
	)

	_, err := agenttranslator.TranslateAgent(ctx, translator, agent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "headers")
}
