package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/lsif/correlation"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/lsif/existence"
)

func readBundle(dumpID int, root string) (*correlation.GroupedBundleDataMaps, error) {
	dumpPath := path.Join(root, "dump.lsif")
	getChildrenFunc := makeExistenceFunc(root)
	file, err := os.Open(dumpPath)
	if err != nil {
		fmt.Printf("Couldn't open file %v\n", dumpPath)
		return nil, err
	}
	defer file.Close()

	bundle, err := correlation.Correlate(context.Background(), file, dumpID, "", getChildrenFunc)
	if err != nil {
		fmt.Println("Correlation failed")
		return nil, err
	}

	return correlation.GroupedBundleDataChansToMaps(context.Background(), bundle), nil
}

func makeExistenceFunc(directory string) existence.GetChildrenFunc {
	return func(ctx context.Context, dirnames []string) (map[string][]string, error) {
		if dirnames[0] == "" {
			dirnames[0] = "."
		}

		// TODO: could probably just walk this in Go...
		cmd := exec.Command("bash", "-c", fmt.Sprintf("find %s -maxdepth 1", strings.Join(dirnames, " ")))
		cmd.Dir = directory

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("find failed with output '%v'\nArgs were '%v'\n", string(out), dirnames)
		}

		res, err := parseDirectoryChildren(dirnames, strings.Split(string(out), "\n")), nil
		fmt.Println("=============")
		fmt.Printf("parseDirectory: %+v\n", res)
		fmt.Println(string(out))
		fmt.Println("=============")
		return res, err
	}
}

// TODO Probably dont want this exactly like this.
func parseDirectoryChildren(dirnames, paths []string) map[string][]string {
	childrenMap := map[string][]string{}

	// Ensure each directory has an entry, even if it has no children
	// listed in the gitserver output.
	for _, dirname := range dirnames {
		childrenMap[dirname] = nil
	}

	// Order directory names by length (biggest first) so that we assign
	// paths to the most specific enclosing directory in the following loop.
	sort.Slice(dirnames, func(i, j int) bool {
		return len(dirnames[i]) > len(dirnames[j])
	})

	for _, path := range paths {
		if strings.Contains(path, "/") {
			for _, dirname := range dirnames {
				if strings.HasPrefix(path, dirname) {
					childrenMap[dirname] = append(childrenMap[dirname], path)
					break
				}
			}
		} else {
			// No need to loop here. If we have a root input directory it
			// will necessarily be the last element due to the previous
			// sorting step.
			if len(dirnames) > 0 && dirnames[len(dirnames)-1] == "" {
				childrenMap[""] = append(childrenMap[""], path)
			}
		}
	}

	// fmt.Println("=============")
	// fmt.Printf("%+v\n", childrenMap)
	// fmt.Println("=============")

	return childrenMap
}
