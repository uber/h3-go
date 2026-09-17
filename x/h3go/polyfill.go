/*
 * Copyright 2026 Uber Technologies, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package h3go

import (
	"iter"
	"math"
)

// ContainmentMode selects which cells PolygonToCellsExperimental includes
// relative to the polygon.
type ContainmentMode uint32

const (
	// ContainmentCenter includes cells whose center is inside the polygon.
	ContainmentCenter ContainmentMode = iota
	// ContainmentFull includes cells that are fully contained by the polygon.
	ContainmentFull
	// ContainmentOverlapping includes cells that overlap the polygon at any point.
	ContainmentOverlapping
	// ContainmentOverlappingBbox includes cells whose bounding box overlaps the
	// polygon.
	ContainmentOverlappingBbox
	// ContainmentInvalid is the first invalid mode value and must not be used.
	ContainmentInvalid
)

// flagContainmentModeMask masks the containment-mode bits out of the flags word.
const flagContainmentModeMask = 15

// cellScaleFactor scales a cell's bounding box to be sure it covers the cell.
// childScaleFactor scales it to cover all of the cell's finer-resolution
// children. Both were chosen empirically.
const (
	cellScaleFactor  = 1.1
	childScaleFactor = 1.4
)

// maxEdgeLengthRads is the maximum cell edge length, in radians, for each
// resolution, taken at the center of each base cell at that resolution.
var maxEdgeLengthRads = [MaxResolution + 1]float64{
	0.21577206265130, 0.08308767068495, 0.03148970436439, 0.01190662871439,
	0.00450053330908, 0.00170105523619, 0.00064293917678, 0.00024300820659,
	0.00009184847087, 0.00003471545901, 0.00001312121017, 0.00000495935129,
	0.00000187445860, 0.00000070847876, 0.00000026777980, 0.00000010121125,
}

// northPoleCells and southPoleCells list the cell containing each pole at every
// resolution, used to expand a pole-covering bounding box to a full circle.
var (
	northPoleCells = [MaxResolution + 1]Cell{
		0x8001fffffffffff, 0x81033ffffffffff, 0x820327fffffffff, 0x830326fffffffff,
		0x8403263ffffffff, 0x85032623fffffff, 0x860326237ffffff, 0x870326233ffffff,
		0x880326233bfffff, 0x890326233abffff, 0x8a0326233ab7fff, 0x8b0326233ab0fff,
		0x8c0326233ab03ff, 0x8d0326233ab03bf, 0x8e0326233ab039f, 0x8f0326233ab0399,
	}
	southPoleCells = [MaxResolution + 1]Cell{
		0x80f3fffffffffff, 0x81f2bffffffffff, 0x82f297fffffffff, 0x83f293fffffffff,
		0x84f2939ffffffff, 0x85f29383fffffff, 0x86f29380fffffff, 0x87f29380effffff,
		0x88f29380e1fffff, 0x89f29380e0fffff, 0x8af29380e0d7fff, 0x8bf29380e0d0fff,
		0x8cf29380e0d0dff, 0x8df29380e0d0cff, 0x8ef29380e0d0cc7, 0x8ff29380e0d0cc4,
	}
)

// res0BBoxesRads holds a precomputed bounding box, in radians, for every
// resolution-0 base cell, indexed by base cell number.
var res0BBoxesRads = [NumBaseCells]bbox{
	{north: 1.52480158339146, south: 1.20305471830087, east: -0.60664883654036, west: 0.00568297271999},
	{north: 1.52480158339146, south: 1.17872424267511, east: -0.60664883654036, west: 2.54046980298264},
	{north: 1.52480158339146, south: 1.09069387298096, east: -2.85286053297673, west: 1.64310689027893},
	{north: 1.41845302535151, south: 1.01285145697208, east: 0.00568297271999, west: -1.16770379632602},
	{north: 1.27950477868453, south: 0.97226652536306, east: 0.55556064983494, west: -0.18229924845326},
	{north: 1.32929586572429, south: 0.91898920750071, east: 2.05622344943192, west: 1.08813154278274},
	{north: 1.32899086063916, south: 0.94271815376360, east: -2.29875289606378, west: 3.01700008041993},
	{north: 1.26020983864103, south: 0.84291228415618, east: -0.89971867664861, west: -1.75967359310997},
	{north: 1.21114673854945, south: 0.86170600921069, east: 1.19129757609455, west: 0.43777608996454},
	{north: 1.21075831414294, south: 0.83795331049498, east: -1.72022875779891, west: -2.43793861727138},
	{north: 1.15546530929588, south: 0.78982455384253, east: 2.53659412229266, west: 1.85709133451243},
	{north: 1.15528445067052, south: 0.76641428724335, east: -3.06738507202411, west: 2.53646110244042},
	{north: 1.10121643537669, south: 0.71330093663066, east: 0.09640581900154, west: -0.52154514518248},
	{north: 1.07042472765165, south: 0.67603948819406, east: -0.47984202840088, west: -1.10306159603090},
	{north: 1.03270228748960, south: 0.72356358827215, east: -2.24990138725146, west: -2.74510220919157},
	{north: 1.01929924623886, south: 0.65491232835426, east: 0.63035574240731, west: 0.03537030096470},
	{north: 1.01786037568858, south: 0.58827636737638, east: 1.53192721817065, west: 0.93672682511233},
	{north: 0.98081434136020, south: 0.61076063532947, east: -2.67100636598529, west: 3.06516463008733},
	{north: 0.98106023192774, south: 0.58679836571570, east: 2.02829766214461, west: 1.51334374970280},
	{north: 0.96374551790056, south: 0.55186491737474, east: -1.42976721313659, west: -1.96852202530104},
	{north: 0.87536136210723, south: 0.50008952762292, east: -1.92435613571430, west: -2.41641343219793},
	{north: 0.88611243445554, south: 0.52742963716774, east: -0.95781946324194, west: -1.47628966305930},
	{north: 0.86881343251986, south: 0.50770567021439, east: 1.03236795495839, west: 0.50347284027426},
	{north: 0.89235638181782, south: 0.48781264892508, east: 2.76430302119150, west: 2.29989716697031},
	{north: 0.82570569254601, south: 0.52173101741059, east: 2.30921681461428, west: 1.93198541828980},
	{north: 0.80599330438546, south: 0.40150819579319, east: -3.06417559403240, west: 2.70079300784409},
	{north: 0.81612079704781, south: 0.38396800633226, east: -0.21614378891839, west: -0.70420149722178},
	{north: 0.75822779851431, south: 0.39943555383751, east: -2.34059978084699, west: -2.82127373822444},
	{north: 0.78861390967531, south: 0.38742018303868, east: 0.23115687731652, west: -0.22599491086066},
	{north: 0.71515840341957, south: 0.33012478438475, east: -0.64847976163163, west: -1.08249728121219},
	{north: 0.70359051048414, south: 0.29148673180722, east: 1.71441081857246, west: 1.28443348381696},
	{north: 0.69190629544818, south: 0.28808313184381, east: 0.64863909244647, west: 0.16372369282557},
	{north: 0.64863235654749, south: 0.26290420067147, east: 2.10318098268379, west: 1.69556122548344},
	{north: 0.65722892279906, south: 0.28222653310929, east: 1.30918693285466, west: 0.87594416271685},
	{north: 0.64750997738584, south: 0.24149865709850, east: -1.30272192474556, west: -1.68708570163242},
	{north: 0.62380174028378, south: 0.25522080363509, east: -2.72428423026826, west: 3.10401473237630},
	{north: 0.64228460410023, south: 0.21206753429148, east: -1.67639240992071, west: -2.11772366767341},
	{north: 0.59919175361146, south: 0.21620460836570, east: 2.48592868387690, west: 2.07350353893591},
	{north: 0.55637406851384, south: 0.25276557437230, east: -0.99885388505694, west: -1.32642489358939},
	{north: 0.55648013300665, south: 0.15187401321019, east: 2.87032088421324, west: 2.44642320475367},
	{north: 0.54603687970450, south: 0.15589091511369, east: -2.06789866067060, west: -2.49091419631961},
	{north: 0.51206347752697, south: 0.15522020377124, east: 0.95446767315996, west: 0.54443262110414},
	{north: 0.49767951537101, south: 0.10944898890579, east: -0.04335162263358, west: -0.42900268178569},
	{north: 0.46538045483671, south: 0.06029968637720, east: -0.41240613713421, west: -0.80603623808166},
	{north: 0.44686891066946, south: 0.06926857458503, east: 0.32053284794952, west: -0.07005748900849},
	{north: 0.43208958202064, south: 0.07796440938140, east: -3.06232453079660, west: 2.80602499990282},
	{north: 0.43103892586713, south: 0.02927431919853, east: -2.41589238618422, west: -2.85735809951951},
	{north: 0.38073727558986, south: -0.00297016159959, east: -0.77039553861218, west: -1.14788248745028},
	{north: 0.39113816687141, south: -0.01518764903038, east: 1.49130246958290, west: 1.14714731736311},
	{north: 0.33421063142418, south: 0.02526613430348, east: 1.15141032578749, west: 0.85000706261644},
	{north: 0.38915669778582, south: -0.04371359825454, east: 1.88046353933242, west: 1.48230231380717},
	{north: 0.33787520825987, south: -0.04835090128296, east: -1.12274014380603, west: -1.49454408844749},
	{north: 0.33601418932337, south: -0.06675068178541, east: 2.23792354204464, west: 1.85723423013211},
	{north: 0.31838318078049, south: -0.05821955623722, east: 0.66058854060373, west: 0.25452572938783},
	{north: 0.33630761471457, south: -0.07589541031521, east: -1.47957331741818, west: -1.85981735718264},
	{north: 0.28924817322870, south: -0.09150638064667, east: -1.83561930288569, west: -2.21855897384292},
	{north: 0.26678632252475, south: -0.10058088990867, east: -2.76808651991421, west: 3.12792953247061},
	{north: 0.29285254112587, south: -0.13483165093783, east: 2.61406468380434, west: 2.20466422911705},
	{north: 0.20150342788824, south: -0.10279852729762, east: 0.06881896344365, west: -0.23925229432978},
	{north: 0.21283813275258, south: -0.18626835417891, east: 2.93800440256577, west: 2.57470747655623},
	{north: 0.19587614179884, south: -0.17237030304155, east: -2.16941795427335, west: -2.55405165906601},
	{north: 0.17237030304155, south: -0.19587614179884, east: 0.97217469931645, west: 0.58754099452378},
	{north: 0.18626835417891, south: -0.21283813275258, east: -0.20358825102402, west: -0.56688517703356},
	{north: 0.10279852729762, south: -0.20150342788824, east: -3.07277369014614, west: 2.90234035926002},
	{north: 0.13483165093783, south: -0.29285254112587, east: -0.52752796978545, west: -0.93692842447275},
	{north: 0.10058088990867, south: -0.26678632252475, east: 0.37350613367558, west: -0.01366312111919},
	{north: 0.09150638064667, south: -0.28924817322870, east: 1.30597335070410, west: 0.92303367974687},
	{north: 0.07589541031521, south: -0.33630761471457, east: 1.66201933617161, west: 1.28177529640715},
	{north: 0.05821955623722, south: -0.31838318078049, east: -2.48100411298606, west: -2.88706692420196},
	{north: 0.06675068178541, south: -0.33601418932337, east: -0.90366911154516, west: -1.28435842345769},
	{north: 0.04835090128296, south: -0.33787520825987, east: 2.01885250978376, west: 1.64704856514230},
	{north: 0.04371359825454, south: -0.38915669778582, east: -1.26112911425737, west: -1.65929033978262},
	{north: -0.02526613430348, south: -0.33421063142418, east: -1.99018232780231, west: -2.29158559097336},
	{north: 0.01518764903038, south: -0.39113816687140, east: -1.65029018400690, west: -1.99444533622668},
	{north: 0.00297016159959, south: -0.38073727558986, east: 2.37119711497761, west: 1.99371016613951},
	{north: -0.02927431919853, south: -0.43103892586713, east: 0.72570026740558, west: 0.28423455407029},
	{north: -0.07796440938140, south: -0.43208958202064, east: 0.07926812279319, west: -0.33556765368697},
	{north: -0.06926857458503, south: -0.44686891066946, east: -2.82105980564027, west: 3.07153516458131},
	{north: -0.06029968637720, south: -0.46538045483671, east: 2.72918651645558, west: 2.33555641550814},
	{north: -0.10944898890579, south: -0.49767951537101, east: 3.09824103095621, west: 2.71258997180410},
	{north: -0.15522020377124, south: -0.51206347752697, east: -2.18712498042983, west: -2.59716003248565},
	{north: -0.15589091511369, south: -0.54603687970450, east: 1.07369399291919, west: 0.65067845727018},
	{north: -0.15187401321019, south: -0.55648013300665, east: -0.27127176937655, west: -0.69516944883612},
	{north: -0.25276557437230, south: -0.55637406851385, east: 2.14273876853285, west: 1.81516776000041},
	{north: -0.21620460836570, south: -0.59919175361146, east: -0.65566396971290, west: -1.06808911465388},
	{north: -0.21206753429148, south: -0.64228460410023, east: 1.46520024366909, west: 1.02386898591638},
	{north: -0.25522080363509, south: -0.62380174028378, east: 0.41730842332153, west: -0.03757792121350},
	{north: -0.24149865709850, south: -0.64750997738584, east: 1.83887072884423, west: 1.45450695195737},
	{north: -0.28222653310929, south: -0.65722892279906, east: -1.83240572073513, west: -2.26564849087294},
	{north: -0.26290420067147, south: -0.64863235654749, east: -1.03841167090601, west: -1.44603142810635},
	{north: -0.28808313184381, south: -0.69190629544818, east: -2.49295356114332, west: -2.97786896076422},
	{north: -0.29148673180722, south: -0.70359051048414, east: -1.42718183501734, west: -1.85715916977284},
	{north: -0.33012478438475, south: -0.71515840341957, east: 2.49311289195816, west: 2.05909537237761},
	{north: -0.38742018303868, south: -0.78861390967531, east: -2.91043577627328, west: 2.91559774272914},
	{north: -0.39943555383751, south: -0.75822779851431, east: 0.80099287274280, west: 0.32031891536535},
	{north: -0.38396800633226, south: -0.81612079704781, east: 2.92544886467140, west: 2.43739115636801},
	{north: -0.40150819579319, south: -0.80599330438546, east: 0.07741705955739, west: -0.44079964574570},
	{north: -0.52173101741059, south: -0.82570569254601, east: -0.83237583897551, west: -1.20960723529999},
	{north: -0.48781264892508, south: -0.89235638181782, east: -0.37728963239830, west: -0.84169548661948},
	{north: -0.50770567021439, south: -0.86881343251986, east: -2.10922469863141, west: -2.63811981331554},
	{north: -0.52742963716774, south: -0.88611243445554, east: 2.18377319034785, west: 1.66530299053050},
	{north: -0.50008952762292, south: -0.87536136210723, east: 1.21723651787549, west: 0.72517922139186},
	{north: -0.55186491737474, south: -0.96374551790056, east: 1.71182544045320, west: 1.17307062828876},
	{north: -0.58679836571570, south: -0.98106023192774, east: -1.11329499144518, west: -1.62824890388699},
	{north: -0.61076063532947, south: -0.98081434136020, east: 0.47058628760450, west: -0.07642802350246},
	{north: -0.58827636737638, south: -1.01786037568858, east: -1.60966543541914, west: -2.20486582847747},
	{north: -0.65491232835426, south: -1.01929924623886, east: -2.51123691118248, west: -3.10622235262510},
	{north: -0.72356358827215, south: -1.03270228748960, east: 0.89169126633833, west: 0.39649044439822},
	{north: -0.67603948819406, south: -1.07042472765165, east: 2.66175062518892, west: 2.03853105755889},
	{north: -0.71330093663066, south: -1.10121643537669, east: -3.04518683458825, west: 2.62004750840731},
	{north: -0.76641428724335, south: -1.15528445067052, east: 0.07420758156568, west: -0.60513155114938},
	{north: -0.78982455384253, south: -1.15546530929588, east: -0.60499853129713, west: -1.28450131907736},
	{north: -0.83795331049498, south: -1.21075831414294, east: 1.42136389579088, west: 0.70365403631841},
	{north: -0.86170600921069, south: -1.21114673854945, east: -1.95029507749525, west: -2.70381656362525},
	{north: -0.84291228415618, south: -1.26020983864103, east: 2.24187397694118, west: 1.38191906047983},
	{north: -0.94271815376360, south: -1.32899086063916, east: 0.84283975752601, west: -0.12459257316986},
	{north: -0.91898920750071, south: -1.32929586572429, east: -1.08536920415787, west: -2.05346111080706},
	{north: -0.97226652536306, south: -1.27950477868453, east: -2.58603200375485, west: 2.95929340513654},
	{north: -1.01285145697208, south: -1.41845302535151, east: -3.13590968086981, west: 1.97388885726377},
	{north: -1.09069387298096, south: -1.52480158339146, east: 0.28873212061306, west: -1.49848576331087},
	{north: -1.17872424267511, south: -1.52480158339146, east: 2.53494381704943, west: -0.60112285060716},
	{north: -1.20305471830087, south: -1.52480158339146, east: -0.60112285060716, west: 2.53494381704943},
}

// validRangeBBox is the full valid latitude/longitude domain in degrees. It
// guards the first-vertex containment check from out-of-range coordinates.
var validRangeBBox = bbox{north: halfPiDeg, south: -halfPiDeg, east: piDeg, west: -piDeg}

// validateContainmentMode reports an error if the mode is out of range or sets
// flag bits outside the containment-mode field.
func validateContainmentMode(mode ContainmentMode) error {
	flags := uint32(mode)
	if flags&^uint32(flagContainmentModeMask) != 0 || flags&flagContainmentModeMask >= uint32(ContainmentInvalid) {
		return ErrOptionInvalid
	}

	return nil
}

// baseCellNumToCell returns the resolution-0 cell for a base cell number, or the
// zero cell if the number is out of range.
func baseCellNumToCell(baseCellNum int) Cell {
	if baseCellNum < 0 || baseCellNum >= NumBaseCells {
		return 0
	}

	return setH3Index(0, baseCellNum, centerDigit)
}

// toDegrees returns the bounding box with each coordinate converted from radians
// to degrees.
func (b bbox) toDegrees() bbox {
	return bbox{
		north: b.north * RadsToDegs,
		south: b.south * RadsToDegs,
		east:  b.east * RadsToDegs,
		west:  b.west * RadsToDegs,
	}
}

// cellToBBox returns the bounding box of a cell in degrees. When coverChildren
// is true the box is guaranteed to contain the cell's children at any finer
// resolution. The box is approximate and may carry a significant margin.
func cellToBBox(cell Cell, coverChildren bool) bbox {
	res := cell.Resolution()

	var out bbox
	if res == 0 {
		out = res0BBoxesRads[cell.BaseCellNumber()].toDegrees()
	} else {
		// cell is always valid here (it comes from the hierarchy walk), so the
		// center projection cannot fail.
		center, _ := cell.LatLng()
		edge := maxEdgeLengthRads[res] * RadsToDegs
		lngRatio := 1 / math.Cos(center.Lat*DegsToRads)
		out = bbox{
			north: center.Lat + edge,
			south: center.Lat - edge,
			east:  center.Lng + edge*lngRatio,
			west:  center.Lng - edge*lngRatio,
		}
	}

	// Scale the box, which also normalizes it to the lat/lng domain.
	scale := cellScaleFactor
	if coverChildren {
		scale = childScaleFactor
	}

	out = out.scaled(scale)

	if cell == northPoleCells[res] {
		out.north = halfPiDeg
	}

	if cell == southPoleCells[res] {
		out.south = -halfPiDeg
	}

	// A box covering a pole spans the full longitude domain, making it a circle
	// around the pole.
	if out.north == halfPiDeg || out.south == -halfPiDeg {
		out.east = piDeg
		out.west = -piDeg
	}

	return out
}

// nextCell returns the next cell to visit in the depth-first traversal of the
// global cell hierarchy: the next sibling, ascending to a parent's next sibling
// when the current cell is the last sibling, or the next base cell at the top.
func nextCell(cell Cell) Cell {
	res := cell.Resolution()
	for {
		if res == 0 {
			return baseCellNumToCell(cell.BaseCellNumber() + 1)
		}

		parent := cell.setResolution(res-1).setIndexDigit(res, digitMask)

		digit := cell.indexDigit(res)
		if digit < invalidDigit-1 {
			step := 1
			// Skip the missing center child of a pentagon.
			if parent.IsPentagon() && digit == centerDigit {
				step = 2
			}

			return cell.setIndexDigit(res, digit+step)
		}

		res--
		cell = parent
	}
}

// targetCellInPolygon reports whether a cell at the target resolution should be
// included for the given containment mode.
func targetCellInPolygon(cell Cell, polygon GeoPolygon, bboxes []bbox, mode ContainmentMode) bool {
	if mode == ContainmentCenter || mode == ContainmentOverlapping || mode == ContainmentOverlappingBbox {
		center, _ := cell.LatLng()
		if pointInsidePolygon(polygon, bboxes, center) {
			return true
		}
	}

	if mode == ContainmentOverlapping || mode == ContainmentOverlappingBbox {
		// If the polygon is wholly contained by the cell, its first vertex maps
		// to this cell. Guard against out-of-range coordinates first.
		firstVertex := polygon.GeoLoop[0]
		if validRangeBBox.contains(firstVertex) {
			polygonCell, _ := LatLngToCell(firstVertex, cell.Resolution())
			if polygonCell == cell {
				return true
			}
		}
	}

	if mode == ContainmentFull || mode == ContainmentOverlapping || mode == ContainmentOverlappingBbox {
		if targetCellBoundaryInPolygon(cell, polygon, bboxes, mode) {
			return true
		}
	}

	if mode == ContainmentOverlappingBbox {
		return cellBBoxOverlapsPolygon(cell, polygon, bboxes)
	}

	return false
}

// targetCellBoundaryInPolygon checks containment or crossing of a target-res
// cell's exact boundary against the polygon, per the containment mode.
func targetCellBoundaryInPolygon(cell Cell, polygon GeoPolygon, bboxes []bbox, mode ContainmentMode) bool {
	boundary, _ := cell.Boundary()
	box := cellToBBox(cell, false)

	if (mode == ContainmentFull || mode == ContainmentOverlappingBbox) &&
		cellBoundaryInsidePolygon(polygon, bboxes, boundary, box) {
		return true
	}

	// Center inclusion was already checked, so for overlap only the boundary
	// crossing remains.
	if (mode == ContainmentOverlapping || mode == ContainmentOverlappingBbox) &&
		cellBoundaryCrossesPolygon(polygon, bboxes, boundary, box) {
		return true
	}

	return false
}

// cellBBoxOverlapsPolygon reports whether a child-covering cell bounding box
// overlaps the polygon, used by the overlapping-bbox mode.
func cellBBoxOverlapsPolygon(cell Cell, polygon GeoPolygon, bboxes []bbox) bool {
	box := cellToBBox(cell, true)
	if !bboxes[0].overlaps(box) {
		return false
	}

	boxBoundary := box.toCellBoundary()

	return box.containsBBox(bboxes[0]) ||
		pointInsidePolygon(polygon, bboxes, boxBoundary[0]) ||
		cellBoundaryCrossesPolygon(polygon, bboxes, boxBoundary, box)
}

// coarseCellInPolygon reports whether a coarser-than-target cell is wholly
// contained by the polygon (so all of its children are included). It returns the
// containment result and whether the traversal should recurse into the children.
func coarseCellInPolygon(cell Cell, polygon GeoPolygon, bboxes []bbox) (contained, recurse bool) {
	box := cellToBBox(cell, true)
	if !bboxes[0].overlaps(box) {
		return false, false
	}

	if bboxes[0].containsBBox(box) {
		boxBoundary := box.toCellBoundary()
		if cellBoundaryInsidePolygon(polygon, bboxes, boxBoundary, box) {
			return true, false
		}
	}

	return false, true
}

// polygonCompactCells yields the compact set of cells covering the polygon: each
// cell is either at the target resolution or a coarser cell whose every child is
// contained. Inclusion at the target resolution follows the containment mode.
func polygonCompactCells(polygon GeoPolygon, bboxes []bbox, res int, mode ContainmentMode) iter.Seq[Cell] {
	return func(yield func(Cell) bool) {
		cell := baseCellNumToCell(0)
		for cell != 0 {
			cellRes := cell.Resolution()

			if cellRes == res {
				if targetCellInPolygon(cell, polygon, bboxes, mode) && !yield(cell) {
					return
				}

				cell = nextCell(cell)

				continue
			}

			contained, recurse := coarseCellInPolygon(cell, polygon, bboxes)
			if contained {
				if !yield(cell) {
					return
				}

				cell = nextCell(cell)

				continue
			}

			if recurse {
				// cell is coarser than the target, so a center child always exists.
				cell, _ = cell.CenterChild(cellRes + 1)

				continue
			}

			cell = nextCell(cell)
		}
	}
}

// PolygonToCellsExperimental fills the polygon with cells of the given
// resolution, including a cell according to the containment mode. The optional
// maxNumCellsReturn caps the number of cells; exceeding it returns
// ErrMemoryBounds. Output ordering is not significant.
func PolygonToCellsExperimental(polygon GeoPolygon, res int, mode ContainmentMode, maxNumCellsReturn ...int64) ([]Cell, error) {
	maxNumCells := int64(math.MaxInt64)
	if len(maxNumCellsReturn) > 0 {
		maxNumCells = maxNumCellsReturn[0]
	}

	if len(polygon.GeoLoop) == 0 {
		return nil, nil
	}

	if res < 0 || res > MaxResolution {
		return nil, ErrResolutionDomain
	}

	if err := validateContainmentMode(mode); err != nil {
		return nil, err
	}

	bboxes := bboxesFromGeoPolygon(polygon)

	var (
		out   []Cell
		count int64
	)

	for compact := range polygonCompactCells(polygon, bboxes, res, mode) {
		for child := range compact.childCells(res) {
			if count >= maxNumCells {
				return nil, ErrMemoryBounds
			}

			out = append(out, child)
			count++
		}
	}

	return out, nil
}
