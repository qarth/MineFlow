// Command mineflow is a Go port of the MineFlow command-line executable
// (mineflow.cpp:4271-4721, MVD_MINEFLOW_EXE). It computes ultimate pit limits:
//
//	mineflow [options] data.dat output.dat
//
// options:
//
//	--regular <nx> <ny> <nz> <slope>  Use a single constant slope angle (deg)
//	--minsearch <file>                Use a single minimum search pattern
//	--explicit <file>                 Use explicit precedence constraints (slow!)
//
// The --to_dimacs option of the C++ executable is not ported: its body was
// commented out there.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"mineflow"
)

const usage = `Usage: mineflow [options] data.dat output.dat
options:
 --regular <nx> <ny> <nz> <slope> Use a single constant slope angle (deg)
 --minsearch <file>               Use a single minimum search pattern
 --explicit <file>                Use explicit precedence constants (slow!)

minsearch format:
<NumX> <NumY> <NumZ>      # Number of blocks in x, y, and z
<SizeX> <SizeY> <SizeZ>   # Size of blocks in x, y, and z
<NumBenches>              # Number of benches to extent pattern
<Azimuth> <Slope>
<Azimuth> <Slope>
...

explicit format:
<num blocks>
<from_block_id> <to_block_id_0> <to_block_id_1> ...
<from_block_id> <to_block_id_0> <to_block_id_1> ...
...

'data.dat' format:
<value_block_0>
<value_block_1>
...

'output.dat' format:
<mine_block_0>
<mine_block_1>
...`

// elapsed formats a duration as hh:mm:ss.mmm (Elapsed in the C++ code).
func elapsed(d time.Duration) string {
	ms := d.Milliseconds()
	hours := ms / 3600000
	minutes := (ms / 60000) % 60
	seconds := (ms / 1000) % 60
	milliseconds := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
}

type argReader struct {
	args []string
}

func (a *argReader) peek() (string, bool) {
	if len(a.args) == 0 {
		return "", false
	}
	return a.args[0], true
}

func (a *argReader) next() {
	a.args = a.args[1:]
}

func (a *argReader) readString() (string, bool) {
	s, ok := a.peek()
	if !ok {
		return "", false
	}
	a.next()
	return s, true
}

