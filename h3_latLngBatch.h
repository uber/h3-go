// LatLng Batch helpers maintained by h3-go on top of the cloned H3 core.
// Lives here so we can ship batched/amortized variants of single-shot
// APIs without modifying the upsteam C library. Each function takes
// input already in H3-native form (e.g. LatLng in radians), matching
// the wire convention the existing Go bindings use after toC().
//
// If H3 core later exposes equivalents, the Go wrappers can be
// re-pointed at the core symbols and these can be deleted.

#ifndef H3_EXT_H
#define H3_EXT_H

#include <h3_h3api.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C"
{
#endif

    // latLngToCellBatch resolved n LatLng inputs (already in radians) to
    // H3 cells at a single resolution. Output buffer must be sized n by
    // the caller. Returns E_SUCCESS on success, or the first H3Error
    // encountered (and stops). Output for rows past the failing index is
    // undefined.
    H3Error latLngToCellBatch(
        const LatLng *lls,
        size_t n,
        int res,
        H3Index *out);

#ifdef __cplusplus
}
#endif

#endif // H3_LAT_LNG_BATCH_H