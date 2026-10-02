# H3 conformance suite

A language-neutral description of what the H3 reference implementation
returns for a fixed set of inputs, so that a port can prove it matches without
building the C library. This directory holds the format specification (this
file), the Go runner that checks `x/h3go` against it, the generator that
produces it from the C library, and the generated files under `testdata/`.

Records are newline-delimited JSON, one per subject, with every key present
on every line and errors spelled as `H3Error` names, so a file can be read
by any language without a schema and a failing line can be quoted as it is.

## Layout

```
manifest.json
inspection/cells.jsonl
hierarchy/cells.jsonl
traversal/cells.jsonl
edges/cells.jsonl
vertexes/cells.jsonl
localij/pairs.jsonl
sets/sets.jsonl
regions/polygons.jsonl
digests/resolutions.jsonl
digests/baseCells.jsonl
```

`manifest.json` is the only file a consumer opens by name. Every record file
is listed in it with its line count and SHA-256, and a consumer must refuse a
file whose hash does not match. Record files are grouped in one directory per
function family; a group directory may hold several files.

### Manifest

```json
{
  "format": "h3-conformance/1",
  "h3Version": "4.5.0",
  "generator": {"name": "...", "seed": 20260929},
  "tolerances": {"angularDeg": 1e-11, "relative": 1e-9},
  "files": {"inspection/cells.jsonl": {"records": 1885, "sha256": "..."}}
}
```

| Key | Meaning |
|---|---|
| `format` | The specification version. A consumer that does not recognize it must not run the suite. |
| `h3Version` | The H3 release the expected values were produced by, without a leading `v`. |
| `generator.name` | The program that produced the files. |
| `generator.seed` | The seed its sampler ran from. Two generators that follow the sampling procedure below with the same seed produce byte-identical record files. |
| `tolerances` | The comparison bounds for floating-point outputs, described under Tolerances. They apply to every file in the suite. |
| `files` | Slash-separated paths relative to the manifest, each with its number of lines and the lowercase hex SHA-256 of its bytes. |

## Record files

A record file is UTF-8 text containing one JSON object per line. Every line,
including the last, ends in a single `\n`. There are no blank lines and no
comments. Generators write keys in the order given in the group definition
and no whitespace outside strings; consumers must not depend on either.

Each line describes one **subject** (an index, a pair of indexes, a point) and
carries the expected output of every function in the group as a sibling key.
Every key of a group is present on every line of that group's files. Nothing
is optional and nothing is inferred from absence.

### Value conventions

- **Indexes** are strings in the form `h3ToString` produces: lowercase
  hexadecimal, no `0x` prefix, no leading zeros. A consumer parses them with
  `stringToH3` and must also check that formatting the parsed value gives the
  same string back.
- **Errors** are objects with a single key: `{"err": "E_CELL_INVALID"}`. The
  value is the `H3Error` enum name, never its number. A key whose function can
  fail holds either the function's output or an error object.
- **Booleans** are JSON `true` and `false`, for functions that return
  `int` used as a boolean in C.
- **Sets** are JSON arrays whose order is not significant. Generators write
  them in ascending order without duplicates; consumers compare them as
  sets. Where the C API returns a sparse array padded with a sentinel (`-1`
  for faces, `0` for indexes), the padding is removed before writing and
  must be ignored when comparing.
- **Sequences** are JSON arrays whose order is significant, for example cell
  boundary vertices or the cells of a grid path. The group definition says
  which arrays are sets and which are sequences.
- **Floating-point** values are JSON numbers with enough digits to round-trip
  a binary64. They are compared using the manifest's tolerances, never for
  equality.
- **Points** are `[latitude, longitude]` pairs in degrees, both as inputs and
  as outputs. A consumer converts to whatever its implementation takes.

### Tolerances

The reference implementation is not bit-reproducible across platforms: the
same C source built for x86-64 and arm64 differs in the last bits of most
coordinate outputs, because of fused multiply-add and libm differences. A
port therefore cannot be asked to match float outputs exactly, and the bound
it is asked to meet has to be one the reference itself meets everywhere.

- **Coordinates** (`cellToLatLng`, `cellToBoundary`, `vertexToLatLng` and any
  other latitude/longitude output) are compared by the **angular distance on
  the sphere** between the expected point and the actual point, in degrees,
  which must not exceed `tolerances.angularDeg`. Angular distance is the
  great-circle separation of the two points, equivalently the angle between
  their unit vectors. It is never compared as separate latitude and longitude
  differences: those are amplified by `1/cos(latitude)` near the poles, so a
  per-coordinate bound that is met at the equator is unmeetable at high
  latitudes and fine resolutions, for the reference as much as for any port.
