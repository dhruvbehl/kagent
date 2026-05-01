"use client";

import { useState } from "react";
import Link from "next/link";
import { Plus, Trash2, Globe } from "lucide-react";
import { Button } from "@/components/ui/button";
import { RemoteAgent, RemoteAgentStatus } from "@/types";
import { deleteRemoteAgent } from "@/app/actions/remoteagents";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { LoadingState } from "@/components/LoadingState";
import { toast } from "sonner";

interface RemoteAgentsViewProps {
  remoteAgents: RemoteAgent[];
  isLoading: boolean;
  loadError: string | null;
  onRefresh: () => void;
}

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

export function RemoteAgentsView({ remoteAgents, isLoading, loadError, onRefresh }: RemoteAgentsViewProps) {
  const [deleteTarget, setDeleteTarget] = useState<{ namespace: string; name: string } | null>(null);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    const res = await deleteRemoteAgent(deleteTarget.namespace, deleteTarget.name);
    if (res.error) {
      toast.error(res.error || "Failed to delete remote agent");
    } else {
      toast.success("Remote agent deleted");
      onRefresh();
    }
    setDeleteTarget(null);
  };

  if (isLoading) {
    return <LoadingState />;
  }

  if (loadError) {
    return (
      <div className="flex min-h-[200px] flex-col items-center justify-center rounded-xl border border-destructive/40 bg-destructive/5 p-8 text-center">
        <p className="text-sm font-medium text-destructive">{loadError}</p>
        <Button type="button" variant="outline" size="sm" className="mt-4" onClick={onRefresh}>
          Retry
        </Button>
      </div>
    );
  }

  if (remoteAgents.length === 0) {
    return (
      <div className="flex h-[min(40vh,320px)] flex-col items-center justify-center rounded-xl border border-dashed border-border/60 bg-card/20 p-6 text-center shadow-sm">
        <Globe className="mb-4 h-12 w-12 text-muted-foreground opacity-20" aria-hidden />
        <h2 className="text-lg font-medium tracking-tight">No remote agents yet</h2>
        <p className="mb-4 mt-1 max-w-sm text-pretty text-sm text-muted-foreground">
          Register a remote A2A endpoint so your local agents can delegate work to agents running elsewhere.
        </p>
        <Button asChild type="button" size="lg">
          <Link href="/remoteagents/new" className="inline-flex">
            <Plus className="mr-2 h-4 w-4" aria-hidden />
            New Remote Agent
          </Link>
        </Button>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-4 flex justify-end">
        <Button asChild size="lg">
          <Link href="/remoteagents/new" className="inline-flex">
            <Plus className="mr-2 h-4 w-4" aria-hidden />
            New Remote Agent
          </Link>
        </Button>
      </div>

      <div className="overflow-hidden rounded-xl border border-border/60 bg-card/30 shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border/60 bg-muted/30">
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">URL</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Accepted</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Reachable</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">Actions</th>
            </tr>
          </thead>
          <tbody>
            {remoteAgents.map((ra) => {
              const ns = ra.metadata.namespace ?? "default";
              const name = ra.metadata.name;
              return (
                <tr key={`${ns}/${name}`} className="border-b border-border/40 last:border-0 hover:bg-muted/20 transition-colors">
                  <td className="px-4 py-3">
                    <Link
                      href={`/remoteagents/${ns}/${name}`}
                      className="font-medium text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 rounded-sm"
                      translate="no"
                    >
                      {name}
                    </Link>
                    {ns !== "default" && (
                      <span className="ml-2 text-xs text-muted-foreground">{ns}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground font-mono text-xs break-all max-w-xs">
                    {ra.spec.url}
                  </td>
                  <td className="px-4 py-3">
                    <ConditionBadge conditions={ra.status?.conditions} type="Accepted" />
                  </td>
                  <td className="px-4 py-3">
                    <ConditionBadge conditions={ra.status?.conditions} type="Reachable" />
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-muted-foreground hover:text-destructive"
                      aria-label={`Delete remote agent ${name}`}
                      onClick={() => setDeleteTarget({ namespace: ns, name })}
                    >
                      <Trash2 className="h-4 w-4" aria-hidden />
                    </Button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        title="Delete remote agent"
        description="This will remove the remote agent. Any agent tool bindings referencing it may break until updated."
        confirmLabel="Delete"
        onConfirm={() => void handleDelete()}
      />
    </div>
  );
}
