package sudoku

import (
	"math/bits"
)

// BoardState represents the current state of candidates for each of the 81 cells.
type BoardState struct {
	Candidates [81]uint16 // Bits 1-9 represent if digits 1-9 are possible candidates.
}

// cellPeers stores the precomputed 20 unique peer indices for each cell.
var cellPeers [81][20]int

func init() {
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			idx := r*9 + c
			peerIdx := 0

			// Row peers (8 cells)
			for c2 := 0; c2 < 9; c2++ {
				if c2 != c {
					cellPeers[idx][peerIdx] = r*9 + c2
					peerIdx++
				}
			}

			// Column peers (8 cells)
			for r2 := 0; r2 < 9; r2++ {
				if r2 != r {
					cellPeers[idx][peerIdx] = r2*9 + c
					peerIdx++
				}
			}

			// Box peers (4 cells that are not in the same row/column)
			boxRowStart := (r / 3) * 3
			boxColStart := (c / 3) * 3
			for r2 := boxRowStart; r2 < boxRowStart+3; r2++ {
				for c2 := boxColStart; c2 < boxColStart+3; c2++ {
					if r2 != r && c2 != c {
						cellPeers[idx][peerIdx] = r2*9 + c2
						peerIdx++
					}
				}
			}
		}
	}
}

// Propagate performs Naked Single propagation:
// If a cell has exactly one candidate left, it eliminates that candidate from all of its 20 peers.
// Returns false if any cell is left with 0 candidates (contradiction).
func (s *BoardState) Propagate() bool {
	var propagated [81]bool
	changed := true

	for changed {
		changed = false
		for i := 0; i < 81; i++ {
			if !propagated[i] {
				cands := s.Candidates[i]
				if bits.OnesCount16(cands) == 1 {
					propagated[i] = true
					valBit := cands // Only one bit is set

					for _, peer := range cellPeers[i] {
						if s.Candidates[peer]&valBit != 0 {
							s.Candidates[peer] &= ^valBit
							if s.Candidates[peer] == 0 {
								return false // Contradiction: peer has no options left
							}
							changed = true
						}
					}
				}
			}
		}
	}
	return true
}

// PropagateHidden performs Hidden Single propagation for all 27 Sudoku units (rows, columns, boxes).
// If a candidate value is only possible in a single cell within a unit, that cell is set to that value.
// Returns (ok, changed) where:
// - ok is false if a candidate has 0 possible cells in a unit (contradiction).
// - changed is true if any cell's candidates were reduced to a single value.
func (s *BoardState) PropagateHidden() (bool, bool) {
	changed := false

	// 1. Row Units
	for r := 0; r < 9; r++ {
		for d := 1; d <= 9; d++ {
			mask := uint16(1 << d)
			possCount := 0
			lastIdx := -1
			for c := 0; c < 9; c++ {
				idx := r*9 + c
				if s.Candidates[idx]&mask != 0 {
					possCount++
					lastIdx = idx
				}
			}
			if possCount == 0 {
				return false, false // Contradiction: digit d must appear in row r but has no possible cells
			}
			if possCount == 1 {
				if s.Candidates[lastIdx] != mask {
					s.Candidates[lastIdx] = mask
					changed = true
				}
			}
		}
	}

	// 2. Column Units
	for c := 0; c < 9; c++ {
		for d := 1; d <= 9; d++ {
			mask := uint16(1 << d)
			possCount := 0
			lastIdx := -1
			for r := 0; r < 9; r++ {
				idx := r*9 + c
				if s.Candidates[idx]&mask != 0 {
					possCount++
					lastIdx = idx
				}
			}
			if possCount == 0 {
				return false, false // Contradiction: digit d must appear in col c but has no possible cells
			}
			if possCount == 1 {
				if s.Candidates[lastIdx] != mask {
					s.Candidates[lastIdx] = mask
					changed = true
				}
			}
		}
	}

	// 3. Box Units
	for b := 0; b < 9; b++ {
		boxRowStart := (b / 3) * 3
		boxColStart := (b % 3) * 3
		for d := 1; d <= 9; d++ {
			mask := uint16(1 << d)
			possCount := 0
			lastIdx := -1
			for dr := 0; dr < 3; dr++ {
				for dc := 0; dc < 3; dc++ {
					idx := (boxRowStart+dr)*9 + (boxColStart + dc)
					if s.Candidates[idx]&mask != 0 {
						possCount++
						lastIdx = idx
					}
				}
			}
			if possCount == 0 {
				return false, false // Contradiction: digit d must appear in box b but has no possible cells
			}
			if possCount == 1 {
				if s.Candidates[lastIdx] != mask {
					s.Candidates[lastIdx] = mask
					changed = true
				}
			}
		}
	}

	return true, changed
}

// PropagateAll runs Naked Single and Hidden Single propagation repeatedly until no more progress is made.
func (s *BoardState) PropagateAll() bool {
	for {
		if !s.Propagate() {
			return false
		}
		ok, changed := s.PropagateHidden()
		if !ok {
			return false
		}
		if !changed {
			break
		}
	}
	return true
}

// Solve recursively solves the Sudoku board using MRV backtracking with constraint propagation.
func (s *BoardState) Solve() bool {
	// Propagate constraints first to reduce search space
	if !s.PropagateAll() {
		return false
	}

	// Find the cell with the Minimum Remaining Values (MRV heuristic) > 1
	bestIdx := -1
	minCands := 10

	for i := 0; i < 81; i++ {
		count := bits.OnesCount16(s.Candidates[i])
		if count == 0 {
			return false // Contradiction
		}
		if count > 1 && count < minCands {
			minCands = count
			bestIdx = i
		}
	}

	// If no cell has more than 1 candidate left, the board is successfully solved
	if bestIdx == -1 {
		return true
	}

	// Try each possible value for the best cell candidate
	cands := s.Candidates[bestIdx]
	for d := 1; d <= 9; d++ {
		mask := uint16(1 << d)
		if cands&mask != 0 {
			// Save current state using value copy
			savedState := *s

			// Speculatively place the value
			s.Candidates[bestIdx] = mask

			// Recurse to solve the rest of the board
			if s.Solve() {
				return true
			}

			// Backtrack: restore saved state
			*s = savedState
		}
	}

	return false
}

// SolveTdoku is the entrypoint to solve Sudoku using the Tdoku-inspired Bit-Triad algorithm.
func SolveTdoku(board *[9][9]int) bool {
	var s BoardState

	// Initialize all cells with all possible candidates (bits 1-9 set: 0x3FE)
	for i := 0; i < 81; i++ {
		s.Candidates[i] = 0x3FE
	}

	// Populate the board state with initial clues
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			v := board[r][c]
			if v != 0 {
				idx := r*9 + c
				mask := uint16(1 << v)

				// Check if the clue is already invalid under initial constraints
				if s.Candidates[idx]&mask == 0 {
					return false
				}
				s.Candidates[idx] = mask
			}
		}
	}

	// Run the solver
	if !s.Solve() {
		return false
	}

	// Write the solution back to the grid
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			idx := r*9 + c
			cands := s.Candidates[idx]
			if bits.OnesCount16(cands) != 1 {
				return false
			}
			board[r][c] = bits.TrailingZeros16(cands)
		}
	}

	return true
}