func (a *argReader) readInt() (int, bool) {
	s, ok := a.readString()
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

func (a *argReader) readFloat() (float64, bool) {
	s, ok := a.readString()
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// initRegular ports InitRegular (mineflow.cpp:4348): a constant slope angle
// with a minimum search pattern over 9 benches.
func initRegular(args *argReader) (mineflow.PrecedenceConstraints, int, error) {
	blockDef := mineflow.UnitModel(1, 1, 1)
	var slope float64
	var ok bool
	if blockDef.NumX, ok = args.readInt(); !ok {
		return nil, 0, fmt.Errorf("failed reading numx")
	}
	if blockDef.NumY, ok = args.readInt(); !ok {
		return nil, 0, fmt.Errorf("failed reading numy")
	}
	if blockDef.NumZ, ok = args.readInt(); !ok {
		return nil, 0, fmt.Errorf("failed reading numz")
	}
	if slope, ok = args.readFloat(); !ok {
		return nil, 0, fmt.Errorf("failed reading slope")
	}

	const nBenches = 9 // could be input?
	slopeDef := mineflow.ConstantSlope(mineflow.ToRadians(slope))
	pattern := mineflow.PatternMinSearch(blockDef, slopeDef, nBenches)
	return mineflow.NewRegular3DBlockModelPatternPrecedence(blockDef, pattern), blockDef.NumBlocks(), nil
}

// initMinSearch ports InitMinSearch (mineflow.cpp:4368): a minimum search
// pattern from a file describing the block model and slope definition.
func initMinSearch(args *argReader) (mineflow.PrecedenceConstraints, int, error) {
	minSearchFile, ok := args.readString()
	if !ok {
		return nil, 0, fmt.Errorf("failed reading min search file argument")
	}

	in, err := os.Open(minSearchFile)
	if err != nil {
		return nil, 0, fmt.Errorf("failed opening min search file")
	}
	defer in.Close()

	scanner := bufio.NewScanner(in)
	readFields := func(what string) ([]string, error) {
		if !scanner.Scan() {
			return nil, fmt.Errorf("failed reading %s", what)
		}
		return strings.Fields(scanner.Text()), nil
	}

	blockDef := mineflow.UnitModel(1, 1, 1)

	fields, err := readFields("NumX NumY NumZ")
	if err != nil {
		return nil, 0, err
	}
	if len(fields) != 3 {
		return nil, 0, fmt.Errorf("failed reading NumX NumY NumZ")
	}
	dims := make([]int, 3)
	for i, f := range fields {
		v, err := strconv.ParseInt(f, 10, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("failed reading NumX NumY NumZ")
		}
		dims[i] = int(v)
	}
	blockDef.NumX, blockDef.NumY, blockDef.NumZ = dims[0], dims[1], dims[2]
	if blockDef.NumX <= 0 || blockDef.NumY <= 0 || blockDef.NumZ <= 0 {
		return nil, 0, fmt.Errorf("invalid NumX NumY NumZ")
	}

	fields, err = readFields("SizeX SizeY SizeZ")
	if err != nil {
		return nil, 0, err
	}
	if len(fields) != 3 {
		return nil, 0, fmt.Errorf("failed reading SizeX SizeY SizeZ")
	}
	sizes := make([]float64, 3)
	for i, f := range fields {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("failed reading SizeX SizeY SizeZ")
		}
		sizes[i] = v
	}
	blockDef.SizeX, blockDef.SizeY, blockDef.SizeZ = sizes[0], sizes[1], sizes[2]
	if blockDef.SizeX <= 0 || blockDef.SizeY <= 0 || blockDef.SizeZ <= 0 {
		return nil, 0, fmt.Errorf("invalid SizeX SizeY SizeZ")
	}

	fields, err = readFields("numBenches")
	if err != nil {
		return nil, 0, err
	}
	if len(fields) != 1 {
		return nil, 0, fmt.Errorf("failed reading numBenches")
	}
	numBenches, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("failed reading numBenches")
	}
	if numBenches <= 1 {
		return nil, 0, fmt.Errorf("invalid num benches")
	}

	var pairs []mineflow.AzmSlopePair
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 {
			return nil, 0, fmt.Errorf("failed reading azimuth slope")
		}
		azimuth, err1 := strconv.ParseFloat(fields[0], 64)
		slope, err2 := strconv.ParseFloat(fields[1], 64)
		if err1 != nil || err2 != nil {
			return nil, 0, fmt.Errorf("failed reading azimuth slope")
		}
		azimuth = mineflow.ToRadians(azimuth)
		slope = mineflow.ToRadians(slope)

		if slope <= 0 {
			return nil, 0, fmt.Errorf("invalid slope")
		}

		for azimuth >= mineflow.TAU {
			azimuth -= mineflow.TAU
		}
		for azimuth < 0 {
			azimuth += mineflow.TAU
		}
		pairs = append(pairs, mineflow.AzmSlopePair{Azimuth: azimuth, Slope: slope})
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed reading min search file: %w", err)
	}

	if len(pairs) == 0 {
		return nil, 0, fmt.Errorf("failed reading slope definition")
	}

	slopeDef := mineflow.NewSlopeDefinition(pairs)
	pattern := mineflow.PatternMinSearch(blockDef, slopeDef, int(numBenches))
	return mineflow.NewRegular3DBlockModelPatternPrecedence(blockDef, pattern), blockDef.NumBlocks(), nil
}

