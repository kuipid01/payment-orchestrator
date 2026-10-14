import Link from "next/link";
import { ChevronRight } from "lucide-react";
import { Logo } from "@/components/ui/logo";

const NAV = [
  { href: "/docs", label: "Introduction" },
  { href: "/docs/quickstart", label: "Quickstart" },
  { href: "/docs/authentication", label: "Authentication" },
  { href: "/docs/payments", label: "Payments" },
  { href: "/docs/webhooks", label: "Webhooks" },
  { href: "/docs/providers", label: "Providers" },
  { href: "/docs/sdks", label: "SDKs" },
  { href: "/docs/errors", label: "Errors" },
];

export default function DocsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="mx-auto max-w-6xl px-5 lg:px-8">
      <div className="flex gap-10">
        {/* Left nav */}
        <aside className="sticky top-16 hidden h-[calc(100vh-4rem)] w-56 shrink-0 overflow-y-auto border-r border-slate-100 py-10 pr-6 lg:block">
          <Link href="/" className="mb-6 flex items-center gap-2 text-[13px] text-slate-400 hover:text-slate-600">
            <Logo mark /> <span>PayConnect Docs</span>
          </Link>
          <p className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wider text-slate-400">Documentation</p>
          <nav className="flex flex-col gap-0.5">
            {NAV.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className="rounded-lg px-2 py-1.5 text-[13.5px] text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
              >
                {item.label}
              </Link>
            ))}
          </nav>
        </aside>

        {/* Content */}
        <div className="min-w-0 flex-1 py-10">{children}</div>

        {/* Mobile docs nav */}
        <div className="fixed inset-x-0 bottom-0 z-30 border-t border-slate-200 bg-white p-3 lg:hidden">
          <div className="flex gap-1.5 overflow-x-auto">
            {NAV.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className="whitespace-nowrap rounded-full border border-slate-200 px-3 py-1.5 text-[12.5px] text-slate-600"
              >
                {item.label}
              </Link>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
