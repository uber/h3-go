#ifndef GOWRAPPER_H
#define GOWRAPPER_H

#include "h3api.h"

//wrapper for go
void polyfillGo(int numVerts, GeoCoord *verts, int numHoles, int *hole_numVerts, GeoCoord **hole_verts,int res, H3Index* out);

int H3_EXPORT(maxPolyfillSizeGo)(int numVerts, GeoCoord *verts, int numHoles, int hole_numVerts, GeoCoord *hole_verts,int res);

#endif