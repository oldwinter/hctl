package model

import "github.com/oldwinter/hctl/internal/secret"

// desiredFields is the single vocabulary for apply/diff -f / set / sync
// projection. Inventory diffs (DiffSnapshots) keep their own names
// (defaultModel, …) because those describe two snapshots, not a write intent.
var desiredFields = []desiredField{
	{
		name:     "model",
		syncName: "model",
		current:  func(s Snapshot) string { return s.DefaultModel },
		wanted:   func(d Desired) string { return d.Model },
		project:  func(d *Desired, v string) { d.Model = v },
	},
	{
		name:     "provider",
		syncName: "provider",
		current:  func(s Snapshot) string { return s.Provider },
		wanted:   func(d Desired) string { return d.Provider },
		project:  func(d *Desired, v string) { d.Provider = v },
	},
	{
		name:     "baseUrl",
		syncName: "base-url",
		current:  func(s Snapshot) string { return s.BaseURLHost },
		wanted:   func(d Desired) string { return d.BaseURL },
		// Snapshots only expose the endpoint host, so equality and projection
		// both run at host level; the full URL only travels the write path.
		equal: func(want, got string) bool {
			return want == got || secret.HostOf(want) == got
		},
	},
	{
		name:     "secretRef",
		syncName: "secret-ref",
		current:  func(s Snapshot) string { return s.SecretRef },
		wanted:   func(d Desired) string { return d.SecretRef },
		project:  func(d *Desired, v string) { d.SecretRef = v },
	},
}

type desiredField struct {
	name     string
	syncName string
	current  func(Snapshot) string
	wanted   func(Desired) string
	equal    func(want, got string) bool
	project  func(*Desired, string)
}

// Empty reports whether any write field is set.
func (d Desired) Empty() bool {
	for _, f := range desiredFields {
		if f.wanted(d) != "" {
			return false
		}
	}
	return true
}

// ChangesFromDesired is the only desired-vs-snapshot compare. It is used by
// diff -f, mutate.Preflight, and post-write verify.
func ChangesFromDesired(harness string, snap Snapshot, d Desired) []Change {
	if harness == "" {
		harness = snap.Name
	}
	var out []Change
	for _, f := range desiredFields {
		want := f.wanted(d)
		if want == "" {
			continue
		}
		got := f.current(snap)
		if f.equal != nil {
			if f.equal(want, got) {
				continue
			}
		} else if want == got {
			continue
		}
		ch := Change{Harness: harness, Field: f.name, From: got, To: want}
		if len(snap.ConfigPaths) > 0 {
			ch.Path = snap.ConfigPaths[0]
		}
		out = append(out, ch)
	}
	return out
}

// Project copies selected snapshot fields into a Desired write intent.
// names may use desired names (model, secretRef) or sync aliases (secret-ref).
func Project(snap Snapshot, names ...string) Desired {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var d Desired
	for _, f := range desiredFields {
		if f.project == nil {
			continue
		}
		if want[f.name] || want[f.syncName] {
			f.project(&d, f.current(snap))
		}
	}
	return d
}

// KnownSyncField reports whether name is a desired field, its sync alias, or
// the bearer-copy token "secret" (not a Desired field).
func KnownSyncField(name string) bool {
	if name == "secret" {
		return true
	}
	for _, f := range desiredFields {
		if name == f.name || name == f.syncName {
			return true
		}
	}
	return false
}
