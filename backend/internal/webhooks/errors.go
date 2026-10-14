package webhooks

import "errors"

// errMissingSigningSecret means an endpoint row lost its encrypted secret; the
// delivery cannot be signed, so it is failed rather than sent unsigned.
var errMissingSigningSecret = errors.New("webhooks: endpoint has no signing secret")
