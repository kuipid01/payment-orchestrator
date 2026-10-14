"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { FlaskConical, Globe, KeyRound, LayoutDashboard, Receipt } from "lucide-react";
import { cn } from "@/lib/utils";

const ITEMS = [
  { href: "/dashboard", label: "Home", icon: LayoutDashboard },
  { href: "/payments", label: "Payments", icon: Receipt },
  { href: "/providers", label: "Providers", icon: Globe },
  { href: "/api-keys", label: "Keys", icon: KeyRound },
  { href: "/test-arena", label: "Arena", icon: FlaskConical },
];

export function BottomNav() {
  const pathname = usePathname();

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-40 border-t border-slate-200 bg-white/95 backdrop-blur lg:hidden">
      <div className="grid grid-cols-5">
        {ITEMS.map((item) => {
          const active = pathname === item.href || pathname.startsWith(item.href + "/");
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex flex-col items-center gap-1 py-2.5 text-[10.5px] font-medium transition-colors",
                active ? "text-brand-600" : "text-slate-400 hover:text-slate-600"
              )}
            >
              <span
                className={cn(
                  "flex size-8 items-center justify-center rounded-full transition-colors",
                  active ? "bg-brand-500 text-slate-950" : ""
                )}
              >
                <Icon className="size-4.5" />
              </span>
              {item.label}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
