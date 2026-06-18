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

import "math"

// Exported limits and conversion constants, matching the H3 C library.
const (
	// MaxResolution is the finest H3 resolution.
	MaxResolution = 15
	// MaxCellBndryVerts is the maximum number of vertices in a CellBoundary.
	MaxCellBndryVerts = 10
	// NumBaseCells is the number of resolution-0 base cells.
	NumBaseCells = 122
	// NumIcosaFaces is the number of faces on the icosahedron.
	NumIcosaFaces = 20
	// NumPentagons is the number of pentagons at each resolution.
	NumPentagons = 12
	// InvalidH3Index is the zero value returned for an invalid index.
	InvalidH3Index = 0
	// DegsToRads converts degrees to radians when multiplied.
	DegsToRads = math.Pi / 180.0
	// RadsToDegs converts radians to degrees when multiplied.
	RadsToDegs = 180.0 / math.Pi

	epsilon          = 1e-16
	m2PI             = 2 * math.Pi
	mSqrt7           = 2.6457513110645905905016157536392604257102
	mRSqrt7          = 0.37796447300922722721451653623418006081576
	mRSin60          = 1.1547005383792515290182975610039149112953
	mSqrt3Half       = 0.8660254037844386467637231707529361834714
	mOneSeventh      = 1.0 / 7.0
	mOneThird        = 1.0 / 3.0
	mAP7RotRads      = 0.333473172251832115336090755351601070065900389
	invRes0UGnomonic = 2.61803398874989588842
	res0UGnomonic    = 0.38196601125010500003
	maxFaceCoord     = 2

	// numHexVerts and numPentVerts are the topological vertex counts of a
	// hexagon and a pentagon cell, respectively.
	numHexVerts  = 6
	numPentVerts = 5

	// fltEpsilon is the 32-bit float epsilon used to detect when a cell-boundary
	// edge intersection coincides with an existing vertex, matching the H3 C
	// library's use of FLT_EPSILON.
	fltEpsilon = 1.1920928955078125e-07

	// faceNeighbors quadrant indices: the direction from a face to the adjacent
	// face that shares the corresponding pair of axes.
	dirIJ = 1
	dirKI = 2
	dirJK = 3

	// earthRadiusKm is the authalic (equal-area) radius of the Earth in
	// kilometers, used to convert spherical measures to physical units.
	earthRadiusKm = 6371.007180918475

	// h3Init has all 15 digit slots set to 7 (invalid); mode/res/base cell are 0.
	h3Init = 35184372088831

	centerDigit  = 0
	kAxesDigit   = 1
	jAxesDigit   = 2
	jkAxesDigit  = 3
	iAxesDigit   = 4
	ikAxesDigit  = 5
	ijAxesDigit  = 6
	invalidDigit = 7
	numDigits    = 7

	// H3 index bit-layout offsets and masks, matching h3_h3Index.h and
	// h3_constants.h.
	modeOffset       = 59  // H3_MODE_OFFSET
	resolutionOffset = 52  // H3_RES_OFFSET
	baseCellOffset   = 45  // H3_BC_OFFSET
	perDigitOffset   = 3   // H3_PER_DIGIT_OFFSET
	digitMask        = 7   // H3_DIGIT_MASK
	resolutionMask   = 0xF // low 4 bits of the resolution field

	// numCellEdges is the number of directed edges originating at a cell.
	numCellEdges = 6
	// numEdgeCells is the number of cells a directed edge connects.
	numEdgeCells = 2
	// invalidVertexNum marks a vertex number that is not valid for a cell.
	invalidVertexNum = -1
	// invalidRotations marks a base cell that is not found on a queried face.
	invalidRotations = -1

	// invalidBaseCell is the sentinel in baseCellNeighbors marking the missing
	// neighbor across a pentagon's deleted k-axis edge.
	invalidBaseCell = 127
)

