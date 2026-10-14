//go:build integration

package mouse

import "testing"

func TestTheRightButtonReadsAsUpWhileNobodyHoldsIt(t *testing.T) {
	if (System{}).Aiming() {
		t.Error("the right button reads as held, though nobody holds it while the tests run")
	}
}
