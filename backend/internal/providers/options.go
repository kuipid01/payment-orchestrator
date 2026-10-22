package providers

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// Codes returned when caller-supplied provider options cannot be honoured. They
// are distinct from provider errors because the request never reached a provider.
const (
	codeOptionsConflict        = "provider_options_conflict"
	codeOptionsUnsupported     = "provider_options_not_supported"
	codeOptionsInvalid         = "provider_options_invalid"
	codeOptionsTestUnsupported = "provider_options_not_available_for_test"
)

// Merchant-of-record models a Stripe Checkout Session can use.
const (
	// StripeStandardPayments is Stripe's normal model: the merchant is the
	// merchant of record and remains responsible for tax.
	StripeStandardPayments = "standard"
	// StripeManagedPayments is Stripe's merchant-of-record model. It is only
	// available to eligible accounts selling eligible digital products, so it is
	// opt-in per payment and never enabled by deployment configuration.
	StripeManagedPayments = "managed"
)

// productTaxCodePattern matches Stripe's product tax code identifiers
// (txcd_ followed by a category). A value that does not match cannot be a real
// code, so it is rejected before the provider round trip rather than after.
var productTaxCodePattern = regexp.MustCompile(`^txcd_[a-z0-9_]{3,48}$`)

// PaymentOptions carries caller-supplied provider-specific payment settings.
//
// It is deliberately namespaced: each provider owns one struct, the payment
// contract carries no provider-specific fields of its own, and nothing here is
// forwarded to a provider API verbatim. An adapter translates the validated
// struct field by field, so a caller can never inject an arbitrary provider
// parameter, and a field that a provider does not support is a validation error
// instead of a silently dropped value.
type PaymentOptions struct {
	Stripe *StripeOptions `json:"stripe,omitempty"`
	PayPal *PayPalOptions `json:"paypal,omitempty"`

	// invalid records a namespace or field that could not be decoded. It is
	// carried to Validate so the caller gets a precise rejection instead of a
	// generic "invalid JSON body", and so an unsupported namespace can never be
	// silently dropped.
	invalid error
}

// UnmarshalJSON decodes the options strictly: an unknown namespace or an
// unknown field inside one is recorded rather than ignored, because silently
// dropping a caller's setting would change what they asked to be charged for.
func (o *PaymentOptions) UnmarshalJSON(data []byte) error {
	type plain PaymentOptions
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded plain
	if err := decoder.Decode(&decoded); err != nil {
		decoded.invalid = err
	}
	*o = PaymentOptions(decoded)
	return nil
}

// StripeOptions are the Stripe Checkout Session settings PayConnect supports.
type StripeOptions struct {
	// MerchantOfRecord selects standard Payments (the default) or Managed
	// Payments. It is a per-payment choice, not a deployment setting.
	MerchantOfRecord string `json:"merchant_of_record,omitempty"`
	// ProductName and ProductDescription describe the inline line item.
	ProductName        string `json:"product_name,omitempty"`
	ProductDescription string `json:"product_description,omitempty"`
	// ProductTaxCode is the Stripe product tax code of the thing being sold.
	// Managed Payments requires an eligible one; standard Payments accepts it as
	// an optional classification.
	ProductTaxCode string `json:"product_tax_code,omitempty"`
	// CustomerEmail pre-fills the Checkout email field.
	CustomerEmail string `json:"customer_email,omitempty"`
	// SuccessURL and CancelURL override the destination derived from callback_url.
	SuccessURL string `json:"success_url,omitempty"`
	CancelURL  string `json:"cancel_url,omitempty"`
	// Checkout page behaviour.
	SubmitType               string `json:"submit_type,omitempty"`
	Locale                   string `json:"locale,omitempty"`
	BillingAddressCollection string `json:"billing_address_collection,omitempty"`
	AllowPromotionCodes      *bool  `json:"allow_promotion_codes,omitempty"`
	// SetupFutureUsage is forwarded to the PaymentIntent so the payment method
	// can be reused later (off_session payments, subscriptions).
	SetupFutureUsage string `json:"setup_future_usage,omitempty"`
	// PaymentIntentMetadata is attached to the PaymentIntent, alongside the
	// payment reference PayConnect always sets.
	PaymentIntentMetadata map[string]string `json:"payment_intent_metadata,omitempty"`
}

