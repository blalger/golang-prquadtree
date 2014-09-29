package prquadtree

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

func validateFind(t *testing.T, node *Node, point Point, expected ...int) {
	vals := node.Find(point)
	if len(vals) != len(expected) {
		t.Errorf("Expected length of %d, but actually %d", len(expected), len(vals))
	}

	for i, val := range expected {
		if vals[i] != val {
			t.Errorf("Expected %d at index %d, actually %d", val, i, vals[i])
		}
	}
}

func validateFindRange(t *testing.T, node *Node, nw, se Point, expected ...int) {
	var sorted []int
	elems := node.FindRange(nw, se, nil)

	if len(elems) != len(expected) {
		t.Errorf("Expected length of %d, but actually %d", len(expected), len(elems))
	}

	for _, curr := range elems {
		sorted = append(sorted, curr.(int))
	}
	sort.Ints(sorted)

	for i, val := range expected {
		if sorted[i] != val {
			t.Errorf("Expected %d at index %d, actually %d", val, i, sorted[i])
		}
	}
}

func validateInsert(t *testing.T, node *Node, point Point, vals ...int) {
	for _, val := range vals {
		if err := node.Insert(point, val); err != nil {
			t.Fatal(err)
		}
	}
}

func modifyPoint(point Point, xdelta int, ydelta int) Point {
	return Point{point.x + xdelta, point.y + ydelta}
}

func testRectangleCollision(
	t *testing.T, nw1, se1, nw2, se2 Point, expectResult bool, msg string) {
	if rectanglesCollide(nw1, se1, nw2, se2) != expectResult {
		t.Errorf("Unexpected result %v for '%s' checking collision %vx%v %vx%v",
			!expectResult, msg, nw1, se1, nw2, se2)
	}
}

func TestRectanglesCollide(t *testing.T) {
	nw := Point{-2, 2}
	se := Point{2, -2}

	testRectangleCollision(t, nw, se, nw, se, true, "Same rectangle")
	testRectangleCollision(t, nw, se, Point{-1, 1}, Point{1, -1}, true, "Inside contains")
	testRectangleCollision(t, nw, se, Point{-3, 3}, Point{3, -3}, true, "Outside contains")
	testRectangleCollision(t, nw, se, Point{0, 10}, Point{10, -10}, true, "Left edge")
	testRectangleCollision(t, nw, se, Point{-10, 10}, Point{0, -10}, true, "Right edge")
	testRectangleCollision(t, nw, se, Point{-10, 0}, Point{10, -10}, true, "Top edge")
	testRectangleCollision(t, nw, se, Point{-10, 10}, Point{10, 0}, true, "Bottom edge")

	testRectangleCollision(t, nw, se, Point{0, 0}, Point{10, -10}, true, "NW corner")
	testRectangleCollision(t, nw, se, Point{-10, 0}, Point{0, -10}, true, "NE corner")
	testRectangleCollision(t, nw, se, Point{-10, 10}, Point{0, 0}, true, "SE corner")
	testRectangleCollision(t, nw, se, Point{0, 10}, Point{10, 0}, true, "SW corner")

	testRectangleCollision(t, nw, se, Point{-10, 10}, nw, true, "NW point")
	testRectangleCollision(t, nw, se, Point{-10, 10}, Point{-3, 2}, false, "NW point -1 west")
	testRectangleCollision(t, nw, se, se, Point{10, -10}, true, "SE point")
	testRectangleCollision(t, nw, se, Point{3, -2}, Point{10, -10}, false, "SE point +1 east")

	testRectangleCollision(t, nw, se, Point{-10, 1}, Point{-3, -1}, false, "Outside west")
	testRectangleCollision(t, nw, se, Point{-10, 3}, Point{-3, -3}, false, "Outside west")
	testRectangleCollision(t, nw, se, Point{3, 1}, Point{10, -1}, false, "Outside east")
	testRectangleCollision(t, nw, se, Point{3, 3}, Point{10, -3}, false, "Outside east")
	testRectangleCollision(t, nw, se, Point{-1, 10}, Point{1, 3}, false, "Outside north")
	testRectangleCollision(t, nw, se, Point{-3, 10}, Point{3, 3}, false, "Outside north")
	testRectangleCollision(t, nw, se, Point{-1, -3}, Point{1, -10}, false, "Outside south")
	testRectangleCollision(t, nw, se, Point{-3, -3}, Point{3, -10}, false, "Outside south")
}

