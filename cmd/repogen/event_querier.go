package main

// EventQuerier holds event queries the generated basic CRUD can't express — the
// DISTINCT lookups that feed the owner / repo filter dropdowns.
type EventQuerier interface {
	// ListOwners returns every distinct owner, sorted ascending.
	//
	// SELECT DISTINCT owner FROM @@table ORDER BY owner
	ListOwners() ([]string, error)

	// ListRepos returns every distinct repo for one owner, sorted ascending.
	//
	// SELECT DISTINCT repo FROM @@table WHERE owner = @owner ORDER BY repo
	ListRepos(owner string) ([]string, error)
}
