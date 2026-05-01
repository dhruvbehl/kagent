import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { AgentsContext } from "@/components/AgentsProvider";
import { AppPageFrame } from "@/components/layout/AppPageFrame";
import { PageHeader } from "@/components/layout/PageHeader";
import { RemoteAgentsView } from "@/components/remoteagents/RemoteAgentsView";
import type { RemoteAgent } from "@/types";
import { createStoryAgentsContext } from "./fixtures";

const storyRemoteAgents: RemoteAgent[] = [
  {
    metadata: { name: "data-retrieval-agent", namespace: "kagent" },
    spec: {
      url: "http://data-agent.data-team.svc.cluster.local:8080",
      description: "Specialized data retrieval agent in the data-team cluster",
      timeout: "30s",
    },
    status: {
      conditions: [
        { type: "Accepted", status: "True", reason: "Valid", message: "Spec is valid" },
        { type: "Reachable", status: "True", reason: "AgentCardFetched", message: "Agent card fetched" },
      ],
      agentName: "data-retrieval-agent",
      agentDescription: "Retrieves structured data from internal databases",
    },
  },
  {
    metadata: { name: "code-execution-agent", namespace: "platform" },
    spec: {
      url: "http://code-agent.platform.svc.cluster.local:8080",
      description: "Sandboxed code execution agent",
    },
    status: {
      conditions: [
        { type: "Accepted", status: "True", reason: "Valid", message: "Spec is valid" },
        { type: "Reachable", status: "False", reason: "AgentCardFetchFailed", message: "connection refused" },
      ],
    },
  },
  {
    metadata: { name: "unvalidated-agent", namespace: "kagent" },
    spec: {
      url: "http://external.example.com/agent",
    },
  },
];

const meta = {
  title: "Pages/View/Remote Agents",
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "`/remoteagents` — `RemoteAgentsView` with mock data (no API).",
      },
    },
  },
  decorators: [
    (Story) => (
      <AgentsContext.Provider value={createStoryAgentsContext({})}>
        <Story />
      </AgentsContext.Provider>
    ),
  ],
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

export const Loaded: Story = {
  render: () => (
    <AppPageFrame ariaLabelledBy="remoteagents-page-title" mainClassName="mx-auto max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
      <PageHeader
        titleId="remoteagents-page-title"
        title="Remote Agents"
        description="Register external A2A endpoints so your local agents can delegate work to agents running elsewhere."
        className="mb-6"
      />
      <RemoteAgentsView remoteAgents={storyRemoteAgents} isLoading={false} loadError={null} onRefresh={async () => {}} />
    </AppPageFrame>
  ),
};

export const Empty: Story = {
  render: () => (
    <AppPageFrame ariaLabelledBy="remoteagents-page-title" mainClassName="mx-auto max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
      <PageHeader
        titleId="remoteagents-page-title"
        title="Remote Agents"
        description="Register external A2A endpoints so your local agents can delegate work to agents running elsewhere."
        className="mb-6"
      />
      <RemoteAgentsView remoteAgents={[]} isLoading={false} loadError={null} onRefresh={async () => {}} />
    </AppPageFrame>
  ),
};

export const Loading: Story = {
  render: () => (
    <AppPageFrame ariaLabelledBy="remoteagents-page-title" mainClassName="mx-auto max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
      <PageHeader
        titleId="remoteagents-page-title"
        title="Remote Agents"
        description="Register external A2A endpoints so your local agents can delegate work to agents running elsewhere."
        className="mb-6"
      />
      <RemoteAgentsView remoteAgents={[]} isLoading loadError={null} onRefresh={async () => {}} />
    </AppPageFrame>
  ),
};

export const LoadError: Story = {
  render: () => (
    <AppPageFrame ariaLabelledBy="remoteagents-page-title" mainClassName="mx-auto max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
      <PageHeader
        titleId="remoteagents-page-title"
        title="Remote Agents"
        description="Register external A2A endpoints so your local agents can delegate work to agents running elsewhere."
        className="mb-6"
      />
      <RemoteAgentsView remoteAgents={[]} isLoading={false} loadError="Could not reach cluster API." onRefresh={async () => {}} />
    </AppPageFrame>
  ),
};
