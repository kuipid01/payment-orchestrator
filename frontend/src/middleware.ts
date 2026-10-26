import { NextRequest, NextResponse } from "next/server";

const APP_URL = process.env.APP_URL ?? "https://app.payconnect.com";
const APP_PATHS = [
  "/login",
  "/register",
  "/forgot-password",
  "/reset-password",
  "/verify-email",
  "/onboarding",
  "/dashboard",
  "/payments",
  "/providers",
  "/applications",
  "/api-keys",
  "/developers",
  "/webhooks",
  "/logs",
  "/settings",
  "/403",
  "/404",
  "/500",
  "/maintenance",
];

function isAppPath(pathname: string) {
  return APP_PATHS.some(
    (path) => pathname === path || pathname.startsWith(`${path}/`),
  );
}

export function middleware(request: NextRequest) {
  const appUrl = new URL(APP_URL);
  const appHost = appUrl.host.toLowerCase();
  const requestHost = request.nextUrl.host.toLowerCase();

  // if (requestHost === appHost && request.nextUrl.pathname === "/") {
  //   return NextResponse.redirect(new URL("/dashboard", appUrl));
  // }

  const appDomain = appUrl.hostname.split(".").slice(1).join(".");
  const marketingHosts = new Set([appDomain, `www.${appDomain}`]);
  if (
    requestHost !== appHost &&
    marketingHosts.has(request.nextUrl.hostname.toLowerCase()) &&
    isAppPath(request.nextUrl.pathname)
  ) {
    const destination = new URL(request.nextUrl);
    destination.protocol = appUrl.protocol;
    destination.host = appUrl.host;
    return NextResponse.redirect(destination);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
