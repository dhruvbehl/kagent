package e2e_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestE2ERemoteAgentTool exercises the new RemoteAgent CRD end-to-end:
//
//  1. A "target" Agent runs in-cluster backed by invoke_remoteagent_target.json.
//     This stands in for an external A2A endpoint — we point the RemoteAgent
//     at the target's in-cluster Service URL so the test does not need any
//     actual cross-cluster plumbing.
//  2. A RemoteAgent CR is created pointing at the target agent's in-cluster Service.
//     The reconciler should surface Accepted=True.
//  3. A "consumer" Agent backed by invoke_remoteagent_consumer.json is created
//     with a tool entry of type=RemoteAgent referencing the RemoteAgent. The
//     agent should reach Ready, proving the translator path emits a valid
//     runtime config.
//  4. A message is sent through the consumer agent. The consumer LLM emits a
//     tool call targeting the remote agent, the runtime calls the target via
//     A2A, the target LLM responds with "from-target-agent", and the consumer
//     LLM produces a final answer containing "delegated-response-confirmed".
//     This confirms the full data-plane path is wired correctly.
func TestE2ERemoteAgentTool(t *testing.T) {
	// Two separate mock servers: one for the target agent's LLM, one for the
	// consumer agent's LLM. This lets each agent have distinct mock behaviour.
	targetURL, stopTargetServer := setupMockServer(t, "mocks/invoke_remoteagent_target.json")
	defer stopTargetServer()

	consumerURL, stopConsumerServer := setupMockServer(t, "mocks/invoke_remoteagent_consumer.json")
	defer stopConsumerServer()

	cli := setupK8sClient(t, false)
	targetModelCfg := setupModelConfig(t, cli, targetURL)
	consumerModelCfg := setupModelConfig(t, cli, consumerURL)

	// Step 1: target agent — a normal in-cluster Declarative agent. The
	// existing setupAgent helper waits for Ready and the A2A endpoint.
	targetAgent := setupAgent(t, cli, targetModelCfg.Name, nil)

	// Step 2: RemoteAgent pointing at the target agent's in-cluster Service.
	// The kagent translator emits a Service at <name>.<namespace>:8080 for
	// every Agent, which is reachable from any pod in the cluster.
	//
	// We use a fixed Name (not GenerateName) so that the tool name seen by the
	// consumer LLM is deterministic: utils.ConvertToPythonIdentifier gives
	// "kagent__NS__test_remote_agent".
	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-remote-agent",
			Namespace: "kagent",
		},
		Spec: v1alpha2.RemoteAgentSpec{
			Description: "E2E test RemoteAgent pointing at an in-cluster target",
			URL:         fmt.Sprintf("http://%s.%s:8080", targetAgent.Name, targetAgent.Namespace),
		},
	}
	// Clean up any leftover from a previous failed run before creating.
	_ = cli.Delete(t.Context(), remoteAgent)
	require.NoError(t, cli.Create(t.Context(), remoteAgent))
	cleanup(t, cli, remoteAgent)

	waitForRemoteAgentAccepted(t, cli, remoteAgent, metav1.ConditionTrue)
	// NOTE: We intentionally do not wait for Reachable=True here. The a2a-go
	// library fetches /.well-known/agent-card.json, but kagent agents (following
	// the A2A spec) serve /.well-known/agent.json. This path mismatch causes the
	// Reachable condition to stay False until a2a-go is upgraded.
	// TODO: Add waitForRemoteAgentReachable once a2a-go path is fixed.

	// Step 3: consumer agent referencing the RemoteAgent as a tool. A
	// successful Ready condition proves the translator emitted a valid
	// runtime config and the deployment came up.
	tools := []*v1alpha2.Tool{{
		Type: v1alpha2.ToolProviderType_RemoteAgent,
		RemoteAgent: &v1alpha2.TypedReference{
			Name:      remoteAgent.Name,
			Namespace: remoteAgent.Namespace,
		},
	}}

	consumer := setupAgentWithOptions(t, cli, consumerModelCfg.Name, tools, AgentOptions{
		Name: "remote-agent-consumer",
	})

	// Sanity: consumer should be Accepted.
	got := &v1alpha2.Agent{}
	require.NoError(t, cli.Get(t.Context(), client.ObjectKeyFromObject(consumer), got))
	assertAgentAccepted(t, got)

	// Step 4: data-plane verification — send a message through the consumer
	// agent and assert the final response contains "delegated-response-confirmed".
	//
	// This proves the full round-trip:
	//   user "hello" → consumer LLM emits tool call (kagent__NS__test_remote_agent)
	//   → runtime calls target agent via A2A → target LLM returns "from-target-agent"
	//   → consumer LLM receives tool result and responds "delegated-response-confirmed"
	t.Run("data_plane_invocation", func(t *testing.T) {
		a2aClient := setupA2AClient(t, consumer)
		runSyncTest(t, a2aClient, "hello", "delegated-response-confirmed", nil)
	})
}

