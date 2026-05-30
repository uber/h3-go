#include <h3_latLngBatch.h>

H3Error latLngToCellBatch(
    const LatLng *lls,
    size_t n,
    int res,
    H3Index *out)
{
    for (size_t i = 0; i < n; i++)
    {
        H3Error err = latLngToCell(&lls[i], res, &out[i]);
        if (err != E_SUCCESS)
        {
            return err;
        }
    }
    return E_SUCCESS;
}