package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/inconshreveable/log15"
	"github.com/pelletier/go-toml"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/lsif/correlation"
	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// TODO:
//   We need to check for lsif-clang, lsif-validate at start time.

type ProjectResult struct {
	success     bool
	memoryUsage int64
	output      string
}

type IndexerResult struct {
	memoryUsage int64
	output      []byte
}

func main() {
	logging.Init()
	trace.Init(false)

	log15.Root().SetHandler(log15.StdoutHandler)

	var directory string
	flag.StringVar(&directory, "dir", ".", "The directory to run the test harness over")

	var indexer string
	flag.StringVar(&indexer, "indexer", "", "The name of the indexer that you want to test")

	var monitor bool
	flag.BoolVar(&monitor, "monitor", true, "Whether to monitor and log stats")

	flag.Parse()

	if indexer == "" {
		log.Fatalf("Indexer is required. Pass with --indexer")
	}

	log15.Info("Starting Execution: ", "directory", directory, "indexer", indexer)

	// w, _ := os.Create("monitor.csv")
	// testContext = context.WithValue(testContext, "output", w)

	testContext := context.Background()
	testContext = context.WithValue(testContext, "monitor", monitor)

	err := testDirectory(testContext, indexer, directory)
	if err != nil {
		log.Fatalf("Failed with: %s", err)
	}
}

func testDirectory(ctx context.Context, indexer string, directory string) error {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, f := range files {
		log15.Info("Running test for: ", "file", f.Name())
		projResult, err := testProject(ctx, indexer, directory+"/"+f.Name())
		if err != nil {
			fmt.Printf("Project %+v", projResult)
			log.Fatalf("Project '%s' failed to complete", f.Name())
		}

		log15.Info("Project result:", "success", projResult.success)
		if !projResult.success {
			log.Fatalf("Project '%s' failed test", f.Name())
		}
	}

	return nil
}

func testProject(ctx context.Context, indexer, project string) (ProjectResult, error) {

	var result IndexerResult
	var err error

	if indexer == "lsif-clang" {
		log15.Debug("... Starting lsif clang")
		result, err = testLsifClang(ctx, project)
		if err != nil {
			return ProjectResult{
				success:     false,
				memoryUsage: result.memoryUsage,
				output:      string(result.output),
			}, err
		}
	}

	output, err := validateDump(project)
	if err != nil {
		fmt.Println("Not valid")
		return ProjectResult{
			success:     false,
			memoryUsage: result.memoryUsage,
			output:      string(output),
		}, err
	} else {
		log15.Debug("... Validated dump.lsif")
	}

	bundle, err := readBundle(1, project)
	fmt.Printf("Bundle: %+v\n", bundle)
	// if err != nil {
	// 	return []byte{}, err
	// }
	// // fmt.Printf("Bundle: %+v\n", bundle)

	path := "src/uses_header.c"
	line := 6
	character := 8

	results, err := correlation.Query(bundle, path, line, character)
	fmt.Printf("Results: %+v\n", results)

	// validateTestCases(directory)

	return ProjectResult{
		success:     true,
		memoryUsage: result.memoryUsage,
		output:      string(output),
	}, nil
}

// TODO this is pretty dumb... we should find some way that isn't hard coded in here to run it.
// But for now I'm going to do it this way.
func testLsifClang(ctx context.Context, project string) (IndexerResult, error) {
	output, err := generateCompileCommands(project)
	if err != nil {
		return IndexerResult{
			memoryUsage: -1,
			output:      output,
		}, err
	} else {
		log15.Debug("... Generated compile_commands.json")
	}

	output, mem, err := generateLsifDump(ctx, project)
	if err != nil {
		log.Println(output)
		log.Fatal(err)
	} else {
		log15.Debug("... Generated dump.lsif")
	}

	return IndexerResult{
		memoryUsage: mem,
		output:      output,
	}, err
}

func generateCompileCommands(directory string) ([]byte, error) {
	cmd := exec.Command("./get_compile_commands.sh")
	cmd.Dir = directory

	return cmd.CombinedOutput()
}

func generateLsifDump(ctx context.Context, directory string) ([]byte, int64, error) {
	cmd := exec.Command("lsif-clang", "compile_commands.json")
	cmd.Dir = directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, -1, err
	}

	sysUsage := cmd.ProcessState.SysUsage()
	mem, _ := MaxMemoryInKB(sysUsage)
	log15.Debug("Max memory", "memory", mem)
	// fmt.Println("Memory Usage:", mem, "kB")
	// fmt.Println("User CPU", sysUsage.Utime)

	return output, mem, err
}

func validateDump(directory string) ([]byte, error) {
	// TODO: Eventually this should use the package, rather than the installed module
	//       but for now this will have to do.
	cmd := exec.Command("lsif-validate", "dump.lsif")
	cmd.Dir = directory

	return cmd.CombinedOutput()
}

func validateTestCases(directory string) {
	doc := []byte(`
		[Definitions]

			[Definitions.example]
			Request.TextDocument = "src/uses_header.c"
			Request.Position.Line = 6
			Request.Position.Character = 8

			[Definitions.other]
			Request.TextDocument = "src/uses_header.c"
			Request.Position.Line = 6
			Request.Position.Character = 8
	`)

	testCase := LsifTest{}
	toml.Unmarshal(doc, &testCase)
	// fmt.Printf("%+v", testCase)
}

func getWriter(ctx context.Context) *os.File {
	return ctx.Value("output").(*os.File)
}