// initExplicit ports InitExplicit (mineflow.cpp:4482): explicit precedence
// constraints from a file.
func initExplicit(args *argReader) (mineflow.PrecedenceConstraints, int, error) {
	explicitFile, ok := args.readString()
	if !ok {
		return nil, 0, fmt.Errorf("failed reading explicit precedence file argument")
	}

	in, err := os.Open(explicitFile)
	if err != nil {
		return nil, 0, fmt.Errorf("failed opening explicit precedence file")
	}
	defer in.Close()

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	if !scanner.Scan() {
		return nil, 0, fmt.Errorf("failed reading num blocks line")
	}
	numBlocks64, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("failed reading num blocks")
	}
	numBlocks := int(numBlocks64)
	if numBlocks <= 0 {
		return nil, 0, fmt.Errorf("invalid num blocks")
	}

	pre := mineflow.NewExplicitPrecedence(numBlocks)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		from64, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("failed reading from index")
		}
		from := int(from64)
		if from < 0 || from >= numBlocks {
			return nil, 0, fmt.Errorf("invalid block index in precedence file: %d", from)
		}
		for _, f := range fields[1:] {
			to64, err := strconv.ParseInt(f, 10, 64)
			if err != nil {
				return nil, 0, fmt.Errorf("failed reading to index")
			}
			to := int(to64)
			if to < 0 || to >= numBlocks {
				return nil, 0, fmt.Errorf("invalid block index in precedence file: %d", to)
			}
			if err := pre.AddConstraint(from, to); err != nil {
				return nil, 0, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed reading explicit precedence file: %w", err)
	}

	return pre, numBlocks, nil
}

// initValues ports InitValues (mineflow.cpp:4544): whitespace-separated
// integer block values, one per block.
func initValues(valuesPath string, numBlocks int) (mineflow.BlockValues, error) {
	in, err := os.Open(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed opening values file")
	}
	defer in.Close()

	values := make([]int64, numBlocks)
	reader := bufio.NewReader(in)
	for i := 0; i < numBlocks; i++ {
		if _, err := fmt.Fscan(reader, &values[i]); err != nil {
			return nil, fmt.Errorf("failed reading values line: %d", i+1)
		}
	}
	return mineflow.SliceBlockValues(values), nil
}

func main() {
	args := &argReader{args: os.Args[1:]}
	if len(args.args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	programStart := time.Now()

	var pre mineflow.PrecedenceConstraints
	numBlocks := 0

	for {
		argument, ok := args.peek()
		if !ok || !strings.HasPrefix(argument, "--") {
			break
		}
		args.next()

		var err error
		switch argument {
		case "--regular":
			pre, numBlocks, err = initRegular(args)
		case "--minsearch":
			pre, numBlocks, err = initMinSearch(args)
		case "--explicit":
			pre, numBlocks, err = initExplicit(args)
		default:
			fmt.Fprintf(os.Stderr, "Unknown argument: %s\n", argument)
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if pre == nil || numBlocks <= 0 {
		fmt.Fprintln(os.Stderr, "No precedence specified, or no blocks in input")
		os.Exit(1)
	}

	dataPath, ok := args.readString()
	if !ok {
		fmt.Fprintln(os.Stderr, "No data file argument")
		os.Exit(1)
	}
	values, err := initValues(dataPath, numBlocks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failure initializing values %s: %v\n", dataPath, err)
		os.Exit(1)
	}

	outputPath, ok := args.readString()
	if !ok {
		fmt.Fprintln(os.Stderr, "No output file argument")
		os.Exit(1)
	}

	readInput := time.Now()
	fmt.Println("  MineFlow - Go Version 1.0")
	fmt.Println("--------------------------")
	fmt.Println("Num blocks  :", numBlocks)

	solver, err := mineflow.NewPseudoSolver(pre, values)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	initialized := time.Now()

	info, err := solver.Solve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	solved := time.Now()

	of, err := os.Create(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed creating output file: %v\n", err)
		os.Exit(1)
	}
	writer := bufio.NewWriter(of)
	for i := 0; i < info.NumNodes; i++ {
		if solver.InMinimumCut(i) {
			fmt.Fprintf(writer, "%d\n", i)
		}
	}
	writer.Flush()
	of.Close()
	output := time.Now()

	fmt.Println("Num mined   :", info.NumContainedNodes)
	fmt.Println("Value       :", info.ContainedValue)
	fmt.Println("--------------------------")
	fmt.Println("Read data   :", elapsed(readInput.Sub(programStart)))
	fmt.Println("Init solver :", elapsed(initialized.Sub(readInput)))
	fmt.Println("Solved      :", elapsed(solved.Sub(initialized)))
	fmt.Println("Saved       :", elapsed(output.Sub(solved)))
	fmt.Println("--------------------------")
	fmt.Println("Total       :", elapsed(output.Sub(programStart)))
}
