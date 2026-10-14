import Link from "next/link";
import { ChevronRight } from "lucide-react";
import { CodeBlock } from "@/components/ui";

export interface DocSection {
  id: string;
  title: string;
  body?: string[];
  code?: { language: string; code: string };
  list?: string[];
  note?: string;
}

export function DocsPage({
  title,
  lede,
  sections,
}: {
  title: string;
  lede: string;
  sections: DocSection[];
}) {
  return (
    <article className="pb-16">
      <nav className="mb-6 flex items-center gap-1.5 text-[12.5px] text-slate-400">
        <Link href="/docs" className="hover:text-slate-600">
          Docs
        </Link>
        <ChevronRight className="size-3.5" />
        <span className="text-slate-600">{title}</span>
      </nav>

      <div className="flex gap-12">
        <div className="min-w-0 max-w-2xl flex-1">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-900">{title}</h1>
          <p className="mt-3 text-[15px] leading-relaxed text-slate-500">{lede}</p>

          <div className="mt-10 space-y-10">
            {sections.map((s) => (
              <section key={s.id} id={s.id} className="scroll-mt-24">
                <h2 className="text-[17px] font-semibold text-slate-900">{s.title}</h2>
                {s.body?.map((p, i) => (
                  <p key={i} className="mt-3 text-[14px] leading-relaxed text-slate-600">
                    {p}
                  </p>
                ))}
                {s.list ? (
                  <ul className="mt-4 space-y-2">
                    {s.list.map((item) => (
                      <li key={item} className="flex items-start gap-2.5 text-[14px] text-slate-600">
                        <span className="mt-2 size-1.5 shrink-0 rounded-full bg-brand-400" />
                        {item}
                      </li>
                    ))}
                  </ul>
                ) : null}
                {s.code ? <CodeBlock className="mt-4" code={s.code.code} language={s.code.language} /> : null}
                {s.note ? (
                  <div className="mt-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-[13px] text-amber-800">
                    {s.note}
                  </div>
                ) : null}
              </section>
            ))}
          </div>

          <div className="mt-14 flex items-center justify-between border-t border-slate-100 pt-6 text-[13px]">
            <span className="text-slate-400">Was this helpful?</span>
            <Link href="/docs/payments" className="text-brand-500 hover:underline">
              Next: Payments →
            </Link>
          </div>
        </div>

        {/* On this page */}
        <aside className="sticky top-16 hidden h-fit w-48 shrink-0 py-1 xl:block">
          <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">On this page</p>
          <nav className="mt-3 flex flex-col gap-2">
            {sections.map((s) => (
              <a key={s.id} href={`#${s.id}`} className="text-[13px] text-slate-500 hover:text-slate-900">
                {s.title}
              </a>
            ))}
          </nav>
        </aside>
      </div>
    </article>
  );
}
