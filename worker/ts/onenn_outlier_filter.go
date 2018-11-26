package ts

import (
	"math"
	"sort"
)

/*
OneNNOutlierFilter .
	主要过滤doc/pics/bad_data_ts.png中离群的坏点;
	输入参数为MaxBadPer: 预期的坏点所占最大比例;

	算法:
		1) 计算所有点, 到最近相邻点的距离, 得到距离集合Dists;
		2) 选取Dists中位的(1-MaxBadPer)个数, 计算他们的avg和sd;
		3) 设置距离参数 dist = avg + 3*sd;
		4) 迭代做并集操作, 将距离小于 dist 的点划分为一类;
		5) 将划分得到的所有类, 按照类内的点数个数, 从大到小排序;
		6) 依次剔除点数个数最小的类, 并保持最多剔除MaxBadPer的点;

	复杂度:
		O(n^2) now
*/
func OneNNOutlierFilter(points XYPoints, maxBadPer float64) XYPoints {
	// step 1
	_, dists := MinEuclidDists(points)

	// step 2
	sort.Float64s(dists)
	badLen := int(float64(len(points)) * maxBadPer)
	bestDists := dists[:len(points)-badLen]

	// step 3
	avg := AVG(bestDists)
	sd := SD(bestDists)
	dist := 6 * (avg + 3*sd)

	// step 4
	classified := EuclidDistClassify(points, dist)

	// step 5
	sort.Slice(classified, func(i, j int) bool {
		return len(classified[i]) > len(classified[j])
	})

	// step 6
	maxRm := int(float64(len(points)) * maxBadPer)
	rmed := 0
	for {
		n := len(classified[len(classified)-1])
		if rmed+n > maxRm {
			break
		}

		rmed += n
		classified = classified[:len(classified)-1]
	}

	filtered := make(XYPoints, 0, len(points)-rmed)
	for _, ps := range classified {
		for _, p := range ps {
			filtered = append(filtered, p)
		}
	}

	filtered.SortByX()
	return filtered
}

/*
EuclidDistClassify 对points做归类, 将相互距离小于dist的点归为一类;

	算法:
		合并利用并查集;
*/
func EuclidDistClassify(points XYPoints, dist float64) []XYPoints {
	// 初始化并查集
	classes := make([]int, len(points))
	for i := range classes {
		classes[i] = i // 初始各自为一类
	}
	root := func(i int) int {
		rt := i
		for classes[rt] != rt {
			rt = classes[rt]
		}
		classes[i] = rt
		return rt
	}
	merge := func(i, j int) {
		classes[root(i)] = root(j)
	}

	// TODO(zhangyuanjia): 优化到O(nlogn)
	for i := range points {
		for j := range points {
			d := EuclidDist(points[i], points[j])
			if d <= dist {
				merge(i, j)
			}
		}
	}

	classMap := make(map[int]XYPoints)
	for i := range points {
		rt := root(i)
		classMap[rt] = append(classMap[rt], points[i])
	}

	classified := make([]XYPoints, len(classMap))
	for _, ps := range classMap {
		classified = append(classified, ps)
	}
	return classified
}

// MinEuclidDists 计算各个点距离最近的点距
// TODO(zhangyuanjia): 优化该函数, 当前复杂度为 O(n^2), 可以优化到 O(nlogn)
func MinEuclidDists(points XYPoints) ([]int, []float64) {
	nns := make([]int, len(points)) // nearest neighbor
	dists := make([]float64, len(points))
	for i := range points {
		nnID := 0
		minDist := math.Inf(1)
		for j := range points {
			if i == j {
				continue
			}
			dist := EuclidDist(points[i], points[j])
			if dist < minDist {
				minDist = dist
				nnID = j
			}
		}

		nns[i] = nnID
		dists[i] = minDist
	}
	return nns, dists
}

// EuclidDist .
func EuclidDist(p1, p2 XYPoint) float64 {
	return math.Sqrt((p1.X-p2.X)*(p1.X-p2.X) +
		(p1.Y-p2.Y)*(p1.Y-p2.Y))
}
