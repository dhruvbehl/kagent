"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Loader2, AlertCircle } from "lucide-react";
import type { RemoteAgent } from "@/types";
import { isResourceNameValid } from "@/lib/utils";

export interface RemoteAgentFormProps {
  onCreate: (data: RemoteAgent) => Promise<void>;
  defaultNamespace?: string;
}

export function RemoteAgentForm({ onCreate, defaultNamespace = "kagent" }: RemoteAgentFormProps) {
  const [formName, setFormName] = useState("");
  const [formNamespace, setFormNamespace] = useState(defaultNamespace);
  const [formUrl, setFormUrl] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formTimeout, setFormTimeout] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitError, setSubmitError] = useState<string | null>(null);

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formName.trim()) {
      newErrors.name = "Name is required";
    } else if (!isResourceNameValid(formName.trim())) {
      newErrors.name =
        "Name must conform to RFC 1123 subdomain format (lowercase alphanumeric, '-' or '.', must start and end with alphanumeric)";
    }

    if (!formUrl.trim()) {
      newErrors.url = "URL is required";
    } else if (!/^https?:\/\//i.test(formUrl.trim())) {
      newErrors.url = "URL must start with http:// or https://";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitError(null);

    if (!validate()) {
      return;
    }

    const remoteAgent: RemoteAgent = {
      apiVersion: "kagent.dev/v1alpha2",
      kind: "RemoteAgent",
      metadata: {
        name: formName.trim(),
        namespace: formNamespace.trim() || "kagent",
      },
      spec: {
        url: formUrl.trim(),
        ...(formDescription.trim() && { description: formDescription.trim() }),
        ...(formTimeout.trim() && { timeout: formTimeout.trim() }),
      },
    };

    setSubmitting(true);
    try {
      await onCreate(remoteAgent);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error occurred";
      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6" noValidate>
      {submitError ? (
        <Alert variant="destructive" role="alert">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Could not create remote agent</AlertTitle>
          <AlertDescription>{submitError}</AlertDescription>
        </Alert>
      ) : null}

      <div className="space-y-4">
        {/* Name */}
        <div className="space-y-2">
          <Label htmlFor="remote-agent-name">
            Name <span aria-hidden className="text-destructive">*</span>
          </Label>
          <Input
            id="remote-agent-name"
            placeholder="my-remote-agent"
            value={formName}
            onChange={(e) => setFormName(e.target.value)}
            className={errors.name ? "border-destructive" : ""}
            aria-describedby={errors.name ? "remote-agent-name-error" : undefined}
            aria-invalid={!!errors.name}
          />
          {errors.name ? (
            <p id="remote-agent-name-error" className="text-xs text-destructive">
              {errors.name}
            </p>
          ) : null}
        </div>

        {/* Namespace */}
        <div className="space-y-2">
          <Label htmlFor="remote-agent-namespace">Namespace</Label>
          <Input
            id="remote-agent-namespace"
            placeholder="kagent"
            value={formNamespace}
            onChange={(e) => setFormNamespace(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">Kubernetes namespace for this resource. Defaults to &quot;kagent&quot;.</p>
        </div>

        {/* URL */}
        <div className="space-y-2">
          <Label htmlFor="remote-agent-url">
            URL <span aria-hidden className="text-destructive">*</span>
          </Label>
          <Input
            id="remote-agent-url"
            type="url"
            placeholder="https://agent.example.com"
            value={formUrl}
            onChange={(e) => setFormUrl(e.target.value)}
            className={errors.url ? "border-destructive" : ""}
            aria-describedby={errors.url ? "remote-agent-url-error" : undefined}
            aria-invalid={!!errors.url}
          />
          {errors.url ? (
            <p id="remote-agent-url-error" className="text-xs text-destructive">
              {errors.url}
            </p>
          ) : null}
        </div>

        {/* Description */}
        <div className="space-y-2">
          <Label htmlFor="remote-agent-description">Description</Label>
          <Input
            id="remote-agent-description"
            placeholder="What this remote agent does"
            value={formDescription}
            onChange={(e) => setFormDescription(e.target.value)}
          />
        </div>

        {/* Timeout */}
        <div className="space-y-2">
          <Label htmlFor="remote-agent-timeout">Timeout</Label>
          <Input
            id="remote-agent-timeout"
            placeholder="30s"
            value={formTimeout}
            onChange={(e) => setFormTimeout(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">e.g. 30s, 1m</p>
        </div>
      </div>

      <div className="flex flex-col gap-3 border-t border-border/50 pt-6 sm:flex-row sm:items-center sm:justify-between">
        <Button type="button" variant="outline" asChild className="w-full sm:w-auto" disabled={submitting}>
          <Link href="/remoteagents">Cancel</Link>
        </Button>
        <Button
          type="submit"
          size="lg"
          className="min-w-[12rem] w-full sm:w-auto"
          disabled={submitting}
          aria-busy={submitting}
        >
          {submitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 shrink-0 animate-spin" aria-hidden />
              Creating…
            </>
          ) : (
            "Create Remote Agent"
          )}
        </Button>
      </div>
    </form>
  );
}
