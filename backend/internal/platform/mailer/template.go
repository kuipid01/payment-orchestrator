package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// Brand tokens, taken from the PayConnect theme (gold primary, teal success,
// red danger, light-first surfaces). Emails can't use CSS variables, so the
// values are inlined into the templates at compile time via tokens.
const (
	cGold     = "#E7B000"
	cGoldDark = "#A37D00"
	cGold50   = "#fdf8e7"
	cGold100  = "#faefc8"
	cGold200  = "#f5e191"
	cGold800  = "#765900"

	cTeal     = "#09B8BD"
	cTeal50   = "#e6fafb"
	cTeal200  = "#8ae6ea"
	cTeal700  = "#05787e"
	cTealDeep = "#015153"

	cRed    = "#A30104"
	cRed50  = "#fceaea"
	cRed200 = "#f4a9b9"

	cInk    = "#0f172a"
	cBody   = "#334155"
	cMuted  = "#64748b"
	cLine   = "#e2e8f0"
	cPage   = "#f6f3ea"
	cBorder = "#ece6d3"

	fSans = "'Kanit','Segoe UI',Helvetica,Arial,sans-serif"
	fMono = "'JetBrains Mono',Menlo,Consolas,'Courier New',monospace"

	// pStyle is the shared paragraph style used across all templates.
	pStyle = "margin:0 0 14px 0;font-family:" + fSans + ";font-size:16px;line-height:1.65;color:" + cBody + ";"
)

var tokens = strings.NewReplacer(
	"__GOLD__", cGold,
	"__GOLD_DARK__", cGoldDark,
	"__GOLD_50__", cGold50,
	"__GOLD_100__", cGold100,
	"__GOLD_200__", cGold200,
	"__GOLD_800__", cGold800,
	"__TEAL__", cTeal,
	"__TEAL_50__", cTeal50,
	"__TEAL_700__", cTeal700,
	"__TEAL_DEEP__", cTealDeep,
	"__RED__", cRed,
	"__INK__", cInk,
	"__BODY__", cBody,
	"__MUTED__", cMuted,
	"__LINE__", cLine,
	"__PAGE__", cPage,
	"__BORDER__", cBorder,
	"__SANS__", fSans,
	"__MONO__", fMono,
	"__P__", pStyle,
)

// tone styles a callout box. template.CSS marks the values as trusted so
// html/template does not filter them inside style attributes.
type tone struct {
	Bg, Border, Accent template.CSS
}

var (
	toneInfo   = tone{Bg: cTeal50, Border: cTeal200, Accent: cTeal700}
	toneDanger = tone{Bg: cRed50, Border: cRed200, Accent: cRed}
)

type step struct {
	Title string
	Body  string
}

// emailData is the single view model shared by every template.
type emailData struct {
	Title     string // <title>, also the subject
	Preheader string // inbox preview text
	Reason    string // footer: why the recipient got this email
	Year      int

	Eyebrow     string
	Heading     string
	Name        string
	Intro       string
	ButtonLabel string
	Link        string
	Expiry      string

	Steps []step

	CalloutTitle string
	CalloutBody  string
	Tone         tone

	// Developer invitation details.
	MerchantName string
	InviterName  string
	Permissions  []string
}

// message is a fully rendered email.
type message struct {
	Subject string
	HTML    string
	Text    string
}

// ---------------------------------------------------------------------------
// Layout and shared partials
// ---------------------------------------------------------------------------

