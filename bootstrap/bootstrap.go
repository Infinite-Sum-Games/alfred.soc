package bootstrap

import (
	"github.com/IAmRiteshKoushik/alfred/pkg"
)

type StructureInfo struct {
	Name string
	Type string
}

var Structures = []StructureInfo{
	// Streams
	{Name: pkg.Admin, Type: "stream"},
	{Name: pkg.IssueClaim, Type: "stream"},
	{Name: pkg.AutomaticEvents, Type: "stream"},
	{Name: pkg.Bounty, Type: "stream"},
	{Name: pkg.SolutionMerge, Type: "stream"},
	{Name: pkg.LiveUpdates, Type: "stream"},

	// HashSets
	{Name: pkg.BugSet, Type: "hash"},
	{Name: pkg.LanguageSet, Type: "hash"},
	{Name: pkg.HelpSet, Type: "hash"},
	{Name: pkg.TestSet, Type: "hash"},
	{Name: pkg.FeatSet, Type: "hash"},
	{Name: pkg.EnamouredSet, Type: "hash"},

	// SortedSets
	{Name: pkg.Leaderboard, Type: "zset"},
	{Name: pkg.CppRank, Type: "zset"},
	{Name: pkg.JavaRank, Type: "zset"},
	{Name: pkg.PyRank, Type: "zset"},
	{Name: pkg.JsRank, Type: "zset"},
	{Name: pkg.GoRank, Type: "zset"},
	{Name: pkg.RustRank, Type: "zset"},
	{Name: pkg.ZigRank, Type: "zset"},
}

// This program checks against all the existing structures required for
// starting Valkey. If the structure does not exist, then it creates the
// required structure.
func BootstrapValkey() bool {
	var (
		streamNames    []string
		hashSetNames   []string
		sortedSetNames []string
	)

	for _, s := range Structures {
		switch s.Type {
		case "stream":
			streamNames = append(streamNames, s.Name)
		case "hash":
			hashSetNames = append(hashSetNames, s.Name)
		case "zset":
			sortedSetNames = append(sortedSetNames, s.Name)
		}
	}

	client := pkg.Valkey
	if client == nil {
		return false
	}

	// Streams
	existStreams, err := VerifyStreams(streamNames, client)
	if err != nil {
		return false
	}
	var missingStreams []string
	for name, exists := range existStreams {
		if !exists {
			missingStreams = append(missingStreams, name)
		}
	}
	if len(missingStreams) > 0 {
		if err := SetupValkeyStreams(missingStreams, client); err != nil {
			return false
		}
	}

	// HashSets
	existHashes, err := VerifyHSet(hashSetNames, client)
	if err != nil {
		return false
	}
	var missingHashes []string
	for name, exists := range existHashes {
		if !exists {
			missingHashes = append(missingHashes, name)
		}
	}
	if len(missingHashes) > 0 {
		if err := SetupValkeyHSet(missingHashes, client); err != nil {
			return false
		}
	}

	// SortedSets
	existZSets, err := VerifySSet(sortedSetNames, client)
	if err != nil {
		return false
	}
	var missingZSets []string
	for name, exists := range existZSets {
		if !exists {
			missingZSets = append(missingZSets, name)
		}
	}
	if len(missingZSets) > 0 {
		if err := SetupValkeySSet(missingZSets, client); err != nil {
			return false
		}
	}

	return true
}
