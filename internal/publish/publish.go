// Package publish moves content from the working tree to where it is served.
//
// Publishing is two steps on purpose. Preflight only looks, and reports
// everything it found wrong at once; Apply acts on a plan that has already
// been examined. Git publishing has many foreseeable failures -- a detached
// head, a merge in progress, missing credentials, a remote that moved -- and
// discovering one of them halfway through leaves a repository in a state
// nobody asked for.
package publish

import (
	"context"
	"time"
)

// Step is how far one stage of delivery has got.
type Step string

const (
	// StepPending means the step has not run yet.
	StepPending Step = "pending"
	StepDone    Step = "done"
	StepFailed  Step = "failed"

	// StepNotApplicable is used where a step has no meaning for this
	// publisher, such as committing when the store is a database.
	StepNotApplicable Step = "not_applicable"
)

// DeliveryState is how far content has actually traveled.
//
// It is deliberately not part of an item's status. Status is the author's
// intent -- "this should be public" -- and says nothing about whether a
// commit was made, a push succeeded or a deployment finished. Folding the two
// together is the mistake that makes "published" mean two different things on
// the same screen.
type DeliveryState struct {
	Local     Step `json:"local"`
	Committed Step `json:"committed"`
	Pushed    Step `json:"pushed"`
	Deployed  Step `json:"deployed"`

	// Branch and Remote say where this repository publishes to.
	Branch string `json:"branch,omitempty"`
	Remote string `json:"remote,omitempty"`

	// Ahead and Behind count commits not yet pushed, and commits on the
	// remote this branch has not seen.
	Ahead  int `json:"ahead"`
	Behind int `json:"behind"`

	// Dirty lists paths Kite can see are uncommitted, so a panel can say what
	// a publish would carry.
	Dirty []string `json:"dirty,omitempty"`

	// DeployedURL is where the host says the deployment is live, once it
	// has said so.
	DeployedURL string `json:"deployed_url,omitempty"`

	LastError *Problem  `json:"last_error,omitempty"`
	CheckedAt time.Time `json:"checked_at,omitzero"`
}