- **Lengths and areas** (`cellArea*`, `edgeLength*`, `greatCircleDistance*`)
  are compared by relative error, `|actual - expected| / |expected|`, which
  must not exceed `tolerances.relative`. An expected value of exactly zero is
  compared for equality.

Discrete outputs, which is everything else, are compared exactly.

- **Digests** are lowercase hex SHA-256 strings over a stream of text lines,
  one per subject, each terminated by `\n`. They stand in for discrete
  outputs that are too large to store, and are compared exactly. Within a
  line, fields are separated by single spaces: indexes as index strings,
  integers in decimal, booleans as `1` and `0`, errors as `H3Error` names,
  sets in ascending order, and `|` between consecutive sets. Text rather
  than binary so that a failing implementation can dump its stream and
  `diff` it against the reference's.

## Groups

### `inspection`

One line per 64-bit index. The subject may be a valid cell, a directed edge,
a vertex, or an arbitrary bit pattern; every function below is defined for
all of them and the record holds whatever the reference returned.

| Key | Function | Type |
|---|---|---|
| `index` | the subject, as `h3ToString` formats it | index string |
| `res` | `getResolution` | integer |
| `baseCell` | `getBaseCellNumber` | integer |
| `validCell` | `isValidCell` | boolean |
| `validIndex` | `isValidIndex` | boolean |
| `resClassIII` | `isResClassIII` | boolean |
| `pentagon` | `isPentagon` | boolean |
| `faces` | `getIcosahedronFaces`, with `-1` padding removed | set of integers, or error |
| `digits` | `getIndexDigit` for resolutions 1 through 15 | sequence of 15 integers |
| `construct` | `constructCell(res, baseCell, digits[0..res))` | index string, or error |

`digits` always has fifteen entries, including the slots beyond `res`, because
`getIndexDigit` is defined for every resolution and a valid cell must hold `7`
in the unused slots. `construct` reads only the first `res` of them, so its
round trip back to `index` is expected exactly when `validCell` is true, and
its error otherwise tells the consumer which rule the index breaks.

Example:

```json
{"index":"804dfffffffffff","res":0,"baseCell":38,"validCell":true,"validIndex":true,"resClassIII":false,"pentagon":true,"faces":[2,3,7,8,12],"digits":[7,7,7,7,7,7,7,7,7,7,7,7,7,7,7],"construct":"804dfffffffffff"}
{"index":"fffffffffffffff","res":15,"baseCell":127,"validCell":false,"validIndex":false,"resClassIII":true,"pentagon":false,"faces":{"err":"E_CELL_INVALID"},"digits":[7,7,7,7,7,7,7,7,7,7,7,7,7,7,7],"construct":{"err":"E_BASE_CELL_DOMAIN"}}
```

### `hierarchy`

One line per **valid cell**. Resolution bounds are the only source of errors:
a resolution-0 cell has no parents, and a resolution-15 cell has no children.

| Key | Function | Type |
|---|---|---|
| `index` | the subject cell | index string |
| `parents` | `cellToParent(index, r)` for `r` from 0 to `res - 1`, in that order | sequence of index strings |
| `centerChild` | `cellToCenterChild(index, res + 1)` | index string, or error |
| `children` | `cellToChildren(index, res + 1)`, with `0` padding removed | set of index strings, or error |
| `childPos` | `cellToChildPos(index, 0)` | integer, or error |

A consumer must also check the round trip `childPosToCell(childPos, parents[0],
res) == index` (using `index` itself as the parent when `res` is 0) whenever
`childPos` succeeded.

Example:

```json
{"index":"8100bffffffffff","parents":["8001fffffffffff"],"centerChild":"820087fffffffff","children":["820087fffffffff","82008ffffffffff","820097fffffffff","82009ffffffffff","8200a7fffffffff","8200affffffffff","8200b7fffffffff"],"childPos":2}
```

### `traversal`

One line per **valid cell**, paired with a `target` drawn from its 3-disk.
`gridDistance` and `gridPathCells` fail with `E_FAILED` when the pair
straddles a pentagon or is otherwise too far apart in local coordinates;
those rows are the group's error cases.