// Create the smallest allowed node, 2x2, fill it, and verify contents
func TestFullNode(t *testing.T) {
	points := [...]Point{
		Point{0, 1},
		Point{1, 1},
		Point{1, 0},
		Point{0, 0},
	}

	// There was a rounding bug when choosing quadrants for a 2x2 square when the
	// points were negative, thus we run two tests.  Once with positive bounds,
	// again with negative.
	for delta := 0; delta >= -1; delta-- {
		node := new(Node)
		node.nw = modifyPoint(Point{0, 1}, delta, delta)
		node.se = modifyPoint(Point{1, 0}, delta, delta)

		for i, point := range points {
			point := modifyPoint(point, delta, delta)
			node.Insert(point, i)
		}

		for i, point := range points {
			point := modifyPoint(point, delta, delta)
			leaf := node.nodes[i].(*Leaf)
			if leaf.point != point {
				t.Errorf("Expected point %s at index %d, but got %s", point, i, leaf.point)
			}
			if len(leaf.elems) != 1 || leaf.elems[0] != i {
				t.Errorf("Expected val %d, but got %s", i, leaf.elems)
			}
			if actual := node.Find(point); !reflect.DeepEqual(leaf.elems, actual) {
				t.Errorf("Expected %s, but got %s", leaf.elems, actual)
			}
		}
	}
}

// Test basic insert and find operations
func TestBasicOperation(t *testing.T) {
	tree := NewTree(-10, 10, -10, 10)
	point1 := Point{3, 1}
	point2 := Point{-2, 8}
	point3 := Point{4, 2}

	if tree.Find(point1) != nil {
		t.Error("Expected not to find point before inserting")
	}

	if err := tree.Insert(Point{11, 0}, 1); err == nil {
		t.Error("Expected error for inserting point out of bounds")
	}

	// Insert a single entry at a point and verify we can find it
	validateInsert(t, tree, point1, 3)
	validateFind(t, tree, point1, 3)

	// Insert a couple more values at the same point and verify we can find it
	validateInsert(t, tree, point1, 4, 5)
	validateFind(t, tree, point1, 3, 4, 5)

	// Insert a point in a new quadrant, validate it can be found
	validateInsert(t, tree, point2, 6)
	validateFind(t, tree, point2, 6)

	// Insert a point and cause split.  Validate it can be found
	validateInsert(t, tree, point3, 7)
	validateFind(t, tree, point3, 7)
	validateFind(t, tree, point1, 3, 4, 5)
}

// Create a tree 21 x 21 and fill it in random order, verify all values are present
func TestFullTree(t *testing.T) {
	tree := NewTree(-10, 10, -10, 10)

	// insert points in random order
	rand.Seed(time.Now().Unix())
	order := rand.Perm(21 * 21)
	for i := range order {
		x := i%21 - 10
		y := i/21 - 10
		validateInsert(t, tree, Point{x, y}, i)
	}

	for y := -10; y <= 10; y++ {
		for x := -10; x <= 10; x++ {
			expected := (y+10)*21 + (x + 10)
			validateFind(t, tree, Point{x, y}, expected)
		}
	}

	validateFindRange(t, tree, Point{-10, -10}, Point{-10, -10}, 0)
	validateFindRange(t, tree, Point{-7, -7}, Point{-6, -7}, 66, 67)
	validateFindRange(t, tree, Point{10, -7}, Point{-10, -7})
	validateFindRange(t, tree, Point{-1, 1}, Point{1, -1},
		198, 199, 200, 219, 220, 221, 240, 241, 242)
}

// Delete is probably broken, well I definitely am not cleaning up the tree
// properly and I'm not checking any corner cases
func TestBasicDelete(t *testing.T) {
	tree := NewTree(-10, 10, -10, 10)

	point := Point{4, 5}

	validateInsert(t, tree, point, 4)
	validateInsert(t, tree, point, 5)
	validateFind(t, tree, point, 4, 5)

	if tree.Delete(Point{0, 20}, 4) {
		t.Error("Delete succeeded on an out of bounds point")
	}

	if tree.Delete(point, 1) {
		t.Error("Delete succeeded on an invalid value")
	}

	if !tree.Delete(point, 4) {
		t.Error("Delete failed for a valid point/value")
	}

	validateFind(t, tree, point, 5)
	if !tree.Delete(point, 5) {
		t.Error("Delete failed for a valid point/value")
	}
}