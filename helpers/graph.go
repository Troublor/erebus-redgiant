package helpers

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/goccy/go-graphviz"
	gonumGraph "gonum.org/v1/gonum/graph"
)

type IterableGraph interface {
	gonumGraph.Graph
	Edges() gonumGraph.Edges
}

func ToGraphvizDot(graph IterableGraph) ([]byte, error) {
	G := graphviz.New()
	g, err := G.Graph()
	if err != nil {
		return nil, err
	}
	defer func() {
		err = g.Close()
		if err != nil {
			panic(err)
		}
		err = G.Close()
		if err != nil {
			panic(err)
		}
	}()

	nodeToName := func(node gonumGraph.Node) string {
		var name string
		if stringer, ok := node.(fmt.Stringer); ok {
			name = fmt.Sprintf("%s@%d", stringer, node.ID())
		} else {
			name = strconv.FormatInt(node.ID(), 10)
		}
		return name
	}

	nodeIter := graph.Nodes()
	nodeIter.Reset()
	for nodeIter.Next() {
		node := nodeIter.Node()
		name := nodeToName(node)
		_, err = g.CreateNode(name)
	}

	edgeIter := graph.Edges()
	edgeIter.Reset()
	for edgeIter.Next() {
		edge := edgeIter.Edge()
		f, err := g.Node(nodeToName(edge.From()))
		if err != nil {
			panic(err)
		}
		t, err := g.Node(nodeToName(edge.To()))
		if err != nil {
			panic(err)
		}
		_, err = g.CreateEdge(fmt.Sprintf("%s->%s", f.Name(), t.Name()), f, t)
		if err != nil {
			panic(err)
		}
	}

	var buf bytes.Buffer
	if err := G.Render(g, "dot", &buf); err != nil {
		panic(err)
	}
	return buf.Bytes(), nil
}