| Key | Function | Type |
|---|---|---|
| `index` | the subject cell | index string |
| `disk1`, `disk2`, `disk3` | `gridDisk(index, k)` for `k` 1, 2, 3, with `0` padding removed | set of index strings |
| `ring1`, `ring2`, `ring3` | `gridRing(index, k)` for `k` 1, 2, 3, with `0` padding removed | set of index strings |
| `diskDistances2` | `gridDiskDistances(index, 2)`, grouped by distance 0, 1, 2 | sequence of three sets of index strings |
| `target` | the second cell of the pair | index string |
| `distance` | `gridDistance(index, target)` | integer, or error |
| `path` | `gridPathCells(index, target)`, `gridPathCellsSize` entries | sequence of index strings, or error |
| `neighbor` | `areNeighborCells(index, target)` | boolean, or error |

Example:

```json
{"index":"8001fffffffffff","disk1":["8001fffffffffff","8003fffffffffff","8005fffffffffff","8007fffffffffff","8009fffffffffff","800bfffffffffff","8011fffffffffff"],"disk2":["8001fffffffffff","8003fffffffffff","8005fffffffffff","8007fffffffffff","8009fffffffffff","800bfffffffffff","800dfffffffffff","800ffffffffffff","8011fffffffffff","8013fffffffffff","8015fffffffffff","8017fffffffffff","8019fffffffffff","801bfffffffffff","801ffffffffffff","8021fffffffffff","8025fffffffffff","802dfffffffffff"],"disk3":["8001fffffffffff","8003fffffffffff","8005fffffffffff","8007fffffffffff","8009fffffffffff","800bfffffffffff","800dfffffffffff","800ffffffffffff","8011fffffffffff","8013fffffffffff","8015fffffffffff","8017fffffffffff","8019fffffffffff","801bfffffffffff","801dfffffffffff","801ffffffffffff","8021fffffffffff","8023fffffffffff","8025fffffffffff","8027fffffffffff","8029fffffffffff","802bfffffffffff","802dfffffffffff","802ffffffffffff","8031fffffffffff","8033fffffffffff","8035fffffffffff","8039fffffffffff","803bfffffffffff","803dfffffffffff","803ffffffffffff","8041fffffffffff","8043fffffffffff","8053fffffffffff"],"ring1":["8003fffffffffff","8005fffffffffff","8007fffffffffff","8009fffffffffff","800bfffffffffff","8011fffffffffff"],"ring2":["800dfffffffffff","800ffffffffffff","8013fffffffffff","8015fffffffffff","8017fffffffffff","8019fffffffffff","801bfffffffffff","801ffffffffffff","8021fffffffffff","8025fffffffffff","802dfffffffffff"],"ring3":["801dfffffffffff","8023fffffffffff","8027fffffffffff","8029fffffffffff","802bfffffffffff","802ffffffffffff","8031fffffffffff","8033fffffffffff","8035fffffffffff","8039fffffffffff","803bfffffffffff","803dfffffffffff","803ffffffffffff","8041fffffffffff","8043fffffffffff","8053fffffffffff"],"diskDistances2":[["8001fffffffffff"],["8003fffffffffff","8005fffffffffff","8007fffffffffff","8009fffffffffff","800bfffffffffff","8011fffffffffff"],["800dfffffffffff","800ffffffffffff","8013fffffffffff","8015fffffffffff","8017fffffffffff","8019fffffffffff","801bfffffffffff","801ffffffffffff","8021fffffffffff","8025fffffffffff","802dfffffffffff"]],"target":"8005fffffffffff","distance":1,"path":["8001fffffffffff","8005fffffffffff"],"neighbor":true}
```

### `edges`

One line per **valid cell**, paired with a `target` drawn from its 1-disk.
The target is the cell itself about one time in seven, which makes
`cellsToDirectedEdge` fail with `E_NOT_NEIGHBORS`.

| Key | Function | Type |
|---|---|---|
| `index` | the subject cell | index string |
| `edges` | `originToDirectedEdges(index)`, with `0` padding removed | set of index strings |
| `destinations` | `getDirectedEdgeDestination(e)` for each `e` in `edges`, in the written (ascending) order | sequence of index strings |
| `target` | the second cell of the pair | index string |
| `edge` | `cellsToDirectedEdge(index, target)` | index string, or error |

A consumer must also check that `isValidDirectedEdge(e)` holds and
`getDirectedEdgeOrigin(e) == index` for every `e` in `edges`.

Example:

```json
{"index":"8005fffffffffff","edges":["11005fffffffffff","12005fffffffffff","13005fffffffffff","14005fffffffffff","15005fffffffffff","16005fffffffffff"],"destinations":["800dfffffffffff","8015fffffffffff","8017fffffffffff","8001fffffffffff","8003fffffffffff","800bfffffffffff"],"target":"8003fffffffffff","edge":"15005fffffffffff"}
```

### `vertexes`

One line per **valid cell**. A pentagon has no vertex number 5, so
`cellToVertex` fails with `E_DOMAIN` there.

| Key | Function | Type |
|---|---|---|
| `index` | the subject cell | index string |
| `vertexes` | `cellToVertexes(index)`, with `0` padding removed | set of index strings |
| `byNumber` | `cellToVertex(index, n)` for `n` from 0 to 5, in that order | sequence of six entries, each an index string or error |

A consumer must also check that `isValidVertex(v)` holds for every `v` in
`vertexes`. `vertexToLatLng` belongs to the floats group.

Example:

```json
{"index":"8009fffffffffff","vertexes":["20001fffffffffff","21009fffffffffff","22009fffffffffff","25001fffffffffff","25007fffffffffff"],"byNumber":["25007fffffffffff","21009fffffffffff","22009fffffffffff","20001fffffffffff","25001fffffffffff",{"err":"E_DOMAIN"}]}
```

### `localij`

One line per **(origin, target) pair** of valid cells at the same
resolution; the line's subject is the pair. Mode is always 0. Pairs that
cross too many base cells fail with `E_FAILED`, and those rows are the
group's error cases.

| Key | Function | Type |
|---|---|---|
| `index` | the origin cell | index string |
| `target` | the cell being located | index string |
| `ij` | `cellToLocalIj(index, target, 0)` | `{"i": integer, "j": integer}`, or error |
| `cell` | `localIjToCell(index, ij, 0)` when `ij` succeeded; otherwise the same error as `ij` | index string, or error |

Example:

```json
{"index":"8003fffffffffff","target":"8013fffffffffff","ij":{"i":-1,"j":0},"cell":"8013fffffffffff"}
{"index":"8003fffffffffff","target":"8001fffffffffff","ij":{"i":1,"j":0},"cell":"8001fffffffffff"}
```

### `sets`

One line per **input set**, built from a subject cell in one of six kinds.
The `id` is the kind and the cell, joined by `:`. `input` is a sequence
because duplicates and invalid entries are part of the test; the two
function outputs are sets.

| Key | Function | Type |
|---|---|---|
| `id` | `<kind>:<cell>` | string |
| `input` | the cells passed to both functions, in order | sequence of index strings |
| `compact` | `compactCells(input)` | set of index strings, or error |
| `uncompactRes` | the resolution passed to `uncompactCells` | integer |
| `uncompact` | `uncompactCells(input, uncompactRes)` | set of index strings, or error |

The kinds, for a subject at resolution `res`:

| Kind | `input` | `uncompactRes` |
|---|---|---|
| `children` | `cellToChildren(cell, min(res + 2, 15))` | `min(res + 2, 15)` |
| `disk` | `gridDisk(cell, 2)` in slot order | `res + 1` |
| `holes` | `gridDisk(cell, 3)` in slot order with one to three slots cleared (see Sampling) | `res` |
| `duplicate` | `cellToChildren(p, res(p) + 1)` followed by its first entry again, where `p` is the cell, or its parent when `res` is 15 | `res(p) + 1` |
| `invalid` | `gridDisk(cell, 1)` in slot order followed by the cell with reserved bits 56-58 set to `001` | `res` |
| `coarse` | `gridDisk(cell, 1)` in slot order | `res - 1` |

Padding is removed from every input. `duplicate` fails `compactCells` with
`E_DUPLICATE_INPUT`; `invalid` fails it with `E_CELL_INVALID` except at
resolution 0, where the reference copies the input through unchecked;
`coarse` fails `uncompactCells` with `E_RES_MISMATCH`, as does `disk` at
resolution 15. Mixed-resolution input to `compactCells` is outside its
contract and is not sampled.

A consumer must also check the round trip `uncompactCells(compact, r) ==
input` as sets, where `r` is the finest resolution in `input`, whenever
`compact` succeeded and every entry of `input` is a valid cell.

Example:

```json
{"id":"coarse:8009fffffffffff","input":["8007fffffffffff","8009fffffffffff","8019fffffffffff","8001fffffffffff","801ffffffffffff","8011fffffffffff"],"compact":["8001fffffffffff","8007fffffffffff","8009fffffffffff","8011fffffffffff","8019fffffffffff","801ffffffffffff"],"uncompactRes":-1,"uncompact":{"err":"E_RES_MISMATCH"}}
{"id":"duplicate:81083ffffffffff","input":["820807fffffffff","820817fffffffff","82081ffffffffff","820827fffffffff","82082ffffffffff","820837fffffffff","820807fffffffff"],"compact":{"err":"E_DUPLICATE_INPUT"},"uncompactRes":2,"uncompact":["820807fffffffff","820817fffffffff","82081ffffffffff","820827fffffffff","82082ffffffffff","820837fffffffff"]}
```

### `regions`

One line per **polygon and fill resolution**. Polygons are scaled cell
boundaries (see Sampling), so each has five to ten vertices and may have one
hole; the `id` is the source cell, the fill resolution, and `:hole` when
present. A fill resolution of 16 gives the `E_RES_DOMAIN` rows.

| Key | Function | Type |
|---|---|---|
| `id` | `<cell>:<res>` or `<cell>:<res>:hole` | string |
| `polygon` | the input: `{"outer": loop, "holes": [loop, ...]}`, each loop a sequence of points | object |
| `res` | the fill resolution | integer |
| `cells` | `polygonToCells(polygon, res, 0)`, with `0` padding removed | set of index strings, or error |
| `center`, `full`, `overlapping`, `overlappingBbox` | `polygonToCellsExperimental(polygon, res, mode)` for containment modes 0 to 3, with `0` padding removed | set of index strings, or error |
| `multiPolygon` | `cellsToLinkedMultiPolygon(cells)` when `cells` succeeded, otherwise the same error | sequence of `{"outer": loop, "holes": [loop, ...]}`, or error |

`multiPolygon` is compared by structure and by points. The polygons are a
sequence, in the reference's order of decreasing outer-loop area. Within a
polygon the outer loop is a sequence of points that may start at any vertex,
compared point by point with the angular tolerance, and the holes are a set
of such loops. The `cells` fed to `cellsToLinkedMultiPolygon` are passed in
ascending order.

Scaling the boundaries matters (see Sampling): a polygon that is exactly a cell boundary
puts the finer cells' vertices on its edges, and whether `full` and
`overlapping` count such a cell then depends on rounding, in the reference as
much as in a port. The scaled polygons keep every cell vertex clear of the
polygon edges.

Example:

```json
{"id":"8009fffffffffff:1","polygon":{"outer":[[63.41604328760733,-6.2487422207291372],[57.505414797708795,6.5261570545257905],[59.661235921869135,22.173417676459856],[68.083966331138825,27.572264214363457],[71.588178973942178,2.3677280966481229]],"holes":[]},"res":1,"cells":["81083ffffffffff","8108bffffffffff","8108fffffffffff","81093ffffffffff","81097ffffffffff","8109bffffffffff"],"center":["81083ffffffffff","8108bffffffffff","8108fffffffffff","81093ffffffffff","81097ffffffffff","8109bffffffffff"],"full":["81083ffffffffff"],"overlapping":["81083ffffffffff","8108bffffffffff","8108fffffffffff","81093ffffffffff","81097ffffffffff","8109bffffffffff"],"overlappingBbox":["81013ffffffffff","81017ffffffffff","81073ffffffffff","81077ffffffffff","8107bffffffffff","81083ffffffffff","8108bffffffffff","8108fffffffffff","81093ffffffffff","81097ffffffffff","8109bffffffffff","81113ffffffffff","81117ffffffffff","81193ffffffffff","81197ffffffffff","811f3ffffffffff","811f7ffffffffff"],"multiPolygon":[{"outer":[[56.632952719050827,18.275235951757786],[58.401544870352701,25.082722326707874],[62.478113451924763,24.517172437523488],[64.873036611864293,31.517537185292614],[68.929957881939814,31.831280499087395],[70.052151519508882,20.597244293835026],[73.487456497717517,14.551845028116778],[73.310223685443972,0.32561035194322951],[69.273859191908159,-1.6665855042241828],[67.153547845205082,-10.558792289389457],[63.095054077525454,-10.444977544778338],[61.545509577880757,-2.2975260876218235],[57.689497374592854,-0.93158716351061854],[55.70676846515228,5.5236465492903095],[58.130531165851423,11.555977692900695]],"holes":[]}]}
```

### `digests`

Digests over **every cell at resolutions 0 through 6**, 16,838,852 cells
in all, so the group checks the whole grid at those resolutions without
storing it. Two files hold the same twelve streams at two levels:

- `digests/resolutions.jsonl`: one line per resolution, `{"res", "count",
  "digests"}`, each stream run over every cell at that resolution in
  ascending order.
- `digests/baseCells.jsonl`: one line per resolution and base cell,
  `{"res", "baseCell", "count", "digests"}`, each stream run over the cells
  of that resolution descending from that base cell, in ascending order.
  Ascending order over all cells is base-cell order then digit order, so the
  per-resolution stream is the concatenation of the 122 per-base-cell
  streams. A per-resolution mismatch therefore always has a per-base-cell
  mismatch that localises it to one subtree, which a bisecting rerun of the
  generator resolves.

`digests` is an object from stream name to hex SHA-256. Every stream has
one line per cell, starting with the cell, and the streams are:

| Stream | Fields after the cell, for a cell at resolution `r` |
|---|---|
| `cells` | none |
| `parents` | `cellToParent(cell, p)` for `p` from `r - 1` down to 0 |
| `children` | `cellToCenterChild(cell, r + 1)`, `cellToChildrenSize(cell, r + 1)`, `cellToChildPos(cell, 0)`, then `childPosToCell(thatPos, parent0, r)` where `parent0` is the resolution-0 ancestor |
| `inspection` | `getBaseCellNumber`, `isPentagon`, `isResClassIII`, then `getIcosahedronFaces` as a set |
| `disks` | `gridDisk(cell, 1)` as a set, `\|`, `gridDisk(cell, 2)` as a set, `\|`, `gridDisk(cell, 3)` as a set |
| `rings` | the same with `gridRing` |
| `distances` | `gridDistance(cell, m)` for each `m` of the 3-disk in ascending order, each an integer or an error name |
| `edges` | for each edge of `originToDirectedEdges(cell)` in ascending order, the edge then `getDirectedEdgeDestination(edge)`; then `\|`; then `areNeighborCells(cell, m)` for each `m` of the 1-disk in ascending order, each `1`, `0` or an error name |
| `vertexes` | `cellToVertexes(cell)` as a set |
| `localIj` | `cellToLocalIj(origin, cell, 0)` as `i j`, or an error name, where `origin` is `cellToCenterChild(parent0, r)` (`parent0` itself at resolution 0) |
| `compact` | `compactCells` of `cellToChildren(cell, r + 1)` sorted ascending with its last entry removed, as a set |
| `roundTrip` | `latLngToCell(cellToLatLng(cell), r)` |

Line examples, for the resolution-1 pentagon `81083ffffffffff`, in the
`cells`, `parents`, `children`, `inspection`, `rings`, `edges`, `localIj`,
`compact` and `roundTrip` streams:

```
81083ffffffffff
81083ffffffffff 8009fffffffffff
81083ffffffffff 820807fffffffff 6 0 81083ffffffffff
81083ffffffffff 4 1 1 0 1 2 3 4
81083ffffffffff 8108bffffffffff 8108fffffffffff 81093ffffffffff 81097ffffffffff 8109bffffffffff | 81013ffffffffff 81017ffffffffff 81073ffffffffff 81077ffffffffff 81113ffffffffff 81117ffffffffff 81193ffffffffff 81197ffffffffff 811f3ffffffffff 811f7ffffffffff | 81003ffffffffff 81007ffffffffff 8101bffffffffff 81063ffffffffff 81067ffffffffff 8107bffffffffff 81103ffffffffff 81107ffffffffff 8111bffffffffff 81183ffffffffff 81187ffffffffff 8119bffffffffff 811e3ffffffffff 811e7ffffffffff 811fbffffffffff
81083ffffffffff 121083ffffffffff 8108bffffffffff 131083ffffffffff 8108fffffffffff 141083ffffffffff 81093ffffffffff 151083ffffffffff 81097ffffffffff 161083ffffffffff 8109bffffffffff | 0 1 1 1 1 1
81083ffffffffff 0 0
81083ffffffffff 820807fffffffff 820817fffffffff 82081ffffffffff 820827fffffffff 82082ffffffffff
81083ffffffffff 81083ffffffffff
```

`roundTrip` lines repeat the cell whenever its center maps back to it, which
is the property being checked. The `compact` stream exercises a set that
must not merge; a full set of children compacting to the cell is covered by
`sets`.

Example record:

```json
{"res":0,"count":122,"digests":{"cells":"...","children":"...","compact":"...","disks":"...","distances":"...","edges":"...","inspection":"...","localIj":"...","parents":"...","rings":"...","roundTrip":"...","vertexes":"..."}}
```

