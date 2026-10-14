"use client";

import * as React from "react";

import Link from "next/link";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { ChevronLeft, ChevronRight, PanelsTopLeft, Plus, Trash2 } from "lucide-react";

import { Button, Card, EmptyState, PageHeader } from "@/components/ui";

import { Modal } from "@/components/ui/part2";

import { toast } from "@/components/ui/toaster";

import { useAsync } from "@/hooks/use-async";

import { deleteApplication, listApplications } from "@/lib/api";

import { cn, formatNumber, timeAgo } from "@/lib/utils";

import type { Application } from "@/lib/types";

/** Applications per page. */

const PAGE_SIZE = 10;

export default function ApplicationsPage() {

  const queryClient = useQueryClient();

  const { data, loading } = useAsync(listApplications, [], ["applications"]);

  const applications = data ?? [];

  const pageCount = Math.max(1, Math.ceil(applications.length / PAGE_SIZE));

  const [page, setPage] = React.useState(0);

  const [pendingDelete, setPendingDelete] = React.useState<Application | null>(null);

  // Keep the current page in range when rows are added or removed.

  React.useEffect(() => {

    if (page > pageCount - 1) setPage(pageCount - 1);

  }, [page, pageCount]);

  const start = page * PAGE_SIZE;

  const pageItems = applications.slice(start, start + PAGE_SIZE);

  const remove = useMutation({

    mutationFn: (id: string) => deleteApplication(id),

    onSuccess: async () => {

      setPendingDelete(null);

      toast("success", "Application deleted", "Its API keys were revoked and webhooks stopped.");

      await queryClient.invalidateQueries({ queryKey: ["applications"] });

    },

    onError: (error) => {

      toast("error", "Delete failed", error instanceof Error ? error.message : "Please try again.");

    },

  });

  const skeleton = (

    <div className="grid gap-5 md:grid-cols-2">

      {Array.from({ length: 4 }).map((_, i) => (

        <div key={i} className="h-44 animate-pulse rounded-[var(--radius-card)] border border-slate-200 bg-white" />

      ))}

    </div>

  );

  return (

    <div className="animate-fade-in space-y-6">

      <PageHeader

        title="Applications"

        description="Each application groups payments, webhooks, logs and developer access, with separate test and production API keys."

        actions={

          <Link href="/applications/new">

            <Button size="sm">

              <Plus className="size-4" /> Create Application

            </Button>

          </Link>

        }

      />

      {loading ? (

        skeleton

      ) : applications.length === 0 ? (

        <Card>

          <EmptyState

            icon={<PanelsTopLeft className="size-5" />}

            title="No applications yet"

            description="Create an application to generate your first API key."

            action={

              <Link href="/applications/new">

                <Button>Create Application</Button>

              </Link>

            }

          />

        </Card>

      ) : (

        <>

          <div className="grid gap-5 md:grid-cols-2">

            {pageItems.map((app) => (

              <div key={app.id} className="group relative min-w-0">

                <Link href={`/applications/${app.id}`} className="block h-full">

                  <Card className="h-full overflow-hidden border-slate-200/80 bg-white transition-all duration-200 hover:-translate-y-0.5 hover:border-brand-200 hover:shadow-lg group-hover:shadow-[var(--shadow-lift)]">

                    <div className="p-5 sm:p-6">

                      <div className="flex items-start justify-between">

                        <div className="flex items-center gap-3">

                          <span className="flex size-10 items-center justify-center rounded-xl bg-brand-50 text-brand-500">

                            <PanelsTopLeft className="size-5" />

                          </span>

                          <div>

                            <p className="text-[15px] font-semibold text-slate-900 group-hover:text-brand-600">

                              {app.name}

                            </p>

                            <div className="mt-1 flex items-center gap-2">

                              <span

                                className={cn(

                                  "inline-flex items-center gap-1.5 text-[12px] font-medium",

                                  app.status === "paused" ? "text-amber-600" : "text-emerald-600"

                                )}

                              >

                                <span

                                  className={cn(

                                    "size-1.5 rounded-full",

                                    app.status === "paused" ? "bg-amber-500" : "bg-emerald-500"

                                  )}

                                />

                                {app.status === "paused" ? "Paused" : "Active"}

                              </span>

                            </div>

                          </div>

                        </div>

                        {/* Reserve space so the title never slides under the delete control. */}

                        <span className="w-8 shrink-0" aria-hidden />

                      </div>

                      <p className="mt-3 line-clamp-2 text-[12.5px] text-slate-400">{app.description}</p>

                      <dl className="mt-4 grid grid-cols-3 gap-3 border-t border-slate-100 pt-4">

                        <div>

                          <dt className="text-[11px] text-slate-400">API requests</dt>

                          <dd className="mt-0.5 font-mono text-[13px] font-medium text-slate-900">

                            {formatNumber(app.stats.apiRequests)}

                          </dd>

                        </div>

                        <div>

                          <dt className="text-[11px] text-slate-400">Successful</dt>

                          <dd className="mt-0.5 font-mono text-[13px] font-medium text-slate-900">

                            {formatNumber(app.stats.successfulPayments)}

                          </dd>

                        </div>

                        <div>

                          <dt className="text-[11px] text-slate-400">Success rate</dt>

                          <dd className="mt-0.5 text-[13px] font-medium text-emerald-600">{app.stats.successRate}%</dd>

                        </div>

                      </dl>

                      <p className="mt-3 text-[11.5px] text-slate-400">

                        Last used {app.lastUsedAt ? timeAgo(app.lastUsedAt) : "never"}

                      </p>

                    </div>

                  </Card>

                </Link>

                <button

                  type="button"

                  aria-label={`Delete ${app.name}`}

                  title="Delete application"

                  onClick={() => setPendingDelete(app)}

                  className="absolute right-3 top-3 z-10 inline-flex size-8 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-400 transition-colors hover:border-red-200 hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500/30"

                >

                  <Trash2 className="size-4" />

                </button>

              </div>

            ))}

          </div>

          <div className="mt-5 flex flex-wrap items-center justify-between gap-3">

            <p className="text-xs text-slate-400">

              Showing {start + 1}–{Math.min(start + PAGE_SIZE, applications.length)} of {applications.length}{" "}

              application{applications.length === 1 ? "" : "s"}

            </p>

            {pageCount > 1 ? (

              <div className="flex items-center gap-1.5">

                <Button

                  variant="secondary"

                  size="sm"

                  disabled={page === 0}

                  onClick={() => setPage((p) => Math.max(0, p - 1))}

                  aria-label="Previous page"

                >

                  <ChevronLeft className="size-4" /> Previous

                </Button>

                <span className="px-2 text-xs text-slate-500">

                  Page {page + 1} of {pageCount}

                </span>

                <Button

                  variant="secondary"

                  size="sm"

                  disabled={page >= pageCount - 1}

                  onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}

                  aria-label="Next page"

                >

                  Next <ChevronRight className="size-4" />

                </Button>

              </div>

            ) : null}

          </div>

        </>

      )}

      <Modal

        open={pendingDelete !== null}

        onClose={() => {

          if (!remove.isPending) setPendingDelete(null);

        }}

        title={`Delete ${pendingDelete?.name ?? "application"}?`}

        description="This revokes its API keys immediately and stops all webhooks. Payment history is preserved, but the application cannot be restored."

        footer={

          <>

            <Button variant="secondary" onClick={() => setPendingDelete(null)} disabled={remove.isPending}>

              Cancel

            </Button>

            <Button

              variant="dangerSolid"

              disabled={remove.isPending}

              onClick={() => pendingDelete && remove.mutate(pendingDelete.id)}

            >

              <Trash2 className="size-4" /> {remove.isPending ? "Deleting…" : "Delete application"}

            </Button>

          </>

        }

      >

        <p className="text-[13px] leading-relaxed text-slate-600">

          Make sure you have a copy of any configuration you rely on. Deleted applications cannot be restored.

        </p>

      </Modal>

    </div>

  );

}
