/**
 * Testable endpoint catalogue for the Test Arena.
 * Each entry describes one API call a developer can fire from the dashboard.
 * The base URL is always prepended at runtime — nothing here is user-editable.
 */

export type FieldType = "string" | "number" | "email" | "select" | "textarea";

export interface FieldOption {
  value: string;
  label: string;
}

export interface FieldDef {
  key: string;
  label: string;
  type: FieldType;
  placeholder?: string;
  required?: boolean;
  defaultValue?: string | number;
  options?: FieldOption[];
  description?: string;
}

export interface PathParam {
  name: string;
  placeholder: string;
  description?: string;
}

export interface EndpointDef {
  id: string;
  label: string;
  description: string;
  method: "GET" | "POST" | "DELETE" | "PATCH";
  /** Relative path; :param segments are replaced at call time. */
  path: string;
  group: string;
  /** If true, only test keys are allowed. Always true for payment operations. */
  requiresTestKey: boolean;
  pathParams?: PathParam[];
  bodySchema?: FieldDef[];
  /** Query params shown as form fields when method is GET. */
  querySchema?: FieldDef[];
}

export const ENDPOINT_GROUPS = [
  "Payments",
  "Refunds",
  "Providers",
  "Webhooks",
  "Transactions",
  "Logs",
  "Customers",
  "Applications",
] as const;

