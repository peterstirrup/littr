// tree contains the functionality of the code tree, which inspects and contains the structure of a program with nested functions.
package tree

import (
	"fmt"
	"strings"
)

// CodeTree simply holds the root node.
type CodeTree struct {
	Root Node
}

// Node is the basic node structure for the tree, and holds a map of function names --> children.
// Usually we would just use a LeftTree and RightTree, but it's not binary so can have > 2 children.
type Node struct {
	Name     string
	Children map[string]*Node
	Time     string
}

// ReadTree starts at the root node and descends through reading the tree.
func (t *CodeTree) ReadTree() {
	fmt.Println("\n\033[38;2;0;0;0m\033[37;1;4m\033[38;2;255;255;255m\033[48;2;0;0;0m   TREE                      \033[0m\033[48;2;0;0;0m\033[38;2;255;255;255m")
	t.DescendTree( t.Root.Children["goexit"].Children["main"].Children["main"], 0)
	fmt.Println("\u001b[0m")
}

// DescendTree reads the tree and prints it in a way to display
// how the call stack is timed and works.
func (t *CodeTree) DescendTree(node *Node, level int) {
	var space string

	for i := 0; i < level; i++ {
		space += "    "
	}

	fmt.Println(space + " ↪ " + node.Name + ": " + node.Time)

	// If no children, return as it's a leaf
	if len(node.Children) == 0 {
		return
	}

	for _, child := range node.Children {
		t.DescendTree(child, level+1)
	}
}

// AddToTree adds the function and time to the code tree.
func (t *CodeTree) AddToTree(funcLs []string, node *Node, i int, time string) {
	// We're below - let's resurface
	if i < 0 {
		return
	}

	if child, ok := node.Children[funcLs[i]]; ok {
		// It's in the map, so let's keep going
		if i == 0 {
			node.Children[funcLs[i]].Time = time
		}
		t.AddToTree(funcLs, child, i-1, time)
		return
	} else {
		// Not in the children
		// If it's the last, add the time
		if i == 0 {
			// Leaf - add time
			node.Children[funcLs[i]] = &Node{
				Name:     funcLs[i],
				Children: map[string]*Node{},
				Time:     time,
			}
		} else {
			// Not leaf - keep going
			node.Children[funcLs[i]] = &Node{
				Name:     funcLs[i],
				Children: map[string]*Node{},
			}

			t.AddToTree(funcLs, node.Children[funcLs[i]], i-1, time)
		}
	}
}

func (t *CodeTree) BuildTree(outLines []string) {
	// Last print is main - or the root node
	t.Root = Node{
		Children: map[string]*Node{},
	}


	for _, line := range outLines {
		if len(line) < 4 || line[:4] != "##/#" {
			// Just a fmt.Print from the prog - ignore
			continue
		}
		lineLs := strings.Split(line[4:], "#")
		t.AddToTree(lineLs, &t.Root, len(lineLs)-2, lineLs[len(lineLs)-1])
	}
}