Enumerating resolution 6 takes minutes in a port, so a runner may stop
earlier by default and offer a way to go further; the Go runner stops at
resolution 3 unless `H3_CONFORMANCE_DIGESTS_MAXRES` says otherwise.

## Sampling

Inputs are chosen by a deterministic procedure from `generator.seed`, so that
a generator written against the C library directly reproduces this suite byte
for byte, and so that a failing line can be traced to the step that produced
it. The procedure for each file is given here in full. Every file starts
from a fresh generator whose state is the seed, so files can be produced
independently and in any order.

### Pseudo-random source

The sampler is splitmix64 with the seed as its initial state. Each draw
returns 64 bits:

```
state += 0x9e3779b97f4a7c15
z = state
z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
z = (z ^ (z >> 27)) * 0x94d049bb133111eb
return z ^ (z >> 31)
```

All arithmetic is modulo 2^64. "Draw `n`" below means one draw reduced by
`% n`. Steps consume draws in exactly the order written; a step that says it
draws nothing must not touch the generator.

### Index construction

`build(mode, res, baseCell, digits)` assembles a 64-bit index: bit 63 clear,
bits 59-62 the mode, bits 56-58 zero, bits 52-55 the resolution, bits 45-51
the base cell, then fifteen 3-bit digit slots from bit 42 down to bit 0, the
first `res` slots holding `digits` in order and the rest holding `7`.

"Draw a cell at `res`" means: draw 122 for the base cell, then draw 7 for
each of `res` digits in order, and `build(1, res, baseCell, digits)`. The
result may be invalid: a pentagon base cell with a leading `1` digit is a
deleted subsequence, and such draws are kept.

### `inspection/cells.jsonl`

1. The fixed indexes `0`, `ffffffffffffffff`, `7fffffffffffffff`,
   `800000000000000`, `8001fffffffffff`, `81283ffffffffff`,
   `804dfffffffffff` and `fffffffffffffff`. No draws.
2. `build(1, 0, b, [])` for every base cell `b` from 0 to 121. No draws.
3. `getPentagons(res)` for `res` 1 through 15, in the order the reference
   returns them. No draws.
4. For `res` 1 through 15: draw 48 cells at `res`.
5. For `res` 0 through 15, four times: draw a cell at `res` as the base, then
   emit these mutations of it in order, each drawing only as stated:
   1. mode field replaced by draw 16;
   2. reserved bits (56-58) replaced by 1 + draw 7;
   3. bit 63 set, no draw;
   4. base cell field replaced by 122 + draw 6;
   5. if `res > 0`: digit slot 1 + draw `res` set to 7;
   6. if `res < 15`: digit slot `res` + 1 + draw (15 - `res`) set to draw 7.
6. For `res` 0 through 15, four times: draw cells at `res` until
   `isValidCell` accepts one; take its `originToDirectedEdges` and
   `cellToVertexes` arrays (six slots each, a zero where a pentagon has no
   edge or vertex); then twice, draw 6 and emit the edge at that slot if it is
   non-zero; then twice, draw 6 and emit the vertex at that slot if it is
   non-zero.
7. 256 raw draws, each emitted as an index.

The emitted indexes are sorted ascending as unsigned 64-bit integers and
duplicates are removed. Each remaining index becomes one line.

### Valid-cell subjects

Several files share one way of choosing subjects, parameterized by a count
`n` per resolution:

1. `build(1, 0, b, [])` for every base cell `b` from 0 to 121. No draws.
2. `getPentagons(res)` for `res` 1 through 15, in the order the reference
   returns them. No draws.
3. For `res` 1 through 15, `n` times: draw cells at `res` until `isValidCell`
   accepts one, and emit it.

Sorted ascending and deduplicated as above. The draws a file's records need
beyond this happen afterwards, for each subject in file order.

"Draw from the `k`-disk of `h`" means: take the `gridDisk(h, k)` array of
`maxGridDiskSize(k)` slots as the reference fills it, including its `0`
padding, and draw `maxGridDiskSize(k)` repeatedly until the slot it selects
is non-zero.

### `hierarchy/cells.jsonl`

Valid-cell subjects with `n` = 32. No further draws.

### `traversal/cells.jsonl`

Valid-cell subjects with `n` = 16. Then for each subject, `target` is drawn
from its 3-disk.

### `edges/cells.jsonl`

Valid-cell subjects with `n` = 16. Then for each subject, `target` is drawn
from its 1-disk.

