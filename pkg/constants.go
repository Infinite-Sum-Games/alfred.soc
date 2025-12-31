package pkg

// Streams for handling Event Driven Architecture
const (
	// The bot command stream is used for participant claiming issues.
	// These include the following:
	// 1. Participants : Claiming issues
	// 2. Participants : Unclaiming issues
	// 3. Maintainers : Marking issue for "bug" badge
	//
	// Producer: Alfred (Webhooks)
	// Consumer: DevPool (GitHub App), Gravemind (Workflows)
	IssueClaim = "issue-stream"

	// All automatic workflows are managed by this stream. This includes :
	// 2. Maintainers : Adding new issues
	//
	// Producer: Alfred (Webhooks)
	// Consumer: DevPool (GitHub App), Gravemind (Workflows)
	AutomaticEvents = "automatic-events-stream"

	// The bounty stream is used to process bounties and penalties received by
	// maintainers from maintainers. A maintainer-repo map is to be accessed for
	// for making sure that only a maintainer had made the comment
	// Producer: Alfred (Webhooks)
	// Consumers: DevPool (GitHub App), Gravemind (Workflows)
	Bounty = "bounty-stream"

	// Whenever a pull request is merged by a maintainer it is captured here for
	// running further workflows on badge distribution.
	//
	// Producer: Alfred (Webhooks)
	// Consumer: Gravemind (Workflows)
	SolutionMerge = "solution-merged-stream"

	// The live-update-stream is used to supply events to the SSE endpoint on
	// leaderboard via the Pulse API server. It handles the following events:
	// 1. Rank Top 3
	// 2. Bounty Dispatch
	// 3. Issue Claimed
	// 4. Issue Accepted (normal, bug-report)
	// 5. Pull Request Opened
	// 6. Pull Request Merged
	// Producers: Alfred (Webhooks), DevPool (GitHub App), Gravemind (Workflows)
	// Consumer: Pulse (API Server)
	LiveUpdates = "live-update-stream"
)

// HashSets for normal badges. These act like buckets grouping participants
// and increasing their counter when more actions are performed in the same.
const (
	DocSet      = "doc-set"
	BugSet      = "bug-hunter-set"
	LanguageSet = "language-set"
	HelpSet     = "helper-set"
	TestSet     = "testing-set"
	FeatSet     = "feature-suggestion-set"

	// This hashset behaves slightly differently. Here, there is an
	// eviction policy so as to maintain the streak. Not being used currently
	// EnamouredSet = "enamoured-set"
)

// SortedSets to handle leaderboard, language badges and
// Pirate of Issue-bians badge
const (
	Leaderboard = "leaderboard-sset"
	CppRank     = "cpp-ranking-sset"
	JavaRank    = "java-ranking-sset"
	PyRank      = "python-ranking-sset"
	JsRank      = "javascript-ranking-sset"
	GoRank      = "go-ranking-sset"
	RustRank    = "rust-ranking-sset"
	// ZigRank     = "zig-ranking-sset"
	FlutterRank = "flutter-ranking-sset"
	KotlinRank  = "kotlin-ranking-sset"
	HaskellRank = "haskell-ranking-sset"
)
