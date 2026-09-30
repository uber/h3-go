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
/** @file
 * @brief Generates the H3 conformance suite from the C library.
 *
 *  usage: `gen -o <out-dir> -v <h3-version> [-s <seed>]`
 *
 *  Writes `<out-dir>/manifest.json` and one newline-delimited JSON record file
 *  per group under `<out-dir>`. Every expected value is the library's own
 *  answer. The sampling procedure, including the pseudo-random source and the
 *  order of draws, is specified in the suite README; two generators that
 *  follow it produce byte-identical files from the same seed.
 *
 *  The program depends only on the public H3 API and the C standard library,
 *  so it builds unchanged as a testapp in the H3 repository.
 */

#include <inttypes.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "h3api.h"

#define FORMAT_VERSION "h3-conformance/1"
#define GENERATOR_NAME "x/h3go/conformance/internal/gen/gen.c"
#define DEFAULT_SEED UINT64_C(20260929)
#define ANGULAR_DEG_TOLERANCE "1e-11"
#define RELATIVE_TOLERANCE "1e-9"

#define MAX_RES 15
#define NUM_BASE_CELLS 122
#define NUM_DIGITS 7
#define NUM_MODES 16
#define NUM_HEX_EDGES 6
#define NUM_PENTAGONS 12
#define MAX_INDEX_LEN 17

#define MODE_OFFSET 59
#define RES_OFFSET 52
#define BASE_CELL_OFFSET 45
#define RESERVED_OFFSET 56
#define DIGIT_BITS 3
#define DIGIT_MASK UINT64_C(7)
#define HIGH_BIT (UINT64_C(1) << 63)
#define CELL_MODE 1

/* Sample sizes; see README.md, "Sampling". */
#define RANDOM_CELLS_PER_RES 48
#define MUTATION_BASES_PER_RES 4
#define INDEX_BASES_PER_RES 4
#define DRAWS_PER_INDEX_BASE 2
#define RANDOM_PATTERNS 256

static const char *const H3_ERROR_NAMES[] = {
    "E_SUCCESS",        "E_FAILED",           "E_DOMAIN",
    "E_LATLNG_DOMAIN",  "E_RES_DOMAIN",       "E_CELL_INVALID",
    "E_DIR_EDGE_INVALID", "E_UNDIR_EDGE_INVALID", "E_VERTEX_INVALID",
    "E_PENTAGON",       "E_DUPLICATE_INPUT",  "E_NOT_NEIGHBORS",
    "E_RES_MISMATCH",   "E_MEMORY_ALLOC",     "E_MEMORY_BOUNDS",
    "E_OPTION_INVALID", "E_INDEX_INVALID",    "E_BASE_CELL_DOMAIN",
    "E_DIGIT_DOMAIN",   "E_DELETED_DIGIT",
};

/* Hand-picked inputs included regardless of seed. */
static const uint64_t FIXED_INDEXES[] = {
    UINT64_C(0x0),
    UINT64_C(0xffffffffffffffff),
    UINT64_C(0x7fffffffffffffff),
    UINT64_C(0x0800000000000000),
    UINT64_C(0x08001fffffffffff),
    UINT64_C(0x081283ffffffffff),
    UINT64_C(0x0804dfffffffffff),
    UINT64_C(0x0fffffffffffffff),
};

static void fail(const char *message) {
    fprintf(stderr, "gen: %s\n", message);
    exit(1);
}

static void must(H3Error err) {
    if (err) {
        fprintf(stderr, "gen: unexpected H3 error %s\n", H3_ERROR_NAMES[err]);
        exit(1);
    }
}

/* ---- pseudo-random source (README, "Pseudo-random source") ---- */

typedef struct {
    uint64_t state;
} Splitmix64;

static uint64_t draw(Splitmix64 *rng) {
    rng->state += UINT64_C(0x9e3779b97f4a7c15);
    uint64_t z = rng->state;
    z = (z ^ (z >> 30)) * UINT64_C(0xbf58476d1ce4e5b9);
    z = (z ^ (z >> 27)) * UINT64_C(0x94d049bb133111eb);
    return z ^ (z >> 31);
}

