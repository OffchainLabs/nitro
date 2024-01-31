package arbvid

/*
#cgo CFLAGS: -g -Wall -I${SRCDIR}/../target/include/
#cgo LDFLAGS: ${SRCDIR}/../target/lib/libbrotlidec-static.a ${SRCDIR}/../target/lib/libbrotlienc-static.a ${SRCDIR}/../target/lib/libbrotlicommon-static.a -lm
#include "brotli/encode.h"
#include "brotli/decode.h"
*/

// This is where we would use cgo to call Rust code to verify a namespace using the C FFI
// TODO stretch goal: https://github.com/EspressoSystems/nitro-espresso-integration/issues/71
func verifyNamespace() {

}