### `vertexes/cells.jsonl`

Valid-cell subjects with `n` = 16. No further draws.

### `localij/pairs.jsonl`

Valid-cell subjects with `n` = 16 are the origins. Then for each origin, a
near target is drawn from its 3-disk, and a far target is drawn by drawing
cells at the origin's resolution until `isValidCell` accepts one. The origin
is written with the near target, then with the far target unless the two
targets are the same cell.

### `sets/sets.jsonl`

Subjects:

1. `getPentagons(res)` for `res` 0 through 15, in the order the reference
   returns them. No draws.
2. For `res` 0 through 15, twice: draw cells at `res` until `isValidCell`
   accepts one, and emit it.

Sorted ascending and deduplicated as above. Then for each subject, the six
kinds are written in the order of the kinds table. Only `holes` draws: draw 3
to get a count from 1 to 3, then that many times draw 37 and clear that slot
of the `gridDisk(cell, 3)` array; a slot that is already zero stays zero.

### `regions/polygons.jsonl`

Subjects:

1. `getPentagons(res)` for `res` 0, 4, 8 and 12, in the order the reference
   returns them. No draws.
2. For `res` 0 through 13, three times: draw cells at `res` until
   `isValidCell` accepts one, and emit it.

Sorted ascending and deduplicated as above. No further draws.

"The boundary of `h` scaled by `f`" is `cellToBoundary(h)` with every vertex
moved away from `cellToLatLng(h)` by the factor `f`, computed in degrees:
with center `(clat, clng)` and vertex `(vlat, vlng)`, the new vertex is
`(clat + f * (vlat - clat), wrap(clng + f * wrap(vlng - clng)))`, where
`wrap` adds or subtracts 360 to bring a value into [-180, 180]. Vertex order
is kept. Degrees are what the file holds, so a generator converts the scaled
degrees to its implementation's units rather than scaling in radians.

For each subject `h` at resolution `res`, four lines:

1. the boundary of `h` scaled by 0.8, filled at `res + 1`;
2. the boundary of `h` scaled by 1.2, filled at `res + 2`;
3. the same outer loop with one hole, the boundary of
   `cellToCenterChild(h, res + 1)` scaled by 1.2, filled at `res + 2`;
4. the boundary of `h` scaled by 1.2 with no hole, filled at 16.

### `digests/resolutions.jsonl` and `digests/baseCells.jsonl`

One line per resolution 0 through 6, in that order, and within
`baseCells.jsonl` one line per base cell 0 through 121 for each resolution.
No draws; the inputs are every cell at the resolution, enumerated as
`cellToChildren(baseCell, res)` for each base cell in order, each sorted
ascending.

## Running and regenerating

`go test ./x/h3go/conformance/...` runs the checked-in suite against `x/h3go`
and verifies the manifest against the vendored `H3_VERSION`. The tests are
pure Go.

`go generate ./x/h3go/conformance` rewrites `testdata/` from the vendored C
library. The generator, `internal/gen/gen.c`, is a standalone C99 program
that depends only on the public H3 API and the C standard library;
`generate.sh` compiles it against the `h3_*.c` sources at the repository root
and runs it. Every expected value is therefore the reference implementation's
own answer, and nothing passes through the Go binding. Regenerate after
bumping the vendored H3 version, and expect a diff only when the reference's
behaviour changed.

```
gen -o <out-dir> -v <h3-version> [-s <seed>]
```

## Where the files should live

The files under `testdata/` are a bootstrap. They are produced here because
the C library does not yet ship a generator, and they are checked in so that
`go test ./...` is pure Go, works offline, and shows exactly what `x/h3go` is
held to. The manifest hash makes any change to them a visible diff.

The intended home is the C library. The generator is written in C against
the public API for that reason: `gen.c` can be added to uber/h3 as a testapp
without modification, and the sampling
procedure above is specified in full so that any other generator following
it produces byte-identical files from the same seed. That property has been
exercised once already: the first version of this suite was produced by a Go
generator, and `gen.c`, written from the README, reproduced its record file
byte for byte. Once upstream publishes the files, in `tests/inputfiles/` or
as a release asset per tag, this package consumes them: `testdata/` becomes a
vendored snapshot verified by hash against the upstream manifest, and
`internal/gen` is removed. The runner, the manifest check against the
vendored `H3_VERSION`, and a copy of the files small enough to test offline
stay here either way. What moves is authorship of the expected values, from
this repository's build of the C library to upstream's own build and CI.
