package application

// Input describes the mutable application fields plus the one-time settings
// that shape the keys minted at creation.
type Input struct {
	Name        string
	Description string
	Status      string
	// KeyScope is "general" (team-shared, the default) or "developer".
	KeyScope string
	// DeveloperAccessID is required when KeyScope is "developer": it names the
	// grant the keys belong to.
	DeveloperAccessID string
}

// ListFilter narrows applications by their owner and optional application scope.
type ListFilter struct {
	MerchantID    string
	ApplicationID string
	Limit         int
}

// CreatedResult contains the application and the one-time secrets of the keys
// minted with it. APIKey mirrors the test key for older clients.
type CreatedResult struct {
	Application *Application
	APIKey      string
	Keys        []CreatedKey
}

// CreatedKey is one minted key: its environment, scope and one-time secret.
type CreatedKey struct {
	ID          string
	Environment string
	Scope       string
	Secret      string
}
