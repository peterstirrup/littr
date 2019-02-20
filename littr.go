// littr contains all type definitions and core functionality for littr.
package littr

import (
	"fmt"
	"io/ioutil"
	"littr/tree"
	"os/exec"
	"sort"
	"strings"
	"time"
)

var (
	timer = `
	t := Timer{}
	t.Start("##/#" + strings.Replace(GetCurrentName(), ".", "#", -1))
	defer t.End()`
)

type littr struct {
	fileName string
	filePath string
	flags    string
	// code is the littr code
	code     string
	// ogCode is the code before littring
	ogCode   string
	// imports is the list of imports to insert
	imports  []string
	// v is the verbosity of the logs
	// (i.e. how many logs littr makes and how descriptive they are)
	v        int
	outLines []string
	// output is a channel which receives the output from the program
	output chan string
	// t is the root node for the code tree
	t tree.CodeTree
	// timerDef is the Go code for the timer
	timerDef string
	// timeout is the amount of time afforded to littr before it times out
	timeout time.Duration
	errorCh chan error
}

// NewLittr returns a new littr object.
func NewLittr(fileName, filePath, flags string, timeout time.Duration) (*littr, error) {
	b, err := ioutil.ReadFile("../../data/timer.txt")
	if err != nil {
		return &littr{}, fmt.Errorf("file not found")
	}

	return &littr{
		fileName: fileName,
		filePath: filePath,
		flags:    flags,
		timerDef: string(b),
		errorCh:  make(chan error, 1),
		timeout:  timeout,
	}, nil
}

// Start runs littr.
func (l *littr) Start() error {
	l.Log(0, "starting littr")

	b, err := ioutil.ReadFile(l.filePath)
	if err != nil {
		return fmt.Errorf("file not found")
	}
	l.code = string(b)

	// Check there's funcs in code
	if !strings.Contains(l.code, "func") {
		return fmt.Errorf("no func in file")
	}

	// Set original code
	l.ogCode = l.code

	// Defer an old code rewrite in case things go belly up
	// We want to be left with the original code file, not some mess
	defer func() {
		// Now rewrite old code back
		err = l.WriteOriginalCode()
		if err != nil {
			l.Log(0, "unable to write original code back")
			return
		}
	}()

	go func() {
		var err error

		// Defer the error chan insert as we always perform this when we return
		defer func() {
			l.errorCh <- err
		}()

		l.Log(1, "starting InsertTimer")
		err = l.InsertTimer()
		if err != nil {
			return
		}

		// Write the new code to file and execute
		l.Log(1, "writing littred code")
		err = l.WriteLittredCode()
		if err != nil {
			return
		}

		l.Log(1, "executing "+l.fileName)
		err = l.Execute()
		if err != nil {
			return
		}

		// Read the tree off
		l.t.ReadTree()
	}()

	select {
	case err = <-l.errorCh:
		// Littr has completed without timing out
	case <-time.After(l.timeout):
		// Timeout occurred
		err = fmt.Errorf("timeout")
	}

	return err
}

// GetFuncName returns the first function name when given code in the format 'func FunctionName(...'.
func (l *littr) GetFuncName(i int) string {
	return l.code[i+5 : i+strings.Index(l.code[i:], "(")]
}

/////////////////////////////////////////////////////////////
///////////////////      WRITING CODE     ///////////////////
/////////////////////////////////////////////////////////////

// WriteOriginalCode writes the updated littred code to file.
func (l *littr) WriteLittredCode() error {
	err := ioutil.WriteFile(l.filePath, []byte(l.code), 0644)
	if err != nil {
		return fmt.Errorf("failed to write littred code to file with %s", err)
	}
	return nil
}

