package util

type IdGenerator struct {
	curId    uint64
	maxId    uint64
	reuseIds []uint64
}

func NewIdGenerator(min uint64, max uint64) *IdGenerator {
	return &IdGenerator{
		curId:    min,
		maxId:    max,
		reuseIds: make([]uint64, 10),
	}
}

func (g *IdGenerator) GetId() uint64 {
	var id uint64 = 0
	l := len(g.reuseIds)
	if l > 0 {
		id = g.reuseIds[l-1]
		g.reuseIds = g.reuseIds[:l-1]
	} else if g.curId > g.maxId {
		id = 0
	} else {
		id = g.curId
		g.curId++
	}

	return id
}

func (g *IdGenerator) ReuseId(id uint64) {
	g.reuseIds = append(g.reuseIds, id)
}
