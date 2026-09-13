package lp

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// findGLPSOL looks for glpsol through GLPK_PATH, which may name the executable
// or the directory holding it, and then through PATH.
func findGLPSOL() (string, error) {
	if dir := os.Getenv("GLPK_PATH"); dir != "" {
		if info, err := os.Stat(dir); err == nil {
			if !info.IsDir() {
				return dir, nil
			}
			name := "glpsol"
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			if candidate := filepath.Join(dir, name); fileExists(candidate) {
				return candidate, nil
			}
		}
	}
	if path, err := exec.LookPath("glpsol"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("%w: glpsol was found neither through GLPK_PATH nor on PATH; "+
		"install GLPK (see the lp package documentation) or use the default engine", ErrEngineUnavailable)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// solveGLPK runs glpsol once on the LP text, asking for both its printable
// report and its solution file.
func solveGLPK(name string, text []byte, timeLimit time.Duration) (*Solution, error) {
	glpsol, err := findGLPSOL()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "insyra-lp-*")
	if err != nil {
		return nil, fmt.Errorf("%w: creating a working directory: %w", ErrSolverFailed, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	modelFile := filepath.Join(dir, "model.lp")
	reportFile := filepath.Join(dir, "report.txt")
	solutionFile := filepath.Join(dir, "solution.txt")
	if err := os.WriteFile(modelFile, text, 0o600); err != nil {
		return nil, fmt.Errorf("%w: writing the model for glpsol: %w", ErrSolverFailed, err)
	}

	args := []string{"--lp", modelFile, "--output", reportFile, "--write", solutionFile}
	if timeLimit > 0 {
		args = append(args, "--tmlim", strconv.Itoa(int(math.Ceil(timeLimit.Seconds()))))
	}
	cmd := exec.Command(glpsol, args...)
	utils.ApplyHideWindow(cmd)
	out, runErr := cmd.CombinedOutput()
	// Messages name the temporary copy; the caller knows the text by name.
	output := strings.ReplaceAll(string(out), modelFile, name)

	if runErr != nil {
		if strings.Contains(output, "CPLEX LP file processing error") {
			return nil, fmt.Errorf("%w: %s", ErrInvalidModel, glpkReadError(output))
		}
		return nil, fmt.Errorf("%w: glpsol: %w\n%s", ErrSolverFailed, runErr, output)
	}

	report, err := os.ReadFile(reportFile)
	if err != nil {
		return nil, fmt.Errorf("%w: glpsol wrote no report: %w", ErrSolverFailed, err)
	}
	solution, err := os.ReadFile(solutionFile)
	if err != nil {
		return nil, fmt.Errorf("%w: glpsol wrote no solution file: %w", ErrSolverFailed, err)
	}
	return readGLPKResult(string(report), string(solution), output)
}

// glpkLineErrorRe matches GLPK's "file:line: message" reading messages.
var glpkLineErrorRe = regexp.MustCompile(`(?m)^(.+):(\d+): (.+)$`)

// glpkReadError picks the reading error out of glpsol's output, skipping the
// warnings printed before it.
func glpkReadError(output string) string {
	for _, m := range glpkLineErrorRe.FindAllStringSubmatch(output, -1) {
		if !strings.HasPrefix(m[3], "warning: ") {
			return m[0]
		}
	}
	return strings.TrimSpace(output)
}

// readGLPKResult builds a Solution from glpsol's report (--output), its
// solution file (--write) and what it printed.
//
// Values and the objective come from the solution file, which GLPK writes with
// DBL_DIG significant digits; the report rounds activities to six (%13.6g).
// Only the column names come from the report. The status comes from the
// solution file's status letters; where GLPK leaves it undefined ("u"), which
// with presolve on is how it writes an infeasible or unbounded LP, its fixed
// messages decide.
func readGLPKResult(report, solution, output string) (*Solution, error) {
	var sline []string
	for _, line := range strings.Split(solution, "\n") {
		if f := strings.Fields(line); len(f) > 0 && f[0] == "s" {
			sline = f
			break
		}
	}

	var (
		status     Status
		err        error
		objective  string
		valueField int // position of the value on a "j" line
	)
	switch {
	case len(sline) == 7 && sline[1] == "bas": // s bas rows cols primal dual objective
		status, err = glpkBasicStatus(sline[4], sline[5], output)
		objective, valueField = sline[6], 3 // j index status primal dual
	case len(sline) == 6 && sline[1] == "mip": // s mip rows cols status objective
		status, err = glpkMIPStatus(sline[4], output)
		objective, valueField = sline[5], 2 // j index value
	default:
		return nil, fmt.Errorf("%w: cannot read glpsol's solution line %q", ErrSolverFailed, strings.Join(sline, " "))
	}
	if err != nil {
		return nil, err
	}

	sol := &Solution{Status: status, Log: strings.TrimRight(output, "\n") + "\n\n" + report}
	if status != StatusOptimal && status != StatusFeasible {
		return sol, nil
	}

	obj, err := strconv.ParseFloat(objective, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read the objective %q", ErrSolverFailed, objective)
	}
	cols, err := strconv.Atoi(sline[3])
	if err != nil || cols < 0 {
		return nil, fmt.Errorf("%w: cannot read the column count %q", ErrSolverFailed, sline[3])
	}
	names, err := glpkColumnNames(report)
	if err != nil {
		return nil, err
	}

	values := make(map[string]float64, cols)
	order := make([]string, cols)
	for _, line := range strings.Split(solution, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || f[0] != "j" {
			continue
		}
		if len(f) <= valueField {
			return nil, fmt.Errorf("%w: cannot read the column line %q", ErrSolverFailed, line)
		}
		index, ierr := strconv.Atoi(f[1])
		value, verr := strconv.ParseFloat(f[valueField], 64)
		name, named := names[index]
		if ierr != nil || verr != nil || index < 1 || index > cols || !named {
			return nil, fmt.Errorf("%w: cannot read the column line %q", ErrSolverFailed, line)
		}
		values[name] = value
		order[index-1] = name
	}
	if len(values) != cols {
		return nil, fmt.Errorf("%w: the solution file has %d column values, want %d", ErrSolverFailed, len(values), cols)
	}

	sol.Objective, sol.Values, sol.order = obj, values, order
	return sol, nil
}

// glpkBasicStatus reads the primal and dual status letters of a simplex
// solution.
func glpkBasicStatus(primal, dual, output string) (Status, error) {
	switch {
	case primal == "f" && dual == "f":
		return StatusOptimal, nil
	case primal == "n":
		return StatusInfeasible, nil
	case primal == "f" && dual == "n":
		return StatusUnbounded, nil
	case primal == "f":
		// Primal feasible with the dual not yet feasible: the time limit
		// stopped the simplex on a feasible, unproven point.
		return StatusFeasible, nil
	}
	return glpkStatusFromMessages(output)
}

// glpkMIPStatus reads the status letter of an integer solution.
func glpkMIPStatus(letter, output string) (Status, error) {
	switch letter {
	case "o":
		return StatusOptimal, nil
	case "f":
		return StatusFeasible, nil
	case "n":
		return StatusInfeasible, nil
	case "u":
		return glpkStatusFromMessages(output)
	}
	return 0, fmt.Errorf("%w: unknown integer solution status %q", ErrSolverFailed, letter)
}

// glpkStatusFromMessages decides an undefined status from glpsol's fixed
// messages, measured on GLPK 5.0. Bounds GLPK refuses are reported only this
// way: glpsol still exits 0 and writes an undefined status.
func glpkStatusFromMessages(output string) (Status, error) {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "incorrect bounds") || strings.Contains(line, "has non-integer") {
			return 0, fmt.Errorf("%w: %s", ErrInvalidModel, strings.TrimSpace(line))
		}
	}
	switch {
	case strings.Contains(output, "HAS NO PRIMAL FEASIBLE SOLUTION"):
		return StatusInfeasible, nil
	case strings.Contains(output, "HAS UNBOUNDED PRIMAL SOLUTION"):
		return StatusUnbounded, nil
	case strings.Contains(output, "TIME LIMIT EXCEEDED"):
		return StatusStopped, nil
	}
	return 0, fmt.Errorf("%w: glpsol left the status undefined without saying why:\n%s", ErrSolverFailed, output)
}

// glpkColumnNames reads column numbers and names from the report's column
// table. GLPK prints each row as "%6d %-12s " and, for a name longer than 12
// characters, puts the rest of the row on the next line, indented; a blank
// line ends the table.
func glpkColumnNames(report string) (map[int]string, error) {
	lines := strings.Split(report, "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, "Column name") {
			start = i + 2 // skip the header and its dashes
			break
		}
	}
	if start < 0 || start > len(lines) {
		return nil, fmt.Errorf("%w: the glpsol report has no column table", ErrSolverFailed)
	}
	names := make(map[int]string)
	for _, line := range lines[start:] {
		if strings.TrimSpace(line) == "" {
			break
		}
		if len(line) < 8 || line[6] != ' ' {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSpace(line[:6]))
		if err != nil {
			continue // the indented rest of a row with a long name
		}
		fields := strings.Fields(line[7:])
		if len(fields) == 0 {
			return nil, fmt.Errorf("%w: the glpsol report has no name for column %d", ErrSolverFailed, index)
		}
		names[index] = fields[0]
	}
	return names, nil
}