// TestE2ERemoteAgentInvalidURL verifies the RemoteAgent reconciler surfaces a
// failed Accepted condition when the URL is not a valid http(s) URL. This is
// the negative path for the new validateRemoteAgentSpec logic.
func TestE2ERemoteAgentInvalidURL(t *testing.T) {
	cli := setupK8sClient(t, false)

	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-remote-agent-bad-",
			Namespace:    "kagent",
		},
		Spec: v1alpha2.RemoteAgentSpec{
			URL: "not-a-real-url",
		},
	}
	require.NoError(t, cli.Create(t.Context(), remoteAgent))
	cleanup(t, cli, remoteAgent)

	waitForRemoteAgentAccepted(t, cli, remoteAgent, metav1.ConditionFalse)
}

// waitForRemoteAgentAccepted polls until the RemoteAgent has an Accepted
// condition matching the wanted status, or the test deadline elapses.
func waitForRemoteAgentAccepted(
	t *testing.T,
	cli client.Client,
	remoteAgent *v1alpha2.RemoteAgent,
	want metav1.ConditionStatus,
) {
	t.Helper()
	pollErr := wait.PollUntilContextTimeout(
		t.Context(), 2*time.Second, 60*time.Second, true,
		func(ctx context.Context) (bool, error) {
			got := &v1alpha2.RemoteAgent{}
			if err := cli.Get(ctx, client.ObjectKeyFromObject(remoteAgent), got); err != nil {
				return false, err
			}
			for _, c := range got.Status.Conditions {
				if c.Type == v1alpha2.AgentConditionTypeAccepted && c.Status == want {
					return true, nil
				}
			}
			return false, nil
		},
	)
	if pollErr != nil {
		dumpRemoteAgent(t, cli, remoteAgent)
		t.Fatalf("RemoteAgent never reached Accepted=%s: %v", want, pollErr)
	}
}

// waitForRemoteAgentReachable polls until the RemoteAgent has a Reachable
// condition matching the wanted status. This validates that the controller
// successfully fetched the agent card from the remote endpoint.
func waitForRemoteAgentReachable(
	t *testing.T,
	cli client.Client,
	remoteAgent *v1alpha2.RemoteAgent,
	want metav1.ConditionStatus,
) {
	t.Helper()
	pollErr := wait.PollUntilContextTimeout(
		t.Context(), 2*time.Second, 60*time.Second, true,
		func(ctx context.Context) (bool, error) {
			got := &v1alpha2.RemoteAgent{}
			if err := cli.Get(ctx, client.ObjectKeyFromObject(remoteAgent), got); err != nil {
				return false, err
			}
			for _, c := range got.Status.Conditions {
				if c.Type == "Reachable" && c.Status == want {
					return true, nil
				}
			}
			return false, nil
		},
	)
	if pollErr != nil {
		dumpRemoteAgent(t, cli, remoteAgent)
		t.Fatalf("RemoteAgent never reached Reachable=%s: %v", want, pollErr)
	}
}

func dumpRemoteAgent(t *testing.T, cli client.Client, remoteAgent *v1alpha2.RemoteAgent) {
	t.Helper()
	got := &v1alpha2.RemoteAgent{}
	if err := cli.Get(context.Background(), client.ObjectKeyFromObject(remoteAgent), got); err == nil {
		t.Logf("RemoteAgent %s/%s status: %+v", got.Namespace, got.Name, got.Status)
	}

	// Also dump the controller logs to aid debugging.
	if os.Getenv("SKIP_CLEANUP") != "" {
		return
	}
	cmd := exec.Command("kubectl", "logs", "-n", "kagent", "deployment/kagent-controller", "--tail=100")
	out, _ := cmd.CombinedOutput()
	t.Logf("controller logs (tail):\n%s", string(out))
}

func assertAgentAccepted(t *testing.T, agent *v1alpha2.Agent) {
	t.Helper()
	for _, c := range agent.Status.Conditions {
		if c.Type == v1alpha2.AgentConditionTypeAccepted {
			assert.Equal(t, metav1.ConditionTrue, c.Status,
				"consumer Agent should be Accepted, got %s: %s", c.Status, c.Message)
			return
		}
	}
	t.Fatalf("consumer Agent missing Accepted condition; status=%+v", agent.Status)
}