// Managed reports whether the caller asked Stripe for Managed Payments.
func (o *StripeOptions) Managed() bool {
	return o != nil && strings.EqualFold(strings.TrimSpace(o.MerchantOfRecord), StripeManagedPayments)
}

// PayPalOptions are the PayPal Orders v2 settings PayConnect supports.
type PayPalOptions struct {
	// Description is the purchase unit description shown to the payer.
	Description string `json:"description,omitempty"`
	// InvoiceID and CustomID are the merchant's own identifiers on the purchase
	// unit. They default to the PayConnect reference so reconciliation by id
	// keeps working.
	InvoiceID string `json:"invoice_id,omitempty"`
	CustomID  string `json:"custom_id,omitempty"`
	// PayerEmail pre-fills the PayPal payer email where the flow supports it.
	PayerEmail string `json:"payer_email,omitempty"`
	// Application context.
	BrandName          string `json:"brand_name,omitempty"`
	Locale             string `json:"locale,omitempty"`
	LandingPage        string `json:"landing_page,omitempty"`
	UserAction         string `json:"user_action,omitempty"`
	ShippingPreference string `json:"shipping_preference,omitempty"`
}

// Provider returns the provider these options belong to, or the empty provider
// when none were supplied.
func (o PaymentOptions) Provider() constants.Provider {
	switch {
	case o.Stripe != nil:
		return constants.ProviderStripe
	case o.PayPal != nil:
		return constants.ProviderPayPal
	default:
		return ""
	}
}

// IsZero reports whether the caller supplied no provider options at all. A
// payload that failed to decode is not zero: it must reach validation.
func (o PaymentOptions) IsZero() bool {
	return o.Stripe == nil && o.PayPal == nil && o.invalid == nil
}

// Validate checks the supplied namespace in isolation: its own fields, its own
// value domains. It is called before provider routing so an invalid request is
// rejected without reserving an idempotency key or writing a payment.
func (o PaymentOptions) Validate() error {
	if o.invalid != nil {
		return apperrors.Validation(codeOptionsUnsupported,
			"Unsupported provider option "+unknownOptionField(o.invalid.Error())+
				". Send options.stripe or options.paypal, using only the fields GET /v1/providers lists in supported_options.")
	}
	switch {
	case o.Stripe != nil && o.PayPal != nil:
		return apperrors.Validation(codeOptionsConflict,
			"options may target only one provider; send either options.stripe or options.paypal.")
	case o.Stripe != nil:
		return o.Stripe.validate()
	case o.PayPal != nil:
		return o.PayPal.validate()
	default:
		return nil
	}
}

// ValidateFor checks the options against the provider that will run the payment.
// test marks a dashboard-initiated sandbox charge, which has its own policy.
func (o PaymentOptions) ValidateFor(provider constants.Provider, test bool) error {
	if o.IsZero() {
		return nil
	}
	supplied := o.Provider()
	if supplied != provider {
		return apperrors.Validation(codeOptionsUnsupported,
			"The options were supplied for "+supplied.String()+" but the payment is routed to "+
				provider.String()+"; pin adapter to "+supplied.String()+" or drop options.")
	}
	if err := o.Validate(); err != nil {
		return err
	}
	if o.Stripe != nil && o.Stripe.Managed() && test {
		// A dashboard test transaction must always use standard Payments: it
		// exists to exercise the merchant's own checkout, and Managed Payments
		// changes who the merchant of record is. Refusing is honest; silently
		// downgrading would hide the mistake.
		return apperrors.Validation(codeOptionsTestUnsupported,
			"merchant_of_record \"managed\" is not available for dashboard test transactions; use \"standard\".")
	}
	return nil
}

