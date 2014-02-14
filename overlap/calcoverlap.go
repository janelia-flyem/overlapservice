package overlap

import (
	"math"
	"sort"
)

// yzPair is a key containing the y and z value for a given x run
type yzPair struct {
	y int32
	z int32
}

// bodyPair contains two bodies, smallest id first
type bodyPair struct {
	body1 uint32
	body2 uint32
}

func newBodyPair(body1, body2 uint32) *bodyPair {
	if body2 < body1 {
		body1, body2 = body2, body1
	}
	return &bodyPair{body1, body2}
}

type xIndex struct {
	bodyID uint32
	x      int32
	length int32
}

type xIndices []xIndex

func (slice xIndices) Len() int {
	return len(slice)
}

func (slice xIndices) Less(i, j int) bool {
	return slice[i].x < slice[j].x
}

func (slice xIndices) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// findLowerBound returns the array position to the element less than or equal to xval
func findLowerBound(xval int32, xindices xIndices) (index int, found bool) {
	index = sort.Search(len(xindices), func(i int) bool { return xindices[len(xindices)-i-1].x <= xval })
	if index == len(xindices) {
		found = false
	} else {
		found = true
		index = len(xindices) - index - 1
	}

	return
}

// findLowerBound returns the array position to the element equal to xval or false
func findEqual(xval int32, xindices xIndices) (index int, found bool) {
	index, found = findLowerBound(xval, xindices)
	if found {
		if xindices[index].x != xval {
			found = false
		}
	}

	return
}

// loadSparseBodyYZs indexes the RLE into a YZ map for easy analysis and returns the size of the body
func loadSparseBodyYZs(sparse_body sparseBody, yzmaplist map[yzPair]xIndices) (uint32) {
        var bodysize uint32
        bodysize = 0

        // slice of x's
        xindices := xIndices{}
        ycurr := int32(-100000000)
        zcurr := int32(-100000000)
        for _, chunk := range sparse_body.rle {
                y := chunk.y
                z := chunk.z
                yzpair := yzPair{y, z}
                bodysize += uint32(chunk.length)

                if y != ycurr || z != zcurr {
                        if len(xindices) > 0 {
                                yzpairold := yzPair{ycurr, zcurr}
                                yzmaplist[yzpairold] = append(yzmaplist[yzpairold], xindices...)
                        }

                        ycurr = y
                        zcurr = z
                        if _, found := yzmaplist[yzpair]; !found {
                                yzmaplist[yzpair] = xIndices{}
                        }
                        xindices = xIndices{}

                }
                xindices = append(xindices, xIndex{sparse_body.bodyID, chunk.x, chunk.length})
        }
        if len(xindices) > 0 {
                yzpairold := yzPair{ycurr, zcurr}
                yzmaplist[yzpairold] = append(yzmaplist[yzpairold], xindices...)
        }

        return bodysize
}

// computeStats finds the volume and surface area for each body
func computeStats(sparse_bodies sparseBodies) resultList {
	stats_slice := resultList{}

        for _, sparse_body := range sparse_bodies {
                // hash of yz value to sorted slice of xIndices
                var yzmaplist = make(map[yzPair]xIndices)
                
                bodyid := sparse_body.bodyID
                // grab volume
                bodyvolume := loadSparseBodyYZs(sparse_body, yzmaplist)

                // sort all xindices
                for _, xindices := range yzmaplist {
                        sort.Sort(xindices)
                }

                // maximum number of adjacencies possible
                var totaladjacencies uint32
                totaladjacencies = 0
	
                // contains overlap results for each body pair
	        body_pairs := make(map[bodyPair]uint32)

 		for _, chunk := range sparse_body.rle {
			y := chunk.y
			z := chunk.z
			xmin := chunk.x
			xmax := xmin + chunk.length

                        // maximum possible adjacency for this run
                        totaladjacencies += uint32(chunk.length * 4 + 2)

			// find total number of adjancencies to itself (use 0 body id since there are no such body ids)
			if xlist, found := yzmaplist[yzPair{y + 1, z}]; found {
				overlap(body_pairs, xlist, xmin, xmax, 0)
			}
			if xlist, found := yzmaplist[yzPair{y - 1, z}]; found {
				overlap(body_pairs, xlist, xmin, xmax, 0)
			}
			if xlist, found := yzmaplist[yzPair{y, z + 1}]; found {
				overlap(body_pairs, xlist, xmin, xmax, 0)
			}
			if xlist, found := yzmaplist[yzPair{y, z - 1}]; found {
				overlap(body_pairs, xlist, xmin, xmax, 0)
			}

			if xlist, found := yzmaplist[yzPair{y, z}]; found {
				// check if there is a pixel with a smaller x in the same body
				if index, found := findLowerBound(xmin-1, xlist); found {
					xval := xlist[index]
					if (xval.length+xval.x-1) == (xmin-1) {
						body_pairs[*newBodyPair(0, xval.bodyID)] += 1
					}
				}

				// check if there is a pixel greater in x in the same body
				if index, found := findEqual(xmax, xlist); found {
					body_pairs[*newBodyPair(0, xlist[index].bodyID)] += 1
				}
			}
		}
                
                bodyarea := totaladjacencies - body_pairs[*newBodyPair(0, bodyid)]
 
		tempslice := []uint32{bodyid, bodyvolume, bodyarea}
		stats_slice = append(stats_slice, tempslice)
	}

	// put body pairs with the largest surface area first
	sort.Sort(sort.Reverse(stats_slice))

	return stats_slice
}

