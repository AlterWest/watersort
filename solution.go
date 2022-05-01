package watersort

import (
	"container/heap"
	"errors"
	"fmt"
	"log"
	"math/rand"
)

type Solution struct {
	State State
	Steps []Step
	Score int
}

// Clone returns a deep copy of s.
func (s Solution) Clone() Solution {
	state := s.State.Clone()
	steps := make([]Step, len(s.Steps))
	copy(steps, s.Steps)

	return Solution{
		State: state,
		Steps: steps,
		Score: s.Score,
	}
}

// PossibleSteps returns all available next steps.
// If there are no possible moves (meaning the game is lost) it returns nil.
//
// First, the function creates a map of colors to possible destinations
// (i.e. bottles that are not full and where c is the top color, and empty bottles).
//
// Then it iterates over each bottle, determines its top color, and finds
// possible destination for this color using the precomputed map.
//
// Precomputing possible destinations per color reduces the bottle selection complexity from O(n²) to O(n)
// (where n is the number of bottles/colors), assuming that Bottle.TopColor() is O(1).
// (Bottle.TopColor() needs to skip empty spaces, of which there are (usually) 2× m, where m is the bottle size.
// Assuming a linear relationship between n and m, armortized runtime of Bottle.TopColor() is constant.
// Usually n > m.)
func (s Solution) PossibleSteps() []Step {
	destinationsByColor := make(map[Color][]int)
	for i, b := range s.State.Bottles {
		if b.FreeSlots() == 0 {
			continue
		}
		tc := b.TopColor()
		destinationsByColor[tc] = append(destinationsByColor[tc], i)
	}

	var ret []Step
	for srcIndex, src := range s.State.Bottles {
		tc := src.TopColor()
		if tc == Empty {
			continue
		}

		for _, dstIndex := range append(destinationsByColor[tc], destinationsByColor[Empty]...) {
			if srcIndex == dstIndex {
				continue
			}

			ret = append(ret, Step{From: srcIndex, To: dstIndex})
		}
	}

	rand.Shuffle(len(ret), func(i, j int) {
		ret[i], ret[j] = ret[j], ret[i]
	})

	return ret
}

type Step struct {
	From, To int
}

func (s Step) String() string {
	return fmt.Sprintf("pour %2d onto %2d", s.From+1, s.To+1)
}

type Heap struct {
	Solutions []Solution
}

func (h Heap) Len() int {
	return len(h.Solutions)
}

func (h Heap) Less(i, j int) bool {
	return h.Solutions[i].Distance < h.Solutions[j].Distance
	}

func (h *Heap) Swap(i, j int) {
	h.Solutions[i], h.Solutions[j] = h.Solutions[j], h.Solutions[i]
}

func (h *Heap) Push(x any) {
	h.Solutions = append(h.Solutions, x.(Solution))
}

func (h *Heap) Pop() any {
	last := h.Len() - 1
	s := h.Solutions[last]
	h.Solutions = h.Solutions[:last]
	return s
}

var ErrNoSolution = errors.New("there is no solution")

// FindSolution calculates an optimal solution for s using an A* search algorithm.
//
// The score of each (partial) solution is calculated as the sum of the number
// of steps so far (len(Solution.Steps)) and Solution.State.MinRequiredMoves().
//
// If s is unsolvable, an error is returned.
// Use `errors.Is(ErrNoSolution)` to distinguish between this and other errors.
func FindSolution(s State) ([]Step, error) {
	sol := Solution{
		State: s,
	}

	// h holds partial solutions.
	// Pop() returns (one of) the solution closest to a solved state.
	h := &Heap{}
	heap.Init(h)
	heap.Push(h, sol)

	// seen holds the CRC32 checksum of previously seen states to avoid cycles.
	seen := make(map[uint32]bool)

	for len(h.Solutions) > 0 {
		base := heap.Pop(h).(Solution)

		for _, step := range base.PossibleSteps() {
			next := base.Clone()

			if err := next.State.Pour(step.From, step.To); err != nil {
				log.Printf("State.Pour(%d, %d): %v", step.From, step.To, err)
				continue
			}

			chk := next.State.Checksum()
			if seen[chk] {
				continue
			}

			next.Steps = append(next.Steps, step)

			minRequiredMoves := next.State.MinRequiredMoves()
			next.Score = len(next.Steps) + minRequiredMoves
			// log.Printf("Distance: %2d + %2d = %2d", len(next.Steps), minRequiredMoves, next.Distance)
			if minRequiredMoves == 0 {
				log.Printf("Evaluated %d states to find solution", len(seen))
				return next.Steps, nil
			}

			seen[chk] = true
			heap.Push(h, next)
		}
	}
	return nil, fmt.Errorf("evaluated %d states: %w", len(seen), ErrNoSolution)
}