func (o *StripeOptions) validate() error {
	o.MerchantOfRecord = strings.ToLower(strings.TrimSpace(o.MerchantOfRecord))
	switch o.MerchantOfRecord {
	case "", StripeStandardPayments, StripeManagedPayments:
	default:
		return apperrors.Validation(codeOptionsInvalid,
			"options.stripe.merchant_of_record must be \"standard\" or \"managed\".")
	}
	if o.Managed() && strings.TrimSpace(o.ProductTaxCode) == "" {
		// Managed Payments needs an eligible product tax code; there is no safe
		// default, and a fabricated one would misclassify the sale.
		return apperrors.Validation(codeOptionsInvalid,
			"options.stripe.product_tax_code is required when merchant_of_record is \"managed\".")
	}
	o.ProductName = strings.TrimSpace(o.ProductName)
	if len(o.ProductName) > 250 {
		return apperrors.Validation(codeOptionsInvalid, "options.stripe.product_name must be at most 250 characters.")
	}
	o.ProductDescription = strings.TrimSpace(o.ProductDescription)
	if len(o.ProductDescription) > 500 {
		return apperrors.Validation(codeOptionsInvalid, "options.stripe.product_description must be at most 500 characters.")
	}
	o.ProductTaxCode = strings.TrimSpace(o.ProductTaxCode)
	if o.ProductTaxCode != "" && !productTaxCodePattern.MatchString(o.ProductTaxCode) {
		return apperrors.Validation(codeOptionsInvalid,
			"options.stripe.product_tax_code must be a Stripe product tax code such as txcd_10000000.")
	}
	o.CustomerEmail = strings.TrimSpace(o.CustomerEmail)
	if o.CustomerEmail != "" && !looksLikeEmail(o.CustomerEmail) {
		return apperrors.Validation(codeOptionsInvalid, "options.stripe.customer_email is not a valid email address.")
	}
	for field, raw := range map[string]*string{
		"success_url": &o.SuccessURL,
		"cancel_url":  &o.CancelURL,
	} {
		*raw = strings.TrimSpace(*raw)
		if *raw == "" {
			continue
		}
		if !isAbsoluteHTTPURL(*raw) {
			return apperrors.Validation(codeOptionsInvalid,
				"options.stripe."+field+" must be an absolute http(s) URL.")
		}
	}
	o.SubmitType = strings.ToLower(strings.TrimSpace(o.SubmitType))
	switch o.SubmitType {
	case "", "pay", "book", "donate":
	default:
		return apperrors.Validation(codeOptionsInvalid,
			"options.stripe.submit_type must be one of pay, book, donate.")
	}
	o.BillingAddressCollection = strings.ToLower(strings.TrimSpace(o.BillingAddressCollection))
	switch o.BillingAddressCollection {
	case "", "auto", "required":
	default:
		return apperrors.Validation(codeOptionsInvalid,
			"options.stripe.billing_address_collection must be \"auto\" or \"required\".")
	}
	o.SetupFutureUsage = strings.ToLower(strings.TrimSpace(o.SetupFutureUsage))
	switch o.SetupFutureUsage {
	case "", "none", "off_session", "on_session":
	default:
		return apperrors.Validation(codeOptionsInvalid,
			"options.stripe.setup_future_usage must be one of none, off_session, on_session.")
	}
	o.Locale = strings.TrimSpace(o.Locale)
	if len(o.Locale) > 16 {
		return apperrors.Validation(codeOptionsInvalid, "options.stripe.locale is too long.")
	}
	return validateMetadata("options.stripe.payment_intent_metadata", o.PaymentIntentMetadata)
}

