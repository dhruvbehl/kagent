import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { AgentsContext } from "@/components/AgentsProvider";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { AppPageFrame } from "@/components/layout/AppPageFrame";
import { PageHeader } from "@/components/layout/PageHeader";
import { RemoteAgentForm } from "@/components/remoteagents/RemoteAgentForm";
import { createStoryAgentsContext } from "./fixtures";

const meta = {
  title: "Pages/Create/Remote Agent",
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "`/remoteagents/new` — creation form shell (no create API).",
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

export const Form: Story = {
  render: () => (
    <AppPageFrame ariaLabelledBy="remoteagent-new-title" mainClassName="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <div>
        <Link
          href="/remoteagents"
          className="mb-8 inline-flex items-center gap-2 rounded-sm text-sm text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Back to Remote Agents
        </Link>
        <PageHeader titleId="remoteagent-new-title" title="New Remote Agent" className="mb-8" />
        <RemoteAgentForm onCreate={async () => {}} />
      </div>
    </AppPageFrame>
  ),
};