static int drawN(Splitmix64 *rng, int n) { return (int)(draw(rng) % (uint64_t)n); }

/* ---- index construction (README, "Index construction") ---- */

static unsigned digitShift(int r) { return (unsigned)((MAX_RES - r) * DIGIT_BITS); }

static uint64_t buildIndex(int mode, int res, int baseCell, const int *digits) {
    uint64_t index = (uint64_t)mode << MODE_OFFSET | (uint64_t)res << RES_OFFSET |
                     (uint64_t)baseCell << BASE_CELL_OFFSET;
    for (int r = 1; r <= MAX_RES; r++) {
        uint64_t digit = r <= res ? (uint64_t)digits[r - 1] : DIGIT_MASK;
        index |= digit << digitShift(r);
    }
    return index;
}

static uint64_t setField(uint64_t index, unsigned offset, unsigned width, uint64_t value) {
    uint64_t mask = ((UINT64_C(1) << width) - 1) << offset;
    return (index & ~mask) | ((value << offset) & mask);
}

static uint64_t drawCell(Splitmix64 *rng, int res) {
    int baseCell = drawN(rng, NUM_BASE_CELLS);
    int digits[MAX_RES];
    for (int i = 0; i < res; i++) {
        digits[i] = drawN(rng, NUM_DIGITS);
    }
    return buildIndex(CELL_MODE, res, baseCell, digits);
}

static uint64_t drawValidCell(Splitmix64 *rng, int res) {
    for (;;) {
        uint64_t index = drawCell(rng, res);
        if (isValidCell(index)) {
            return index;
        }
    }
}

/* ---- growable list of indexes ---- */

typedef struct {
    uint64_t *items;
    size_t len;
    size_t cap;
} IndexList;

static void push(IndexList *list, uint64_t index) {
    if (list->len == list->cap) {
        list->cap = list->cap ? list->cap * 2 : 1024;
        list->items = realloc(list->items, list->cap * sizeof *list->items);
        if (!list->items) {
            fail("out of memory");
        }
    }
    list->items[list->len++] = index;
}

static int compareIndexes(const void *a, const void *b) {
    uint64_t x = *(const uint64_t *)a, y = *(const uint64_t *)b;
    return x < y ? -1 : x > y;
}

static void sortAndDedup(IndexList *list) {
    qsort(list->items, list->len, sizeof *list->items, compareIndexes);
    size_t out = 0;
    for (size_t i = 0; i < list->len; i++) {
        if (out == 0 || list->items[out - 1] != list->items[i]) {
            list->items[out++] = list->items[i];
        }
    }
    list->len = out;
}

/* ---- inspection group sampling (README, "inspection/cells.jsonl") ---- */

static void pushMutations(IndexList *list, Splitmix64 *rng, int res, uint64_t base) {
    push(list, setField(base, MODE_OFFSET, 4, (uint64_t)drawN(rng, NUM_MODES)));
    push(list, setField(base, RESERVED_OFFSET, 3, (uint64_t)(1 + drawN(rng, (int)DIGIT_MASK))));
    push(list, base | HIGH_BIT);
    push(list, setField(base, BASE_CELL_OFFSET, 7,
                        (uint64_t)(NUM_BASE_CELLS + drawN(rng, 128 - NUM_BASE_CELLS))));
    if (res > 0) {
        push(list, setField(base, digitShift(1 + drawN(rng, res)), DIGIT_BITS, DIGIT_MASK));
    }
    if (res < MAX_RES) {
        int position = res + 1 + drawN(rng, MAX_RES - res);
        push(list, setField(base, digitShift(position), DIGIT_BITS, (uint64_t)drawN(rng, NUM_DIGITS)));
    }
}

