import { Sidebar } from "@/components/app/sidebar";
import { Topbar } from "@/components/app/topbar";
import { BottomNav } from "@/components/app/bottom-nav";
import { RequireAuth } from "@/components/app/require-auth";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <RequireAuth>
      <div className="min-h-screen bg-[#F8FAFC]">
        <div className="flex">
          {/* Desktop sidebar */}
          <div className="sticky top-0 hidden h-screen lg:block">
            <Sidebar />
          </div>

          <div className="flex min-w-0 flex-1 flex-col">
            <Topbar />
            <main className="flex-1 px-3 pb-24 pt-6 sm:px-4 lg:px-5 lg:pb-12">
              <div className="mx-auto max-w-6xl">{children}</div>
            </main>
          </div>
        </div>
        <BottomNav />
      </div>
    </RequireAuth>
  );
}
