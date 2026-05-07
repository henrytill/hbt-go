package belnap

import "testing"

func TestScalarNotTruthTable(t *testing.T) {
	expected := [4]Value{
		Unknown, False, True, Both,
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		got := a.Not()
		if got != expected[i] {
			t.Errorf("%v.Not() = %v, want %v", a, got, expected[i])
		}
	}
}

func TestToBool(t *testing.T) {
	val, ok := True.ToBool()
	if !ok || !val {
		t.Errorf("True.ToBool() = (%v, %v), want (true, true)", val, ok)
	}
	val, ok = False.ToBool()
	if !ok || val {
		t.Errorf("False.ToBool() = (%v, %v), want (false, true)", val, ok)
	}
	val, ok = Unknown.ToBool()
	if ok {
		t.Errorf("Unknown.ToBool() = (%v, %v), want (_, false)", val, ok)
	}
	val, ok = Both.ToBool()
	if ok {
		t.Errorf("Both.ToBool() = (%v, %v), want (_, false)", val, ok)
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		v    Value
		want string
	}{
		{True, "True"},
		{False, "False"},
		{Unknown, "Unknown"},
		{Both, "Both"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestScalarAndTruthTable(t *testing.T) {
	expected := [4][4]Value{
		{Unknown, Unknown, False, False},
		{Unknown, True, False, Both},
		{False, False, False, False},
		{False, Both, False, Both},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.And(b)
			if got != expected[i][j] {
				t.Errorf("%v.And(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarOrTruthTable(t *testing.T) {
	expected := [4][4]Value{
		{Unknown, True, Unknown, True},
		{True, True, True, True},
		{Unknown, True, False, Both},
		{True, True, Both, Both},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.Or(b)
			if got != expected[i][j] {
				t.Errorf("%v.Or(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarImpliesTruthTable(t *testing.T) {
	expected := [4][4]Value{
		{Unknown, True, Unknown, True},
		{Unknown, True, False, Both},
		{True, True, True, True},
		{True, True, Both, Both},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.Implies(b)
			if got != expected[i][j] {
				t.Errorf("%v.Implies(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarMergeTruthTable(t *testing.T) {
	expected := [4][4]Value{
		{Unknown, True, False, Both},
		{True, True, Both, Both},
		{False, Both, False, Both},
		{Both, Both, Both, Both},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.Merge(b)
			if got != expected[i][j] {
				t.Errorf("%v.Merge(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarConsensusTruthTable(t *testing.T) {
	expected := [4][4]Value{
		{Unknown, Unknown, Unknown, Unknown},
		{Unknown, True, Unknown, True},
		{Unknown, Unknown, False, False},
		{Unknown, True, False, Both},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.Consensus(b)
			if got != expected[i][j] {
				t.Errorf("%v.Consensus(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarLeqTruthTable(t *testing.T) {
	// Truth order: False < {Unknown, Both} < True; Unknown and Both incomparable.
	expected := [4][4]bool{
		{true, true, false, false},
		{false, true, false, false},
		{true, true, true, true},
		{false, true, false, true},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.LeqTruth(b)
			if got != expected[i][j] {
				t.Errorf("%v.LeqTruth(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarLeqKnowledgeTable(t *testing.T) {
	// Knowledge order: Unknown < {True, False} < Both; True and False incomparable.
	expected := [4][4]bool{
		{true, true, true, true},
		{false, true, false, true},
		{false, false, true, true},
		{false, false, false, true},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.LeqKnowledge(b)
			if got != expected[i][j] {
				t.Errorf("%v.LeqKnowledge(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarQueries(t *testing.T) {
	if Unknown.IsKnown() {
		t.Error("Unknown.IsKnown() should be false")
	}
	if !True.IsKnown() {
		t.Error("True.IsKnown() should be true")
	}
	if !False.IsKnown() {
		t.Error("False.IsKnown() should be true")
	}
	if !Both.IsKnown() {
		t.Error("Both.IsKnown() should be true")
	}

	if Unknown.IsDetermined() {
		t.Error("Unknown.IsDetermined() should be false")
	}
	if !True.IsDetermined() {
		t.Error("True.IsDetermined() should be true")
	}
	if !False.IsDetermined() {
		t.Error("False.IsDetermined() should be true")
	}
	if Both.IsDetermined() {
		t.Error("Both.IsDetermined() should be false")
	}

	if Unknown.IsContradicted() {
		t.Error("Unknown.IsContradicted() should be false")
	}
	if True.IsContradicted() {
		t.Error("True.IsContradicted() should be false")
	}
	if False.IsContradicted() {
		t.Error("False.IsContradicted() should be false")
	}
	if !Both.IsContradicted() {
		t.Error("Both.IsContradicted() should be true")
	}
}
