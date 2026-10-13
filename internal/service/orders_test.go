package service

import "testing"

func TestOrderQuantitiesAndRounding(t *testing.T) {
	for _, test := range []struct {
		quantity, unit string
		want           int64
	}{{"4.125", "KG", 4125}, {"1", "ITEM", 1000}, {"1.000", "ITEM", 1000}, {"0.001", "KG", 1}} {
		got, err := ParseQuantity(test.quantity, test.unit)
		if err != nil || got != test.want {
			t.Fatalf("%s %s: %d %v", test.quantity, test.unit, got, err)
		}
	}
	for _, test := range []struct{ quantity, unit string }{{"0", "KG"}, {"-1", "KG"}, {"1.5", "ITEM"}, {"NaN", "KG"}, {"1e3", "KG"}, {"1.0001", "KG"}} {
		if _, err := ParseQuantity(test.quantity, test.unit); err == nil {
			t.Fatalf("accepted %s", test.quantity)
		}
	}
	for _, test := range []struct{ price, milli, want int64 }{{8000, 4000, 32000}, {999, 500, 500}, {1, 499, 0}, {1, 500, 1}, {25000, 1000, 25000}} {
		got, err := ItemSubtotal(test.price, test.milli)
		if err != nil || got != test.want {
			t.Fatalf("rounding=%d want=%d", got, test.want)
		}
	}
}
