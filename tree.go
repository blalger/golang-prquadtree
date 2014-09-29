package prquadtree

import (
	"errors"
	"fmt"
)

// A node has 4 quadrants in clockwise order:
//  (-1, 1) ==================== (1, 1)
//          |   0    |    1    |
//          |        |         |
//          =====  (0, 0)  =====
//          |        |         |
//          |   3    |    2    |
// (-1, -1) ==================== (1, -1)
type Node struct {
	// For convenience
	nw Point
	se Point

	// Can contain nil, Node* or Leaf*
	nodes [4]interface{}
}

// Leafs can store multiple values
type Leaf struct {
	point Point
	elems []interface{}
}

type Point struct {
	x int
	y int
}

// Given the nw and se boundaries of a rectangle and a point, determine which
// quadrant that point resides in if the rectangle were to be split in half vertically
// and horizontally.  Returns the quadrant as well as the updated boundaries
// for the chosen quad.
func chooseQuadrant(nw Point, se Point, point Point) (Point, Point, int) {
	center := Point{(nw.x + se.x) / 2, (nw.y + se.y) / 2}

	westernHemisphere := true
	southernHemisphere := true

	// rounding truncation means we favor western for positive coords
	// and east for negative coords so handle the case where width or
	// height is only 2 units specially
	if se.x-nw.x == 1 {
		westernHemisphere = point.x == nw.x
	} else {
		westernHemisphere = point.x <= center.x
		if westernHemisphere {
			se.x = center.x
		} else {
			nw.x = center.x
		}
	}

	if nw.y-se.y == 1 {
		southernHemisphere = point.y == se.y
	} else {
		southernHemisphere = point.y <= center.y
		if southernHemisphere {
			nw.y = center.y
		} else {
			se.y = center.y
		}
	}

	quadrant := 0
	if westernHemisphere {
		// quadrant 0 or 3
		if southernHemisphere {
			quadrant = 3
		} else {
			quadrant = 0
		}
	} else {
		// quadrant 1 or 2
		if southernHemisphere {
			quadrant = 2
		} else {
			quadrant = 1
		}
	}

	return nw, se, quadrant
}

func (leaf *Leaf) insert(point Point, val interface{}) {
	if point != leaf.point {
		panic(fmt.Sprintf(
			"Tried to insert at leaf %s for val destined for %s", leaf.point, point))
	}

	leaf.elems = append(leaf.elems, val)
}

func (node *Node) Insert(point Point, val interface{}) error {
	if !node.inBounds(point) {
		return errors.New("Attempt to insert point out of bounds")
	}

	nw, se, quadrant := chooseQuadrant(node.nw, node.se, point)
	if node.nodes[quadrant] == nil {
		var leaf *Leaf = &Leaf{point, nil}
		leaf.insert(point, val)
		node.nodes[quadrant] = leaf
	} else {
		switch next := node.nodes[quadrant].(type) {
		case *Node:
			return next.Insert(point, val)
		case *Leaf:
			if next.point == point {
				next.insert(point, val)
			} else {
				// Replace leaf with node and call recursively
				var newNode = new(Node)
				newNode.nw = nw
				newNode.se = se
				for _, oldVal := range next.elems {
					if err := newNode.Insert(next.point, oldVal); err != nil {
						panic(err)
					}
				}
				if err := newNode.Insert(point, val); err != nil {
					panic(err)
				}
				node.nodes[quadrant] = newNode
			}
		default:
			panic("Unexpected node type")
		}
	}

	return nil
}

func inBounds(nw, se, point Point) bool {
	return point.x >= nw.x &&
		point.x <= se.x &&
		point.y >= se.y &&
		point.y <= nw.y
}

func (node *Node) inBounds(point Point) bool {
	return inBounds(node.nw, node.se, point)
}

func rectanglesCollide(nw1, se1, nw2, se2 Point) bool {
	// Check rect2 contains rect1
	xContains := nw2.x <= nw1.x && se2.x >= se1.x
	yContains := nw2.y >= nw1.y && se2.y <= se1.y
	xOverlap := (nw2.x >= nw1.x && nw2.x <= se1.x) ||
		(se2.x >= nw1.x && se2.x <= se1.x)
	yOverlap := (nw2.y >= se1.y && nw2.y <= nw1.y) ||
		(se2.y >= se1.y && se2.y <= nw1.y)

	return (xOverlap || xContains) && (yOverlap || yContains)
}

func (node *Node) FindRange(nw, se Point, elems []interface{}) []interface{} {
	for _, curr := range node.nodes {
		if curr != nil {
			switch next := curr.(type) {
			case *Leaf:
				if inBounds(nw, se, next.point) {
					elems = append(elems, next.elems...)
				}
			case *Node:
				if rectanglesCollide(nw, se, next.nw, next.se) {
					elems = next.FindRange(nw, se, elems)
				}
			default:
				panic("Unexpected node type")
			}
		}
	}
	return elems
}

func (node *Node) Find(point Point) []interface{} {
	if !node.inBounds(point) {
		return nil
	}

	_, _, quadrant := chooseQuadrant(node.nw, node.se, point)
	if node.nodes[quadrant] == nil {
		return nil
	}

	switch next := node.nodes[quadrant].(type) {
	case *Node:
		return next.Find(point)
	case *Leaf:
		if next.point.x == point.x && next.point.y == point.y {
			return next.elems
		}
	default:
		panic("Unexpected node type")
	}
	return nil
}

func (leaf *Leaf) Delete(point Point, val interface{}) bool {
	if point.x != leaf.point.x || point.y != leaf.point.y {
		return false
	}

	for i, curr := range leaf.elems {
		if curr == val {
			length := len(leaf.elems)
			leaf.elems[i] = leaf.elems[length-1]
			leaf.elems = leaf.elems[:length-1]
			return true
		}
	}
	return false
}

// This is a lame start to proper deletion.  This will not cleanup empty
// leaf nodes in the tree.
func (node *Node) Delete(point Point, val interface{}) bool {
	if !node.inBounds(point) {
		return false
	}

	_, _, quadrant := chooseQuadrant(node.nw, node.se, point)
	if node.nodes[quadrant] == nil {
		return false
	}
	switch next := node.nodes[quadrant].(type) {
	case *Node:
		return next.Delete(point, val)
	case *Leaf:
		return next.Delete(point, val)
	default:
		panic("Unexpected node type")
	}
	return false
}

// A Tree is just a Node, this is a helper to initialize a valid tree
func NewTree(xmin, xmax, ymin, ymax int) *Node {
	if xmax <= xmin || ymax <= ymin {
		panic(
			fmt.Sprintf("Cannot create tree with boundaries: (x) %d-%d; (y)%d-%d",
				xmin, xmax, ymin, ymax))
	}

	newNode := new(Node)
	newNode.nw = Point{xmin, ymax}
	newNode.se = Point{xmax, ymin}
	return newNode
}