static void sampleInspection(IndexList *list, Splitmix64 *rng) {
    for (size_t i = 0; i < sizeof FIXED_INDEXES / sizeof FIXED_INDEXES[0]; i++) {
        push(list, FIXED_INDEXES[i]);
    }

    for (int baseCell = 0; baseCell < NUM_BASE_CELLS; baseCell++) {
        push(list, buildIndex(CELL_MODE, 0, baseCell, NULL));
    }

    for (int res = 1; res <= MAX_RES; res++) {
        H3Index pentagons[NUM_PENTAGONS];
        must(getPentagons(res, pentagons));
        for (int i = 0; i < NUM_PENTAGONS; i++) {
            push(list, pentagons[i]);
        }
    }

    for (int res = 1; res <= MAX_RES; res++) {
        for (int i = 0; i < RANDOM_CELLS_PER_RES; i++) {
            push(list, drawCell(rng, res));
        }
    }

    for (int res = 0; res <= MAX_RES; res++) {
        for (int i = 0; i < MUTATION_BASES_PER_RES; i++) {
            pushMutations(list, rng, res, drawCell(rng, res));
        }
    }

    for (int res = 0; res <= MAX_RES; res++) {
        for (int i = 0; i < INDEX_BASES_PER_RES; i++) {
            H3Index cell = drawValidCell(rng, res);
            H3Index edges[NUM_HEX_EDGES], vertexes[NUM_HEX_EDGES];
            must(originToDirectedEdges(cell, edges));
            must(cellToVertexes(cell, vertexes));
            for (int j = 0; j < DRAWS_PER_INDEX_BASE; j++) {
                H3Index edge = edges[drawN(rng, NUM_HEX_EDGES)];
                if (edge) {
                    push(list, edge);
                }
            }
            for (int j = 0; j < DRAWS_PER_INDEX_BASE; j++) {
                H3Index vertex = vertexes[drawN(rng, NUM_HEX_EDGES)];
                if (vertex) {
                    push(list, vertex);
                }
            }
        }
    }

    for (int i = 0; i < RANDOM_PATTERNS; i++) {
        push(list, draw(rng));
    }

    sortAndDedup(list);
}

/* ---- inspection group records ---- */

static void printIndex(FILE *out, H3Index h) {
    char buf[MAX_INDEX_LEN];
    must(h3ToString(h, buf, sizeof buf));
    fprintf(out, "\"%s\"", buf);
}

static void printError(FILE *out, H3Error err) {
    fprintf(out, "{\"err\":\"%s\"}", H3_ERROR_NAMES[err]);
}

static void printBool(FILE *out, int value) { fputs(value ? "true" : "false", out); }

static void writeInspectionRecord(FILE *out, H3Index h) {
    int res = getResolution(h);
    int baseCell = getBaseCellNumber(h);

    fputs("{\"index\":", out);
    printIndex(out, h);
    fprintf(out, ",\"res\":%d,\"baseCell\":%d,\"validCell\":", res, baseCell);
    printBool(out, isValidCell(h));
    fputs(",\"validIndex\":", out);
    printBool(out, isValidIndex(h));
    fputs(",\"resClassIII\":", out);
    printBool(out, isResClassIII(h));
    fputs(",\"pentagon\":", out);
    printBool(out, isPentagon(h));

    fputs(",\"faces\":", out);
    int faceCount;
    must(maxFaceCount(h, &faceCount));
    int *faces = calloc((size_t)faceCount, sizeof *faces);
    if (!faces) {
        fail("out of memory");
    }
    H3Error faceErr = getIcosahedronFaces(h, faces);
    if (faceErr) {
        printError(out, faceErr);
    } else {
        /* Drop the -1 padding and sort ascending; the set is small. */
        int kept = 0;
        for (int i = 0; i < faceCount; i++) {
            if (faces[i] >= 0) {
                faces[kept++] = faces[i];
            }
        }
        for (int i = 1; i < kept; i++) {
            for (int j = i; j > 0 && faces[j - 1] > faces[j]; j--) {
                int tmp = faces[j];
                faces[j] = faces[j - 1];
                faces[j - 1] = tmp;
            }
        }
        fputc('[', out);
        for (int i = 0; i < kept; i++) {
            fprintf(out, i ? ",%d" : "%d", faces[i]);
        }
        fputc(']', out);
    }
    free(faces);

    int digits[MAX_RES];
    fputs(",\"digits\":[", out);
    for (int r = 1; r <= MAX_RES; r++) {
        must(getIndexDigit(h, r, &digits[r - 1]));
        fprintf(out, r > 1 ? ",%d" : "%d", digits[r - 1]);
    }
    fputs("],\"construct\":", out);

    H3Index constructed;
    H3Error constructErr = constructCell(res, baseCell, digits, &constructed);
    if (constructErr) {
        printError(out, constructErr);
    } else {
        printIndex(out, constructed);
    }
    fputs("}\n", out);
}

