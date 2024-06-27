package ollama

import (
	"fmt"
	"math"
)

// cosineDistance calculates the cosine distance between two slices of float64.
func cosineDistance(slice1, slice2 []float64) (float64, error) {
	// Calculate the dot product and magnitudes
	var dotProduct, magnitude1, magnitude2 float64
	for i := range slice1 {
		dotProduct += slice1[i] * slice2[i]
		magnitude1 += slice1[i] * slice1[i]
		magnitude2 += slice2[i] * slice2[i]
	}

	// Calculate the cosine similarity
	magnitude1 = math.Sqrt(magnitude1)
	magnitude2 = math.Sqrt(magnitude2)

	if magnitude1 == 0 || magnitude2 == 0 {
		return 0, fmt.Errorf("magnitude of embeddings must not be zero")
	}

	cosineSimilarity := dotProduct / (magnitude1 * magnitude2)
	distance := 1 - cosineSimilarity // Convert similarity to distance
	return distance, nil
}

// euclideanDistance calculates the Euclidean distance between two slices of float64.
func euclideanDistance(slice1, slice2 []float64) (float64, error) {
	var sum float64
	for i := range slice1 {
		diff := slice1[i] - slice2[i]
		sum += diff * diff
	}
	return math.Sqrt(sum), nil
}

// manhattanDistance calculates the Manhattan distance between two slices of float64.
func manhattanDistance(slice1, slice2 []float64) (float64, error) {
	var sum float64
	for i := range slice1 {
		sum += math.Abs(slice1[i] - slice2[i])
	}
	return sum, nil
}

// chebyshevDistance calculates the Chebyshev distance between two slices of float64.
func chebyshevDistance(slice1, slice2 []float64) (float64, error) {
	var maxDiff float64
	for i := range slice1 {
		diff := math.Abs(slice1[i] - slice2[i])
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff, nil
}

// hammingDistance calculates the Hamming distance between two slices of float64.
func hammingDistance(slice1, slice2 []float64) (float64, error) {
	var count float64
	for i := range slice1 {
		if slice1[i] != slice2[i] {
			count++
		}
	}
	return count, nil
}