export const ENDPOINTS: EndpointDef[] = [
  /* ─────────────────────────── PAYMENTS ─────────────────────────── */
  {
    id: "initiate-payment",
    label: "Initiate Payment",
    description: "Create a new payment and get a provider checkout URL back.",
    method: "POST",
    path: "/payments",
    group: "Payments",
    requiresTestKey: true,
    bodySchema: [
      {
        key: "amount",
        label: "Amount (minor units)",
        type: "number",
        placeholder: "5000",
        required: true,
        defaultValue: 5000,
        description: "Amount in minor units e.g. 5000 = ₦50.00",
      },
      {
        key: "currency",
        label: "Currency",
        type: "select",
        required: true,
        defaultValue: "NGN",
        options: [
          { value: "NGN", label: "NGN – Nigerian Naira" },
          { value: "GHS", label: "GHS – Ghanaian Cedi" },
          { value: "USD", label: "USD – US Dollar" },
          { value: "KES", label: "KES – Kenyan Shilling" },
        ],
      },
      {
        key: "email",
        label: "Customer Email",
        type: "email",
        placeholder: "customer@example.com",
        required: true,
        description: "Payer email address. Required to initiate payment.",
      },
      {
        key: "adapter",
        label: "Provider Adapter",
        type: "select",
        required: false,
        defaultValue: "auto",
        options: [
          { value: "auto", label: "auto – Smart Routing (Auto-failover)" },
          { value: "paystack", label: "paystack – Paystack" },
          { value: "flutterwave", label: "flutterwave – Flutterwave" },
          { value: "monnify", label: "monnify – Monnify" },
          { value: "stripe", label: "stripe – Stripe" },
          { value: "paypal", label: "paypal – PayPal" },
        ],
        description: "Choose a specific provider or let smart routing decide.",
      },
      {
        key: "reference",
        label: "Reference",
        type: "string",
        placeholder: "PAY-001 (auto-generated if blank)",
        description: "Your unique transaction reference",
      },
      {
        key: "description",
        label: "Description",
        type: "string",
        placeholder: "Test payment from Test Arena",
      },
    ],
  },
  {
    id: "verify-payment-ref",
    label: "Verify Payment (by Reference)",
    description: "Check payment status and verification state using reference.",
    method: "GET",
    path: "/payments/verify",
    group: "Payments",
    requiresTestKey: true,
    querySchema: [
      {
        key: "reference",
        label: "Payment Reference",
        type: "string",
        placeholder: "e.g. ref_xxxxxxxx or PAY-001",
        required: true,
        description: "Unique reference used when initiating the payment.",
      },
    ],
  },
  {
    id: "verify-payment-id",
    label: "Verify Payment (by ID)",
    description: "Verify and reconcile payment state with provider by payment ID.",
    method: "GET",
    path: "/payments/:id/verify",
    group: "Payments",
    requiresTestKey: true,
    pathParams: [
      { name: "id", placeholder: "pay_xxxxxxxx", description: "Payment ID" },
    ],
  },
  {
    id: "get-payment",
    label: "Get Payment",
    description: "Retrieve a single payment by ID, including its timeline.",
    method: "GET",
    path: "/payments/:id",
    group: "Payments",
    requiresTestKey: true,
    pathParams: [
      { name: "id", placeholder: "pay_xxxxxxxx", description: "Payment ID" },
    ],
  },
  {
    id: "list-payments",
    label: "List Payments",
    description: "Fetch a page of payments with optional filters.",
    method: "GET",
    path: "/payments",
    group: "Payments",
    requiresTestKey: true,
    querySchema: [
      {
        key: "status",
        label: "Status",
        type: "select",
        options: [
          { value: "", label: "All statuses" },
          { value: "pending", label: "Pending" },
          { value: "successful", label: "Successful" },
          { value: "failed", label: "Failed" },
        ],
      },
      {
        key: "limit",
        label: "Limit",
        type: "number",
        placeholder: "10",
        defaultValue: 10,
      },
    ],
  },

  /* ─────────────────────────── REFUNDS ─────────────────────────── */
  {
    id: "create-refund",
    label: "Create Refund",
    description: "Initiate a full or partial refund for a successful payment.",
    method: "POST",
    path: "/payments/:id/refund",
    group: "Refunds",
    requiresTestKey: true,
    pathParams: [
      { name: "id", placeholder: "pay_xxxxxxxx", description: "Payment ID to refund" },
    ],
    bodySchema: [
      {
        key: "amount",
        label: "Refund Amount (minor units, leave blank for full refund)",
        type: "number",
        placeholder: "5000",
        description: "Leave blank to refund the full amount",
      },
      {
        key: "reason",
        label: "Reason",
        type: "string",
        placeholder: "Customer requested refund",
      },
    ],
  },

  /* ─────────────────────────── PROVIDERS ─────────────────────────── */
  {
    id: "list-providers",
    label: "List Providers",
    description: "List all supported payment providers and check connection status for your account.",
    method: "GET",
    path: "/providers",
    group: "Providers",
    requiresTestKey: false,
  },

  /* ─────────────────────────── WEBHOOKS ─────────────────────────── */
  {
    id: "list-webhook-endpoints",
    label: "List Webhook Endpoints",
    description: "Fetch all registered webhook destination endpoints and their delivery stats.",
    method: "GET",
    path: "/webhook-endpoints",
    group: "Webhooks",
    requiresTestKey: false,
  },
  {
    id: "list-webhook-deliveries",
    label: "List Webhook Deliveries",
    description: "Fetch outbound webhook delivery attempts with HTTP status codes and payloads.",
    method: "GET",
    path: "/webhook-deliveries",
    group: "Webhooks",
    requiresTestKey: false,
    querySchema: [
      {
        key: "limit",
        label: "Limit",
        type: "number",
        placeholder: "10",
        defaultValue: 10,
      },
    ],
  },
  {
    id: "list-webhook-events",
    label: "List Inbound Events",
    description: "View inbound webhook events received from connected payment providers.",
    method: "GET",
    path: "/webhook-events",
    group: "Webhooks",
    requiresTestKey: false,
    querySchema: [
      {
        key: "limit",
        label: "Limit",
        type: "number",
        placeholder: "10",
        defaultValue: 10,
      },
    ],
  },

  /* ─────────────────────────── TRANSACTIONS ─────────────────────────── */
  {
    id: "list-transactions",
    label: "List Transactions",
    description: "List ledger transactions and settlement records.",
    method: "GET",
    path: "/transactions",
    group: "Transactions",
    requiresTestKey: false,
    querySchema: [
      {
        key: "type",
        label: "Type",
        type: "select",
        options: [
          { value: "", label: "All types" },
          { value: "payment", label: "Payment" },
          { value: "refund", label: "Refund" },
          { value: "fee", label: "Fee" },
        ],
      },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: [
          { value: "", label: "All statuses" },
          { value: "successful", label: "Successful" },
          { value: "pending", label: "Pending" },
          { value: "failed", label: "Failed" },
        ],
      },
      {
        key: "limit",
        label: "Limit",
        type: "number",
        placeholder: "10",
        defaultValue: 10,
      },
    ],
  },
  {
    id: "get-transaction",
    label: "Get Transaction",
    description: "Retrieve a single ledger transaction entry by ID.",
    method: "GET",
    path: "/transactions/:id",
    group: "Transactions",
    requiresTestKey: false,
    pathParams: [
      { name: "id", placeholder: "tx_xxxxxxxx", description: "Transaction ID" },
    ],
  },

  /* ─────────────────────────── LOGS ─────────────────────────── */
  {
    id: "list-logs",
    label: "List API Logs",
    description: "Query developer API request logs (method, path, status, duration).",
    method: "GET",
    path: "/logs",
    group: "Logs",
    requiresTestKey: false,
    querySchema: [
      {
        key: "status_class",
        label: "Status Class",
        type: "select",
        options: [
          { value: "", label: "All status codes" },
          { value: "2xx", label: "2xx Success" },
          { value: "4xx", label: "4xx Client Error" },
          { value: "5xx", label: "5xx Server Error" },
        ],
      },
      {
        key: "method",
        label: "HTTP Method",
        type: "select",
        options: [
          { value: "", label: "All methods" },
          { value: "GET", label: "GET" },
          { value: "POST", label: "POST" },
          { value: "PATCH", label: "PATCH" },
          { value: "DELETE", label: "DELETE" },
        ],
      },
      {
        key: "limit",
        label: "Limit",
        type: "number",
        placeholder: "10",
        defaultValue: 10,
      },
    ],
  },
  {
    id: "get-log",
    label: "Get API Log",
    description: "Retrieve full details and headers of a specific API request log.",
    method: "GET",
    path: "/logs/:id",
    group: "Logs",
    requiresTestKey: false,
    pathParams: [
      { name: "id", placeholder: "log_xxxxxxxx", description: "Log ID" },
    ],
  },

  /* ─────────────────────────── CUSTOMERS ─────────────────────────── */
  {
    id: "create-customer",
    label: "Create Customer",
    description: "Create a customer profile for payer tracking and records.",
    method: "POST",
    path: "/customers",
    group: "Customers",
    requiresTestKey: false,
    bodySchema: [
      {
        key: "email",
        label: "Customer Email",
        type: "email",
        placeholder: "customer@example.com",
        required: true,
      },
      {
        key: "name",
        label: "Full Name",
        type: "string",
        placeholder: "Ada Lovelace",
      },
      {
        key: "phone",
        label: "Phone Number",
        type: "string",
        placeholder: "+2348012345678",
      },
    ],
  },
  {
    id: "list-customers",
    label: "List Customers",
    description: "Fetch customer records with optional email/search filter.",
    method: "GET",
    path: "/customers",
    group: "Customers",
    requiresTestKey: false,
    querySchema: [
      {
        key: "email",
        label: "Email Filter",
        type: "string",
        placeholder: "filter by email...",
      },
      {
        key: "limit",
        label: "Limit",
        type: "number",
        placeholder: "10",
        defaultValue: 10,
      },
    ],
  },
  {
    id: "get-customer",
    label: "Get Customer",
    description: "Retrieve customer details and metadata by ID.",
    method: "GET",
    path: "/customers/:id",
    group: "Customers",
    requiresTestKey: false,
    pathParams: [
      { name: "id", placeholder: "cust_xxxxxxxx", description: "Customer ID" },
    ],
  },

  /* ─────────────────────────── APPLICATIONS ─────────────────────────── */
  {
    id: "list-applications",
    label: "List Applications",
    description: "Fetch all applications registered under your account.",
    method: "GET",
    path: "/applications",
    group: "Applications",
    requiresTestKey: false,
  },
  {
    id: "get-application",
    label: "Get Application",
    description: "Retrieve a single application with its lifetime stats.",
    method: "GET",
    path: "/applications/:id",
    group: "Applications",
    requiresTestKey: false,
    pathParams: [
      { name: "id", placeholder: "app_xxxxxxxx", description: "Application ID" },
    ],
  },
];
