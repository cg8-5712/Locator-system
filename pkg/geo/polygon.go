package geo

import "math"

const epsilon = 1e-9

func ContainsPoint(latitude float64, longitude float64, polygon [][]float64) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	j := len(polygon) - 1
	for i := 0; i < len(polygon); i++ {
		a := polygon[i]
		b := polygon[j]
		if len(a) < 2 || len(b) < 2 {
			j = i
			continue
		}

		if pointOnSegment(longitude, latitude, a[1], a[0], b[1], b[0]) {
			return true
		}

		intersects := ((a[0] > latitude) != (b[0] > latitude)) &&
			(longitude < (b[1]-a[1])*(latitude-a[0])/(b[0]-a[0])+a[1])
		if intersects {
			inside = !inside
		}

		j = i
	}

	return inside
}

func pointOnSegment(px float64, py float64, ax float64, ay float64, bx float64, by float64) bool {
	cross := (px-ax)*(by-ay) - (py-ay)*(bx-ax)
	if math.Abs(cross) > epsilon {
		return false
	}

	dot := (px-ax)*(bx-ax) + (py-ay)*(by-ay)
	if dot < -epsilon {
		return false
	}

	lengthSquared := (bx-ax)*(bx-ax) + (by-ay)*(by-ay)
	if dot-lengthSquared > epsilon {
		return false
	}

	return true
}
