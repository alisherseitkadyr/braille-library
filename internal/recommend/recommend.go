package recommend

import (
	"bytes"
	"cmp"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Candidate is a recommended item. Because is the history item that gave
// the biggest contribution to Score.
type Candidate struct {
	ItemID  bson.ObjectID
	Score   float64
	Because bson.ObjectID
}

// LikedHistory returns the user's last HistorySize liked items (score > 0),
// most recent first.
func LikedHistory(history []UserItemScore) []UserItemScore {
	var liked []UserItemScore
	for _, h := range history {
		if h.Score > 0 {
			liked = append(liked, h)
		}
	}
	slices.SortFunc(liked, func(a, b UserItemScore) int {
		return b.LastAt.Compare(a.LastAt)
	})
	if len(liked) > HistorySize {
		liked = liked[:HistorySize]
	}
	return liked
}

// Recommend scores every neighbor j of the user's liked items:
//
//	score(j) = sum_i sim(i,j) * s(u,i)
//
// Items in history (any score) are excluded. It returns the top n
// candidates, or all of them if n <= 0.
func Recommend(history []UserItemScore, neighbors map[bson.ObjectID][]Neighbor, n int) []Candidate {
	seen := make(map[bson.ObjectID]bool, len(history))
	for _, h := range history {
		seen[h.ItemID] = true
	}

	type acc struct {
		score, best float64
		because     bson.ObjectID
	}
	totals := map[bson.ObjectID]*acc{}
	for _, h := range LikedHistory(history) {
		for _, nb := range neighbors[h.ItemID] {
			if seen[nb.ItemID] {
				continue
			}
			a := totals[nb.ItemID]
			if a == nil {
				a = &acc{}
				totals[nb.ItemID] = a
			}
			contribution := nb.Score * h.Score
			a.score += contribution
			if contribution > a.best {
				a.best, a.because = contribution, h.ItemID
			}
		}
	}

	result := make([]Candidate, 0, len(totals))
	for id, a := range totals {
		result = append(result, Candidate{ItemID: id, Score: a.score, Because: a.because})
	}
	slices.SortFunc(result, func(a, b Candidate) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return bytes.Compare(a.ItemID[:], b.ItemID[:])
	})
	if n > 0 && len(result) > n {
		result = result[:n]
	}
	return result
}
