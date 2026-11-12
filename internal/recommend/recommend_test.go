package recommend

import (
	"math"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

func intPtr(v int) *int { return &v }

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestWeight(t *testing.T) {
	tests := []struct {
		typ   domain.InteractionType
		value *int
		want  float64
	}{
		{domain.InteractionView, nil, 1},
		{domain.InteractionListen, nil, 2},
		{domain.InteractionCorrection, nil, 1},
		{domain.InteractionFavorite, nil, 4},
		{domain.InteractionRead, intPtr(0), 1.5},
		{domain.InteractionRead, intPtr(79), 1.5},
		{domain.InteractionRead, intPtr(80), 3},
		{domain.InteractionRead, intPtr(100), 3},
		{domain.InteractionRate, intPtr(1), -2},
		{domain.InteractionRate, intPtr(2), -1},
		{domain.InteractionRate, intPtr(3), 1},
		{domain.InteractionRate, intPtr(4), 3},
		{domain.InteractionRate, intPtr(5), 5},
	}
	for _, tt := range tests {
		if got := Weight(tt.typ, tt.value); got != tt.want {
			t.Errorf("Weight(%s, %v) = %v, want %v", tt.typ, tt.value, got, tt.want)
		}
	}
}

func in(typ domain.InteractionType, v *int) domain.Interaction {
	return domain.Interaction{Type: typ, Value: v}
}

func TestScoreIsMaxWeightWithoutRating(t *testing.T) {
	tests := []struct {
		name string
		ins  []domain.Interaction
		want float64
	}{
		{"none", nil, 0},
		{"view and listen", []domain.Interaction{in(domain.InteractionView, nil), in(domain.InteractionListen, nil)}, 2},
		{"short and finished read", []domain.Interaction{in(domain.InteractionRead, intPtr(30)), in(domain.InteractionRead, intPtr(90))}, 3},
		{"favorite and view", []domain.Interaction{in(domain.InteractionView, nil), in(domain.InteractionFavorite, nil)}, 4},
	}
	for _, tt := range tests {
		if got := Score(tt.ins); got != tt.want {
			t.Errorf("%s: Score = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A rating is explicit feedback: it replaces the implicit score, both up and down.
func TestScoreRatingOverridesImplicit(t *testing.T) {
	tests := []struct {
		name string
		ins  []domain.Interaction
		want float64
	}{
		{"only bad rating", []domain.Interaction{in(domain.InteractionRate, intPtr(1))}, -2},
		{"view then rating 1 is disliked", []domain.Interaction{in(domain.InteractionView, nil), in(domain.InteractionRate, intPtr(1))}, -2},
		{"finished read then rating 2", []domain.Interaction{in(domain.InteractionRead, intPtr(100)), in(domain.InteractionRate, intPtr(2))}, -1},
		{"rating before favorite still wins", []domain.Interaction{in(domain.InteractionRate, intPtr(4)), in(domain.InteractionFavorite, nil)}, 3},
		{"rating 3 below finished read", []domain.Interaction{in(domain.InteractionRead, intPtr(90)), in(domain.InteractionRate, intPtr(3))}, 1},
		{"rating 5 above listen", []domain.Interaction{in(domain.InteractionListen, nil), in(domain.InteractionRate, intPtr(5))}, 5},
	}
	for _, tt := range tests {
		if got := Score(tt.ins); got != tt.want {
			t.Errorf("%s: Score = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSimilarities(t *testing.T) {
	u1, u2, u3 := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	a, b, c := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	scores := []UserItemScore{
		{UserID: u1, ItemID: a, Score: 2}, {UserID: u1, ItemID: b, Score: 4},
		{UserID: u2, ItemID: a, Score: 1}, {UserID: u2, ItemID: b, Score: 2}, {UserID: u2, ItemID: c, Score: 3},
		{UserID: u3, ItemID: c, Score: 5},
		{UserID: u3, ItemID: a, Score: -2}, // not liked, must be ignored
	}
	sims := Similarities(scores)

	// ||a|| = sqrt(5), ||b|| = sqrt(20), ||c|| = sqrt(34)
	wantAB := (2*4 + 1*2) / (math.Sqrt(5) * math.Sqrt(20)) * 2 / (2 + Shrinkage)
	wantAC := (1 * 3) / (math.Sqrt(5) * math.Sqrt(34)) * 1 / (1 + Shrinkage)
	wantBC := (2 * 3) / (math.Sqrt(20) * math.Sqrt(34)) * 1 / (1 + Shrinkage)

	na := sims[a]
	if len(na) != 2 || na[0].ItemID != b || na[1].ItemID != c {
		t.Fatalf("neighbors of a = %+v, want [b c]", na)
	}
	if !near(na[0].Score, wantAB) || !near(na[1].Score, wantAC) {
		t.Errorf("sim(a,b)=%v sim(a,c)=%v, want %v %v", na[0].Score, na[1].Score, wantAB, wantAC)
	}

	// Similarity is symmetric.
	for _, nb := range sims[b] {
		if nb.ItemID == a && !near(nb.Score, wantAB) {
			t.Errorf("sim(b,a) = %v, want %v", nb.Score, wantAB)
		}
		if nb.ItemID == c && !near(nb.Score, wantBC) {
			t.Errorf("sim(b,c) = %v, want %v", nb.Score, wantBC)
		}
	}
	for id, ns := range sims {
		for _, nb := range ns {
			if nb.ItemID == id {
				t.Errorf("item %s is its own neighbor", id.Hex())
			}
		}
	}
}

func TestSimilaritiesKeepsTopNeighbors(t *testing.T) {
	// One user who liked 30 items: every item gets 29 neighbors before the cut.
	u := bson.NewObjectID()
	var scores []UserItemScore
	for i := range 30 {
		scores = append(scores, UserItemScore{UserID: u, ItemID: bson.NewObjectID(), Score: float64(i + 1)})
	}
	for id, ns := range Similarities(scores) {
		if len(ns) != TopNeighbors {
			t.Fatalf("item %s has %d neighbors, want %d", id.Hex(), len(ns), TopNeighbors)
		}
		for k := 1; k < len(ns); k++ {
			if ns[k].Score > ns[k-1].Score {
				t.Fatalf("neighbors are not sorted: %+v", ns)
			}
		}
	}
}

func TestRecommend(t *testing.T) {
	read, liked, disliked := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	x, y := bson.NewObjectID(), bson.NewObjectID()
	now := time.Now()
	history := []UserItemScore{
		{ItemID: read, Score: 3, LastAt: now},
		{ItemID: liked, Score: 1, LastAt: now.Add(-time.Hour)},
		{ItemID: disliked, Score: -2, LastAt: now},
	}
	neighbors := map[bson.ObjectID][]Neighbor{
		read:     {{ItemID: x, Score: 0.2}, {ItemID: liked, Score: 0.9}},
		liked:    {{ItemID: x, Score: 0.5}, {ItemID: y, Score: 0.3}},
		disliked: {{ItemID: y, Score: 0.9}}, // not liked, must not count
	}

	got := Recommend(history, neighbors, 10)
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want 2: %+v", len(got), got)
	}
	// x = 0.2*3 + 0.5*1 = 1.1 (biggest part from read), y = 0.3*1 = 0.3
	if got[0].ItemID != x || !near(got[0].Score, 1.1) || got[0].Because != read {
		t.Errorf("first = %+v, want x with score 1.1 because of read", got[0])
	}
	if got[1].ItemID != y || !near(got[1].Score, 0.3) || got[1].Because != liked {
		t.Errorf("second = %+v, want y with score 0.3 because of liked", got[1])
	}

	if top := Recommend(history, neighbors, 1); len(top) != 1 || top[0].ItemID != x {
		t.Errorf("Recommend(n=1) = %+v, want only x", top)
	}
}

func TestLikedHistoryKeepsLastLiked(t *testing.T) {
	now := time.Now()
	var history []UserItemScore
	for i := range HistorySize + 10 {
		history = append(history, UserItemScore{ItemID: bson.NewObjectID(), Score: 1, LastAt: now.Add(time.Duration(i) * time.Minute)})
	}
	history = append(history, UserItemScore{ItemID: bson.NewObjectID(), Score: -1, LastAt: now.Add(time.Hour * 24)})

	liked := LikedHistory(history)
	if len(liked) != HistorySize {
		t.Fatalf("got %d items, want %d", len(liked), HistorySize)
	}
	if liked[0].Score <= 0 || !liked[0].LastAt.Equal(now.Add(time.Duration(HistorySize+9)*time.Minute)) {
		t.Errorf("first item = %+v, want the newest liked one", liked[0])
	}
}

// The live mode builds neighbors from pair statistics; they must equal the
// precomputed ones for the same data.
func TestNeighborsFromStatsMatchesSimilarities(t *testing.T) {
	users := make([]bson.ObjectID, 30)
	items := make([]bson.ObjectID, 40)
	for i := range users {
		users[i] = bson.NewObjectID()
	}
	for i := range items {
		items[i] = bson.NewObjectID()
	}
	// A fixed pattern: user u likes items (u*7+k) % 40 with different scores.
	var scores []UserItemScore
	byUser := map[bson.ObjectID]map[bson.ObjectID]float64{}
	for u, uid := range users {
		byUser[uid] = map[bson.ObjectID]float64{}
		for k := range 8 {
			item := items[(u*7+k*k)%len(items)]
			s := float64((u+k)%5 + 1)
			if _, dup := byUser[uid][item]; dup {
				continue
			}
			byUser[uid][item] = s
			scores = append(scores, UserItemScore{UserID: uid, ItemID: item, Score: s})
		}
	}

	norms := map[bson.ObjectID]float64{}
	type key struct{ i, j bson.ObjectID }
	stats := map[key]*PairStat{}
	for _, liked := range byUser {
		for i, si := range liked {
			norms[i] += si * si
			for j, sj := range liked {
				if i == j {
					continue
				}
				p := stats[key{i, j}]
				if p == nil {
					p = &PairStat{ItemI: i, ItemJ: j}
					stats[key{i, j}] = p
				}
				p.Dot += si * sj
				p.Common++
			}
		}
	}
	for id, sq := range norms {
		norms[id] = math.Sqrt(sq)
	}
	var pairs []PairStat
	for _, p := range stats {
		pairs = append(pairs, *p)
	}

	want := Similarities(scores)
	got := NeighborsFromStats(pairs, norms)
	if len(got) != len(want) {
		t.Fatalf("got neighbors for %d items, want %d", len(got), len(want))
	}
	for id, w := range want {
		g := got[id]
		if len(g) != len(w) {
			t.Fatalf("item %s: %d neighbors, want %d", id.Hex(), len(g), len(w))
		}
		for k := range w {
			if g[k].ItemID != w[k].ItemID || !near(g[k].Score, w[k].Score) {
				t.Fatalf("item %s neighbor %d = %+v, want %+v", id.Hex(), k, g[k], w[k])
			}
		}
	}
}
