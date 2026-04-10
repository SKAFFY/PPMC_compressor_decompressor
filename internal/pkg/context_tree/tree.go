package context_tree

type ContextTree struct {
	root     *Node
	maxOrder int
}

func NewContextTree(maxOrder int) *ContextTree {
	return &ContextTree{
		root: &Node{
			Freq:     make(map[byte]int),
			Children: make(map[byte]*Node),
			Total:    0,
		},
		maxOrder: maxOrder,
	}
}

// GetNode возвращает узел для заданного контекста (последовательности байт).
// Если контекст не существует, возвращает nil.
func (t *ContextTree) GetNode(context []byte) *Node {
	node := t.root
	for _, c := range context {
		child := node.Children[c]
		if child == nil {
			return nil
		}
		node = child
	}
	return node
}

// Update обновляет статистику для одного конкретного контекста.
// Создаёт недостающие узлы по пути, но обновляет частоты только в конечном узле.
func (t *ContextTree) Update(sym byte, context []byte) {
	node := t.root
	// Спускаемся по контексту, создавая при необходимости узлы
	for _, c := range context {
		if node.Children[c] == nil {
			node.Children[c] = &Node{
				Freq:     make(map[byte]int),
				Children: make(map[byte]*Node),
				Total:    0,
			}
		}
		node = node.Children[c]
	}
	// Обновляем частоту в найденном (или созданном) узле
	node.Freq[sym]++
	node.Total++
}
