"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  LockKeyhole,
  Mail,
  ShieldCheck,
  TriangleAlert,
  UserRoundPlus,
} from "lucide-react";

import {
  Button,
  Card,
  Checkbox,
  Input,
  Label,
  PageHeader,
} from "@/components/ui";
import { SecurityNotice } from "@/components/ui/part2";
import { toast } from "@/components/ui/toaster";
import { inviteDeveloper } from "@/lib/api";
import {
  DEVELOPER_PERMISSIONS,
  type DeveloperPermission,
} from "@/lib/types";

export default function InviteDeveloperPage() {
  const router = useRouter();

  const [email, setEmail] = React.useState("");
  const [name, setName] = React.useState("");
  const [permissions, setPermissions] = React.useState<
    Set<DeveloperPermission>
  >(
    new Set([
      "payments.create",
      "payments.verify",
      "payments.view",
      "webhooks.receive",
    ])
  );
  const [sending, setSending] = React.useState(false);

  const safePerms = DEVELOPER_PERMISSIONS.filter(
    (p) => !("dangerous" in p && p.dangerous)
  );

  const dangerousPerms = DEVELOPER_PERMISSIONS.filter(
    (p) => "dangerous" in p && p.dangerous
  );

  function toggle(id: DeveloperPermission, checked: boolean) {
    setPermissions((current) => {
      const next = new Set(current);

      if (checked) next.add(id);
      else next.delete(id);

      return next;
    });
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();

    const normalizedEmail = email.trim().toLowerCase();

    if (!normalizedEmail) {
      toast(
        "warning",
        "Email required",
        "Enter the developer's email address."
      );
      return;
    }

    if (permissions.size === 0) {
      toast(
        "warning",
        "Choose permissions",
        "Select at least one permission for this developer."
      );
      return;
    }

    setSending(true);

    try {
      await inviteDeveloper({
        email: normalizedEmail,
        name: name.trim() || normalizedEmail.split("@")[0],
        permissions: Array.from(permissions),
      });

      toast(
        "success",
        "Invitation sent",
        `${normalizedEmail} will receive access instructions by email.`
      );

      router.push("/developers");
    } catch (error) {
      toast(
        "error",
        "Invitation failed",
        error instanceof Error
          ? error.message
          : "Unable to send the invitation."
      );
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="mx-auto max-w-5xl animate-fade-in space-y-6">
      <PageHeader
        breadcrumb={[
          { label: "Developers", href: "/developers" },
          { label: "Invite developer" },
        ]}
        title="Invite a developer"
        description="Give a developer the access they need to build with your payment infrastructure."
      />

      <form onSubmit={onSubmit}>
        <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_280px]">
          <div className="space-y-5">
            <Card>
              <div className="border-b border-slate-100 px-5 py-4 sm:px-6">
                <div className="flex items-center gap-3">
                  <span className="flex size-9 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
                    <UserRoundPlus className="size-4.5" />
                  </span>

                  <div>
                    <h2 className="text-sm font-semibold text-slate-900">
                      Developer details
                    </h2>
                    <p className="mt-0.5 text-xs text-slate-500">
                      Who should receive the invitation?
                    </p>
                  </div>
                </div>
              </div>

              <div className="grid gap-5 p-5 sm:grid-cols-2 sm:p-6">
                <div className="space-y-2">
                  <Label htmlFor="email">Email address</Label>
                  <Input
                    id="email"
                    type="email"
                    autoComplete="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="developer@example.com"
                    required
                  />
                  <p className="text-xs text-slate-400">
                    The invitation will be sent to this address.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="name">
                    Developer name{" "}
                    <span className="font-normal text-slate-400">
                      (optional)
                    </span>
                  </Label>
                  <Input
                    id="name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g. Ada Lovelace"
                  />
                  <p className="text-xs text-slate-400">
                    Helps your team identify this developer.
                  </p>
                </div>
              </div>

              <div className="border-t border-slate-100 px-5 py-4 sm:px-6">
                <div className="flex items-start gap-3 rounded-xl border border-brand-100 bg-brand-50/50 p-3.5">
                  <LockKeyhole className="mt-0.5 size-4 shrink-0 text-brand-600" />
                  <p className="text-xs leading-relaxed text-brand-900">
                    This invitation applies to your current merchant
                    account. The developer must accept the invitation
                    before their access becomes active.
                  </p>
                </div>
              </div>
            </Card>

            <Card>
              <div className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-100 px-5 py-4 sm:px-6">
                <div>
                  <h2 className="text-sm font-semibold text-slate-900">
                    Permissions
                  </h2>
                  <p className="mt-1 text-xs leading-relaxed text-slate-500">
                    Select only the capabilities this developer needs.
                  </p>
                </div>

                <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600">
                  {permissions.size} selected
                </span>
              </div>

              <div className="space-y-6 p-5 sm:p-6">
                <section>
                  <div className="mb-3">
                    <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                      Standard access
                    </h3>
                  </div>

                  <div className="grid gap-3 sm:grid-cols-2">
                    {safePerms.map((permission) => {
                      const selected = permissions.has(permission.id);

                      return (
                        <label
                          key={permission.id}
                          className={`flex cursor-pointer items-start gap-3 rounded-xl border p-4 transition-colors ${
                            selected
                              ? "border-brand-200 bg-brand-50/40"
                              : "border-slate-200 hover:border-slate-300"
                          }`}
                        >
                          <Checkbox
                            checked={selected}
                            onCheckedChange={(checked) =>
                              toggle(permission.id, Boolean(checked))
                            }
                          />

                          <span className="min-w-0 flex-1">
                            <span className="block text-sm font-medium text-slate-800">
                              {permission.label}
                            </span>
                            <span className="mt-1 block text-xs leading-relaxed text-slate-500">
                              {permission.id === "payments.create" &&
                                "Create new payment requests."}
                              {permission.id === "payments.verify" &&
                                "Verify payment status and references."}
                              {permission.id === "payments.view" &&
                                "Read payment details and records."}
                              {permission.id === "webhooks.receive" &&
                                "Access webhook events for permitted applications."}
                            </span>
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </section>

                {dangerousPerms.length > 0 && (
                  <section className="border-t border-slate-100 pt-5">
                    <div className="mb-3 flex items-start gap-2.5">
                      <TriangleAlert className="mt-0.5 size-4 shrink-0 text-amber-600" />
                      <div>
                        <h3 className="text-xs font-semibold uppercase tracking-wide text-amber-800">
                          Elevated permissions
                        </h3>
                        <p className="mt-1 text-xs leading-relaxed text-slate-500">
                          These permissions can change sensitive
                          account settings and require owner approval.
                        </p>
                      </div>
                    </div>

                    <div className="space-y-3">
                      {dangerousPerms.map((permission) => {
                        const selected = permissions.has(permission.id);

                        return (
                          <label
                            key={permission.id}
                            className={`flex cursor-pointer items-start gap-3 rounded-xl border p-4 transition-colors ${
                              selected
                                ? "border-amber-300 bg-amber-50/60"
                                : "border-slate-200 hover:border-amber-200"
                            }`}
                          >
                            <Checkbox
                              checked={selected}
                              onCheckedChange={(checked) =>
                                toggle(
                                  permission.id,
                                  Boolean(checked)
                                )
                              }
                            />

                            <span>
                              <span className="block text-sm font-medium text-slate-800">
                                {permission.label}
                              </span>
                              <span className="mt-1 block text-xs text-slate-500">
                                {permission.id === "providers.manage"
                                  ? "Manage payment provider connections."
                                  : "Create, manage, or revoke API keys."}
                              </span>
                            </span>
                          </label>
                        );
                      })}
                    </div>
                  </section>
                )}
              </div>
            </Card>

            <div className="flex flex-wrap items-center justify-between gap-3">
              <Link href="/developers">
                <Button variant="ghost">
                  <ArrowLeft className="size-4" />
                  Cancel
                </Button>
              </Link>

              <Button type="submit" disabled={sending || !email.trim()}>
                <Mail className="size-4" />
                {sending ? "Sending invitation…" : "Send invitation"}
                {!sending && <ArrowRight className="size-4" />}
              </Button>
            </div>
          </div>

          <aside className="space-y-4">
            <Card>
              <div className="p-5">
                <div className="flex size-10 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700">
                  <ShieldCheck className="size-5" />
                </div>

                <h3 className="mt-4 text-sm font-semibold text-slate-900">
                  Access stays scoped
                </h3>

                <p className="mt-2 text-xs leading-relaxed text-slate-500">
                  Developers authenticate with PayConnect API keys.
                  They cannot see or export provider credentials.
                </p>

                <div className="mt-4 space-y-3 border-t border-slate-100 pt-4">
                  {[
                    "Merchant-scoped access",
                    "Granular permissions",
                    "Access can be revoked",
                  ].map((item) => (
                    <div key={item} className="flex items-center gap-2">
                      <Check className="size-4 shrink-0 text-emerald-600" />
                      <span className="text-xs text-slate-600">
                        {item}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </Card>

            <SecurityNotice>
              Give each developer the minimum permissions required
              for their work. You can review their access later.
            </SecurityNotice>
          </aside>
        </div>
      </form>
    </div>
  );
}