func (o *PayPalOptions) validate() error {
	o.Description = strings.TrimSpace(o.Description)
	if len(o.Description) > 127 {
		return apperrors.Validation(codeOptionsInvalid, "options.paypal.description must be at most 127 characters.")
	}
	o.InvoiceID = strings.TrimSpace(o.InvoiceID)
	if len(o.InvoiceID) > 127 {
		return apperrors.Validation(codeOptionsInvalid, "options.paypal.invoice_id must be at most 127 characters.")
	}
	o.CustomID = strings.TrimSpace(o.CustomID)
	if len(o.CustomID) > 127 {
		return apperrors.Validation(codeOptionsInvalid, "options.paypal.custom_id must be at most 127 characters.")
	}
	o.PayerEmail = strings.TrimSpace(o.PayerEmail)
	if o.PayerEmail != "" && !looksLikeEmail(o.PayerEmail) {
		return apperrors.Validation(codeOptionsInvalid, "options.paypal.payer_email is not a valid email address.")
	}
	o.BrandName = strings.TrimSpace(o.BrandName)
	if len(o.BrandName) > 127 {
		return apperrors.Validation(codeOptionsInvalid, "options.paypal.brand_name must be at most 127 characters.")
	}
	o.Locale = strings.TrimSpace(o.Locale)
	if len(o.Locale) > 10 {
		return apperrors.Validation(codeOptionsInvalid, "options.paypal.locale must be a language tag such as en-US.")
	}
	o.LandingPage = strings.ToUpper(strings.TrimSpace(o.LandingPage))
	switch o.LandingPage {
	case "", "LOGIN", "GUEST_CHECKOUT", "NO_PREFERENCE":
	default:
		return apperrors.Validation(codeOptionsInvalid,
			"options.paypal.landing_page must be one of LOGIN, GUEST_CHECKOUT, NO_PREFERENCE.")
	}
	o.UserAction = strings.ToUpper(strings.TrimSpace(o.UserAction))
	switch o.UserAction {
	case "", "CONTINUE", "PAY_NOW":
	default:
		return apperrors.Validation(codeOptionsInvalid, "options.paypal.user_action must be one of CONTINUE, PAY_NOW.")
	}
	o.ShippingPreference = strings.ToUpper(strings.TrimSpace(o.ShippingPreference))
	switch o.ShippingPreference {
	case "", "GET_FROM_FILE", "NO_SHIPPING", "SET_PROVIDED_ADDRESS":
	default:
		return apperrors.Validation(codeOptionsInvalid,
			"options.paypal.shipping_preference must be one of GET_FROM_FILE, NO_SHIPPING, SET_PROVIDED_ADDRESS.")
	}
	return nil
}

// unknownOptionField extracts the offending field name from a strict decoder
// error, whose shape is `json: unknown field "name"`.
func unknownOptionField(message string) string {
	const prefix = "json: unknown field "
	if !strings.HasPrefix(message, prefix) {
		return "in the options object"
	}
	return strings.TrimPrefix(message, prefix)
}

func validateMetadata(label string, values map[string]string) error {
	if len(values) > 20 {
		return apperrors.Validation(codeOptionsInvalid, label+" accepts at most 20 entries.")
	}
	for key, value := range values {
		if strings.TrimSpace(key) == "" {
			return apperrors.Validation(codeOptionsInvalid, label+" keys must not be empty.")
		}
		if len(key) > 40 {
			return apperrors.Validation(codeOptionsInvalid, label+" keys must be at most 40 characters.")
		}
		if len(value) > 500 {
			return apperrors.Validation(codeOptionsInvalid, label+" values must be at most 500 characters.")
		}
	}
	return nil
}

// isAbsoluteHTTPURL reports whether raw is an absolute http(s) destination. A
// relative path, a scheme such as javascript:, or a host-less value is rejected
// before it can be used as a redirect target.
func isAbsoluteHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

func looksLikeEmail(value string) bool {
	at := strings.IndexByte(value, '@')
	if at <= 0 || at == len(value)-1 {
		return false
	}
	domain := value[at+1:]
	return !strings.ContainsAny(value, " \t\n") && strings.Contains(domain, ".")
}

// OptionKeys lists the option fields a provider accepts, so the dashboard can
// render exactly the inputs that provider understands instead of hard-coding a
// provider's form. It mirrors the struct fields above.
func OptionKeys(provider constants.Provider) []string {
	switch provider {
	case constants.ProviderStripe:
		return []string{
			"merchant_of_record", "product_name", "product_description", "product_tax_code",
			"customer_email", "success_url", "cancel_url", "submit_type", "locale",
			"billing_address_collection", "allow_promotion_codes", "setup_future_usage",
			"payment_intent_metadata",
		}
	case constants.ProviderPayPal:
		return []string{
			"description", "invoice_id", "custom_id", "payer_email", "brand_name",
			"locale", "landing_page", "user_action", "shipping_preference",
		}
	default:
		return []string{}
	}
}