// Problem is something a publish found, whether or not it stops the publish.
type Problem struct {
	// Code is stable and machine readable; Detail and Fix are for a person.
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

func (p Problem) Error() string { return p.Detail }

// Problem codes. A client switches on these; the wording may change.
const (
	CodeNotARepository    = "not_a_repository"
	CodeDetachedHead      = "detached_head"
	CodeOperationInFlight = "operation_in_flight"
	CodeSubmodule         = "submodule"
	CodeLFSMissing        = "lfs_missing"
	CodeNothingToPublish  = "nothing_to_publish"
	CodeNoRemote          = "no_remote"
	CodeNoCredentials     = "no_credentials"
	CodeRemoteMoved       = "remote_moved"
	CodeStagedElsewhere   = "staged_elsewhere"
	CodePathNotChanged    = "path_not_changed"
	CodeQuotaExceeded     = "quota_exceeded"
	CodeLocked            = "locked"
	CodeGitMissing        = "git_missing"
	CodeGitFailed         = "git_failed"
	CodeNothingToPush     = "nothing_to_push"
	CodeHookRefused       = "hook_refused"

	// Why an unpushed commit cannot be replayed on top of a remote that has
	// moved on. See [RemoteChange].
	CodeRemoteOverlap    = "remote_overlap"
	CodeUnpushedCommits  = "unpushed_commits"
	CodeLocalChanges     = "local_changes"
	CodeUnrelatedHistory = "unrelated_history"
)

// Plan is what a publish would do, produced by looking and changing nothing.
type Plan struct {
	Publisher string `json:"publisher"`
	Branch    string `json:"branch,omitempty"`
	Remote    string `json:"remote,omitempty"`
	Message   string `json:"message"`

	// Paths are exactly what will be committed, project relative and slash
	// separated. Nothing outside this list is touched, which is what lets a
	// publish run while the author has other work in progress.
	Paths []string `json:"paths"`

	// Problems stop the publish; Warnings do not, but a person should see
	// them first. Both are collected rather than returned one at a time, so
	// a repository that needs three things fixed says so once.
	Problems []Problem `json:"problems,omitempty"`
	Warnings []Problem `json:"warnings,omitempty"`

	// Push says whether the plan will reach the remote, which it will not
	// when there is no remote configured.
	Push bool `json:"push"`

	// SkipHooks says the commit will be made without running the
	// repository's commit hooks, as the request asked.
	SkipHooks bool `json:"skip_hooks,omitempty"`
}

// OK reports whether a plan can be applied.
func (p *Plan) OK() bool { return len(p.Problems) == 0 && len(p.Paths) > 0 }

// Result reports what a publish did.
type Result struct {
	Commit    string    `json:"commit,omitempty"`
	Committed []string  `json:"committed,omitempty"`
	Pushed    bool      `json:"pushed"`
	At        time.Time `json:"at,omitzero"`

	// Rebased says the commit was replayed on top of the remote before it
	// was pushed, so Commit is not the hash it was first made with.
	Rebased bool `json:"rebased,omitempty"`

	// Remote is what the remote had that this branch did not, when that is
	// why a push was refused.
	Remote *RemoteChange `json:"remote,omitempty"`
}

// RemoteChange is what a remote has that this branch does not.
//
// A push is never forced and a remote's commits are never merged on the
// author's behalf. What can be offered is narrower: when the remote changed
// nothing the unpushed commit changed, that commit can be replayed on top of
// the remote, which is what a person would do with a rebase.
type RemoteChange struct {
	// Upstream names the remote branch, as in origin/main, and Behind is how
	// many commits it has that this branch does not. Commits lists the
	// newest of them, newest first.
	Upstream string   `json:"upstream"`
	Behind   int      `json:"behind"`
	Commits  []Commit `json:"commits"`

	// Overlap lists the files the unpushed commit changed that the remote
	// changed too, and Diff is the remote's side of them, for a person to
	// read before deciding what to keep. Diff is cut short when very long.
	Overlap []string `json:"overlap,omitempty"`
	Diff    string   `json:"diff,omitempty"`

	// Rebase says whether the commit can be replayed on top of the remote;
	// when it cannot, Blocked says why.
	Rebase  bool     `json:"rebase"`
	Blocked *Problem `json:"blocked,omitempty"`
}

// Commit is one commit, as a person would recognize it.
type Commit struct {
	Hash    string    `json:"hash"`
	Subject string    `json:"subject"`
	Author  string    `json:"author"`
	At      time.Time `json:"at"`
}

// Request is what a caller wants published.
type Request struct {
	// Paths are project relative and slash separated. A caller resolves an
	// item to its files; the publisher never decides for itself what "this
	// post" means on disk.
	Paths []string

	// Message is the commit message. Empty means the publisher composes one.
	Message string

	// Push sends the commit to the remote. A publish that only commits is
	// still a publish: whether it should leave the machine is the caller's
	// decision, and must be an explicit one.
	Push bool

	// Force allows applying a plan that carries warnings.
	Force bool

	// SkipHooks commits without running the repository's commit hooks, for
	// an author who has read why a hook refused and decided to publish
	// anyway. Never assumed: a hook is the repository owner's rule.
	SkipHooks bool
}

// PushRequest is what a caller wants done with commits already made.
type PushRequest struct {
	// Rebase replays the one unpushed commit on top of a remote that has
	// moved on, when the remote changed nothing that commit changed.
	Rebase bool
}

// Publisher moves a set of paths to where they are served.
type Publisher interface {
	Name() string

	// Preflight examines the repository and reports what a publish would do.
	// It must have no side effects whatsoever.
	Preflight(ctx context.Context, req Request) (*Plan, error)

	// Apply carries out a plan that Preflight produced.
	Apply(ctx context.Context, plan *Plan) (*Result, error)

	// Push sends what is already committed, for a publish whose commit
	// was made but whose push did not go through.
	Push(ctx context.Context, req PushRequest) (*Result, error)

	// State reports how far the content has traveled.
	State(ctx context.Context) (*DeliveryState, error)
}