/* ---- SHA-256 (FIPS 180-4) for the manifest ---- */

static const uint32_t SHA256_K[64] = {
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
};

static uint32_t rotr(uint32_t x, int n) { return (x >> n) | (x << (32 - n)); }

static void sha256Block(uint32_t state[8], const uint8_t block[64]) {
    uint32_t w[64];
    for (int i = 0; i < 16; i++) {
        w[i] = (uint32_t)block[4 * i] << 24 | (uint32_t)block[4 * i + 1] << 16 |
               (uint32_t)block[4 * i + 2] << 8 | (uint32_t)block[4 * i + 3];
    }
    for (int i = 16; i < 64; i++) {
        uint32_t s0 = rotr(w[i - 15], 7) ^ rotr(w[i - 15], 18) ^ (w[i - 15] >> 3);
        uint32_t s1 = rotr(w[i - 2], 17) ^ rotr(w[i - 2], 19) ^ (w[i - 2] >> 10);
        w[i] = w[i - 16] + s0 + w[i - 7] + s1;
    }
    uint32_t a = state[0], b = state[1], c = state[2], d = state[3];
    uint32_t e = state[4], f = state[5], g = state[6], h = state[7];
    for (int i = 0; i < 64; i++) {
        uint32_t t1 = h + (rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25)) + ((e & f) ^ (~e & g)) +
                      SHA256_K[i] + w[i];
        uint32_t t2 = (rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22)) + ((a & b) ^ (a & c) ^ (b & c));
        h = g;
        g = f;
        f = e;
        e = d + t1;
        d = c;
        c = b;
        b = a;
        a = t1 + t2;
    }
    state[0] += a;
    state[1] += b;
    state[2] += c;
    state[3] += d;
    state[4] += e;
    state[5] += f;
    state[6] += g;
    state[7] += h;
}

/* sha256Hex writes the lowercase hex digest of data into hex (65 bytes). */
static void sha256Hex(const uint8_t *data, size_t len, char hex[65]) {
    uint32_t state[8] = {0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
                         0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19};
    size_t offset = 0;
    for (; offset + 64 <= len; offset += 64) {
        sha256Block(state, data + offset);
    }
    uint8_t tail[128] = {0};
    size_t rest = len - offset;
    memcpy(tail, data + offset, rest);
    tail[rest] = 0x80;
    size_t tailLen = rest + 1 + 8 <= 64 ? 64 : 128;
    uint64_t bitLen = (uint64_t)len * 8;
    for (size_t i = 0; i < 8; i++) {
        tail[tailLen - 1 - i] = (uint8_t)(bitLen >> (8 * i));
    }
    for (size_t i = 0; i < tailLen; i += 64) {
        sha256Block(state, tail + i);
    }
    for (int i = 0; i < 8; i++) {
        sprintf(hex + 8 * i, "%08" PRIx32, state[i]);
    }
}

/* ---- files and manifest ---- */

typedef struct {
    const char *name; /* slash-separated path relative to the manifest */
    long records;
    char sha256[65];
} FileInfo;

static char *joinPath(const char *dir, const char *name) {
    size_t len = strlen(dir) + 1 + strlen(name) + 1;
    char *path = malloc(len);
    if (!path) {
        fail("out of memory");
    }
    snprintf(path, len, "%s/%s", dir, name);
    return path;
}

