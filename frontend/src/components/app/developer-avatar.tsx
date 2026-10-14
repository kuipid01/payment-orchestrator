import {
  Braces,
  Code2,
  Command,
  Cpu,
  Database,
  Fingerprint,
  Layers3,
  Rocket,
  Sparkles,
  Terminal,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

const ICONS: LucideIcon[] = [
  Braces,
  Code2,
  Command,
  Cpu,
  Database,
  Fingerprint,
  Layers3,
  Rocket,
  Sparkles,
  Terminal,
];

const PALETTES = [
  { background: "from-violet-500 to-indigo-700", glow: "bg-fuchsia-300", icon: "text-white" },
  { background: "from-cyan-400 to-blue-700", glow: "bg-sky-200", icon: "text-white" },
  { background: "from-amber-300 to-orange-600", glow: "bg-yellow-100", icon: "text-white" },
  { background: "from-emerald-400 to-teal-700", glow: "bg-lime-200", icon: "text-white" },
  { background: "from-pink-400 to-rose-700", glow: "bg-orange-200", icon: "text-white" },
  { background: "from-slate-600 to-slate-950", glow: "bg-cyan-300", icon: "text-white" },
];

function hashSeed(seed: string): number {
  let hash = 2166136261;
  for (let index = 0; index < seed.length; index += 1) {
    hash ^= seed.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

export function DeveloperAvatar({
  seed,
  className,
}: {
  seed: string;
  className?: string;
}) {
  const hash = hashSeed(seed);
  const Icon = ICONS[hash % ICONS.length];
  const palette = PALETTES[(hash >>> 8) % PALETTES.length];

  return (
    <span
      aria-hidden="true"
      className={cn(
        "relative isolate inline-flex shrink-0 items-center justify-center overflow-hidden rounded-2xl bg-linear-to-br shadow-sm ring-1 ring-black/5",
        palette.background,
        className,
      )}
    >
      <span className={cn("absolute -right-1/4 -top-1/4 size-3/4 rounded-full opacity-50 blur-md", palette.glow)} />
      <span className="absolute -bottom-1/3 -left-1/4 size-3/4 rounded-full bg-white/20 blur-md" />
      <Icon className={cn("relative z-10 size-1/2 drop-shadow-sm", palette.icon)} strokeWidth={1.8} />
      <Sparkles className="absolute right-[18%] top-[17%] z-10 size-[18%] text-white/90" />
      <span className="absolute inset-1 rounded-[inherit] border border-white/20" />
    </span>
  );
}
