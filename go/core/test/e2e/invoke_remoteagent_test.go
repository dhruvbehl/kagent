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
//  1. A "target" Agent runs in-cluster (using the mock LLM for its model).
//     This stands in for an external A2A endpoint — we point the RemoteAgent
//     at the target's in-cluster Service URL so the test does not need any
//     actual cross-cluster plumbing.
//  2. A RemoteAgent CR is created pointing at the target's A2A URL.
//     The reconciler should surface Accepted=True.
//  3. A "consumer" Agent is created with a tool entry of type=RemoteAgent
//     referencing the RemoteAgent. The agent should reach Ready, proving
//     the translator path emits a valid runtime config.
//  4. A message is sent through the consumer agent to verify the data-plane
//     path works end-to-end (consumer A2A endpoint → runtime → response).
func TestE2ERemoteAgentTool(t *testing.T) {
	baseURL, stopServer := setupMockServer(t, "mocks/invoke_remoteagent.json")
	defer stopServer()

	cli := setupK8sClient(t, false)
	modelCfg := setupModelConfig(t, cli, baseURL)

	// Step 1: target agent — a normal in-cluster Declarative agent. The
	// existing setupAgent helper waits for Ready and the A2A endpoint.
	targetAgent := setupAgent(t, cli, modelCfg.Name, nil)

	// Step 2: RemoteAgent pointing at the target agent's in-cluster Service.
	// The kagent translator emits a Service at <name>.<namespace>:8080 for
	// every Agent, which is reachable from any pod in the cluster.
	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-remote-agent-",
			Namespace:    "kagent",
		},
		Spec: v1alpha2.RemoteAgentSpec{
			Description: "E2E test RemoteAgent pointing at an in-cluster target",
			URL:         fmt.Sprintf("http://%s.%s:8080", targetAgent.Name, targetAgent.Namespace),
		},
	}
	require.NoError(t, cli.Create(t.Context(), remoteAgent))
	cleanup(t, cli, remoteAgent)

	waitForRemoteAgentAccepted(t, cli, remoteAgent, metav1.ConditionTrue)

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

	consumer := setupAgentWithOptions(t, cli, modelCfg.Name, tools, AgentOptions{
		Name: "remote-agent-consumer",
	})

	// Sanity: consumer should be Accepted.
	got := &v1alpha2.Agent{}
	require.NoError(t, cli.Get(t.Context(), client.ObjectKeyFromObject(consumer), got))
	assertAgentAccepted(t, got)

	// Step 4: data-plane verification — send a message through the consumer
	// agent and assert a non-error response is returned. This confirms that
	// the A2A proxy, the consumer's runtime, and the RemoteAgent tool
	// configuration are all wired up correctly end-to-end.
	t.Run("data_plane_invocation", func(t *testing.T) {
		a2aClient := setupA2AClient(t, consumer)
		runSyncTest(t, a2aClient, "hello", "remote-agent-consumer", nil)
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
