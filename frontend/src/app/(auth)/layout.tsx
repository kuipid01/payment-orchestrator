/**
 * Every page in this group renders its own full-height shell: sign-in and
 * sign-up carry bespoke product panels, and the recovery, verification and
 * invitation screens share `AuthShell`. The group therefore adds no chrome of
 * its own — it exists to keep the auth routes together and to hold the layout
 * boundary, not to wrap them in a second design.
 */
export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return children;
}
