package bootstrap

import (
	"fmt"

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
	{Name: pkg.DocSet, Type: "hash"},
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
	{Name: pkg.FlutterRank, Type: "zset"},
	{Name: pkg.KotlinRank, Type: "zset"},
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
		pkg.Log.SetupFail("Valkey client is nil", nil)
		return false
	}

	// Streams
	existStreams, err := VerifyStreams(streamNames, client)
	if err != nil {
		pkg.Log.SetupFail("Failed to verify streams", err)
		return false
	}
	var missingStreams []string
	for name, exists := range existStreams {
		if !exists {
			pkg.Log.SetupInfo(
				fmt.Sprintf("[ISSUE]: Missing stream %s. Attempting to create.", name),
			)
			missingStreams = append(missingStreams, name)
			continue
		}
		pkg.Log.SetupInfo(fmt.Sprintf("[OK-STREAM]: %s exist.", name))
	}
	if len(missingStreams) > 0 {
		if err := SetupValkeyStreams(missingStreams, client); err != nil {
			pkg.Log.SetupFail("Failed to setup missing streams", err)
			return false
		}
		pkg.Log.SetupInfo("[DONE]: Created missing streams")
	}

	// HashSets
	existHashes, err := VerifyHSet(hashSetNames, client)
	if err != nil {
		pkg.Log.SetupFail("Failed to verify hash sets", err)
		return false
	}
	var missingHashes []string
	for name, exists := range existHashes {
		if !exists {
			pkg.Log.SetupInfo(
				fmt.Sprintf("[ISSUE]: Missing hash-set %s. Attempting to create.", name),
			)
			missingHashes = append(missingHashes, name)
			continue
		}
		pkg.Log.SetupInfo(fmt.Sprintf("[OK-HSET]: %s exist.", name))
	}
	if len(missingHashes) > 0 {
		if err := SetupValkeyHSet(missingHashes, client); err != nil {
			pkg.Log.SetupFail("Failed to setup missing hash sets", err)
			return false
		}
		pkg.Log.SetupInfo("[DONE]: Created missing hash sets")
	}

	// SortedSets
	existZSets, err := VerifySSet(sortedSetNames, client)
	if err != nil {
		pkg.Log.SetupFail("Failed to verify sorted sets", err)
		return false
	}
	var missingZSets []string
	for name, exists := range existZSets {
		if !exists {
			pkg.Log.SetupInfo(
				fmt.Sprintf("[ISSUE]: Missing sorted-set %s. Attempting to create.", name),
			)
			missingZSets = append(missingZSets, name)
			continue
		}
		pkg.Log.SetupInfo(fmt.Sprintf("[OK-SORTED-SET]: %s exist.", name))
	}
	if len(missingZSets) > 0 {
		if err := SetupValkeySSet(missingZSets, client); err != nil {
			pkg.Log.SetupFail("Failed to setup missing sorted sets", err)
			return false
		}
		pkg.Log.SetupInfo("[DONE]: Created missing sorted sets")
	}

	pkg.Log.SetupInfo("[ACTIVE]: Valkey bootstrap complete")
	return true
}
