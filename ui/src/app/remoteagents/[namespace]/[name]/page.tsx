"use client";

import { use, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Trash2 } from "lucide-react";
import { AppPageFrame } from "@/components/layout/AppPageFrame";
import { PageHeader } from "@/components/layout/PageHeader";
import { LoadingState } from "@/components/LoadingState";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { Button } from "@/components/ui/button";
import { getRemoteAgent, deleteRemoteAgent } from "@/app/actions/remoteagents";
import { toast } from "sonner";
import type { RemoteAgent, RemoteAgentStatus } from "@/types";

function ConditionBadge({ conditions, type }: { conditions?: RemoteAgentStatus["conditions"]; type: string }) {
  const cond = conditions?.find((c) => c.type === type);
  if (!cond) return <span className="text-xs text-muted-foreground">—</span>;
  const isTrue = cond.status === "True";
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
        isTrue
          ? "bg-green-500/15 text-green-600 dark:text-green-400"
          : "bg-red-500/15 text-red-600 dark:text-red-400"
      }`}
    >
      {isTrue ? type : `Not ${type}`}
    </span>
  );
}

function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 py-3 sm:flex-row sm:gap-4">
      <dt className="w-36 shrink-0 text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className="text-sm text-foreground">{children}</dd>
    </div>
  );
}

export default function RemoteAgentDetailPage({
  params,
}: {
  params: Promise<{ namespace: string; name: string }>;
}) {
  const { namespace, name } = use(params);
  const router = useRouter();
  const [agent, setAgent] = useState<RemoteAgent | null>(null);
  const [loading, setLoading] = useState(true);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      const res = await getRemoteAgent(namespace, name);
      if (cancelled) return;
      if (res.error || !res.data) {
        toast.error(res.error || "Could not load remote agent");
        setLoading(false);
        return;
      }
      setAgent(res.data);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
    };
  }, [namespace, name]);

  const handleDelete = async () => {
    setDeleting(true);
    const res = await deleteRemoteAgent(namespace, name);
    setDeleting(false);
    if (res.error) {
      toast.error(res.error || "Failed to delete remote agent");
      return;
    }
    toast.success("Remote agent deleted");
    router.push("/remoteagents");
  };

  if (loading) {
    return (
      <AppPageFrame mainClassName="mx-auto max-w-3xl px-4 py-10 sm:px-6">
        <div className="relative" role="status" aria-live="polite" aria-busy="true">
          <span className="sr-only">Loading remote agent…</span>
          <LoadingState />
        </div>
      </AppPageFrame>
    );
  }

  if (!agent) {
    return (
      <AppPageFrame
        ariaLabelledBy="remoteagent-detail-title"
        mainClassName="mx-auto max-w-3xl px-4 py-10 sm:px-6"
      >
        <Link
          href="/remoteagents"
          className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Back to Remote Agents
        </Link>
        <p className="mt-8 text-center text-sm text-muted-foreground">Remote agent not found.</p>
      </AppPageFrame>
    );
  }

  const conditions = agent.status?.conditions;

  return (
    <AppPageFrame
      ariaLabelledBy="remoteagent-detail-title"
      mainClassName="mx-auto max-w-3xl px-4 py-8 sm:px-6 sm:py-10"
    >
      <Link
        href="/remoteagents"
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 rounded-sm"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Back to Remote Agents
      </Link>

      <PageHeader
        titleId="remoteagent-detail-title"
        title={agent.metadata.name}
        isMonospaceTitle
        description={agent.spec.description}
        className="mt-4 mb-6"
        end={
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="text-destructive hover:bg-destructive/10 hover:text-destructive border-destructive/40"
            onClick={() => setConfirmOpen(true)}
            disabled={deleting}
            aria-label={`Delete remote agent ${agent.metadata.name}`}
          >
            <Trash2 className="mr-2 h-4 w-4" aria-hidden />
            Delete
          </Button>
        }
      />

      <div className="rounded-xl border border-border/60 bg-card/30 px-4 shadow-sm divide-y divide-border/40">
        <dl>
          <DetailRow label="Name">{agent.metadata.name}</DetailRow>
          <DetailRow label="Namespace">{agent.metadata.namespace ?? "default"}</DetailRow>
          <DetailRow label="URL">
            <span className="font-mono text-xs break-all">{agent.spec.url}</span>
          </DetailRow>
          {agent.spec.timeout && (
            <DetailRow label="Timeout">{agent.spec.timeout}</DetailRow>
          )}
          {agent.status?.agentName && (
            <DetailRow label="Agent Name">{agent.status.agentName}</DetailRow>
          )}
          {agent.status?.agentDescription && (
            <DetailRow label="Agent Description">{agent.status.agentDescription}</DetailRow>
          )}
          <DetailRow label="Accepted">
            <ConditionBadge conditions={conditions} type="Accepted" />
            {conditions?.find((c) => c.type === "Accepted")?.message && (
              <span className="ml-2 text-xs text-muted-foreground">
                {conditions.find((c) => c.type === "Accepted")?.message}
              </span>
            )}
          </DetailRow>
          <DetailRow label="Reachable">
            <ConditionBadge conditions={conditions} type="Reachable" />
            {conditions?.find((c) => c.type === "Reachable")?.message && (
              <span className="ml-2 text-xs text-muted-foreground">
                {conditions.find((c) => c.type === "Reachable")?.message}
              </span>
            )}
          </DetailRow>
        </dl>
      </div>

      {agent.status?.agentCard && (
        <div className="mt-6">
          <h2 className="mb-2 text-sm font-semibold text-foreground">Agent Card</h2>
          <pre className="overflow-x-auto rounded-xl border border-border/60 bg-muted/40 p-4 text-xs leading-relaxed text-muted-foreground">
            {(() => {
              try {
                return JSON.stringify(JSON.parse(agent.status!.agentCard!), null, 2);
              } catch {
                return agent.status!.agentCard;
              }
            })()}
          </pre>
        </div>
      )}

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={(open) => {
          if (!open) setConfirmOpen(false);
        }}
        title="Delete remote agent"
        description="This will remove the remote agent. Any agent tool bindings referencing it may break until updated."
        confirmLabel="Delete"
        onConfirm={() => void handleDelete()}
      />
    </AppPageFrame>
  );
}
