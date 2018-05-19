package main

/*
#include <stdint.h>
#include <string.h>
#include <errno.h>
#include <stdio.h>
#include <stdlib.h>

typedef uint8_t wg_key[32];
typedef char wg_key_b64_string[((sizeof(wg_key) + 2) / 3) * 4 + 1];
extern void wg_generate_private_key(wg_key private_key);
extern void wg_key_to_base64(wg_key_b64_string base64, const wg_key key);
extern void wg_generate_public_key(wg_key public_key, const wg_key private_key);
extern int wg_add_device(const char *device_name);

void add() {
	const char *ifname = "wg0";
	wg_add_device(ifname);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func genkey() string {
	key := C.CString("")
	//defer C.free(unsafe.Pointer(key))
	wkey := (*C.uchar)(unsafe.Pointer(key))
	C.wg_generate_private_key(wkey)

	b64 := C.CString("")
	//defer C.free(unsafe.Pointer(b64))
	C.wg_key_to_base64(b64, wkey)
	return C.GoString(b64)
}

func genpubkey(privatekey string) string {
	key := C.CString("")
	//defer C.free(unsafe.Pointer(key))
	wkey := (*C.uchar)(unsafe.Pointer(key))

	C.wg_generate_public_key(wkey, (*C.uchar)(unsafe.Pointer(C.CString(privatekey))))

	b64 := C.CString("")
	//defer C.free(unsafe.Pointer(b64))
	C.wg_key_to_base64(b64, wkey)

	return C.GoString(b64)
}

func addDevice() {
	// TODO(fntlnz): find a way to pass name
	C.add()
	errStr := C.CString("error detected")
	//defer C.free(unsafe.Pointer(errStr))
	C.perror(errStr)
}

func main() {

	addDevice()
	priv := genkey()
	pub := genpubkey(priv)
	fmt.Printf("priv: %s\n", priv)
	fmt.Printf("pub: %s\n", pub)
}
