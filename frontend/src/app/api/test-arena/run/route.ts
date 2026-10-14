import { NextRequest, NextResponse } from "next/server";

interface RunRequestBody {
  url: string;
  method?: string;
  apiKey: string;
  data?: unknown;
  params?: Record<string, string>;
  headers?: Record<string, string>;
}

export async function POST(req: NextRequest) {
  const start = performance.now();

  try {
    const body: RunRequestBody = await req.json();
    const { url, method = "GET", apiKey, data, params, headers: customHeaders } = body;

    if (!url) {
      return NextResponse.json(
        {
          status: 400,
          statusText: "Bad Request",
          durationMs: 0,
          data: {
            error: {
              code: "missing_url",
              message: "Target endpoint URL is required.",
            },
          },
        },
        { status: 400 }
      );
    }

    if (!apiKey) {
      return NextResponse.json(
        {
          status: 401,
          statusText: "Unauthorized",
          durationMs: 0,
          data: {
            error: {
              code: "missing_api_key",
              message: "API key is required. Please select or provide a test key.",
            },
          },
        },
        { status: 401 }
      );
    }

    // Safety guard: Test Arena only runs in test environment
    const trimmedKey = apiKey.trim();
    if (trimmedKey.startsWith("pc_live_") || trimmedKey.startsWith("vpay_live_")) {
      return NextResponse.json(
        {
          status: 403,
          statusText: "Forbidden",
          durationMs: 0,
          data: {
            error: {
              code: "live_key_forbidden",
              message: "Live API keys cannot be used in the Test Arena. Use a test key.",
            },
          },
        },
        { status: 403 }
      );
    }

    // Resolve full target URL
    const defaultBase = process.env.API_URL || process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/v1";
    let targetUrlString = url;
    if (!url.startsWith("http://") && !url.startsWith("https://")) {
      const cleanPath = url.startsWith("/") ? url : `/${url}`;
      targetUrlString = `${defaultBase.replace(/\/+$/, "")}${cleanPath}`;
    }

    const targetUrl = new URL(targetUrlString);
    if (params) {
      Object.entries(params).forEach(([k, v]) => {
        if (v !== undefined && v !== "") {
          targetUrl.searchParams.set(k, v);
        }
      });
    }

    const uppercaseMethod = method.toUpperCase();
    const hasBody = ["POST", "PUT", "PATCH"].includes(uppercaseMethod);

    const requestHeaders: Record<string, string> = {
      "Content-Type": "application/json",
      Authorization: `Bearer ${trimmedKey}`,
      ...(customHeaders || {}),
    };

    const fetchOptions: RequestInit = {
      method: uppercaseMethod,
      headers: requestHeaders,
      cache: "no-store",
    };

    if (hasBody && data !== undefined) {
      fetchOptions.body = JSON.stringify(data);
    }

    const res = await fetch(targetUrl.toString(), fetchOptions);
    const durationMs = Math.round(performance.now() - start);

    let responseData: unknown = null;
    const contentType = res.headers.get("content-type") || "";

    if (contentType.includes("application/json")) {
      try {
        responseData = await res.json();
      } catch {
        responseData = null;
      }
    } else {
      const text = await res.text();
      try {
        responseData = JSON.parse(text);
      } catch {
        responseData = text;
      }
    }

    // Return the result with the actual status code from backend
    return NextResponse.json({
      status: res.status,
      statusText: res.statusText,
      durationMs,
      data: responseData,
    });
  } catch (err: unknown) {
    const durationMs = Math.round(performance.now() - start);
    const message = err instanceof Error ? err.message : "Internal proxy error";

    return NextResponse.json({
      status: 502,
      statusText: "Bad Gateway",
      durationMs,
      data: {
        error: {
          code: "gateway_error",
          message: "Failed to connect to backend server. Make sure the backend is running.",
          details: message,
        },
      },
    });
  }
}