// computeOverlap finds the overlap between the list of bodies using the RLE, only bodies with overlap are returned
func computeOverlap(sparse_bodies sparseBodies) resultList {
	// smallest rle first -- more memory use (or largest first for more computation)
	sort.Sort(sparse_bodies)

	// each pair of body will have 2x the overlap except for the first and last body
	var first_sparse_body = sparse_bodies[0]
	var last_sparse_body = sparse_bodies[len(sparse_bodies)-1]

	// hash of yz value to sorted slice of xIndices
	var yzmaplist = make(map[yzPair]xIndices)

	// preprocess rles -- do not load the first body
	for _, sparse_body := range sparse_bodies[1:] {
                loadSparseBodyYZs(sparse_body, yzmaplist)
	}

	// sort all xindices
	for _, xindices := range yzmaplist {
		sort.Sort(xindices)
	}

	// contains overlap results for each body pair
	body_pairs := make(map[bodyPair]uint32)

	// iterate one body at a time to calculate overlap, do not need to examine the last body
	for _, sparse_body := range sparse_bodies[0 : len(sparse_bodies)-1] {
		bodyid1 := sparse_body.bodyID
		for _, chunk := range sparse_body.rle {
			y := chunk.y
			z := chunk.z
			xmin := chunk.x
			xmax := xmin + chunk.length

			// examine adjacencies
			if xlist, found := yzmaplist[yzPair{y + 1, z}]; found {
				overlap(body_pairs, xlist, xmin, xmax, bodyid1)
			}
			if xlist, found := yzmaplist[yzPair{y - 1, z}]; found {
				overlap(body_pairs, xlist, xmin, xmax, bodyid1)
			}
			if xlist, found := yzmaplist[yzPair{y, z + 1}]; found {
				overlap(body_pairs, xlist, xmin, xmax, bodyid1)
			}
			if xlist, found := yzmaplist[yzPair{y, z - 1}]; found {
				overlap(body_pairs, xlist, xmin, xmax, bodyid1)
			}

			if xlist, found := yzmaplist[yzPair{y, z}]; found {
				// check if there is a pixel with a smaller x
				// the pixel could be of the same body so check
				if index, found := findLowerBound(xmin-1, xlist); found {
					xval := xlist[index]
					if (bodyid1 != xval.bodyID) && (xval.length+xval.x-1) == (xmin-1) {
						body_pairs[*newBodyPair(bodyid1, xval.bodyID)] += 1
					}
				}

				// check if there is a pixel greater in x
				// the pixel could be of the same body so check
				if index, found := findEqual(xmax, xlist); found {
					if bodyid1 != xlist[index].bodyID {
						body_pairs[*newBodyPair(bodyid1, xlist[index].bodyID)] += 1
					}
				}
			}
		}
	}

	for pair, val := range body_pairs {
		if pair.body1 != first_sparse_body.bodyID && pair.body1 != last_sparse_body.bodyID && pair.body2 != first_sparse_body.bodyID && pair.body2 != last_sparse_body.bodyID {
			body_pairs[pair] = val / 2
		}
	}

	overlap_slice := resultList{}
	for pair, val := range body_pairs {
		tempslice := []uint32{pair.body1, pair.body2, val}
		overlap_slice = append(overlap_slice, tempslice)
	}

	// put body pairs with the largest overlap first
	sort.Sort(sort.Reverse(overlap_slice)) // by size of overlap

	return overlap_slice
}

// overlap calculates the overlap between bodyid1 and different bodies and puts the value in body_pairs
func overlap(body_pairs map[bodyPair]uint32, xlist xIndices, xmin int32, xmax int32, bodyid1 uint32) {
	var maxindex int
	var minindex int
	var found bool
	// grab the last index less than or equal to the largest index in the body
	if maxindex, found = findLowerBound(xmax-1, xlist); !found {
		return
	}

	// get lower bound from min
	if minindex, found = findLowerBound(xmin-1, xlist); !found {
		minindex = 0
	}

	for i := minindex; i <= maxindex; i += 1 {
		if xlist[i].bodyID != bodyid1 {
			length := xlist[i].length
			start := xlist[i].x
			if start < xmin {
				length -= (xmin - start)
				start = xmin
			}

			if length > 0 {
				body_pairs[*(newBodyPair(bodyid1, xlist[i].bodyID))] += uint32(math.Min(float64(length), float64(xmax-start)))
			}
		}
	}
}