const layoutSrc = `{{define "layout"}}<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="x-apple-disable-message-reformatting">
<meta name="color-scheme" content="light only">
<meta name="supported-color-schemes" content="light only">
<title>{{.Title}}</title>
<link href="https://fonts.googleapis.com/css2?family=Kanit:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
body { margin:0; padding:0; width:100% !important; -webkit-text-size-adjust:100%; -ms-text-size-adjust:100%; }
table { border-collapse:collapse; }
a { color:__GOLD_DARK__; }
@media only screen and (max-width:620px) {
  .container { width:100% !important; }
  .px { padding-left:24px !important; padding-right:24px !important; }
  .h1 { font-size:26px !important; }
  .btn a { display:block !important; }
}
</style>
</head>
<body style="margin:0;padding:0;background-color:__PAGE__;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;mso-hide:all;font-size:1px;line-height:1px;color:__PAGE__;">{{.Preheader}}&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;&#8204;&nbsp;</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="__PAGE__" style="background-color:__PAGE__;">
<tr><td align="center" style="padding:32px 16px;">
  <table role="presentation" class="container" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px;max-width:600px;">

    <!-- Brand header -->
    <tr><td style="padding:0 4px 18px 4px;">
      <table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>
        <td width="36" height="36" align="center" valign="middle" bgcolor="__GOLD__" style="width:36px;height:36px;border-radius:10px;background-color:__GOLD__;font-family:__SANS__;font-size:20px;font-weight:700;line-height:36px;color:__INK__;">P</td>
        <td style="padding-left:12px;font-family:__SANS__;font-size:20px;font-weight:700;letter-spacing:0.06em;text-transform:uppercase;color:__INK__;">PayConnect</td>
      </tr></table>
    </td></tr>

    <!-- Card -->
    <tr><td style="background-color:#ffffff;border:1px solid __BORDER__;border-radius:16px;overflow:hidden;">
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
        <tr><td style="font-size:0;line-height:0;">
          <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>
            <td width="80%" height="6" bgcolor="__GOLD__" style="height:6px;font-size:0;line-height:6px;background-color:__GOLD__;">&nbsp;</td>
            <td width="20%" height="6" bgcolor="__TEAL__" style="height:6px;font-size:0;line-height:6px;background-color:__TEAL__;">&nbsp;</td>
          </tr></table>
        </td></tr>
        <tr><td class="px" style="padding:36px 40px 40px 40px;font-family:__SANS__;color:__BODY__;">
          {{template "content" .}}
        </td></tr>
      </table>
    </td></tr>

    <!-- Footer -->
    <tr><td class="px" style="padding:24px 12px 0 12px;font-family:__SANS__;font-size:12px;line-height:1.7;color:__MUTED__;text-align:center;">
      <p style="margin:0 0 8px 0;">{{.Reason}}</p>
      <p style="margin:0 0 8px 0;">PayConnect will never ask for your password or API keys by email.</p>
      <p style="margin:0;">&copy; {{.Year}} PayConnect. All rights reserved.</p>
    </td></tr>

  </table>
</td></tr>
</table>
</body>
</html>{{end}}`

const partialsSrc = `
{{define "heading"}}<span style="display:inline-block;padding:7px 12px;background-color:__GOLD_100__;border-radius:999px;font-family:__SANS__;font-size:12px;font-weight:600;letter-spacing:0.08em;text-transform:uppercase;color:__GOLD_800__;">{{.Eyebrow}}</span>
<h1 class="h1" style="margin:18px 0 14px 0;font-family:__SANS__;font-size:30px;line-height:1.1;font-weight:700;letter-spacing:-0.01em;text-transform:uppercase;color:__INK__;">{{.Heading}}</h1>{{end}}

{{define "button"}}<table role="presentation" class="btn" cellpadding="0" cellspacing="0" border="0" style="margin:28px 0 8px 0;"><tr>
<td align="center" bgcolor="__GOLD__" style="border-radius:10px;background-color:__GOLD__;">
<a href="{{.Link}}" target="_blank" style="display:inline-block;padding:15px 34px;border-radius:10px;border-bottom:3px solid __GOLD_DARK__;font-family:__SANS__;font-size:16px;font-weight:600;letter-spacing:0.02em;color:#1a1300;text-decoration:none;">{{.ButtonLabel}}</a>
</td></tr></table>{{end}}

{{define "expiry"}}<p style="margin:14px 0 0 0;font-family:__SANS__;font-size:13px;line-height:1.5;color:__MUTED__;">This link expires in <strong style="color:__INK__;">{{.Expiry}}</strong>.</p>{{end}}

{{define "linkbox"}}<p style="margin:28px 0 8px 0;font-family:__SANS__;font-size:13px;line-height:1.5;color:__MUTED__;">Button not working? Copy and paste this link into your browser:</p>
<p style="margin:0;padding:12px 14px;background-color:#f8fafc;border:1px solid __LINE__;border-radius:8px;font-family:__MONO__;font-size:12px;line-height:1.6;color:__BODY__;word-break:break-all;"><a href="{{.Link}}" style="color:__BODY__;text-decoration:none;">{{.Link}}</a></p>{{end}}

{{define "callout"}}<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:28px 0 0 0;"><tr>
<td style="padding:14px 16px;background-color:{{.Tone.Bg}};border:1px solid {{.Tone.Border}};border-left:4px solid {{.Tone.Accent}};border-radius:8px;">
<p style="margin:0 0 4px 0;font-family:__SANS__;font-size:14px;font-weight:600;color:{{.Tone.Accent}};">{{.CalloutTitle}}</p>
<p style="margin:0;font-family:__SANS__;font-size:14px;line-height:1.6;color:__BODY__;">{{.CalloutBody}}</p>
</td></tr></table>{{end}}
`

// ---------------------------------------------------------------------------
// Per-email content
// ---------------------------------------------------------------------------

