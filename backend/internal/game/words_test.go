package game

import (
	"slices"
	"testing"
)

func TestWordBankChangesAtNextTurn(t *testing.T) {
	r, _, _ := setup(t)
	choices := slices.Clone(r.choices)
	var bank WordBank
	r.bank = &bank
	if _, err := bank.Replace([]byte("风筝\n沙漏\n雪橇\n")); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{"", "风筝\n沙漏", "风筝\n风筝\n沙漏", "风筝\n沙漏\n<html>"} {
		if _, err := bank.Replace([]byte(invalid)); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
	if !slices.Equal(choices, r.choices) {
		t.Fatal("updated choices already shown to the drawer")
	}
	r.startDrawing(1)
	if r.word != choices[1] {
		t.Fatal("changed the chosen answer")
	}
	r.nextTurn()
	slices.Sort(r.choices)
	want := []string{"风筝", "沙漏", "雪橇"}
	slices.Sort(want)
	if !slices.Equal(r.choices, want) {
		t.Fatalf("new choices: %v", r.choices)
	}
}
