package scraper

import (
	"testing"
)

type testPinCount struct {
	input  string
	result int
}

type testSrcSet struct {
	srcset   string
	original string
}

//func TestPinsCountStringToInt(t *testing.T) {
//
//	var pinCountTests = []testPinCount{
//		{"0", 0},
//		{"15", 15},
//		{"999", 999},
//		{"1,291", 1291},
//		{"8,432", 8432},
//		{"15,987", 15987},
//	}
//
//	for _, testPair := range pinCountTests {
//		result := pinsCountStringToInt(testPair.input)
//		if result != testPair.result {
//			t.Error("Pin count string ", testPair.input, " should result in ", testPair.result)
//		}
//	}
//}

func TestGetOriginalImage(t *testing.T) {

	var srcSetTests = []testSrcSet{
		{"https://i.pinimg.com/236x/93/74/99/93749980da966aef00c4e18d1000f4e1.jpg 1x, https://i.pinimg.com/474x/93/74/99/93749980da966aef00c4e18d1000f4e1.jpg 2x, https://i.pinimg.com/736x/93/74/99/93749980da966aef00c4e18d1000f4e1.jpg 3x, https://i.pinimg.com/originals/93/74/99/93749980da966aef00c4e18d1000f4e1.jpg 4x", "https://i.pinimg.com/originals/93/74/99/93749980da966aef00c4e18d1000f4e1.jpg"},
		{"https://i.pinimg.com/236x/49/18/b7/4918b740da399c815f6fdba556a0fb2b.jpg 1x, https://i.pinimg.com/474x/49/18/b7/4918b740da399c815f6fdba556a0fb2b.jpg 2x, https://i.pinimg.com/736x/49/18/b7/4918b740da399c815f6fdba556a0fb2b.jpg 3x, https://i.pinimg.com/originals/49/18/b7/4918b740da399c815f6fdba556a0fb2b.jpg 4x", "https://i.pinimg.com/originals/49/18/b7/4918b740da399c815f6fdba556a0fb2b.jpg"},
		{"https://i.pinimg.com/236x/e9/48/9b/e9489b231c1c8393622c6eec79c5e6f8.jpg 1x, https://i.pinimg.com/474x/e9/48/9b/e9489b231c1c8393622c6eec79c5e6f8.jpg 2x, https://i.pinimg.com/736x/e9/48/9b/e9489b231c1c8393622c6eec79c5e6f8.jpg 3x, https://i.pinimg.com/originals/e9/48/9b/e9489b231c1c8393622c6eec79c5e6f8.jpg 4x", "https://i.pinimg.com/originals/e9/48/9b/e9489b231c1c8393622c6eec79c5e6f8.jpg"},
	}

	for _, srcset := range srcSetTests {
		if srcset.original != regexOriginalImageLink.FindString(srcset.srcset) {
			t.Error("srcset ", srcset.srcset, " should contain ", srcset.original)
		}
	}
}
