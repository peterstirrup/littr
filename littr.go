// littr contains all type definitions and core functionality for littr.
package littr

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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
	fmt.Println(fmt.Sprintf("%s: %v", t.text, time.Since(t.start)))
}`

	timerStart = `
	t := Timer{}
	t.Start("%s")`

	timerEnd = `
	t.End()`
)

type littr struct {
	fileName string
	filePath string
	flags    string
	code     string
	ogCode   string
	// v is the verbosity of the logs
	// (i.e. how many logs littr makes and how descriptive they are)
	v int
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

	return nil
}

func (l *littr) InsertTimer() error {
	var inBrackets int
	var inFunc int

	// Loop over the file until found '{', then insert timer code
	for i, char := range l.code {
		if i+4 > len(l.code) {
			// Reached EOF
			break
		}

		if string(char) == "}" {
			inFunc -= 1
		}

		if l.code[i:i+4] == "func" && inFunc == 0 {
			// We're in a func, so add to inFunc
			inFunc += 1
		} else {
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
		l.Insert(fmt.Sprintf(timerStart, l.GetFuncName(i)), funcStart+i+1)
		l.Log(2, "inserted timer start")

		// Now let's find ending closing bracket
		inBrackets = 0
		for j, char := range l.code[i+funcStart+1:] {
			if string(char) == "{" {
				inBrackets += 1
				continue
			}

			if string(char) == "}" {
				if inBrackets != 0 {
					inBrackets -= 1
					continue
				}

				// Found closing }, now insert last code
				l.Insert(timerEnd, funcStart+j+i)
				l.Log(2, "inserted timer end")
				break
			}
		}
	}

	// Insert import time
	// Import should be first bracket
	i := strings.Index(l.code, "(")
	if i == -1 {
		return fmt.Errorf("no import found")
	}

	l.Insert(`
	"time"
`, i+1)

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

func (l *littr) WriteLittredCode() error {
	err := ioutil.WriteFile(l.filePath, []byte(l.code), 0644)
	if err != nil {
		return fmt.Errorf("failed to write littred code to file with %s", err)
	}
	return nil
}

func (l *littr) WriteOriginalCode() error {
	err := ioutil.WriteFile(l.filePath, []byte(l.ogCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write old code to file with %s", err)
	}

	return nil
}

// Execute runs the go file specified by l.filePath along with any flags.
func (l *littr) Execute() error {
	cmd := exec.Command("/usr/local/go/bin/go", "run", l.filePath, l.flags)
	fmt.Println("-------------- " + l.fileName + " OUTPUT --------------")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("l.Execute failed with %s\n", err)
	}
	fmt.Println("------------------------------------------------------")
	return nil
}

// SetVerbosity sets the logging level.
func (l *littr) SetVerbosity(level int) {
	l.v = level
}
