"use client";

import { Suspense, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft, Loader2 } from "lucide-react";
import { AppPageFrame } from "@/components/layout/AppPageFrame";
import { PageHeader } from "@/components/layout/PageHeader";
import { RemoteAgentForm } from "@/components/remoteagents/RemoteAgentForm";
import { createRemoteAgent } from "@/app/actions/remoteagents";
import type { RemoteAgent } from "@/types";

function NewRemoteAgentContent() {
  const router = useRouter();

  const onCreate = useCallback(
    async (data: RemoteAgent) => {
      const r = await createRemoteAgent(data);
      if (r.error) throw new Error(r.error);
      toast.success("Remote Agent created");
      router.push("/remoteagents");
    },
    [router],
  );

  return (
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

        <RemoteAgentForm onCreate={onCreate} />
      </div>
    </AppPageFrame>
  );
}

export default function NewRemoteAgentPage() {
  return (
    <Suspense
      fallback={
        <AppPageFrame mainClassName="mx-auto max-w-3xl px-4 py-20 sm:px-6">
          <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground" role="status" aria-live="polite">
            <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
            Loading…
          </div>
        </AppPageFrame>
      }
    >
      <NewRemoteAgentContent />
    </Suspense>
  );
}