const verificationSrc = `{{define "content"}}{{template "heading" .}}
<p style="__P__">Hello {{.Name}},</p>
<p style="__P__">{{.Intro}}</p>
{{template "button" .}}
{{template "expiry" .}}

<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:32px 0 0 0;border-top:1px solid __LINE__;">
  <tr><td style="padding:22px 0 6px 0;font-family:__SANS__;font-size:12px;font-weight:600;letter-spacing:0.08em;text-transform:uppercase;color:__MUTED__;">What happens next</td></tr>
  {{range $i, $s := .Steps}}<tr><td style="padding:10px 0;">
    <table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>
      <td width="28" height="28" align="center" valign="middle" bgcolor="__GOLD_100__" style="width:28px;height:28px;border-radius:14px;background-color:__GOLD_100__;font-family:__SANS__;font-size:13px;font-weight:600;line-height:28px;color:__GOLD_800__;">{{inc $i}}</td>
      <td style="padding-left:14px;font-family:__SANS__;font-size:15px;line-height:1.5;color:__INK__;"><strong>{{$s.Title}}</strong><br><span style="font-size:14px;color:__MUTED__;">{{$s.Body}}</span></td>
    </tr></table>
  </td></tr>{{end}}
</table>

{{template "linkbox" .}}
{{template "callout" .}}
{{end}}`

const passwordResetSrc = `{{define "content"}}{{template "heading" .}}
<p style="__P__">Hello {{.Name}},</p>
<p style="__P__">{{.Intro}}</p>
{{template "button" .}}
{{template "expiry" .}}
{{template "callout" .}}
{{template "linkbox" .}}
{{end}}`

const invitationSrc = `{{define "content"}}{{template "heading" .}}
<p style="__P__">Hello {{.Name}},</p>
<p style="__P__">{{.Intro}}</p>

<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:22px 0 0 0;background-color:__GOLD_50__;border:1px solid __GOLD_200__;border-radius:12px;">
<tr><td style="padding:18px 20px;font-family:__SANS__;">
  {{if .MerchantName}}<p style="margin:0 0 2px 0;font-size:11px;font-weight:600;letter-spacing:0.08em;text-transform:uppercase;color:__GOLD_800__;">Workspace</p>
  <p style="margin:0 0 14px 0;font-size:16px;font-weight:600;color:__INK__;">{{.MerchantName}}</p>{{end}}
  {{if .InviterName}}<p style="margin:0 0 2px 0;font-size:11px;font-weight:600;letter-spacing:0.08em;text-transform:uppercase;color:__GOLD_800__;">Invited by</p>
  <p style="margin:0 0 14px 0;font-size:16px;color:__INK__;">{{.InviterName}}</p>{{end}}
  <p style="margin:0 0 2px 0;font-size:11px;font-weight:600;letter-spacing:0.08em;text-transform:uppercase;color:__GOLD_800__;">Role</p>
  <p style="margin:0;font-size:16px;color:__INK__;">Developer</p>
  {{if .Permissions}}<p style="margin:14px 0 6px 0;font-size:11px;font-weight:600;letter-spacing:0.08em;text-transform:uppercase;color:__GOLD_800__;">Permissions</p>
  <p style="margin:0;line-height:2.1;">{{range .Permissions}}<span style="display:inline-block;margin:0 6px 0 0;padding:3px 9px;background-color:#ffffff;border:1px solid __GOLD_200__;border-radius:6px;font-family:__MONO__;font-size:12px;line-height:1.4;color:__GOLD_800__;">{{.}}</span>{{end}}</p>{{end}}
</td></tr></table>

{{template "button" .}}
{{template "expiry" .}}
{{template "callout" .}}
{{template "linkbox" .}}
{{end}}`

// ---------------------------------------------------------------------------
// Compilation and rendering
// ---------------------------------------------------------------------------

func compile(name, content string) *template.Template {
	t := template.New(name).Funcs(template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	})
	for _, src := range []string{layoutSrc, partialsSrc, content} {
		template.Must(t.Parse(tokens.Replace(src)))
	}
	return t
}

var (
	verificationTmpl  = compile("verification", verificationSrc)
	passwordResetTmpl = compile("password_reset", passwordResetSrc)
	invitationTmpl    = compile("developer_invitation", invitationSrc)
)

func renderHTML(t *template.Template, d emailData) (string, error) {
	d.Year = time.Now().UTC().Year()
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", d); err != nil {
		return "", fmt.Errorf("mailer: render %s template: %w", t.Name(), err)
	}
	return buf.String(), nil
}

func greetingName(name string) string {
	if n := strings.TrimSpace(name); n != "" {
		return n
	}
	return "there"
}

