package webhooks

import (
	"encoding/json"
	"testing"
	"time"
)

// TestSuccessRate pins the metric's arithmetic and its refusal to invent a rate
// when no delivery concluded. The dashboard renders whatever this returns, so a
// silent zero here would read as "0% of deliveries reached you".
func TestSuccessRate(t *testing.T) {
	cases := []struct {
		name      string
		delivered int64
		total     int64
		want      *float64
	}{
		{name: "nothing concluded", delivered: 0, total: 0, want: nil},
		{name: "all delivered", delivered: 4, total: 4, want: ptr(100)},
		{name: "nothing delivered", delivered: 0, total: 3, want: ptr(0)},
		{name: "rounded to one decimal", delivered: 2, total: 3, want: ptr(66.7)},
		{name: "single failure among many", delivered: 19, total: 20, want: ptr(95)},
		{name: "repeating fraction", delivered: 1, total: 3, want: ptr(33.3)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SuccessRate(tc.delivered, tc.total)
			switch {
			case tc.want == nil && got != nil:
				t.Fatalf("SuccessRate(%d, %d) = %v, want nil", tc.delivered, tc.total, *got)
			case tc.want != nil && got == nil:
				t.Fatalf("SuccessRate(%d, %d) = nil, want %v", tc.delivered, tc.total, *tc.want)
			case tc.want != nil && *got != *tc.want:
				t.Fatalf("SuccessRate(%d, %d) = %v, want %v", tc.delivered, tc.total, *got, *tc.want)
			}
		})
	}
}

// TestEndpointResponseDeliveryStats checks that an endpoint response carries both
// the rate and the counts it was computed from, and omits the rate entirely when
// the window holds no concluded delivery.
func TestEndpointResponseDeliveryStats(t *testing.T) {
	endpoint := &Endpoint{
		ID:          "wh_1",
		URL:         "https://example.test/hooks",
		Environment: "test",
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	response := ToEndpointResponse(endpoint)
	if response.DeliveryWindowDays != DeliveryWindowDays {
		t.Fatalf("DeliveryWindowDays = %d, want %d", response.DeliveryWindowDays, DeliveryWindowDays)
	}
	if response.DeliverySuccessRate != nil {
		t.Fatalf("rate = %v before stats were attached, want nil", *response.DeliverySuccessRate)
	}

	response.WithDeliveryStats(EndpointDeliveryCount{Delivered: 6, Total: 8})
	if response.Delivered != 6 || response.Deliveries != 8 {
		t.Fatalf("counts = %d/%d, want 6/8", response.Delivered, response.Deliveries)
	}
	if response.DeliverySuccessRate == nil || *response.DeliverySuccessRate != 75 {
		t.Fatalf("rate = %v, want 75", response.DeliverySuccessRate)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["delivery_success_rate"] != 75.0 {
		t.Fatalf("delivery_success_rate = %v, want 75", decoded["delivery_success_rate"])
	}
	if decoded["deliveries"] != 8.0 || decoded["delivered"] != 6.0 {
		t.Fatalf("counts in JSON = %v/%v, want 6/8", decoded["delivered"], decoded["deliveries"])
	}
	if decoded["delivery_window_days"] != float64(DeliveryWindowDays) {
		t.Fatalf("delivery_window_days = %v, want %d", decoded["delivery_window_days"], DeliveryWindowDays)
	}

	empty := ToEndpointResponse(endpoint)
	encodedEmpty, err := json.Marshal(empty)
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	var decodedEmpty map[string]any
	if err := json.Unmarshal(encodedEmpty, &decodedEmpty); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
	if _, present := decodedEmpty["delivery_success_rate"]; present {
		t.Fatalf("delivery_success_rate present with no concluded deliveries: %v", decodedEmpty["delivery_success_rate"])
	}
}

func ptr(value float64) *float64 { return &value }