// WriteOriginalCode writes the code before it was littered to file.
func (l *littr) WriteOriginalCode() error {
	l.Log(1, "writing original code")
	err := ioutil.WriteFile(l.filePath, []byte(l.ogCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write old code to file with %s", err)
	}

	return nil
}

// Insert places s into l.code at position i.
func (l *littr) Insert(s string, i int) {
	l.code = l.code[:i] + s + l.code[i:]
}

// AddImports is a pretty janky function which assesses what imports exist in the program already
// and adds necessary imports if they're not already there.
func (l *littr) AddImports() error {
	start := strings.Index(l.code, "import (")
	if start == -1 {
		// if it's -1 it cannot be found
		return fmt.Errorf("cannot find import in code")
	}

	end := strings.Index(l.code, ")")
	if end == -1 {
		// if it's -1 it cannot be found
		return fmt.Errorf("invalid import in code")
	}

	importLs := strings.Split(strings.Replace(l.code[start+10:end-1], "\t", "", -1), "\n")

	// Range through imports and add any that should be there but aren't already
	for _, im := range l.imports {
		if !l.Contains(importLs, im) {
			// It's not there, so add it to the import list
			importLs = append(importLs, im)
		}
	}

	// Sort slice because Go will panic if imports are not alphabetically ordered
	sort.Strings(importLs)

	// Now add them back by inserting into code
	l.InsertImports(&importLs, start, end)

	return nil
}

// InsertImports generates an imports string by comparing what's already there in the imports list
// and what needs to be added.
func (l *littr) InsertImports(imports *[]string, start, end int) {
	// Take out imports
	l.code = l.code[:start] + l.code[end+1:]

	// Init import string and generate it by cycling through necessary imports
	importStr := "\n\nimport (\n"
	for _, im := range *imports {
		importStr += "\t" + im + "\n"
	}
	importStr += ")\n"

	// Insert the import at the first \n (which should be after package line)
	insertAt := strings.Index(l.code, "\n")

	l.code = l.code[:insertAt] + importStr + l.code[insertAt:]
}


// InsertTimer goes through the code and inserts timers at the start and end of functions.
func (l *littr) InsertTimer() error {
	var inFunc, numInserted int

	// Loop over the file until found '{', then insert timer code
	for i, char := range l.code {
		i += numInserted * len(timer)

		if i+4 > len(l.code) {
			// Reached EOF
			break
		}

		if string(char) == "}" {
			inFunc -= 1
		}

		if string(char) == "{" {
			inFunc += 1
		}

		// Debug
		l.Log(2, fmt.Sprintf("char: %v    inFunc: %v    numInserted: %v    last_four_chars: %v", string(char), inFunc, numInserted, l.code[i:i+4]))

		if l.code[i:i+4] != "func" || inFunc != 0 {
			// We're not at func, so try again
			continue
		}

		// We're at a func! Let's find index of first '{' (start of func)
		funcStart := strings.Index(l.code[i:], "{")
		// Equals -1 if not found
		if funcStart == -1 {
			return fmt.Errorf("bad syntax: cannot find func opening bracket")
		}

		// Insert timer here
		l.Insert(timer, funcStart+i+1)
		l.Log(2, "inserted timer start")

		numInserted += 1
	}

	// Insert import time
	// Import should be first bracket
	i := strings.Index(l.code, "(")
	if i == -1 {
		return fmt.Errorf("no import found")
	}

	// Add import list and add them to the code
	l.imports = []string{`"runtime"`, `"strings"`, `"time"`}
	l.AddImports()

	// Finally, insert the timer definition
	l.Insert(l.timerDef, len(l.code))
	l.Log(2, "inserted timer struct definition")
	return nil
}

/////////////////////////////////////////////////////////////

// Execute runs the go file specified by l.filePath along with any flags and builds the tree.
func (l *littr) Execute() error {
	b, err := exec.Command("/usrcd/local/go/bin/go", "run", l.filePath, l.flags).Output()
	if err != nil {
		return fmt.Errorf("l.Execute failed with %s\n", err.Error())
	}

	// Convert the output into separate lines
	l.outLines = strings.Split(string(b), "\n")
	l.t.BuildTree(l.outLines)

	// Print the program output
	l.PrintOutput()
	return nil
}

/////////////////////////////////////////////////////////////
///////////////      PRINTING AND LOGGING     ///////////////
/////////////////////////////////////////////////////////////

// Log logs the passed in message with a formatted date and time.
// v is the verbosity of the message.
func (l *littr) Log(v int, s interface{}) {
	// Check logging level
	if v > l.v {
		return
	}
	t := time.Now()
	fmt.Printf("%v%v\n", t.Format("15:04:05   "), s)
}

// PrintOutput prints the program output, ignoring any littr.
func (l *littr) PrintOutput() {
	fmt.Println("\n\033[38;2;0;0;0m\033[37;1;4m\033[38;2;0;0;0m\033[48;2;240;240;240m___PROGRAM_OUTPUT____________________________\033[0m\033[48;2;240;240;240m\033[38;2;0;0;0m")
	for _, line := range l.outLines[:len(l.outLines)-2] {
		if len(line) < 4 || line[:4] != "##/#" {
			fmt.Println(" " + line)
		}
	}
	fmt.Print("\u001b[0m")
}

// SetVerbosity sets the logging level.
func (l *littr) SetVerbosity(level int) {
	l.v = level
}

/////////////////////////////////////////////////////////////

// Contains checks whether an item is in a slice.
func (l *littr) Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}