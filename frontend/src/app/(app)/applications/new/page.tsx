"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Check,
  Copy,
  KeyRound,
  Lock,
  ShieldCheck,
  TriangleAlert,
  UserPlus,
} from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";

import {
  Button,
  Card,
  EnvironmentBadge,
  Input,
  Label,
  PageHeader,
  Radio,
  Select,
  Textarea,
} from "@/components/ui";
import { toast } from "@/components/ui/toaster";
import { useAsync } from "@/hooks/use-async";
import {
  createApplication,
  inviteDeveloper,
  listDevelopers,
  type CreatedKey,
} from "@/lib/api";
import type {
  Application,
  DeveloperPermission,
} from "@/lib/types";

const DEFAULT_PERMISSIONS: DeveloperPermission[] = [
  "payments.create",
  "payments.verify",
  "payments.view",
  "webhooks.receive",
];

type Scope = "general" | "developer";
type DeveloperMode = "existing" | "invite";

export default function NewApplicationPage() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const developers = useAsync(
    listDevelopers,
    [],
    ["developers"]
  );

  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [scope, setScope] = React.useState<Scope>("general");
  const [developerMode, setDeveloperMode] =
    React.useState<DeveloperMode>("existing");
  const [selectedAccessId, setSelectedAccessId] =
    React.useState("");
  const [inviteEmail, setInviteEmail] = React.useState("");
  const [inviteName, setInviteName] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);

  const [created, setCreated] = React.useState<{
    application: Application;
    keys: CreatedKey[];
    invitedDeveloper?: string;
  } | null>(null);

  const [copiedId, setCopiedId] =
    React.useState<string | null>(null);

  const hasValidName = name.trim().length > 0;

  const selectableDevelopers = (
    developers.data ?? []
  ).filter((dev) => dev.status !== "revoked");

  const selectedDeveloper =
    selectableDevelopers.find(
      (dev) => dev.accessId === selectedAccessId
    ) ?? null;

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!name.trim()) {
      toast(
        "warning",
        "Name required",
        "Give your application a recognizable name."
      );
      return;
    }

    let developerAccessId: string | undefined;
    let invitedEmail: string | undefined;

    if (scope === "developer") {
      if (developerMode === "existing") {
        if (!selectedAccessId) {
          toast(
            "warning",
            "Select a developer",
            "Choose who these keys belong to, or invite someone new."
          );
          return;
        }

        developerAccessId = selectedAccessId;
      } else {
        const email = inviteEmail.trim();

        if (!email) {
          toast(
            "warning",
            "Email required",
            "Enter the developer's email address to invite them."
          );
          return;
        }

        invitedEmail = email;
      }
    }

    setSubmitting(true);

    try {
      // Create the developer grant before creating its application keys.
      if (
        scope === "developer" &&
        developerMode === "invite" &&
        invitedEmail
      ) {
        const invitation = await inviteDeveloper({
          email: invitedEmail,
          name:
            inviteName.trim() ||
            invitedEmail.split("@")[0],
          permissions: DEFAULT_PERMISSIONS,
        });

        if (!invitation.accessId) {
          throw new Error(
            "The invitation was created without an access id."
          );
        }

        developerAccessId = invitation.accessId;
      }

      const result = await createApplication({
        name: name.trim(),
        description: description.trim(),
        keyScope: scope,
        developerAccessId,
      });

      toast(
        "success",
        "Application created",
        `${result.application.name} is ready to use.`
      );

      void queryClient.invalidateQueries({
        queryKey: ["applications"],
      });

      if (invitedEmail) {
        void queryClient.invalidateQueries({
          queryKey: ["developers"],
        });
      }

      setCreated({
        application: result.application,
        keys:
          result.keys.length > 0
            ? result.keys
            : [
                {
                  id: "test",
                  environment: "test",
                  scope,
                  secret: result.apiKey,
                },
              ],
        invitedDeveloper: invitedEmail,
      });
    } catch (error) {
      toast(
        "error",
        "Application creation failed",
        error instanceof Error
          ? error.message
          : "Please try again."
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function copySecret(key: CreatedKey) {
    try {
      await navigator.clipboard.writeText(key.secret);
      setCopiedId(key.id);

      setTimeout(() => {
        setCopiedId((current) =>
          current === key.id ? null : current
        );
      }, 1800);
    } catch {
      toast(
        "error",
        "Copy failed",
        "Select and copy the key manually."
      );
    }
  }

  if (created) {
    return (
      <div className="mx-auto max-w-3xl animate-fade-in space-y-6">
        <Card>
          <div className="p-6 sm:p-8">
            <div className="text-center">
              <div className="mx-auto flex size-12 items-center justify-center rounded-2xl border border-emerald-200 bg-emerald-50">
                <Check className="size-6 text-emerald-600" />
              </div>

              <h1 className="mt-4 text-xl font-semibold tracking-tight text-slate-900">
                Application created.
              </h1>

              <p className="mt-1.5 text-sm text-slate-500">
                <span className="font-medium text-slate-700">
                  {created.application.name}
                </span>{" "}
                is ready. We minted{" "}
                {created.keys.length === 1
                  ? "one key"
                  : "both a test and a production key"}{" "}
                for you.
              </p>
            </div>

            <div className="mt-7 space-y-3">
              {created.keys.map((key) => (
                <div
                  key={key.id}
                  className="rounded-xl border border-slate-200 bg-slate-50 p-4"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="text-[12px] font-medium text-slate-500">
                      {key.environment === "live"
                        ? "Production"
                        : "Test"}{" "}
                      API key
                    </p>

                    <EnvironmentBadge
                      environment={key.environment}
                    />

                    <span className="rounded-full border border-slate-200 bg-white px-2 py-0.5 text-[11px] text-slate-500">
                      {key.scope === "developer"
                        ? "Developer-specific"
                        : "General (shared)"}
                    </span>
                  </div>

                  <div className="mt-3 flex min-w-0 items-center gap-2">
                    <KeyRound className="size-4 shrink-0 text-brand-500" />

                    <code className="min-w-0 flex-1 truncate font-mono text-[13px] text-slate-800">
                      {key.secret}
                    </code>

                    <button
                      type="button"
                      onClick={() => copySecret(key)}
                      className="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-[12px] font-medium text-slate-600 transition-colors hover:bg-slate-100"
                    >
                      {copiedId === key.id ? (
                        <Check className="size-3.5 text-emerald-500" />
                      ) : (
                        <Copy className="size-3.5" />
                      )}
                      {copiedId === key.id ? "Copied" : "Copy"}
                    </button>
                  </div>
                </div>
              ))}
            </div>

            {created.invitedDeveloper && (
              <div className="mt-4 flex items-start gap-2.5 rounded-xl border border-brand-100 bg-brand-50/70 p-3.5">
                <UserPlus className="mt-0.5 size-4 shrink-0 text-brand-600" />
                <p className="text-[12.5px] leading-relaxed text-brand-900">
                  We invited <b>{created.invitedDeveloper}</b>.
                  These developer keys activate the moment the
                  invitation is accepted — and you can revoke them
                  later without touching your shared key.
                </p>
              </div>
            )}

            <div className="mt-4 flex items-start gap-2.5 rounded-xl border border-amber-200 bg-amber-50 p-3.5">
              <TriangleAlert className="mt-0.5 size-4 shrink-0 text-amber-600" />
              <p className="text-[12.5px] leading-relaxed text-amber-800">
                Copy these keys now — you won't be able to view
                the full secrets again.{" "}
                <b>
                  These are the keys your developers use. Never
                  share your provider secret.
                </b>
              </p>
            </div>

            <div className="mt-8 flex flex-wrap justify-end gap-2">
              <Button
                variant="secondary"
                onClick={() => router.push("/applications")}
              >
                Back to applications
              </Button>

              <Button
                onClick={() =>
                  router.push(
                    `/applications/${created.application.id}`
                  )
                }
              >
                Open application
              </Button>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl animate-fade-in space-y-6">
      <PageHeader
        breadcrumb={[
          { label: "Applications", href: "/applications" },
          { label: "New" },
        ]}
        title="Create application"
        description="Applications group payments, API keys, webhooks and logs for one product surface."
      />

      <Card>
        <form onSubmit={onSubmit} className="p-6 sm:p-8">
          <div className="space-y-1.5">
            <Label htmlFor="name">Application name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Ecommerce Store"
              required
            />
          </div>

          <div className="mt-5 space-y-1.5">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Online store payment integration"
            />
          </div>

          <div className="mt-7 rounded-2xl border border-slate-200/80 bg-slate-50/50 p-5">
            <div className="flex items-center gap-2">
              <KeyRound className="size-4 text-slate-400" />
              <p className="text-[13px] font-medium text-slate-700">
                Who owns these keys?
              </p>
            </div>

            <p className="mt-1 text-[12px] leading-relaxed text-slate-500">
              We create a test key and a production key either way.
              The scope decides who they belong to.
            </p>

            <div className="mt-4 space-y-2.5">
              <label className="flex cursor-pointer items-start gap-3 rounded-xl border border-slate-200 bg-white p-4 transition-colors hover:border-brand-200">
                <Radio
                  name="key-scope"
                  className="mt-0.5"
                  checked={scope === "general"}
                  onChange={() => setScope("general")}
                />

                <span>
                  <span className="flex items-center gap-1.5 text-[13.5px] font-medium text-slate-800">
                    <Lock className="size-3.5 text-slate-400" />
                    General — shared by your team
                  </span>

                  <span className="mt-1 block text-[12px] leading-relaxed text-slate-500">
                    Deploy it in the application's server
                    configuration. Revoking it affects everyone
                    using it.
                  </span>
                </span>
              </label>

              <label className="flex cursor-pointer items-start gap-3 rounded-xl border border-slate-200 bg-white p-4 transition-colors hover:border-brand-200">
                <Radio
                  name="key-scope"
                  className="mt-0.5"
                  checked={scope === "developer"}
                  onChange={() => setScope("developer")}
                />

                <span>
                  <span className="flex items-center gap-1.5 text-[13.5px] font-medium text-slate-800">
                    <ShieldCheck className="size-3.5 text-emerald-500" />
                    Developer-specific — one person
                  </span>

                  <span className="mt-1 block text-[12px] leading-relaxed text-slate-500">
                    Bound to one developer. You can revoke these
                    keys later without rotating your shared key.
                  </span>
                </span>
              </label>
            </div>

            {scope === "developer" && (
              <div className="mt-4 rounded-xl border border-dashed border-slate-200 bg-white p-4">
                <div className="flex flex-wrap gap-x-5 gap-y-3">
                  <label className="flex cursor-pointer items-center gap-2 text-[13px] text-slate-700">
                    <Radio
                      name="developer-mode"
                      checked={developerMode === "existing"}
                      onChange={() =>
                        setDeveloperMode("existing")
                      }
                    />
                    Existing developer
                  </label>

                  <label className="flex cursor-pointer items-center gap-2 text-[13px] text-slate-700">
                    <Radio
                      name="developer-mode"
                      checked={developerMode === "invite"}
                      onChange={() => setDeveloperMode("invite")}
                    />
                    Invite someone new
                  </label>
                </div>

                {developerMode === "existing" ? (
                  <div className="mt-4 space-y-1.5">
                    <Label htmlFor="developer">Developer</Label>

                    <Select
                      id="developer"
                      value={selectedAccessId}
                      onChange={(e) =>
                        setSelectedAccessId(e.target.value)
                      }
                    >
                      <option value="">
                        Select a developer…
                      </option>

                      {selectableDevelopers.map((dev) => (
                        <option
                          key={dev.accessId}
                          value={dev.accessId}
                        >
                          {dev.name} · {dev.email}
                          {dev.status === "pending"
                            ? " (invite pending)"
                            : ""}
                        </option>
                      ))}
                    </Select>

                    {selectableDevelopers.length === 0 && (
                      <p className="text-[12px] text-slate-500">
                        No developers yet — switch to{" "}
                        <b>Invite someone new</b> to add one.
                      </p>
                    )}
                  </div>
                ) : (
                  <div className="mt-4 grid gap-4 sm:grid-cols-2">
                    <div className="space-y-1.5">
                      <Label htmlFor="invite-email">
                        Email address
                      </Label>
                      <Input
                        id="invite-email"
                        type="email"
                        value={inviteEmail}
                        onChange={(e) =>
                          setInviteEmail(e.target.value)
                        }
                        placeholder="developer@example.com"
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="invite-name">Name</Label>
                      <Input
                        id="invite-name"
                        value={inviteName}
                        onChange={(e) =>
                          setInviteName(e.target.value)
                        }
                        placeholder="Ada Lovelace"
                      />
                    </div>
                  </div>
                )}

                {developerMode === "existing" &&
                  selectedDeveloper && (
                    <p className="mt-3 text-[12px] text-slate-500">
                      {selectedDeveloper.permissions.length}{" "}
                      permission
                      {selectedDeveloper.permissions.length === 1
                        ? ""
                        : "s"}{" "}
                      · {selectedDeveloper.applicationIds.length}{" "}
                      application
                      {selectedDeveloper.applicationIds.length === 1
                        ? ""
                        : "s"}
                    </p>
                  )}

                {developerMode === "invite" && (
                  <p className="mt-3 text-[12px] leading-relaxed text-slate-500">
                    The developer gets the safe default permissions:
                    create, verify and view payments, and receive
                    webhooks. You can change them later.
                  </p>
                )}
              </div>
            )}
          </div>

          <div className="mt-6 flex items-center justify-between">
            <Link href="/applications">
              <Button variant="ghost">Cancel</Button>
            </Link>

            <Button
              type="submit"
              disabled={submitting || !hasValidName}
            >
              {submitting ? "Creating…" : "Create application"}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}