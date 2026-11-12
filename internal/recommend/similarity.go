package recommend

import (
	"bytes"
	"cmp"
	"math"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	// Shrinkage lowers the similarity of item pairs with few common users.
	Shrinkage = 10.0
	// TopNeighbors is how many similar items are kept per item.
	TopNeighbors = 20
	// HistorySize is how many of the user's last liked items are used.
	HistorySize = 50
)

// UserItemScore is s(u,i) plus the time of the user's last interaction with the item.
type UserItemScore struct {
	UserID bson.ObjectID `bson:"userId"`
	ItemID bson.ObjectID `bson:"itemId"`
	Score  float64       `bson:"score"`
	LastAt time.Time     `bson:"lastAt"`
}

type Neighbor struct {
	ItemID bson.ObjectID `bson:"itemId" json:"itemId"`
	Score  float64       `bson:"score" json:"score"`
}

type entry struct {
	index int
	score float64
}

// Similarities computes, for every item, its TopNeighbors most similar items:
//
//	sim(i,j) = sum_u s(u,i)*s(u,j) / (||i|| * ||j||) * n_ij / (n_ij + Shrinkage)
//
// Only liked pairs (score > 0) are used; n_ij is the number of common users.
func Similarities(scores []UserItemScore) map[bson.ObjectID][]Neighbor {
	itemIndex := map[bson.ObjectID]int{}
	userIndex := map[bson.ObjectID]int{}
	var itemIDs []bson.ObjectID
	var byItem, byUser [][]entry

	for _, s := range scores {
		if s.Score <= 0 {
			continue
		}
		i, ok := itemIndex[s.ItemID]
		if !ok {
			i = len(itemIDs)
			itemIndex[s.ItemID] = i
			itemIDs = append(itemIDs, s.ItemID)
			byItem = append(byItem, nil)
		}
		u, ok := userIndex[s.UserID]
		if !ok {
			u = len(byUser)
			userIndex[s.UserID] = u
			byUser = append(byUser, nil)
		}
		byItem[i] = append(byItem[i], entry{u, s.Score})
		byUser[u] = append(byUser[u], entry{i, s.Score})
	}

	norms := make([]float64, len(itemIDs))
	for i, users := range byItem {
		for _, e := range users {
			norms[i] += e.score * e.score
		}
		norms[i] = math.Sqrt(norms[i])
	}

	// For item i, walk its users and their other items, summing the dot
	// product and the number of common users in arrays reused for every i.
	dot := make([]float64, len(itemIDs))
	common := make([]int, len(itemIDs))
	result := make(map[bson.ObjectID][]Neighbor, len(itemIDs))
	for i, users := range byItem {
		var touched []int
		for _, u := range users {
			for _, j := range byUser[u.index] {
				if j.index == i {
					continue
				}
				if common[j.index] == 0 {
					touched = append(touched, j.index)
				}
				dot[j.index] += u.score * j.score
				common[j.index]++
			}
		}

		var neighbors []Neighbor
		for _, j := range touched {
			n := float64(common[j])
			sim := dot[j] / (norms[i] * norms[j]) * n / (n + Shrinkage)
			if sim > 0 {
				neighbors = append(neighbors, Neighbor{ItemID: itemIDs[j], Score: sim})
			}
			dot[j], common[j] = 0, 0
		}
		if len(neighbors) == 0 {
			continue
		}
		sortNeighbors(neighbors)
		if len(neighbors) > TopNeighbors {
			neighbors = neighbors[:TopNeighbors]
		}
		result[itemIDs[i]] = neighbors
	}
	return result
}

// sortNeighbors sorts by score, highest first; ties by id so results are stable.
func sortNeighbors(ns []Neighbor) {
	slices.SortFunc(ns, func(a, b Neighbor) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return bytes.Compare(a.ItemID[:], b.ItemID[:])
	})
}
