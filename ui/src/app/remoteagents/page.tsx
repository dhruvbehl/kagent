"use client";

import { useCallback, useEffect, useState } from "react";
import { AppPageFrame } from "@/components/layout/AppPageFrame";
import { PageHeader } from "@/components/layout/PageHeader";
import { RemoteAgentsView } from "@/components/remoteagents/RemoteAgentsView";
import { getRemoteAgents } from "@/app/actions/remoteagents";
import type { RemoteAgent } from "@/types";

export default function RemoteAgentsPage() {
  const [remoteAgents, setRemoteAgents] = useState<RemoteAgent[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    const res = await getRemoteAgents();
    if (res.error) {
      setLoadError(res.error);
      setRemoteAgents([]);
    } else {
      const sorted = [...(res.data ?? [])].sort((a, b) => a.metadata.name.localeCompare(b.metadata.name));
      setRemoteAgents(sorted);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    const raf = requestAnimationFrame(() => {
      void load();
    });
    return () => cancelAnimationFrame(raf);
  }, [load]);

  return (
    <AppPageFrame
      ariaLabelledBy="remoteagents-page-title"
      mainClassName="mx-auto max-w-6xl px-4 py-8 sm:px-6 sm:py-10"
    >
      <PageHeader
        titleId="remoteagents-page-title"
        title="Remote Agents"
        description="Manage A2A endpoints that agents can delegate work to."
        className="mb-6"
      />
      <RemoteAgentsView
        remoteAgents={remoteAgents}
        isLoading={loading}
        loadError={loadError}
        onRefresh={load}
      />
    </AppPageFrame>
  );
}