var (
	// isBaseCellPentagon maps base cell number to whether it is a pentagon. There
	// are exactly 12 pentagons at every resolution, one for each vertex of the
	// icosahedron.
	isBaseCellPentagon = [128]bool{
		4: true, 14: true, 24: true, 38: true, 49: true, 58: true,
		63: true, 72: true, 83: true, 97: true, 107: true, 117: true,
	}

	// unitIjkToDigitLUT precomputes the digit for each normalized unit IJK.
	// Valid unit IJK coordinates have at most one of i/j/k as 1 and the rest 0.
	// Indexed by [i][j][k] for i,j,k in {0,1}. Invalid entries map to invalidDigit.
	unitIjkToDigitLUT = [2][2][2]int{
		{
			{centerDigit, kAxesDigit},
			{jAxesDigit, jkAxesDigit},
		},
		{
			{iAxesDigit, ikAxesDigit},
			{ijAxesDigit, invalidDigit},
		},
	}

	faceCenterPoint = [NumIcosaFaces]vec3d{
		{0.2199307791404606, 0.6583691780274996, 0.7198475378926182},
		{-0.2139234834501421, 0.1478171829550703, 0.9656017935214205},
		{0.1092625278784797, -0.4811951572873210, 0.8697775121287253},
		{0.7428567301586791, -0.3593941678278028, 0.5648005936517033},
		{0.8112534709140969, 0.3448953237639384, 0.4721387736413930},
		{-0.1055498149613921, 0.9794457296411413, 0.1718874610009365},
		{-0.8075407579970092, 0.1533552485898818, 0.5695261994882688},
		{-0.2846148069787907, -0.8644080972654206, 0.4144792552473539},
		{0.7405621473854482, -0.6673299564565524, -0.0789837646326737},
		{0.8512303986474293, 0.4722343788582681, -0.2289137388687808},
		{-0.7405621473854481, 0.6673299564565524, 0.0789837646326737},
		{-0.8512303986474292, -0.4722343788582682, 0.2289137388687808},
		{0.1055498149613919, -0.9794457296411413, -0.1718874610009365},
		{0.8075407579970092, -0.1533552485898819, -0.5695261994882688},
		{0.2846148069787908, 0.8644080972654204, -0.4144792552473539},
		{-0.7428567301586791, 0.3593941678278027, -0.5648005936517033},
		{-0.8112534709140971, -0.3448953237639382, -0.4721387736413930},
		{-0.2199307791404607, -0.6583691780274996, -0.7198475378926182},
		{0.2139234834501420, -0.1478171829550704, -0.9656017935214205},
		{-0.1092625278784796, 0.4811951572873210, -0.8697775121287253},
	}

	faceAxesAzRadsCII = [NumIcosaFaces][3]float64{
		{5.619958268523939882, 3.525563166130744542, 1.431168063737548730},
		{5.760339081714187279, 3.665943979320991689, 1.571548876927796127},
		{0.780213654393430055, 4.969003859179821079, 2.874608756786625655},
		{0.430469363979999913, 4.619259568766391033, 2.524864466373195467},
		{6.130269123335111400, 4.035874020941915804, 1.941478918548720291},
		{2.692877706530642877, 0.598482604137447119, 4.787272808923838195},
		{2.982963003477243874, 0.888567901084048369, 5.077358105870439581},
		{3.532912002790141181, 1.438516900396945656, 5.627307105183336758},
		{3.494305004259568154, 1.399909901866372864, 5.588700106652763840},
		{3.003214169499538391, 0.908819067106342928, 5.097609271892733906},
		{5.930472956509811562, 3.836077854116615875, 1.741682751723420374},
		{0.138378484090254847, 4.327168688876645809, 2.232773586483450311},
		{0.448714947059150361, 4.637505151845541521, 2.543110049452346120},
		{0.158629650112549365, 4.347419854898940135, 2.253024752505744869},
		{5.891865957979238535, 3.797470855586042958, 1.703075753192847583},
		{2.711123289609793325, 0.616728187216597771, 4.805518392002988683},
		{3.294508837434268316, 1.200113735041072948, 5.388903939827463911},
		{3.804819692245439833, 1.710424589852244509, 5.899214794638635174},
		{3.664438879055192436, 1.570043776661997111, 5.758833981448388027},
		{2.361378999196363184, 0.266983896803167583, 4.455774101589558636},
	}

	baseCellCWOffsetPent = map[int][2]int{
		4:   {-1, -1},
		14:  {2, 6},
		24:  {1, 5},
		38:  {3, 7},
		49:  {0, 9},
		58:  {4, 8},
		63:  {11, 15},
		72:  {12, 16},
		83:  {10, 19},
		97:  {13, 17},
		107: {14, 18},
		117: {-1, -1},
	}

	faceIjkBaseCells = [20][3][3][3]baseCellRotation{
		{ // face 0
			{{{16, 0}, {18, 0}, {24, 0}}, {{33, 0}, {30, 0}, {32, 3}}, {{49, 1}, {48, 3}, {50, 3}}},
			{{{8, 0}, {5, 5}, {10, 5}}, {{22, 0}, {16, 0}, {18, 0}}, {{41, 1}, {33, 0}, {30, 0}}},
			{{{4, 0}, {0, 5}, {2, 5}}, {{15, 1}, {8, 0}, {5, 5}}, {{31, 1}, {22, 0}, {16, 0}}},
		},
		{ // face 1
			{{{2, 0}, {6, 0}, {14, 0}}, {{10, 0}, {11, 0}, {17, 3}}, {{24, 1}, {23, 3}, {25, 3}}},
			{{{0, 0}, {1, 5}, {9, 5}}, {{5, 0}, {2, 0}, {6, 0}}, {{18, 1}, {10, 0}, {11, 0}}},
			{{{4, 1}, {3, 5}, {7, 5}}, {{8, 1}, {0, 0}, {1, 5}}, {{16, 1}, {5, 0}, {2, 0}}},
		},
		{ // face 2
			{{{7, 0}, {21, 0}, {38, 0}}, {{9, 0}, {19, 0}, {34, 3}}, {{14, 1}, {20, 3}, {36, 3}}},
			{{{3, 0}, {13, 5}, {29, 5}}, {{1, 0}, {7, 0}, {21, 0}}, {{6, 1}, {9, 0}, {19, 0}}},
			{{{4, 2}, {12, 5}, {26, 5}}, {{0, 1}, {3, 0}, {13, 5}}, {{2, 1}, {1, 0}, {7, 0}}},
		},
		{ // face 3
			{{{26, 0}, {42, 0}, {58, 0}}, {{29, 0}, {43, 0}, {62, 3}}, {{38, 1}, {47, 3}, {64, 3}}},
			{{{12, 0}, {28, 5}, {44, 5}}, {{13, 0}, {26, 0}, {42, 0}}, {{21, 1}, {29, 0}, {43, 0}}},
			{{{4, 3}, {15, 5}, {31, 5}}, {{3, 1}, {12, 0}, {28, 5}}, {{7, 1}, {13, 0}, {26, 0}}},
		},
		{ // face 4
			{{{31, 0}, {41, 0}, {49, 0}}, {{44, 0}, {53, 0}, {61, 3}}, {{58, 1}, {65, 3}, {75, 3}}},
			{{{15, 0}, {22, 5}, {33, 5}}, {{28, 0}, {31, 0}, {41, 0}}, {{42, 1}, {44, 0}, {53, 0}}},
			{{{4, 4}, {8, 5}, {16, 5}}, {{12, 1}, {15, 0}, {22, 5}}, {{26, 1}, {28, 0}, {31, 0}}},
		},
		{ // face 5
			{{{50, 0}, {48, 0}, {49, 3}}, {{32, 0}, {30, 3}, {33, 3}}, {{24, 3}, {18, 3}, {16, 3}}},
			{{{70, 0}, {67, 0}, {66, 3}}, {{52, 3}, {50, 0}, {48, 0}}, {{37, 3}, {32, 0}, {30, 3}}},
			{{{83, 0}, {87, 3}, {85, 3}}, {{74, 3}, {70, 0}, {67, 0}}, {{57, 1}, {52, 3}, {50, 0}}},
		},
		{ // face 6
			{{{25, 0}, {23, 0}, {24, 3}}, {{17, 0}, {11, 3}, {10, 3}}, {{14, 3}, {6, 3}, {2, 3}}},
			{{{45, 0}, {39, 0}, {37, 3}}, {{35, 3}, {25, 0}, {23, 0}}, {{27, 3}, {17, 0}, {11, 3}}},
			{{{63, 0}, {59, 3}, {57, 3}}, {{56, 3}, {45, 0}, {39, 0}}, {{46, 3}, {35, 3}, {25, 0}}},
		},
		{ // face 7
			{{{36, 0}, {20, 0}, {14, 3}}, {{34, 0}, {19, 3}, {9, 3}}, {{38, 3}, {21, 3}, {7, 3}}},
			{{{55, 0}, {40, 0}, {27, 3}}, {{54, 3}, {36, 0}, {20, 0}}, {{51, 3}, {34, 0}, {19, 3}}},
			{{{72, 0}, {60, 3}, {46, 3}}, {{73, 3}, {55, 0}, {40, 0}}, {{71, 3}, {54, 3}, {36, 0}}},
		},
		{ // face 8
			{{{64, 0}, {47, 0}, {38, 3}}, {{62, 0}, {43, 3}, {29, 3}}, {{58, 3}, {42, 3}, {26, 3}}},
			{{{84, 0}, {69, 0}, {51, 3}}, {{82, 3}, {64, 0}, {47, 0}}, {{76, 3}, {62, 0}, {43, 3}}},
			{{{97, 0}, {89, 3}, {71, 3}}, {{98, 3}, {84, 0}, {69, 0}}, {{96, 3}, {82, 3}, {64, 0}}},
		},
		{ // face 9
			{{{75, 0}, {65, 0}, {58, 3}}, {{61, 0}, {53, 3}, {44, 3}}, {{49, 3}, {41, 3}, {31, 3}}},
			{{{94, 0}, {86, 0}, {76, 3}}, {{81, 3}, {75, 0}, {65, 0}}, {{66, 3}, {61, 0}, {53, 3}}},
			{{{107, 0}, {104, 3}, {96, 3}}, {{101, 3}, {94, 0}, {86, 0}}, {{85, 3}, {81, 3}, {75, 0}}},
		},
		{ // face 10
			{{{57, 0}, {59, 0}, {63, 3}}, {{74, 0}, {78, 3}, {79, 3}}, {{83, 3}, {92, 3}, {95, 3}}},
			{{{37, 0}, {39, 3}, {45, 3}}, {{52, 0}, {57, 0}, {59, 0}}, {{70, 3}, {74, 0}, {78, 3}}},
			{{{24, 0}, {23, 3}, {25, 3}}, {{32, 3}, {37, 0}, {39, 3}}, {{50, 3}, {52, 0}, {57, 0}}},
		},
		{ // face 11
			{{{46, 0}, {60, 0}, {72, 3}}, {{56, 0}, {68, 3}, {80, 3}}, {{63, 3}, {77, 3}, {90, 3}}},
			{{{27, 0}, {40, 3}, {55, 3}}, {{35, 0}, {46, 0}, {60, 0}}, {{45, 3}, {56, 0}, {68, 3}}},
			{{{14, 0}, {20, 3}, {36, 3}}, {{17, 3}, {27, 0}, {40, 3}}, {{25, 3}, {35, 0}, {46, 0}}},
		},
		{ // face 12
			{{{71, 0}, {89, 0}, {97, 3}}, {{73, 0}, {91, 3}, {103, 3}}, {{72, 3}, {88, 3}, {105, 3}}},
			{{{51, 0}, {69, 3}, {84, 3}}, {{54, 0}, {71, 0}, {89, 0}}, {{55, 3}, {73, 0}, {91, 3}}},
			{{{38, 0}, {47, 3}, {64, 3}}, {{34, 3}, {51, 0}, {69, 3}}, {{36, 3}, {54, 0}, {71, 0}}},
		},
		{ // face 13
			{{{96, 0}, {104, 0}, {107, 3}}, {{98, 0}, {110, 3}, {115, 3}}, {{97, 3}, {111, 3}, {119, 3}}},
			{{{76, 0}, {86, 3}, {94, 3}}, {{82, 0}, {96, 0}, {104, 0}}, {{84, 3}, {98, 0}, {110, 3}}},
			{{{58, 0}, {65, 3}, {75, 3}}, {{62, 3}, {76, 0}, {86, 3}}, {{64, 3}, {82, 0}, {96, 0}}},
		},
		{ // face 14
			{{{85, 0}, {87, 0}, {83, 3}}, {{101, 0}, {102, 3}, {100, 3}}, {{107, 3}, {112, 3}, {114, 3}}},
			{{{66, 0}, {67, 3}, {70, 3}}, {{81, 0}, {85, 0}, {87, 0}}, {{94, 3}, {101, 0}, {102, 3}}},
			{{{49, 0}, {48, 3}, {50, 3}}, {{61, 3}, {66, 0}, {67, 3}}, {{75, 3}, {81, 0}, {85, 0}}},
		},
		{ // face 15
			{{{95, 0}, {92, 0}, {83, 0}}, {{79, 0}, {78, 0}, {74, 3}}, {{63, 1}, {59, 3}, {57, 3}}},
			{{{109, 0}, {108, 0}, {100, 5}}, {{93, 1}, {95, 0}, {92, 0}}, {{77, 1}, {79, 0}, {78, 0}}},
			{{{117, 4}, {118, 5}, {114, 5}}, {{106, 1}, {109, 0}, {108, 0}}, {{90, 1}, {93, 1}, {95, 0}}},
		},
		{ // face 16
			{{{90, 0}, {77, 0}, {63, 0}}, {{80, 0}, {68, 0}, {56, 3}}, {{72, 1}, {60, 3}, {46, 3}}},
			{{{106, 0}, {93, 0}, {79, 5}}, {{99, 1}, {90, 0}, {77, 0}}, {{88, 1}, {80, 0}, {68, 0}}},
			{{{117, 3}, {109, 5}, {95, 5}}, {{113, 1}, {106, 0}, {93, 0}}, {{105, 1}, {99, 1}, {90, 0}}},
		},
		{ // face 17
			{{{105, 0}, {88, 0}, {72, 0}}, {{103, 0}, {91, 0}, {73, 3}}, {{97, 1}, {89, 3}, {71, 3}}},
			{{{113, 0}, {99, 0}, {80, 5}}, {{116, 1}, {105, 0}, {88, 0}}, {{111, 1}, {103, 0}, {91, 0}}},
			{{{117, 2}, {106, 5}, {90, 5}}, {{121, 1}, {113, 0}, {99, 0}}, {{119, 1}, {116, 1}, {105, 0}}},
		},
		{ // face 18
			{{{119, 0}, {111, 0}, {97, 0}}, {{115, 0}, {110, 0}, {98, 3}}, {{107, 1}, {104, 3}, {96, 3}}},
			{{{121, 0}, {116, 0}, {103, 5}}, {{120, 1}, {119, 0}, {111, 0}}, {{112, 1}, {115, 0}, {110, 0}}},
			{{{117, 1}, {113, 5}, {105, 5}}, {{118, 1}, {121, 0}, {116, 0}}, {{114, 1}, {120, 1}, {119, 0}}},
		},
		{ // face 19
			{{{114, 0}, {112, 0}, {107, 0}}, {{100, 0}, {102, 0}, {101, 3}}, {{83, 1}, {87, 3}, {85, 3}}},
			{{{118, 0}, {120, 0}, {115, 5}}, {{108, 1}, {114, 0}, {112, 0}}, {{92, 1}, {100, 0}, {102, 0}}},
			{{{117, 0}, {121, 5}, {119, 5}}, {{109, 1}, {118, 0}, {120, 0}}, {{95, 1}, {108, 1}, {114, 0}}},
		},
	}

	// unitVecs holds the IJK unit vector for each H3 digit direction, used to step
	// from a cell to its neighbor in that direction.
	unitVecs = [7]coordIJK{
		{0, 0, 0}, // center
		{0, 0, 1}, // k axis
		{0, 1, 0}, // j axis
		{0, 1, 1}, // jk axis
		{1, 0, 0}, // i axis
		{1, 0, 1}, // ik axis
		{1, 1, 0}, // ij axis
	}

	// faceNeighbors describes, for each icosahedron face, how to transform an IJK
	// coordinate into the adjacent face across each of the three axis-pair edges
	// (plus the identity transform for the face itself at index 0). It is indexed by
	// [face][dir] where dir is one of dirIJ, dirKI, dirJK.
	faceNeighbors = [NumIcosaFaces][4]faceOrientIJK{
		{ // face 0
			{0, coordIJK{0, 0, 0}, 0}, // central face
			{4, coordIJK{2, 0, 2}, 1}, // ij quadrant
			{1, coordIJK{2, 2, 0}, 5}, // ki quadrant
			{5, coordIJK{0, 2, 2}, 3}, // jk quadrant
		},
		{ // face 1
			{1, coordIJK{0, 0, 0}, 0},
			{0, coordIJK{2, 0, 2}, 1},
			{2, coordIJK{2, 2, 0}, 5},
			{6, coordIJK{0, 2, 2}, 3},
		},
		{ // face 2
			{2, coordIJK{0, 0, 0}, 0},
			{1, coordIJK{2, 0, 2}, 1},
			{3, coordIJK{2, 2, 0}, 5},
			{7, coordIJK{0, 2, 2}, 3},
		},
		{ // face 3
			{3, coordIJK{0, 0, 0}, 0},
			{2, coordIJK{2, 0, 2}, 1},
			{4, coordIJK{2, 2, 0}, 5},
			{8, coordIJK{0, 2, 2}, 3},
		},
		{ // face 4
			{4, coordIJK{0, 0, 0}, 0},
			{3, coordIJK{2, 0, 2}, 1},
			{0, coordIJK{2, 2, 0}, 5},
			{9, coordIJK{0, 2, 2}, 3},
		},
		{ // face 5
			{5, coordIJK{0, 0, 0}, 0},
			{10, coordIJK{2, 2, 0}, 3},
			{14, coordIJK{2, 0, 2}, 3},
			{0, coordIJK{0, 2, 2}, 3},
		},
		{ // face 6
			{6, coordIJK{0, 0, 0}, 0},
			{11, coordIJK{2, 2, 0}, 3},
			{10, coordIJK{2, 0, 2}, 3},
			{1, coordIJK{0, 2, 2}, 3},
		},
		{ // face 7
			{7, coordIJK{0, 0, 0}, 0},
			{12, coordIJK{2, 2, 0}, 3},
			{11, coordIJK{2, 0, 2}, 3},
			{2, coordIJK{0, 2, 2}, 3},
		},
		{ // face 8
			{8, coordIJK{0, 0, 0}, 0},
			{13, coordIJK{2, 2, 0}, 3},
			{12, coordIJK{2, 0, 2}, 3},
			{3, coordIJK{0, 2, 2}, 3},
		},
		{ // face 9
			{9, coordIJK{0, 0, 0}, 0},
			{14, coordIJK{2, 2, 0}, 3},
			{13, coordIJK{2, 0, 2}, 3},
			{4, coordIJK{0, 2, 2}, 3},
		},
		{ // face 10
			{10, coordIJK{0, 0, 0}, 0},
			{5, coordIJK{2, 2, 0}, 3},
			{6, coordIJK{2, 0, 2}, 3},
			{15, coordIJK{0, 2, 2}, 3},
		},
		{ // face 11
			{11, coordIJK{0, 0, 0}, 0},
			{6, coordIJK{2, 2, 0}, 3},
			{7, coordIJK{2, 0, 2}, 3},
			{16, coordIJK{0, 2, 2}, 3},
		},
		{ // face 12
			{12, coordIJK{0, 0, 0}, 0},
			{7, coordIJK{2, 2, 0}, 3},
			{8, coordIJK{2, 0, 2}, 3},
			{17, coordIJK{0, 2, 2}, 3},
		},
		{ // face 13
			{13, coordIJK{0, 0, 0}, 0},
			{8, coordIJK{2, 2, 0}, 3},
			{9, coordIJK{2, 0, 2}, 3},
			{18, coordIJK{0, 2, 2}, 3},
		},
		{ // face 14
			{14, coordIJK{0, 0, 0}, 0},
			{9, coordIJK{2, 2, 0}, 3},
			{5, coordIJK{2, 0, 2}, 3},
			{19, coordIJK{0, 2, 2}, 3},
		},
		{ // face 15
			{15, coordIJK{0, 0, 0}, 0},
			{16, coordIJK{2, 0, 2}, 1},
			{19, coordIJK{2, 2, 0}, 5},
			{10, coordIJK{0, 2, 2}, 3},
		},
		{ // face 16
			{16, coordIJK{0, 0, 0}, 0},
			{17, coordIJK{2, 0, 2}, 1},
			{15, coordIJK{2, 2, 0}, 5},
			{11, coordIJK{0, 2, 2}, 3},
		},
		{ // face 17
			{17, coordIJK{0, 0, 0}, 0},
			{18, coordIJK{2, 0, 2}, 1},
			{16, coordIJK{2, 2, 0}, 5},
			{12, coordIJK{0, 2, 2}, 3},
		},
		{ // face 18
			{18, coordIJK{0, 0, 0}, 0},
			{19, coordIJK{2, 0, 2}, 1},
			{17, coordIJK{2, 2, 0}, 5},
			{13, coordIJK{0, 2, 2}, 3},
		},
		{ // face 19
			{19, coordIJK{0, 0, 0}, 0},
			{15, coordIJK{2, 0, 2}, 1},
			{18, coordIJK{2, 2, 0}, 5},
			{14, coordIJK{0, 2, 2}, 3},
		},
	}

	// adjacentFaceDir gives the direction (dirIJ/dirKI/dirJK) from the origin face
	// to the destination face, in the origin face's coordinate system, 0 if they
	// are the same face, or -1 if the faces are not adjacent.
	adjacentFaceDir = [NumIcosaFaces][NumIcosaFaces]int{
		{0, dirKI, -1, -1, dirIJ, dirJK, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // face 0
		{dirIJ, 0, dirKI, -1, -1, -1, dirJK, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // face 1
		{-1, dirIJ, 0, dirKI, -1, -1, -1, dirJK, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // face 2
		{-1, -1, dirIJ, 0, dirKI, -1, -1, -1, dirJK, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // face 3
		{dirKI, -1, -1, dirIJ, 0, -1, -1, -1, -1, dirJK, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // face 4
		{dirJK, -1, -1, -1, -1, 0, -1, -1, -1, -1, dirIJ, -1, -1, -1, dirKI, -1, -1, -1, -1, -1}, // face 5
		{-1, dirJK, -1, -1, -1, -1, 0, -1, -1, -1, dirKI, dirIJ, -1, -1, -1, -1, -1, -1, -1, -1}, // face 6
		{-1, -1, dirJK, -1, -1, -1, -1, 0, -1, -1, -1, dirKI, dirIJ, -1, -1, -1, -1, -1, -1, -1}, // face 7
		{-1, -1, -1, dirJK, -1, -1, -1, -1, 0, -1, -1, -1, dirKI, dirIJ, -1, -1, -1, -1, -1, -1}, // face 8
		{-1, -1, -1, -1, dirJK, -1, -1, -1, -1, 0, -1, -1, -1, dirKI, dirIJ, -1, -1, -1, -1, -1}, // face 9
		{-1, -1, -1, -1, -1, dirIJ, dirKI, -1, -1, -1, 0, -1, -1, -1, -1, dirJK, -1, -1, -1, -1}, // face 10
		{-1, -1, -1, -1, -1, -1, dirIJ, dirKI, -1, -1, -1, 0, -1, -1, -1, -1, dirJK, -1, -1, -1}, // face 11
		{-1, -1, -1, -1, -1, -1, -1, dirIJ, dirKI, -1, -1, -1, 0, -1, -1, -1, -1, dirJK, -1, -1}, // face 12
		{-1, -1, -1, -1, -1, -1, -1, -1, dirIJ, dirKI, -1, -1, -1, 0, -1, -1, -1, -1, dirJK, -1}, // face 13
		{-1, -1, -1, -1, -1, dirKI, -1, -1, -1, dirIJ, -1, -1, -1, -1, 0, -1, -1, -1, -1, dirJK}, // face 14
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, dirJK, -1, -1, -1, -1, 0, dirIJ, -1, -1, dirKI}, // face 15
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, dirJK, -1, -1, -1, dirKI, 0, dirIJ, -1, -1}, // face 16
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, dirJK, -1, -1, -1, dirKI, 0, dirIJ, -1}, // face 17
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, dirJK, -1, -1, -1, dirKI, 0, dirIJ}, // face 18
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, dirJK, dirIJ, -1, -1, dirKI, 0}, // face 19
	}

	// maxDimByCIIres is the maximum IJK dimension value, by Class II resolution,
	// used to detect overage past a face edge. Odd (Class III) entries are -1
	// because the overage check operates on Class II grids only.
	maxDimByCIIres = [MaxResolution + 2]int{
		2, -1, 14, -1, 98, -1, 686, -1, 4802, -1, 33614, -1, 235298, -1, 1647086, -1, 11529602,
	}

	// unitScaleByCIIres is the unit-scale distance, by Class II resolution, used to
	// translate IJK coordinates onto an adjacent face. Odd entries are -1 for the
	// same reason as maxDimByCIIres.
	unitScaleByCIIres = [MaxResolution + 2]int{
		1, -1, 7, -1, 49, -1, 343, -1, 2401, -1, 16807, -1, 117649, -1, 823543, -1, 5764801,
	}

	// baseCellHomeFijk maps each base cell to its "home" face and the normalized IJK
	// coordinates of its center on that face — the starting point for decoding a
	// cell back into a face-centered coordinate.
	baseCellHomeFijk = [NumBaseCells]faceIJK{
		{1, coordIJK{1, 0, 0}},  // base cell 0
		{2, coordIJK{1, 1, 0}},  // base cell 1
		{1, coordIJK{0, 0, 0}},  // base cell 2
		{2, coordIJK{1, 0, 0}},  // base cell 3
		{0, coordIJK{2, 0, 0}},  // base cell 4
		{1, coordIJK{1, 1, 0}},  // base cell 5
		{1, coordIJK{0, 0, 1}},  // base cell 6
		{2, coordIJK{0, 0, 0}},  // base cell 7
		{0, coordIJK{1, 0, 0}},  // base cell 8
		{2, coordIJK{0, 1, 0}},  // base cell 9
		{1, coordIJK{0, 1, 0}},  // base cell 10
		{1, coordIJK{0, 1, 1}},  // base cell 11
		{3, coordIJK{1, 0, 0}},  // base cell 12
		{3, coordIJK{1, 1, 0}},  // base cell 13
		{11, coordIJK{2, 0, 0}}, // base cell 14
		{4, coordIJK{1, 0, 0}},  // base cell 15
		{0, coordIJK{0, 0, 0}},  // base cell 16
		{6, coordIJK{0, 1, 0}},  // base cell 17
		{0, coordIJK{0, 0, 1}},  // base cell 18
		{2, coordIJK{0, 1, 1}},  // base cell 19
		{7, coordIJK{0, 0, 1}},  // base cell 20
		{2, coordIJK{0, 0, 1}},  // base cell 21
		{0, coordIJK{1, 1, 0}},  // base cell 22
		{6, coordIJK{0, 0, 1}},  // base cell 23
		{10, coordIJK{2, 0, 0}}, // base cell 24
		{6, coordIJK{0, 0, 0}},  // base cell 25
		{3, coordIJK{0, 0, 0}},  // base cell 26
		{11, coordIJK{1, 0, 0}}, // base cell 27
		{4, coordIJK{1, 1, 0}},  // base cell 28
		{3, coordIJK{0, 1, 0}},  // base cell 29
		{0, coordIJK{0, 1, 1}},  // base cell 30
		{4, coordIJK{0, 0, 0}},  // base cell 31
		{5, coordIJK{0, 1, 0}},  // base cell 32
		{0, coordIJK{0, 1, 0}},  // base cell 33
		{7, coordIJK{0, 1, 0}},  // base cell 34
		{11, coordIJK{1, 1, 0}}, // base cell 35
		{7, coordIJK{0, 0, 0}},  // base cell 36
		{10, coordIJK{1, 0, 0}}, // base cell 37
		{12, coordIJK{2, 0, 0}}, // base cell 38
		{6, coordIJK{1, 0, 1}},  // base cell 39
		{7, coordIJK{1, 0, 1}},  // base cell 40
		{4, coordIJK{0, 0, 1}},  // base cell 41
		{3, coordIJK{0, 0, 1}},  // base cell 42
		{3, coordIJK{0, 1, 1}},  // base cell 43
		{4, coordIJK{0, 1, 0}},  // base cell 44
		{6, coordIJK{1, 0, 0}},  // base cell 45
		{11, coordIJK{0, 0, 0}}, // base cell 46
		{8, coordIJK{0, 0, 1}},  // base cell 47
		{5, coordIJK{0, 0, 1}},  // base cell 48
		{14, coordIJK{2, 0, 0}}, // base cell 49
		{5, coordIJK{0, 0, 0}},  // base cell 50
		{12, coordIJK{1, 0, 0}}, // base cell 51
		{10, coordIJK{1, 1, 0}}, // base cell 52
		{4, coordIJK{0, 1, 1}},  // base cell 53
		{12, coordIJK{1, 1, 0}}, // base cell 54
		{7, coordIJK{1, 0, 0}},  // base cell 55
		{11, coordIJK{0, 1, 0}}, // base cell 56
		{10, coordIJK{0, 0, 0}}, // base cell 57
		{13, coordIJK{2, 0, 0}}, // base cell 58
		{10, coordIJK{0, 0, 1}}, // base cell 59
		{11, coordIJK{0, 0, 1}}, // base cell 60
		{9, coordIJK{0, 1, 0}},  // base cell 61
		{8, coordIJK{0, 1, 0}},  // base cell 62
		{6, coordIJK{2, 0, 0}},  // base cell 63
		{8, coordIJK{0, 0, 0}},  // base cell 64
		{9, coordIJK{0, 0, 1}},  // base cell 65
		{14, coordIJK{1, 0, 0}}, // base cell 66
		{5, coordIJK{1, 0, 1}},  // base cell 67
		{16, coordIJK{0, 1, 1}}, // base cell 68
		{8, coordIJK{1, 0, 1}},  // base cell 69
		{5, coordIJK{1, 0, 0}},  // base cell 70
		{12, coordIJK{0, 0, 0}}, // base cell 71
		{7, coordIJK{2, 0, 0}},  // base cell 72
		{12, coordIJK{0, 1, 0}}, // base cell 73
		{10, coordIJK{0, 1, 0}}, // base cell 74
		{9, coordIJK{0, 0, 0}},  // base cell 75
		{13, coordIJK{1, 0, 0}}, // base cell 76
		{16, coordIJK{0, 0, 1}}, // base cell 77
		{15, coordIJK{0, 1, 1}}, // base cell 78
		{15, coordIJK{0, 1, 0}}, // base cell 79
		{16, coordIJK{0, 1, 0}}, // base cell 80
		{14, coordIJK{1, 1, 0}}, // base cell 81
		{13, coordIJK{1, 1, 0}}, // base cell 82
		{5, coordIJK{2, 0, 0}},  // base cell 83
		{8, coordIJK{1, 0, 0}},  // base cell 84
		{14, coordIJK{0, 0, 0}}, // base cell 85
		{9, coordIJK{1, 0, 1}},  // base cell 86
		{14, coordIJK{0, 0, 1}}, // base cell 87
		{17, coordIJK{0, 0, 1}}, // base cell 88
		{12, coordIJK{0, 0, 1}}, // base cell 89
		{16, coordIJK{0, 0, 0}}, // base cell 90
		{17, coordIJK{0, 1, 1}}, // base cell 91
		{15, coordIJK{0, 0, 1}}, // base cell 92
		{16, coordIJK{1, 0, 1}}, // base cell 93
		{9, coordIJK{1, 0, 0}},  // base cell 94
		{15, coordIJK{0, 0, 0}}, // base cell 95
		{13, coordIJK{0, 0, 0}}, // base cell 96
		{8, coordIJK{2, 0, 0}},  // base cell 97
		{13, coordIJK{0, 1, 0}}, // base cell 98
		{17, coordIJK{1, 0, 1}}, // base cell 99
		{19, coordIJK{0, 1, 0}}, // base cell 100
		{14, coordIJK{0, 1, 0}}, // base cell 101
		{19, coordIJK{0, 1, 1}}, // base cell 102
		{17, coordIJK{0, 1, 0}}, // base cell 103
		{13, coordIJK{0, 0, 1}}, // base cell 104
		{17, coordIJK{0, 0, 0}}, // base cell 105
		{16, coordIJK{1, 0, 0}}, // base cell 106
		{9, coordIJK{2, 0, 0}},  // base cell 107
		{15, coordIJK{1, 0, 1}}, // base cell 108
		{15, coordIJK{1, 0, 0}}, // base cell 109
		{18, coordIJK{0, 1, 1}}, // base cell 110
		{18, coordIJK{0, 0, 1}}, // base cell 111
		{19, coordIJK{0, 0, 1}}, // base cell 112
		{17, coordIJK{1, 0, 0}}, // base cell 113
		{19, coordIJK{0, 0, 0}}, // base cell 114
		{18, coordIJK{0, 1, 0}}, // base cell 115
		{18, coordIJK{1, 0, 1}}, // base cell 116
		{19, coordIJK{2, 0, 0}}, // base cell 117
		{19, coordIJK{1, 0, 0}}, // base cell 118
		{18, coordIJK{0, 0, 0}}, // base cell 119
		{19, coordIJK{1, 0, 1}}, // base cell 120
		{18, coordIJK{1, 0, 0}}, // base cell 121
	}

	// baseCellNeighbors[baseCell][dir] is the base cell reached by stepping from
	// baseCell in direction dir, or invalidBaseCell across a pentagon's deleted edge.
	baseCellNeighbors = [NumBaseCells][7]int{
		{0, 1, 5, 2, 4, 3, 8},                       // base cell 0
		{1, 7, 6, 9, 0, 3, 2},                       // base cell 1
		{2, 6, 10, 11, 0, 1, 5},                     // base cell 2
		{3, 13, 1, 7, 4, 12, 0},                     // base cell 3
		{4, invalidBaseCell, 15, 8, 3, 0, 12},       // base cell 4 (pentagon)
		{5, 2, 18, 10, 8, 0, 16},                    // base cell 5
		{6, 14, 11, 17, 1, 9, 2},                    // base cell 6
		{7, 21, 9, 19, 3, 13, 1},                    // base cell 7
		{8, 5, 22, 16, 4, 0, 15},                    // base cell 8
		{9, 19, 14, 20, 1, 7, 6},                    // base cell 9
		{10, 11, 24, 23, 5, 2, 18},                  // base cell 10
		{11, 17, 23, 25, 2, 6, 10},                  // base cell 11
		{12, 28, 13, 26, 4, 15, 3},                  // base cell 12
		{13, 26, 21, 29, 3, 12, 7},                  // base cell 13
		{14, invalidBaseCell, 17, 27, 9, 20, 6},     // base cell 14 (pentagon)
		{15, 22, 28, 31, 4, 8, 12},                  // base cell 15
		{16, 18, 33, 30, 8, 5, 22},                  // base cell 16
		{17, 11, 14, 6, 35, 25, 27},                 // base cell 17
		{18, 24, 30, 32, 5, 10, 16},                 // base cell 18
		{19, 34, 20, 36, 7, 21, 9},                  // base cell 19
		{20, 14, 19, 9, 40, 27, 36},                 // base cell 20
		{21, 38, 19, 34, 13, 29, 7},                 // base cell 21
		{22, 16, 41, 33, 15, 8, 31},                 // base cell 22
		{23, 24, 11, 10, 39, 37, 25},                // base cell 23
		{24, invalidBaseCell, 32, 37, 10, 23, 18},   // base cell 24 (pentagon)
		{25, 23, 17, 11, 45, 39, 35},                // base cell 25
		{26, 42, 29, 43, 12, 28, 13},                // base cell 26
		{27, 40, 35, 46, 14, 20, 17},                // base cell 27
		{28, 31, 42, 44, 12, 15, 26},                // base cell 28
		{29, 43, 38, 47, 13, 26, 21},                // base cell 29
		{30, 32, 48, 50, 16, 18, 33},                // base cell 30
		{31, 41, 44, 53, 15, 22, 28},                // base cell 31
		{32, 30, 24, 18, 52, 50, 37},                // base cell 32
		{33, 30, 49, 48, 22, 16, 41},                // base cell 33
		{34, 19, 38, 21, 54, 36, 51},                // base cell 34
		{35, 46, 45, 56, 17, 27, 25},                // base cell 35
		{36, 20, 34, 19, 55, 40, 54},                // base cell 36
		{37, 39, 52, 57, 24, 23, 32},                // base cell 37
		{38, invalidBaseCell, 34, 51, 29, 47, 21},   // base cell 38 (pentagon)
		{39, 37, 25, 23, 59, 57, 45},                // base cell 39
		{40, 27, 36, 20, 60, 46, 55},                // base cell 40
		{41, 49, 53, 61, 22, 33, 31},                // base cell 41
		{42, 58, 43, 62, 28, 44, 26},                // base cell 42
		{43, 62, 47, 64, 26, 42, 29},                // base cell 43
		{44, 53, 58, 65, 28, 31, 42},                // base cell 44
		{45, 39, 35, 25, 63, 59, 56},                // base cell 45
		{46, 60, 56, 68, 27, 40, 35},                // base cell 46
		{47, 38, 43, 29, 69, 51, 64},                // base cell 47
		{48, 49, 30, 33, 67, 66, 50},                // base cell 48
		{49, invalidBaseCell, 61, 66, 33, 48, 41},   // base cell 49 (pentagon)
		{50, 48, 32, 30, 70, 67, 52},                // base cell 50
		{51, 69, 54, 71, 38, 47, 34},                // base cell 51
		{52, 57, 70, 74, 32, 37, 50},                // base cell 52
		{53, 61, 65, 75, 31, 41, 44},                // base cell 53
		{54, 71, 55, 73, 34, 51, 36},                // base cell 54
		{55, 40, 54, 36, 72, 60, 73},                // base cell 55
		{56, 68, 63, 77, 35, 46, 45},                // base cell 56
		{57, 59, 74, 78, 37, 39, 52},                // base cell 57
		{58, invalidBaseCell, 62, 76, 44, 65, 42},   // base cell 58 (pentagon)
		{59, 63, 78, 79, 39, 45, 57},                // base cell 59
		{60, 72, 68, 80, 40, 55, 46},                // base cell 60
		{61, 53, 49, 41, 81, 75, 66},                // base cell 61
		{62, 43, 58, 42, 82, 64, 76},                // base cell 62
		{63, invalidBaseCell, 56, 45, 79, 59, 77},   // base cell 63 (pentagon)
		{64, 47, 62, 43, 84, 69, 82},                // base cell 64
		{65, 58, 53, 44, 86, 76, 75},                // base cell 65
		{66, 67, 81, 85, 49, 48, 61},                // base cell 66
		{67, 66, 50, 48, 87, 85, 70},                // base cell 67
		{68, 56, 60, 46, 90, 77, 80},                // base cell 68
		{69, 51, 64, 47, 89, 71, 84},                // base cell 69
		{70, 67, 52, 50, 83, 87, 74},                // base cell 70
		{71, 89, 73, 91, 51, 69, 54},                // base cell 71
		{72, invalidBaseCell, 73, 55, 80, 60, 88},   // base cell 72 (pentagon)
		{73, 91, 72, 88, 54, 71, 55},                // base cell 73
		{74, 78, 83, 92, 52, 57, 70},                // base cell 74
		{75, 65, 61, 53, 94, 86, 81},                // base cell 75
		{76, 86, 82, 96, 58, 65, 62},                // base cell 76
		{77, 63, 68, 56, 93, 79, 90},                // base cell 77
		{78, 74, 59, 57, 95, 92, 79},                // base cell 78
		{79, 78, 63, 59, 93, 95, 77},                // base cell 79
		{80, 68, 72, 60, 99, 90, 88},                // base cell 80
		{81, 85, 94, 101, 61, 66, 75},               // base cell 81
		{82, 96, 84, 98, 62, 76, 64},                // base cell 82
		{83, invalidBaseCell, 74, 70, 100, 87, 92},  // base cell 83 (pentagon)
		{84, 69, 82, 64, 97, 89, 98},                // base cell 84
		{85, 87, 101, 102, 66, 67, 81},              // base cell 85
		{86, 76, 75, 65, 104, 96, 94},               // base cell 86
		{87, 83, 102, 100, 67, 70, 85},              // base cell 87
		{88, 72, 91, 73, 99, 80, 105},               // base cell 88
		{89, 97, 91, 103, 69, 84, 71},               // base cell 89
		{90, 77, 80, 68, 106, 93, 99},               // base cell 90
		{91, 73, 89, 71, 105, 88, 103},              // base cell 91
		{92, 83, 78, 74, 108, 100, 95},              // base cell 92
		{93, 79, 90, 77, 109, 95, 106},              // base cell 93
		{94, 86, 81, 75, 107, 104, 101},             // base cell 94
		{95, 92, 79, 78, 109, 108, 93},              // base cell 95
		{96, 104, 98, 110, 76, 86, 82},              // base cell 96
		{97, invalidBaseCell, 98, 84, 103, 89, 111}, // base cell 97 (pentagon)
		{98, 110, 97, 111, 82, 96, 84},              // base cell 98
		{99, 80, 105, 88, 106, 90, 113},             // base cell 99
		{100, 102, 83, 87, 108, 114, 92},            // base cell 100
		{101, 102, 107, 112, 81, 85, 94},            // base cell 101
		{102, 101, 87, 85, 114, 112, 100},           // base cell 102
		{103, 91, 97, 89, 116, 105, 111},            // base cell 103
		{104, 107, 110, 115, 86, 94, 96},            // base cell 104
		{105, 88, 103, 91, 113, 99, 116},            // base cell 105
		{106, 93, 99, 90, 117, 109, 113},            // base cell 106
		{107, invalidBaseCell, 101, 94, 115, 104,
			112}, // base cell 107 (pentagon)
		{108, 100, 95, 92, 118, 114, 109},   // base cell 108
		{109, 108, 93, 95, 117, 118, 106},   // base cell 109
		{110, 98, 104, 96, 119, 111, 115},   // base cell 110
		{111, 97, 110, 98, 116, 103, 119},   // base cell 111
		{112, 107, 102, 101, 120, 115, 114}, // base cell 112
		{113, 99, 116, 105, 117, 106, 121},  // base cell 113
		{114, 112, 100, 102, 118, 120, 108}, // base cell 114
		{115, 110, 107, 104, 120, 119, 112}, // base cell 115
		{116, 103, 119, 111, 113, 105, 121}, // base cell 116
		{117, invalidBaseCell, 109, 118, 113, 121,
			106}, // base cell 117 (pentagon)
		{118, 120, 108, 114, 117, 121, 109}, // base cell 118
		{119, 111, 115, 110, 121, 116, 120}, // base cell 119
		{120, 115, 114, 112, 121, 119, 118}, // base cell 120
		{121, 116, 120, 119, 117, 113, 118}, // base cell 121
	}

	// baseCellNeighbor60CCWRots[baseCell][dir] is the number of 60° ccw rotations
	// to apply when stepping from baseCell in direction dir.
	baseCellNeighbor60CCWRots = [NumBaseCells][7]int{
		{0, 5, 0, 0, 1, 5, 1},  // base cell 0
		{0, 0, 1, 0, 1, 0, 1},  // base cell 1
		{0, 0, 0, 0, 0, 5, 0},  // base cell 2
		{0, 5, 0, 0, 2, 5, 1},  // base cell 3
		{0, -1, 1, 0, 3, 4, 2}, // base cell 4 (pentagon)
		{0, 0, 1, 0, 1, 0, 1},  // base cell 5
		{0, 0, 0, 3, 5, 5, 0},  // base cell 6
		{0, 0, 0, 0, 0, 5, 0},  // base cell 7
		{0, 5, 0, 0, 0, 5, 1},  // base cell 8
		{0, 0, 1, 3, 0, 0, 1},  // base cell 9
		{0, 0, 1, 3, 0, 0, 1},  // base cell 10
		{0, 3, 3, 3, 0, 0, 0},  // base cell 11
		{0, 5, 0, 0, 3, 5, 1},  // base cell 12
		{0, 0, 1, 0, 1, 0, 1},  // base cell 13
		{0, -1, 3, 0, 5, 2, 0}, // base cell 14 (pentagon)
		{0, 5, 0, 0, 4, 5, 1},  // base cell 15
		{0, 0, 0, 0, 0, 5, 0},  // base cell 16
		{0, 3, 3, 3, 3, 0, 3},  // base cell 17
		{0, 0, 0, 3, 5, 5, 0},  // base cell 18
		{0, 3, 3, 3, 0, 0, 0},  // base cell 19
		{0, 3, 3, 3, 0, 3, 0},  // base cell 20
		{0, 0, 0, 3, 5, 5, 0},  // base cell 21
		{0, 0, 1, 0, 1, 0, 1},  // base cell 22
		{0, 3, 3, 3, 0, 3, 0},  // base cell 23
		{0, -1, 3, 0, 5, 2, 0}, // base cell 24 (pentagon)
		{0, 0, 0, 3, 0, 0, 3},  // base cell 25
		{0, 0, 0, 0, 0, 5, 0},  // base cell 26
		{0, 3, 0, 0, 0, 3, 3},  // base cell 27
		{0, 0, 1, 0, 1, 0, 1},  // base cell 28
		{0, 0, 1, 3, 0, 0, 1},  // base cell 29
		{0, 3, 3, 3, 0, 0, 0},  // base cell 30
		{0, 0, 0, 0, 0, 5, 0},  // base cell 31
		{0, 3, 3, 3, 3, 0, 3},  // base cell 32
		{0, 0, 1, 3, 0, 0, 1},  // base cell 33
		{0, 3, 3, 3, 3, 0, 3},  // base cell 34
		{0, 0, 3, 0, 3, 0, 3},  // base cell 35
		{0, 0, 0, 3, 0, 0, 3},  // base cell 36
		{0, 3, 0, 0, 0, 3, 3},  // base cell 37
		{0, -1, 3, 0, 5, 2, 0}, // base cell 38 (pentagon)
		{0, 3, 0, 0, 3, 3, 0},  // base cell 39
		{0, 3, 0, 0, 3, 3, 0},  // base cell 40
		{0, 0, 0, 3, 5, 5, 0},  // base cell 41
		{0, 0, 0, 3, 5, 5, 0},  // base cell 42
		{0, 3, 3, 3, 0, 0, 0},  // base cell 43
		{0, 0, 1, 3, 0, 0, 1},  // base cell 44
		{0, 0, 3, 0, 0, 3, 3},  // base cell 45
		{0, 0, 0, 3, 0, 3, 0},  // base cell 46
		{0, 3, 3, 3, 0, 3, 0},  // base cell 47
		{0, 3, 3, 3, 0, 3, 0},  // base cell 48
		{0, -1, 3, 0, 5, 2, 0}, // base cell 49 (pentagon)
		{0, 0, 0, 3, 0, 0, 3},  // base cell 50
		{0, 3, 0, 0, 0, 3, 3},  // base cell 51
		{0, 0, 3, 0, 3, 0, 3},  // base cell 52
		{0, 3, 3, 3, 0, 0, 0},  // base cell 53
		{0, 0, 3, 0, 3, 0, 3},  // base cell 54
		{0, 0, 3, 0, 0, 3, 3},  // base cell 55
		{0, 3, 3, 3, 0, 0, 3},  // base cell 56
		{0, 0, 0, 3, 0, 3, 0},  // base cell 57
		{0, -1, 3, 0, 5, 2, 0}, // base cell 58 (pentagon)
		{0, 3, 3, 3, 3, 3, 0},  // base cell 59
		{0, 3, 3, 3, 3, 3, 0},  // base cell 60
		{0, 3, 3, 3, 3, 0, 3},  // base cell 61
		{0, 3, 3, 3, 3, 0, 3},  // base cell 62
		{0, -1, 3, 0, 5, 2, 0}, // base cell 63 (pentagon)
		{0, 0, 0, 3, 0, 0, 3},  // base cell 64
		{0, 3, 3, 3, 0, 3, 0},  // base cell 65
		{0, 3, 0, 0, 0, 3, 3},  // base cell 66
		{0, 3, 0, 0, 3, 3, 0},  // base cell 67
		{0, 3, 3, 3, 0, 0, 0},  // base cell 68
		{0, 3, 0, 0, 3, 3, 0},  // base cell 69
		{0, 0, 3, 0, 0, 3, 3},  // base cell 70
		{0, 0, 0, 3, 0, 3, 0},  // base cell 71
		{0, -1, 3, 0, 5, 2, 0}, // base cell 72 (pentagon)
		{0, 3, 3, 3, 0, 0, 3},  // base cell 73
		{0, 3, 3, 3, 0, 0, 3},  // base cell 74
		{0, 0, 0, 3, 0, 0, 3},  // base cell 75
		{0, 3, 0, 0, 0, 3, 3},  // base cell 76
		{0, 0, 0, 3, 0, 5, 0},  // base cell 77
		{0, 3, 3, 3, 0, 0, 0},  // base cell 78
		{0, 0, 1, 3, 1, 0, 1},  // base cell 79
		{0, 0, 1, 3, 1, 0, 1},  // base cell 80
		{0, 0, 3, 0, 3, 0, 3},  // base cell 81
		{0, 0, 3, 0, 3, 0, 3},  // base cell 82
		{0, -1, 3, 0, 5, 2, 0}, // base cell 83 (pentagon)
		{0, 0, 3, 0, 0, 3, 3},  // base cell 84
		{0, 0, 0, 3, 0, 3, 0},  // base cell 85
		{0, 3, 0, 0, 3, 3, 0},  // base cell 86
		{0, 3, 3, 3, 3, 3, 0},  // base cell 87
		{0, 0, 0, 3, 0, 5, 0},  // base cell 88
		{0, 3, 3, 3, 3, 3, 0},  // base cell 89
		{0, 0, 0, 0, 0, 0, 1},  // base cell 90
		{0, 3, 3, 3, 0, 0, 0},  // base cell 91
		{0, 0, 0, 3, 0, 5, 0},  // base cell 92
		{0, 5, 0, 0, 5, 5, 0},  // base cell 93
		{0, 0, 3, 0, 0, 3, 3},  // base cell 94
		{0, 0, 0, 0, 0, 0, 1},  // base cell 95
		{0, 0, 0, 3, 0, 3, 0},  // base cell 96
		{0, -1, 3, 0, 5, 2, 0}, // base cell 97 (pentagon)
		{0, 3, 3, 3, 0, 0, 3},  // base cell 98
		{0, 5, 0, 0, 5, 5, 0},  // base cell 99
		{0, 0, 1, 3, 1, 0, 1},  // base cell 100
		{0, 3, 3, 3, 0, 0, 3},  // base cell 101
		{0, 3, 3, 3, 0, 0, 0},  // base cell 102
		{0, 0, 1, 3, 1, 0, 1},  // base cell 103
		{0, 3, 3, 3, 3, 3, 0},  // base cell 104
		{0, 0, 0, 0, 0, 0, 1},  // base cell 105
		{0, 0, 1, 0, 3, 5, 1},  // base cell 106
		{0, -1, 3, 0, 5, 2, 0}, // base cell 107 (pentagon)
		{0, 5, 0, 0, 5, 5, 0},  // base cell 108
		{0, 0, 1, 0, 4, 5, 1},  // base cell 109
		{0, 3, 3, 3, 0, 0, 0},  // base cell 110
		{0, 0, 0, 3, 0, 5, 0},  // base cell 111
		{0, 0, 0, 3, 0, 5, 0},  // base cell 112
		{0, 0, 1, 0, 2, 5, 1},  // base cell 113
		{0, 0, 0, 0, 0, 0, 1},  // base cell 114
		{0, 0, 1, 3, 1, 0, 1},  // base cell 115
		{0, 5, 0, 0, 5, 5, 0},  // base cell 116
		{0, -1, 1, 0, 3, 4, 2}, // base cell 117 (pentagon)
		{0, 0, 1, 0, 0, 5, 1},  // base cell 118
		{0, 0, 0, 0, 0, 0, 1},  // base cell 119
		{0, 5, 0, 0, 5, 5, 0},  // base cell 120
		{0, 0, 1, 0, 1, 5, 1},  // base cell 121
	}

	// pentagonRotations maps the base-cell direction (or leading digit) to the
	// number of 60° clockwise index rotations needed when unfolding a pentagon,
	// indexed by [origin leading digit][direction]. A -1 marks an invalid k-axis
	// combination.
	pentagonRotations = [7][7]int{
		{0, -1, 0, 0, 0, 0, 0},
		{-1, -1, -1, -1, -1, -1, -1},
		{0, -1, 0, 0, 0, 1, 0},
		{0, -1, 0, 0, 1, 1, 0},
		{0, -1, 0, 5, 0, 0, 0},
		{0, -1, 5, 5, 0, 0, 0},
		{0, -1, 0, 0, 0, 0, 0},
	}

	// pentagonRotationsReverse reverses the rotation pentagonRotations introduces
	// when the origin is on a pentagon, indexed by [leading digit][direction].
	pentagonRotationsReverse = [7][7]int{
		{0, 0, 0, 0, 0, 0, 0},
		{-1, -1, -1, -1, -1, -1, -1},
		{0, 1, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 1, 0},
		{0, 5, 0, 0, 0, 0, 0},
		{0, 5, 0, 5, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0},
	}

	// pentagonRotationsReverseNonpolar reverses the pentagon rotation when the index
	// is on a non-polar pentagon and the origin is not, indexed by
	// [reverse direction][leading digit].
	pentagonRotationsReverseNonpolar = [7][7]int{
		{0, 0, 0, 0, 0, 0, 0},
		{-1, -1, -1, -1, -1, -1, -1},
		{0, 1, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 1, 0},
		{0, 5, 0, 0, 0, 0, 0},
		{0, 1, 0, 5, 1, 1, 0},
		{0, 0, 0, 0, 0, 0, 0},
	}

	// pentagonRotationsReversePolar reverses the pentagon rotation when the index is
	// on a polar pentagon and the origin is not, indexed by
	// [reverse direction][leading digit].
	pentagonRotationsReversePolar = [7][7]int{
		{0, 0, 0, 0, 0, 0, 0},
		{-1, -1, -1, -1, -1, -1, -1},
		{0, 1, 1, 1, 1, 1, 1},
		{0, 1, 0, 0, 0, 1, 0},
		{0, 1, 0, 0, 1, 1, 1},
		{0, 1, 0, 5, 1, 1, 0},
		{0, 1, 1, 0, 1, 1, 1},
	}

	// failedDirections marks the direction pairs that cannot be unfolded across a
	// pentagon (any unfolding across more than one icosahedron face), indexed by
	// [origin direction][index direction].
	failedDirections = [7][7]bool{
		{false, false, false, false, false, false, false},
		{false, false, false, false, false, false, false},
		{false, false, false, false, true, true, false},
		{false, false, false, false, true, false, true},
		{false, false, true, true, false, false, false},
		{false, false, true, false, false, false, true},
		{false, false, false, true, false, true, false},
	}
)
