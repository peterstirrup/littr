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
	timerDef = `
type Timer struct {
	text string
	start time.Time
}

func (t *Timer) Start(title string) {
	t.text = title
	t.start = time.Now()
}

func (t *Timer) End() {
	fmt.Println(fmt.Sprintf("%s%v", t.text, time.Since(t.start)))
}

func GetCurrentName() string {
	return getFrame()
}

func getFrame() string {
	var s string
	for i := 0; i < 10; i++ {
		// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
		targetFrameIndex := i + 3

		// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
		programCounters := make([]uintptr, targetFrameIndex+2)
		n := runtime.Callers(0, programCounters)

		frame := runtime.Frame{Function: "unknown"}
		if n > 0 {
			frames := runtime.CallersFrames(programCounters[:n])
				for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
				var frameCandidate runtime.Frame
				frameCandidate, more = frames.Next()
				if frameIndex == targetFrameIndex {
					frame = frameCandidate
				}
			}
		}
		switch frame.Function {
		case "unknown", "goexit":
			return s
		}
		s += strings.Split(frame.Function, ".")[1] + "#"
	}

	return s
}
`

	timer = `
	t := Timer{}
	t.Start("##/#" + strings.Replace(GetCurrentName(), ".", "#", -1))
	defer t.End()`
)

type littr struct {
	fileName string
	filePath string
	flags    string
	code     string
	ogCode   string
	imports  []string
	// v is the verbosity of the logs
	// (i.e. how many logs littr makes and how descriptive they are)
	v        int
	out      string
	outLines []string
	t        tree.Node
}

// NewLittr returns a new littr object with file path field set.
func NewLittr(fileName, filePath, flags string) *littr {
	return &littr{
		fileName: fileName,
		filePath: filePath,
		flags:    flags,
	}
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
		l.Log(1, "writing original code back")
		err = l.WriteOriginalCode()
		if err != nil {
			l.Log(0, "unable to write original code back")
			return
		}
	}()

	l.Log(1, "starting InsertTimer")
	err = l.InsertTimer()
	if err != nil {
		return err
	}

	// Write the new code to file and execute
	l.Log(1, "writing littred code")
	err = l.WriteLittredCode()
	if err != nil {
		return err
	}

	l.Log(1, "executing "+l.fileName)
	err = l.Execute()
	if err != nil {
		return err
	}

	// Read the tree off
	fmt.Println("\n\033[38;2;0;0;0m\033[37;1;4m\033[38;2;255;255;255m\033[48;2;0;0;0m___TREE_______________________________________\033[0m\033[48;2;0;0;0m\033[38;2;255;255;255m")
	l.ReadTree(l.t.Children["goexit"].Children["main"].Children["main"], 0)
	fmt.Println("\u001b[0m")
	return nil
}

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
	l.Insert(timerDef, len(l.code))
	l.Log(2, "inserted timer struct definition")
	return nil
}

// Insert places s into l.code at position i.
func (l *littr) Insert(s string, i int) {
	l.code = l.code[:i] + s + l.code[i:]
}

// GetFuncName returns the first function name when given code in the format 'func FunctionName(...'.
func (l *littr) GetFuncName(i int) string {
	return l.code[i+5 : i+strings.Index(l.code[i:], "(")]
}

// Log logs the passed in message with a formatted date and time.
// v is the verbosity of the message.
func (l *littr) Log(v int, s interface{}) {
	// Check logging level
	if v > l.v {
		return
	}
	t := time.Now()
	fmt.Printf("%v%v\n", t.Format("15:04:05 > "), s)
}

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
	err := ioutil.WriteFile(l.filePath, []byte(l.ogCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write old code to file with %s", err)
	}

	return nil
}

// Execute runs the go file specified by l.filePath along with any flags and builds the tree.
func (l *littr) Execute() error {
	b, err := exec.Command("/usr/local/go/bin/go", "run", l.filePath, l.flags).Output()
	if err != nil {
		return fmt.Errorf("l.Execute failed with %s\n", err.Error())
	}
	l.out = string(b)
	l.outLines = strings.Split(l.out, "\n")
	l.BuildTree()

	fmt.Println("\n\033[38;2;0;0;0m\033[37;1;4m\033[38;2;0;0;0m\033[48;2;240;240;240m___PROGRAM_OUTPUT____________________________\033[0m\033[48;2;240;240;240m\033[38;2;0;0;0m")
	// Actually print the program output
	l.PrintOutput()
	fmt.Print("\u001b[0m")
	return nil
}

// SetVerbosity sets the logging level.
func (l *littr) SetVerbosity(level int) {
	l.v = level
}

func (l *littr) BuildTree() {
	// Last print is main - or the root node
	l.t = tree.Node{
		Children: map[string]*tree.Node{},
	}


	for _, line := range l.outLines {
		if len(line) < 4 || line[:4] != "##/#" {
			// Just a fmt.Print from the prog - ignore
			continue
		}
		lineLs := strings.Split(line[4:], "#")
		l.AddToTree(lineLs, &l.t, len(lineLs)-2, lineLs[len(lineLs)-1])
	}
}

// ParseTime takes the line of function and time and returns [caller, current func name, time].
func (l *littr) ParseTime(s string) (string, string, string) {
	// Remove starting #.#/#
	// Split so we get [caller func, current func, time]
	f := strings.Split(s[4:], ".")
	return strings.Split(f[0], ".")[1], strings.Split(f[1], ".")[1], f[2]
}

// AddToTree adds the function and time to the code tree.
func (l *littr) AddToTree(funcLs []string, node *tree.Node, i int, time string) {
	// We're below - let's resurface
	if i < 0 {
		return
	}

	if child, ok := node.Children[funcLs[i]]; ok {
		// It's in the map, so let's keep going
		if i == 0 {
			node.Children[funcLs[i]].Time = time
		}
		l.AddToTree(funcLs, child, i-1, time)
		return
	} else {
		// Not in the children
		// If it's the last, add the time
		if i == 0 {
			// Leaf - add time
			node.Children[funcLs[i]] = &tree.Node{
				Name:     funcLs[i],
				Children: map[string]*tree.Node{},
				Time:     time,
			}
		} else {
			// Not leaf - keep going
			node.Children[funcLs[i]] = &tree.Node{
				Name:     funcLs[i],
				Children: map[string]*tree.Node{},
			}

			l.AddToTree(funcLs, node.Children[funcLs[i]], i-1, time)
		}
	}
}

// ReadTree reads the tree and prints it in a way to display
// how the call stack is timed and works.
func (l *littr) ReadTree(node *tree.Node, level int) {
	var space string

	for i := 0; i < level; i++ {
		space += "    "
	}

	fmt.Println(space + "↪ " + node.Name + ": " + node.Time)

	// If no children, return as it's a leaf
	if len(node.Children) == 0 {
		return
	}

	for _, child := range node.Children {
		l.ReadTree(child, level+1)
	}
}

// PrintOutput prints the program output, ignoring any littr.
func (l *littr) PrintOutput() {
	for _, line := range l.outLines[:len(l.outLines)-2] {
		if len(line) < 4 || line[:4] != "##/#" {
			fmt.Println(line)
		}
	}
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

// Contains checks whether an item is in a slice.
func (l *littr) Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
