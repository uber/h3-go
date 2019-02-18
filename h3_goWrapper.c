#include "goWrapper.h"

int H3_EXPORT(maxPolyfillSizeGo)(int numVerts, GeoCoord *verts, int numHoles, int hole_numVerts, GeoCoord *hole_verts,int res) {
    Geofence geofence = {numVerts, verts};
    Geofence holes = {hole_numVerts,hole_verts};
    GeoPolygon geoPolygon = {geofence, numHoles, &holes};
    return maxPolyfillSize(&geoPolygon, res);
}

void H3_EXPORT(polyfillGo)(int numVerts, GeoCoord *verts, int numHoles, int *hole_numVerts, GeoCoord **hole_verts,int res, H3Index* out) {
    Geofence geofence = {numVerts, verts};
    Geofence *holes = calloc(numHoles, sizeof(Geofence));
    for (int i = 0; i < numHoles; i++) {
        holes[i].numVerts = hole_numVerts[i];
        holes[i].verts = hole_verts[i];
    }
    GeoPolygon geoPolygon = {geofence, numHoles, holes};
    polyfill(&geoPolygon, res, out);
    if (numHoles > 0){
        free(holes);
    }
}