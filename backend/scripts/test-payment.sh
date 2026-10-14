#!/usr/bin/env bash
# Sends one test payment through the API so the whole path can be watched.
#
#   PC_TEST_KEY=pc_test_sk_… ./scripts/test-payment.sh
#
# Optional: API_URL (default http://localhost:8080/v1), AMOUNT_MINOR, CURRENCY,
# EMAIL, NAME.
#
# Two outcomes are both informative:
#   * the API answers 201 with a payment — the provider accepted the charge;
#   * the API answers with a provider error — the payment row was still written
#     with its attribution, so the listing, the filters and the trace all work.
# Either way the server log shows the flow line by line: grep for the request id
# printed at the end.
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080/v1}"
KEY="${PC_TEST_KEY:-}"
if [[ -z "$KEY" ]]; then
  echo "PC_TEST_KEY must be a test key (pc_test_sk_…): copy one from the dashboard's API Keys page." >&2
  exit 1
fi
case "$KEY" in
  pc_test_sk_*) ;;
  *) echo "Refusing to run: $KEY does not look like a test key. Test keys cannot touch live money." >&2; exit 1 ;;
esac

AMOUNT_MINOR="${AMOUNT_MINOR:-500000}"
CURRENCY="${CURRENCY:-NGN}"
EMAIL="${EMAIL:-ada@example.com}"
NAME="${NAME:-Ada Lovelace}"
REFERENCE="${REFERENCE:-curl_$(date +%s)}"
IDEMPOTENCY_KEY="${IDEMPOTENCY_KEY:-test-payment-${REFERENCE}}"

echo "POST $API_URL/payments  reference=$REFERENCE  amount=$AMOUNT_MINOR $CURRENCY"
RESPONSE="$(curl -sS -w '\n%{http_code}' -X POST "$API_URL/payments" \
  -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d "{\"amount\": $AMOUNT_MINOR, \"currency\": \"$CURRENCY\", \"reference\": \"$REFERENCE\",
       \"description\": \"Scripted test payment\",
       \"customer\": {\"email\": \"$EMAIL\", \"name\": \"$NAME\"},
       \"metadata\": {\"source\": \"test-payment.sh\"}}")"

BODY="$(printf '%s' "$RESPONSE" | sed '$d')"
STATUS="$(printf '%s' "$RESPONSE" | tail -n1)"
printf '%s\n' "$BODY" | python3 -m json.tool 2>/dev/null || printf '%s\n' "$BODY"
echo "HTTP $STATUS"

REQUEST_ID="$(printf '%s' "$BODY" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("meta",{}).get("request_id",""))' 2>/dev/null || true)"
PAYMENT_ID="$(printf '%s' "$BODY" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("data",{}).get("id",""))' 2>/dev/null || true)"

cat <<EOF

Follow it in the API log — one line per step, all carrying the same ids:
  grep -E "api key authenticated|payment create start|payment chain|payment recorded|payment sent to provider|payment (marked failed|failed|outcome unknown|provider outcome applied)" <api.log>
  request_id: ${REQUEST_ID:-—}   payment_id: ${PAYMENT_ID:-—}
EOF