/* describeFile reads a written record file back to count lines and hash it. */
static void describeFile(const char *path, FileInfo *info) {
    FILE *f = fopen(path, "rb");
    if (!f) {
        fail("could not reopen record file");
    }
    if (fseek(f, 0, SEEK_END)) {
        fail("seek");
    }
    long size = ftell(f);
    if (size < 0 || fseek(f, 0, SEEK_SET)) {
        fail("tell");
    }
    uint8_t *data = malloc((size_t)size + 1);
    if (!data) {
        fail("out of memory");
    }
    if (fread(data, 1, (size_t)size, f) != (size_t)size) {
        fail("read");
    }
    fclose(f);

    info->records = 0;
    for (long i = 0; i < size; i++) {
        info->records += data[i] == '\n';
    }
    sha256Hex(data, (size_t)size, info->sha256);
    free(data);
}

static void writeInspectionFile(const char *outDir, Splitmix64 *rng, FileInfo *info) {
    IndexList list = {0};
    sampleInspection(&list, rng);

    char *path = joinPath(outDir, info->name);
    FILE *out = fopen(path, "wb");
    if (!out) {
        fail("could not create inspection/cells.jsonl (does <out-dir>/inspection exist?)");
    }
    for (size_t i = 0; i < list.len; i++) {
        writeInspectionRecord(out, list.items[i]);
    }
    if (fclose(out)) {
        fail("write");
    }
    free(list.items);

    describeFile(path, info);
    free(path);
}

static void writeManifest(const char *outDir, const char *version, uint64_t seed,
                          const FileInfo *files, size_t numFiles) {
    char *path = joinPath(outDir, "manifest.json");
    FILE *out = fopen(path, "wb");
    if (!out) {
        fail("could not create manifest.json");
    }
    fprintf(out,
            "{\n"
            "  \"format\": \"" FORMAT_VERSION "\",\n"
            "  \"h3Version\": \"%s\",\n"
            "  \"generator\": {\n"
            "    \"name\": \"" GENERATOR_NAME "\",\n"
            "    \"seed\": %" PRIu64 "\n"
            "  },\n"
            "  \"tolerances\": {\n"
            "    \"angularDeg\": " ANGULAR_DEG_TOLERANCE ",\n"
            "    \"relative\": " RELATIVE_TOLERANCE "\n"
            "  },\n"
            "  \"files\": {\n",
            version, seed);
    for (size_t i = 0; i < numFiles; i++) {
        fprintf(out,
                "    \"%s\": {\n"
                "      \"records\": %ld,\n"
                "      \"sha256\": \"%s\"\n"
                "    }%s\n",
                files[i].name, files[i].records, files[i].sha256, i + 1 < numFiles ? "," : "");
    }
    fputs("  }\n}\n", out);
    if (fclose(out)) {
        fail("write");
    }
    free(path);
}

int main(int argc, char *argv[]) {
    const char *outDir = NULL;
    const char *version = NULL;
    uint64_t seed = DEFAULT_SEED;

    for (int i = 1; i < argc; i++) {
        if (!strcmp(argv[i], "-o") && i + 1 < argc) {
            outDir = argv[++i];
        } else if (!strcmp(argv[i], "-v") && i + 1 < argc) {
            version = argv[++i];
        } else if (!strcmp(argv[i], "-s") && i + 1 < argc) {
            seed = strtoull(argv[++i], NULL, 10);
        } else {
            outDir = NULL;
            break;
        }
    }
    if (!outDir || !version) {
        fprintf(stderr, "usage: %s -o <out-dir> -v <h3-version> [-s <seed>]\n", argv[0]);
        return 1;
    }
    if (version[0] == 'v') {
        version++;
    }

    Splitmix64 rng = {seed};
    FileInfo files[] = {{"inspection/cells.jsonl", 0, ""}};
    writeInspectionFile(outDir, &rng, &files[0]);
    writeManifest(outDir, version, seed, files, sizeof files / sizeof files[0]);
    return 0;
}
