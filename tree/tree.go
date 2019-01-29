package tree

type Node struct {
	Name     string
	Children map[string]*Node
	Time     string
}
