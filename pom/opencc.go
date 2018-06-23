package main

import "unsafe"

/*
#cgo LDFLAGS: -L ./opencc/lib -lopencc

#include <string.h>
#include "./opencc/include/opencc.h"

void Convert_free_string(char *p) {
	opencc_convert_utf8_free(p);
}

void* Opencc_New(const char *configFile) {
	return opencc_open(configFile);
}

void Opencc_Delete(void *id) {
	opencc_close(id);
}

const char *Opencc_Convert(void *id, const char *input) {
	char *output = opencc_convert_utf8(id, input, strlen(input));
	output[strlen(input)] = '\0';
	return output;
}

void Opencc_Free_String(char *p) {
	opencc_convert_utf8_free(p);
}
*/
import "C"

type opencconv struct {
	id unsafe.Pointer
}

func newOpencConv(config string) *opencconv {
	c := opencconv{}
	c.id = C.Opencc_New(C.CString(config))
	return &c
}

func (c *opencconv) convert(input string) string {
	output := C.Opencc_Convert(c.id, C.CString(input))
	defer C.Opencc_Free_String(output)
	return C.GoString(output)
}

func (c *opencconv) close() {
	C.Opencc_Delete(c.id)
}