func renderVerification(name, link string) (message, error) {
	const subject = "Verify your PayConnect email"
	d := emailData{
		Title:       subject,
		Preheader:   "Confirm your email to activate your PayConnect account.",
		Reason:      "You received this email because someone signed up for PayConnect with this address.",
		Eyebrow:     "Welcome aboard",
		Heading:     "Confirm your email",
		Name:        greetingName(name),
		Intro:       "Thanks for signing up. Confirm your email address to activate your PayConnect account and start connecting your payment providers.",
		ButtonLabel: "Verify email",
		Link:        link,
		Expiry:      "1 hour",
		Steps: []step{
			{"Verify your email", "One click on the button above activates your account."},
			{"Set up your business", "Add your business details and default currency."},
			{"Connect and go live", "Link a provider, create an application and generate your API keys."},
		},
		Tone:         toneInfo,
		CalloutTitle: "Didn't create an account?",
		CalloutBody:  "You can safely ignore this email. No account will be activated unless this link is used.",
	}
	h, err := renderHTML(verificationTmpl, d)
	if err != nil {
		return message{}, err
	}
	text := fmt.Sprintf(`Hello %s,

Thanks for signing up. Confirm your email address to activate your PayConnect account:

%s

This link expires in %s.

What happens next:
1. Verify your email
2. Set up your business
3. Connect a provider and generate your API keys

If you didn't create a PayConnect account, you can safely ignore this email.

PayConnect will never ask for your password or API keys by email.
`, d.Name, link, d.Expiry)
	return message{Subject: subject, HTML: h, Text: text}, nil
}

func renderPasswordReset(name, link string) (message, error) {
	const subject = "Reset your PayConnect password"
	d := emailData{
		Title:        subject,
		Preheader:    "Use this link to choose a new PayConnect password.",
		Reason:       "You received this email because a password reset was requested for your PayConnect account.",
		Eyebrow:      "Account security",
		Heading:      "Reset your password",
		Name:         greetingName(name),
		Intro:        "We received a request to reset the password for your PayConnect account. Use the button below to choose a new one.",
		ButtonLabel:  "Reset password",
		Link:         link,
		Expiry:       "30 minutes",
		Tone:         toneDanger,
		CalloutTitle: "Didn't request this?",
		CalloutBody:  "Ignore this email and your current password will keep working. If you keep receiving these, consider changing your password from the dashboard and contact support.",
	}
	h, err := renderHTML(passwordResetTmpl, d)
	if err != nil {
		return message{}, err
	}
	text := fmt.Sprintf(`Hello %s,

We received a request to reset the password for your PayConnect account. Choose a new password here:

%s

This link expires in %s.

If you didn't request this, ignore this email and your current password will keep working.

PayConnect will never ask for your password or API keys by email.
`, d.Name, link, d.Expiry)
	return message{Subject: subject, HTML: h, Text: text}, nil
}

func renderDeveloperInvitation(inv DeveloperInvitation) (message, error) {
	const subject = "Join PayConnect as a developer"

	intro := "You have been invited to join a PayConnect merchant workspace as a developer. Accept the invitation to get access to its applications and API keys."
	if m := strings.TrimSpace(inv.MerchantName); m != "" {
		intro = fmt.Sprintf("You have been invited to join %s on PayConnect as a developer. Accept the invitation to get access to its applications and API keys.", m)
	}

	d := emailData{
		Title:        subject,
		Preheader:    "You've been invited to build on PayConnect. Accept within 7 days.",
		Reason:       "You received this email because someone invited this address to a PayConnect workspace.",
		Eyebrow:      "Developer invitation",
		Heading:      "You're invited to build",
		Name:         greetingName(inv.Name),
		Intro:        intro,
		ButtonLabel:  "Accept invitation",
		Link:         inv.Link,
		Expiry:       "7 days",
		MerchantName: strings.TrimSpace(inv.MerchantName),
		InviterName:  strings.TrimSpace(inv.InviterName),
		Permissions:  inv.Permissions,
		Tone:         toneInfo,
		CalloutTitle: "Not expecting this?",
		CalloutBody:  "You can ignore this email. No access is granted until you accept the invitation.",
	}
	h, err := renderHTML(invitationTmpl, d)
	if err != nil {
		return message{}, err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Hello %s,\n\n%s\n\n", d.Name, intro)
	if d.MerchantName != "" {
		fmt.Fprintf(&b, "Workspace: %s\n", d.MerchantName)
	}
	if d.InviterName != "" {
		fmt.Fprintf(&b, "Invited by: %s\n", d.InviterName)
	}
	b.WriteString("Role: Developer\n")
	if len(d.Permissions) > 0 {
		fmt.Fprintf(&b, "Permissions: %s\n", strings.Join(d.Permissions, ", "))
	}
	fmt.Fprintf(&b, "\nAccept the invitation:\n%s\n\nThis link expires in %s.\n\n", inv.Link, d.Expiry)
	b.WriteString("Not expecting this? Ignore this email; no access is granted until you accept.\n\n")
	b.WriteString("PayConnect will never ask for your password or API keys by email.\n")

	return message{Subject: subject, HTML: h, Text: b.String()}, nil
}
