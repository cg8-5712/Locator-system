package geo

import "math"

const earthRadiusMeters = 6371000.0

func DistanceMeters(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	phi1 := degreesToRadians(lat1)
	phi2 := degreesToRadians(lat2)
	deltaPhi := degreesToRadians(lat2 - lat1)
	deltaLambda := degreesToRadians(lng2 - lng1)

	sinPhi := math.Sin(deltaPhi / 2)
	sinLambda := math.Sin(deltaLambda / 2)

	a := sinPhi*sinPhi + math.Cos(phi1)*math.Cos(phi2)*sinLambda*sinLambda
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

func BearingDegrees(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	phi1 := degreesToRadians(lat1)
	phi2 := degreesToRadians(lat2)
	deltaLambda := degreesToRadians(lng2 - lng1)

	y := math.Sin(deltaLambda) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(deltaLambda)
	theta := math.Atan2(y, x)

	return math.Mod(radiansToDegrees(theta)+360, 360)
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
}

func radiansToDegrees(value float64) float64 {
	return value * 180 / math.Pi
}
